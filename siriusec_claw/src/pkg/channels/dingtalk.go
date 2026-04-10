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

// DingTalkChannel implements the Channel interface for DingTalk with full API support.
type DingTalkChannel struct {
	appKey      string
	appSecret   string
	webhookURL  string
	accessToken string
	tokenExpire time.Time
	enabled     bool
	connected   bool
	httpClient  *http.Client
	mu          sync.RWMutex
}

// DingTalkConfig holds configuration for DingTalk channel.
type DingTalkConfig struct {
	AppKey     string `json:"appKey"`
	AppSecret  string `json:"appSecret"`
	WebhookURL string `json:"webhookUrl"`
}

// NewDingTalkChannel creates a new DingTalk channel with the given config.
func NewDingTalkChannel(cfg *DingTalkConfig) *DingTalkChannel {
	if cfg == nil {
		return &DingTalkChannel{enabled: true}
	}
	return &DingTalkChannel{
		appKey:     cfg.AppKey,
		appSecret:  cfg.AppSecret,
		webhookURL: cfg.WebhookURL,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func NewDingTalkChannelWithCreds(appKey, appSecret string) *DingTalkChannel {
	return &DingTalkChannel{
		appKey:     appKey,
		appSecret:  appSecret,
		enabled:    true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *DingTalkChannel) ID() string          { return "dingtalk" }
func (c *DingTalkChannel) DisplayName() string { return "钉钉 (DingTalk)" }

func (c *DingTalkChannel) Start() error {
	chLog.Info("dingtalk channel starting...")

	// If webhook URL is configured, we can use it for outgoing messages
	if c.webhookURL != "" {
		c.connected = true
		chLog.Info("dingtalk webhook mode enabled")
		return nil
	}

	// If app credentials are configured, get access token
	if c.appKey != "" && c.appSecret != "" {
		if err := c.refreshAccessToken(); err != nil {
			chLog.Error("dingtalk failed to get access token: %v", err)
			return err
		}
		c.connected = true
		chLog.Info("dingtalk API mode enabled, access token obtained")
		return nil
	}

	chLog.Info("dingtalk channel started (no credentials configured)")
	c.connected = false
	return nil
}

func (c *DingTalkChannel) Stop() error {
	c.mu.Lock()
	c.connected = false
	c.accessToken = ""
	c.mu.Unlock()
	chLog.Info("dingtalk channel stopped")
	return nil
}

func (c *DingTalkChannel) Status() *ChannelStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status := &ChannelStatus{
		ID:          "dingtalk",
		DisplayName: "钉钉 (DingTalk)",
		Connected:   c.connected,
		Enabled:     c.enabled,
	}
	if c.appKey != "" {
		status.AccountID = c.appKey
	}
	return status
}

// refreshAccessToken gets a new access token from DingTalk API.
func (c *DingTalkChannel) refreshAccessToken() error {
	url := "https://oapi.dingtalk.com/gettoken?appkey=" + c.appKey + "&appsecret=" + c.appSecret

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
		return fmt.Errorf("dingtalk API error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	c.mu.Lock()
	c.accessToken = result.AccessToken
	c.tokenExpire = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	c.mu.Unlock()

	chLog.Debug("dingtalk access token refreshed, expires in %d seconds", result.ExpiresIn)
	return nil
}

// getAccessToken returns a valid access token, refreshing if necessary.
func (c *DingTalkChannel) getAccessToken() (string, error) {
	c.mu.RLock()
	token := c.accessToken
	expire := c.tokenExpire
	c.mu.RUnlock()

	if token != "" && time.Now().Before(expire) {
		return token, nil
	}

	if c.appKey == "" || c.appSecret == "" {
		return "", fmt.Errorf("no app credentials configured")
	}

	if err := c.refreshAccessToken(); err != nil {
		return "", err
	}

	c.mu.RLock()
	token = c.accessToken
	c.mu.RUnlock()
	return token, nil
}

// SendMessage sends a message via DingTalk.
func (c *DingTalkChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	chLog.Debug("dingtalk send to=%s chatType=%s len=%d", target, chatType, len(text))

	// Use webhook if configured
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
func (c *DingTalkChannel) sendViaWebhook(text string, opts *SendOptions) error {
	msgType := "text"
	content := map[string]interface{}{
		"content": text,
	}

	if opts != nil && opts.Format == "markdown" {
		msgType = "markdown"
		content = map[string]interface{}{
			"title": opts.Header,
			"text":  text,
		}
	}

	payload := map[string]interface{}{
		"msgtype": msgType,
		msgType:   content,
	}

	// Add @mentions if specified
	if opts != nil && opts.Extra != nil {
		if atMobiles, ok := opts.Extra["atMobiles"].([]string); ok && len(atMobiles) > 0 {
			payload["at"] = map[string]interface{}{
				"atMobiles": atMobiles,
				"isAtAll":   opts.Extra["isAtAll"] == true,
			}
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
		return fmt.Errorf("dingtalk webhook error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	chLog.Debug("dingtalk webhook message sent successfully")
	return nil
}

// sendTextMessage sends a text message via DingTalk API.
func (c *DingTalkChannel) sendTextMessage(token, target, chatType, text string, opts *SendOptions) error {
	url := fmt.Sprintf("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token=%s", token)

	// Determine agent ID from opts or default
	agentID := ""
	if opts != nil && opts.Extra != nil {
		if id, ok := opts.Extra["agentId"].(string); ok {
			agentID = id
		}
	}

	payload := map[string]interface{}{
		"agent_id": agentID,
		"msg": map[string]interface{}{
			"msgtype": "text",
			"text": map[string]interface{}{
				"content": text,
			},
		},
	}

	// Set recipient based on chat type
	switch chatType {
	case "dm":
		payload["userid_list"] = target
	case "group":
		payload["chat_id"] = target
	default:
		payload["userid_list"] = target
	}

	return c.sendAPIRequest(url, payload)
}

// sendMarkdownMessage sends a markdown message via DingTalk API.
func (c *DingTalkChannel) sendMarkdownMessage(token, target, chatType, text string, opts *SendOptions) error {
	url := fmt.Sprintf("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token=%s", token)

	agentID := ""
	header := "消息通知"
	if opts != nil {
		if opts.Extra != nil {
			if id, ok := opts.Extra["agentId"].(string); ok {
				agentID = id
			}
		}
		if opts.Header != "" {
			header = opts.Header
		}
	}

	payload := map[string]interface{}{
		"agent_id": agentID,
		"msg": map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]interface{}{
				"title": header,
				"text":  text,
			},
		},
	}

	switch chatType {
	case "dm":
		payload["userid_list"] = target
	case "group":
		payload["chat_id"] = target
	default:
		payload["userid_list"] = target
	}

	return c.sendAPIRequest(url, payload)
}

// sendCardMessage sends an action card message via DingTalk.
func (c *DingTalkChannel) sendCardMessage(token, target, chatType, text string, opts *SendOptions) error {
	url := fmt.Sprintf("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token=%s", token)

	agentID := ""
	header := "卡片消息"
	if opts != nil {
		if opts.Extra != nil {
			if id, ok := opts.Extra["agentId"].(string); ok {
				agentID = id
			}
		}
		if opts.Header != "" {
			header = opts.Header
		}
	}

	payload := map[string]interface{}{
		"agent_id": agentID,
		"msg": map[string]interface{}{
			"msgtype": "actionCard",
			"actionCard": map[string]interface{}{
				"title":       header,
				"markdownURL": text,
				"singleTitle": "查看详情",
				"singleURL":   text,
			},
		},
	}

	switch chatType {
	case "dm":
		payload["userid_list"] = target
	case "group":
		payload["chat_id"] = target
	default:
		payload["userid_list"] = target
	}

	return c.sendAPIRequest(url, payload)
}

// sendAPIRequest sends a request to DingTalk API.
func (c *DingTalkChannel) sendAPIRequest(url string, payload map[string]interface{}) error {
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
		TaskID  int64  `json:"task_id,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse API response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk API error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	chLog.Debug("dingtalk API message sent, task_id=%d", result.TaskID)
	return nil
}

// HandleWebhook processes incoming webhook requests from DingTalk.
func (c *DingTalkChannel) HandleWebhook(body []byte) (*IncomingMessage, error) {
	var webhookData struct {
		MsgType   string `json:"msgtype"`
		TimeStamp int64  `json:"timeStamp"`
		Content   struct {
			Content string `json:"content"`
		} `json:"text,omitempty"`
		Markdown struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		} `json:"markdown,omitempty"`
		SenderNick string `json:"senderNick"`
		SenderID   string `json:"senderId"`
		ChatID     string `json:"chatid,omitempty"`
		ChatType   string `json:"chatType"`
	}

	if err := json.Unmarshal(body, &webhookData); err != nil {
		return nil, fmt.Errorf("parse webhook data failed: %w", err)
	}

	msg := &IncomingMessage{
		ChannelID: "dingtalk",
		From:      webhookData.SenderID,
		FromName:  webhookData.SenderNick,
		Timestamp: webhookData.TimeStamp / 1000, // DingTalk uses milliseconds
		Raw:       map[string]interface{}{},
	}

	if webhookData.ChatID != "" {
		msg.GroupID = webhookData.ChatID
	}

	// Determine chat type
	if webhookData.ChatType == "group" {
		msg.ChatType = "group"
	} else {
		msg.ChatType = "dm"
	}

	// Extract message content based on type
	switch webhookData.MsgType {
	case "text":
		msg.Text = webhookData.Content.Content
	case "markdown":
		msg.Text = webhookData.Markdown.Text
		if webhookData.Markdown.Title != "" {
			msg.Text = "**" + webhookData.Markdown.Title + "**\n\n" + msg.Text
		}
	default:
		msg.Text = fmt.Sprintf("[%s message]", webhookData.MsgType)
	}

	_ = json.Unmarshal(body, &msg.Raw)

	chLog.Debug("dingtalk webhook message received: from=%s type=%s", msg.From, msg.ChatType)
	return msg, nil
}
