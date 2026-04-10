package handlers

import (
	"os"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/session"
)

// --- sessions.usage ---

func SessionsUsageHandler(opts HandlerOpts) error {
	env := envGetter()
	agentID := stringParam(opts.Params, "agentId", "main")
	startMs := int64(intParam(opts.Params, "startMs", 0))
	endMs := int64(intParam(opts.Params, "endMs", 0))

	storePath := session.ResolveDefaultSessionStorePath(agentID, env)
	store, err := session.LoadSessionStore(storePath)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	type usageEntry struct {
		Key       string                      `json:"key"`
		Label     string                      `json:"label"`
		SessionID string                      `json:"sessionId"`
		UpdatedAt int64                       `json:"updatedAt"`
		Channel   string                      `json:"channel"`
		ChatType  string                      `json:"chatType"`
		Usage     *session.SessionCostSummary `json:"usage,omitempty"`
	}

	var entries []usageEntry
	totalInput, totalOutput, totalTokens := 0, 0, 0
	totalCost := 0.0

	for key, entry := range store {
		if entry.SessionID == "" {
			continue
		}
		transcriptPath := session.ResolveSessionFilePath(entry.SessionID, &agentID, env)
		if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
			continue
		}
		summary, err := session.LoadSessionCostSummary(transcriptPath, startMs, endMs)
		if err != nil {
			continue
		}
		summary.SessionID = entry.SessionID

		totalInput += summary.Input
		totalOutput += summary.Output
		totalTokens += summary.TotalTokens
		totalCost += summary.TotalCost

		entries = append(entries, usageEntry{
			Key:       key,
			Label:     entry.Label,
			SessionID: entry.SessionID,
			UpdatedAt: entry.UpdatedAt,
			Channel:   entry.Channel,
			ChatType:  entry.ChatType,
			Usage:     summary,
		})
	}

	opts.Respond(true, map[string]interface{}{
		"updatedAt": time.Now().UnixMilli(),
		"sessions":  entries,
		"totals": map[string]interface{}{
			"input":       totalInput,
			"output":      totalOutput,
			"totalTokens": totalTokens,
			"totalCost":   totalCost,
		},
	}, nil, nil)
	return nil
}

// --- sessions.usage.timeseries ---

func SessionsUsageTimeseriesHandler(opts HandlerOpts) error {
	env := envGetter()
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	sessionID := stringParam(opts.Params, "sessionId", "")
	maxPoints := intParam(opts.Params, "maxPoints", 200)

	agentID := session.ResolveSessionAgentID(sessionKey)

	if sessionID == "" && sessionKey != "" {
		storePath := session.ResolveDefaultSessionStorePath(agentID, env)
		store, _ := session.LoadSessionStore(storePath)
		if entry, ok := store[sessionKey]; ok {
			sessionID = entry.SessionID
		}
	}

	if sessionID == "" {
		opts.Respond(true, map[string]interface{}{
			"sessionId": "",
			"points":    []interface{}{},
		}, nil, nil)
		return nil
	}

	transcriptPath := session.ResolveSessionFilePath(sessionID, &agentID, env)
	points, err := session.LoadSessionUsageTimeSeries(transcriptPath, maxPoints)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(true, map[string]interface{}{
				"sessionId": sessionID,
				"points":    []interface{}{},
			}, nil, nil)
			return nil
		}
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"sessionId": sessionID,
		"points":    points,
	}, nil, nil)
	return nil
}

// --- sessions.usage.logs ---

func SessionsUsageLogsHandler(opts HandlerOpts) error {
	env := envGetter()
	sessionKey := stringParam(opts.Params, "sessionKey", "")
	sessionID := stringParam(opts.Params, "sessionId", "")
	limit := intParam(opts.Params, "limit", 200)

	agentID := session.ResolveSessionAgentID(sessionKey)

	if sessionID == "" && sessionKey != "" {
		storePath := session.ResolveDefaultSessionStorePath(agentID, env)
		store, _ := session.LoadSessionStore(storePath)
		if entry, ok := store[sessionKey]; ok {
			sessionID = entry.SessionID
		}
	}

	if sessionID == "" {
		opts.Respond(true, map[string]interface{}{
			"logs": []interface{}{},
		}, nil, nil)
		return nil
	}

	transcriptPath := session.ResolveSessionFilePath(sessionID, &agentID, env)
	logs, err := session.LoadSessionLogs(transcriptPath, limit)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(true, map[string]interface{}{
				"logs": []interface{}{},
			}, nil, nil)
			return nil
		}
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"logs": logs,
	}, nil, nil)
	return nil
}
