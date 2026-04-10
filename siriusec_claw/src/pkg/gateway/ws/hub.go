// Package ws provides WebSocket connection management and req/res dispatch.
package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/siriusec/siriusec_claw/pkg/gateway/handlers"
	"github.com/siriusec/siriusec_claw/pkg/gateway/protocol"
	"github.com/siriusec/siriusec_claw/pkg/logging"
)

var wsLog = logging.Sub("ws")

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client represents a connected WebSocket client.
type Client struct {
	ConnID      string
	Conn        *websocket.Conn
	Send        chan []byte
	Hub         *Hub
	Handlers    *handlers.Registry
	Context     *handlers.Context
	Connect     *protocol.ConnectParams
	mu          sync.RWMutex
	queuedBytes int64
}

// Hub manages WebSocket clients.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	methods    []string
	events     []string
	version    string
	handlers   *handlers.Registry
	context    *handlers.Context
	mu         sync.RWMutex
	seq        int64
	startedAt  time.Time
}

const maxBufferedBytes = 4 << 20 // 4MB

// NewHub creates a new Hub.
func NewHub(version string) *Hub {
	methods := []string{
		"health", "status", "logs.tail",
		"channels.status", "channels.logout",
		"config.get", "config.env", "config.set", "config.apply", "config.patch", "config.schema",
		"mcp.servers.delete",
		"models.list",
		"agents.list", "agents.create", "agents.update", "agents.delete",
		"agents.files.list", "agents.files.get", "agents.files.set",
		"employees.list", "employees.get", "employees.create", "employees.delete",
		"skills.status", "skills.getDoc", "skills.bins", "skills.install", "skills.update", "skills.delete",
		"files.read",
		"sessions.list", "sessions.create", "sessions.ensure", "sessions.preview",
		"sessions.patch", "sessions.reset", "sessions.delete", "sessions.compact",
		"sessions.usage", "sessions.usage.timeseries", "sessions.usage.logs",
		"chat.history", "chat.send", "chat.abort", "chat.inject",
		"cron.list", "cron.status", "cron.add", "cron.remove", "cron.update", "cron.run", "cron.runs",
		"approvals.list", "approvals.approve", "approvals.deny", "approvals.whitelistSession",
		"exec.approvals.get", "exec.approvals.set",
		"exec.approval.request", "exec.approval.resolve",
		"node.pair.request", "node.pair.list", "node.pair.approve", "node.pair.reject",
		"node.list", "node.describe", "node.invoke",
		"system-presence", "system-event", "send", "agent",
		"agent.identity.get", "agent.wait",
		"usage.status", "usage.cost",
		"last-heartbeat", "set-heartbeats", "wake",
		"update.run",
	}
	events := []string{
		"connect.challenge", "agent", "chat", "presence", "tick",
		"talk.mode", "shutdown", "health", "heartbeat", "cron",
		"node.pair.requested", "node.pair.resolved",
		"exec.approval.requested", "exec.approval.resolved",
	}
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		methods:    methods,
		events:     events,
		version:    version,
		startedAt:  time.Now(),
	}
}

// SetContext updates the hub's context.
func (h *Hub) SetContext(ctx *handlers.Context) {
	h.context = ctx
}

// SetHandlers updates the hub's handlers registry.
func (h *Hub) SetHandlers(reg *handlers.Registry) {
	h.handlers = reg
}

// Run starts the hub loop.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastOptions configures broadcast behavior.
type BroadcastOptions struct {
	DropIfSlow   bool
	StateVersion *protocol.StateVersion
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(event string, payload interface{}, opts *BroadcastOptions) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	h.mu.Lock()
	h.seq++
	seq := h.seq
	h.mu.Unlock()

	if opts == nil {
		opts = &BroadcastOptions{DropIfSlow: true}
	}

	eventFrame := protocol.EventFrame{
		Type:    "event",
		Event:   event,
		Payload: payload,
		Seq:     &seq,
	}
	frameJSON, err := json.Marshal(eventFrame)
	if err != nil {
		wsLog.Error("failed to marshal event frame event=%s err=%v", event, err)
		return
	}

	for _, c := range clients {
		buffered := int(atomic.LoadInt64(&c.queuedBytes))
		if buffered > maxBufferedBytes {
			if opts.DropIfSlow {
				continue
			}
			c.Conn.Close()
			continue
		}
		c.enqueue(frameJSON, true)
	}
}

