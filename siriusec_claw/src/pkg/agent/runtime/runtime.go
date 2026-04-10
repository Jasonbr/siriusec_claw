package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/embed"
	"github.com/siriusec/siriusec_claw/pkg/agent"
	"github.com/siriusec/siriusec_claw/pkg/agent/skills"
	"github.com/siriusec/siriusec_claw/pkg/agent/tools"
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/logging"
	"github.com/siriusec/siriusec_claw/pkg/session"
)

var rtLog = logging.Sub("runtime")

// Options configures the agent runtime.
type Options struct {
	ModelFactory         agent.ModelFactory
	Tools                []tools.Tool
	ExecContext          tools.ExecutionContext
	ProjectRoot          string
	Config               *config.ClawConfig
	EnableSkills         bool
	EnableSandbox        bool
	EnableApprovalQueue  bool
	TokenTracking        bool
	AgentID              string
	SystemPromptOverride string
	MaxTurns             int
	TimeoutSeconds       int
}

// Request is a chat request to the agent runtime.
type Request struct {
	Message    string
	SessionID  string
	SessionKey string
	RunID      string
}

// Response is the result of an agent run.
type Response struct {
	RunID      string
	Output     string
	Thinking   string
	StopReason string
	Usage      *agent.TokenUsage
	Model      string
	Provider   string
	DurationMs int64
	ToolCalls  int
	Error      error
}

// StreamEvent is an event emitted during streaming execution.
type StreamEvent struct {
	Type      string // "text_delta", "tool_use", "tool_result", "message_stop", "error", "turn_complete"
	Text      string
	ToolID    string
	ToolName  string
	Input     interface{}
	Result    string
	IsError   bool
	TurnIndex int
	Model     string
}

// Runtime is the agent execution engine.
type Runtime struct {
	opts           Options
	model          agent.Model
	tools          []tools.Tool
	toolHandlerMap map[string]tools.ToolHandler
	skills         []skills.Entry
	mu             sync.Mutex
	cancelFuncs    map[string]context.CancelFunc
}

// New creates a new agent runtime with the given options.
func New(opts Options) (*Runtime, error) {
	if opts.AgentID == "" {
		opts.AgentID = "main"
	}
	if opts.MaxTurns <= 0 {
		opts.MaxTurns = 25
	}
	if opts.TimeoutSeconds <= 0 {
		opts.TimeoutSeconds = 300
	}

	// Build tool list with real handlers
	allTools := tools.BuiltinToolsWithHandlers(opts.ExecContext)
	allTools = append(allTools, opts.Tools...)

	// Build handler map
	handlerMap := make(map[string]tools.ToolHandler)
	for _, t := range allTools {
		if t.Handler != nil {
			handlerMap[t.Name] = t.Handler
		}
	}

	// Load skills if enabled
	var loadedSkills []skills.Entry
	if opts.EnableSkills {
		var err error
		bundledFS, _ := embed.SkillsFS()
		loadedSkills, err = skills.LoadWorkspaceEntriesWithFS(bundledFS, os.Getenv, opts.ProjectRoot, nil)
		if err != nil {
			rtLog.Warn("failed to load skills: %v", err)
		}
	}

	// Create model from factory
	var model agent.Model
	if opts.ModelFactory != nil {
		provider, modelID := resolveModelFromConfig(opts.Config, opts.AgentID)
		var err error
		model, err = opts.ModelFactory.CreateModel(provider, modelID)
		if err != nil {
			rtLog.Warn("failed to create model: %v", err)
		}
	}

	rt := &Runtime{
		opts:           opts,
		model:          model,
		tools:          allTools,
		toolHandlerMap: handlerMap,
		skills:         loadedSkills,
		cancelFuncs:    make(map[string]context.CancelFunc),
	}

	return rt, nil
}

// Run executes a synchronous agent request (wraps RunStream).
func (rt *Runtime) Run(ctx context.Context, req Request) *Response {
	ch, runID := rt.RunStream(ctx, req)
	req.RunID = runID

	startTime := time.Now()
	var output string
	var model string
	var stopReason string
	var runErr error
	toolCalls := 0

	for ev := range ch {
		switch ev.Type {
		case "text_delta":
			output += ev.Text
		case "tool_use":
			toolCalls++
		case "message_stop":
			stopReason = "end_turn"
			model = ev.Model
		case "error":
			runErr = fmt.Errorf("%s", ev.Text)
			stopReason = "error"
		}
	}

	if stopReason == "" {
		stopReason = "end_turn"
	}

	return &Response{
		RunID:      runID,
		Output:     output,
		StopReason: stopReason,
		Model:      model,
		DurationMs: time.Since(startTime).Milliseconds(),
		ToolCalls:  toolCalls,
		Error:      runErr,
	}
}

