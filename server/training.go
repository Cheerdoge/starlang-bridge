package server

import (
	"context"

	"gorm.io/gorm"

	"starlang-bridge/model"
)

// TrainingService 负责训练会话、轮次记录与家庭任务卡的持久化
type TrainingService struct {
	db *gorm.DB
}

// NewTrainingService 创建训练记录服务
func NewTrainingService(db *gorm.DB) *TrainingService {
	return &TrainingService{db: db}
}

// CreateSession 新建训练会话
func (s *TrainingService) CreateSession(ctx context.Context, session *model.TrainingSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

// GetSession 按 ID 与用户查询会话（校验归属）
func (s *TrainingService) GetSession(ctx context.Context, id, userID uint64) (*model.TrainingSession, error) {
	var session model.TrainingSession
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// SaveSession 保存会话
func (s *TrainingService) SaveSession(ctx context.Context, session *model.TrainingSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

// AddTurn 追加一条轮次记录
func (s *TrainingService) AddTurn(ctx context.Context, turn *model.TrainingTurn) error {
	return s.db.WithContext(ctx).Create(turn).Error
}

// ListTurns 查询会话下的全部轮次记录
func (s *TrainingService) ListTurns(ctx context.Context, sessionID uint64) ([]model.TrainingTurn, error) {
	var turns []model.TrainingTurn
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("id asc").
		Find(&turns).Error; err != nil {
		return nil, err
	}
	return turns, nil
}

// CreateTaskCard 生成家庭任务卡
func (s *TrainingService) CreateTaskCard(ctx context.Context, card *model.TaskCard) error {
	return s.db.WithContext(ctx).Create(card).Error
}

// ListTaskCards 查询家长的任务卡
func (s *TrainingService) ListTaskCards(ctx context.Context, userID uint64, limit int) ([]model.TaskCard, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var cards []model.TaskCard
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id desc").
		Limit(limit).
		Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

// GetTaskCard 按 ID 与用户查询任务卡
func (s *TrainingService) GetTaskCard(ctx context.Context, id, userID uint64) (*model.TaskCard, error) {
	var card model.TaskCard
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&card).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

// UpdateTaskCard 更新任务卡（用于家长回填）
func (s *TrainingService) UpdateTaskCard(ctx context.Context, card *model.TaskCard) error {
	return s.db.WithContext(ctx).Save(card).Error
}
