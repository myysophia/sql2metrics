package alerts

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// Notifier is an interface for sending alert notifications
type Notifier interface {
	SendAlert(alert Alert) error
	SendResolved(alert Alert) error
}

// Service interface for getting current metric values
type Service interface {
	GetMetricValue(metricName string) (float64, bool)
}

// Evaluator manages alert rule evaluation
type Evaluator struct {
	storage      *Storage
	history      *History
	alertmanager *AlertmanagerClient
	metricStore  *MetricValueStore
	service      Service
	notifier     Notifier // Can be nil or any type implementing Notifier interface
	mu           sync.RWMutex

	// Track state transitions for duration requirements
	pendingAlerts map[string]time.Time // ruleID -> first trigger time
}

// NewEvaluator creates a new alert evaluator
func NewEvaluator(storage *Storage, history *History, alertmanager *AlertmanagerClient, metricStore *MetricValueStore, service Service) *Evaluator {
	return &Evaluator{
		storage:        storage,
		history:        history,
		alertmanager:   alertmanager,
		metricStore:    metricStore,
		service:        service,
		pendingAlerts:  make(map[string]time.Time),
	}
}

// EvaluateRule evaluates a single alert rule
func (e *Evaluator) EvaluateRule(ctx context.Context, rule AlertRule) (*EvaluationResult, error) {
	currentVal, ok := e.service.GetMetricValue(rule.MetricName)
	if !ok {
		return &EvaluationResult{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Triggered:   false,
			Message:     fmt.Sprintf("指标 %s 无可用值", rule.MetricName),
			EvaluatedAt: time.Now(),
		}, nil
	}

	var triggered bool
	var message string
	var err error

	switch rule.Condition.Type {
	case "threshold":
		triggered, message, err = e.evaluateThresholdCondition(ctx, rule, currentVal)
	case "trend":
		triggered, message, err = e.evaluateTrendCondition(ctx, rule)
	case "anomaly":
		triggered, message, err = e.evaluateAnomalyCondition(ctx, rule)
	default:
		return nil, fmt.Errorf("不支持的告警条件类型: %s", rule.Condition.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("评估告警规则 %s 失败: %w", rule.Name, err)
	}

	result := &EvaluationResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Triggered:   triggered,
		Value:       currentVal,
		Message:     message,
		EvaluatedAt: time.Now(),
	}

	return result, nil
}

// evaluateThresholdCondition evaluates threshold conditions
func (e *Evaluator) evaluateThresholdCondition(ctx context.Context, rule AlertRule, currentVal float64) (bool, string, error) {
	condition := rule.Condition.Threshold
	if condition == nil {
		return false, "阈值条件未配置", nil
	}

	triggered := false

	switch condition.Operator {
	case ">":
		triggered = currentVal > condition.Value
	case ">=":
		triggered = currentVal >= condition.Value
	case "<":
		triggered = currentVal < condition.Value
	case "<=":
		triggered = currentVal <= condition.Value
	case "==":
		triggered = currentVal == condition.Value
	case "!=":
		triggered = currentVal != condition.Value
	default:
		return false, "", fmt.Errorf("不支持的阈值操作符: %s", condition.Operator)
	}

	message := fmt.Sprintf("指标值 %.2f %s 阈值 %.2f", currentVal, condition.Operator, condition.Value)

	// Handle duration requirement
	if triggered && condition.Duration != "" {
		duration, err := time.ParseDuration(condition.Duration)
		if err != nil {
			return false, "", fmt.Errorf("解析持续时间失败: %w", err)
		}

		firstTriggerTime, exists := e.pendingAlerts[rule.ID]
		if !exists {
			e.pendingAlerts[rule.ID] = time.Now()
			return false, fmt.Sprintf("%s (等待持续时间 %s)", message, condition.Duration), nil
		}

		if time.Since(firstTriggerTime) < duration {
			return false, fmt.Sprintf("%s (等待持续时间 %s, 已等待 %s)", message, condition.Duration, time.Since(firstTriggerTime).Round(time.Second)), nil
		}
	} else if !triggered {
		// Clear pending state if condition is no longer met
		delete(e.pendingAlerts, rule.ID)
	}

	return triggered, message, nil
}

