package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"starlang-bridge/model"
)

var (
	// ErrScenarioNotFound 情境不存在
	ErrScenarioNotFound = errors.New("情境不存在")
	// ErrSessionNotFound 训练会话不存在
	ErrSessionNotFound = errors.New("训练会话不存在")
	// ErrSessionNotActive 训练会话已结束
	ErrSessionNotActive = errors.New("训练会话已结束")
)

// DialogueService 对话状态机
//
// 状态机掌握轮次、重试、跳过与结束的控制权，大模型只负责受限判断与反馈生成。
type DialogueService struct {
	trainings *TrainingService
	scenarios *ScenarioService
	children  *ChildService
	evaluator Evaluator
	safety    *SafetyFilter
}

// NewDialogueService 创建对话状态机
func NewDialogueService(trainings *TrainingService, scenarios *ScenarioService, children *ChildService, evaluator Evaluator) *DialogueService {
	return &DialogueService{
		trainings: trainings,
		scenarios: scenarios,
		children:  children,
		evaluator: evaluator,
		safety:    NewSafetyFilter(),
	}
}

// EvaluatorName 返回当前使用的模型适配器名称
func (s *DialogueService) EvaluatorName() string { return s.evaluator.Name() }

// StartInput 开始训练参数
type StartInput struct {
	ScenarioID string `json:"scenario_id"`
}

// SessionView 开始训练时返回给儿童端的内容
type SessionView struct {
	SessionID    uint64   `json:"session_id"`
	ScenarioID   string   `json:"scenario_id"`
	ScenarioName string   `json:"scenario_name"`
	Goal         string   `json:"goal"`
	Image        string   `json:"image"`
	Status       string   `json:"status"`
	Round        int      `json:"round"`
	TotalRounds  int      `json:"total_rounds"`
	Question     string   `json:"question"`
	WordBlocks   []string `json:"word_blocks"`
	ParentTip    string   `json:"parent_tip"`
}

// AnswerInput 儿童一次回答（text 为 ASR 转写后的文本）
type AnswerInput struct {
	Text          string  `json:"text"`
	ASRConfidence float64 `json:"asr_confidence"`
}

// AnswerView 一轮判断后返回给前端的动作
type AnswerView struct {
	SessionID      uint64        `json:"session_id"`
	Status         string        `json:"status"`
	Round          int           `json:"round"`
	TotalRounds    int           `json:"total_rounds"`
	Validity       string        `json:"validity"`
	NextAction     string        `json:"next_action"`
	Feedback       string        `json:"feedback"`
	TargetSentence string        `json:"target_sentence,omitempty"`
	ParentTip      string        `json:"parent_tip,omitempty"`
	WordBlocks     []string      `json:"word_blocks,omitempty"`
	Question       string        `json:"question,omitempty"`
	RetryCount     int           `json:"retry_count"`
	RiskFlag       bool          `json:"risk_flag"`
	Summary        string        `json:"summary,omitempty"`
	TaskCard       *TaskCardView `json:"task_card,omitempty"`
}

// Start 创建一次训练并返回第一轮提问
func (s *DialogueService) Start(ctx context.Context, userID uint64, in StartInput) (*SessionView, error) {
	sc := s.resolveScenario(in.ScenarioID)
	if sc == nil {
		return nil, ErrScenarioNotFound
	}
	if len(sc.Rounds) == 0 {
		return nil, ErrScenarioNotFound
	}

	session := &model.TrainingSession{
		UserID:          userID,
		ScenarioID:      sc.ID,
		ScenarioVersion: s.scenarios.Version(),
		Status:          model.SessionStatusActive,
		CurrentRound:    1,
		TotalRounds:     len(sc.Rounds),
		StartedAt:       time.Now(),
	}
	if child, err := s.children.GetByUserID(ctx, userID); err == nil {
		session.ChildID = child.ID
	}

	if err := s.trainings.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	round := sc.Round(1)
	return &SessionView{
		SessionID:    session.ID,
		ScenarioID:   sc.ID,
		ScenarioName: sc.Name,
		Goal:         sc.Goal,
		Image:        sc.Image,
		Status:       session.Status,
		Round:        1,
		TotalRounds:  session.TotalRounds,
		Question:     round.AIQuestion,
		WordBlocks:   round.WordBlocks,
		ParentTip:    round.ParentTip,
	}, nil
}

