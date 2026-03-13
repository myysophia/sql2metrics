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

// FeishuNotifier sends notifications to Feishu (飞书)
type FeishuNotifier struct {
	config *FeishuConfig
	client *http.Client
}

// NewFeishuNotifier creates a new Feishu notifier
func NewFeishuNotifier(config *FeishuConfig) *FeishuNotifier {
	return &FeishuNotifier{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// feishuMessage represents the message format for Feishu webhook
type feishuMessage struct {
	MsgType string      `json:"msg_type"`
	Content interface{} `json:"content"`
}

type feishuTextContent struct {
	Text string `json:"text"`
}

type feishuPostContent struct {
	Post zhCn `json:"post"`
}

type zhCn struct {
	Title   string                    `json:"title"`
	Content [][]feishuContentElement `json:"content"`
}

type feishuContentElement struct {
	Tag string `json:"tag"`
	Text string `json:"text,omitempty"`
	Href string `json:"href,omitempty"`
}

// SendNotification sends a notification to Feishu
func (n *FeishuNotifier) SendNotification(notification AlertNotification) error {
	if n.config == nil || !n.config.Enabled || n.config.Webhook == "" {
		return fmt.Errorf("飞书通知未配置或未启用")
	}

	log.Printf("[NOTIFIER-FEISHU] 准备发送通知: %s (状态: %s)", notification.AlertName, notification.Status)

	// Build message
	msg := feishuMessage{
		MsgType: "interactive",
		Content: feishuPostContent{
			Post: zhCn{
				Title: n.buildTitle(notification),
				Content: n.buildContent(notification),
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	log.Printf("[NOTIFIER-FEISHU] 发送消息: %s", string(data))

	// Send to webhook
	resp, err := n.client.Post(n.config.Webhook, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("飞书返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err == nil {
		if result.Code != 0 {
			return fmt.Errorf("飞书返回错误: %d - %s", result.Code, result.Msg)
		}
	}

	log.Printf("[NOTIFIER-FEISHU] ✅ 通知发送成功")
	return nil
}

// buildTitle builds the message title
func (n *FeishuNotifier) buildTitle(notification AlertNotification) string {
	if notification.Status == "firing" {
		return fmt.Sprintf("🚨 告警通知: %s", notification.AlertName)
	}
	return fmt.Sprintf("✅ 告警恢复: %s", notification.AlertName)
}

// buildContent builds the message content
func (n *FeishuNotifier) buildContent(notification AlertNotification) [][]feishuContentElement {
	var content [][]feishuContentElement

	// Add status line
	statusIcon := "✅"
	statusText := "已恢复"
	if notification.Status == "firing" {
		statusIcon = "🚨"
		statusText = "告警中"
	}

	content = append(content, []feishuContentElement{
		{Tag: "text", Text: fmt.Sprintf("%s 告警状态: %s", statusIcon, statusText)},
	})

	// Add current value
	content = append(content, []feishuContentElement{
		{Tag: "text", Text: fmt.Sprintf("当前值: %.2f", notification.Value)},
	})

	// Add duration if available
	if notification.Duration != "" {
		content = append(content, []feishuContentElement{
			{Tag: "text", Text: fmt.Sprintf("持续时间要求: %s", notification.Duration)},
		})
	}

	// Add labels
	if len(notification.Labels) > 0 {
		content = append(content, []feishuContentElement{
			{Tag: "text", Text: "标签:"},
		})
		for k, v := range notification.Labels {
			content = append(content, []feishuContentElement{
				{Tag: "text", Text: fmt.Sprintf("- %s: %s", k, v)},
			})
		}
	}

	// Add annotations
	if len(notification.Annotations) > 0 {
		content = append(content, []feishuContentElement{
			{Tag: "text", Text: "详情:"},
		})
		for k, v := range notification.Annotations {
			content = append(content, []feishuContentElement{
				{Tag: "text", Text: fmt.Sprintf("- %s: %s", k, v)},
			})
		}
	}

	// Add time
	content = append(content, []feishuContentElement{
		{Tag: "text", Text: fmt.Sprintf("开始时间: %s", notification.StartsAt.Format("2006-01-02 15:04:05"))},
	})

	if !notification.EndsAt.IsZero() {
		content = append(content, []feishuContentElement{
			{Tag: "text", Text: fmt.Sprintf("结束时间: %s", notification.EndsAt.Format("2006-01-02 15:04:05"))},
		})
	}

	// Add call to action for firing alerts
	if notification.Status == "firing" {
		content = append(content, []feishuContentElement{
			{Tag: "text", Text: "请相关人员处理"},
		})
	}

	return content
}
