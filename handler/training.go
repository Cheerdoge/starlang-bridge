package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"starlang-bridge/server"
)

// TrainingHandler 训练对话接口
type TrainingHandler struct {
	dialogue *server.DialogueService
}

// NewTrainingHandler 创建训练处理器
func NewTrainingHandler(dialogue *server.DialogueService) *TrainingHandler {
	return &TrainingHandler{dialogue: dialogue}
}

// Start 开始一次训练
func (h *TrainingHandler) Start(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}

	var in server.StartInput
	if err := c.ShouldBindJSON(&in); err != nil && !errors.Is(err, io.EOF) {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "参数错误")
		return
	}

	view, err := h.dialogue.Start(c.Request.Context(), uid, in)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	OK(c, view)
}

// Answer 提交儿童一次回答，获取下一步动作
func (h *TrainingHandler) Answer(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}
	sessionID, ok := parseIDParam(c)
	if !ok {
		return
	}

	var in server.AnswerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "参数错误：text 字段不合法")
		return
	}

	view, err := h.dialogue.Answer(c.Request.Context(), uid, sessionID, in)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	OK(c, view)
}

// Abort 暂停 / 退出当前训练
func (h *TrainingHandler) Abort(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}
	sessionID, ok := parseIDParam(c)
	if !ok {
		return
	}

	var in struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&in)

	if err := h.dialogue.Abort(c.Request.Context(), uid, sessionID, in.Reason); err != nil {
		mapServiceError(c, err)
		return
	}
	OK(c, gin.H{"status": "aborted"})
}

// Detail 查询训练会话与轮次记录
func (h *TrainingHandler) Detail(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}
	sessionID, ok := parseIDParam(c)
	if !ok {
		return
	}

	session, turns, err := h.dialogue.Detail(c.Request.Context(), uid, sessionID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	OK(c, gin.H{
		"session": session,
		"turns":   turns,
	})
}

func parseIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "无效的 ID")
		return 0, false
	}
	return id, true
}
