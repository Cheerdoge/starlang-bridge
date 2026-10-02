package server

import (
	"encoding/json"

	"starlang-bridge/model"
)

// TaskCardView 返回给家长端的任务卡
type TaskCardView struct {
	ID             uint64   `json:"id"`
	Name           string   `json:"name"`
	TargetSentence string   `json:"target_sentence"`
	Scene          string   `json:"scene"`
	Steps          []string `json:"steps"`
	Observation    string   `json:"observation"`
	Status         int8     `json:"status"`
	Feedback       int8     `json:"feedback"`
	Note           string   `json:"note"`
}

// buildTaskCard 基于固定模板和训练结果生成任务卡（不让模型自由创造）
func buildTaskCard(session *model.TrainingSession, sc *model.Scenario, targetSentence string) *model.TaskCard {
	steps, _ := json.Marshal(sc.TaskCard.Steps)
	return &model.TaskCard{
		UserID:         session.UserID,
		ChildID:        session.ChildID,
		SessionID:      session.ID,
		ScenarioID:     sc.ID,
		Name:           sc.TaskCard.Name,
		TargetSentence: targetSentence,
		Scene:          sc.TaskCard.Scene,
		Steps:          string(steps),
		Observation:    sc.TaskCard.Observation,
		Status:         model.TaskStatusPending,
	}
}

// toTaskCardView 转换任务卡视图
func toTaskCardView(card *model.TaskCard) *TaskCardView {
	steps := []string{}
	if card.Steps != "" {
		_ = json.Unmarshal([]byte(card.Steps), &steps)
	}
	return &TaskCardView{
		ID:             card.ID,
		Name:           card.Name,
		TargetSentence: card.TargetSentence,
		Scene:          card.Scene,
		Steps:          steps,
		Observation:    card.Observation,
		Status:         card.Status,
		Feedback:       card.Feedback,
		Note:           card.Note,
	}
}
