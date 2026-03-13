package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// AlertmanagerClient sends alerts to Alertmanager
type AlertmanagerClient struct {
	baseURL string
	client  *http.Client
}

// alertmanagerPayload represents the payload sent to Alertmanager
// Note: We use a slice directly instead of wrapping in an object for compatibility
type alertmanagerPayload []alertmanagerAlert

// alertmanagerAlert represents a single alert in Alertmanager format
type alertmanagerAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	StartsAt     string            `json:"startsAt,omitempty"`
	EndsAt       string            `json:"endsAt,omitempty"`
}

// NewAlertmanagerClient creates a new Alertmanager client
func NewAlertmanagerClient(baseURL string) *AlertmanagerClient {
	return &AlertmanagerClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendAlert sends a firing alert to Alertmanager
func (c *AlertmanagerClient) SendAlert(alert Alert) error {
	log.Printf("[ALERT] 开始发送告警: ruleID=%s, ruleName=%s", alert.RuleID, alert.RuleName)

	if c.baseURL == "" {
		log.Printf("[ALERT] ❌ Alertmanager URL 未配置")
		return fmt.Errorf("Alertmanager URL 未配置")
	}

	amAlert := c.convertAlert(alert, "firing")
	payload := alertmanagerPayload{amAlert}

	// Log the alert details
	log.Printf("[ALERT] 告警详情:")
	log.Printf("  - 标签: %v", amAlert.Labels)
	log.Printf("  - 注解: %v", amAlert.Annotations)
	log.Printf("  - 开始时间: %s", amAlert.StartsAt)

	return c.send(payload)
}

// SendResolved sends a resolved alert to Alertmanager
func (c *AlertmanagerClient) SendResolved(alert Alert) error {
	log.Printf("[ALERT] 开始发送告警恢复通知: ruleID=%s, ruleName=%s", alert.RuleID, alert.RuleName)

	if c.baseURL == "" {
		log.Printf("[ALERT] ❌ Alertmanager URL 未配置")
		return fmt.Errorf("Alertmanager URL 未配置")
	}

	amAlert := c.convertAlert(alert, "resolved")
	payload := alertmanagerPayload{amAlert}

	// Log the alert details
	log.Printf("[ALERT] 告警恢复详情:")
	log.Printf("  - 标签: %v", amAlert.Labels)
	log.Printf("  - 注解: %v", amAlert.Annotations)
	log.Printf("  - 开始时间: %s, 结束时间: %s", amAlert.StartsAt, amAlert.EndsAt)

	return c.send(payload)
}

// convertAlert converts an internal Alert to Alertmanager format
func (c *AlertmanagerClient) convertAlert(alert Alert, status string) alertmanagerAlert {
	amAlert := alertmanagerAlert{
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
		GeneratorURL: fmt.Sprintf("http://sql2metrics/alerts/%s", alert.RuleID),
		StartsAt:     alert.StartsAt.Format(time.RFC3339),
	}

	// Copy labels
	for k, v := range alert.Labels {
		amAlert.Labels[k] = v
	}
	// Add standard labels
	amAlert.Labels["alertname"] = alert.RuleName
	amAlert.Labels["alert_id"] = alert.RuleID

	// Add severity label
	if status == "resolved" {
		amAlert.Labels["severity"] = "resolved"
	} else {
		amAlert.Labels["severity"] = "firing"
	}

	// Copy annotations
	for k, v := range alert.Annotations {
		amAlert.Annotations[k] = v
	}
	amAlert.Annotations["rule_id"] = alert.RuleID

	// Set endsAt for resolved alerts
	if status == "resolved" && !alert.EndsAt.IsZero() {
		amAlert.EndsAt = alert.EndsAt.Format(time.RFC3339)
	}

	return amAlert
}

// send sends the payload to Alertmanager
func (c *AlertmanagerClient) send(payload alertmanagerPayload) error {
	log.Printf("[ALERT] 准备发送到 Alertmanager: %s", c.baseURL)

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ALERT] ❌ 序列化告警数据失败: %v", err)
		return fmt.Errorf("序列化告警数据失败: %w", err)
	}

	log.Printf("[ALERT] 发送 Payload: %s", string(data))

	url := fmt.Sprintf("%s/api/v1/alerts", c.baseURL)
	log.Printf("[ALERT] 请求 URL: %s", url)

	resp, err := c.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("[ALERT] ❌ HTTP 请求失败: %v", err)
		return fmt.Errorf("发送告警到 Alertmanager 失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	log.Printf("[ALERT] 响应状态码: %d", resp.StatusCode)
	log.Printf("[ALERT] 响应内容: %s", string(body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[ALERT] ❌ Alertmanager 返回错误状态码 %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("Alertmanager 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[ALERT] ✅ 告警已成功发送到 Alertmanager: %s", c.baseURL)
	return nil
}

// SetTimeout sets the HTTP client timeout
func (c *AlertmanagerClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}
