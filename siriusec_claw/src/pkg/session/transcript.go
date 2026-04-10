package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// TranscriptHeader is the first line of a transcript JSONL file.
type TranscriptHeader struct {
	Type      string `json:"type"`      // "session"
	Version   int    `json:"version"`   // 2
	ID        string `json:"id"`        // session UUID
	Timestamp string `json:"timestamp"` // ISO8601
	Cwd       string `json:"cwd,omitempty"`
}

// TranscriptMessage represents a single chat message in the transcript.
type TranscriptMessage struct {
	Role       string         `json:"role"` // "user", "assistant", "toolResult"
	Content    []ContentBlock `json:"content"`
	Timestamp  int64          `json:"timestamp"`
	Usage      *Usage         `json:"usage,omitempty"`
	StopReason string         `json:"stopReason,omitempty"`
	Provider   string         `json:"provider,omitempty"`
	Model      string         `json:"model,omitempty"`
	DurationMs *int64         `json:"durationMs,omitempty"`
	ToolCallID string         `json:"toolCallId,omitempty"`
	ToolName   string         `json:"toolName,omitempty"`
	IsError    bool           `json:"isError,omitempty"`
}

// WrappedMessage is the envelope format for transcript entries.
type WrappedMessage struct {
	Type      string             `json:"type"` // "message", "token_usage"
	ID        string             `json:"id,omitempty"`
	ParentID  string             `json:"parentId,omitempty"`
	Timestamp string             `json:"timestamp,omitempty"`
	Message   *TranscriptMessage `json:"message,omitempty"`
	// token_usage fields
	SessionID     string `json:"sessionId,omitempty"`
	RequestID     string `json:"requestId,omitempty"`
	Input         int    `json:"input,omitempty"`
	Output        int    `json:"output,omitempty"`
	CacheRead     int    `json:"cacheRead,omitempty"`
	CacheWrite    int    `json:"cacheWrite,omitempty"`
	CacheCreation int    `json:"cacheCreation,omitempty"`
	TotalTokens   int    `json:"totalTokens,omitempty"`
}

// ContentBlock is a single content element in a message.
type ContentBlock struct {
	Type     string      `json:"type"` // "text", "tool_use", "tool_result", "thinking"
	Text     string      `json:"text,omitempty"`
	Name     string      `json:"name,omitempty"`  // tool name
	ID       string      `json:"id,omitempty"`    // tool_use ID
	Input    interface{} `json:"input,omitempty"` // tool_use input
	MimeType string      `json:"mimeType,omitempty"`
	Data     string      `json:"data,omitempty"` // base64
	IsError  bool        `json:"is_error,omitempty"`
}

// Usage represents token usage for a single message.
type Usage struct {
	Input       int   `json:"input"`
	Output      int   `json:"output"`
	CacheRead   int   `json:"cacheRead"`
	CacheWrite  int   `json:"cacheWrite"`
	TotalTokens int   `json:"totalTokens"`
	Cost        *Cost `json:"cost,omitempty"`
}

// Cost represents the cost breakdown for a message.
type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead,omitempty"`
	CacheWrite float64 `json:"cacheWrite,omitempty"`
	Total      float64 `json:"total"`
}

// PreviewItem is a single message in a session preview.
type PreviewItem struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// EnsureTranscriptFile creates a transcript file with header if it doesn't exist.
func EnsureTranscriptFile(path, sessionID string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	header := TranscriptHeader{
		Type:      "session",
		Version:   2,
		ID:        sessionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if cwd, err := os.Getwd(); err == nil {
		header.Cwd = cwd
	}
	data, err := json.Marshal(header)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}

// AppendTranscriptLine appends a JSON object as a new line to the transcript.
func AppendTranscriptLine(path string, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// AppendUserMessage appends a user message to the transcript.
func AppendUserMessage(path, text string) error {
	msg := TranscriptMessage{
		Role:      "user",
		Content:   []ContentBlock{{Type: "text", Text: text}},
		Timestamp: time.Now().UnixMilli(),
	}
	return AppendTranscriptLine(path, msg)
}

// AppendTranscriptMessage appends a pre-built TranscriptMessage to the transcript.
func AppendTranscriptMessage(path string, msg *TranscriptMessage) error {
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	return AppendTranscriptLine(path, msg)
}

// AppendAssistantMessage appends a plain assistant message to the transcript.
func AppendAssistantMessage(path, text string) error {
	msg := TranscriptMessage{
		Role:      "assistant",
		Content:   []ContentBlock{{Type: "text", Text: text}},
		Timestamp: time.Now().UnixMilli(),
	}
	return AppendTranscriptLine(path, msg)
}

// AssistantMessageOpts provides options for assistant messages with usage data.
type AssistantMessageOpts struct {
	Usage      *Usage
	Provider   string
	Model      string
	StopReason string
	DurationMs *int64
}

// AppendAssistantMessageWithUsage appends an assistant message with token/cost data.
func AppendAssistantMessageWithUsage(path, text string, opts *AssistantMessageOpts) error {
	msg := TranscriptMessage{
		Role:      "assistant",
		Content:   []ContentBlock{{Type: "text", Text: text}},
		Timestamp: time.Now().UnixMilli(),
	}
	if opts != nil {
		msg.Usage = opts.Usage
		msg.Provider = opts.Provider
		msg.Model = opts.Model
		msg.StopReason = opts.StopReason
		msg.DurationMs = opts.DurationMs
	}
	return AppendTranscriptLine(path, msg)
}

// ReadTranscriptMessages reads all messages from a transcript file.
func ReadTranscriptMessages(path string, limit int) ([]TranscriptMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []TranscriptMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Skip header (first line)
		if lineNum == 1 {
			var peek map[string]interface{}
			if json.Unmarshal(line, &peek) == nil {
				if t, _ := peek["type"].(string); t == "session" {
					continue
				}
			}
		}

		// Try wrapped format first
		var wrapped WrappedMessage
		if json.Unmarshal(line, &wrapped) == nil && wrapped.Type == "message" && wrapped.Message != nil {
			messages = append(messages, *wrapped.Message)
			continue
		}

		// Try direct message format
		var msg TranscriptMessage
		if json.Unmarshal(line, &msg) == nil && msg.Role != "" {
			messages = append(messages, msg)
		}
		// Skip token_usage and other non-message lines
	}

	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
	return messages, scanner.Err()
}

// ReadSessionPreviewItems reads the last N messages as preview items.
func ReadSessionPreviewItems(path string, maxItems, maxChars int) ([]PreviewItem, error) {
	if maxItems <= 0 {
		maxItems = 12
	}
	if maxChars <= 0 {
		maxChars = 240
	}

	messages, err := ReadTranscriptMessages(path, maxItems)
	if err != nil {
		return nil, err
	}

	items := make([]PreviewItem, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		text := extractTextFromContent(msg.Content)
		if len(text) > maxChars {
			text = text[:maxChars] + "..."
		}
		if text == "" {
			continue
		}
		items = append(items, PreviewItem{
			Role: msg.Role,
			Text: text,
		})
	}
	return items, nil
}

func extractTextFromContent(blocks []ContentBlock) string {
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			return b.Text
		}
	}
	return ""
}
