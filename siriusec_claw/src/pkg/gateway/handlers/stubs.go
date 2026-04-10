package handlers

import "github.com/siriusec/siriusec_claw/pkg/gateway/protocol"

// StubHandler returns "not implemented" for methods that are registered but not yet implemented.
func StubHandler(opts HandlerOpts) error {
	opts.Respond(false, nil, &protocol.ErrorShape{
		Code:    protocol.ErrCodeNotImplemented,
		Message: "method '" + opts.Req.Method + "' not yet implemented",
	}, nil)
	return nil
}

// Helper functions for creating error shapes

func errNotConfigured(what string) *protocol.ErrorShape {
	return &protocol.ErrorShape{
		Code:    protocol.ErrCodeInternal,
		Message: what + " not configured",
	}
}

func errInternal(msg string) *protocol.ErrorShape {
	return &protocol.ErrorShape{
		Code:    protocol.ErrCodeInternal,
		Message: msg,
	}
}

func errInvalidParams(msg string) *protocol.ErrorShape {
	return &protocol.ErrorShape{
		Code:    protocol.ErrCodeInvalidRequest,
		Message: msg,
	}
}
