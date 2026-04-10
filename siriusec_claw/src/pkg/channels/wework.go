package channels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// WeWorkChannel implements the Channel interface for WeCom (企业微信) with full API support.
type WeWorkChannel struct {
	corpID      string
	agentID     string
	secret      string
	webhookURL  string // Robot webhook key
	accessToken string
	tokenExpire time.Time
	enabled     bool
	connected   bool
	httpClient  *http.Client
	mu          sync.RWMutex
}

// WeWorkConfig holds configuration for WeWork channel.
type WeWorkConfig struct {
	CorpID     string `json:"corpId"`
	AgentID    string `json:"agentId"`
	Secret     string `json:"secret"`
	WebhookKey string `json:"webhookKey"`
}

// NewWeWorkChannel creates a new WeWork channel with the given config.
func NewWeWorkChannel(cfg *WeWorkConfig) *WeWorkChannel {
	if cfg == nil {
		return &WeWorkChannel{enabled: true}
	}
	webhookURL := ""
	if cfg.WebhookKey != "" {
		webhookURL = fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", cfg.WebhookKey)
	}
	return &WeWorkChannel{
		corpID:     cfg.CorpID,
		agentID:    cfg.AgentID,
		secret:     cfg.Secret,
		webhookURL: webhookURL,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func NewWeWorkChannelWithCreds(corpID, agentID, secret string) *WeWorkChannel {
	return &WeWorkChannel{
		corpID:     corpID,
		agentID:    agentID,
		secret:     secret,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *WeWorkChannel) ID() string          { return "wework" }
func (c *WeWorkChannel) DisplayName() string { return "企业微信 (WeCom)" }

func (c *WeWorkChannel) Start() error {
	chLog.Info("wework channel starting...")

	// If webhook URL is configured, use robot mode
	if c.webhookURL != "" {
		c.connected = true
		chLog.Info("wework robot webhook mode enabled")
		return nil
	}

	// If corp credentials are configured, get access token
	if c.corpID != "" && c.secret != "" {
		if err := c.refreshAccessToken(); err != nil {
			chLog.Error("wework failed to get access token: %v", err)
			return err
		}
		c.connected = true
		chLog.Info("wework API mode enabled, access token obtained")
		return nil
	}

	chLog.Info("wework channel started (no credentials configured)")
	c.connected = false
	return nil
}

func (c *WeWorkChannel) Stop() error {
	c.mu.Lock()
	c.connected = false
	c.accessToken = ""
	c.mu.Unlock()
	chLog.Info("wework channel stopped")
	return nil
}

func (c *WeWorkChannel) Status() *ChannelStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status := &ChannelStatus{
		ID:          "wework",
		DisplayName: "企业微信 (WeCom)",
		Connected:   c.connected,
		Enabled:     c.enabled,
	}
	if c.corpID != "" {
		status.AccountID = c.corpID
	}
	if c.agentID != "" {
		status.AccountName = "Agent " + c.agentID
	}
	return status
}

// refreshAccessToken gets a new access token from WeWork API.
func (c *WeWorkChannel) refreshAccessToken() error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", c.corpID, c.secret)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wework API error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	c.mu.Lock()
	c.accessToken = result.AccessToken
	c.tokenExpire = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	c.mu.Unlock()

	chLog.Debug("wework access token refreshed, expires in %d seconds", result.ExpiresIn)
	return nil
}

// getAccessToken returns a valid access token, refreshing if necessary.
func (c *WeWorkChannel) getAccessToken() (string, error) {
	c.mu.RLock()
	token := c.accessToken
	expire := c.tokenExpire
	c.mu.RUnlock()

	if token != "" && time.Now().Before(expire) {
		return token, nil
	}

	if c.corpID == "" || c.secret == "" {
		return "", fmt.Errorf("no corp credentials configured")
	}

	if err := c.refreshAccessToken(); err != nil {
		return "", err
	}

	c.mu.RLock()
	token = c.accessToken
	c.mu.RUnlock()
	return token, nil
}

// SendMessage sends a message via WeWork.
func (c *WeWorkChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	chLog.Debug("wework send to=%s chatType=%s len=%d", target, chatType, len(text))

	// Use robot webhook if configured
	if c.webhookURL != "" {
		return c.sendViaWebhook(text, opts)
	}

	// Use API if credentials are configured
	token, err := c.getAccessToken()
	if err != nil {
		return fmt.Errorf("no access token: %w", err)
	}

	// Determine message type based on opts
	format := "text"
	if opts != nil && opts.Format != "" {
		format = opts.Format
	}

	switch format {
	case "markdown":
		return c.sendMarkdownMessage(token, target, chatType, text, opts)
	case "card":
		return c.sendCardMessage(token, target, chatType, text, opts)
	default:
		return c.sendTextMessage(token, target, chatType, text, opts)
	}
}

// sendViaWebhook sends message using robot webhook.
func (c *WeWorkChannel) sendViaWebhook(text string, opts *SendOptions) error {
	msgType := "text"
	content := map[string]interface{}{
		"content": text,
	}

	if opts != nil && opts.Format == "markdown" {
		msgType = "markdown"
		content = map[string]interface{}{
			"content": text,
		}
	}

	payload := map[string]interface{}{
		"msgtype": msgType,
		msgType:   content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(c.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read webhook response failed: %w", err)
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse webhook response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wework webhook error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	chLog.Debug("wework webhook message sent successfully")
	return nil
}

// sendTextMessage sends a text message via WeWork API.
func (c *WeWorkChannel) sendTextMessage(token, target, chatType, text string, opts *SendOptions) error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": text,
		},
		"safe": 0,
	}

	// Set agent ID
	if c.agentID != "" {
		payload["agentid"] = c.agentID
	}

	// Set recipient based on chat type
	switch chatType {
	case "dm":
		payload["touser"] = target
	case "group":
		payload["toparty"] = target
	default:
		payload["touser"] = target
	}

	return c.sendAPIRequest(url, payload)
}

