package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeepSeekEvaluator 调用 DeepSeek Chat Completions 完成受限判断
type DeepSeekEvaluator struct {
	baseURL     string
	apiKey      string
	model       string
	temperature float64
	maxRetries  int
	http        *http.Client
}

// NewDeepSeekEvaluator 创建 DeepSeek 适配器
func NewDeepSeekEvaluator(cfg AIConfig) *DeepSeekEvaluator {
	return &DeepSeekEvaluator{
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		maxRetries:  cfg.MaxRetries,
		http:        &http.Client{Timeout: cfg.Timeout},
	}
}

// Name 适配器名称
func (e *DeepSeekEvaluator) Name() string { return "deepseek:" + e.model }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
	ResponseFormat map[string]any `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Evaluate 调用模型并解析结构化 JSON 结果
func (e *DeepSeekEvaluator) Evaluate(ctx context.Context, req *EvalRequest) (*EvalResult, error) {
	payload := chatRequest{
		Model: e.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildUserPrompt(req)},
		},
		Temperature:    e.temperature,
		MaxTokens:      512,
		ResponseFormat: map[string]any{"type": "json_object"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		content, err := e.doRequest(ctx, body)
		if err != nil {
			lastErr = err
			continue
		}

		var res EvalResult
		if err := json.Unmarshal([]byte(extractJSON(content)), &res); err != nil {
			lastErr = fmt.Errorf("解析模型 JSON 失败: %w", err)
			continue
		}
		normalizeResult(req, &res)
		return &res, nil
	}
	return nil, lastErr
}

func (e *DeepSeekEvaluator) doRequest(ctx context.Context, body []byte) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.http.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("模型服务返回 %d: %s", resp.StatusCode, string(raw))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("解析模型响应失败: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("模型服务错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("模型返回为空")
	}
	return parsed.Choices[0].Message.Content, nil
}

// extractJSON 容错地截取模型输出中的 JSON 对象
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
