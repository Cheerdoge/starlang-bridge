package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"starlang-bridge/server"
)

// ScenarioHandler 提供情境配置
type ScenarioHandler struct {
	scenarios *server.ScenarioService
}

// NewScenarioHandler 创建情境处理器
func NewScenarioHandler(scenarios *server.ScenarioService) *ScenarioHandler {
	return &ScenarioHandler{scenarios: scenarios}
}

// List 返回全部情境概要
func (h *ScenarioHandler) List(c *gin.Context) {
	OK(c, gin.H{
		"version":   h.scenarios.Version(),
		"scenarios": h.scenarios.ListBrief(),
	})
}

// Get 返回单个情境详情
func (h *ScenarioHandler) Get(c *gin.Context) {
	sc, ok := h.scenarios.Get(c.Param("id"))
	if !ok {
		Fail(c, http.StatusNotFound, CodeNotFound, "情境不存在")
		return
	}
	OK(c, sc)
}
