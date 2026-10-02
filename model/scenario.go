package model

// ScenarioRound 单个轮次的受控对话配置
type ScenarioRound struct {
	Round          int      `json:"round"`
	AIQuestion     string   `json:"ai_question"`
	Intent         string   `json:"intent"`
	TargetSentence string   `json:"target_sentence"`
	WordBlocks     []string `json:"word_blocks"`
	Acceptable     []string `json:"acceptable"`
	Keywords       []string `json:"keywords"`
	ParentTip      string   `json:"parent_tip"`
}

// ScenarioTaskCard 场景对应的家庭任务卡模板
type ScenarioTaskCard struct {
	Name        string   `json:"name"`
	Scene       string   `json:"scene"`
	Steps       []string `json:"steps"`
	Observation string   `json:"observation"`
}

// Scenario 一个结构化社会情境
type Scenario struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Goal            string           `json:"goal"`
	Image           string           `json:"image"`
	TargetSentences []string         `json:"target_sentences"`
	Rounds          []ScenarioRound  `json:"rounds"`
	TaskCard        ScenarioTaskCard `json:"task_card"`
}

// Round 返回指定轮次的配置，未找到返回 nil
func (s *Scenario) Round(no int) *ScenarioRound {
	for i := range s.Rounds {
		if s.Rounds[i].Round == no {
			return &s.Rounds[i]
		}
	}
	return nil
}

// ScenarioConfig 全部情境配置（来自后台 JSON 数据文件）
type ScenarioConfig struct {
	Version   string     `json:"version"`
	Scenarios []Scenario `json:"scenarios"`
}

// Find 按 ID 查找情境
func (c *ScenarioConfig) Find(id string) *Scenario {
	for i := range c.Scenarios {
		if c.Scenarios[i].ID == id {
			return &c.Scenarios[i]
		}
	}
	return nil
}
