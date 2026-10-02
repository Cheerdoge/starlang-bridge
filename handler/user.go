package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"starlang-bridge/model"
	"starlang-bridge/server"
)

// UserHandler 处理家长账户相关接口
type UserHandler struct {
	users *server.UserService
	wx    *server.WeChatClient
	jwt   *server.JWTManager
}

// NewUserHandler 创建用户接口处理器
func NewUserHandler(users *server.UserService, wx *server.WeChatClient, jwt *server.JWTManager) *UserHandler {
	return &UserHandler{users: users, wx: wx, jwt: jwt}
}

type loginRequest struct {
	Code      string `json:"code" binding:"required"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Phone     string `json:"phone"`
}

type loginResponse struct {
	Token       string      `json:"token"`
	ExpiresIn   int64       `json:"expires_in"`
	NeedConsent bool        `json:"need_consent"`
	User        *model.User `json:"user"`
}

// Login 微信小程序登录
//
// 前端调用 wx.login 获取 code 后传入；后端换取 openid，
// 完成注册或续登，并签发本系统登录令牌。
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "参数错误：code 不能为空")
		return
	}

	sess, err := h.wx.Code2Session(c.Request.Context(), req.Code)
	if err != nil {
		var wxErr *server.WeChatError
		if errors.As(err, &wxErr) {
			Fail(c, http.StatusBadRequest, CodeWeChatLoginFail, "微信登录失败："+wxErr.Msg)
			return
		}
		Fail(c, http.StatusBadGateway, CodeWeChatLoginFail, "微信服务暂不可用，请稍后重试")
		return
	}

	user, err := h.users.LoginOrRegister(c.Request.Context(), &server.LoginInput{
		OpenID:     sess.OpenID,
		UnionID:    sess.UnionID,
		SessionKey: sess.SessionKey,
		Nickname:   req.Nickname,
		AvatarURL:  req.AvatarURL,
		Phone:      req.Phone,
		IP:         c.ClientIP(),
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, CodeInternal, "登录失败，请稍后重试")
		return
	}

	if user.Status != model.UserStatusNormal {
		Fail(c, http.StatusForbidden, CodeUserDisabled, "账户不可用，请联系客服")
		return
	}

	token, expiresIn, err := h.jwt.Generate(user.ID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, CodeInternal, "签发令牌失败")
		return
	}

	OK(c, loginResponse{
		Token:       token,
		ExpiresIn:   expiresIn,
		NeedConsent: !user.IsConsentAgreed(),
		User:        user,
	})
}

// Me 获取当前登录家长信息
func (h *UserHandler) Me(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, http.StatusNotFound, CodeNotFound, "用户不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, CodeInternal, "查询用户失败")
		return
	}

	OK(c, gin.H{
		"need_consent": !user.IsConsentAgreed(),
		"user":         user,
	})
}

type consentRequest struct {
	Consent int8 `json:"consent" binding:"oneof=1 2"`
}

// UpdateConsent 更新知情同意状态（1=同意，2=撤回）
func (h *UserHandler) UpdateConsent(c *gin.Context) {
	uid, ok := CurrentUserID(c)
	if !ok {
		Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
		return
	}

	var req consentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "参数错误：consent 仅支持 1(同意) 或 2(撤回)")
		return
	}

	user, err := h.users.UpdateConsent(c.Request.Context(), uid, req.Consent)
	if err != nil {
		Fail(c, http.StatusInternalServerError, CodeInternal, "更新同意状态失败")
		return
	}

	OK(c, gin.H{
		"need_consent": !user.IsConsentAgreed(),
		"user":         user,
	})
}
