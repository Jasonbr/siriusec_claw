package channels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TelegramChannel implements the Channel interface for Telegram Bot API.
type TelegramChannel struct {
	botToken    string
	webhookURL  string // For receiving updates
	apiURL      string
	me          *TelegramBotInfo
	enabled     bool
	connected   bool
	httpClient  *http.Client
	mu          sync.RWMutex
	stopPolling chan struct{}
}

// TelegramConfig holds configuration for Telegram channel.
type TelegramConfig struct {
	BotToken   string `json:"botToken"`
	WebhookURL string `json:"webhookUrl,omitempty"` // Optional: set webhook for receiving updates
}

// TelegramBotInfo contains information about the bot.
type TelegramBotInfo struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// TelegramUser represents a Telegram user.
type TelegramUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// TelegramChat represents a Telegram chat.
type TelegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// TelegramMessage represents a Telegram message.
type TelegramMessage struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      *TelegramChat `json:"chat"`
	Date      int64         `json:"date"`
	Text      string        `json:"text,omitempty"`
	Caption   string        `json:"caption,omitempty"`
}

// TelegramUpdate represents an incoming update from Telegram.
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// NewTelegramChannel creates a new Telegram channel with the given config.
func NewTelegramChannel(cfg *TelegramConfig) *TelegramChannel {
	if cfg == nil {
		return &TelegramChannel{enabled: true}
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s", cfg.BotToken)
	return &TelegramChannel{
		botToken:    cfg.BotToken,
		webhookURL:  cfg.WebhookURL,
		apiURL:      apiURL,
		enabled:     true,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		stopPolling: make(chan struct{}),
	}
}

func NewTelegramChannelWithToken(botToken string) *TelegramChannel {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s", botToken)
	return &TelegramChannel{
		botToken:    botToken,
		apiURL:      apiURL,
		enabled:     true,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		stopPolling: make(chan struct{}),
	}
}

func (c *TelegramChannel) ID() string          { return "telegram" }
func (c *TelegramChannel) DisplayName() string { return "Telegram" }

func (c *TelegramChannel) Start() error {
	chLog.Info("telegram channel starting...")

	if c.botToken == "" {
		chLog.Info("telegram channel started (no bot token configured)")
		c.connected = false
		return nil
	}

	// Get bot info to verify token
	me, err := c.getMe()
	if err != nil {
		chLog.Error("telegram failed to get bot info: %v", err)
		return err
	}

	c.mu.Lock()
	c.me = me
	c.connected = true
	c.mu.Unlock()

	chLog.Info("telegram bot connected: @%s (ID: %d)", me.Username, me.ID)

	// Set webhook if configured
	if c.webhookURL != "" {
		if err := c.setWebhook(c.webhookURL); err != nil {
			chLog.Error("telegram failed to set webhook: %v", err)
			// Continue anyway - polling can still work
		} else {
			chLog.Info("telegram webhook set: %s", c.webhookURL)
		}
	}

	return nil
}

func (c *TelegramChannel) Stop() error {
	c.mu.Lock()
	c.connected = false
	c.me = nil
	c.mu.Unlock()

	// Stop polling if active
	close(c.stopPolling)

	// Delete webhook if set
	if c.webhookURL != "" && c.botToken != "" {
		c.deleteWebhook()
	}

	chLog.Info("telegram channel stopped")
	return nil
}

func (c *TelegramChannel) Status() *ChannelStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status := &ChannelStatus{
		ID:          "telegram",
		DisplayName: "Telegram",
		Connected:   c.connected,
		Enabled:     c.enabled,
	}
	if c.me != nil {
		status.AccountID = strconv.FormatInt(c.me.ID, 10)
		status.AccountName = "@" + c.me.Username
	}
	return status
}

// getMe retrieves bot information.
func (c *TelegramChannel) getMe() (*TelegramBotInfo, error) {
	url := c.apiURL + "/getMe"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var result struct {
		Ok          bool             `json:"ok"`
		Result      *TelegramBotInfo `json:"result"`
		Description string           `json:"description,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if !result.Ok {
		return nil, fmt.Errorf("telegram API error: %s", result.Description)
	}

	return result.Result, nil
}

// setWebhook sets the webhook URL for receiving updates.
func (c *TelegramChannel) setWebhook(webhookURL string) error {
	url := c.apiURL + "/setWebhook"

	payload := map[string]interface{}{
		"url":             webhookURL,
		"allowed_updates": []string{"message"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read webhook response failed: %w", err)
	}

	var result struct {
		Ok          bool   `json:"ok"`
		Result      bool   `json:"result"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse webhook response failed: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("telegram webhook error: %s", result.Description)
	}

	return nil
}

// deleteWebhook removes the webhook.
func (c *TelegramChannel) deleteWebhook() error {
	url := c.apiURL + "/deleteWebhook"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SendMessage sends a message via Telegram.
func (c *TelegramChannel) SendMessage(target, chatType, text string, opts *SendOptions) error {
	chLog.Debug("telegram send to=%s chatType=%s len=%d", target, chatType, len(text))

	if c.botToken == "" {
		return fmt.Errorf("telegram bot token not configured")
	}

	// Parse target as chat ID
	chatID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		// Maybe it's a username (@username)
		if strings.HasPrefix(target, "@") {
			chatID = 0 // Will use username directly
		} else {
			return fmt.Errorf("invalid chat ID: %s", target)
		}
	}

	// Determine parse mode based on opts
	parseMode := ""
	if opts != nil && opts.Format == "markdown" {
		parseMode = "MarkdownV2"
	}

	return c.sendTextMessage(chatID, target, text, parseMode, opts)
}