// evaluateTrendCondition evaluates trend conditions
func (e *Evaluator) evaluateTrendCondition(ctx context.Context, rule AlertRule) (bool, string, error) {
	condition := rule.Condition.Trend
	if condition == nil {
		return false, "趋势条件未配置", nil
	}

	window := time.Duration(condition.WindowMs) * time.Millisecond
	if window <= 0 {
		return false, "", fmt.Errorf("无效的时间窗口: %s", condition.Window)
	}

	values := e.metricStore.GetValues(rule.MetricName, window)
	if len(values) < 2 {
		return false, fmt.Sprintf("数据不足，无法进行趋势分析（当前 %d 个数据点，至少需要 2 个）", len(values)), nil
	}

	currentValue := values[len(values)-1].Value
	firstValue := values[0].Value

	triggered := false
	var message string

	switch condition.Type {
	case "increase":
		increase := currentValue - firstValue
		triggered = increase >= condition.Threshold
		message = fmt.Sprintf("值增加了 %.2f（从 %.2f 到 %.2f，阈值 %.2f）", increase, firstValue, currentValue, condition.Threshold)
	case "decrease":
		decrease := firstValue - currentValue
		triggered = decrease >= condition.Threshold
		message = fmt.Sprintf("值减少了 %.2f（从 %.2f 到 %.2f，阈值 %.2f）", decrease, firstValue, currentValue, condition.Threshold)
	case "percentage_change":
		change := ((currentValue - firstValue) / firstValue) * 100
		triggered = math.Abs(change) >= condition.Threshold
		message = fmt.Sprintf("值变化了 %.2f%%（从 %.2f 到 %.2f，阈值 %.2f%%）", change, firstValue, currentValue, condition.Threshold)
	default:
		return false, "", fmt.Errorf("不支持的趋势类型: %s", condition.Type)
	}

	return triggered, message, nil
}

// evaluateAnomalyCondition evaluates anomaly detection conditions
func (e *Evaluator) evaluateAnomalyCondition(ctx context.Context, rule AlertRule) (bool, string, error) {
	condition := rule.Condition.Anomaly
	if condition == nil {
		return false, "异常检测条件未配置", nil
	}

	window := time.Duration(condition.WindowMs) * time.Millisecond
	if window <= 0 {
		return false, "", fmt.Errorf("无效的时间窗口: %s", condition.Window)
	}

	values := e.metricStore.GetValues(rule.MetricName, window)
	if len(values) < 10 {
		return false, fmt.Sprintf("数据不足，无法进行异常检测（当前 %d 个数据点，至少需要 10 个）", len(values)), nil
	}

	currentValue := values[len(values)-1].Value
	historical := make([]float64, len(values)-1)
	for i := 0; i < len(historical); i++ {
		historical[i] = values[i].Value
	}

	triggered := false
	var message string

	switch condition.Algorithm {
	case "zscore":
		triggered, message = e.detectZScore(currentValue, historical, condition.Threshold)
	case "iqr":
		triggered, message = e.detectIQR(currentValue, historical, condition.Threshold)
	case "moving_average":
		triggered, message = e.detectMovingAverage(currentValue, historical, condition.Threshold)
	default:
		return false, "", fmt.Errorf("不支持的异常检测算法: %s", condition.Algorithm)
	}

	return triggered, message, nil
}

// detectZScore detects anomalies using z-score algorithm
func (e *Evaluator) detectZScore(currentValue float64, historical []float64, threshold float64) (bool, string) {
	mean, stdDev := calculateStatistics(historical)
	if stdDev == 0 {
		return false, fmt.Sprintf("历史数据标准差为 0，无法计算 z-score（值: %.2f, 均值: %.2f）", currentValue, mean)
	}

	zscore := (currentValue - mean) / stdDev
	triggered := math.Abs(zscore) >= threshold

	message := fmt.Sprintf("Z-score: %.2f（阈值: %.2f）, 当前值: %.2f, 均值: %.2f, 标准差: %.2f", zscore, threshold, currentValue, mean, stdDev)
	return triggered, message
}

// detectIQR detects anomalies using IQR (Interquartile Range) method
func (e *Evaluator) detectIQR(currentValue float64, historical []float64, threshold float64) (bool, string) {
	q1, q3 := calculateQuartiles(historical)
	iqr := q3 - q1

	// Use threshold as multiplier (default 1.5 for standard IQR method)
	if threshold == 0 {
		threshold = 1.5
	}

	lowerBound := q1 - (threshold * iqr)
	upperBound := q3 + (threshold * iqr)

	triggered := currentValue < lowerBound || currentValue > upperBound

	message := fmt.Sprintf("当前值: %.2f, IQR 范围: [%.2f, %.2f]（Q1: %.2f, Q3: %.2f, IQR: %.2f, 倍数: %.2f）",
		currentValue, lowerBound, upperBound, q1, q3, iqr, threshold)
	return triggered, message
}

