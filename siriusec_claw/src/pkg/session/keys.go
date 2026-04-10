package session

import (
	"regexp"
	"strings"
)

var safeSessionIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// ValidateSessionID validates and returns the session ID.
func ValidateSessionID(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", nil
	}
	sessionID = strings.ToLower(sessionID)
	if safeSessionIDRe.MatchString(sessionID) {
		return sessionID, nil
	}
	return SanitizeForSessionID(sessionID), nil
}

// SanitizeForSessionID converts an arbitrary string to a valid session ID.
func SanitizeForSessionID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' {
			return r
		}
		return '-'
	}, s)
	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")
	if len(s) > 128 {
		s = s[:128]
	}
	if s == "" {
		s = "session"
	}
	return s
}

// SessionIDFromSessionKey derives a sessionID from a session key.
func SessionIDFromSessionKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "main"
	}
	// Strip "agent:{agentId}:" prefix
	if strings.HasPrefix(key, "agent:") {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) >= 3 {
			key = parts[2]
		}
	}
	return SanitizeForSessionID(key)
}

// ResolveSessionAgentID extracts the agent ID from a session key.
func ResolveSessionAgentID(sessionKey string) string {
	if strings.HasPrefix(sessionKey, "agent:") {
		parts := strings.SplitN(sessionKey, ":", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return "main"
}

// IsMainSession returns true if the key represents the main/default session.
func IsMainSession(key string) bool {
	return key == "main" || key == "agent:main:main"
}

// IsCronRunSession returns true if the key is a cron run session (not persistent cron).
func IsCronRunSession(key string) bool {
	return strings.Contains(key, ":cron:") && strings.Contains(key, ":run:")
}
