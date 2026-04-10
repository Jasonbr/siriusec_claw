package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/channels"
	"github.com/siriusec/siriusec_claw/pkg/logging"
)

var whLog = logging.Sub("webhooks")

// Handler handles incoming webhooks from various channels.
type Handler struct {
	manager       *channels.Manager
	messageRouter MessageRouter
	secrets       map[string]string // channel ID -> secret
	mu            sync.RWMutex
}

// MessageRouter defines how incoming messages are routed to agents.
type MessageRouter interface {
	// RouteMessage processes an incoming message and optionally triggers an agent.
	RouteMessage(msg *channels.IncomingMessage) error
}

// HandlerConfig configures the webhook handler.
type HandlerConfig struct {
	Secrets map[string]string // channel secrets for signature verification
}

// NewHandler creates a new webhook handler.
func NewHandler(manager *channels.Manager, router MessageRouter, cfg *HandlerConfig) *Handler {
	secrets := make(map[string]string)
	if cfg != nil && cfg.Secrets != nil {
		secrets = cfg.Secrets
	}
	return &Handler{
		manager:       manager,
		messageRouter: router,
		secrets:       secrets,
	}
}

// SetSecret sets the secret for a channel.
func (h *Handler) SetSecret(channelID, secret string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.secrets[channelID] = secret
}

// ServeHTTP handles incoming webhook requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Extract channel from path: /hooks/{channel} or /hooks/{channel}/{subpath}
	path := strings.TrimPrefix(r.URL.Path, "/hooks/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.SplitN(path, "/", 2)

	channelID := parts[0]
	if channelID == "" {
		http.Error(w, `{"error":"channel ID required in path"}`, http.StatusBadRequest)
		return
	}

	whLog.Debug("webhook received: channel=%s method=%s path=%s", channelID, r.Method, r.URL.Path)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
		return
	}

	// Verify signature if secret is configured
	if !h.verifySignature(channelID, r, body) {
		whLog.Warn("webhook signature verification failed for channel: %s", channelID)
		http.Error(w, `{"error":"signature verification failed"}`, http.StatusUnauthorized)
		return
	}

	// Route to appropriate channel handler
	var response interface{}
	var httpStatus int = http.StatusOK

	switch channelID {
	case "dingtalk":
		response, httpStatus = h.handleDingTalk(r, body)
	case "wework", "weixin":
		response, httpStatus = h.handleWeWork(r, body)
	case "telegram":
		response, httpStatus = h.handleTelegram(r, body)
	case "feishu":
		response, httpStatus = h.handleFeishu(r, body)
	case "slack":
		response, httpStatus = h.handleSlack(r, body)
	default:
		// Generic webhook handling
		response, httpStatus = h.handleGeneric(channelID, r, body)
	}

	// Log processing time
	duration := time.Since(start)
	whLog.Debug("webhook processed: channel=%s status=%d duration=%s", channelID, httpStatus, duration)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if response != nil {
		json.NewEncoder(w).Encode(response)
	} else {
		w.Write([]byte(`{"ok":true}`))
	}
}

// verifySignature verifies the webhook signature for the channel.
func (h *Handler) verifySignature(channelID string, r *http.Request, body []byte) bool {
	h.mu.RLock()
	secret, hasSecret := h.secrets[channelID]
	h.mu.RUnlock()

	if !hasSecret || secret == "" {
		return true // No secret configured, skip verification
	}

	switch channelID {
	case "dingtalk":
		return h.verifyDingTalkSignature(secret, r, body)
	case "wework", "weixin":
		// WeWork uses different verification
		return true
	case "feishu":
		return h.verifyFeishuSignature(secret, r, body)
	default:
		return true
	}
}

// verifyDingTalkSignature verifies DingTalk webhook signature.
func (h *Handler) verifyDingTalkSignature(secret string, r *http.Request, body []byte) bool {
	timestamp := r.URL.Query().Get("timestamp")
	sign := r.URL.Query().Get("sign")

	if timestamp == "" || sign == "" {
		return true // No signature params, skip verification
	}

	// Calculate expected signature
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	expectedSign := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sign), []byte(expectedSign))
}

// verifyFeishuSignature verifies Feishu webhook signature.
func (h *Handler) verifyFeishuSignature(secret string, r *http.Request, body []byte) bool {
	timestamp := r.Header.Get("X-Lark-Signature-Timestamp")
	signature := r.Header.Get("X-Lark-Signature")

	if timestamp == "" || signature == "" {
		return true
	}

	stringToSign := timestamp + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	expectedSign := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSign))
}

