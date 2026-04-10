package webhooks

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/channels"
	"github.com/siriusec/siriusec_claw/pkg/logging"
	"github.com/siriusec/siriusec_claw/pkg/memory"
)

var routerLog = logging.Sub("router")

// MessageRouterImpl implements MessageRouter and routes messages to agents.
type MessageRouterImpl struct {
	manager     *channels.Manager
	agentRunner AgentRunner
	memoryStore *memory.Store
	sessions    map[string]*ChatSession
	mu          sync.RWMutex
}

// AgentRunner defines the interface for running agents.
type AgentRunner interface {
	Run(sessionKey, agentID, message string) (string, error)
}

// ChatSession represents an active chat session.
type ChatSession struct {
	SessionKey string
	ChannelID  string
	AgentID    string
	LastActive time.Time
}

// NewMessageRouter creates a new message router.
func NewMessageRouter(manager *channels.Manager, agentRunner AgentRunner, memStore *memory.Store) *MessageRouterImpl {
	return &MessageRouterImpl{
		manager:     manager,
		agentRunner: agentRunner,
		memoryStore: memStore,
		sessions:    make(map[string]*ChatSession),
	}
}

// RouteMessage routes an incoming message to the appropriate agent.
func (r *MessageRouterImpl) RouteMessage(msg *channels.IncomingMessage) error {
	routerLog.Info("routing message: channel=%s from=%s chatType=%s",
		msg.ChannelID, msg.From, msg.ChatType)

	// Build session key
	sessionKey := r.buildSessionKey(msg)

	// Get or create session
	session := r.getOrCreateSession(sessionKey, msg)

	// Store incoming message in memory for context
	if r.memoryStore != nil {
		entry := &memory.Entry{
			Title:    fmt.Sprintf("Incoming from %s", msg.FromName),
			Content:  msg.Text,
			Source:   msg.ChannelID,
			Category: "chat",
			Tags:     []string{"incoming", msg.ChannelID, msg.ChatType},
		}
		r.memoryStore.Add(entry)
	}

	// Check for commands
	if strings.HasPrefix(strings.TrimSpace(msg.Text), "/") {
		return r.handleCommand(msg, session)
	}

	// Run agent to generate response
	if r.agentRunner != nil {
		go r.runAgentAndRespond(msg, session)
	} else {
		routerLog.Warn("no agent runner configured, cannot process message")
	}

	return nil
}

// buildSessionKey creates a unique session key for the conversation.
func (r *MessageRouterImpl) buildSessionKey(msg *channels.IncomingMessage) string {
	channelPrefix := msg.ChannelID
	if len(channelPrefix) > 3 {
		channelPrefix = channelPrefix[:3]
	}

	switch msg.ChatType {
	case "group":
		return fmt.Sprintf("%s:%s:main", channelPrefix, msg.GroupID)
	default:
		return fmt.Sprintf("%s:%s:main", channelPrefix, msg.From)
	}
}

// getOrCreateSession gets an existing session or creates a new one.
func (r *MessageRouterImpl) getOrCreateSession(sessionKey string, msg *channels.IncomingMessage) *ChatSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionKey]
	if !exists {
		session = &ChatSession{
			SessionKey: sessionKey,
			ChannelID:  msg.ChannelID,
			AgentID:    "main", // Default agent
		}
		r.sessions[sessionKey] = session
		routerLog.Info("created new session: %s", sessionKey)
	}

	session.LastActive = time.Now()
	return session
}

// handleCommand handles slash commands.
func (r *MessageRouterImpl) handleCommand(msg *channels.IncomingMessage, session *ChatSession) error {
	text := strings.TrimSpace(msg.Text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/help":
		r.sendReply(msg, "Available commands:\n/help - Show this help\n/agent <name> - Switch agent\n/status - Show session status\n/clear - Clear conversation history")
	case "/agent":
		if len(args) > 0 {
			session.AgentID = args[0]
			r.sendReply(msg, fmt.Sprintf("Switched to agent: %s", args[0]))
		} else {
			r.sendReply(msg, fmt.Sprintf("Current agent: %s", session.AgentID))
		}
	case "/status":
		r.sendReply(msg, fmt.Sprintf("Session: %s\nChannel: %s\nAgent: %s\nLast active: %s",
			session.SessionKey, session.ChannelID, session.AgentID, session.LastActive.Format(time.RFC3339)))
	case "/clear":
		r.mu.Lock()
		delete(r.sessions, session.SessionKey)
		r.mu.Unlock()
		r.sendReply(msg, "Session cleared. Starting fresh conversation.")
	default:
		// Unknown command, pass to agent
		if r.agentRunner != nil {
			go r.runAgentAndRespond(msg, session)
		}
	}

	return nil
}

// runAgentAndRespond runs the agent and sends the response.
func (r *MessageRouterImpl) runAgentAndRespond(msg *channels.IncomingMessage, session *ChatSession) {
	if r.agentRunner == nil {
		return
	}

	routerLog.Debug("running agent %s for session %s", session.AgentID, session.SessionKey)

	// Run the agent
	response, err := r.agentRunner.Run(session.SessionKey, session.AgentID, msg.Text)
	if err != nil {
		routerLog.Error("agent run failed: %v", err)
		r.sendReply(msg, fmt.Sprintf("Error: %v", err))
		return
	}

	// Send response
	if response != "" {
		r.sendReply(msg, response)
	}
}

// sendReply sends a reply to the message.
func (r *MessageRouterImpl) sendReply(msg *channels.IncomingMessage, text string) {
	// Determine target
	target := msg.From
	chatType := msg.ChatType

	if msg.ChatType == "group" && msg.GroupID != "" {
		target = msg.GroupID
	}

	// Build send options
	opts := &channels.SendOptions{
		RootMessageID: msg.MessageID,
		Format:        "markdown",
	}

	// Send via channel manager
	if r.manager != nil {
		err := r.manager.DeliverMessage(msg.ChannelID, target, chatType, text, opts)
		if err != nil {
			routerLog.Error("failed to send reply: %v", err)
		}
	}
}

// GetSession returns a session by key.
func (r *MessageRouterImpl) GetSession(sessionKey string) *ChatSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessions[sessionKey]
}

// ListSessions returns all active sessions.
func (r *MessageRouterImpl) ListSessions() []*ChatSession {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*ChatSession, 0, len(r.sessions))
	for _, s := range r.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// CleanupStaleSessions removes sessions inactive for more than the given duration.
func (r *MessageRouterImpl) CleanupStaleSessions(maxAge time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0

	for key, session := range r.sessions {
		if session.LastActive.Before(cutoff) {
			delete(r.sessions, key)
			count++
		}
	}

	if count > 0 {
		routerLog.Info("cleaned up %d stale sessions", count)
	}

	return count
}
