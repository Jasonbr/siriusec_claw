package handlers

import (
	"github.com/siriusec/siriusec_claw/pkg/channels"
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/memory"
	"github.com/siriusec/siriusec_claw/pkg/security"
)

var (
	channelManager *channels.Manager
	approvalQueue  *security.ApprovalQueue
	memoryStore    *memory.Store
)

func init() {
	channelManager = channels.NewManager()
	channelManager.Register(channels.NewWebChatChannel())
	approvalQueue = security.NewApprovalQueue(120000)
}

// InitChannelsFromConfig initializes channels from configuration.
func InitChannelsFromConfig(cfg *config.ClawConfig) {
	if cfg == nil || cfg.Channels == nil {
		return
	}

	chCfg := cfg.Channels

	// Initialize DingTalk
	dingtalk := chCfg.GetChannelConfig("dingtalk")
	if dingtalk != nil {
		appKey := getStringFromMap(dingtalk, "appKey", "")
		appSecret := getStringFromMap(dingtalk, "appSecret", "")
		webhookURL := getStringFromMap(dingtalk, "webhookUrl", "")

		if appKey != "" || webhookURL != "" {
			dingCfg := &channels.DingTalkConfig{
				AppKey:     appKey,
				AppSecret:  appSecret,
				WebhookURL: webhookURL,
			}
			ch := channels.NewDingTalkChannel(dingCfg)
			channelManager.Register(ch)
		}
	}

	// Initialize WeWork (Enterprise WeChat)
	wework := chCfg.GetChannelConfig("wework")
	if wework != nil {
		corpID := getStringFromMap(wework, "corpId", "")
		agentID := getStringFromMap(wework, "agentId", "")
		secret := getStringFromMap(wework, "secret", "")
		webhookKey := getStringFromMap(wework, "webhookKey", "")

		if corpID != "" || webhookKey != "" {
			wwCfg := &channels.WeWorkConfig{
				CorpID:     corpID,
				AgentID:    agentID,
				Secret:     secret,
				WebhookKey: webhookKey,
			}
			ch := channels.NewWeWorkChannel(wwCfg)
			channelManager.Register(ch)
		}
	}

	// Initialize Weixin (Personal WeChat)
	weixin := chCfg.GetChannelConfig("weixin")
	if weixin != nil {
		webhookURL := getStringFromMap(weixin, "webhookUrl", "")
		if webhookURL != "" {
			wxCfg := &channels.WeixinConfig{
				WebhookURL: webhookURL,
			}
			ch := channels.NewWeixinChannel(wxCfg)
			channelManager.Register(ch)
		}
	}

	// Initialize Telegram
	telegram := chCfg.GetChannelConfig("telegram")
	if telegram != nil {
		botToken := getStringFromMap(telegram, "botToken", "")
		webhookURL := getStringFromMap(telegram, "webhookUrl", "")
		if botToken != "" {
			tgCfg := &channels.TelegramConfig{
				BotToken:   botToken,
				WebhookURL: webhookURL,
			}
			ch := channels.NewTelegramChannel(tgCfg)
			channelManager.Register(ch)
		}
	}

	// Initialize Slack
	slack := chCfg.GetChannelConfig("slack")
	if slack != nil {
		botToken := getStringFromMap(slack, "botToken", "")
		webhookURL := getStringFromMap(slack, "webhookUrl", "")
		if botToken != "" || webhookURL != "" {
			slCfg := &channels.SlackConfig{
				BotToken:   botToken,
				WebhookURL: webhookURL,
			}
			ch := channels.NewSlackChannel(slCfg)
			channelManager.Register(ch)
		}
	}

	// Initialize Feishu
	feishu := chCfg.GetChannelConfig("feishu")
	if feishu != nil {
		appID := getStringFromMap(feishu, "appId", "")
		appSecret := getStringFromMap(feishu, "appSecret", "")
		webhookURL := getStringFromMap(feishu, "webhookUrl", "")
		if appID != "" || webhookURL != "" {
			fsCfg := &channels.FeishuConfig{
				AppID:      appID,
				AppSecret:  appSecret,
				WebhookURL: webhookURL,
			}
			ch := channels.NewFeishuChannel(fsCfg)
			channelManager.Register(ch)
		}
	}
}