// detectMovingAverage detects anomalies using moving average
func (e *Evaluator) detectMovingAverage(currentValue float64, historical []float64, threshold float64) (bool, string) {
	windowSize := 5
	if len(historical) < windowSize {
		windowSize = len(historical)
	}

	recentValues := historical[len(historical)-windowSize:]
	avg := calculateMean(recentValues)

	// Threshold is percentage (e.g., 10 means 10%)
	thresholdValue := avg * (threshold / 100.0)
	deviation := math.Abs(currentValue - avg)

	triggered := deviation > thresholdValue

	message := fmt.Sprintf("当前值: %.2f, 移动平均: %.2f, 偏差: %.2f, 阈值: %.2f%%",
		currentValue, avg, deviation, threshold)
	return triggered, message
}

// EvaluateCollectionModeAlerts evaluates all collection-mode alerts
func (e *Evaluator) EvaluateCollectionModeAlerts(ctx context.Context) {
	rules := e.storage.GetByEvaluationMode("collection")
	for _, rule := range rules {
		result, err := e.EvaluateRule(ctx, rule)
		if err != nil {
			log.Printf("评估告警规则 %s 失败: %v", rule.Name, err)
			continue
		}

		e.updateRuleState(&rule, result)
	}
}

// EvaluateScheduledModeAlerts evaluates all scheduled-mode alerts
func (e *Evaluator) EvaluateScheduledModeAlerts(ctx context.Context) {
	rules := e.storage.GetByEvaluationMode("scheduled")
	for _, rule := range rules {
		// Check if it's time to evaluate
		if rule.LastEvaluation != "" {
			lastEval, err := time.Parse(time.RFC3339, rule.LastEvaluation)
			if err == nil && !rule.ShouldEvaluateNow(lastEval) {
				continue
			}
		}

		result, err := e.EvaluateRule(ctx, rule)
		if err != nil {
			log.Printf("评估告警规则 %s 失败: %v", rule.Name, err)
			continue
		}

		e.updateRuleState(&rule, result)
	}
}