// Answer 处理一次儿童回答，推进状态机并返回下一步动作
func (s *DialogueService) Answer(ctx context.Context, userID, sessionID uint64, in AnswerInput) (*AnswerView, error) {
	session, err := s.trainings.GetSession(ctx, sessionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	if session.Status != model.SessionStatusActive {
		return nil, ErrSessionNotActive
	}

	sc, ok := s.scenarios.Get(session.ScenarioID)
	if !ok {
		return nil, ErrScenarioNotFound
	}
	round := sc.Round(session.CurrentRound)
	if round == nil {
		return nil, ErrScenarioNotFound
	}

	text := strings.TrimSpace(in.Text)
	attempt := session.RetryCount + 1
	isLastRound := session.CurrentRound >= session.TotalRounds

	child, _ := s.children.GetByUserID(ctx, userID)
	evalReq := &EvalRequest{
		Scenario:   sc,
		Round:      round,
		Child:      child,
		Text:       text,
		RetryCount: session.RetryCount,
	}

	var result *EvalResult
	if text == "" {
		result = &EvalResult{
			Intent:         round.Intent,
			Validity:       model.ValiditySilence,
			ChildFeedback:  defaultFeedback(model.ValiditySilence),
			NextAction:     model.NextRetry,
			TargetSentence: round.TargetSentence,
			ParentTip:      round.ParentTip,
		}
	} else {
		result, err = s.evaluator.Evaluate(ctx, evalReq)
		if err != nil {
			// 模型异常降级为固定话术，绝不把控制权交给模型
			result = fallbackResult(round, text)
		}
	}
	normalizeResult(evalReq, result)

	if result.RiskFlag || s.safety.IsRisky(text) {
		return s.handleRisk(ctx, session, round, attempt, text, result)
	}

	feedback := s.safety.SanitizeFeedback(result.ChildFeedback, defaultFeedback(result.Validity))
	action := decideAction(result.Validity, session.RetryCount, isLastRound)

	s.applyAction(session, result, action, attempt)
	if err := s.trainings.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	if err := s.trainings.AddTurn(ctx, &model.TrainingTurn{
		SessionID:      session.ID,
		RoundNo:        round.Round,
		Attempt:        attempt,
		AIQuestion:     round.AIQuestion,
		ChildText:      text,
		Intent:         result.Intent,
		Validity:       result.Validity,
		ChildFeedback:  feedback,
		NextAction:     action,
		TargetSentence: result.TargetSentence,
		ParentTip:      result.ParentTip,
	}); err != nil {
		return nil, err
	}

	return s.buildAnswerView(ctx, session, sc, action, result, feedback)
}

// Abort 用户主动退出或安全中止：保存退出原因，不评价儿童
func (s *DialogueService) Abort(ctx context.Context, userID, sessionID uint64, reason string) error {
	session, err := s.trainings.GetSession(ctx, sessionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return err
	}
	if session.Status != model.SessionStatusActive {
		return nil
	}
	now := time.Now()
	session.Status = model.SessionStatusAborted
	session.ExitReason = strings.TrimSpace(reason)
	session.EndedAt = &now
	return s.trainings.SaveSession(ctx, session)
}

// Detail 查询训练会话与轮次记录
func (s *DialogueService) Detail(ctx context.Context, userID, sessionID uint64) (*model.TrainingSession, []model.TrainingTurn, error) {
	session, err := s.trainings.GetSession(ctx, sessionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSessionNotFound
		}
		return nil, nil, err
	}
	turns, err := s.trainings.ListTurns(ctx, session.ID)
	if err != nil {
		return nil, nil, err
	}
	return session, turns, nil
}

func (s *DialogueService) resolveScenario(id string) *model.Scenario {
	if strings.TrimSpace(id) == "" {
		sc, _ := s.scenarios.Default()
		return sc
	}
	sc, ok := s.scenarios.Get(id)
	if !ok {
		return nil
	}
	return sc
}

// applyAction 按下一步动作更新会话计数与轮次
func (s *DialogueService) applyAction(session *model.TrainingSession, result *EvalResult, action string, attempt int) {
	switch action {
	case model.NextShowHint:
		session.RetryCount++
		session.HintCount++
	case model.NextParentAssist:
		session.RetryCount++
		session.ParentAssistCount++
	case model.NextRetry:
		session.RetryCount++
	case model.NextTurn:
		s.countOutcome(session, result, attempt)
		session.CurrentRound++
		session.RetryCount = 0
	case model.NextFinish:
		s.countOutcome(session, result, attempt)
		session.Status = model.SessionStatusFinished
		now := time.Now()
		session.EndedAt = &now
		session.Summary = buildSummary(session)
	}
}

func (s *DialogueService) countOutcome(session *model.TrainingSession, result *EvalResult, attempt int) {
	if result.Validity == model.ValidityValid && attempt == 1 {
		session.IndependentCount++
	}
	if result.Validity == model.ValidityRefused || result.Validity == model.ValiditySilence {
		session.SkipCount++
	}
}