// BroadcastToConnIds sends an event to specific client connection IDs.
func (h *Hub) BroadcastToConnIds(event string, payload interface{}, connIds map[string]bool, opts *BroadcastOptions) {
	if len(connIds) == 0 {
		return
	}
	h.mu.RLock()
	clients := make([]*Client, 0)
	for c := range h.clients {
		if connIds[c.ConnID] {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()

	eventFrame := protocol.EventFrame{
		Type:    "event",
		Event:   event,
		Payload: payload,
	}
	frameJSON, err := json.Marshal(eventFrame)
	if err != nil {
		return
	}
	for _, c := range clients {
		c.enqueue(frameJSON, true)
	}
}

// ServeWS upgrades HTTP to WebSocket and handles the connection.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		wsLog.Warn("ws upgrade failed err=%v", err)
		return
	}
	connID := uuid.New().String()
	client := &Client{
		ConnID:   connID,
		Conn:     conn,
		Send:     make(chan []byte, 2048),
		Hub:      h,
		Handlers: h.handlers,
		Context:  h.context,
	}
	h.register <- client
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wsLog.Debug("ws read error connId=%s err=%v", c.ConnID, err)
			}
			break
		}
		c.handleMessage(data)
	}
}

func (c *Client) handleMessage(data []byte) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		c.sendError("", protocol.ErrCodeInvalidRequest, "invalid JSON")
		return
	}
	typ, _ := raw["type"].(string)
	if typ != "req" {
		c.sendError("", protocol.ErrCodeInvalidRequest, "expected req frame")
		return
	}
	id, _ := raw["id"].(string)
	method, _ := raw["method"].(string)
	params := raw["params"]

	// First message must be connect (handshake)
	c.mu.RLock()
	connected := c.Connect != nil
	c.mu.RUnlock()
	if !connected {
		if method != "connect" {
			c.sendError(id, protocol.ErrCodeInvalidRequest, "first request must be connect")
			return
		}
		c.handleConnect(id, params)
		return
	}

	// Dispatch to handler
	req := protocol.RequestFrame{Type: "req", ID: id, Method: method, Params: params}
	respond := func(ok bool, payload interface{}, err *protocol.ErrorShape, _ map[string]interface{}) {
		res := protocol.ResponseFrame{
			Type:    "res",
			ID:      id,
			OK:      ok,
			Payload: payload,
			Error:   err,
		}
		b, marshalErr := json.Marshal(res)
		if marshalErr != nil {
			return
		}
		select {
		case c.Send <- b:
			atomic.AddInt64(&c.queuedBytes, int64(len(b)))
		default:
			wsLog.Warn("response channel full connId=%s id=%s method=%s", c.ConnID, id, method)
		}
	}

	// Convert params to map[string]interface{}
	var paramsMap map[string]interface{}
	if params != nil {
		if m, ok := params.(map[string]interface{}); ok {
			paramsMap = m
		} else {
			// Try re-marshal and unmarshal
			b, _ := json.Marshal(params)
			_ = json.Unmarshal(b, &paramsMap)
		}
	}

	opts := handlers.HandlerOpts{
		Req:     req,
		Params:  paramsMap,
		Client:  &handlers.Client{Connect: *c.Connect, ConnID: c.ConnID},
		Respond: respond,
		Context: c.Context,
	}
	if err := (*c.Handlers).Dispatch(opts); err != nil {
		respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
	}
}

