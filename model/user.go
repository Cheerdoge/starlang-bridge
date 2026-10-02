package model

import (
	"time"

	"gorm.io/gorm"
)

// 用户状态
const (
	UserStatusDisabled int8 = 0 // 已停用
	UserStatusNormal   int8 = 1 // 正常
)

// 知情同意状态
const (
	UserConsentPending  int8 = 0 // 未确认
	UserConsentAgreed   int8 = 1 // 已确认
	UserConsentWithdraw int8 = 2 // 已撤回
)

// User 家长 / 监护人账户
//
// 本产品为微信小程序，家长（监护人）是账户和知情同意的责任人，
// 儿童数据通过去标识化 ID（Child 表）与家长账户关联。
// 该表不存储任何诊断信息与原始录音。
type User struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 微信身份标识
	OpenID     string `gorm:"column:open_id;type:varchar(64);uniqueIndex:uk_users_open_id;not null" json:"-"`
	UnionID    string `gorm:"column:union_id;type:varchar(64);index:idx_users_union_id" json:"-"`
	SessionKey string `gorm:"column:session_key;type:varchar(128)" json:"-"`

	// 基础资料
	Nickname  string `gorm:"column:nickname;type:varchar(64)" json:"nickname"`
	AvatarURL string `gorm:"column:avatar_url;type:varchar(512)" json:"avatar_url"`
	Phone     string `gorm:"column:phone;type:varchar(32);index:idx_users_phone" json:"phone"`

	// 状态与合规
	Status    int8       `gorm:"column:status;type:tinyint;not null;default:1" json:"status"`
	Consent   int8       `gorm:"column:consent;type:tinyint;not null;default:0" json:"consent"`
	ConsentAt *time.Time `gorm:"column:consent_at" json:"consent_at"`

	// 登录轨迹
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	LastLoginIP string     `gorm:"column:last_login_ip;type:varchar(45)" json:"-"`

	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index:idx_users_deleted_at" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// IsConsentAgreed 判断家长是否已完成知情同意
func (u *User) IsConsentAgreed() bool {
	return u.Consent == UserConsentAgreed
}
