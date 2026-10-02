package server

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"starlang-bridge/model"
)

// ChildService 负责儿童最小画像的读写
type ChildService struct {
	db *gorm.DB
}

// NewChildService 创建儿童画像服务
func NewChildService(db *gorm.DB) *ChildService {
	return &ChildService{db: db}
}

// ChildInput 画像写入参数
type ChildInput struct {
	Name             string `json:"name" binding:"required"`
	Age              int    `json:"age" binding:"gte=1,lte=18"`
	LanguageLevel    string `json:"language_level"`
	Interests        string `json:"interests"`
	PromptPreference string `json:"prompt_preference"`
	WaitSeconds      int    `json:"wait_seconds"`
	SensorySensitive string `json:"sensory_sensitive"`
	StopSignal       string `json:"stop_signal"`
	RiskNote         string `json:"risk_note"`
}

// GetByUserID 查询家长账户下的儿童画像
func (s *ChildService) GetByUserID(ctx context.Context, userID uint64) (*model.ChildProfile, error) {
	var child model.ChildProfile
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&child).Error; err != nil {
		return nil, err
	}
	return &child, nil
}

// Upsert 创建或更新儿童画像
func (s *ChildService) Upsert(ctx context.Context, userID uint64, in *ChildInput) (*model.ChildProfile, error) {
	wait := in.WaitSeconds
	if wait <= 0 {
		wait = 5
	}

	child, err := s.GetByUserID(ctx, userID)
	if err == nil {
		updates := map[string]interface{}{
			"name":              in.Name,
			"age":               in.Age,
			"language_level":    in.LanguageLevel,
			"interests":         in.Interests,
			"prompt_preference": in.PromptPreference,
			"wait_seconds":      wait,
			"sensory_sensitive": in.SensorySensitive,
			"stop_signal":       in.StopSignal,
			"risk_note":         in.RiskNote,
		}
		if err := s.db.WithContext(ctx).Model(child).Updates(updates).Error; err != nil {
			return nil, err
		}
		return s.GetByUserID(ctx, userID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	child = &model.ChildProfile{
		UserID:           userID,
		Name:             in.Name,
		Age:              in.Age,
		LanguageLevel:    in.LanguageLevel,
		Interests:        in.Interests,
		PromptPreference: in.PromptPreference,
		WaitSeconds:      wait,
		SensorySensitive: in.SensorySensitive,
		StopSignal:       in.StopSignal,
		RiskNote:         in.RiskNote,
	}
	if err := s.db.WithContext(ctx).Create(child).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return s.GetByUserID(ctx, userID)
		}
		return nil, err
	}
	return child, nil
}