func (c *Client) handleConnect(id string, params interface{}) {
	b, err := json.Marshal(params)
	if err != nil {
		c.sendError(id, protocol.ErrCodeInvalidRequest, "invalid connect params")
		return
	}
	var cp protocol.ConnectParams
	if err := json.Unmarshal(b, &cp); err != nil {
		c.sendError(id, protocol.ErrCodeInvalidRequest, "invalid connect params")
		return
	}
	if cp.Client.ID == "" || cp.Client.Version == "" || cp.Client.Platform == "" || cp.Client.Mode == "" {
		c.sendError(id, protocol.ErrCodeInvalidRequest, "connect params missing required client fields")
		return
	}

	// Gateway token validation
	expectedToken := ""
	if c.Context != nil && c.Context.LoadConfigSnapshot != nil {
		if snap, err := c.Context.LoadConfigSnapshot(); err == nil && snap != nil {
			expectedToken = snap.GatewayToken
		}
	}
	if expectedToken != "" {
		gotToken := ""
		if cp.Auth != nil {
			gotToken = strings.TrimSpace(cp.Auth.Token)
			if gotToken == "" {
				gotToken = strings.TrimSpace(cp.Auth.Password)
			}
		}
		if gotToken == "" || gotToken != expectedToken {
			c.sendError(id, "invalid_gateway_token", "Authentication failed: invalid or missing gateway token")
			return
		}
	}

	minP, maxP := cp.MinProtocol, cp.MaxProtocol
	if maxP < protocol.PROTOCOL_VERSION || minP > protocol.PROTOCOL_VERSION {
		c.sendError(id, protocol.ErrCodeInvalidRequest, "protocol mismatch")
		return
	}

	role := cp.Role
	if role == "" {
		role = "operator"
	}
	if role != "operator" && role != "node" {
		c.sendError(id, protocol.ErrCodeInvalidRequest, "invalid role: "+role)
		return
	}
	cp.Role = role
	if cp.Scopes == nil {
		cp.Scopes = []string{}
	}

	c.mu.Lock()
	c.Connect = &cp
	c.mu.Unlock()

	hello := protocol.HelloOk{
		Type:     "hello-ok",
		Protocol: protocol.PROTOCOL_VERSION,
		Server: protocol.HelloServer{
			Version: c.Hub.version,
			ConnID:  c.ConnID,
		},
		Features: protocol.HelloFeatures{
			Methods: c.Hub.methods,
			Events:  c.Hub.events,
		},
		Snapshot: protocol.Snapshot{
			Presence:     []protocol.PresenceEntry{},
			Health:       map[string]interface{}{},
			StateVersion: protocol.StateVersion{Presence: 0, Health: 0},
			UptimeMs:     time.Since(c.Hub.startedAt).Milliseconds(),
		},
		Policy: protocol.HelloPolicy{
			MaxPayload:       1 << 20,
			MaxBufferedBytes: 4 << 20,
			TickIntervalMs:   5000,
		},
	}
	res := protocol.ResponseFrame{
		Type:    "res",
		ID:      id,
		OK:      true,
		Payload: hello,
	}
	msg, err := json.Marshal(res)
	if err != nil {
		return
	}
	select {
	case c.Send <- msg:
		atomic.AddInt64(&c.queuedBytes, int64(len(msg)))
	default:
	}
}

func (c *Client) sendError(id string, code string, msg string) {
	res := protocol.ResponseFrame{
		Type:  "res",
		ID:    id,
		OK:    false,
		Error: &protocol.ErrorShape{Code: code, Message: msg},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return
	}
	select {
	case c.Send <- b:
		atomic.AddInt64(&c.queuedBytes, int64(len(b)))
	default:
	}
}

func (c *Client) enqueue(msg []byte, dropOldest bool) bool {
	for {
		select {
		case c.Send <- msg:
			atomic.AddInt64(&c.queuedBytes, int64(len(msg)))
			return true
		default:
			if !dropOldest {
				return false
			}
			select {
			case old := <-c.Send:
				atomic.AddInt64(&c.queuedBytes, -int64(len(old)))
			default:
				return false
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if ok {
				atomic.AddInt64(&c.queuedBytes, -int64(len(msg)))
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
