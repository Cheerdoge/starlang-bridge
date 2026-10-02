package model

import (
	"time"

	"gorm.io/gorm"
)

// ChildProfile 儿童最小画像（NCP 最小充分字段）
//
// MVP 不做多儿童账号，一个家长账户对应一份儿童画像。
// 画像只用于调整语速、句长、兴趣示例和提示策略，不存放诊断标签。
type ChildProfile struct {
	ID     uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"column:user_id;uniqueIndex:uk_child_user;not null" json:"user_id"`

	Name             string `gorm:"column:name;type:varchar(32)" json:"name"`                           // 称呼
	Age              int    `gorm:"column:age" json:"age"`                                              // 年龄
	LanguageLevel    string `gorm:"column:language_level;type:varchar(32)" json:"language_level"`       // 单字/短句、能说完整句
	Interests        string `gorm:"column:interests;type:varchar(255)" json:"interests"`                // 兴趣，逗号分隔
	PromptPreference string `gorm:"column:prompt_preference;type:varchar(32)" json:"prompt_preference"` // 图片优先/语言优先
	WaitSeconds      int    `gorm:"column:wait_seconds;default:5" json:"wait_seconds"`                  // 等待时间
	SensorySensitive string `gorm:"column:sensory_sensitive;type:varchar(255)" json:"sensory_sensitive"`
	StopSignal       string `gorm:"column:stop_signal;type:varchar(128)" json:"stop_signal"` // 停止信号
	RiskNote         string `gorm:"column:risk_note;type:varchar(255)" json:"risk_note"`     // 风险提示（仅用于安全流程）

	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index:idx_child_deleted_at" json:"-"`
}

// TableName 指定表名
func (ChildProfile) TableName() string {
	return "child_profiles"
}
