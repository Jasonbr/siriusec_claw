// Package protocol defines the Gateway WebSocket frame formats and handshake.
package protocol

// PROTOCOL_VERSION is the current Gateway protocol version.
const PROTOCOL_VERSION = 3

// Error codes
const (
	ErrCodeInvalidRequest = "invalid_request"
	ErrCodeInternal       = "internal_error"
	ErrCodeNotFound       = "not_found"
	ErrCodeNotImplemented = "not_implemented"
	ErrCodeUnauthorized   = "unauthorized"
	ErrCodeForbidden      = "forbidden"
	ErrCodeTimeout        = "timeout"
)
