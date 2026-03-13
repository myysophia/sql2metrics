package notifier

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// DingTalkNotifier sends notifications to DingTalk (钉钉)
type DingTalkNotifier struct {
	config *DingTalkConfig
	client *http.Client
}

// NewDingTalkNotifier creates a new DingTalk notifier
func NewDingTalkNotifier(config *DingTalkConfig) *DingTalkNotifier {
	return &DingTalkNotifier{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// dingTalkMessage represents the message format for DingTalk webhook
type dingTalkMessage struct {
	MsgType  string          `json:"msgtype"`
	Text     *dingTalkText   `json:"text,omitempty"`
	Markdown *dingTalkMarkdown `json:"markdown,omitempty"`
	At       *dingTalkAt     `json:"at,omitempty"`
}

type dingTalkText struct {
	Content string `json:"content"`
}

type dingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type dingTalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIDs []string `json:"atUserIds,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

// SendNotification sends a notification to DingTalk
func (n *DingTalkNotifier) SendNotification(notification AlertNotification) error {
	if n.config == nil || !n.config.Enabled || n.config.Webhook == "" {
		return fmt.Errorf("钉钉通知未配置或未启用")
	}

	log.Printf("[NOTIFIER-DINGTALK] 准备发送通知: %s (状态: %s)", notification.AlertName, notification.Status)

	// Build message
	msg := dingTalkMessage{
		MsgType: "markdown",
		Markdown: &dingTalkMarkdown{
			Title: fmt.Sprintf("告警通知: %s", notification.AlertName),
			Text:  n.buildMarkdownContent(notification),
		},
		At: &dingTalkAt{
			AtMobiles: n.config.AtMobiles,
			AtUserIDs: n.config.AtUserIDs,
			IsAtAll:   n.config.IsAtAll,
		},
	}

	// Add signature if secret is configured
	webhookURL := n.config.Webhook
	if n.config.Secret != "" {
		timestamp := time.Now().UnixMilli()
		sign := n.generateSign(timestamp, n.config.Secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, sign)
		log.Printf("[NOTIFIER-DINGTALK] 使用签名验证")
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	log.Printf("[NOTIFIER-DINGTALK] 发送消息到: %s", webhookURL)

	// Send to webhook
	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("钉钉返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err == nil {
		if result.ErrCode != 0 {
			return fmt.Errorf("钉钉返回错误: %d - %s", result.ErrCode, result.ErrMsg)
		}
	}

	log.Printf("[NOTIFIER-DINGTALK] ✅ 通知发送成功")
	return nil
}

// generateSign generates signature for DingTalk webhook
func (n *DingTalkNotifier) generateSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
}

// buildMarkdownContent builds markdown content for DingTalk
func (n *DingTalkNotifier) buildMarkdownContent(notification AlertNotification) string {
	var md string
	md += fmt.Sprintf("## %s %s\n\n", n.getStatusIcon(notification.Status), notification.AlertName)
	md += fmt.Sprintf("- **当前值**: %.2f\n", notification.Value)

	// Add duration if available
	if notification.Duration != "" {
		md += fmt.Sprintf("- **持续时间要求**: %s\n", notification.Duration)
	}

	// Add labels
	if len(notification.Labels) > 0 {
		md += "### 标签\n"
		for k, v := range notification.Labels {
			md += fmt.Sprintf("- %s: **%s**\n", k, v)
		}
		md += "\n"
	}

	// Add annotations (message)
	if len(notification.Annotations) > 0 {
		md += "### 详情\n"
		for k, v := range notification.Annotations {
			md += fmt.Sprintf("- **%s**: %s\n", k, v)
		}
		md += "\n"
	}

	md += fmt.Sprintf("- **开始时间**: %s\n", notification.StartsAt.Format("2006-01-02 15:04:05"))

	if !notification.EndsAt.IsZero() {
		md += fmt.Sprintf("- **结束时间**: %s\n", notification.EndsAt.Format("2006-01-02 15:04:05"))
	}

	// Add call to action for firing alerts
	if notification.Status == "firing" {
		md += "\n请相关人员处理"
	}

	return md
}

func (n *DingTalkNotifier) getStatusIcon(status string) string {
	if status == "firing" {
		return "🚨"
	}
	return "✅"
}
