package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/agent"
	"github.com/siriusec/siriusec_claw/pkg/agent/runtime"
	"github.com/siriusec/siriusec_claw/pkg/agent/tools"
	"github.com/siriusec/siriusec_claw/pkg/security"
	"github.com/siriusec/siriusec_claw/pkg/session"
)

// ActiveRun tracks a running chat invocation.
type ActiveRun struct {
	RunID      string
	SessionKey string
	StartedAt  time.Time
	Cancel     func()
}

var (
	activeRuns   = make(map[string]*ActiveRun) // sessionKey → ActiveRun
	activeRunsMu sync.Mutex
)

// --- chat.send ---

func ChatSendHandler(opts HandlerOpts) error {
	env := envGetter()
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	message := stringParam(opts.Params, "message", "")

	slog.Debug("chat.send called", "sessionKey", sessionKey, "messageLen", len(message))

	if sessionKey == "" {
		opts.Respond(false, nil, errInvalidParams("sessionKey required"), nil)
		return nil
	}
	if message == "" {
		opts.Respond(false, nil, errInvalidParams("message required"), nil)
		return nil
	}

	agentID := session.ResolveSessionAgentID(sessionKey)

	// Ensure session exists
	entry, _, err := session.EnsureSession(agentID, sessionKey, env, "")
	if err != nil {
		slog.Debug("EnsureSession failed", "error", err)
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	transcriptPath := session.ResolveSessionFilePath(entry.SessionID, &agentID, env)
	slog.Debug("session resolved", "agentID", agentID, "sessionID", entry.SessionID)

	// Handle special commands
	switch {
	case message == "/new" || message == "!new" || message == "/reset" || message == "!reset":
		newEntry, err := session.ResetSession(agentID, sessionKey, env)
		if err != nil {
			opts.Respond(false, nil, errInternal(err.Error()), nil)
			return nil
		}
		if opts.Context != nil && opts.Context.Broadcast != nil {
			opts.Context.Broadcast("chat", map[string]interface{}{
				"type":       "session.reset",
				"sessionKey": sessionKey,
				"sessionId":  newEntry.SessionID,
			}, &BroadcastOptions{DropIfSlow: true})
		}
		opts.Respond(true, map[string]interface{}{
			"ok":     true,
			"action": "reset",
			"key":    sessionKey,
		}, nil, nil)
		return nil

	case message == "/stop" || message == "!stop":
		activeRunsMu.Lock()
		run, ok := activeRuns[sessionKey]
		if ok && run.Cancel != nil {
			run.Cancel()
			delete(activeRuns, sessionKey)
		}
		activeRunsMu.Unlock()
		opts.Respond(true, map[string]interface{}{
			"ok":      true,
			"action":  "abort",
			"aborted": ok,
		}, nil, nil)
		return nil
	}

	// Append user message to transcript
	if err := session.AppendUserMessage(transcriptPath, message); err != nil {
		opts.Respond(false, nil, errInternal("failed to write transcript: "+err.Error()), nil)
		return nil
	}

	// Generate run ID
	runID := stringParam(opts.Params, "idempotencyKey", "")
	if runID == "" {
		runID = uuid.New().String()
	}

	// Broadcast user message event
	if opts.Context != nil && opts.Context.Broadcast != nil {
		opts.Context.Broadcast("chat", map[string]interface{}{
			"type":       "message.user",
			"sessionKey": sessionKey,
			"runId":      runID,
			"text":       message,
		}, &BroadcastOptions{DropIfSlow: true})
	}

	// Launch agent runtime to process the message
	go func() {
		slog.Debug("starting agent runtime", "agentID", agentID, "runID", runID)
		factory := agent.CreateModelFactoryFromConfig(opts.Context.Config)
		if factory == nil {
			slog.Error("CreateModelFactoryFromConfig returned nil")
			reply := "Failed to create model factory: check configuration"
			_ = session.AppendAssistantMessage(transcriptPath, reply)
			broadcastAssistant(opts, sessionKey, runID, reply, "", 0)
			activeRunsMu.Lock()
			delete(activeRuns, sessionKey)
			activeRunsMu.Unlock()
			return
		}
		slog.Debug("model factory created")

		// Build security policy
		var secPolicy *security.CommandPolicy
		if opts.Context.Config != nil && opts.Context.Config.Security != nil &&
			opts.Context.Config.Security.CommandPolicy != nil {
			cp := opts.Context.Config.Security.CommandPolicy
			maxLen := 0
			if cp.MaxLength != nil {
				maxLen = *cp.MaxLength
			}
			secPolicy = &security.CommandPolicy{
				Enabled:        cp.Enabled != nil && *cp.Enabled,
				DefaultPolicy:  "allow",
				Deny:           cp.Deny,
				Ask:            cp.Ask,
				Allow:          cp.Allow,
				BanArguments:   cp.BanArguments,
				MaxLength:      maxLen,
				SecretPatterns: cp.SecretPatterns,
			}
			if cp.DefaultPolicy != nil {
				secPolicy.DefaultPolicy = *cp.DefaultPolicy
			}
		}

		rt, rtErr := runtime.New(runtime.Options{
			ModelFactory:   factory,
			Config:         opts.Context.Config,
			AgentID:        agentID,
			TimeoutSeconds: 120,
			MaxTurns:       25,
			EnableSkills:   true,
			ExecContext: tools.ExecutionContext{
				SecurityPolicy: secPolicy,
			},
		})
		if rtErr != nil {
			slog.Error("runtime.New failed", "error", rtErr)
			reply := "Failed to create runtime: " + rtErr.Error()
			_ = session.AppendAssistantMessage(transcriptPath, reply)
			broadcastAssistant(opts, sessionKey, runID, reply, "", 0)
			return
		}
		slog.Debug("runtime created successfully")

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		activeRunsMu.Lock()
		activeRuns[sessionKey] = &ActiveRun{RunID: runID, SessionKey: sessionKey, StartedAt: time.Now(), Cancel: cancel}
		activeRunsMu.Unlock()

		slog.Debug("calling RunStream", "sessionID", entry.SessionID, "messageLen", len(message))
		events, _ := rt.RunStream(ctx, runtime.Request{
			Message: message, SessionID: entry.SessionID, SessionKey: sessionKey, RunID: runID,
		})

		var finalText string
		var finalModel string
		var totalDurationMs int64
		startTime := time.Now()

		// Consume stream events and broadcast to WebSocket clients
		eventCount := 0
		for ev := range events {
			eventCount++
			slog.Debug("received event", "type", ev.Type, "turnIndex", ev.TurnIndex)
			payload := map[string]interface{}{
				"sessionKey": sessionKey,
				"runId":      runID,
			}
			switch ev.Type {
			case "text_delta":
				payload["type"] = "stream.text_delta"
				payload["text"] = ev.Text
				payload["turnIndex"] = ev.TurnIndex
				finalText += ev.Text
			case "tool_use":
				payload["type"] = "stream.tool_use"
				payload["toolId"] = ev.ToolID
				payload["toolName"] = ev.ToolName
				payload["input"] = ev.Input
				payload["turnIndex"] = ev.TurnIndex
				// Also persist tool use to transcript
				persistToolUse(transcriptPath, ev)
			case "tool_result":
				payload["type"] = "stream.tool_result"
				payload["toolId"] = ev.ToolID
				payload["toolName"] = ev.ToolName
				payload["result"] = truncateForBroadcast(ev.Result, 2000)
				payload["isError"] = ev.IsError
				payload["turnIndex"] = ev.TurnIndex
				// Persist tool result to transcript
				persistToolResult(transcriptPath, ev)
			case "turn_complete":
				payload["type"] = "stream.turn_complete"
				payload["turnIndex"] = ev.TurnIndex
			case "message_stop":
				finalModel = ev.Model
				totalDurationMs = time.Since(startTime).Milliseconds()
				payload["type"] = "stream.message_stop"
				payload["model"] = finalModel
				payload["durationMs"] = totalDurationMs
				payload["text"] = finalText
			case "error":
				payload["type"] = "stream.error"
				payload["text"] = ev.Text
				payload["isError"] = true
				if finalText == "" {
					finalText = "Error: " + ev.Text
				}
			default:
				continue
			}

			if opts.Context != nil && opts.Context.Broadcast != nil {
				opts.Context.Broadcast("chat", payload, &BroadcastOptions{DropIfSlow: true})
			}
		}

		cancel()
		activeRunsMu.Lock()
		delete(activeRuns, sessionKey)
		activeRunsMu.Unlock()

		if finalText == "" {
			finalText = "(empty response)"
		}

		slog.Debug("run completed", "eventCount", eventCount, "finalTextLen", len(finalText), "durationMs", time.Since(startTime).Milliseconds())

		// Persist final assistant message
		_ = session.AppendAssistantMessage(transcriptPath, finalText)
		_ = session.UpdateSessionUpdatedAt(agentID, sessionKey, env, 0)

		// Note: message is already broadcast via stream.message_stop event
	}()

	opts.Respond(true, map[string]interface{}{
		"ok":     true,
		"runId":  runID,
		"status": "started",
	}, nil, nil)
	return nil
}

// --- chat.history ---

func ChatHistoryHandler(opts HandlerOpts) error {
	env := envGetter()
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	sessionID := stringParam(opts.Params, "sessionId", "")
	limit := intParam(opts.Params, "limit", 100)

	if limit > 1000 {
		limit = 1000
	}

	agentID := session.ResolveSessionAgentID(sessionKey)

	// Resolve session ID
	if sessionID == "" && sessionKey != "" {
		storePath := session.ResolveDefaultSessionStorePath(agentID, env)
		store, _ := session.LoadSessionStore(storePath)
		if entry, ok := store[sessionKey]; ok {
			sessionID = entry.SessionID
		}
	}

	if sessionID == "" {
		opts.Respond(true, map[string]interface{}{
			"sessionKey": sessionKey,
			"messages":   []interface{}{},
		}, nil, nil)
		return nil
	}

	transcriptPath := session.ResolveSessionFilePath(sessionID, &agentID, env)
	messages, err := session.ReadTranscriptMessages(transcriptPath, limit)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(true, map[string]interface{}{
				"sessionKey": sessionKey,
				"sessionId":  sessionID,
				"messages":   []interface{}{},
			}, nil, nil)
			return nil
		}
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	// Get thinking/verbose level from session entry
	storePath := session.ResolveDefaultSessionStorePath(agentID, env)
	store, _ := session.LoadSessionStore(storePath)
	thinkingLevel := ""
	verboseLevel := ""
	if entry, ok := store[sessionKey]; ok {
		thinkingLevel = entry.ThinkingLevel
		verboseLevel = entry.VerboseLevel
	}

	opts.Respond(true, map[string]interface{}{
		"sessionKey":    sessionKey,
		"sessionId":     sessionID,
		"messages":      messages,
		"thinkingLevel": thinkingLevel,
		"verboseLevel":  verboseLevel,
	}, nil, nil)
	return nil
}