// handleDingTalk handles DingTalk webhook requests.
func (h *Handler) handleDingTalk(r *http.Request, body []byte) (interface{}, int) {
	ch := h.manager.Get("dingtalk")
	if ch == nil {
		return map[string]interface{}{"error": "dingtalk channel not configured"}, http.StatusServiceUnavailable
	}

	dingtalkCh, ok := ch.(*channels.DingTalkChannel)
	if !ok {
		return map[string]interface{}{"error": "invalid channel type"}, http.StatusInternalServerError
	}

	msg, err := dingtalkCh.HandleWebhook(body)
	if err != nil {
		whLog.Error("dingtalk webhook error: %v", err)
		return map[string]interface{}{"error": err.Error()}, http.StatusBadRequest
	}

	// Route message to agent
	if h.messageRouter != nil {
		go func() {
			if err := h.messageRouter.RouteMessage(msg); err != nil {
				whLog.Error("failed to route dingtalk message: %v", err)
			}
		}()
	}

	return map[string]interface{}{"ok": true, "msgId": msg.MessageID}, http.StatusOK
}

// handleWeWork handles WeWork/WeChat webhook requests.
func (h *Handler) handleWeWork(r *http.Request, body []byte) (interface{}, int) {
	channelID := "wework"
	ch := h.manager.Get(channelID)
	if ch == nil {
		// Try weixin
		ch = h.manager.Get("weixin")
		channelID = "weixin"
	}
	if ch == nil {
		return map[string]interface{}{"error": "wework/weixin channel not configured"}, http.StatusServiceUnavailable
	}

	weworkCh, ok := ch.(*channels.WeWorkChannel)
	if !ok {
		return map[string]interface{}{"error": "invalid channel type"}, http.StatusInternalServerError
	}

	msg, err := weworkCh.HandleWebhook(body)
	if err != nil {
		whLog.Error("wework webhook error: %v", err)
		return map[string]interface{}{"error": err.Error()}, http.StatusBadRequest
	}

	// Route message to agent
	if h.messageRouter != nil {
		go func() {
			if err := h.messageRouter.RouteMessage(msg); err != nil {
				whLog.Error("failed to route wework message: %v", err)
			}
		}()
	}

	return map[string]interface{}{"ok": true, "msgId": msg.MessageID}, http.StatusOK
}

// handleTelegram handles Telegram webhook requests.
func (h *Handler) handleTelegram(r *http.Request, body []byte) (interface{}, int) {
	ch := h.manager.Get("telegram")
	if ch == nil {
		return map[string]interface{}{"error": "telegram channel not configured"}, http.StatusServiceUnavailable
	}

	telegramCh, ok := ch.(*channels.TelegramChannel)
	if !ok {
		return map[string]interface{}{"error": "invalid channel type"}, http.StatusInternalServerError
	}

	msg, err := telegramCh.HandleWebhook(body)
	if err != nil {
		whLog.Error("telegram webhook error: %v", err)
		return map[string]interface{}{"error": err.Error()}, http.StatusBadRequest
	}

	// Route message to agent
	if h.messageRouter != nil {
		go func() {
			if err := h.messageRouter.RouteMessage(msg); err != nil {
				whLog.Error("failed to route telegram message: %v", err)
			}
		}()
	}

	return map[string]interface{}{"ok": true}, http.StatusOK
}