func getStringFromMap(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

// GetChannelManager returns the global channel manager.
func GetChannelManager() *channels.Manager {
	return channelManager
}

func getMemoryStore() *memory.Store {
	if memoryStore == nil {
		var err error
		memoryStore, err = memory.NewStore(func(k string) string { return "" })
		if err != nil {
			return nil
		}
	}
	return memoryStore
}

// --- channels.status ---

func ChannelsStatusHandler(opts HandlerOpts) error {
	statuses := channelManager.StatusAll()
	opts.Respond(true, map[string]interface{}{
		"channels": statuses,
		"count":    len(statuses),
	}, nil, nil)
	return nil
}

// --- channels.logout ---

func ChannelsLogoutHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "channelId", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("channelId required"), nil)
		return nil
	}
	ch := channelManager.Get(id)
	if ch == nil {
		opts.Respond(false, nil, errInvalidParams("channel not found: "+id), nil)
		return nil
	}
	_ = ch.Stop()
	opts.Respond(true, map[string]interface{}{"ok": true, "channelId": id}, nil, nil)
	return nil
}

// --- approvals.list ---

func ApprovalsListHandler(opts HandlerOpts) error {
	pending := approvalQueue.List()
	opts.Respond(true, map[string]interface{}{
		"approvals": pending,
		"count":     len(pending),
	}, nil, nil)
	return nil
}

// --- approvals.approve ---

func ApprovalsApproveHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}
	ok := approvalQueue.Approve(id, "operator")
	opts.Respond(true, map[string]interface{}{"ok": ok, "id": id}, nil, nil)
	return nil
}

// --- approvals.deny ---

func ApprovalsDenyHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}
	reason := stringParam(opts.Params, "reason", "")
	ok := approvalQueue.Deny(id, "operator", reason)
	opts.Respond(true, map[string]interface{}{"ok": ok, "id": id}, nil, nil)
	return nil
}

// --- approvals.whitelistSession ---

func ApprovalsWhitelistSessionHandler(opts HandlerOpts) error {
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	if sessionKey == "" {
		opts.Respond(false, nil, errInvalidParams("sessionKey required"), nil)
		return nil
	}
	approvalQueue.WhitelistSession(sessionKey)
	opts.Respond(true, map[string]interface{}{"ok": true, "sessionKey": sessionKey}, nil, nil)
	return nil
}

// --- memory.list ---

func MemoryListHandler(opts HandlerOpts) error {
	store := getMemoryStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("memory store not available"), nil)
		return nil
	}
	category := stringParam(opts.Params, "category", "")
	entries := store.List(category)
	opts.Respond(true, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	}, nil, nil)
	return nil
}

// --- memory.get ---

func MemoryGetHandler(opts HandlerOpts) error {
	store := getMemoryStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("memory store not available"), nil)
		return nil
	}
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}
	entry := store.Get(id)
	if entry == nil {
		opts.Respond(false, nil, errInvalidParams("entry not found: "+id), nil)
		return nil
	}
	opts.Respond(true, entry, nil, nil)
	return nil
}

// --- memory.add ---

func MemoryAddHandler(opts HandlerOpts) error {
	store := getMemoryStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("memory store not available"), nil)
		return nil
	}
	entry := &memory.Entry{
		Title:    stringParam(opts.Params, "title", ""),
		Content:  stringParam(opts.Params, "content", ""),
		Source:   stringParam(opts.Params, "source", "user"),
		Category: stringParam(opts.Params, "category", "note"),
	}
	if entry.Title == "" && entry.Content == "" {
		opts.Respond(false, nil, errInvalidParams("title or content required"), nil)
		return nil
	}
	if tags, ok := opts.Params["tags"]; ok {
		if tagSlice, ok := tags.([]interface{}); ok {
			for _, t := range tagSlice {
				if s, ok := t.(string); ok {
					entry.Tags = append(entry.Tags, s)
				}
			}
		}
	}
	if err := store.Add(entry); err != nil {
		opts.Respond(false, nil, errInternal("failed to add entry: "+err.Error()), nil)
		return nil
	}
	opts.Respond(true, entry, nil, nil)
	return nil
}

