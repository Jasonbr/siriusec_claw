// Package http provides the Gateway HTTP server.
package http

import (
	"net/http"
	"strings"

	"github.com/siriusec/siriusec_claw/pkg/gateway/handlers"
)

// requireGatewayToken wraps an HTTP handler with gateway token authentication.
func (s *Server) requireGatewayToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedToken := handlers.GetExpectedGatewayToken(s.ctx.LoadConfigSnapshot)
		if expectedToken == "" {
			next(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "" {
			token := strings.TrimPrefix(auth, "Bearer ")
			token = strings.TrimSpace(token)
			if token == expectedToken {
				next(w, r)
				return
			}
		}

		// Check query parameter
		if token := r.URL.Query().Get("token"); token == expectedToken {
			next(w, r)
			return
		}

		// Check X-Gateway-Token header
		if token := r.Header.Get("X-Gateway-Token"); token == expectedToken {
			next(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"ok":false,"error":"unauthorized: invalid or missing gateway token"}`))
	}
}
