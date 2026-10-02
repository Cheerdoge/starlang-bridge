package server

import (
	"context"
	"strings"

	"starlang-bridge/model"
)

// MockEvaluator 本地确定性判断器，不依赖外网与模型预算，
// 用于本地联调、演示与自动化测试。
type MockEvaluator struct {
	safety *SafetyFilter
}

// NewMockEvaluator 创建 Mock 判断器
func NewMockEvaluator() *MockEvaluator {
	return &MockEvaluator{safety: NewSafetyFilter()}
}

// Name 适配器名称
func (m *MockEvaluator) Name() string { return "mock" }

// Evaluate 基于关键词与语义包含关系给出结构化判断
func (m *MockEvaluator) Evaluate(_ context.Context, req *EvalRequest) (*EvalResult, error) {
	text := strings.TrimSpace(req.Text)
	res := &EvalResult{Validity: model.ValidityUnclear, UnderstoodText: text}
	if req.Round != nil {
		res.Intent = req.Round.Intent
		res.TargetSentence = req.Round.TargetSentence
		res.ParentTip = req.Round.ParentTip
	}

	switch {
	case text == "":
		res.Validity = model.ValiditySilence
		res.ChildFeedback = "没关系，想好了再慢慢说一遍。"
		res.NextAction = model.NextRetry
	case m.safety.IsRisky(text):
		res.Validity = model.ValidityOffTopic
		res.RiskFlag = true
		res.ChildFeedback = "我们先停下来，请家长来帮忙。"
		res.NextAction = model.NextParentAssist
	case containsAny(text, req.Round):
		res.Validity = model.ValidityValid
		res.ChildFeedback = praiseFor(req.Round)
		res.NextAction = model.NextTurn
	case isPartial(text, req.Round):
		res.Validity = model.ValidityPartial
		res.ChildFeedback = "已经很接近了，把话说得更完整一点。"
		res.NextAction = model.NextRetry
	default:
		res.Validity = model.ValidityOffTopic
		res.ChildFeedback = "我们还在说刚才的事情，换个说法再试试。"
		res.NextAction = model.NextRetry
	}
	return res, nil
}

func containsAny(text string, round *model.ScenarioRound) bool {
	if round == nil {
		return false
	}
	if round.TargetSentence != "" && strings.Contains(text, strings.TrimRight(round.TargetSentence, "。！？")) {
		return true
	}
	for _, a := range round.Acceptable {
		if strings.Contains(text, strings.TrimRight(a, "。！？")) {
			return true
		}
	}
	for _, kw := range round.Keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func isPartial(text string, round *model.ScenarioRound) bool {
	if round == nil || runeLen(text) == 0 {
		return false
	}
	candidates := append([]string{}, round.Keywords...)
	candidates = append(candidates, round.Acceptable...)
	if round.TargetSentence != "" {
		candidates = append(candidates, round.TargetSentence)
	}
	for _, c := range candidates {
		c = strings.TrimRight(c, "。！？")
		if c != "" && (strings.Contains(c, text) || strings.Contains(text, c)) {
			return true
		}
	}
	return false
}

func praiseFor(round *model.ScenarioRound) string {
	if round != nil && strings.Contains(round.Intent, "need") {
		return "你说清楚了想要什么，真棒。"
	}
	return "你说得很清楚，我们一起继续。"
}
