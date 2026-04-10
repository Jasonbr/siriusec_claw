package channels

import (
	"fmt"
	"sync"

	"github.com/siriusec/siriusec_claw/pkg/logging"
)

var chLog = logging.Sub("channels")

// Channel is the interface that all messaging channel adapters must implement.
type Channel interface {
	// ID returns the channel identifier (e.g. "feishu", "dingtalk", "telegram").
	ID() string

	// DisplayName returns a human-readable name.
	DisplayName() string

	// Start initializes the channel (e.g. starts webhook server, connects to API).
	Start() error

	// Stop gracefully shuts down the channel.
	Stop() error

	// Status returns the current channel status.
	Status() *ChannelStatus

	// SendMessage delivers a message to the specified target.
	SendMessage(target string, chatType string, text string, opts *SendOptions) error
}

// ChannelStatus represents the current state of a channel.
type ChannelStatus struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Connected   bool   `json:"connected"`
	Enabled     bool   `json:"enabled"`
	Error       string `json:"error,omitempty"`
	AccountID   string `json:"accountId,omitempty"`
	AccountName string `json:"accountName,omitempty"`
}

// SendOptions configures message delivery.
type SendOptions struct {
	RootMessageID string                 // for threaded replies
	Header        string                 // card header
	Format        string                 // "text", "markdown", "card"
	Extra         map[string]interface{} // channel-specific options
}

// IncomingMessage represents a message received from a channel.
type IncomingMessage struct {
	ChannelID string                 `json:"channelId"`
	From      string                 `json:"from"`
	FromName  string                 `json:"fromName,omitempty"`
	To        string                 `json:"to"`
	ChatType  string                 `json:"chatType"` // "dm", "group", "channel"
	Text      string                 `json:"text"`
	MessageID string                 `json:"messageId,omitempty"`
	GroupID   string                 `json:"groupId,omitempty"`
	GroupName string                 `json:"groupName,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Raw       map[string]interface{} `json:"raw,omitempty"`
}

// MessageHandler is called when a new message arrives from a channel.
type MessageHandler func(msg *IncomingMessage) error

// Manager manages all registered channels.
type Manager struct {
	mu             sync.RWMutex
	channels       map[string]Channel
	messageHandler MessageHandler
}

// NewManager creates a new channel manager.
func NewManager() *Manager {
	return &Manager{
		channels: make(map[string]Channel),
	}
}

// Register adds a channel to the manager.
func (m *Manager) Register(ch Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.ID()] = ch
	chLog.Info("channel registered: %s (%s)", ch.ID(), ch.DisplayName())
}

// SetMessageHandler sets the callback for incoming messages.
func (m *Manager) SetMessageHandler(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageHandler = handler
}

// Get returns a channel by ID.
func (m *Manager) Get(id string) Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channels[id]
}

// List returns all registered channels.
func (m *Manager) List() []Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		result = append(result, ch)
	}
	return result
}

// StatusAll returns status of all channels.
func (m *Manager) StatusAll() []*ChannelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make([]*ChannelStatus, 0, len(m.channels))
	for _, ch := range m.channels {
		statuses = append(statuses, ch.Status())
	}
	return statuses
}

// StartAll starts all enabled channels.
func (m *Manager) StartAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.channels {
		if err := ch.Start(); err != nil {
			chLog.Error("failed to start channel %s: %v", ch.ID(), err)
		}
	}
}

// StopAll gracefully stops all channels.
func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.channels {
		if err := ch.Stop(); err != nil {
			chLog.Error("failed to stop channel %s: %v", ch.ID(), err)
		}
	}
}

// DeliverMessage sends a message via the specified channel.
func (m *Manager) DeliverMessage(channelID, target, chatType, text string, opts *SendOptions) error {
	ch := m.Get(channelID)
	if ch == nil {
		return fmt.Errorf("channel not found: %s", channelID)
	}
	return ch.SendMessage(target, chatType, text, opts)
}

// HandleIncoming dispatches an incoming message to the registered handler.
func (m *Manager) HandleIncoming(msg *IncomingMessage) error {
	m.mu.RLock()
	handler := m.messageHandler
	m.mu.RUnlock()
	if handler == nil {
		return fmt.Errorf("no message handler registered")
	}
	return handler(msg)
}
