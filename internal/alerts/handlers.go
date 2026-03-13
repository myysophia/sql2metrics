package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Handler provides alert API handlers
type Handler struct {
	storage  *Storage
	history  *History
	evaluator *Evaluator
}

// NewHandler creates a new alert handler
func NewHandler(storage *Storage, history *History, evaluator *Evaluator) *Handler {
	return &Handler{
		storage:  storage,
		history:  history,
		evaluator: evaluator,
	}
}

// ListAlerts returns all alert rules
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	rules := h.storage.List()
	writeJSON(w, http.StatusOK, rules)
}

// GetAlert returns a specific alert rule
func (h *Handler) GetAlert(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	id = strings.TrimSuffix(id, "/")

	rule, ok := h.storage.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "告警规则不存在")
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// CreateAlert creates a new alert rule
func (h *Handler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	// Validate
	if err := validateAlertRule(&rule); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("验证失败: %v", err))
		return
	}

	// Parse durations
	if err := parseDurations(&rule); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("解析时间参数失败: %v", err))
		return
	}

	// Set default values
	if rule.ID == "" {
		rule.ID = generateID()
	}
	if rule.CreatedAt == "" {
		rule.CreatedAt = time.Now().Format(time.RFC3339)
	}
	rule.UpdatedAt = time.Now().Format(time.RFC3339)
	if rule.State == "" {
		rule.State = "pending"
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	if rule.Labels == nil {
		rule.Labels = make(map[string]string)
	}
	if rule.Annotations == nil {
		rule.Annotations = make(map[string]string)
	}

	if err := h.storage.Add(rule); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Save to file
	if err := h.storage.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存失败: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

// UpdateAlert updates an existing alert rule
func (h *Handler) UpdateAlert(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	id = strings.TrimSuffix(id, "/")

	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	// Validate
	if err := validateAlertRule(&rule); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("验证失败: %v", err))
		return
	}

	// Parse durations
	if err := parseDurations(&rule); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("解析时间参数失败: %v", err))
		return
	}

	// Ensure ID matches
	rule.ID = id
	rule.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.storage.Update(rule); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusConflict, err.Error())
		}
		return
	}

	// Save to file
	if err := h.storage.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存失败: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// DeleteAlert deletes an alert rule
func (h *Handler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	id = strings.TrimSuffix(id, "/")

	if err := h.storage.Delete(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Save to file
	if err := h.storage.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存失败: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "告警规则已删除"})
}

// EnableAlert enables an alert rule
func (h *Handler) EnableAlert(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimSuffix(id, "/enable")

	if err := h.storage.SetEnabled(id, true); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Save to file
	if err := h.storage.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存失败: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "告警规则已启用"})
}

// DisableAlert disables an alert rule
func (h *Handler) DisableAlert(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimSuffix(id, "/disable")

	if err := h.storage.SetEnabled(id, false); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Save to file
	if err := h.storage.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存失败: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "告警规则已禁用"})
}

// TestAlert tests an alert rule with current data
func (h *Handler) TestAlert(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimSuffix(id, "/test")

	rule, ok := h.storage.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "告警规则不存在")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := h.evaluator.EvaluateRule(ctx, rule)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("评估失败: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetAlertHistory returns alert history
func (h *Handler) GetAlertHistory(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	ruleID := r.URL.Query().Get("rule_id")

	page := 1
	pageSize := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// If rule_id is specified, return history for that rule only
	if ruleID != "" {
		entries := h.history.GetByRuleID(ruleID, pageSize)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":  entries,
			"total": len(entries),
			"page":  1,
			"page_size": pageSize,
		})
		return
	}

	// Return all history with pagination
	entries, total := h.history.List(page, pageSize)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":      entries,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// EvaluateAllAlerts manually triggers evaluation for all rules
func (h *Handler) EvaluateAllAlerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rules := h.storage.List()
	var results []EvaluationResult

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		result, err := h.evaluator.EvaluateRule(ctx, rule)
		if err != nil {
			logMsg := fmt.Sprintf("评估告警规则 %s 失败: %v", rule.Name, err)
			results = append(results, EvaluationResult{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Triggered:   false,
				Message:     logMsg,
				EvaluatedAt: time.Now(),
			})
			continue
		}

		// Update state
		h.evaluator.updateRuleState(&rule, result)
		results = append(results, *result)
	}

	writeJSON(w, http.StatusOK, results)
}

// GetAlertStats returns alert statistics
func (h *Handler) GetAlertStats(w http.ResponseWriter, r *http.Request) {
	stats := h.history.GetStats()
	stats["total_rules"] = h.storage.Count()
	stats["enabled_rules"] = len(h.storage.GetEnabled())

	writeJSON(w, http.StatusOK, stats)
}

// Helper functions

func validateAlertRule(rule *AlertRule) error {
	if rule.Name == "" {
		return fmt.Errorf("告警规则名称不能为空")
	}
	if rule.MetricName == "" {
		return fmt.Errorf("指标名称不能为空")
	}
	if rule.EvaluationMode == "" {
		rule.EvaluationMode = "collection"
	}
	if rule.EvaluationMode != "collection" && rule.EvaluationMode != "scheduled" {
		return fmt.Errorf("无效的评估模式: %s", rule.EvaluationMode)
	}
	if rule.Condition.Type == "" {
		return fmt.Errorf("告警条件类型不能为空")
	}
	if rule.Condition.Type != "threshold" && rule.Condition.Type != "trend" && rule.Condition.Type != "anomaly" {
		return fmt.Errorf("无效的告警条件类型: %s", rule.Condition.Type)
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	if rule.Severity != "critical" && rule.Severity != "warning" && rule.Severity != "info" {
		return fmt.Errorf("无效的严重级别: %s", rule.Severity)
	}
	return nil
}

func parseDurations(rule *AlertRule) error {
	// Parse evaluation interval
	if rule.EvaluationInterval != "" {
		d, err := time.ParseDuration(rule.EvaluationInterval)
		if err != nil {
			return fmt.Errorf("解析评估间隔失败: %w", err)
		}
		rule.EvaluationIntervalMs = int64(d.Milliseconds())
	}

	// Parse threshold duration
	if rule.Condition.Threshold != nil && rule.Condition.Threshold.Duration != "" {
		if _, err := time.ParseDuration(rule.Condition.Threshold.Duration); err != nil {
			return fmt.Errorf("解析阈值持续时间失败: %w", err)
		}
	}

	// Parse trend window
	if rule.Condition.Trend != nil && rule.Condition.Trend.Window != "" {
		d, err := time.ParseDuration(rule.Condition.Trend.Window)
		if err != nil {
			return fmt.Errorf("解析趋势时间窗口失败: %w", err)
		}
		rule.Condition.Trend.WindowMs = int64(d.Milliseconds())
	}

	// Parse anomaly window
	if rule.Condition.Anomaly != nil && rule.Condition.Anomaly.Window != "" {
		d, err := time.ParseDuration(rule.Condition.Anomaly.Window)
		if err != nil {
			return fmt.Errorf("解析异常检测时间窗口失败: %w", err)
		}
		rule.Condition.Anomaly.WindowMs = int64(d.Milliseconds())
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
