package context

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
	"github.com/siriusec/siriusec_claw/pkg/agent"
)

// TokenEncoder wraps tiktoken for thread-safe usage.
type TokenEncoder struct {
	encoding string
}

// NewTokenEncoder creates a new encoder for the given model.
// Common encodings: "cl100k_base" (GPT-4, Claude), "o200k_base" (GPT-4o)
func NewTokenEncoder(model string) *TokenEncoder {
	encoding := "cl100k_base" // default
	// Map common model names to encodings
	switch model {
	case "gpt-4o", "gpt-4o-mini":
		encoding = "o200k_base"
	case "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "claude-3-opus", "claude-3-sonnet", "claude-3-haiku":
		encoding = "cl100k_base"
	}
	return &TokenEncoder{encoding: encoding}
}

// Encode returns the token IDs for the given text.
func (e *TokenEncoder) Encode(text string) ([]int, error) {
	encoding, err := tiktoken.GetEncoding(e.encoding)
	if err != nil {
		return nil, err
	}
	tokens := encoding.Encode(text, nil, nil)
	return tokens, nil
}

// Count returns the number of tokens in the text.
func (e *TokenEncoder) Count(text string) int {
	tokens, err := e.Encode(text)
	if err != nil {
		// Fallback to estimation
		return EstimateTokens(text)
	}
	return len(tokens)
}

// CountMessages returns the total token count for a slice of messages.
func (e *TokenEncoder) CountMessages(messages []agent.Message) int {
	total := 0
	for _, msg := range messages {
		total += e.Count(contentToString(msg.Content))
		// Tool calls overhead
		if len(msg.ToolCalls) > 0 {
			total += len(msg.ToolCalls) * 20
		}
	}
	return total
}

// Budget defines token allocation across context components.
type Budget struct {
	// SystemPromptMax is the max tokens for the system prompt.
	SystemPromptMax int `json:"systemPromptMax"`
	// HistoryMax is the max tokens for conversation history.
	HistoryMax int `json:"historyMax"`
	// BootstrapMax is the max tokens for bootstrap files.
	BootstrapMax int `json:"bootstrapMax"`
	// CompletionMax is the max tokens reserved for model output.
	CompletionMax int `json:"completionMax"`
	// ModelContextWindow is the total context window of the model.
	ModelContextWindow int `json:"modelContextWindow"`
}

// DefaultBudget returns a sensible default token budget.
// modelContext is the model's total context window (e.g. 128000 for Claude).
func DefaultBudget(modelContext int) Budget {
	if modelContext <= 0 {
		modelContext = 128000
	}
	completion := 4096
	available := modelContext - completion

	return Budget{
		ModelContextWindow: modelContext,
		CompletionMax:      completion,
		SystemPromptMax:    available / 5,     // 20%
		HistoryMax:         available * 3 / 5, // 60%
		BootstrapMax:       available / 5,     // 20%
	}
}

// Usage tracks actual token usage across components.
type Usage struct {
	SystemPromptTokens int `json:"systemPromptTokens"`
	HistoryTokens      int `json:"historyTokens"`
	BootstrapTokens    int `json:"bootstrapTokens"`
	CompletionTokens   int `json:"completionTokens"`
	TotalTokens        int `json:"totalTokens"`
}

// InBudget checks if usage is within the budget.
func (u *Usage) InBudget(b Budget) bool {
	return u.SystemPromptTokens <= b.SystemPromptMax &&
		u.HistoryTokens <= b.HistoryMax &&
		u.BootstrapTokens <= b.BootstrapMax
}

// OverBudgetComponent returns which component is over budget, if any.
func (u *Usage) OverBudgetComponent(b Budget) string {
	if u.SystemPromptTokens > b.SystemPromptMax {
		return "system_prompt"
	}
	if u.HistoryTokens > b.HistoryMax {
		return "history"
	}
	if u.BootstrapTokens > b.BootstrapMax {
		return "bootstrap"
	}
	return ""
}

