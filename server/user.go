package server

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"starlang-bridge/model"
)

// UserService 负责家长账户的数据库读写
type UserService struct {
	db *gorm.DB
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// LoginInput 登录 / 注册所需的用户信息
type LoginInput struct {
	OpenID     string
	UnionID    string
	SessionKey string
	Nickname   string
	AvatarURL  string
	Phone      string
	IP         string
}

// GetByID 按主键查询用户
func (s *UserService) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByOpenID 按微信 openid 查询用户
func (s *UserService) GetByOpenID(ctx context.Context, openID string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Where("open_id = ?", openID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// LoginOrRegister 按 openid 查询用户，不存在则创建；存在则更新登录信息
func (s *UserService) LoginOrRegister(ctx context.Context, in *LoginInput) (*model.User, error) {
	user, err := s.GetByOpenID(ctx, in.OpenID)
	if err == nil {
		if err := s.refreshLoginInfo(ctx, user, in); err != nil {
			return nil, err
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	newUser := model.User{
		OpenID:      in.OpenID,
		UnionID:     in.UnionID,
		SessionKey:  in.SessionKey,
		Nickname:    in.Nickname,
		AvatarURL:   in.AvatarURL,
		Phone:       in.Phone,
		Status:      model.UserStatusNormal,
		Consent:     model.UserConsentPending,
		LastLoginAt: &now,
		LastLoginIP: in.IP,
	}

	if err := s.db.WithContext(ctx).Create(&newUser).Error; err != nil {
		// openid 唯一键冲突：可能是并发首次登录，或用户曾被软删除后再次登录
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return s.restoreDeleted(ctx, in)
		}
		return nil, err
	}
	return &newUser, nil
}

// restoreDeleted 复用被软删除的账户，使其可以重新登录
func (s *UserService) restoreDeleted(ctx context.Context, in *LoginInput) (*model.User, error) {
	now := time.Now()
	updates := map[string]interface{}{
		"deleted_at":    nil,
		"status":        model.UserStatusNormal,
		"session_key":   in.SessionKey,
		"last_login_at": now,
		"last_login_ip": in.IP,
	}
	if in.UnionID != "" {
		updates["union_id"] = in.UnionID
	}
	if in.Nickname != "" {
		updates["nickname"] = in.Nickname
	}
	if in.AvatarURL != "" {
		updates["avatar_url"] = in.AvatarURL
	}
	if in.Phone != "" {
		updates["phone"] = in.Phone
	}

	if err := s.db.WithContext(ctx).Unscoped().Model(&model.User{}).
		Where("open_id = ?", in.OpenID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByOpenID(ctx, in.OpenID)
}

// UpdateConsent 更新知情同意状态，同意时记录同意时间
func (s *UserService) UpdateConsent(ctx context.Context, userID uint64, consent int8) (*model.User, error) {
	updates := map[string]interface{}{"consent": consent}
	if consent == model.UserConsentAgreed {
		updates["consent_at"] = time.Now()
	}
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(ctx, userID)
}

func (s *UserService) refreshLoginInfo(ctx context.Context, user *model.User, in *LoginInput) error {
	now := time.Now()
	updates := map[string]interface{}{
		"session_key":   in.SessionKey,
		"last_login_at": now,
		"last_login_ip": in.IP,
	}
	if in.UnionID != "" {
		updates["union_id"] = in.UnionID
	}
	if in.Nickname != "" {
		updates["nickname"] = in.Nickname
	}
	if in.AvatarURL != "" {
		updates["avatar_url"] = in.AvatarURL
	}
	if in.Phone != "" {
		updates["phone"] = in.Phone
	}

	if err := s.db.WithContext(ctx).Model(user).Updates(updates).Error; err != nil {
		return err
	}

	user.SessionKey = in.SessionKey
	user.LastLoginAt = &now
	user.LastLoginIP = in.IP
	if in.UnionID != "" {
		user.UnionID = in.UnionID
	}
	if in.Nickname != "" {
		user.Nickname = in.Nickname
	}
	if in.AvatarURL != "" {
		user.AvatarURL = in.AvatarURL
	}
	if in.Phone != "" {
		user.Phone = in.Phone
	}
	return nil
}
