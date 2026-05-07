package notifier

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/company/ems-devices/internal/alerts"
)

// Manager is the built-in alertmanager that handles alert routing, grouping, deduplication, and suppression
type Manager struct {
	config *NotifierConfig
	router *Router // Router for intelligent notification routing

	// Notifiers (legacy, kept for backward compatibility)
	wechat   *WeChatNotifier
	dingtalk *DingTalkNotifier
	feishu   *FeishuNotifier

	// State
	mu                sync.RWMutex
	pendingAlerts    map[string]*PendingAlert      // key: groupKey
	alertHistory      map[string]*AlertHistory      // key: alertID
	lastNotifiedTime map[string]time.Time          // key: groupKey, for repeat interval

	// Channels
	alertChan chan alerts.Alert

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

// PendingAlert represents a group of alerts that are waiting to be sent
type PendingAlert struct {
	GroupKey     string
	Alerts       []alerts.Alert
	FirstSeen    time.Time
	LastUpdated  time.Time
	Notified     bool
}

// AlertHistory tracks the notification history
type AlertHistory struct {
	AlertID     string
	Status      string
	NotifiedAt  time.Time
	ClearedAt   time.Time
}

// NewManager creates a new built-in alertmanager
func NewManager(config *NotifierConfig, router *Router) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	mgr := &Manager{
		config:            config,
		router:            router,
		alertChan:         make(chan alerts.Alert, 100),
		pendingAlerts:     make(map[string]*PendingAlert),
		alertHistory:      make(map[string]*AlertHistory),
		lastNotifiedTime:  make(map[string]time.Time),
		ctx:               ctx,
		cancel:            cancel,
	}

	// Initialize legacy notifiers for backward compatibility
	if config.WeChat != nil && config.WeChat.Enabled {
		mgr.wechat = NewWeChatNotifier(config.WeChat)
		log.Printf("[NOTIFIER] 企业微信通知已启用 (legacy)")
	}
	if config.DingTalk != nil && config.DingTalk.Enabled {
		mgr.dingtalk = NewDingTalkNotifier(config.DingTalk)
		log.Printf("[NOTIFIER] 钉钉通知已启用 (legacy)")
	}
	if config.Feishu != nil && config.Feishu.Enabled {
		mgr.feishu = NewFeishuNotifier(config.Feishu)
		log.Printf("[NOTIFIER] 飞书通知已启用 (legacy)")
	}

	// Start processing goroutine
	go mgr.processAlerts()

	return mgr
}

// NewLegacyManager creates a manager without router (for backward compatibility)
func NewLegacyManager(config *NotifierConfig) *Manager {
	return NewManager(config, nil)
}

// SendAlert sends an alert to the built-in alertmanager
func (m *Manager) SendAlert(alert alerts.Alert) error {
	log.Printf("[NOTIFIER] 收到告警: %s (状态: firing)", alert.RuleName)

	select {
	case m.alertChan <- alert:
		return nil
	case <-m.ctx.Done():
		return fmt.Errorf("告警管理器已关闭")
	}
}

// SendResolved sends a resolved notification
func (m *Manager) SendResolved(alert alerts.Alert) error {
	log.Printf("[NOTIFIER] 收到告警恢复: %s (状态: resolved)", alert.RuleName)

	// Clear from history immediately
	m.mu.Lock()
	delete(m.alertHistory, alert.RuleID)
	m.mu.Unlock()

	notification := AlertNotification{
		AlertName:   alert.RuleName,
		Status:      "resolved",
		Labels:      alert.Labels,
		Annotations: alert.Annotations,
		StartsAt:    alert.StartsAt,
		EndsAt:      alert.EndsAt,
		Value:       alert.Value,
		Duration:    alert.Duration,
	}

	// Use router if available, otherwise use legacy method
	if m.router != nil {
		results := m.router.SendNotification(context.Background(), alert, notification)
		// Log results
		for _, result := range results {
			if result.Success {
				log.Printf("[NOTIFIER] ✅ %s 通知发送成功", result.Channel)
			} else {
				log.Printf("[NOTIFIER] ❌ %s 通知发送失败: %s", result.Channel, result.Error)
			}
		}
	} else {
		m.sendToAllChannelsLegacy(notification)
	}

	return nil
}

// processAlerts processes incoming alerts in a loop
func (m *Manager) processAlerts() {
	groupInterval := m.parseDuration(m.config.GroupInterval, 10*time.Second)

	ticker := time.NewTicker(groupInterval)
	defer ticker.Stop()

	for {
		select {
		case alert := <-m.alertChan:
			m.handleIncomingAlert(alert)

		case <-ticker.C:
			m.checkPendingAlerts()

		case <-m.ctx.Done():
			log.Printf("[NOTIFIER] 告警管理器已停止")
			return
		}
	}
}

// handleIncomingAlert handles an incoming alert
func (m *Manager) handleIncomingAlert(alert alerts.Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	groupKey := m.calculateGroupKey(alert)
	log.Printf("[NOTIFIER] 告警分组键: %s", groupKey)

	// Add to pending alerts
	now := time.Now()
	if pending, exists := m.pendingAlerts[groupKey]; exists {
		// Add to existing group
		pending.Alerts = append(pending.Alerts, alert)
		pending.LastUpdated = now
		log.Printf("[NOTIFIER] 告警已添加到分组 %s (当前数量: %d)", groupKey, len(pending.Alerts))
	} else {
		// Create new pending alert
		m.pendingAlerts[groupKey] = &PendingAlert{
			GroupKey:    groupKey,
			Alerts:      []alerts.Alert{alert},
			FirstSeen:   now,
			LastUpdated: now,
			Notified:    false,
		}
		log.Printf("[NOTIFIER] 创建新告警分组: %s", groupKey)

		// Store in history
		m.alertHistory[alert.RuleID] = &AlertHistory{
			AlertID:    alert.RuleID,
			Status:     "firing",
			NotifiedAt: now,
		}
	}
}