// updateRuleState updates the rule state and sends notifications if needed
func (e *Evaluator) updateRuleState(rule *AlertRule, result *EvaluationResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule.LastEvaluation = result.EvaluatedAt.Format(time.RFC3339)
	oldState := rule.State

	log.Printf("[EVALUATOR] 评估告警规则: %s, 触发=%v, 旧状态=%s, 消息=%s", rule.Name, result.Triggered, oldState, result.Message)

	if result.Triggered && oldState != "firing" {
		// Transition to firing state
		log.Printf("[EVALUATOR] ✅ 状态转换: %s -> firing", oldState)
		rule.State = "firing"
		rule.LastTriggered = result.EvaluatedAt.Format(time.RFC3339)
		rule.TriggerCount++

		// Create alert object
		// Get duration from condition
		duration := ""
		if rule.Condition.Threshold != nil && rule.Condition.Threshold.Duration != "" {
			duration = rule.Condition.Threshold.Duration
		} else if rule.Condition.Trend != nil && rule.Condition.Trend.Window != "" {
			duration = rule.Condition.Trend.Window
		} else if rule.Condition.Anomaly != nil && rule.Condition.Anomaly.Window != "" {
			duration = rule.Condition.Anomaly.Window
		}

		alert := Alert{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Labels:      rule.Labels,
			Annotations: rule.Annotations,
			StartsAt:    result.EvaluatedAt,
			Value:       result.Value,
			Duration:    duration,
		}

		// Send to built-in notifier if configured
		if e.notifier != nil {
			log.Printf("[EVALUATOR] 准备发送告警到内置通知服务...")
			if err := e.notifier.SendAlert(alert); err != nil {
				log.Printf("[EVALUATOR] ❌ 发送告警到内置通知服务失败: %v", err)
			} else {
				log.Printf("[EVALUATOR] ✅ 告警已发送到内置通知服务")
			}
		}

		// Send to external Alertmanager if configured
		if e.alertmanager != nil {
			log.Printf("[EVALUATOR] 准备发送告警到 Alertmanager...")
			if err := e.alertmanager.SendAlert(alert); err != nil {
				log.Printf("[EVALUATOR] ❌ 发送告警到 Alertmanager 失败: %v", err)
			} else {
				log.Printf("[EVALUATOR] ✅ 告警发送成功")
			}
		} else {
			log.Printf("[EVALUATOR] ⚠️ Alertmanager 客户端未初始化，跳过发送")
		}

		// Add to history
		entry := AlertHistoryEntry{
			ID:            generateID(),
			AlertRuleID:   rule.ID,
			AlertRuleName: rule.Name,
			State:         "firing",
			Value:         result.Value,
			Message:       result.Message,
			TriggeredAt:   result.EvaluatedAt.Format(time.RFC3339),
			Labels:        rule.Labels,
		}
		e.history.Add(entry)

		log.Printf("[EVALUATOR] 🚨 告警触发: %s - %s", rule.Name, result.Message)

	} else if !result.Triggered && oldState == "firing" {
		// 检查是否真的不满足条件（不是在等待 duration）
		// 如果在 pendingAlerts 中存在，说明条件满足但还在等待 duration，不应该转换为 resolved
		_, isWaiting := e.pendingAlerts[rule.ID]
		if isWaiting {
			log.Printf("[EVALUATOR] 告警条件仍满足，等待 duration 中，保持 firing 状态")
			return
		}

		// Transition to resolved state
		log.Printf("[EVALUATOR] ✅ 状态转换: %s -> resolved", oldState)
		rule.State = "resolved"

		// Create alert object for resolved notification
		// Get duration from condition
		duration := ""
		if rule.Condition.Threshold != nil && rule.Condition.Threshold.Duration != "" {
			duration = rule.Condition.Threshold.Duration
		} else if rule.Condition.Trend != nil && rule.Condition.Trend.Window != "" {
			duration = rule.Condition.Trend.Window
		} else if rule.Condition.Anomaly != nil && rule.Condition.Anomaly.Window != "" {
			duration = rule.Condition.Anomaly.Window
		}

		alert := Alert{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Labels:      rule.Labels,
			Annotations: rule.Annotations,
			StartsAt:    time.Now(), // Should use actual trigger time
			EndsAt:      result.EvaluatedAt,
			Value:       result.Value,
			Duration:    duration,
		}

		// Send resolved notification to built-in notifier if configured
		if e.notifier != nil {
			log.Printf("[EVALUATOR] 准备发送告警恢复通知到内置通知服务...")
			if err := e.notifier.SendResolved(alert); err != nil {
				log.Printf("[EVALUATOR] ❌ 发送告警恢复通知到内置通知服务失败: %v", err)
			} else {
				log.Printf("[EVALUATOR] ✅ 告警恢复通知已发送到内置通知服务")
			}
		}

		// Send resolved notification to external Alertmanager if configured
		if e.alertmanager != nil {
			log.Printf("[EVALUATOR] 准备发送告警恢复通知到 Alertmanager...")
			if err := e.alertmanager.SendResolved(alert); err != nil {
				log.Printf("[EVALUATOR] ❌ 发送告警恢复通知到 Alertmanager 失败: %v", err)
			} else {
				log.Printf("[EVALUATOR] ✅ 告警恢复通知发送成功")
			}
		}

		// Update history
		e.history.UpdateResolved(rule.ID, result.EvaluatedAt)

		log.Printf("[EVALUATOR] ✅ 告警恢复: %s - %s", rule.Name, result.Message)
	}

	// Save updated rule state to storage
	if err := e.storage.Update(*rule); err != nil {
		log.Printf("[EVALUATOR] ❌ 更新告警规则状态失败: %v", err)
	} else {
		// Persist to file
		if err := e.storage.Save(); err != nil {
			log.Printf("[EVALUATOR] ❌ 保存告警规则状态到文件失败: %v", err)
		} else {
			log.Printf("[EVALUATOR] ✅ 告警规则状态已保存到文件")
		}
	}
}

// MetricStore returns the metric value store
func (e *Evaluator) MetricStore() *MetricValueStore {
	return e.metricStore
}

// SetNotifier sets the built-in notifier manager
func (e *Evaluator) SetNotifier(notifier Notifier) {
	e.notifier = notifier
}

// Helper functions for statistical calculations

func calculateStatistics(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	mean = calculateMean(values)

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateQuartiles(values []float64) (q1, q3 float64) {
	if len(values) == 0 {
		return 0, 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	q1Index := n / 4
	q3Index := (3 * n) / 4

	if q1Index < 0 {
		q1Index = 0
	}
	if q3Index >= n {
		q3Index = n - 1
	}

	return sorted[q1Index], sorted[q3Index]
}