// --- chat.abort ---

func ChatAbortHandler(opts HandlerOpts) error {
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	runID := stringParam(opts.Params, "runId", "")

	activeRunsMu.Lock()
	defer activeRunsMu.Unlock()

	if runID != "" {
		// Abort specific run
		for k, run := range activeRuns {
			if run.RunID == runID {
				if run.Cancel != nil {
					run.Cancel()
				}
				delete(activeRuns, k)
				opts.Respond(true, map[string]interface{}{"ok": true, "aborted": true}, nil, nil)
				return nil
			}
		}
	} else if sessionKey != "" {
		run, ok := activeRuns[sessionKey]
		if ok {
			if run.Cancel != nil {
				run.Cancel()
			}
			delete(activeRuns, sessionKey)
			opts.Respond(true, map[string]interface{}{"ok": true, "aborted": true}, nil, nil)
			return nil
		}
	}

	opts.Respond(true, map[string]interface{}{"ok": true, "aborted": false}, nil, nil)
	return nil
}

// --- chat.inject ---

func ChatInjectHandler(opts HandlerOpts) error {
	env := envGetter()
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	message := stringParam(opts.Params, "message", "")
	label := stringParam(opts.Params, "label", "")

	if sessionKey == "" || message == "" {
		opts.Respond(false, nil, errInvalidParams("sessionKey and message required"), nil)
		return nil
	}

	agentID := session.ResolveSessionAgentID(sessionKey)
	storePath := session.ResolveDefaultSessionStorePath(agentID, env)
	store, _ := session.LoadSessionStore(storePath)
	entry, ok := store[sessionKey]
	if !ok {
		opts.Respond(false, nil, errInvalidParams("session not found"), nil)
		return nil
	}

	text := message
	if label != "" {
		text = label + " " + message
	}

	transcriptPath := session.ResolveSessionFilePath(entry.SessionID, &agentID, env)
	if err := session.AppendAssistantMessage(transcriptPath, text); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	_ = session.UpdateSessionUpdatedAt(agentID, sessionKey, env, 0)

	if opts.Context != nil && opts.Context.Broadcast != nil {
		opts.Context.Broadcast("chat", map[string]interface{}{
			"type":       "message.injected",
			"sessionKey": sessionKey,
			"text":       text,
		}, &BroadcastOptions{DropIfSlow: true})
	}

	opts.Respond(true, map[string]interface{}{"ok": true}, nil, nil)
	return nil
}