// checkPendingAlerts checks and sends pending alerts
func (m *Manager) checkPendingAlerts() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	groupWait := m.parseDuration(m.config.GroupWait, 10*time.Second)

	for groupKey, pending := range m.pendingAlerts {
		// Check if should send notification
		shouldSend := false

		if !pending.Notified {
			// First notification
			if now.Sub(pending.FirstSeen) >= groupWait {
				shouldSend = true
				pending.Notified = true
			}
		} else {
			// Repeat notification
			repeatInterval := m.parseDuration(m.config.RepeatInterval, 0)
			if repeatInterval > 0 {
				if lastNotified, exists := m.lastNotifiedTime[groupKey]; exists {
					if now.Sub(lastNotified) >= repeatInterval {
						shouldSend = true
					}
				}
			}
		}

		if shouldSend {
			log.Printf("[NOTIFIER] 发送分组告警通知: %s (包含 %d 个告警)", groupKey, len(pending.Alerts))
			m.sendGroupNotification(pending)
			m.lastNotifiedTime[groupKey] = now
		}
	}
}

// sendGroupNotification sends a notification for a group of alerts
func (m *Manager) sendGroupNotification(pending *PendingAlert) {
	if len(pending.Alerts) == 0 {
		return
	}

	// Create a combined notification from all alerts in the group
	// For now, send the first alert as representative
	// TODO: Aggregate all alerts into a single notification
	representative := pending.Alerts[0]

	notification := AlertNotification{
		AlertName:   representative.RuleName,
		Status:      "firing",
		Labels:      representative.Labels,
		Annotations: representative.Annotations,
		StartsAt:    representative.StartsAt,
		Value:       representative.Value,
		Duration:    representative.Duration,
	}

	// Add group info
	if len(pending.Alerts) > 1 {
		if notification.Annotations == nil {
			notification.Annotations = make(map[string]string)
		}
		notification.Annotations["group_size"] = fmt.Sprintf("%d", len(pending.Alerts))
		notification.Annotations["group_members"] = m.getAlertNames(pending.Alerts)
	}

	// Use router if available, otherwise use legacy method
	if m.router != nil {
		results := m.router.SendNotification(context.Background(), representative, notification)
		// Log results
		for _, result := range results {
			if result.Success {
				log.Printf("[NOTIFIER] ✅ %s 通知发送成功", result.Channel)
			} else {
				log.Printf("[NOTIFIER] ❌ %s 通知发送失败: %s", result.Channel, result.Error)
			}
		}
	} else {
		m.sendToAllChannelsLegacy(notification)
	}
}

// sendToAllChannelsLegacy sends notification to all enabled channels (legacy method)
func (m *Manager) sendToAllChannelsLegacy(notification AlertNotification) {
	results := make([]NotificationResult, 0)

	// WeChat
	if m.wechat != nil {
		err := m.wechat.SendNotification(notification)
		results = append(results, NotificationResult{
			Channel: "wechat",
			Success: err == nil,
			Error:   getErrorString(err),
		})
	}

	// DingTalk
	if m.dingtalk != nil {
		err := m.dingtalk.SendNotification(notification)
		results = append(results, NotificationResult{
			Channel: "dingtalk",
			Success: err == nil,
			Error:   getErrorString(err),
		})
	}

	// Feishu
	if m.feishu != nil {
		err := m.feishu.SendNotification(notification)
		results = append(results, NotificationResult{
			Channel: "feishu",
			Success: err == nil,
			Error:   getErrorString(err),
		})
	}

	// Log results
	for _, result := range results {
		if result.Success {
			log.Printf("[NOTIFIER] ✅ %s 通知发送成功", result.Channel)
		} else {
			log.Printf("[NOTIFIER] ❌ %s 通知发送失败: %s", result.Channel, result.Error)
		}
	}
}

// calculateGroupKey calculates a group key for an alert
func (m *Manager) calculateGroupKey(alert alerts.Alert) string {
	// Simple grouping: by alertname
	// TODO: Add more sophisticated grouping logic based on labels
	parts := []string{alert.RuleName}

	// Add severity if available
	if severity, ok := alert.Labels["severity"]; ok {
		parts = append(parts, severity)
	}

	return strings.Join(parts, "|")
}

// getAlertNames gets alert names from a list of alerts
func (m *Manager) getAlertNames(alerts []alerts.Alert) string {
	names := make([]string, len(alerts))
	for i, alert := range alerts {
		names[i] = alert.RuleName
	}
	return strings.Join(names, ", ")
}

// parseDuration parses a duration string, returns default if invalid
func (m *Manager) parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}
	dur, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("[NOTIFIER] ⚠️ 解析时长失败 '%s': %v，使用默认值 %v", durationStr, err, defaultDuration)
		return defaultDuration
	}
	return dur
}

// getErrorString converts error to string
func getErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Stop stops the alertmanager
func (m *Manager) Stop() {
	m.cancel()
}
