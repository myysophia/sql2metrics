package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// WeChatNotifier sends notifications to WeChat Work (企业微信)
type WeChatNotifier struct {
	config *WeChatConfig
	client *http.Client
}

// NewWeChatNotifier creates a new WeChat notifier
func NewWeChatNotifier(config *WeChatConfig) *WeChatNotifier {
	return &WeChatNotifier{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// weChatMessage represents the message format for WeChat Work webhook
type weChatMessage struct {
	MsgType string `json:"msgtype"`
	Text     *struct {
		Content             string   `json:"content"`
		MentionedList      []string `json:"mentioned_list,omitempty"`
		MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
	} `json:"text,omitempty"`
	Markdown *struct {
		Content string `json:"content"`
	} `json:"markdown,omitempty"`
}

// SendNotification sends a notification to WeChat Work
func (n *WeChatNotifier) SendNotification(notification AlertNotification) error {
	if n.config == nil || !n.config.Enabled || n.config.Webhook == "" {
		return fmt.Errorf("WeChat 通知未配置或未启用")
	}

	log.Printf("[NOTIFIER-WECHAT] 准备发送通知: %s (状态: %s)", notification.AlertName, notification.Status)

	// Build message content - always use text type to support @mentions
	var content string
	if notification.Status == "firing" {
		content = n.buildFiringText(notification)
	} else {
		content = n.buildResolvedText(notification)
	}

	// Build message with text type (required for mentioned_list to work)
	msg := weChatMessage{
		MsgType: "text",
		Text: &struct {
			Content             string   `json:"content"`
			MentionedList       []string `json:"mentioned_list,omitempty"`
			MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
		}{
			Content:             content,
			MentionedList:       n.config.MentionedList,
			MentionedMobileList: n.config.MentionedMobileList,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	log.Printf("[NOTIFIER-WECHAT] 发送消息内容: %s", string(data))

	// Send to webhook
	resp, err := n.client.Post(n.config.Webhook, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("企业微信返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err == nil {
		if result.ErrCode != 0 {
			return fmt.Errorf("企业微信返回错误: %d - %s", result.ErrCode, result.ErrMsg)
		}
	}

	log.Printf("[NOTIFIER-WECHAT] ✅ 通知发送成功")
	return nil
}

// buildFiringText builds text content for firing alerts
func (n *WeChatNotifier) buildFiringText(notification AlertNotification) string {
	var content string
	content += fmt.Sprintf("🚨 告警通知\n\n")
	content += fmt.Sprintf("告警名称: %s\n", notification.AlertName)
	content += fmt.Sprintf("当前值: %.2f\n", notification.Value)

	// Add duration if available
	if notification.Duration != "" {
		content += fmt.Sprintf("持续时间要求: %s\n", notification.Duration)
	}

	// Add labels
	if len(notification.Labels) > 0 {
		content += "\n标签:\n"
		for k, v := range notification.Labels {
			content += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Add annotations (message)
	if len(notification.Annotations) > 0 {
		for k, v := range notification.Annotations {
			content += fmt.Sprintf("%s: %s\n", k, v)
		}
	}

	content += fmt.Sprintf("\n开始时间: %s", notification.StartsAt.Format("2006-01-02 15:04:05"))

	if !notification.EndsAt.IsZero() {
		content += fmt.Sprintf("\n结束时间: %s", notification.EndsAt.Format("2006-01-02 15:04:05"))
	}

	// Add call to action
	content += "\n\n请相关人员处理"

	return content
}

// buildResolvedText builds text content for resolved alerts
func (n *WeChatNotifier) buildResolvedText(notification AlertNotification) string {
	return fmt.Sprintf("✅ 告警恢复: %s\n\n当前值: %.2f\n恢复时间: %s",
		notification.AlertName,
		notification.Value,
		notification.EndsAt.Format("2006-01-02 15:04:05"))
}