// --- memory.update ---

func MemoryUpdateHandler(opts HandlerOpts) error {
	store := getMemoryStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("memory store not available"), nil)
		return nil
	}
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}
	existing := store.Get(id)
	if existing == nil {
		opts.Respond(false, nil, errInvalidParams("entry not found: "+id), nil)
		return nil
	}
	if v := stringParam(opts.Params, "title", ""); v != "" {
		existing.Title = v
	}
	if v := stringParam(opts.Params, "content", ""); v != "" {
		existing.Content = v
	}
	if v := stringParam(opts.Params, "category", ""); v != "" {
		existing.Category = v
	}
	if tags, ok := opts.Params["tags"]; ok {
		if tagSlice, ok := tags.([]interface{}); ok {
			existing.Tags = nil
			for _, t := range tagSlice {
				if s, ok := t.(string); ok {
					existing.Tags = append(existing.Tags, s)
				}
			}
		}
	}
	if err := store.Update(existing); err != nil {
		opts.Respond(false, nil, errInternal("failed to update: "+err.Error()), nil)
		return nil
	}
	opts.Respond(true, existing, nil, nil)
	return nil
}

// --- memory.delete ---

func MemoryDeleteHandler(opts HandlerOpts) error {
	store := getMemoryStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("memory store not available"), nil)
		return nil
	}
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}
	if err := store.Delete(id); err != nil {
		opts.Respond(false, nil, errInternal("failed to delete: "+err.Error()), nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"ok": true, "id": id}, nil, nil)
	return nil
}

// --- memory.search ---

func MemorySearchHandler(opts HandlerOpts) error {
	store := getMemoryStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("memory store not available"), nil)
		return nil
	}
	query := stringParam(opts.Params, "query", "")
	if query == "" {
		opts.Respond(false, nil, errInvalidParams("query required"), nil)
		return nil
	}
	limit := intParam(opts.Params, "limit", 20)
	results := store.Search(query, limit)
	opts.Respond(true, map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   query,
	}, nil, nil)
	return nil
}

// --- channels.start ---

func ChannelsStartHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "channelId", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("channelId required"), nil)
		return nil
	}
	ch := channelManager.Get(id)
	if ch == nil {
		opts.Respond(false, nil, errInvalidParams("channel not found: "+id), nil)
		return nil
	}
	if err := ch.Start(); err != nil {
		opts.Respond(false, nil, errInternal("failed to start channel: "+err.Error()), nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"channelId": id,
		"status":    ch.Status(),
	}, nil, nil)
	return nil
}

// --- channels.restart ---

func ChannelsRestartHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "channelId", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("channelId required"), nil)
		return nil
	}
	ch := channelManager.Get(id)
	if ch == nil {
		opts.Respond(false, nil, errInvalidParams("channel not found: "+id), nil)
		return nil
	}
	_ = ch.Stop()
	if err := ch.Start(); err != nil {
		opts.Respond(false, nil, errInternal("failed to restart channel: "+err.Error()), nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"channelId": id,
		"status":    ch.Status(),
	}, nil, nil)
	return nil
}

// --- channels.send ---

func ChannelsSendHandler(opts HandlerOpts) error {
	channelID := stringParam(opts.Params, "channelId", "")
	target := stringParam(opts.Params, "target", "")
	chatType := stringParam(opts.Params, "chatType", "dm")
	text := stringParam(opts.Params, "text", "")

	if channelID == "" {
		opts.Respond(false, nil, errInvalidParams("channelId required"), nil)
		return nil
	}
	if target == "" {
		opts.Respond(false, nil, errInvalidParams("target required"), nil)
		return nil
	}
	if text == "" {
		opts.Respond(false, nil, errInvalidParams("text required"), nil)
		return nil
	}

	// Build send options
	sendOpts := &channels.SendOptions{
		Format: stringParam(opts.Params, "format", "text"),
		Header: stringParam(opts.Params, "header", ""),
	}

	// Extract extra options
	if extra, ok := opts.Params["extra"].(map[string]interface{}); ok {
		sendOpts.Extra = extra
	}

	if err := channelManager.DeliverMessage(channelID, target, chatType, text, sendOpts); err != nil {
		opts.Respond(false, nil, errInternal("failed to send message: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"channelId": channelID,
		"target":    target,
		"chatType":  chatType,
	}, nil, nil)
	return nil
}

// --- channels.startAll ---

func ChannelsStartAllHandler(opts HandlerOpts) error {
	channelManager.StartAll()
	statuses := channelManager.StatusAll()
	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"channels": statuses,
		"count":    len(statuses),
	}, nil, nil)
	return nil
}

