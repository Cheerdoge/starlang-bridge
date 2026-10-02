package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"starlang-bridge/server"
)

const ctxUserIDKey = "auth_user_id"

// AuthMiddleware 校验 Bearer 令牌，并将用户 ID 写入上下文
func AuthMiddleware(jwtManager *server.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			Fail(c, http.StatusUnauthorized, CodeUnauthorized, "未登录")
			c.Abort()
			return
		}

		claims, err := jwtManager.Parse(token)
		if err != nil {
			Fail(c, http.StatusUnauthorized, CodeUnauthorized, "登录已失效，请重新登录")
			c.Abort()
			return
		}

		c.Set(ctxUserIDKey, claims.UserID)
		c.Next()
	}
}

// CurrentUserID 读取上下文中的用户 ID
func CurrentUserID(c *gin.Context) (uint64, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	uid, ok := v.(uint64)
	return uid, ok
}

func extractToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(header)
}
