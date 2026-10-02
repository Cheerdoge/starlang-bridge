package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const wxCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"

// WeChatSession 微信登录凭证校验结果
type WeChatSession struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// WeChatError 微信接口返回的业务错误
type WeChatError struct {
	Code int
	Msg  string
}

func (e *WeChatError) Error() string {
	return fmt.Sprintf("微信接口错误(%d): %s", e.Code, e.Msg)
}

// WeChatClient 微信小程序服务端接口客户端
type WeChatClient struct {
	appID     string
	appSecret string
	mock      bool
	http      *http.Client
}

// NewWeChatClient 创建微信客户端
func NewWeChatClient(cfg WeChatConfig) *WeChatClient {
	return &WeChatClient{
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
		mock:      cfg.Mock,
		http:      &http.Client{Timeout: 5 * time.Second},
	}
}

// Code2Session 用小程序 wx.login 返回的 code 换取 openid / session_key
func (c *WeChatClient) Code2Session(ctx context.Context, code string) (*WeChatSession, error) {
	if code == "" {
		return nil, &WeChatError{Code: 40029, Msg: "缺少 code"}
	}

	if c.mock {
		return &WeChatSession{
			OpenID:     "mock_openid_" + code,
			UnionID:    "",
			SessionKey: "mock_session_key",
		}, nil
	}

	if c.appID == "" || c.appSecret == "" {
		return nil, fmt.Errorf("未配置 WECHAT_APP_ID / WECHAT_APP_SECRET")
	}

	query := url.Values{}
	query.Set("appid", c.appID)
	query.Set("secret", c.appSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wxCode2SessionURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求微信服务器失败: %w", err)
	}
	defer resp.Body.Close()

	var sess WeChatSession
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return nil, fmt.Errorf("解析微信响应失败: %w", err)
	}

	if sess.ErrCode != 0 || sess.OpenID == "" {
		return nil, &WeChatError{Code: sess.ErrCode, Msg: sess.ErrMsg}
	}
	return &sess, nil
}
