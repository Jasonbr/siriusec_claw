package session

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"sort"
	"time"
)

// CostUsageTotals is the aggregate usage total.
type CostUsageTotals struct {
	Input       int     `json:"input"`
	Output      int     `json:"output"`
	CacheRead   int     `json:"cacheRead"`
	CacheWrite  int     `json:"cacheWrite"`
	TotalTokens int     `json:"totalTokens"`
	TotalCost   float64 `json:"totalCost"`
	InputCost   float64 `json:"inputCost"`
	OutputCost  float64 `json:"outputCost"`
}

// SessionCostSummary is the full usage summary for a single session.
type SessionCostSummary struct {
	Input              int                   `json:"input"`
	Output             int                   `json:"output"`
	CacheRead          int                   `json:"cacheRead"`
	CacheWrite         int                   `json:"cacheWrite"`
	TotalTokens        int                   `json:"totalTokens"`
	TotalCost          float64               `json:"totalCost"`
	InputCost          float64               `json:"inputCost"`
	OutputCost         float64               `json:"outputCost"`
	MissingCostEntries int                   `json:"missingCostEntries"`
	SessionID          string                `json:"sessionId"`
	SessionFile        string                `json:"sessionFile"`
	FirstActivity      *int64                `json:"firstActivity,omitempty"`
	LastActivity       *int64                `json:"lastActivity,omitempty"`
	DurationMs         *int64                `json:"durationMs,omitempty"`
	ActivityDates      []string              `json:"activityDates,omitempty"`
	DailyBreakdown     []SessionDailyUsage   `json:"dailyBreakdown,omitempty"`
	MessageCounts      *SessionMessageCounts `json:"messageCounts,omitempty"`
	ToolUsage          *SessionToolUsage     `json:"toolUsage,omitempty"`
	ModelUsage         []SessionModelUsage   `json:"modelUsage,omitempty"`
	Latency            *SessionLatencyStats  `json:"latency,omitempty"`
}

// SessionDailyUsage is per-day usage statistics.
type SessionDailyUsage struct {
	Date        string  `json:"date"` // "YYYY-MM-DD"
	Input       int     `json:"input"`
	Output      int     `json:"output"`
	TotalTokens int     `json:"totalTokens"`
	TotalCost   float64 `json:"totalCost"`
}

// SessionMessageCounts tracks message type counts.
type SessionMessageCounts struct {
	Total       int `json:"total"`
	User        int `json:"user"`
	Assistant   int `json:"assistant"`
	ToolCalls   int `json:"toolCalls"`
	ToolResults int `json:"toolResults"`
	Errors      int `json:"errors"`
}

// SessionToolUsage tracks tool usage statistics.
type SessionToolUsage struct {
	TotalCalls  int                `json:"totalCalls"`
	UniqueTools int                `json:"uniqueTools"`
	Tools       []SessionToolCount `json:"tools"`
}

// SessionToolCount is a tool name and its call count.
type SessionToolCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// SessionModelUsage tracks per-model usage.
type SessionModelUsage struct {
	Provider string          `json:"provider"`
	Model    string          `json:"model"`
	Count    int             `json:"count"`
	Totals   CostUsageTotals `json:"totals"`
}

// SessionLatencyStats tracks response latency statistics.
type SessionLatencyStats struct {
	Count int     `json:"count"`
	AvgMs float64 `json:"avgMs"`
	MinMs float64 `json:"minMs"`
	MaxMs float64 `json:"maxMs"`
	P95Ms float64 `json:"p95Ms"`
}

// UsageTimePoint is a single data point in a usage time series.
type UsageTimePoint struct {
	Timestamp        int64   `json:"timestamp"`
	Input            int     `json:"input"`
	Output           int     `json:"output"`
	CacheRead        int     `json:"cacheRead"`
	CacheWrite       int     `json:"cacheWrite"`
	TotalTokens      int     `json:"totalTokens"`
	Cost             float64 `json:"cost"`
	CumulativeTokens int     `json:"cumulativeTokens"`
	CumulativeCost   float64 `json:"cumulativeCost"`
}

// LogEntry is a message log entry for usage logs.
type LogEntry struct {
	Timestamp int64    `json:"timestamp"`
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Tokens    *int     `json:"tokens,omitempty"`
	Cost      *float64 `json:"cost,omitempty"`
}

