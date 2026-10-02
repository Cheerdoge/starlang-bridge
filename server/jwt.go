package server

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken 令牌无效或已过期
var ErrInvalidToken = errors.New("无效的登录令牌")

// Claims 自定义 JWT 载荷，仅存放去标识化的用户 ID
type Claims struct {
	UserID uint64 `json:"uid"`
	jwt.RegisteredClaims
}

// JWTManager 负责登录令牌的签发与校验
type JWTManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewJWTManager 创建令牌管理器
func NewJWTManager(cfg JWTConfig) *JWTManager {
	return &JWTManager{
		secret: []byte(cfg.Secret),
		issuer: cfg.Issuer,
		ttl:    cfg.TTL,
	}
}

// Generate 为指定用户签发令牌，返回令牌与有效期秒数
func (m *JWTManager) Generate(userID uint64) (string, int64, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   "user",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}
	return signed, int64(m.ttl.Seconds()), nil
}

// Parse 校验令牌并返回载荷
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
