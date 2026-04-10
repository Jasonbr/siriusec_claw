package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/siriusec/siriusec_claw/pkg/config"
)

// ModelFactory creates LLM model instances for agent execution.
type ModelFactory interface {
	// CreateModel returns a model instance for the given provider and model ID.
	CreateModel(provider, modelID string) (Model, error)
}

// Model represents an LLM model interface.
type Model interface {
	// Complete sends a prompt and returns a response.
	Complete(messages []Message, opts *CompletionOpts) (*CompletionResult, error)
	// CompleteWithContext sends a prompt with context for cancellation.
	CompleteWithContext(ctx context.Context, messages []Message, opts *CompletionOpts) (*CompletionResult, error)
}

// Message represents a chat message for the LLM.
type Message struct {
	Role       string         `json:"role"`    // "system", "user", "assistant", "tool"
	Content    interface{}    `json:"content"` // string or []ContentBlock
	ToolCalls  []ToolCallInfo `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// ToolCallInfo describes a tool call requested by the assistant.
type ToolCallInfo struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

// CompletionOpts configures a completion request.
type CompletionOpts struct {
	MaxTokens     int       `json:"maxTokens,omitempty"`
	Temperature   *float64  `json:"temperature,omitempty"`
	StopSequences []string  `json:"stopSequences,omitempty"`
	Tools         []ToolDef `json:"tools,omitempty"`
	SystemPrompt  string    `json:"systemPrompt,omitempty"`
}

// ToolDef defines a tool available to the model.
type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// CompletionResult is the response from a model completion.
type CompletionResult struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stopReason"`
	Usage      *TokenUsage    `json:"usage,omitempty"`
	Model      string         `json:"model,omitempty"`
}

// ContentBlock is a piece of content in a completion response.
type ContentBlock struct {
	Type  string      `json:"type"` // "text", "tool_use"
	Text  string      `json:"text,omitempty"`
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	InputTokens      int `json:"inputTokens"`
	OutputTokens     int `json:"outputTokens"`
	CacheReadTokens  int `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int `json:"cacheWriteTokens,omitempty"`
}

// ProviderInfo holds resolved provider connection details.
type ProviderInfo struct {
	Name    string
	BaseURL string
	APIKey  string
	APIType string // "anthropic-messages" or "openai"
}

// ResolveModelRef resolves a model reference string to provider and model ID.
// Format: "provider/modelId" or just "modelId" (defaults to anthropic).
func ResolveModelRef(ref string) (provider, modelID string) {
	if ref == "" {
		return "anthropic", "claude-sonnet-4-5-20250929"
	}
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "anthropic", ref
}

// ResolveProviderInfo resolves a provider's connection details from config.
func ResolveProviderInfo(providerName string, cfg *config.ClawConfig) *ProviderInfo {
	info := &ProviderInfo{Name: providerName}

	// Check config for custom providers
	if cfg != nil && cfg.Models != nil && cfg.Models.Providers != nil {
		if p, ok := cfg.Models.Providers[providerName]; ok {
			info.BaseURL = p.BaseURL
			info.APIKey = resolveAPIKey(p.APIKey)
			if p.API != nil {
				info.APIType = *p.API
			}
			return info
		}
	}

	// Built-in provider defaults
	switch providerName {
	case "anthropic":
		info.BaseURL = "https://api.anthropic.com"
		info.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		info.APIType = "anthropic-messages"
	case "openai":
		info.BaseURL = "https://api.openai.com/v1"
		info.APIKey = os.Getenv("OPENAI_API_KEY")
		info.APIType = "openai"
	case "deepseek":
		info.BaseURL = "https://api.deepseek.com"
		info.APIKey = os.Getenv("DEEPSEEK_API_KEY")
		info.APIType = "openai"
	case "openrouter":
		info.BaseURL = "https://openrouter.ai/api/v1"
		info.APIKey = os.Getenv("OPENROUTER_API_KEY")
		info.APIType = "openai"
	case "groq":
		info.BaseURL = "https://api.groq.com/openai/v1"
		info.APIKey = os.Getenv("GROQ_API_KEY")
		info.APIType = "openai"
	case "together":
		info.BaseURL = "https://api.together.xyz/v1"
		info.APIKey = os.Getenv("TOGETHER_API_KEY")
		info.APIType = "openai"
	case "moonshot", "kimi":
		info.BaseURL = "https://api.moonshot.cn/v1"
		info.APIKey = os.Getenv("MOONSHOT_API_KEY")
		info.APIType = "openai"
	case "minimax":
		info.BaseURL = "https://api.minimax.chat/v1"
		info.APIKey = os.Getenv("MINIMAX_API_KEY")
		info.APIType = "openai"
	case "mistral":
		info.BaseURL = "https://api.mistral.ai/v1"
		info.APIKey = os.Getenv("MISTRAL_API_KEY")
		info.APIType = "openai"
	case "ollama":
		info.BaseURL = envOrDefault("OLLAMA_HOST", "http://localhost:11434")
		info.APIKey = ""
		info.APIType = "openai"
	default:
		// Try environment variable pattern: {PROVIDER}_API_KEY
		envKey := strings.ToUpper(strings.ReplaceAll(providerName, "-", "_")) + "_API_KEY"
		info.APIKey = os.Getenv(envKey)
		info.APIType = "openai"
	}

	return info
}

// CreateModelFactoryFromConfig builds a ModelFactory from the application config.
func CreateModelFactoryFromConfig(cfg *config.ClawConfig) ModelFactory {
	return &configModelFactory{cfg: cfg}
}

type configModelFactory struct {
	cfg *config.ClawConfig
}

func (f *configModelFactory) CreateModel(provider, modelID string) (Model, error) {
	info := ResolveProviderInfo(provider, f.cfg)
	if info.APIKey == "" && provider != "ollama" {
		return nil, fmt.Errorf("no API key found for provider %q (set %s_API_KEY env var)",
			provider, strings.ToUpper(strings.ReplaceAll(provider, "-", "_")))
	}

	// Use real OpenAI-compatible HTTP client for all providers
	// (DashScope/Bailian, DeepSeek, Moonshot, Groq, etc. are all OpenAI-compatible)
	return newOpenAIModel(provider, modelID, info), nil
}

func resolveAPIKey(key string) string {
	if strings.HasPrefix(key, "$") {
		envVar := strings.TrimPrefix(key, "$")
		return os.Getenv(envVar)
	}
	return key
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
