package handlers

import (
	ctxmgr "github.com/siriusec/siriusec_claw/pkg/agent/context"
)

// --- context.list ---

func ContextListHandler(opts HandlerOpts) error {
	budget := ctxmgr.DefaultBudget(128000)
	opts.Respond(true, map[string]interface{}{
		"budget": budget,
		"components": []map[string]interface{}{
			{"name": "system_prompt", "maxTokens": budget.SystemPromptMax, "percentage": "20%"},
			{"name": "history", "maxTokens": budget.HistoryMax, "percentage": "60%"},
			{"name": "bootstrap", "maxTokens": budget.BootstrapMax, "percentage": "20%"},
			{"name": "completion", "maxTokens": budget.CompletionMax, "percentage": "reserved"},
		},
	}, nil, nil)
	return nil
}

// --- context.detail ---

func ContextDetailHandler(opts HandlerOpts) error {
	sessionID := stringParam(opts.Params, "sessionId", "")
	if sessionID == "" {
		opts.Respond(false, nil, errInvalidParams("sessionId required"), nil)
		return nil
	}

	budget := ctxmgr.DefaultBudget(128000)

	// Estimate current usage from session history
	var systemPromptTokens, historyTokens, bootstrapTokens int

	// System prompt estimate (rough)
	systemPromptTokens = 500 // base estimate

	// Load and estimate history tokens
	// Note: This is a simplified estimation. Full implementation would
	// track actual token counts during the agentic loop.
	historyTokens = 0 // would be calculated from actual messages

	usage := ctxmgr.Usage{
		SystemPromptTokens: systemPromptTokens,
		HistoryTokens:      historyTokens,
		BootstrapTokens:    bootstrapTokens,
		TotalTokens:        systemPromptTokens + historyTokens + bootstrapTokens,
	}

	overBudget := usage.OverBudgetComponent(budget)

	opts.Respond(true, map[string]interface{}{
		"budget":     budget,
		"usage":      usage,
		"inBudget":   usage.InBudget(budget),
		"overBudget": overBudget,
		"sessionId":  sessionID,
	}, nil, nil)
	return nil
}
