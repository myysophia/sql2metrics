package alerts

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Storage manages alert rules persistence
type Storage struct {
	filepath string
	rules    []AlertRule
	mu       sync.RWMutex
}

// NewStorage creates a new alert rule storage
func NewStorage(filepath string) *Storage {
	return &Storage{
		filepath: filepath,
		rules:    make([]AlertRule, 0),
	}
}

// Load loads alert rules from file
func (s *Storage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If file doesn't exist, initialize with empty rules
	if _, err := os.Stat(s.filepath); os.IsNotExist(err) {
		s.rules = make([]AlertRule, 0)
		return nil
	}

	data, err := os.ReadFile(s.filepath)
	if err != nil {
		return fmt.Errorf("读取告警规则文件失败: %w", err)
	}

	if len(data) == 0 {
		s.rules = make([]AlertRule, 0)
		return nil
	}

	if err := json.Unmarshal(data, &s.rules); err != nil {
		return fmt.Errorf("解析告警规则文件失败: %w", err)
	}

	return nil
}

// Save saves alert rules to file
func (s *Storage) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.rules, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化告警规则失败: %w", err)
	}

	if err := os.WriteFile(s.filepath, data, 0644); err != nil {
		return fmt.Errorf("写入告警规则文件失败: %w", err)
	}

	return nil
}

// List returns all alert rules
func (s *Storage) List() []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]AlertRule, len(s.rules))
	copy(result, s.rules)
	return result
}

// Get returns an alert rule by ID
func (s *Storage) Get(id string) (AlertRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rule := range s.rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return AlertRule{}, false
}

// Add adds a new alert rule
func (s *Storage) Add(rule AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate name
	for _, r := range s.rules {
		if r.Name == rule.Name {
			return fmt.Errorf("告警规则名称 %s 已存在", rule.Name)
		}
	}

	s.rules = append(s.rules, rule)
	return nil
}

// Update updates an existing alert rule
func (s *Storage) Update(rule AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, r := range s.rules {
		if r.ID == rule.ID {
			// Check for duplicate name
			if r.Name != rule.Name {
				for _, other := range s.rules {
					if other.ID != rule.ID && other.Name == rule.Name {
						return fmt.Errorf("告警规则名称 %s 已存在", rule.Name)
					}
				}
			}
			s.rules[i] = rule
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("告警规则 %s 不存在", rule.ID)
	}

	return nil
}

// Delete removes an alert rule by ID
func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, rule := range s.rules {
		if rule.ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("告警规则 %s 不存在", id)
}

// SetEnabled enables or disables an alert rule
func (s *Storage) SetEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.rules {
		if s.rules[i].ID == id {
			s.rules[i].Enabled = enabled
			return nil
		}
	}

	return fmt.Errorf("告警规则 %s 不存在", id)
}

// GetEnabled returns all enabled alert rules
func (s *Storage) GetEnabled() []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []AlertRule
	for _, rule := range s.rules {
		if rule.Enabled {
			result = append(result, rule)
		}
	}
	return result
}

// GetByMetricName returns alert rules for a specific metric
func (s *Storage) GetByMetricName(metricName string) []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []AlertRule
	for _, rule := range s.rules {
		if rule.MetricName == metricName {
			result = append(result, rule)
		}
	}
	return result
}

// GetByEvaluationMode returns alert rules by evaluation mode
func (s *Storage) GetByEvaluationMode(mode string) []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []AlertRule
	for _, rule := range s.rules {
		if rule.EvaluationMode == mode && rule.Enabled {
			result = append(result, rule)
		}
	}
	return result
}

// Count returns the total number of alert rules
func (s *Storage) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rules)
}
