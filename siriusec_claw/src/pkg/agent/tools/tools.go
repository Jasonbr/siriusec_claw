package tools

import "context"

// Tool represents a callable tool available to the agent.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
	Handler     ToolHandler `json:"-"`
	Category    string      `json:"category,omitempty"` // "builtin", "mcp", "custom"
}

// ToolHandler is the function signature for tool execution.
type ToolHandler func(ctx context.Context, input map[string]interface{}) (*ToolResult, error)

// ToolResult is the output of a tool execution.
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError,omitempty"`
}

// BuiltinTools returns the standard set of built-in tools.
func BuiltinTools() []Tool {
	return []Tool{
		{
			Name:        "bash",
			Description: "Execute a bash command. Use for running scripts, installing packages, and system operations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The bash command to execute",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds (default 120)",
					},
				},
				"required": []string{"command"},
			},
			Category: "builtin",
		},
		{
			Name:        "read",
			Description: "Read the contents of a file from the filesystem.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the file to read",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Line number to start reading from",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of lines to read",
					},
				},
				"required": []string{"file_path"},
			},
			Category: "builtin",
		},
		{
			Name:        "write",
			Description: "Write content to a file, creating it if it doesn't exist.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to write",
					},
				},
				"required": []string{"file_path", "content"},
			},
			Category: "builtin",
		},
		{
			Name:        "edit",
			Description: "Edit a file by replacing a specific string with new content.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the file to edit",
					},
					"old_string": map[string]interface{}{
						"type":        "string",
						"description": "The exact string to replace",
					},
					"new_string": map[string]interface{}{
						"type":        "string",
						"description": "The replacement string",
					},
				},
				"required": []string{"file_path", "old_string", "new_string"},
			},
			Category: "builtin",
		},
		{
			Name:        "grep",
			Description: "Search for a pattern in files using regex.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The regex pattern to search for",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory or file path to search in",
					},
					"include": map[string]interface{}{
						"type":        "string",
						"description": "Glob pattern to filter files",
					},
				},
				"required": []string{"pattern"},
			},
			Category: "builtin",
		},
		{
			Name:        "glob",
			Description: "Find files matching a glob pattern.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The glob pattern to match",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Base directory to search from",
					},
				},
				"required": []string{"pattern"},
			},
			Category: "builtin",
		},
		{
			Name:        "fetch",
			Description: "Make HTTP requests to fetch data from URLs.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch",
					},
					"method": map[string]interface{}{
						"type":        "string",
						"description": "HTTP method (GET, POST, etc.)",
					},
					"headers": map[string]interface{}{
						"type":        "object",
						"description": "HTTP headers to send",
					},
				},
				"required": []string{"url"},
			},
			Category: "builtin",
		},
		{
			Name:        "calculator",
			Description: "Evaluate mathematical expressions.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "Mathematical expression to evaluate (e.g., '2+2', '100/5')",
					},
				},
				"required": []string{"expression"},
			},
			Category: "builtin",
		},
		{
			Name:        "datetime",
			Description: "Get current date and time in various formats.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Output format: RFC3339, Unix, Date, Time, DateTime, or custom format",
					},
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "Timezone (e.g., 'UTC', 'America/New_York')",
					},
				},
				"required": []string{},
			},
			Category: "builtin",
		},
	}
}
