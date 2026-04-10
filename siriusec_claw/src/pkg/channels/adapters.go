package channels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FeishuChannel implements the Channel interface for Feishu (Lark) with full API support.
type FeishuChannel struct {
	appID       string
	appSecret   string
	webhookURL  string
	accessToken string
	tokenExpire time.Time
	enabled     bool
	connected   bool
	httpClient  *http.Client
}

// FeishuConfig holds configuration for Feishu channel.
type FeishuConfig struct {
	AppID      string `json:"appId"`
	AppSecret  string `json:"appSecret"`
	WebhookURL string `json:"webhookUrl,omitempty"`
}

func NewFeishuChannel(cfg *FeishuConfig) *FeishuChannel {
	if cfg == nil {
		return &FeishuChannel{enabled: true}
	}
	return &FeishuChannel{
		appID:      cfg.AppID,
		appSecret:  cfg.AppSecret,
		webhookURL: cfg.WebhookURL,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func NewFeishuChannelWithCreds(appID, appSecret string) *FeishuChannel {
	return &FeishuChannel{
		appID:      appID,
		appSecret:  appSecret,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *FeishuChannel) ID() string          { return "feishu" }
func (c *FeishuChannel) DisplayName() string { return "飞书 (Feishu/Lark)" }

func (c *FeishuChannel) Start() error {
	chLog.Info("feishu channel starting...")

	if c.webhookURL != "" {
		c.connected = true
		chLog.Info("feishu webhook mode enabled")
		return nil
	}

	if c.appID != "" && c.appSecret != "" {
		// Get tenant access token
		url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
		payload := map[string]string{"app_id": c.appID, "app_secret": c.appSecret}
		body, _ := json.Marshal(payload)

		resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			chLog.Error("feishu auth failed: %v", err)
			return err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result struct {
			Code   int    `json:"code"`
			Msg    string `json:"msg"`
			Token  string `json:"tenant_access_token"`
			Expire int    `json:"expire"`
		}
		_ = json.Unmarshal(respBody, &result)

		if result.Code != 0 {
			chLog.Error("feishu auth error: %s (code: %d)", result.Msg, result.Code)
			return fmt.Errorf("feishu auth error: %s", result.Msg)
		}

		c.accessToken = result.Token
		c.tokenExpire = time.Now().Add(time.Duration(result.Expire-60) * time.Second)
		c.connected = true
		chLog.Info("feishu API mode enabled, token obtained")
	}

	return nil
}

func (c *FeishuChannel) Stop() error {
	c.connected = false
	c.accessToken = ""
	return nil
}

func (c *FeishuChannel) Status() *ChannelStatus {
	return &ChannelStatus{
		ID:          "feishu",
		DisplayName: "飞书 (Feishu/Lark)",
		Connected:   c.connected,
		Enabled:     c.enabled,
		AccountID:   c.appID,
	}
}

func (c *FeishuChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	chLog.Debug("feishu send to=%s chatType=%s len=%d", target, chatType, len(text))

	// Use webhook if configured
	if c.webhookURL != "" {
		msgType := "text"
		content := map[string]interface{}{"text": text}

		if opts != nil && opts.Format == "markdown" {
			msgType = "post"
			content = map[string]interface{}{
				"zh_cn": map[string]interface{}{
					"title":   opts.Header,
					"content": [][]interface{}{{map[string]string{"tag": "text", "text": text}}},
				},
			}
		}

		payload := map[string]interface{}{
			"msg_type": msgType,
			"content":  content,
		}

		body, _ := json.Marshal(payload)
		resp, err := c.httpClient.Post(c.webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("webhook request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		_ = json.Unmarshal(respBody, &result)
		if result.Code != 0 {
			return fmt.Errorf("feishu webhook error: %s (code: %d)", result.Msg, result.Code)
		}
		return nil
	}

	// Use API if token available
	if c.accessToken == "" {
		return fmt.Errorf("no access token")
	}

	url := "https://open.feishu.cn/open-api/im/v1/messages?receive_id_type=" + chatType
	payload := map[string]interface{}{
		"receive_id": target,
		"msg_type":   "text",
		"content":    fmt.Sprintf("{\"text\":\"%s\"}", text),
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// SlackChannel implements the Channel interface for Slack.
type SlackChannel struct {
	botToken   string
	appToken   string // Socket mode token
	webhookURL string
	enabled    bool
	connected  bool
	httpClient *http.Client
}

// SlackConfig holds configuration for Slack channel.
type SlackConfig struct {
	BotToken   string `json:"botToken"`
	AppToken   string `json:"appToken,omitempty"`
	WebhookURL string `json:"webhookUrl,omitempty"`
}

func NewSlackChannel(cfg *SlackConfig) *SlackChannel {
	if cfg == nil {
		return &SlackChannel{enabled: true}
	}
	return &SlackChannel{
		botToken:   cfg.BotToken,
		appToken:   cfg.AppToken,
		webhookURL: cfg.WebhookURL,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func NewSlackChannelWithToken(botToken string) *SlackChannel {
	return &SlackChannel{
		botToken:   botToken,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *SlackChannel) ID() string          { return "slack" }
func (c *SlackChannel) DisplayName() string { return "Slack" }

func (c *SlackChannel) Start() error {
	chLog.Info("slack channel starting...")

	if c.botToken != "" {
		// Verify token by calling auth.test
		url := "https://slack.com/api/auth.test"
		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+c.botToken)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			chLog.Error("slack auth failed: %v", err)
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Ok    bool   `json:"ok"`
			Error string `json:"error,omitempty"`
			User  string `json:"user,omitempty"`
			Team  string `json:"team,omitempty"`
		}
		_ = json.Unmarshal(body, &result)

		if !result.Ok {
			chLog.Error("slack auth error: %s", result.Error)
			return fmt.Errorf("slack auth error: %s", result.Error)
		}

		c.connected = true
		chLog.Info("slack connected: user=%s team=%s", result.User, result.Team)
		return nil
	}

	chLog.Info("slack channel started (no token configured)")
	return nil
}

func (c *SlackChannel) Stop() error {
	c.connected = false
	return nil
}

func (c *SlackChannel) Status() *ChannelStatus {
	return &ChannelStatus{
		ID:          "slack",
		DisplayName: "Slack",
		Connected:   c.connected,
		Enabled:     c.enabled,
	}
}

func (c *SlackChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	chLog.Debug("slack send to=%s chatType=%s len=%d", target, chatType, len(text))

	if c.botToken == "" {
		return fmt.Errorf("slack bot token not configured")
	}

	url := "https://slack.com/api/chat.postMessage"
	payload := map[string]interface{}{
		"channel": target,
		"text":    text,
	}

	if opts != nil {
		if opts.Format == "markdown" {
			payload["mrkdwn"] = true
		}
		if opts.RootMessageID != "" {
			payload["thread_ts"] = opts.RootMessageID
		}
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
		Ts    string `json:"ts,omitempty"`
	}
	_ = json.Unmarshal(respBody, &result)

	if !result.Ok {
		return fmt.Errorf("slack error: %s", result.Error)
	}

	chLog.Debug("slack message sent, ts=%s", result.Ts)
	return nil
}

// WebChatChannel is the built-in web chat channel (via Gateway WebSocket).
type WebChatChannel struct {
	enabled   bool
	connected bool
}

func NewWebChatChannel() *WebChatChannel {
	return &WebChatChannel{enabled: true, connected: true}
}

func (c *WebChatChannel) ID() string          { return "webchat" }
func (c *WebChatChannel) DisplayName() string { return "Web Chat" }
func (c *WebChatChannel) Start() error        { return nil }
func (c *WebChatChannel) Stop() error         { return nil }
func (c *WebChatChannel) Status() *ChannelStatus {
	return &ChannelStatus{ID: "webchat", DisplayName: "Web Chat", Connected: true, Enabled: true}
}
func (c *WebChatChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	// Web chat messages are delivered via WebSocket broadcast, not this method
	return nil
}