// LoadSessionCostSummary parses a transcript and computes usage statistics.
func LoadSessionCostSummary(sessionFile string, startMs, endMs int64) (*SessionCostSummary, error) {
	f, err := os.Open(sessionFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	summary := &SessionCostSummary{
		SessionFile: sessionFile,
	}
	msgCounts := &SessionMessageCounts{}
	toolMap := make(map[string]int)
	modelMap := make(map[string]*SessionModelUsage)
	dailyMap := make(map[string]*SessionDailyUsage)
	dateSet := make(map[string]bool)
	var latencies []float64

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Skip header
		if lineNum == 1 {
			var peek map[string]interface{}
			if json.Unmarshal(line, &peek) == nil {
				if t, _ := peek["type"].(string); t == "session" {
					continue
				}
			}
		}

		var raw map[string]interface{}
		if json.Unmarshal(line, &raw) != nil {
			continue
		}

		// Handle token_usage lines
		if t, _ := raw["type"].(string); t == "token_usage" {
			ts := int64(0)
			if tsStr, _ := raw["timestamp"].(string); tsStr != "" {
				if parsed, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
					ts = parsed.UnixMilli()
				}
			}
			if startMs > 0 && ts > 0 && ts < startMs {
				continue
			}
			if endMs > 0 && ts > 0 && ts > endMs {
				continue
			}
			inp := intFromJSON(raw["input"])
			out := intFromJSON(raw["output"])
			cr := intFromJSON(raw["cacheRead"])
			cw := intFromJSON(raw["cacheWrite"])
			total := intFromJSON(raw["totalTokens"])
			summary.Input += inp
			summary.Output += out
			summary.CacheRead += cr
			summary.CacheWrite += cw
			summary.TotalTokens += total
			continue
		}

		// Try to extract message
		var msg TranscriptMessage
		if t, _ := raw["type"].(string); t == "message" {
			// Wrapped format
			var wrapped WrappedMessage
			if json.Unmarshal(line, &wrapped) == nil && wrapped.Message != nil {
				msg = *wrapped.Message
			} else {
				continue
			}
		} else if role, _ := raw["role"].(string); role != "" {
			if json.Unmarshal(line, &msg) != nil {
				continue
			}
		} else {
			continue
		}

		ts := msg.Timestamp
		if startMs > 0 && ts > 0 && ts < startMs {
			continue
		}
		if endMs > 0 && ts > 0 && ts > endMs {
			continue
		}

		// Track first/last activity
		if ts > 0 {
			if summary.FirstActivity == nil || ts < *summary.FirstActivity {
				t := ts
				summary.FirstActivity = &t
			}
			if summary.LastActivity == nil || ts > *summary.LastActivity {
				t := ts
				summary.LastActivity = &t
			}
			day := FormatDayKey(ts)
			dateSet[day] = true
		}

		// Count messages
		msgCounts.Total++
		switch msg.Role {
		case "user":
			msgCounts.User++
		case "assistant":
			msgCounts.Assistant++
			if msg.DurationMs != nil {
				latencies = append(latencies, float64(*msg.DurationMs))
			}
		case "toolResult":
			msgCounts.ToolResults++
			if msg.IsError {
				msgCounts.Errors++
			}
		}

		// Count tool calls in content blocks
		for _, block := range msg.Content {
			if block.Type == "tool_use" && block.Name != "" {
				msgCounts.ToolCalls++
				toolMap[block.Name]++
			}
		}

		// Track usage per message
		if msg.Usage != nil {
			summary.Input += msg.Usage.Input
			summary.Output += msg.Usage.Output
			summary.CacheRead += msg.Usage.CacheRead
			summary.CacheWrite += msg.Usage.CacheWrite
			summary.TotalTokens += msg.Usage.TotalTokens
			if msg.Usage.Cost != nil {
				summary.TotalCost += msg.Usage.Cost.Total
				summary.InputCost += msg.Usage.Cost.Input
				summary.OutputCost += msg.Usage.Cost.Output
			} else {
				summary.MissingCostEntries++
			}

			// Daily breakdown
			if ts > 0 {
				day := FormatDayKey(ts)
				d, ok := dailyMap[day]
				if !ok {
					d = &SessionDailyUsage{Date: day}
					dailyMap[day] = d
				}
				d.Input += msg.Usage.Input
				d.Output += msg.Usage.Output
				d.TotalTokens += msg.Usage.TotalTokens
				if msg.Usage.Cost != nil {
					d.TotalCost += msg.Usage.Cost.Total
				}
			}
		}

		// Track model usage
		if msg.Provider != "" || msg.Model != "" {
			key := msg.Provider + ":" + msg.Model
			m, ok := modelMap[key]
			if !ok {
				m = &SessionModelUsage{Provider: msg.Provider, Model: msg.Model}
				modelMap[key] = m
			}
			m.Count++
			if msg.Usage != nil {
				m.Totals.Input += msg.Usage.Input
				m.Totals.Output += msg.Usage.Output
				m.Totals.TotalTokens += msg.Usage.TotalTokens
				if msg.Usage.Cost != nil {
					m.Totals.TotalCost += msg.Usage.Cost.Total
				}
			}
		}
	}

	// Compute duration
	if summary.FirstActivity != nil && summary.LastActivity != nil {
		d := *summary.LastActivity - *summary.FirstActivity
		summary.DurationMs = &d
	}

	// Build activity dates
	for d := range dateSet {
		summary.ActivityDates = append(summary.ActivityDates, d)
	}
	sort.Strings(summary.ActivityDates)

	// Build daily breakdown
	for _, d := range dailyMap {
		summary.DailyBreakdown = append(summary.DailyBreakdown, *d)
	}
	sort.Slice(summary.DailyBreakdown, func(i, j int) bool {
		return summary.DailyBreakdown[i].Date < summary.DailyBreakdown[j].Date
	})

	summary.MessageCounts = msgCounts

	// Build tool usage
	if len(toolMap) > 0 {
		tools := make([]SessionToolCount, 0, len(toolMap))
		totalCalls := 0
		for name, count := range toolMap {
			tools = append(tools, SessionToolCount{Name: name, Count: count})
			totalCalls += count
		}
		sort.Slice(tools, func(i, j int) bool { return tools[i].Count > tools[j].Count })
		summary.ToolUsage = &SessionToolUsage{
			TotalCalls:  totalCalls,
			UniqueTools: len(tools),
			Tools:       tools,
		}
	}

	// Build model usage
	for _, m := range modelMap {
		summary.ModelUsage = append(summary.ModelUsage, *m)
	}

	// Build latency stats
	if len(latencies) > 0 {
		sort.Float64s(latencies)
		sum := 0.0
		for _, v := range latencies {
			sum += v
		}
		p95Idx := int(math.Ceil(float64(len(latencies))*0.95)) - 1
		if p95Idx < 0 {
			p95Idx = 0
		}
		summary.Latency = &SessionLatencyStats{
			Count: len(latencies),
			AvgMs: sum / float64(len(latencies)),
			MinMs: latencies[0],
			MaxMs: latencies[len(latencies)-1],
			P95Ms: latencies[p95Idx],
		}
	}

	return summary, scanner.Err()
}