// sendTextMessage sends a text message.
func (c *TelegramChannel) sendTextMessage(chatID int64, target, text, parseMode string, opts *SendOptions) error {
	url := c.apiURL + "/sendMessage"

	payload := map[string]interface{}{
		"text": text,
	}

	// Use chat ID or username
	if chatID != 0 {
		payload["chat_id"] = chatID
	} else {
		payload["chat_id"] = target
	}

	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}

	// Add reply_to for threaded replies
	if opts != nil && opts.RootMessageID != "" {
		replyToID, err := strconv.ParseInt(opts.RootMessageID, 10, 64)
		if err == nil {
			payload["reply_to_message_id"] = replyToID
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send message request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var result struct {
		Ok          bool             `json:"ok"`
		Result      *TelegramMessage `json:"result,omitempty"`
		Description string           `json:"description,omitempty"`
		ErrorCode   int              `json:"error_code,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("telegram API error: %s (code: %d)", result.Description, result.ErrorCode)
	}

	if result.Result != nil {
		chLog.Debug("telegram message sent, message_id=%d", result.Result.MessageID)
	}
	return nil
}

// HandleWebhook processes incoming webhook update from Telegram.
func (c *TelegramChannel) HandleWebhook(body []byte) (*IncomingMessage, error) {
	var update TelegramUpdate

	if err := json.Unmarshal(body, &update); err != nil {
		return nil, fmt.Errorf("parse update failed: %w", err)
	}

	if update.Message == nil {
		return nil, fmt.Errorf("no message in update")
	}

	msg := update.Message

	incoming := &IncomingMessage{
		ChannelID: "telegram",
		MessageID: strconv.FormatInt(msg.MessageID, 10),
		Timestamp: msg.Date,
		Raw:       map[string]interface{}{},
	}

	// Set sender info
	if msg.From != nil {
		incoming.From = strconv.FormatInt(msg.From.ID, 10)
		incoming.FromName = msg.From.FirstName
		if msg.From.LastName != "" {
			incoming.FromName += " " + msg.From.LastName
		}
		if msg.From.Username != "" {
			incoming.FromName += " (@" + msg.From.Username + ")"
		}
	}

	// Set chat info
	if msg.Chat != nil {
		incoming.To = strconv.FormatInt(msg.Chat.ID, 10)
		switch msg.Chat.Type {
		case "private":
			incoming.ChatType = "dm"
		case "group", "supergroup":
			incoming.ChatType = "group"
			incoming.GroupID = strconv.FormatInt(msg.Chat.ID, 10)
			incoming.GroupName = msg.Chat.Title
		case "channel":
			incoming.ChatType = "channel"
		default:
			incoming.ChatType = "dm"
		}
	}

	// Extract text content
	if msg.Text != "" {
		incoming.Text = msg.Text
	} else if msg.Caption != "" {
		incoming.Text = msg.Caption
	} else {
		incoming.Text = "[non-text message]"
	}

	_ = json.Unmarshal(body, &incoming.Raw)

	chLog.Debug("telegram webhook message received: from=%s chat=%s type=%s",
		incoming.From, incoming.To, incoming.ChatType)
	return incoming, nil
}

// GetUpdates fetches updates via long polling (alternative to webhook).
func (c *TelegramChannel) GetUpdates(offset int64, timeout int) ([]TelegramUpdate, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d&allowed_updates=message",
		c.apiURL, offset, timeout)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getUpdates request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var result struct {
		Ok          bool             `json:"ok"`
		Result      []TelegramUpdate `json:"result"`
		Description string           `json:"description,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if !result.Ok {
		return nil, fmt.Errorf("telegram API error: %s", result.Description)
	}

	return result.Result, nil
}

// StartPolling starts long polling for updates.
func (c *TelegramChannel) StartPolling(handler func(*IncomingMessage)) {
	go func() {
		offset := int64(0)
		for {
			select {
			case <-c.stopPolling:
				return
			default:
			}

			updates, err := c.GetUpdates(offset, 30)
			if err != nil {
				chLog.Error("telegram polling error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, update := range updates {
				offset = update.UpdateID + 1

				// Convert to incoming message
				body, _ := json.Marshal(update)
				incoming, err := c.HandleWebhook(body)
				if err != nil {
					chLog.Error("telegram handle update error: %v", err)
					continue
				}

				if handler != nil {
					handler(incoming)
				}
			}
		}
	}()
}

// SendPhoto sends a photo message.
func (c *TelegramChannel) SendPhoto(target, photoURL, caption string, opts *SendOptions) error {
	url := c.apiURL + "/sendPhoto"

	chatID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %s", target)
	}

	payload := map[string]interface{}{
		"chat_id": chatID,
		"photo":   photoURL,
	}

	if caption != "" {
		payload["caption"] = caption
	}

	if opts != nil {
		if opts.Format == "markdown" {
			payload["parse_mode"] = "MarkdownV2"
		}
		if opts.RootMessageID != "" {
			replyToID, err := strconv.ParseInt(opts.RootMessageID, 10, 64)
			if err == nil {
				payload["reply_to_message_id"] = replyToID
			}
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send photo request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var result struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	chLog.Debug("telegram photo sent to %s", target)
	return nil
}

// SendDocument sends a document/file.
func (c *TelegramChannel) SendDocument(target, documentURL, filename, caption string, opts *SendOptions) error {
	url := c.apiURL + "/sendDocument"

	chatID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %s", target)
	}

	payload := map[string]interface{}{
		"chat_id":  chatID,
		"document": documentURL,
	}

	if filename != "" {
		payload["filename"] = filename
	}
	if caption != "" {
		payload["caption"] = caption
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send document request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var result struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	chLog.Debug("telegram document sent to %s", target)
	return nil
}
