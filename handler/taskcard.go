package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"starlang-bridge/model"
	"starlang-bridge/server"
)

// TaskCardHandler 家庭任务卡接口（家长端）
type TaskCardHandler struct {
	trainings *server.TrainingService
}

// NewTaskCardHandler 创建任务卡处理器
func NewTaskCardHandler(trainings *server.TrainingService) *TaskCardHandler {
	return &TaskCardHandler{trainings: trainings}
}

// List 查询最近的任务卡
func (h *TaskCardHandler) List(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}

	cards, err := h.trainings.ListTaskCards(c.Request.Context(), uid, 10)
	if err != nil {
		Fail(c, http.StatusInternalServerError, CodeInternal, "查询任务卡失败")
		return
	}
	OK(c, gin.H{"task_cards": cards})
}

type backfillRequest struct {
	Feedback int8   `json:"feedback" binding:"required,oneof=1 2 3 4 5"`
	Note     string `json:"note" binding:"max=500"`
}

// Backfill 家长回填任务完成情况
func (h *TaskCardHandler) Backfill(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}
	cardID, ok := parseIDParam(c)
	if !ok {
		return
	}

	var in backfillRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "参数错误：feedback 仅支持 1~5")
		return
	}

	card, err := h.trainings.GetTaskCard(c.Request.Context(), cardID, uid)
	if err != nil {
		Fail(c, http.StatusNotFound, CodeNotFound, "任务卡不存在")
		return
	}

	card.Status = model.TaskStatusDone
	card.Feedback = in.Feedback
	card.Note = in.Note
	if err := h.trainings.UpdateTaskCard(c.Request.Context(), card); err != nil {
		Fail(c, http.StatusInternalServerError, CodeInternal, "回填失败")
		return
	}
	OK(c, card)
}
