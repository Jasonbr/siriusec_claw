package security

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/logging"
)

var secLog = logging.Sub("security")

// CommandPolicy defines rules for command execution.
type CommandPolicy struct {
	Enabled       bool     `json:"enabled"`
	DefaultPolicy string   `json:"defaultPolicy"` // "deny", "ask", "allow"
	Deny          []string `json:"deny,omitempty"`
	Ask           []string `json:"ask,omitempty"`
	Allow         []string `json:"allow,omitempty"`
	BanArguments  []string `json:"banArguments,omitempty"`
	MaxLength     int      `json:"maxLength,omitempty"`
	SecretPatterns []string `json:"secretPatterns,omitempty"`
}

// EvaluationResult is the outcome of a command policy evaluation.
type EvaluationResult struct {
	Decision string // "allow", "deny", "ask"
	Reason   string
	Rule     string // which rule triggered the decision
}

// EvaluateCommand checks a command against the policy.
func (p *CommandPolicy) EvaluateCommand(command string) *EvaluationResult {
	if !p.Enabled {
		return &EvaluationResult{Decision: "allow", Reason: "policy disabled"}
	}

	cmd := strings.TrimSpace(command)
	cmdLower := strings.ToLower(cmd)
	base := extractCommandBase(cmdLower)

	// Check max length
	if p.MaxLength > 0 && len(cmd) > p.MaxLength {
		return &EvaluationResult{Decision: "deny", Reason: "command exceeds max length", Rule: "maxLength"}
	}

	// Check banned arguments
	for _, arg := range p.BanArguments {
		if strings.Contains(cmdLower, strings.ToLower(arg)) {
			return &EvaluationResult{Decision: "deny", Reason: "banned argument: " + arg, Rule: "banArguments"}
		}
	}

	// Check secret patterns
	for _, pat := range p.SecretPatterns {
		if re, err := regexp.Compile(pat); err == nil && re.MatchString(cmd) {
			return &EvaluationResult{Decision: "deny", Reason: "contains secret pattern", Rule: "secretPatterns"}
		}
	}

	// Check deny rules
	for _, rule := range p.Deny {
		ruleLower := strings.ToLower(rule)
		if base == ruleLower || strings.Contains(cmdLower, ruleLower) {
			return &EvaluationResult{Decision: "deny", Reason: "denied by rule", Rule: rule}
		}
	}

	// Check allow rules
	for _, rule := range p.Allow {
		ruleLower := strings.ToLower(rule)
		if base == ruleLower || strings.HasPrefix(cmdLower, ruleLower) {
			return &EvaluationResult{Decision: "allow", Reason: "allowed by rule", Rule: rule}
		}
	}

	// Check ask rules
	for _, rule := range p.Ask {
		ruleLower := strings.ToLower(rule)
		if base == ruleLower || strings.Contains(cmdLower, ruleLower) {
			return &EvaluationResult{Decision: "ask", Reason: "requires approval", Rule: rule}
		}
	}

	// Default policy
	return &EvaluationResult{Decision: p.DefaultPolicy, Reason: "default policy"}
}

