package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"starlang-bridge/server"
)

// mapServiceError 将服务层错误映射为统一响应
func mapServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, server.ErrScenarioNotFound):
		Fail(c, http.StatusNotFound, CodeNotFound, "情境不存在")
	case errors.Is(err, server.ErrSessionNotFound):
		Fail(c, http.StatusNotFound, CodeNotFound, "训练会话不存在")
	case errors.Is(err, server.ErrSessionNotActive):
		Fail(c, http.StatusBadRequest, CodeBadRequest, "训练会话已结束")
	default:
		Fail(c, http.StatusInternalServerError, CodeInternal, "服务异常，请稍后重试")
	}
}
