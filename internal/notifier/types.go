package notifier

import (
	"time"

	"github.com/company/ems-devices/internal/config"
)

// NotifierConfig defines the configuration for notifications
type NotifierConfig struct {
	// Enable built-in alertmanager (if false, use external Alertmanager)
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Notification channels
	WeChat   *WeChatConfig   `yaml:"wechat,omitempty" json:"wechat,omitempty"`
	DingTalk *DingTalkConfig `yaml:"dingtalk,omitempty" json:"dingtalk,omitempty"`
	Feishu   *FeishuConfig   `yaml:"feishu,omitempty" json:"feishu,omitempty"`

	// Grouping settings
	GroupWait      string `yaml:"group_wait,omitempty" json:"group_wait,omitempty"`           // Wait time before sending first notification
	GroupInterval  string `yaml:"group_interval,omitempty" json:"group_interval,omitempty"`   // Wait time between sending notifications for the same group
	RepeatInterval string `yaml:"repeat_interval,omitempty" json:"repeat_interval,omitempty"` // How long to wait before resending a notification
}

// WeChatConfig defines WeChat Work (企业微信) webhook configuration
type WeChatConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Webhook string   `yaml:"webhook" json:"webhook"`
	// Mention users
	MentionedList       []string `yaml:"mentioned_list,omitempty" json:"mentioned_list,omitempty"`
	MentionedMobileList []string `yaml:"mentioned_mobile_list,omitempty" json:"mentioned_mobile_list,omitempty"`
}

// DingTalkConfig defines DingTalk (钉钉) webhook configuration
type DingTalkConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Webhook  string   `yaml:"webhook" json:"webhook"`
	Secret   string   `yaml:"secret,omitempty" json:"secret,omitempty"` // For signature verification
	AtMobiles []string `yaml:"at_mobiles,omitempty" json:"at_mobiles,omitempty"`
	AtUserIDs []string `yaml:"at_user_ids,omitempty" json:"at_user_ids,omitempty"`
	IsAtAll    bool     `yaml:"is_at_all,omitempty" json:"is_at_all,omitempty"`
}

// FeishuConfig defines Feishu (飞书) webhook configuration
type FeishuConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Webhook string `yaml:"webhook" json:"webhook"`
}

// FromConfig converts config.NotifierConfig to notifier.NotifierConfig
func FromConfig(cfg *config.NotifierConfig) *NotifierConfig {
	if cfg == nil {
		return nil
	}

	result := &NotifierConfig{
		Enabled:        cfg.Enabled,
		GroupWait:      cfg.GroupWait,
		GroupInterval:  cfg.GroupInterval,
		RepeatInterval: cfg.RepeatInterval,
	}

	if cfg.WeChat != nil {
		result.WeChat = &WeChatConfig{
			Enabled:             cfg.WeChat.Enabled,
			Webhook:             cfg.WeChat.Webhook,
			MentionedList:       cfg.WeChat.MentionedList,
			MentionedMobileList: cfg.WeChat.MentionedMobileList,
		}
	}

	if cfg.DingTalk != nil {
		result.DingTalk = &DingTalkConfig{
			Enabled:   cfg.DingTalk.Enabled,
			Webhook:   cfg.DingTalk.Webhook,
			Secret:    cfg.DingTalk.Secret,
			AtMobiles: cfg.DingTalk.AtMobiles,
			AtUserIDs: cfg.DingTalk.AtUserIDs,
			IsAtAll:   cfg.DingTalk.IsAtAll,
		}
	}

	if cfg.Feishu != nil {
		result.Feishu = &FeishuConfig{
			Enabled: cfg.Feishu.Enabled,
			Webhook: cfg.Feishu.Webhook,
		}
	}

	return result
}

// AlertNotification represents a notification to be sent
type AlertNotification struct {
	AlertName   string            `json:"alert_name"`
	Status      string            `json:"status"` // firing, resolved
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      time.Time         `json:"ends_at,omitempty"`
	Value       float64           `json:"value"`      // 当前指标值
	Duration    string            `json:"duration,omitempty"` // 告警持续时间要求
}

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	Channel string `json:"channel"` // wechat, dingtalk, feishu
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
