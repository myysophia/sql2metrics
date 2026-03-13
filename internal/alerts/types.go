package alerts

import (
	"time"
)

// AlertRule defines an alert rule configuration
type AlertRule struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`

	// Metric reference
	MetricName string `json:"metric_name"`

	// Evaluation settings
	EvaluationMode       string `json:"evaluation_mode"` // "collection" or "scheduled"
	EvaluationInterval   string `json:"evaluation_interval,omitempty"`
	EvaluationIntervalMs int64  `json:"evaluation_interval_ms,omitempty"` // parsed duration in milliseconds

	// Condition definition
	Condition AlertCondition `json:"condition"`

	// Alert metadata
	Severity    string            `json:"severity"` // "critical", "warning", "info"
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// State tracking
	State          string `json:"state"` // "pending", "firing", "resolved"
	LastEvaluation string `json:"last_evaluation,omitempty"`
	LastTriggered  string `json:"last_triggered,omitempty"`
	TriggerCount   int    `json:"trigger_count"`

	// Timestamps
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// AlertCondition defines the alert condition
type AlertCondition struct {
	Type      string              `json:"type"` // "threshold", "trend", "anomaly"
	Threshold *ThresholdCondition `json:"threshold,omitempty"`
	Trend     *TrendCondition     `json:"trend,omitempty"`
	Anomaly   *AnomalyCondition   `json:"anomaly,omitempty"`
}

// ThresholdCondition for simple value comparisons
type ThresholdCondition struct {
	Operator string  `json:"operator"` // ">", ">=", "<", "<=", "==", "!="
	Value    float64 `json:"value"`
	Duration string  `json:"duration,omitempty"` // e.g., "5m" - how long condition must be true
}

// TrendCondition for detecting rate-of-change patterns
type TrendCondition struct {
	Type       string  `json:"type"` // "increase", "decrease", "percentage_change"
	Window     string  `json:"window"`
	WindowMs   int64   `json:"window_ms,omitempty"` // parsed window in milliseconds
	Threshold  float64 `json:"threshold"`
	Comparison string  `json:"comparison,omitempty"` // "previous_window", "fixed_value"
}

// AnomalyCondition for statistical anomaly detection
type AnomalyCondition struct {
	Algorithm   string  `json:"algorithm"` // "zscore", "iqr", "moving_average"
	Window      string  `json:"window"`
	WindowMs    int64   `json:"window_ms,omitempty"` // parsed window in milliseconds
	Threshold   float64 `json:"threshold"` // e.g., 3.0 for z-score (3 sigma)
	Sensitivity string  `json:"sensitivity,omitempty"` // "low", "medium", "high"
}

// AlertHistoryEntry records alert state changes
type AlertHistoryEntry struct {
	ID            string            `json:"id"`
	AlertRuleID   string            `json:"alert_rule_id"`
	AlertRuleName string            `json:"alert_rule_name"`
	State         string            `json:"state"` // "firing", "resolved"
	Value         float64           `json:"value"`
	Message       string            `json:"message"`
	TriggeredAt   string            `json:"triggered_at"`
	ResolvedAt    string            `json:"resolved_at,omitempty"`
	Labels        map[string]string `json:"labels"`
}

// EvaluationResult represents the result of evaluating an alert rule
type EvaluationResult struct {
	RuleID      string    `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Triggered   bool      `json:"triggered"`
	Value       float64   `json:"value"`
	Message     string    `json:"message"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

// Alert represents an alert sent to Alertmanager
type Alert struct {
	RuleID      string
	RuleName    string
	Labels      map[string]string
	Annotations map[string]string
	StartsAt    time.Time
	EndsAt      time.Time
	Value       float64   // 当前指标值
	Duration    string    // 告警持续时间要求
}

// NewAlertRule creates a new alert rule with initialized fields
func NewAlertRule(name, metricName string) *AlertRule {
	now := time.Now().Format(time.RFC3339)
	return &AlertRule{
		ID:            generateID(),
		Name:          name,
		Enabled:       true,
		MetricName:    metricName,
		EvaluationMode: "collection",
		State:         "pending",
		Severity:      "warning",
		Labels:        make(map[string]string),
		Annotations:   make(map[string]string),
		TriggerCount:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// generateID generates a unique ID for an alert rule
func generateID() string {
	return "alert-" + time.Now().Format("20060102150405") + "-" + randString(4)
}

// randString generates a random string of n characters
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// ShouldEvaluateNow checks if a scheduled alert should be evaluated now
func (r *AlertRule) ShouldEvaluateNow(lastEvaluation time.Time) bool {
	if r.EvaluationMode != "scheduled" {
		return false
	}
	if r.EvaluationIntervalMs <= 0 {
		return false
	}
	return time.Since(lastEvaluation) >= time.Duration(r.EvaluationIntervalMs)*time.Millisecond
}

// Clone creates a deep copy of the alert rule
func (r *AlertRule) Clone() *AlertRule {
	clone := *r
	if r.Labels != nil {
		clone.Labels = make(map[string]string, len(r.Labels))
		for k, v := range r.Labels {
			clone.Labels[k] = v
		}
	}
	if r.Annotations != nil {
		clone.Annotations = make(map[string]string, len(r.Annotations))
		for k, v := range r.Annotations {
			clone.Annotations[k] = v
		}
	}
	return &clone
}