// ApprovalRequest represents a pending command approval.
type ApprovalRequest struct {
	ID         string `json:"id"`
	SessionKey string `json:"sessionKey"`
	Command    string `json:"command"`
	Tool       string `json:"tool"`
	RequestedAt int64 `json:"requestedAt"`
	Status     string `json:"status"` // "pending", "approved", "denied"
	ResolvedAt *int64 `json:"resolvedAt,omitempty"`
	ResolvedBy string `json:"resolvedBy,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// ApprovalQueue manages pending approval requests.
type ApprovalQueue struct {
	mu         sync.RWMutex
	requests   map[string]*ApprovalRequest
	whitelist  map[string]bool // sessionKey → auto-approve
	timeoutMs  int64
}

// NewApprovalQueue creates a new approval queue.
func NewApprovalQueue(timeoutMs int64) *ApprovalQueue {
	if timeoutMs <= 0 {
		timeoutMs = 120000 // 2 minutes default
	}
	return &ApprovalQueue{
		requests:  make(map[string]*ApprovalRequest),
		whitelist: make(map[string]bool),
		timeoutMs: timeoutMs,
	}
}

// Submit adds a new approval request.
func (q *ApprovalQueue) Submit(sessionKey, command, tool string) *ApprovalRequest {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Auto-approve if session is whitelisted
	if q.whitelist[sessionKey] {
		now := time.Now().UnixMilli()
		return &ApprovalRequest{
			ID:         uuid.New().String(),
			SessionKey: sessionKey,
			Command:    command,
			Tool:       tool,
			RequestedAt: now,
			Status:     "approved",
			ResolvedAt: &now,
			ResolvedBy: "whitelist",
		}
	}

	req := &ApprovalRequest{
		ID:          uuid.New().String(),
		SessionKey:  sessionKey,
		Command:     command,
		Tool:        tool,
		RequestedAt: time.Now().UnixMilli(),
		Status:      "pending",
	}
	q.requests[req.ID] = req
	secLog.Info("approval requested id=%s session=%s command=%s", req.ID, sessionKey, truncate(command, 80))
	return req
}

// Approve resolves an approval request as approved.
func (q *ApprovalQueue) Approve(requestID, resolvedBy string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	req, ok := q.requests[requestID]
	if !ok || req.Status != "pending" {
		return false
	}
	now := time.Now().UnixMilli()
	req.Status = "approved"
	req.ResolvedAt = &now
	req.ResolvedBy = resolvedBy
	return true
}

// Deny resolves an approval request as denied.
func (q *ApprovalQueue) Deny(requestID, resolvedBy, reason string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	req, ok := q.requests[requestID]
	if !ok || req.Status != "pending" {
		return false
	}
	now := time.Now().UnixMilli()
	req.Status = "denied"
	req.ResolvedAt = &now
	req.ResolvedBy = resolvedBy
	req.Reason = reason
	return true
}

// WhitelistSession auto-approves all future requests for a session.
func (q *ApprovalQueue) WhitelistSession(sessionKey string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.whitelist[sessionKey] = true
	secLog.Info("session whitelisted: %s", sessionKey)
}

// List returns all pending requests.
func (q *ApprovalQueue) List() []*ApprovalRequest {
	q.mu.RLock()
	defer q.mu.RUnlock()
	var result []*ApprovalRequest
	now := time.Now().UnixMilli()
	for _, req := range q.requests {
		if req.Status == "pending" {
			// Auto-expire timed-out requests
			if now-req.RequestedAt > q.timeoutMs {
				req.Status = "denied"
				req.Reason = "timeout"
				continue
			}
			result = append(result, req)
		}
	}
	return result
}

// Get returns a specific request.
func (q *ApprovalQueue) Get(requestID string) *ApprovalRequest {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.requests[requestID]
}

// Cleanup removes old resolved requests.
func (q *ApprovalQueue) Cleanup(maxAgeMs int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now().UnixMilli()
	for id, req := range q.requests {
		if req.Status != "pending" && now-req.RequestedAt > maxAgeMs {
			delete(q.requests, id)
		}
	}
}

// extractCommandBase extracts the base command from a full command string.
func extractCommandBase(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	// Strip leading env vars like "FOO=bar cmd"
	for strings.Contains(cmd, "=") {
		parts := strings.SplitN(cmd, " ", 2)
		if strings.Contains(parts[0], "=") && len(parts) > 1 {
			cmd = strings.TrimSpace(parts[1])
		} else {
			break
		}
	}
	// Strip sudo prefix
	if strings.HasPrefix(cmd, "sudo ") {
		cmd = strings.TrimSpace(cmd[5:])
	}
	// Get first word
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		// Strip path
		base := parts[0]
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		return base
	}
	return cmd
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
