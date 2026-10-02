package server

import (
	"strings"
	"unicode"
)

// SafetyFilter 负责对话安全校验：高风险关键词识别与 AI 反馈内容过滤
type SafetyFilter struct {
	riskKeywords      []string
	feedbackForbidden []string
	maxFeedbackRunes  int
}

// NewSafetyFilter 创建安全过滤器
func NewSafetyFilter() *SafetyFilter {
	return &SafetyFilter{
		riskKeywords: []string{
			"自杀", "自伤", "不想活", "去死", "打死", "打人", "咬人", "掐人",
			"虐待", "吃药", "服药", "用药", "诊断", "孤独症", "自闭症", "精神",
		},
		feedbackForbidden: []string{
			"错误", "失败", "笨", "不配合", "诊断", "治疗", "康复", "孤独症",
			"自闭症", "吃药", "药", "聪明", "落后", "差",
		},
		maxFeedbackRunes: 30,
	}
}

// IsRisky 判断儿童表达是否触发高风险规则
func (f *SafetyFilter) IsRisky(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return false
	}
	for _, kw := range f.riskKeywords {
		if strings.Contains(t, kw) {
			return true
		}
	}
	return false
}

// SanitizeFeedback 校验 AI 生成的儿童反馈：
// 含禁用词或超长时，退回固定安全话术。
func (f *SafetyFilter) SanitizeFeedback(feedback, fallback string) string {
	fb := strings.TrimSpace(feedback)
	if fb == "" {
		return fallback
	}
	for _, kw := range f.feedbackForbidden {
		if strings.Contains(fb, kw) {
			return fallback
		}
	}
	if runeLen(fb) > f.maxFeedbackRunes {
		return fallback
	}
	return fb
}

func runeLen(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		n++
	}
	return n
}
