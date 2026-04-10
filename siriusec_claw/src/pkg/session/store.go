package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// SessionEntry represents a single session's metadata.
type SessionEntry struct {
	SessionID          string      `json:"sessionId"`
	UpdatedAt          int64       `json:"updatedAt"`
	SessionFile        string      `json:"sessionFile,omitempty"`
	Label              string      `json:"label,omitempty"`
	Channel            string      `json:"channel,omitempty"`
	ChatType           string      `json:"chatType,omitempty"`
	ThinkingLevel      string      `json:"thinkingLevel,omitempty"`
	VerboseLevel       string      `json:"verboseLevel,omitempty"`
	SystemPromptReport interface{} `json:"systemPromptReport,omitempty"`
	SkillsSnapshot     interface{} `json:"skillsSnapshot,omitempty"`
	Tools              interface{} `json:"tools,omitempty"`
	Model              string      `json:"model,omitempty"`
	ModelProvider      string      `json:"modelProvider,omitempty"`
	SendPolicy         string      `json:"sendPolicy,omitempty"`
}

// SessionStore is a map of session key → SessionEntry.
type SessionStore map[string]SessionEntry

var storeMu sync.Mutex

// ResolveAgentSessionsDir returns the sessions directory for a given agent.
func ResolveAgentSessionsDir(agentID string, env func(string) string) string {
	if agentID == "" {
		agentID = "main"
	}
	stateDir := paths.ResolveStateDir(env)
	return filepath.Join(stateDir, "agents", agentID, "sessions")
}

// ResolveDefaultSessionStorePath returns the sessions.json path for a given agent.
func ResolveDefaultSessionStorePath(agentID string, env func(string) string) string {
	return filepath.Join(ResolveAgentSessionsDir(agentID, env), "sessions.json")
}

// ResolveSessionFilePath returns the transcript file path for a sessionID.
func ResolveSessionFilePath(sessionID string, agentID *string, env func(string) string) string {
	aid := "main"
	if agentID != nil && *agentID != "" {
		aid = *agentID
	}
	dir := ResolveAgentSessionsDir(aid, env)
	return filepath.Join(dir, sessionID+".jsonl")
}

// LoadSessionStore reads sessions.json from disk.
func LoadSessionStore(storePath string) (SessionStore, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(SessionStore), nil
		}
		return nil, err
	}
	var store SessionStore
	if err := json.Unmarshal(data, &store); err != nil {
		return make(SessionStore), nil
	}
	return store, nil
}

// SaveSessionStore writes the session store to disk atomically.
func SaveSessionStore(storePath string, store SessionStore) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	dir := filepath.Dir(storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storePath, data, 0600)
}

// LoadCombinedSessionStore loads and merges session stores from multiple agents.
func LoadCombinedSessionStore(agentIDs []string, env func(string) string) (SessionStore, error) {
	combined := make(SessionStore)
	for _, agentID := range agentIDs {
		storePath := ResolveDefaultSessionStorePath(agentID, env)
		store, err := LoadSessionStore(storePath)
		if err != nil {
			continue
		}
		for k, v := range store {
			combined[k] = v
		}
	}
	return combined, nil
}

// UpdateSessionUpdatedAt updates the updatedAt timestamp for a session.
func UpdateSessionUpdatedAt(agentID, sessionKey string, env func(string) string, tsMs int64) error {
	if agentID == "" {
		agentID = "main"
	}
	if tsMs == 0 {
		tsMs = time.Now().UnixMilli()
	}
	storePath := ResolveDefaultSessionStorePath(agentID, env)
	store, err := LoadSessionStore(storePath)
	if err != nil {
		return err
	}
	entry, ok := store[sessionKey]
	if !ok {
		return nil
	}
	entry.UpdatedAt = tsMs
	store[sessionKey] = entry
	return SaveSessionStore(storePath, store)
}