// sendMarkdownMessage sends a markdown message via WeWork API.
func (c *WeWorkChannel) sendMarkdownMessage(token, target, chatType, text string, opts *SendOptions) error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": text,
		},
		"safe": 0,
	}

	if c.agentID != "" {
		payload["agentid"] = c.agentID
	}

	switch chatType {
	case "dm":
		payload["touser"] = target
	case "group":
		payload["toparty"] = target
	default:
		payload["touser"] = target
	}

	return c.sendAPIRequest(url, payload)
}

// sendCardMessage sends a textcard message via WeWork.
func (c *WeWorkChannel) sendCardMessage(token, target, chatType, text string, opts *SendOptions) error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	header := "卡片消息"
	if opts != nil && opts.Header != "" {
		header = opts.Header
	}

	description := text
	urlLink := "#"
	if opts != nil && opts.Extra != nil {
		if u, ok := opts.Extra["url"].(string); ok {
			urlLink = u
		}
	}

	payload := map[string]interface{}{
		"msgtype": "textcard",
		"textcard": map[string]interface{}{
			"title":       header,
			"description": description,
			"url":         urlLink,
			"btntxt":      "查看详情",
		},
		"safe": 0,
	}

	if c.agentID != "" {
		payload["agentid"] = c.agentID
	}

	switch chatType {
	case "dm":
		payload["touser"] = target
	case "group":
		payload["toparty"] = target
	default:
		payload["touser"] = target
	}

	return c.sendAPIRequest(url, payload)
}