// LoadSessionUsageTimeSeries returns time-series data points for a session.
func LoadSessionUsageTimeSeries(sessionFile string, maxPoints int) ([]UsageTimePoint, error) {
	if maxPoints <= 0 {
		maxPoints = 200
	}
	messages, err := ReadTranscriptMessages(sessionFile, 0)
	if err != nil {
		return nil, err
	}

	var points []UsageTimePoint
	cumTokens := 0
	cumCost := 0.0

	for _, msg := range messages {
		if msg.Usage == nil {
			continue
		}
		cumTokens += msg.Usage.TotalTokens
		cost := 0.0
		if msg.Usage.Cost != nil {
			cost = msg.Usage.Cost.Total
		}
		cumCost += cost
		points = append(points, UsageTimePoint{
			Timestamp:        msg.Timestamp,
			Input:            msg.Usage.Input,
			Output:           msg.Usage.Output,
			CacheRead:        msg.Usage.CacheRead,
			CacheWrite:       msg.Usage.CacheWrite,
			TotalTokens:      msg.Usage.TotalTokens,
			Cost:             cost,
			CumulativeTokens: cumTokens,
			CumulativeCost:   cumCost,
		})
	}

	// Downsample if needed
	if len(points) > maxPoints {
		step := float64(len(points)) / float64(maxPoints)
		sampled := make([]UsageTimePoint, 0, maxPoints)
		for i := 0; i < maxPoints; i++ {
			idx := int(float64(i) * step)
			if idx >= len(points) {
				idx = len(points) - 1
			}
			sampled = append(sampled, points[idx])
		}
		points = sampled
	}

	return points, nil
}

// LoadSessionLogs returns message logs for a session.
func LoadSessionLogs(sessionFile string, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	messages, err := ReadTranscriptMessages(sessionFile, limit)
	if err != nil {
		return nil, err
	}

	logs := make([]LogEntry, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		entry := LogEntry{
			Timestamp: msg.Timestamp,
			Role:      msg.Role,
			Content:   extractTextFromContent(msg.Content),
		}
		if msg.Usage != nil {
			t := msg.Usage.TotalTokens
			entry.Tokens = &t
			if msg.Usage.Cost != nil {
				c := msg.Usage.Cost.Total
				entry.Cost = &c
			}
		}
		logs = append(logs, entry)
	}
	return logs, nil
}

// FormatDayKey formats a millisecond timestamp to "YYYY-MM-DD".
func FormatDayKey(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("2006-01-02")
}

func intFromJSON(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
