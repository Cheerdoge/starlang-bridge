package model

import (
	"time"

	"gorm.io/gorm"
)

// 训练会话状态
const (
	SessionStatusActive   = "active"   // 进行中
	SessionStatusFinished = "finished" // 已完成
	SessionStatusAborted  = "aborted"  // 暂停/安全中止
)

// 儿童表达有效性判断
const (
	ValidityValid    = "valid"     // 有效
	ValidityPartial  = "partial"   // 部分有效
	ValidityUnclear  = "unclear"   // 未听清 / 低置信度
	ValidityOffTopic = "off_topic" // 答非所问
	ValidityRefused  = "refused"   // 拒绝
	ValiditySilence  = "silence"   // 沉默
)

// 下一步动作
const (
	NextPraise       = "praise"        // 肯定并进入下一轮
	NextRetry        = "retry"         // 重说
	NextShowHint     = "show_hint"     // 显示词块
	NextParentAssist = "parent_assist" // 家长辅助
	NextTurn         = "next_turn"     // 进入下一轮
	NextFinish       = "finish"        // 结束训练
)

// 任务卡状态
const (
	TaskStatusPending int8 = 0 // 待完成
	TaskStatusDone    int8 = 1 // 已回填
)

// 任务卡回填选项
const (
	TaskFeedbackIndependent int8 = 1 // 独立完成
	TaskFeedbackHinted      int8 = 2 // 提示后完成
	TaskFeedbackAssisted    int8 = 3 // 家长代答
	TaskFeedbackIncomplete  int8 = 4 // 未完成
	TaskFeedbackNotApplica  int8 = 5 // 暂不适用
)

// TrainingSession 一次完整训练（儿童端一次进入训练）
type TrainingSession struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint64 `gorm:"column:user_id;index:idx_session_user;not null" json:"user_id"`
	ChildID         uint64 `gorm:"column:child_id;index:idx_session_child" json:"child_id"`
	ScenarioID      string `gorm:"column:scenario_id;type:varchar(64);not null" json:"scenario_id"`
	ScenarioVersion string `gorm:"column:scenario_version;type:varchar(32)" json:"scenario_version"`

	Status       string `gorm:"column:status;type:varchar(16);not null;default:active" json:"status"`
	CurrentRound int    `gorm:"column:current_round;default:1" json:"current_round"`
	TotalRounds  int    `gorm:"column:total_rounds" json:"total_rounds"`
	RetryCount   int    `gorm:"column:retry_count" json:"retry_count"`

	IndependentCount  int    `gorm:"column:independent_count" json:"independent_count"`
	HintCount         int    `gorm:"column:hint_count" json:"hint_count"`
	ParentAssistCount int    `gorm:"column:parent_assist_count" json:"parent_assist_count"`
	SkipCount         int    `gorm:"column:skip_count" json:"skip_count"`
	Summary           string `gorm:"column:summary;type:varchar(512)" json:"summary"`
	ExitReason        string `gorm:"column:exit_reason;type:varchar(64)" json:"exit_reason"`

	StartedAt time.Time      `gorm:"column:started_at" json:"started_at"`
	EndedAt   *time.Time     `gorm:"column:ended_at" json:"ended_at"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index:idx_session_deleted_at" json:"-"`
}

// TableName 指定表名
func (TrainingSession) TableName() string {
	return "training_sessions"
}

// TrainingTurn 一次儿童回答的记录（用于训练总结，不保存原始录音）
type TrainingTurn struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID uint64 `gorm:"column:session_id;index:idx_turn_session;not null" json:"session_id"`
	RoundNo   int    `gorm:"column:round_no" json:"round_no"`
	Attempt   int    `gorm:"column:attempt" json:"attempt"`

	AIQuestion     string `gorm:"column:ai_question;type:varchar(255)" json:"ai_question"`
	ChildText      string `gorm:"column:child_text;type:varchar(255)" json:"child_text"` // ASR 整理后的文本
	Intent         string `gorm:"column:intent;type:varchar(64)" json:"intent"`
	Validity       string `gorm:"column:validity;type:varchar(16)" json:"validity"`
	ChildFeedback  string `gorm:"column:child_feedback;type:varchar(255)" json:"child_feedback"`
	NextAction     string `gorm:"column:next_action;type:varchar(24)" json:"next_action"`
	TargetSentence string `gorm:"column:target_sentence;type:varchar(255)" json:"target_sentence"`
	ParentTip      string `gorm:"column:parent_tip;type:varchar(255)" json:"parent_tip"`
	RiskFlag       bool   `gorm:"column:risk_flag;default:false" json:"risk_flag"`

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName 指定表名
func (TrainingTurn) TableName() string {
	return "training_turns"
}

// TaskCard 家庭任务卡
type TaskCard struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64 `gorm:"column:user_id;index:idx_task_user;not null" json:"user_id"`
	ChildID    uint64 `gorm:"column:child_id" json:"child_id"`
	SessionID  uint64 `gorm:"column:session_id;index:idx_task_session" json:"session_id"`
	ScenarioID string `gorm:"column:scenario_id;type:varchar(64)" json:"scenario_id"`

	Name           string `gorm:"column:name;type:varchar(128)" json:"name"`
	TargetSentence string `gorm:"column:target_sentence;type:varchar(255)" json:"target_sentence"`
	Scene          string `gorm:"column:scene;type:varchar(255)" json:"scene"`
	Steps          string `gorm:"column:steps;type:text" json:"steps"` // JSON 数组
	Observation    string `gorm:"column:observation;type:varchar(255)" json:"observation"`

	Status   int8   `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	Feedback int8   `gorm:"column:feedback;type:tinyint;default:0" json:"feedback"`
	Note     string `gorm:"column:note;type:varchar(500)" json:"note"`

	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index:idx_task_deleted_at" json:"-"`
}

// TableName 指定表名
func (TaskCard) TableName() string {
	return "task_cards"
}