// --- channels.stopAll ---

func ChannelsStopAllHandler(opts HandlerOpts) error {
	channelManager.StopAll()
	statuses := channelManager.StatusAll()
	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"channels": statuses,
		"count":    len(statuses),
	}, nil, nil)
	return nil
}

// --- channels.configure ---

func ChannelsConfigureHandler(opts HandlerOpts) error {
	channelID := stringParam(opts.Params, "channelId", "")
	if channelID == "" {
		opts.Respond(false, nil, errInvalidParams("channelId required"), nil)
		return nil
	}

	configParams, ok := opts.Params["config"].(map[string]interface{})
	if !ok {
		opts.Respond(false, nil, errInvalidParams("config object required"), nil)
		return nil
	}

	// Stop existing channel if present
	existing := channelManager.Get(channelID)
	if existing != nil {
		existing.Stop()
	}

	// Create new channel based on type
	var newChannel channels.Channel
	switch channelID {
	case "dingtalk":
		dingCfg := &channels.DingTalkConfig{
			AppKey:     getStringFromMap(configParams, "appKey", ""),
			AppSecret:  getStringFromMap(configParams, "appSecret", ""),
			WebhookURL: getStringFromMap(configParams, "webhookUrl", ""),
		}
		newChannel = channels.NewDingTalkChannel(dingCfg)
	case "wework":
		wwCfg := &channels.WeWorkConfig{
			CorpID:     getStringFromMap(configParams, "corpId", ""),
			AgentID:    getStringFromMap(configParams, "agentId", ""),
			Secret:     getStringFromMap(configParams, "secret", ""),
			WebhookKey: getStringFromMap(configParams, "webhookKey", ""),
		}
		newChannel = channels.NewWeWorkChannel(wwCfg)
	case "weixin":
		wxCfg := &channels.WeixinConfig{
			WebhookURL: getStringFromMap(configParams, "webhookUrl", ""),
		}
		newChannel = channels.NewWeixinChannel(wxCfg)
	case "telegram":
		tgCfg := &channels.TelegramConfig{
			BotToken:   getStringFromMap(configParams, "botToken", ""),
			WebhookURL: getStringFromMap(configParams, "webhookUrl", ""),
		}
		newChannel = channels.NewTelegramChannel(tgCfg)
	case "slack":
		slCfg := &channels.SlackConfig{
			BotToken:   getStringFromMap(configParams, "botToken", ""),
			WebhookURL: getStringFromMap(configParams, "webhookUrl", ""),
		}
		newChannel = channels.NewSlackChannel(slCfg)
	case "feishu":
		fsCfg := &channels.FeishuConfig{
			AppID:      getStringFromMap(configParams, "appId", ""),
			AppSecret:  getStringFromMap(configParams, "appSecret", ""),
			WebhookURL: getStringFromMap(configParams, "webhookUrl", ""),
		}
		newChannel = channels.NewFeishuChannel(fsCfg)
	default:
		opts.Respond(false, nil, errInvalidParams("unknown channel type: "+channelID), nil)
		return nil
	}

	// Start the new channel
	if err := newChannel.Start(); err != nil {
		opts.Respond(false, nil, errInternal("failed to start channel: "+err.Error()), nil)
		return nil
	}

	// Register the channel
	channelManager.Register(newChannel)

	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"channelId": channelID,
		"status":    newChannel.Status(),
	}, nil, nil)
	return nil
}