// sendAPIRequest sends a request to WeWork API.
func (c *WeWorkChannel) sendAPIRequest(url string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read API response failed: %w", err)
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		MsgID   string `json:"msgid,omitempty"`
		Invalid string `json:"invaliduser,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse API response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wework API error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	chLog.Debug("wework API message sent, msgid=%s", result.MsgID)
	return nil
}

// HandleWebhook processes incoming callback from WeWork.
func (c *WeWorkChannel) HandleWebhook(body []byte) (*IncomingMessage, error) {
	var callbackData struct {
		ToUserName   string `json:"ToUserName"`
		FromUserName string `json:"FromUserName"`
		CreateTime   int64  `json:"CreateTime"`
		MsgType      string `json:"MsgType"`
		Content      string `json:"Content,omitempty"`
		AgentID      string `json:"AgentID,omitempty"`
		ChatID       string `json:"ChatId,omitempty"`
	}

	if err := json.Unmarshal(body, &callbackData); err != nil {
		return nil, fmt.Errorf("parse callback data failed: %w", err)
	}

	msg := &IncomingMessage{
		ChannelID: "wework",
		From:      callbackData.FromUserName,
		To:        callbackData.ToUserName,
		Timestamp: callbackData.CreateTime,
		Raw:       map[string]interface{}{},
	}

	// Determine chat type based on FromUserName format
	if len(callbackData.FromUserName) > 0 && callbackData.FromUserName[0] == 'c' {
		msg.ChatType = "group"
		msg.GroupID = callbackData.ChatID
	} else {
		msg.ChatType = "dm"
	}

	// Extract message content
	switch callbackData.MsgType {
	case "text":
		msg.Text = callbackData.Content
	default:
		msg.Text = fmt.Sprintf("[%s message]", callbackData.MsgType)
	}

	_ = json.Unmarshal(body, &msg.Raw)

	chLog.Debug("wework callback message received: from=%s type=%s", msg.From, msg.ChatType)
	return msg, nil
}

// WeixinChannel (微信个人号) - limited support via business webhook.
type WeixinChannel struct {
	webhookURL string
	enabled    bool
	connected  bool
	httpClient *http.Client
}

// WeixinConfig holds configuration for Weixin channel.
type WeixinConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

func NewWeixinChannel(cfg *WeixinConfig) *WeixinChannel {
	if cfg == nil {
		return &WeixinChannel{enabled: true}
	}
	return &WeixinChannel{
		webhookURL: cfg.WebhookURL,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *WeixinChannel) ID() string          { return "weixin" }
func (c *WeixinChannel) DisplayName() string { return "微信 (Weixin)" }

func (c *WeixinChannel) Start() error {
	chLog.Info("weixin channel starting...")
	if c.webhookURL != "" {
		c.connected = true
		chLog.Info("weixin webhook mode enabled")
	} else {
		chLog.Info("weixin channel started (no webhook configured)")
	}
	return nil
}

func (c *WeixinChannel) Stop() error {
	c.connected = false
	chLog.Info("weixin channel stopped")
	return nil
}

func (c *WeixinChannel) Status() *ChannelStatus {
	return &ChannelStatus{
		ID:          "weixin",
		DisplayName: "微信 (Weixin)",
		Connected:   c.connected,
		Enabled:     c.enabled,
	}
}

func (c *WeixinChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	chLog.Debug("weixin send to=%s chatType=%s len=%d", target, chatType, len(text))

	if c.webhookURL == "" {
		return fmt.Errorf("weixin webhook not configured")
	}

	payload := map[string]interface{}{
		"target":  target,
		"type":    chatType,
		"content": text,
	}

	if opts != nil {
		if opts.Header != "" {
			payload["title"] = opts.Header
		}
		if opts.Format != "" {
			payload["format"] = opts.Format
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(c.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		// Non-standard response, just check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			chLog.Debug("weixin message sent (status: %d)", resp.StatusCode)
			return nil
		}
		return fmt.Errorf("weixin send failed (status: %d)", resp.StatusCode)
	}

	if !result.Success {
		return fmt.Errorf("weixin error: %s", result.Error)
	}

	chLog.Debug("weixin message sent successfully")
	return nil
}