// RunStream executes an agent request with multi-turn agentic loop.
// Returns a channel of StreamEvents and the run ID.
func (rt *Runtime) RunStream(ctx context.Context, req Request) (<-chan StreamEvent, string) {
	ch := make(chan StreamEvent, 100)

	if req.RunID == "" {
		req.RunID = uuid.New().String()
	}
	runID := req.RunID

	go func() {
		defer close(ch)

		// Create cancellable context with timeout
		ctx, cancel := context.WithTimeout(ctx, time.Duration(rt.opts.TimeoutSeconds)*time.Second)
		defer cancel()

		rt.mu.Lock()
		rt.cancelFuncs[runID] = cancel
		rt.mu.Unlock()
		defer func() {
			rt.mu.Lock()
			delete(rt.cancelFuncs, runID)
			rt.mu.Unlock()
		}()

		if rt.model == nil {
			ch <- StreamEvent{Type: "error", Text: "No model configured. Please set up a model provider in the configuration.", IsError: true}
			return
		}

		// Build messages
		var messages []agent.Message

		systemPrompt := rt.buildSystemPrompt()
		if systemPrompt != "" {
			messages = append(messages, agent.Message{Role: "system", Content: systemPrompt})
		}

		// Load history
		if req.SessionID != "" {
			history := rt.loadHistory(req.SessionID)
			messages = append(messages, history...)
		}

		// User message
		messages = append(messages, agent.Message{Role: "user", Content: req.Message})

		// Build tool definitions
		var toolDefs []agent.ToolDef
		for _, t := range rt.tools {
			toolDefs = append(toolDefs, agent.ToolDef{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}

		completionOpts := &agent.CompletionOpts{
			MaxTokens:    4096,
			Tools:        toolDefs,
			SystemPrompt: systemPrompt,
		}

		var lastModel string

		// === Agentic Loop ===
		for turn := 0; turn < rt.opts.MaxTurns; turn++ {
			// Check context
			select {
			case <-ctx.Done():
				ch <- StreamEvent{Type: "error", Text: "Request cancelled or timed out", IsError: true}
				return
			default:
			}

			rtLog.Info("turn %d/%d, messages=%d", turn+1, rt.opts.MaxTurns, len(messages))

			// Call model
			result, err := rt.model.CompleteWithContext(ctx, messages, completionOpts)
			if err != nil {
				ch <- StreamEvent{Type: "error", Text: "Model error: " + err.Error(), IsError: true}
				return
			}

			if result.Model != "" {
				lastModel = result.Model
			}

			// Extract text and tool calls from response
			var textOutput string
			var pendingCalls []agent.ToolCallInfo

			for _, block := range result.Content {
				switch block.Type {
				case "text":
					textOutput += block.Text
				case "tool_use":
					pendingCalls = append(pendingCalls, agent.ToolCallInfo{
						ID:    block.ID,
						Name:  block.Name,
						Input: block.Input,
					})
				}
			}

			// Emit text if any
			if textOutput != "" {
				ch <- StreamEvent{Type: "text_delta", Text: textOutput, TurnIndex: turn}
			}

			// Build assistant message for conversation history
			assistantMsg := agent.Message{
				Role:    "assistant",
				Content: textOutput,
			}
			if len(pendingCalls) > 0 {
				assistantMsg.ToolCalls = pendingCalls
			}
			messages = append(messages, assistantMsg)

			// If no tool calls or model says done -> finish
			if len(pendingCalls) == 0 || result.StopReason != "tool_use" {
				ch <- StreamEvent{Type: "message_stop", Model: lastModel}
				return
			}

			// Execute each tool call
			for _, call := range pendingCalls {
				// Emit tool_use event
				ch <- StreamEvent{
					Type:      "tool_use",
					ToolID:    call.ID,
					ToolName:  call.Name,
					Input:     call.Input,
					TurnIndex: turn,
				}

				handler, ok := rt.toolHandlerMap[call.Name]
				var toolResult *tools.ToolResult
				if !ok {
					toolResult = &tools.ToolResult{
						Content: fmt.Sprintf("Error: unknown tool '%s'", call.Name),
						IsError: true,
					}
				} else {
					// Parse input to map
					inputMap := toStringMap(call.Input)
					var execErr error
					toolResult, execErr = handler(ctx, inputMap)
					if execErr != nil {
						toolResult = &tools.ToolResult{
							Content: "Tool execution error: " + execErr.Error(),
							IsError: true,
						}
					}
				}

				// Emit tool_result event
				ch <- StreamEvent{
					Type:      "tool_result",
					ToolID:    call.ID,
					ToolName:  call.Name,
					Result:    toolResult.Content,
					IsError:   toolResult.IsError,
					TurnIndex: turn,
				}

				// Append tool result message
				messages = append(messages, agent.Message{
					Role:       "tool",
					Content:    toolResult.Content,
					ToolCallID: call.ID,
				})
			}

			// Emit turn complete
			ch <- StreamEvent{Type: "turn_complete", TurnIndex: turn}
		}

		// MaxTurns reached
		ch <- StreamEvent{Type: "text_delta", Text: "\n\n[Reached maximum turns limit]"}
		ch <- StreamEvent{Type: "message_stop", Model: lastModel}
	}()

	return ch, runID
}

// Abort cancels a running agent request.
func (rt *Runtime) Abort(runID string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if cancel, ok := rt.cancelFuncs[runID]; ok {
		cancel()
		delete(rt.cancelFuncs, runID)
		return true
	}
	return false
}

// Close cleans up runtime resources.
func (rt *Runtime) Close() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for id, cancel := range rt.cancelFuncs {
		cancel()
		delete(rt.cancelFuncs, id)
	}
}

// Skills returns the loaded skills.
func (rt *Runtime) Skills() []skills.Entry {
	return rt.skills
}

// Tools returns the configured tools.
func (rt *Runtime) Tools() []tools.Tool {
	return rt.tools
}

func (rt *Runtime) buildSystemPrompt() string {
	if rt.opts.SystemPromptOverride != "" {
		return rt.opts.SystemPromptOverride
	}

	prompt := "You are a helpful AI assistant powered by SiriuSec Claw. You have access to tools for file operations, code editing, and system tasks. Use them when needed to help the user."

	// Append skill descriptions
	for _, s := range rt.skills {
		if len(s.EmbeddedContent) > 0 {
			prompt += "\n\n## Skill: " + s.Name + "\n" + string(s.EmbeddedContent)
		}
	}

	return prompt
}

func (rt *Runtime) loadHistory(sessionID string) []agent.Message {
	env := os.Getenv
	transcriptPath := session.ResolveSessionFilePath(sessionID, nil, env)
	tMessages, err := session.ReadTranscriptMessages(transcriptPath, 50)
	if err != nil {
		return nil
	}

	var history []agent.Message
	for _, msg := range tMessages {
		switch msg.Role {
		case "user":
			text := extractText(msg.Content)
			if text != "" {
				history = append(history, agent.Message{Role: "user", Content: text})
			}
		case "assistant":
			text := extractText(msg.Content)
			// Reconstruct tool calls if present
			var toolCalls []agent.ToolCallInfo
			for _, block := range msg.Content {
				if block.Type == "tool_use" && block.ID != "" {
					toolCalls = append(toolCalls, agent.ToolCallInfo{
						ID:    block.ID,
						Name:  block.Name,
						Input: block.Input,
					})
				}
			}
			if text != "" || len(toolCalls) > 0 {
				m := agent.Message{Role: "assistant", Content: text, ToolCalls: toolCalls}
				history = append(history, m)
			}
		case "toolResult":
			text := extractText(msg.Content)
			if msg.ToolCallID != "" {
				history = append(history, agent.Message{
					Role:       "tool",
					Content:    text,
					ToolCallID: msg.ToolCallID,
				})
			}
		}
	}
	return history
}

func extractText(blocks []session.ContentBlock) string {
	var text string
	for _, b := range blocks {
		if b.Type == "text" {
			text += b.Text
		}
	}
	return text
}

// toStringMap converts an interface{} to map[string]interface{}.
func toStringMap(v interface{}) map[string]interface{} {
	if v == nil {
		return make(map[string]interface{})
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	// Try JSON round-trip for other types
	data, err := json.Marshal(v)
	if err != nil {
		return make(map[string]interface{})
	}
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		return make(map[string]interface{})
	}
	return m
}

func resolveModelFromConfig(cfg *config.ClawConfig, agentID string) (string, string) {
	if cfg != nil && cfg.Agents != nil {
		// Check agent-specific model
		if cfg.Agents.List != nil {
			for _, a := range cfg.Agents.List {
				if a.ID == agentID && a.Model != nil {
					if ref := extractModelRef(a.Model); ref != "" {
						return agent.ResolveModelRef(ref)
					}
				}
			}
		}
		// Check defaults
		if cfg.Agents.Defaults != nil && cfg.Agents.Defaults.Model != nil {
			if cfg.Agents.Defaults.Model.Primary != nil && *cfg.Agents.Defaults.Model.Primary != "" {
				return agent.ResolveModelRef(*cfg.Agents.Defaults.Model.Primary)
			}
		}
	}
	return agent.ResolveModelRef("")
}

// extractModelRef extracts the primary model reference from a model config (interface{}).
func extractModelRef(model interface{}) string {
	switch v := model.(type) {
	case string:
		return v
	case map[string]interface{}:
		if primary, ok := v["primary"]; ok {
			if s, ok := primary.(string); ok {
				return s
			}
		}
	}
	return ""
}