// EnsureSession ensures a session exists in the store; creates if missing.
// Returns the entry and whether it was newly created.
func EnsureSession(agentID, sessionKey string, env func(string) string, label string) (SessionEntry, bool, error) {
	if agentID == "" {
		agentID = "main"
	}
	storePath := ResolveDefaultSessionStorePath(agentID, env)
	store, err := LoadSessionStore(storePath)
	if err != nil {
		store = make(SessionStore)
	}

	if entry, ok := store[sessionKey]; ok {
		return entry, false, nil
	}

	sessionID := uuid.New().String()
	entry := SessionEntry{
		SessionID:   sessionID,
		UpdatedAt:   time.Now().UnixMilli(),
		SessionFile: sessionID + ".jsonl",
		Label:       label,
	}
	store[sessionKey] = entry
	if err := SaveSessionStore(storePath, store); err != nil {
		return entry, false, err
	}

	transcriptPath := ResolveSessionFilePath(sessionID, &agentID, env)
	_ = EnsureTranscriptFile(transcriptPath, sessionID)

	return entry, true, nil
}

// CreateSession creates a new session in the store.
func CreateSession(agentID, sessionKey string, env func(string) string, entry SessionEntry) error {
	if agentID == "" {
		agentID = "main"
	}
	storePath := ResolveDefaultSessionStorePath(agentID, env)
	store, err := LoadSessionStore(storePath)
	if err != nil {
		store = make(SessionStore)
	}
	if entry.SessionID == "" {
		entry.SessionID = uuid.New().String()
	}
	if entry.UpdatedAt == 0 {
		entry.UpdatedAt = time.Now().UnixMilli()
	}
	if entry.SessionFile == "" {
		entry.SessionFile = entry.SessionID + ".jsonl"
	}
	store[sessionKey] = entry
	if err := SaveSessionStore(storePath, store); err != nil {
		return err
	}

	transcriptPath := ResolveSessionFilePath(entry.SessionID, &agentID, env)
	return EnsureTranscriptFile(transcriptPath, entry.SessionID)
}

// DeleteSession removes a session from the store and optionally deletes its transcript.
func DeleteSession(agentID, sessionKey string, env func(string) string, deleteTranscript bool) (bool, []string, error) {
	if agentID == "" {
		agentID = "main"
	}
	storePath := ResolveDefaultSessionStorePath(agentID, env)
	store, err := LoadSessionStore(storePath)
	if err != nil {
		return false, nil, err
	}

	entry, ok := store[sessionKey]
	if !ok {
		return false, nil, nil
	}

	var archived []string
	if deleteTranscript && entry.SessionID != "" {
		transcriptPath := ResolveSessionFilePath(entry.SessionID, &agentID, env)
		if _, err := os.Stat(transcriptPath); err == nil {
			archived = append(archived, transcriptPath)
			_ = os.Remove(transcriptPath)
		}
	}

	delete(store, sessionKey)
	if err := SaveSessionStore(storePath, store); err != nil {
		return false, archived, err
	}
	return true, archived, nil
}

// ResetSession creates a new session ID/transcript but preserves settings.
func ResetSession(agentID, sessionKey string, env func(string) string) (SessionEntry, error) {
	if agentID == "" {
		agentID = "main"
	}
	storePath := ResolveDefaultSessionStorePath(agentID, env)
	store, err := LoadSessionStore(storePath)
	if err != nil {
		return SessionEntry{}, err
	}

	oldEntry, ok := store[sessionKey]
	if !ok {
		return SessionEntry{}, nil
	}

	newSessionID := uuid.New().String()
	newEntry := SessionEntry{
		SessionID:     newSessionID,
		UpdatedAt:     time.Now().UnixMilli(),
		SessionFile:   newSessionID + ".jsonl",
		Label:         oldEntry.Label,
		Channel:       oldEntry.Channel,
		ChatType:      oldEntry.ChatType,
		ThinkingLevel: oldEntry.ThinkingLevel,
		VerboseLevel:  oldEntry.VerboseLevel,
		Model:         oldEntry.Model,
		ModelProvider: oldEntry.ModelProvider,
		SendPolicy:    oldEntry.SendPolicy,
	}
	store[sessionKey] = newEntry
	if err := SaveSessionStore(storePath, store); err != nil {
		return newEntry, err
	}

	transcriptPath := ResolveSessionFilePath(newSessionID, &agentID, env)
	_ = EnsureTranscriptFile(transcriptPath, newSessionID)
	return newEntry, nil
}
