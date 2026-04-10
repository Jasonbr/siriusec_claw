package handlers

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/session"
)

// --- sessions.list ---

func SessionsListHandler(opts HandlerOpts) error {
	env := envGetter()
	agentID := stringParam(opts.Params, "agentId", "main")

	store, err := session.LoadSessionStore(
		session.ResolveDefaultSessionStorePath(agentID, env),
	)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	includeGlobal := boolParam(opts.Params, "includeGlobal", false)
	includeUnknown := boolParam(opts.Params, "includeUnknown", false)
	limit := intParam(opts.Params, "limit", 100)
	search := stringParam(opts.Params, "search", "")

	type sessionRow struct {
		Key           string  `json:"key"`
		Kind          string  `json:"kind"`
		Label         *string `json:"label,omitempty"`
		DisplayName   *string `json:"displayName,omitempty"`
		Channel       *string `json:"channel,omitempty"`
		ChatType      *string `json:"chatType,omitempty"`
		UpdatedAt     *int64  `json:"updatedAt,omitempty"`
		SessionID     *string `json:"sessionId,omitempty"`
		Model         *string `json:"model,omitempty"`
		ModelProvider *string `json:"modelProvider,omitempty"`
	}

	var rows []sessionRow
	for key, entry := range store {
		// Filter cron run sessions
		if session.IsCronRunSession(key) {
			continue
		}
		kind := classifySessionKind(key)
		if kind == "global" && !includeGlobal {
			continue
		}
		if kind == "unknown" && !includeUnknown {
			continue
		}
		// Search filter
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(key), searchLower) &&
				!strings.Contains(strings.ToLower(entry.Label), searchLower) {
				continue
			}
		}

		row := sessionRow{
			Key:  key,
			Kind: kind,
		}
		if entry.Label != "" {
			l := entry.Label
			row.Label = &l
		}
		if entry.Channel != "" {
			c := entry.Channel
			row.Channel = &c
		}
		if entry.ChatType != "" {
			ct := entry.ChatType
			row.ChatType = &ct
		}
		if entry.UpdatedAt > 0 {
			u := entry.UpdatedAt
			row.UpdatedAt = &u
		}
		if entry.SessionID != "" {
			s := entry.SessionID
			row.SessionID = &s
		}
		if entry.Model != "" {
			m := entry.Model
			row.Model = &m
		}
		if entry.ModelProvider != "" {
			mp := entry.ModelProvider
			row.ModelProvider = &mp
		}

		rows = append(rows, row)
	}

	// Sort by updatedAt desc
	sort.Slice(rows, func(i, j int) bool {
		a, b := int64(0), int64(0)
		if rows[i].UpdatedAt != nil {
			a = *rows[i].UpdatedAt
		}
		if rows[j].UpdatedAt != nil {
			b = *rows[j].UpdatedAt
		}
		return a > b
	})

	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}

	storePath := session.ResolveDefaultSessionStorePath(agentID, env)
	opts.Respond(true, map[string]interface{}{
		"ts":       time.Now().UnixMilli(),
		"path":     storePath,
		"count":    len(rows),
		"sessions": rows,
	}, nil, nil)
	return nil
}

// --- sessions.create ---

func SessionsCreateHandler(opts HandlerOpts) error {
	env := envGetter()
	agentID := "main"

	label := stringParam(opts.Params, "label", "")
	if label == "" {
		label = fmt.Sprintf("Custom session %d", time.Now().UnixMilli()%10000)
	}

	sessionID := uuid.New().String()
	key := "custom:" + sessionID

	entry := session.SessionEntry{
		SessionID:   sessionID,
		UpdatedAt:   time.Now().UnixMilli(),
		SessionFile: sessionID + ".jsonl",
		Label:       label,
		Channel:     "webchat",
	}

	if err := session.CreateSession(agentID, key, env, entry); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"key":       key,
		"sessionId": sessionID,
		"path":      session.ResolveDefaultSessionStorePath(agentID, env),
		"entry":     entry,
	}, nil, nil)
	return nil
}

// --- sessions.ensure ---

