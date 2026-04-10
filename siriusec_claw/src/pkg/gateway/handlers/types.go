// Package handlers provides the Gateway WebSocket RPC handler registry.
package handlers

import (
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/gateway/protocol"
)

// Handler is the signature for a Gateway RPC method handler.
type Handler func(HandlerOpts) error

// RespondFn sends a response back to the client.
type RespondFn func(ok bool, payload interface{}, err *protocol.ErrorShape, meta map[string]interface{})

// HandlerOpts is passed to every handler invocation.
type HandlerOpts struct {
	Req     protocol.RequestFrame
	Params  map[string]interface{}
	Client  *Client
	Respond RespondFn
	Context *Context
}

// Client identifies the connected WebSocket client.
type Client struct {
	Connect protocol.ConnectParams
	ConnID  string
}

// BroadcastOptions configures broadcast behavior.
type BroadcastOptions struct {
	DropIfSlow   bool
	StateVersion *protocol.StateVersion
}

// ConfigSnapshot represents a point-in-time config state.
type ConfigSnapshot struct {
	Path         string             `json:"path"`
	Exists       bool               `json:"exists"`
	Raw          string             `json:"raw,omitempty"`
	Parsed       *config.ClawConfig `json:"parsed,omitempty"`
	Hash         string             `json:"hash,omitempty"`
	GatewayToken string             `json:"-"`
}

// Context is the shared dependency injection container for all handlers.
type Context struct {
	Version            string
	Config             *config.ClawConfig
	LoadConfigSnapshot func() (*ConfigSnapshot, error)
	Broadcast          func(event string, payload interface{}, opts *BroadcastOptions)
	BroadcastToConnIds func(event string, payload interface{}, connIds map[string]bool, opts *BroadcastOptions)
	InvokeMethod       func(method string, params map[string]interface{}) (ok bool, payload interface{}, err *protocol.ErrorShape)
	AgentRunSeq        map[string]int64

	// Hooks
	HooksWake  func(text string, mode string)
	HooksAgent func(p HooksAgentParams) string
}

// HooksAgentParams holds parameters for the hooks.agent callback.
type HooksAgentParams struct {
	Message        string
	SessionKey     string
	Channel        string
	To             string
	ChatType       string
	MessageID      string
	Thinking       string
	TimeoutSeconds *int
}

// Registry maps method names to handlers.
type Registry map[string]Handler

// Dispatch finds and executes the handler for the given request method.
func (r Registry) Dispatch(opts HandlerOpts) error {
	method := opts.Req.Method
	handler, ok := r[method]
	if !ok {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeNotFound,
			Message: "method not found: " + method,
		}, nil)
		return nil
	}
	return handler(opts)
}