func broadcastAssistant(opts HandlerOpts, sessionKey, runID, text, model string, durationMs int64) {
	if opts.Context != nil && opts.Context.Broadcast != nil {
		opts.Context.Broadcast("chat", map[string]interface{}{
			"type":       "message.assistant",
			"sessionKey": sessionKey,
			"runId":      runID,
			"text":       text,
			"model":      model,
			"durationMs": durationMs,
			"final":      true,
		}, &BroadcastOptions{DropIfSlow: true})
	}
}

func persistToolUse(transcriptPath string, ev runtime.StreamEvent) {
	inputJSON, _ := json.Marshal(ev.Input)
	msg := session.TranscriptMessage{
		Role:      "assistant",
		Timestamp: time.Now().UnixMilli(),
		Content: []session.ContentBlock{
			{Type: "tool_use", ID: ev.ToolID, Name: ev.ToolName, Input: json.RawMessage(inputJSON)},
		},
	}
	session.AppendTranscriptMessage(transcriptPath, &msg)
}

func persistToolResult(transcriptPath string, ev runtime.StreamEvent) {
	msg := session.TranscriptMessage{
		Role:       "toolResult",
		Timestamp:  time.Now().UnixMilli(),
		ToolCallID: ev.ToolID,
		ToolName:   ev.ToolName,
		IsError:    ev.IsError,
		Content: []session.ContentBlock{
			{Type: "text", Text: ev.Result},
		},
	}
	session.AppendTranscriptMessage(transcriptPath, &msg)
}

func truncateForBroadcast(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