func SessionsEnsureHandler(opts HandlerOpts) error {
	env := envGetter()
	key := stringParam(opts.Params, "key", "")
	if key == "" {
		opts.Respond(false, nil, errInvalidParams("key required"), nil)
		return nil
	}

	agentID := session.ResolveSessionAgentID(key)
	label := stringParam(opts.Params, "label", "")

	entry, created, err := session.EnsureSession(agentID, key, env, label)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"key":       key,
		"created":   created,
		"sessionId": entry.SessionID,
		"entry":     entry,
	}, nil, nil)
	return nil
}

// --- sessions.preview ---

func SessionsPreviewHandler(opts HandlerOpts) error {
	env := envGetter()
	keys := stringSliceParam(opts.Params, "keys")
	if len(keys) == 0 {
		opts.Respond(false, nil, errInvalidParams("keys required"), nil)
		return nil
	}
	if len(keys) > 64 {
		keys = keys[:64]
	}

	limit := intParam(opts.Params, "limit", 12)
	maxChars := intParam(opts.Params, "maxChars", 240)
	if limit > 50 {
		limit = 50
	}
	if maxChars > 2000 {
		maxChars = 2000
	}

	type previewResult struct {
		Key    string                `json:"key"`
		Status string                `json:"status"`
		Items  []session.PreviewItem `json:"items,omitempty"`
	}

	previews := make([]previewResult, 0, len(keys))
	for _, key := range keys {
		agentID := session.ResolveSessionAgentID(key)
		storePath := session.ResolveDefaultSessionStorePath(agentID, env)
		store, _ := session.LoadSessionStore(storePath)

		entry, ok := store[key]
		if !ok {
			previews = append(previews, previewResult{Key: key, Status: "missing"})
			continue
		}

		transcriptPath := session.ResolveSessionFilePath(entry.SessionID, &agentID, env)
		items, err := session.ReadSessionPreviewItems(transcriptPath, limit, maxChars)
		if err != nil {
			previews = append(previews, previewResult{Key: key, Status: "error"})
			continue
		}

		status := "ok"
		if len(items) == 0 {
			status = "empty"
		}
		previews = append(previews, previewResult{Key: key, Status: status, Items: items})
	}

	opts.Respond(true, map[string]interface{}{
		"ts":       time.Now().UnixMilli(),
		"previews": previews,
	}, nil, nil)
	return nil
}

// --- sessions.patch ---

func SessionsPatchHandler(opts HandlerOpts) error {
	env := envGetter()
	key := stringParam(opts.Params, "key", "")
	if key == "" {
		opts.Respond(false, nil, errInvalidParams("key required"), nil)
		return nil
	}

	agentID := session.ResolveSessionAgentID(key)
	storePath := session.ResolveDefaultSessionStorePath(agentID, env)
	store, err := session.LoadSessionStore(storePath)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	entry, ok := store[key]
	if !ok {
		opts.Respond(false, nil, errInvalidParams("session not found: "+key), nil)
		return nil
	}

	if v, ok := opts.Params["label"]; ok {
		if s, ok := v.(string); ok {
			entry.Label = s
		}
	}
	if v, ok := opts.Params["thinkingLevel"]; ok {
		if s, ok := v.(string); ok {
			entry.ThinkingLevel = s
		}
	}
	if v, ok := opts.Params["verboseLevel"]; ok {
		if s, ok := v.(string); ok {
			entry.VerboseLevel = s
		}
	}
	if v, ok := opts.Params["model"]; ok {
		if s, ok := v.(string); ok {
			entry.Model = s
		}
	}
	if v, ok := opts.Params["modelProvider"]; ok {
		if s, ok := v.(string); ok {
			entry.ModelProvider = s
		}
	}
	if v, ok := opts.Params["sendPolicy"]; ok {
		if s, ok := v.(string); ok {
			entry.SendPolicy = s
		}
	}

	entry.UpdatedAt = time.Now().UnixMilli()
	store[key] = entry

	if err := session.SaveSessionStore(storePath, store); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":    true,
		"path":  storePath,
		"key":   key,
		"entry": entry,
	}, nil, nil)
	return nil
}

// --- sessions.reset ---

func SessionsResetHandler(opts HandlerOpts) error {
	env := envGetter()
	key := stringParam(opts.Params, "key", "")
	if key == "" {
		opts.Respond(false, nil, errInvalidParams("key required"), nil)
		return nil
	}

	agentID := session.ResolveSessionAgentID(key)
	newEntry, err := session.ResetSession(agentID, key, env)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":    true,
		"key":   key,
		"entry": newEntry,
	}, nil, nil)
	return nil
}

