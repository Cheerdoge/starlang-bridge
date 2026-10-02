package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"starlang-bridge/server"
)

// ChildHandler 儿童最小画像（家长端填写）
type ChildHandler struct {
	children *server.ChildService
}

// NewChildHandler 创建画像处理器
func NewChildHandler(children *server.ChildService) *ChildHandler {
	return &ChildHandler{children: children}
}

// Get 获取当前家长账户下的儿童画像
func (h *ChildHandler) Get(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}

	child, err := h.children.GetByUserID(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			OK(c, gin.H{"child": nil})
			return
		}
		Fail(c, http.StatusInternalServerError, CodeInternal, "查询画像失败")
		return
	}
	OK(c, gin.H{"child": child})
}

// Upsert 创建或更新儿童画像
func (h *ChildHandler) Upsert(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}

	var in server.ChildInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "参数错误："+err.Error())
		return
	}

	child, err := h.children.Upsert(c.Request.Context(), uid, &in)
	if err != nil {
		Fail(c, http.StatusInternalServerError, CodeInternal, "保存画像失败")
		return
	}
	OK(c, gin.H{"child": child})
}
