package notifier

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/company/ems-devices/internal/alerts"
	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/internal/routes"
)

// Router integrates routing system with notification manager
type Router struct {
	config    *config.NotifierConfig
	routeMgr  *routes.Manager

	// Channel notifiers (dynamic)
	channels map[string]*ChannelNotifier
	mu       sync.RWMutex
}

// ChannelNotifier wraps a notification channel with its notifier
type ChannelNotifier struct {
	channel  routes.NotificationChannel
	wechat   *WeChatNotifier
	dingtalk *DingTalkNotifier
	feishu   *FeishuNotifier
}

// NewRouter creates a new notification router
func NewRouter(cfg *config.NotifierConfig, routeMgr *routes.Manager) *Router {
	r := &Router{
		config:   cfg,
		routeMgr: routeMgr,
		channels: make(map[string]*ChannelNotifier),
	}

	// Load initial channels
	r.reloadChannels()

	return r
}

// reloadChannels reloads channels from config
func (r *Router) reloadChannels() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing
	r.channels = make(map[string]*ChannelNotifier)

	// Load from config if routing enabled
	if r.config != nil {
		// For now, we'll create legacy channels from old config
		// New routing system channels will be loaded from routes storage
		r.createLegacyChannels()
	}
}

// createLegacyChannels creates channels from legacy config
func (r *Router) createLegacyChannels() {
	// Create legacy channels from old config
	if r.config.WeChat != nil && r.config.WeChat.Enabled {
		ch := routes.NotificationChannel{
			ID:      "wechat-default",
			Type:    "wechat",
			Name:    "默认企业微信",
			Enabled: true,
		}
		r.channels[ch.ID] = r.createChannelNotifier(ch)
		log.Printf("[ROUTER] Loaded legacy channel: %s", ch.ID)
	}

	if r.config.DingTalk != nil && r.config.DingTalk.Enabled {
		ch := routes.NotificationChannel{
			ID:      "dingtalk-default",
			Type:    "dingtalk",
			Name:    "默认钉钉",
			Enabled: true,
		}
		r.channels[ch.ID] = r.createChannelNotifier(ch)
		log.Printf("[ROUTER] Loaded legacy channel: %s", ch.ID)
	}

	if r.config.Feishu != nil && r.config.Feishu.Enabled {
		ch := routes.NotificationChannel{
			ID:      "feishu-default",
			Type:    "feishu",
			Name:    "默认飞书",
			Enabled: true,
		}
		r.channels[ch.ID] = r.createChannelNotifier(ch)
		log.Printf("[ROUTER] Loaded legacy channel: %s", ch.ID)
	}
}

// LoadChannelsFromRoutes loads channels from routes storage
func (r *Router) LoadChannelsFromRoutes(channels []routes.NotificationChannel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, ch := range channels {
		if ch.Enabled {
			r.channels[ch.ID] = r.createChannelNotifier(ch)
			log.Printf("[ROUTER] Loaded channel: %s (%s)", ch.ID, ch.Name)
		}
	}
}

// createChannelNotifier creates a notifier for a channel
func (r *Router) createChannelNotifier(ch routes.NotificationChannel) *ChannelNotifier {
	cn := &ChannelNotifier{
		channel: ch,
	}

	// Use legacy config for default channels
	switch ch.Type {
	case "wechat":
		var cfg *WeChatConfig
		if ch.ID == "wechat-default" && r.config.WeChat != nil {
			// Convert config.WeChatNotifierConfig to notifier.WeChatConfig
			cfg = &WeChatConfig{
				Enabled:             r.config.WeChat.Enabled,
				Webhook:             r.config.WeChat.Webhook,
				MentionedList:       r.config.WeChat.MentionedList,
				MentionedMobileList: r.config.WeChat.MentionedMobileList,
			}
		}
		if cfg != nil && cfg.Enabled {
			cn.wechat = NewWeChatNotifier(cfg)
		}

	case "dingtalk":
		var cfg *DingTalkConfig
		if ch.ID == "dingtalk-default" && r.config.DingTalk != nil {
			// Convert config.DingTalkNotifierConfig to notifier.DingTalkConfig
			cfg = &DingTalkConfig{
				Enabled:   r.config.DingTalk.Enabled,
				Webhook:   r.config.DingTalk.Webhook,
				Secret:    r.config.DingTalk.Secret,
				AtMobiles: r.config.DingTalk.AtMobiles,
				AtUserIDs: r.config.DingTalk.AtUserIDs,
				IsAtAll:   r.config.DingTalk.IsAtAll,
			}
		}
		if cfg != nil && cfg.Enabled {
			cn.dingtalk = NewDingTalkNotifier(cfg)
		}

	case "feishu":
		var cfg *FeishuConfig
		if ch.ID == "feishu-default" && r.config.Feishu != nil {
			// Convert config.FeishuNotifierConfig to notifier.FeishuConfig
			cfg = &FeishuConfig{
				Enabled: r.config.Feishu.Enabled,
				Webhook: r.config.Feishu.Webhook,
			}
		}
		if cfg != nil && cfg.Enabled {
			cn.feishu = NewFeishuNotifier(cfg)
		}
	}

	return cn
}

// SendNotification sends notification to evaluated channels
func (r *Router) SendNotification(ctx context.Context, alert alerts.Alert, notification AlertNotification) []NotificationResult {
	// Evaluate which channels should receive this alert
	channelIDs := r.routeMgr.EvaluateRoutes(alert, nil)

	if len(channelIDs) == 0 {
		log.Printf("[ROUTER] No channels matched for alert: %s", alert.RuleName)
		return nil
	}

	log.Printf("[ROUTER] Sending alert %s to %d channels: %v", alert.RuleName, len(channelIDs), channelIDs)

	// Send to all matched channels in parallel
	results := make([]NotificationResult, 0, len(channelIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, channelID := range channelIDs {
		wg.Add(1)
		go func(cid string) {
			defer wg.Done()

			result := r.sendToChannel(cid, notification)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(channelID)
	}

	wg.Wait()
	return results
}

// sendToChannel sends notification to a specific channel
func (r *Router) sendToChannel(channelID string, notification AlertNotification) NotificationResult {
	r.mu.RLock()
	cn, ok := r.channels[channelID]
	r.mu.RUnlock()

	if !ok {
		return NotificationResult{
			Channel: channelID,
			Success: false,
			Error:   "Channel not found",
		}
	}

	var err error
	switch cn.channel.Type {
	case "wechat":
		if cn.wechat != nil {
			err = cn.wechat.SendNotification(notification)
		} else {
			err = fmt.Errorf("WeChat notifier not initialized")
		}
	case "dingtalk":
		if cn.dingtalk != nil {
			err = cn.dingtalk.SendNotification(notification)
		} else {
			err = fmt.Errorf("DingTalk notifier not initialized")
		}
	case "feishu":
		if cn.feishu != nil {
			err = cn.feishu.SendNotification(notification)
		} else {
			err = fmt.Errorf("Feishu notifier not initialized")
		}
	default:
		err = fmt.Errorf("Unknown channel type: %s", cn.channel.Type)
	}

	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}

	return NotificationResult{
		Channel: channelID,
		Success: err == nil,
		Error:   errorStr,
	}
}

// getDefaultChannelIDs returns default channel IDs
func (r *Router) getDefaultChannelIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.channels))
	for id := range r.channels {
		ids = append(ids, id)
	}
	return ids
}

// Reload reloads channels from config
func (r *Router) Reload() {
	r.reloadChannels()
}