// EstimateTokens provides a rough token count estimation (fallback).
// Average: ~4 chars per token for English, ~2 chars per token for CJK.
// Note: Use TokenEncoder.Count() for precise counting with tiktoken.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	// Simple heuristic: count CJK characters separately
	cjk := 0
	ascii := 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3040 && r <= 0x309F || r >= 0x30A0 && r <= 0x30FF {
			cjk++
		} else if r < 128 {
			ascii++
		} else {
			cjk++
		}
	}
	return (ascii / 4) + (cjk / 2) + 1
}

// PreciseTokens uses tiktoken for accurate token counting.
// Returns the token count and any error encountered.
func PreciseTokens(text string, model string) (int, error) {
	encoder := NewTokenEncoder(model)
	return encoder.Count(text), nil
}

// EstimateMessagesTokens estimates total tokens for a slice of messages.
// Deprecated: Use TokenEncoder.CountMessages() for precise counting.
func EstimateMessagesTokens(messages []agent.Message) int {
	total := 0
	for _, msg := range messages {
		total += EstimateTokens(contentToString(msg.Content))
		// Tool calls have overhead
		if len(msg.ToolCalls) > 0 {
			total += len(msg.ToolCalls) * 20 // rough overhead per tool call
		}
	}
	return total
}

// PreciseMessagesTokens returns accurate token count using tiktoken.
func PreciseMessagesTokens(messages []agent.Message, model string) int {
	encoder := NewTokenEncoder(model)
	return encoder.CountMessages(messages)
}

// contentToString converts Message.Content (interface{}) to string.
func contentToString(content interface{}) string {
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var result string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if text, ok := m["text"].(string); ok {
						result += text
					}
				}
			}
		}
		return result
	default:
		return fmt.Sprintf("%v", content)
	}
}

// CompactionResult holds the result of history compaction.
type CompactionResult struct {
	OriginalMessages  int    `json:"originalMessages"`
	CompactedMessages int    `json:"compactedMessages"`
	Summary           string `json:"summary"`
	TokensSaved       int    `json:"tokensSaved"`
}

// ShouldCompact determines if history needs compaction based on budget.
func ShouldCompact(messages []agent.Message, budget Budget) bool {
	historyTokens := EstimateMessagesTokens(messages)
	return historyTokens > budget.HistoryMax
}

// CompactHistory compacts older messages into a summary.
// Keeps the last `keepRecent` messages intact and summarizes the rest.
func CompactHistory(messages []agent.Message, budget Budget, keepRecent int) ([]agent.Message, *CompactionResult) {
	if keepRecent <= 0 {
		keepRecent = 10
	}

	if len(messages) <= keepRecent {
		return messages, nil
	}

	// Split messages: older ones to summarize, recent ones to keep
	older := messages[:len(messages)-keepRecent]
	recent := messages[len(messages)-keepRecent:]

	// Build a simple summary from older messages
	olderTokens := EstimateMessagesTokens(older)
	summary := buildSummary(older)

	// Create a system-like message with the summary
	summaryMsg := agent.Message{
		Role:    "user",
		Content: "[Conversation Summary]\n" + summary + "\n[End of Summary - Recent messages follow]",
	}

	result := make([]agent.Message, 0, 1+len(recent))
	result = append(result, summaryMsg)
	result = append(result, recent...)

	newTokens := EstimateMessagesTokens(result)

	return result, &CompactionResult{
		OriginalMessages:  len(messages),
		CompactedMessages: len(result),
		Summary:           summary,
		TokensSaved:       olderTokens - newTokens,
	}
}

// buildSummary creates a text summary from older messages.
func buildSummary(messages []agent.Message) string {
	var summary string
	for _, msg := range messages {
		contentStr := contentToString(msg.Content)
		switch msg.Role {
		case "user":
			if len(contentStr) > 200 {
				summary += "User asked: " + contentStr[:200] + "...\n"
			} else {
				summary += "User asked: " + contentStr + "\n"
			}
		case "assistant":
			if len(contentStr) > 200 {
				summary += "Assistant replied: " + contentStr[:200] + "...\n"
			} else if contentStr != "" {
				summary += "Assistant replied: " + contentStr + "\n"
			}
			if len(msg.ToolCalls) > 0 {
				summary += "  (used tools)\n"
			}
		}
	}
	if len(summary) > 3000 {
		summary = summary[:3000] + "\n... [summary truncated]"
	}
	return summary
}
