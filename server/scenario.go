package server

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"starlang-bridge/model"
)

// ScenarioService 加载并缓存后台情境配置
type ScenarioService struct {
	mu      sync.RWMutex
	version string
	byID    map[string]*model.Scenario
	order   []string
}

// LoadScenarios 从 JSON 数据文件加载情境配置
func LoadScenarios(path string) (*ScenarioService, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取情境配置失败(%s): %w", path, err)
	}

	var cfg model.ScenarioConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("解析情境配置失败: %w", err)
	}
	if len(cfg.Scenarios) == 0 {
		return nil, fmt.Errorf("情境配置为空: %s", path)
	}

	svc := &ScenarioService{
		version: cfg.Version,
		byID:    make(map[string]*model.Scenario, len(cfg.Scenarios)),
		order:   make([]string, 0, len(cfg.Scenarios)),
	}
	for i := range cfg.Scenarios {
		sc := cfg.Scenarios[i]
		if sc.ID == "" {
			return nil, fmt.Errorf("情境缺少 id")
		}
		svc.byID[sc.ID] = &cfg.Scenarios[i]
		svc.order = append(svc.order, sc.ID)
	}
	return svc, nil
}

// Version 返回配置版本
func (s *ScenarioService) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// List 按配置顺序返回全部情境
func (s *ScenarioService) List() []model.Scenario {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.Scenario, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, *s.byID[id])
	}
	return out
}

// ListBrief 返回情境概要（不含轮次细节）
func (s *ScenarioService) ListBrief() []ScenarioBrief {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ScenarioBrief, 0, len(s.order))
	for _, id := range s.order {
		sc := s.byID[id]
		out = append(out, ScenarioBrief{
			ID:    sc.ID,
			Name:  sc.Name,
			Goal:  sc.Goal,
			Image: sc.Image,
		})
	}
	return out
}

// Get 按 ID 获取情境
func (s *ScenarioService) Get(id string) (*model.Scenario, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.byID[id]
	return sc, ok
}

// Default 返回首个情境
func (s *ScenarioService) Default() (*model.Scenario, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.order) == 0 {
		return nil, false
	}
	return s.byID[s.order[0]], true
}

// ScenarioBrief 情境概要
type ScenarioBrief struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Goal  string `json:"goal"`
	Image string `json:"image"`
}