// buildAnswerView 组装当前轮次应展示的提问、词块与任务卡
func (s *DialogueService) buildAnswerView(ctx context.Context, session *model.TrainingSession, sc *model.Scenario, action string, result *EvalResult, feedback string) (*AnswerView, error) {
	view := &AnswerView{
		SessionID:   session.ID,
		Status:      session.Status,
		Round:       session.CurrentRound,
		TotalRounds: session.TotalRounds,
		Validity:    result.Validity,
		NextAction:  action,
		Feedback:    feedback,
		RetryCount:  session.RetryCount,
	}

	if session.Status == model.SessionStatusFinished {
		view.Summary = session.Summary
		card, err := s.createTaskCard(ctx, session, sc, result.TargetSentence)
		if err != nil {
			return nil, err
		}
		view.TaskCard = toTaskCardView(card)
		return view, nil
	}

	current := sc.Round(session.CurrentRound)
	if current != nil {
		view.Round = current.Round
		view.Question = current.AIQuestion
		view.WordBlocks = current.WordBlocks
		view.ParentTip = current.ParentTip
	}
	if action == model.NextShowHint || action == model.NextParentAssist {
		view.TargetSentence = result.TargetSentence
	}
	return view, nil
}

func (s *DialogueService) createTaskCard(ctx context.Context, session *model.TrainingSession, sc *model.Scenario, target string) (*model.TaskCard, error) {
	if strings.TrimSpace(target) == "" {
		if len(sc.TargetSentences) > 0 {
			target = sc.TargetSentences[0]
		}
	}
	card := buildTaskCard(session, sc, target)
	if err := s.trainings.CreateTaskCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

// handleRisk 高风险内容：中止自由生成，转入固定安全流程
func (s *DialogueService) handleRisk(ctx context.Context, session *model.TrainingSession, round *model.ScenarioRound, attempt int, text string, result *EvalResult) (*AnswerView, error) {
	const childSafety = "我们先停下来，请家长来帮忙。"
	const parentSafety = "检测到需要关注的内容，请按既定安全流程处理，必要时联系专业人员。"

	now := time.Now()
	session.Status = model.SessionStatusAborted
	session.ExitReason = "safety"
	session.EndedAt = &now
	if err := s.trainings.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	if err := s.trainings.AddTurn(ctx, &model.TrainingTurn{
		SessionID:      session.ID,
		RoundNo:        round.Round,
		Attempt:        attempt,
		AIQuestion:     round.AIQuestion,
		ChildText:      text,
		Intent:         result.Intent,
		Validity:       result.Validity,
		ChildFeedback:  childSafety,
		NextAction:     model.NextParentAssist,
		TargetSentence: round.TargetSentence,
		ParentTip:      parentSafety,
		RiskFlag:       true,
	}); err != nil {
		return nil, err
	}

	return &AnswerView{
		SessionID:   session.ID,
		Status:      session.Status,
		Round:       session.CurrentRound,
		TotalRounds: session.TotalRounds,
		Validity:    result.Validity,
		NextAction:  model.NextParentAssist,
		Feedback:    childSafety,
		ParentTip:   parentSafety,
		RiskFlag:    true,
	}, nil
}

func decideAction(validity string, retryCount int, isLastRound bool) string {
	switch validity {
	case model.ValidityValid, model.ValidityRefused:
		if isLastRound {
			return model.NextFinish
		}
		return model.NextTurn
	case model.ValiditySilence:
		if retryCount == 0 {
			return model.NextRetry
		}
		if isLastRound {
			return model.NextFinish
		}
		return model.NextTurn
	default: // partial / unclear / off_topic
		switch retryCount {
		case 0:
			return model.NextRetry
		case 1:
			return model.NextShowHint
		case 2:
			return model.NextParentAssist
		default:
			if isLastRound {
				return model.NextFinish
			}
			return model.NextTurn
		}
	}
}

func defaultFeedback(validity string) string {
	switch validity {
	case model.ValidityValid:
		return "你说得很清楚，我们一起继续。"
	case model.ValidityPartial:
		return "已经很接近了，把话说得更完整一点。"
	case model.ValidityUnclear:
		return "我刚才没听清，请慢慢再说一遍。"
	case model.ValidityOffTopic:
		return "我们还在说刚才的事情，换个说法再试试。"
	case model.ValidityRefused, model.ValiditySilence:
		return "没关系，想说了再告诉我。"
	default:
		return "我们再试一次。"
	}
}

func fallbackResult(round *model.ScenarioRound, text string) *EvalResult {
	res := &EvalResult{
		Validity:       model.ValidityUnclear,
		UnderstoodText: text,
		ChildFeedback:  defaultFeedback(model.ValidityUnclear),
		NextAction:     model.NextRetry,
	}
	if round != nil {
		res.Intent = round.Intent
		res.TargetSentence = round.TargetSentence
		res.ParentTip = round.ParentTip
	}
	return res
}

func buildSummary(session *model.TrainingSession) string {
	return fmt.Sprintf(
		"本次完成 %d 轮；独立表达 %d 次；显示词块 %d 次；家长辅助 %d 次；跳过 %d 次。",
		session.TotalRounds, session.IndependentCount, session.HintCount,
		session.ParentAssistCount, session.SkipCount,
	)
}