// --- sessions.delete ---

func SessionsDeleteHandler(opts HandlerOpts) error {
	env := envGetter()
	key := stringParam(opts.Params, "key", "")
	if key == "" {
		opts.Respond(false, nil, errInvalidParams("key required"), nil)
		return nil
	}

	// Protect main session
	if session.IsMainSession(key) {
		opts.Respond(false, nil, errInvalidParams("cannot delete main session"), nil)
		return nil
	}

	deleteTranscript := boolParam(opts.Params, "deleteTranscript", true)
	agentID := session.ResolveSessionAgentID(key)

	deleted, archived, err := session.DeleteSession(agentID, key, env, deleteTranscript)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"key":      key,
		"deleted":  deleted,
		"archived": archived,
	}, nil, nil)
	return nil
}

// --- sessions.compact ---

func SessionsCompactHandler(opts HandlerOpts) error {
	env := envGetter()
	key := stringParam(opts.Params, "key", "")
	if key == "" {
		opts.Respond(false, nil, errInvalidParams("key required"), nil)
		return nil
	}

	agentID := session.ResolveSessionAgentID(key)
	storePath := session.ResolveDefaultSessionStorePath(agentID, env)
	store, _ := session.LoadSessionStore(storePath)
	entry, ok := store[key]
	if !ok || entry.SessionID == "" {
		reason := "no sessionId"
		opts.Respond(true, map[string]interface{}{
			"ok":        true,
			"key":       key,
			"compacted": false,
			"reason":    reason,
		}, nil, nil)
		return nil
	}

	maxLines := intParam(opts.Params, "maxLines", 400)
	transcriptPath := session.ResolveSessionFilePath(entry.SessionID, &agentID, env)

	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		reason := "no transcript"
		opts.Respond(true, map[string]interface{}{
			"ok":        true,
			"key":       key,
			"compacted": false,
			"reason":    reason,
		}, nil, nil)
		return nil
	}

	// Read all lines
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	lines := strings.Split(string(data), "\n")
	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) <= maxLines {
		opts.Respond(true, map[string]interface{}{
			"ok":        true,
			"key":       key,
			"compacted": false,
			"reason":    fmt.Sprintf("only %d lines (max %d)", len(lines), maxLines),
		}, nil, nil)
		return nil
	}

	// Backup original file
	backupPath := transcriptPath + ".bak"
	_ = os.Rename(transcriptPath, backupPath)

	// Keep header (first line) + last maxLines-1 lines
	kept := []string{lines[0]}
	startIdx := len(lines) - (maxLines - 1)
	if startIdx < 1 {
		startIdx = 1
	}
	kept = append(kept, lines[startIdx:]...)

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(kept, "\n")+"\n"), 0600); err != nil {
		// Restore from backup
		_ = os.Rename(backupPath, transcriptPath)
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	keptCount := len(kept)
	opts.Respond(true, map[string]interface{}{
		"ok":        true,
		"key":       key,
		"compacted": true,
		"archived":  backupPath,
		"kept":      keptCount,
	}, nil, nil)
	return nil
}

// --- Helper functions ---

func classifySessionKind(key string) string {
	if session.IsMainSession(key) || strings.HasPrefix(key, "custom:") {
		return "direct"
	}
	if strings.Contains(key, ":group:") {
		return "group"
	}
	if strings.Contains(key, "global") {
		return "global"
	}
	parts := strings.Split(key, ":")
	for _, ch := range []string{"feishu", "dingtalk", "telegram", "slack", "wechat", "weixin", "wework", "qq"} {
		for _, p := range parts {
			if p == ch {
				return "direct"
			}
		}
	}
	return "unknown"
}

func envGetter() func(string) string {
	return os.Getenv
}

func stringParam(params map[string]interface{}, key, defaultVal string) string {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return defaultVal
	}
	return s
}

func intParam(params map[string]interface{}, key string, defaultVal int) int {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return defaultVal
	}
}

func boolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
}

func stringSliceParam(params map[string]interface{}, key string) []string {
	if params == nil {
		return nil
	}
	v, ok := params[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return arr
	default:
		return nil
	}
}
