package server

import (
	"context"
	"fmt"
	"strings"

	"starlang-bridge/model"
)

// Evaluator 大模型适配层接口
//
// 屏蔽不同模型服务（DeepSeek / 通义千问 / 本地 ASD-iLLM）的差异，
// 统一负责请求格式、超时、重试、JSON 校验与内容过滤。
type Evaluator interface {
	Evaluate(ctx context.Context, req *EvalRequest) (*EvalResult, error)
	Name() string
}

// EvalRequest 一次受限判断所需的全部上下文
type EvalRequest struct {
	Scenario   *model.Scenario
	Round      *model.ScenarioRound
	Child      *model.ChildProfile
	Text       string
	RetryCount int
}

// EvalResult 与 PRD 8.4 输出契约一致的模型返回
type EvalResult struct {
	Intent         string `json:"intent"`
	Validity       string `json:"validity"`
	UnderstoodText string `json:"understood_text"`
	ChildFeedback  string `json:"child_feedback"`
	NextAction     string `json:"next_action"`
	TargetSentence string `json:"target_sentence"`
	ParentTip      string `json:"parent_tip"`
	RiskFlag       bool   `json:"risk_flag"`
}

// NewEvaluator 按配置选择模型适配器；未配置密钥时使用本地 Mock
func NewEvaluator(cfg AIConfig) Evaluator {
	if cfg.Provider == "deepseek" && cfg.APIKey != "" {
		return NewDeepSeekEvaluator(cfg)
	}
	return NewMockEvaluator()
}

// normalizeResult 用情境配置补齐模型未返回的字段
func normalizeResult(req *EvalRequest, res *EvalResult) {
	if res.Validity == "" {
		res.Validity = model.ValidityUnclear
	}
	if !validValidity(res.Validity) {
		res.Validity = model.ValidityUnclear
	}
	if res.Intent == "" && req.Round != nil {
		res.Intent = req.Round.Intent
	}
	if strings.TrimSpace(res.UnderstoodText) == "" {
		res.UnderstoodText = req.Text
	}
	if strings.TrimSpace(res.TargetSentence) == "" && req.Round != nil {
		res.TargetSentence = req.Round.TargetSentence
	}
	if strings.TrimSpace(res.ParentTip) == "" && req.Round != nil {
		res.ParentTip = req.Round.ParentTip
	}
	if res.NextAction == "" {
		res.NextAction = model.NextRetry
	}
}

func validValidity(v string) bool {
	switch v {
	case model.ValidityValid, model.ValidityPartial, model.ValidityUnclear,
		model.ValidityOffTopic, model.ValidityRefused, model.ValiditySilence:
		return true
	default:
		return false
	}
}

const systemPrompt = `系统身份：你是面向 4 至 8 岁 ASD 儿童的语言情景训练教练。你只围绕当前情境工作，不诊断、不治疗、不评价儿童。
任务：根据当前情境、轮次、目标意图、儿童画像和 ASR 结果，判断儿童表达状态，并选择一个下一步动作。
输出：严格输出约定的 JSON 字段，不输出任何额外解释或 Markdown。
儿童反馈要求：一句话、不超过 25 个汉字、温和、具体、可执行；先肯定已有表达，再说明下一步。
安全要求：涉及诊断、用药、自伤、他伤、虐待或危机时，不自行处理，返回固定安全状态并提示家长。
判定原则：语义等价的表达视为有效；低置信度或未听清不得判为儿童故意错误；不显示"错误""失败""不配合"等标签。`

func buildUserPrompt(req *EvalRequest) string {
	var b strings.Builder
	b.WriteString("【当前情境】\n")
	if req.Scenario != nil {
		fmt.Fprintf(&b, "名称：%s\n目标：%s\n", req.Scenario.Name, req.Scenario.Goal)
	}
	if req.Round != nil {
		fmt.Fprintf(&b, "轮次：第 %d 轮\nAI 提问：%s\n目标意图：%s\n建议目标句：%s\n",
			req.Round.Round, req.Round.AIQuestion, req.Round.Intent, req.Round.TargetSentence)
		if len(req.Round.Acceptable) > 0 {
			fmt.Fprintf(&b, "可接受变体：%s\n", strings.Join(req.Round.Acceptable, " / "))
		}
		if len(req.Round.Keywords) > 0 {
			fmt.Fprintf(&b, "关键词：%s\n", strings.Join(req.Round.Keywords, "、"))
		}
	}
	b.WriteString("\n【儿童画像（最小充分字段）】\n")
	if req.Child != nil {
		fmt.Fprintf(&b, "称呼：%s\n年龄：%d\n语言水平：%s\n兴趣：%s\n提示偏好：%s\n",
			req.Child.Name, req.Child.Age, req.Child.LanguageLevel, req.Child.Interests, req.Child.PromptPreference)
	} else {
		b.WriteString("暂无画像，按 6 岁、能说短句处理。\n")
	}
	fmt.Fprintf(&b, "\n【儿童本次表达（ASR）】\n%s\n", strings.TrimSpace(req.Text))
	fmt.Fprintf(&b, "【当前重试次数】\n%d\n", req.RetryCount)

	b.WriteString(`
【输出 JSON 契约】
{
  "intent": "目标意图标签",
  "validity": "valid|partial|unclear|off_topic|refused|silence",
  "understood_text": "整理后的儿童表达",
  "child_feedback": "面向儿童的一句话反馈，不超过25个汉字",
  "next_action": "praise|retry|show_hint|parent_assist|next_turn|finish",
  "target_sentence": "当前轮建议目标句",
  "parent_tip": "仅家长可见的一条短提示",
  "risk_flag": false
}
只输出 JSON。`)
	return b.String()
}