// handleFeishu handles Feishu webhook requests.
func (h *Handler) handleFeishu(r *http.Request, body []byte) (interface{}, int) {
	ch := h.manager.Get("feishu")
	if ch == nil {
		return map[string]interface{}{"error": "feishu channel not configured"}, http.StatusServiceUnavailable
	}

	// Parse Feishu event
	var event struct {
		Type   string `json:"type"`
		Header struct {
			EventID    string `json:"event_id"`
			EventType  string `json:"event_type"`
			CreateTime string `json:"create_time"`
			Token      string `json:"token"`
			AppID      string `json:"app_id"`
			TenantKey  string `json:"tenant_key"`
		} `json:"header"`
		Event map[string]interface{} `json:"event"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		return map[string]interface{}{"error": "invalid json"}, http.StatusBadRequest
	}

	// Handle URL verification challenge
	if event.Type == "url_verification" {
		var challenge struct {
			Challenge string `json:"challenge"`
		}
		if err := json.Unmarshal(body, &challenge); err == nil {
			return map[string]interface{}{"challenge": challenge.Challenge}, http.StatusOK
		}
	}

	// Extract message content
	msg := &channels.IncomingMessage{
		ChannelID: "feishu",
		Timestamp: time.Now().Unix(),
		Raw:       event.Event,
	}

	// Try to extract common fields
	eventMap := event.Event
	if eventMap != nil {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			if senderID, ok := sender["sender_id"].(map[string]interface{}); ok {
				if uid, ok := senderID["union_id"].(string); ok {
					msg.From = uid
				}
			}
		}
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				msg.Text = content
			}
			if chatID, ok := message["chat_id"].(string); ok {
				msg.GroupID = chatID
			}
		}
	}

	// Route message to agent
	if h.messageRouter != nil {
		go func() {
			if err := h.messageRouter.RouteMessage(msg); err != nil {
				whLog.Error("failed to route feishu message: %v", err)
			}
		}()
	}

	return map[string]interface{}{"ok": true}, http.StatusOK
}

// handleSlack handles Slack webhook requests.
func (h *Handler) handleSlack(r *http.Request, body []byte) (interface{}, int) {
	// Handle URL verification challenge
	var challenge struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Token     string `json:"token"`
	}

	if err := json.Unmarshal(body, &challenge); err == nil && challenge.Type == "url_verification" {
		return map[string]interface{}{"challenge": challenge.Challenge}, http.StatusOK
	}

	// Parse Slack event
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return map[string]interface{}{"error": "invalid json"}, http.StatusBadRequest
	}

	msg := &channels.IncomingMessage{
		ChannelID: "slack",
		Timestamp: time.Now().Unix(),
		Raw:       event,
	}

	// Extract common fields
	if eventMap, ok := event["event"].(map[string]interface{}); ok {
		if user, ok := eventMap["user"].(string); ok {
			msg.From = user
		}
		if text, ok := eventMap["text"].(string); ok {
			msg.Text = text
		}
		if channel, ok := eventMap["channel"].(string); ok {
			msg.To = channel
		}
		if ts, ok := eventMap["ts"].(string); ok {
			msg.MessageID = ts
		}
		if channelType, ok := eventMap["channel_type"].(string); ok {
			switch channelType {
			case "channel", "group":
				msg.ChatType = "group"
			default:
				msg.ChatType = "dm"
			}
		}
	}

	// Route message to agent
	if h.messageRouter != nil {
		go func() {
			if err := h.messageRouter.RouteMessage(msg); err != nil {
				whLog.Error("failed to route slack message: %v", err)
			}
		}()
	}

	return map[string]interface{}{"ok": true}, http.StatusOK
}

// handleGeneric handles webhooks for unknown/custom channels.
func (h *Handler) handleGeneric(channelID string, r *http.Request, body []byte) (interface{}, int) {
	// Try to parse as JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return map[string]interface{}{"error": "invalid json"}, http.StatusBadRequest
	}

	msg := &channels.IncomingMessage{
		ChannelID: channelID,
		Timestamp: time.Now().Unix(),
		Raw:       payload,
	}

	// Try to extract common fields
	if from, ok := payload["from"].(string); ok {
		msg.From = from
	}
	if to, ok := payload["to"].(string); ok {
		msg.To = to
	}
	if text, ok := payload["text"].(string); ok {
		msg.Text = text
	}
	if content, ok := payload["content"].(string); ok {
		msg.Text = content
	}
	if chatType, ok := payload["chatType"].(string); ok {
		msg.ChatType = chatType
	}

	// Route message to agent
	if h.messageRouter != nil {
		go func() {
			if err := h.messageRouter.RouteMessage(msg); err != nil {
				whLog.Error("failed to route %s message: %v", channelID, err)
			}
		}()
	}

	return map[string]interface{}{"ok": true, "channel": channelID}, http.StatusOK
}

// WebhookInfo returns information about configured webhook endpoints.
func (h *Handler) WebhookInfo(baseURL string) map[string]interface{} {
	channels := h.manager.List()
	endpoints := make([]map[string]interface{}, 0, len(channels))

	for _, ch := range channels {
		info := map[string]interface{}{
			"channelId":   ch.ID(),
			"displayName": ch.DisplayName(),
			"webhookUrl":  fmt.Sprintf("%s/hooks/%s", baseURL, ch.ID()),
		}
		endpoints = append(endpoints, info)
	}

	return map[string]interface{}{
		"baseUrl":   baseURL,
		"endpoints": endpoints,
	}
}
