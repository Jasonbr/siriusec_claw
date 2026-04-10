package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// openaiModel implements the Model interface using the OpenAI-compatible chat completions API.
type openaiModel struct {
	provider string
	modelID  string
	info     *ProviderInfo
	client   *http.Client
}

// newOpenAIModel creates a model backed by an OpenAI-compatible API.
func newOpenAIModel(provider, modelID string, info *ProviderInfo) *openaiModel {
	return &openaiModel{
		provider: provider,
		modelID:  modelID,
		info:     info,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// --- OpenAI request/response types ---

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	Stream      bool            `json:"stream"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"` // string or []openaiContentPart
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openaiResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Model   string          `json:"model"`
	Choices []openaiChoice  `json:"choices"`
	Usage   *openaiUsage    `json:"usage,omitempty"`
	Error   *openaiAPIError `json:"error,omitempty"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"` // "stop", "tool_calls", "length"
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Complete sends messages to the OpenAI-compatible API and returns the result.
func (m *openaiModel) Complete(messages []Message, opts *CompletionOpts) (*CompletionResult, error) {
	return m.CompleteWithContext(context.Background(), messages, opts)
}

// CompleteWithContext sends messages with a context for cancellation.
func (m *openaiModel) CompleteWithContext(ctx context.Context, messages []Message, opts *CompletionOpts) (*CompletionResult, error) {
	// Convert messages
	var oaiMsgs []openaiMessage
	for _, msg := range messages {
		oaiMsg := openaiMessage{Role: msg.Role}

		// Handle tool role messages
		if msg.Role == "tool" {
			oaiMsg.ToolCallID = msg.ToolCallID
			switch c := msg.Content.(type) {
			case string:
				oaiMsg.Content = c
			default:
				oaiMsg.Content = c
			}
			oaiMsgs = append(oaiMsgs, oaiMsg)
			continue
		}

		// Handle assistant messages with tool calls
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			switch c := msg.Content.(type) {
			case string:
				if c != "" {
					oaiMsg.Content = c
				}
			default:
				oaiMsg.Content = c
			}
			for _, tc := range msg.ToolCalls {
				args := "{}"
				if tc.Input != nil {
					if b, err := json.Marshal(tc.Input); err == nil {
						args = string(b)
					}
				}
				oaiMsg.ToolCalls = append(oaiMsg.ToolCalls, openaiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Name,
						Arguments: args,
					},
				})
			}
			oaiMsgs = append(oaiMsgs, oaiMsg)
			continue
		}

		// Normal messages
		switch c := msg.Content.(type) {
		case string:
			oaiMsg.Content = c
		default:
			oaiMsg.Content = c
		}
		oaiMsgs = append(oaiMsgs, oaiMsg)
	}

	// Build request
	reqBody := openaiRequest{
		Model:    m.modelID,
		Messages: oaiMsgs,
		Stream:   false,
	}
	if opts != nil {
		if opts.MaxTokens > 0 {
			reqBody.MaxTokens = opts.MaxTokens
		}
		reqBody.Temperature = opts.Temperature
		reqBody.Stop = opts.StopSequences

		// Convert tools
		for _, t := range opts.Tools {
			reqBody.Tools = append(reqBody.Tools, openaiTool{
				Type: "function",
				Function: openaiToolFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Build HTTP request
	endpoint := m.info.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if m.info.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.info.APIKey)
	}

	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr openaiResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != nil {
			return nil, fmt.Errorf("API error %d: [%s] %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var oaiResp openaiResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from API (no choices)")
	}

	choice := oaiResp.Choices[0]

	// Convert to our format
	var content []ContentBlock
	if textContent, ok := choice.Message.Content.(string); ok && textContent != "" {
		content = append(content, ContentBlock{
			Type: "text",
			Text: textContent,
		})
	}

	// Convert tool calls
	for _, tc := range choice.Message.ToolCalls {
		var input interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
		content = append(content, ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	// Map finish reason
	stopReason := "end_turn"
	switch choice.FinishReason {
	case "stop":
		stopReason = "end_turn"
	case "tool_calls":
		stopReason = "tool_use"
	case "length":
		stopReason = "max_tokens"
	}

	result := &CompletionResult{
		Content:    content,
		StopReason: stopReason,
		Model:      oaiResp.Model,
	}

	if oaiResp.Usage != nil {
		result.Usage = &TokenUsage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
		}
	}

	return result, nil
}
