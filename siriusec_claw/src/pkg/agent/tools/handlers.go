package tools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/security"
)

const (
	maxOutputBytes   = 32 * 1024 // 32KB output limit for bash
	defaultReadLimit = 2000      // default line limit for read
	maxLineLen       = 2000      // truncate lines longer than this
	maxGrepMatches   = 100
	maxGlobResults   = 500
)

// ExecutionContext provides runtime context for tool handlers.
type ExecutionContext struct {
	ProjectRoot    string
	SecurityPolicy *security.CommandPolicy
}

// BuiltinToolsWithHandlers returns built-in tools with real handler implementations.
func BuiltinToolsWithHandlers(execCtx ExecutionContext) []Tool {
	all := BuiltinTools()
	handlers := map[string]ToolHandler{
		"bash":       makeBashHandler(execCtx),
		"read":       makeReadHandler(),
		"write":      makeWriteHandler(),
		"edit":       makeEditHandler(),
		"grep":       makeGrepHandler(execCtx),
		"glob":       makeGlobHandler(execCtx),
		"fetch":      makeFetchHandler(),
		"calculator": makeCalculatorHandler(),
		"datetime":   makeDatetimeHandler(),
	}
	for i := range all {
		if h, ok := handlers[all[i].Name]; ok {
			all[i].Handler = h
		}
	}
	return all
}

// --- bash ---

func makeBashHandler(execCtx ExecutionContext) ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		command, _ := input["command"].(string)
		if command == "" {
			return &ToolResult{Content: "Error: command is required", IsError: true}, nil
		}

		timeoutSec := 120
		if t, ok := input["timeout"].(float64); ok && t > 0 {
			timeoutSec = int(t)
		}

		// Security policy check
		if execCtx.SecurityPolicy != nil {
			eval := execCtx.SecurityPolicy.EvaluateCommand(command)
			if eval.Decision == "deny" {
				return &ToolResult{
					Content: fmt.Sprintf("Command denied by security policy: %s (rule: %s)", eval.Reason, eval.Rule),
					IsError: true,
				}, nil
			}
			if eval.Decision == "ask" {
				return &ToolResult{
					Content: fmt.Sprintf("Command requires approval: %s (rule: %s). Auto-approval not yet implemented.", eval.Reason, eval.Rule),
					IsError: true,
				}, nil
			}
		}

		cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()

		cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		err := cmd.Run()
		output := combined.String()

		// Truncate if too long
		if len(output) > maxOutputBytes {
			output = output[:maxOutputBytes] + "\n... (output truncated at 32KB)"
		}

		if err != nil {
			if cmdCtx.Err() == context.DeadlineExceeded {
				return &ToolResult{Content: output + "\nError: command timed out after " + fmt.Sprintf("%d", timeoutSec) + "s", IsError: true}, nil
			}
			return &ToolResult{Content: output + "\nExit error: " + err.Error(), IsError: true}, nil
		}
		if output == "" {
			output = "(no output)"
		}
		return &ToolResult{Content: output}, nil
	}
}

// --- read ---

func makeReadHandler() ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		filePath, _ := input["file_path"].(string)
		if filePath == "" {
			return &ToolResult{Content: "Error: file_path is required", IsError: true}, nil
		}
		if !filepath.IsAbs(filePath) {
			return &ToolResult{Content: "Error: file_path must be an absolute path", IsError: true}, nil
		}

		offset := 0
		if o, ok := input["offset"].(float64); ok && o > 0 {
			offset = int(o)
		}
		limit := defaultReadLimit
		if l, ok := input["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		f, err := os.Open(filePath)
		if err != nil {
			return &ToolResult{Content: "Error: " + err.Error(), IsError: true}, nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
		var lines []string
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if lineNum <= offset {
				continue
			}
			if len(lines) >= limit {
				break
			}
			line := scanner.Text()
			if len(line) > maxLineLen {
				line = line[:maxLineLen] + "...(truncated)"
			}
			lines = append(lines, fmt.Sprintf("%6d\t%s", lineNum, line))
		}
		if err := scanner.Err(); err != nil {
			return &ToolResult{Content: "Error reading file: " + err.Error(), IsError: true}, nil
		}
		if len(lines) == 0 {
			return &ToolResult{Content: "(empty file or offset past end)"}, nil
		}
		return &ToolResult{Content: strings.Join(lines, "\n")}, nil
	}
}

// --- write ---

func makeWriteHandler() ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		filePath, _ := input["file_path"].(string)
		content, _ := input["content"].(string)
		if filePath == "" {
			return &ToolResult{Content: "Error: file_path is required", IsError: true}, nil
		}
		if !filepath.IsAbs(filePath) {
			return &ToolResult{Content: "Error: file_path must be an absolute path", IsError: true}, nil
		}

		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return &ToolResult{Content: "Error creating directory: " + err.Error(), IsError: true}, nil
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return &ToolResult{Content: "Error writing file: " + err.Error(), IsError: true}, nil
		}
		return &ToolResult{Content: fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), filePath)}, nil
	}
}

// --- edit ---

func makeEditHandler() ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		filePath, _ := input["file_path"].(string)
		oldStr, _ := input["old_string"].(string)
		newStr, _ := input["new_string"].(string)
		if filePath == "" || oldStr == "" {
			return &ToolResult{Content: "Error: file_path and old_string are required", IsError: true}, nil
		}
		if !filepath.IsAbs(filePath) {
			return &ToolResult{Content: "Error: file_path must be an absolute path", IsError: true}, nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return &ToolResult{Content: "Error reading file: " + err.Error(), IsError: true}, nil
		}

		content := string(data)
		count := strings.Count(content, oldStr)
		if count == 0 {
			return &ToolResult{Content: "Error: old_string not found in file", IsError: true}, nil
		}
		if count > 1 {
			return &ToolResult{
				Content: fmt.Sprintf("Error: old_string found %d times. Provide more context to make it unique.", count),
				IsError: true,
			}, nil
		}

		newContent := strings.Replace(content, oldStr, newStr, 1)
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return &ToolResult{Content: "Error writing file: " + err.Error(), IsError: true}, nil
		}
		return &ToolResult{Content: "Successfully edited " + filePath}, nil
	}
}

// --- grep ---

func makeGrepHandler(execCtx ExecutionContext) ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		pattern, _ := input["pattern"].(string)
		if pattern == "" {
			return &ToolResult{Content: "Error: pattern is required", IsError: true}, nil
		}

		searchPath, _ := input["path"].(string)
		if searchPath == "" {
			searchPath = execCtx.ProjectRoot
		}
		if searchPath == "" {
			cwd, _ := os.Getwd()
			searchPath = cwd
		}

		includeGlob, _ := input["include"].(string)

		re, err := regexp.Compile(pattern)
		if err != nil {
			return &ToolResult{Content: "Error: invalid regex: " + err.Error(), IsError: true}, nil
		}

		var matches []string
		_ = filepath.WalkDir(searchPath, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				// Skip hidden dirs
				if d != nil && d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
					return filepath.SkipDir
				}
				return nil
			}
			if len(matches) >= maxGrepMatches {
				return filepath.SkipAll
			}
			// Apply include filter
			if includeGlob != "" {
				matched, _ := filepath.Match(includeGlob, d.Name())
				if !matched {
					return nil
				}
			}
			// Skip binary-looking files
			if isBinaryExt(d.Name()) {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()

			relPath, _ := filepath.Rel(searchPath, path)
			if relPath == "" {
				relPath = path
			}

			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				if len(matches) >= maxGrepMatches {
					break
				}
				line := scanner.Text()
				if re.MatchString(line) {
					if len(line) > maxLineLen {
						line = line[:maxLineLen] + "..."
					}
					matches = append(matches, fmt.Sprintf("%s:%d:%s", relPath, lineNum, line))
				}
			}
			return nil
		})

		if len(matches) == 0 {
			return &ToolResult{Content: "No matches found"}, nil
		}
		result := strings.Join(matches, "\n")
		if len(matches) >= maxGrepMatches {
			result += fmt.Sprintf("\n... (showing first %d matches)", maxGrepMatches)
		}
		return &ToolResult{Content: result}, nil
	}
}

// --- glob ---

func makeGlobHandler(execCtx ExecutionContext) ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		pattern, _ := input["pattern"].(string)
		if pattern == "" {
			return &ToolResult{Content: "Error: pattern is required", IsError: true}, nil
		}

		basePath, _ := input["path"].(string)
		if basePath == "" {
			basePath = execCtx.ProjectRoot
		}
		if basePath == "" {
			cwd, _ := os.Getwd()
			basePath = cwd
		}

		var results []string
		_ = filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if len(results) >= maxGlobResults {
				return filepath.SkipAll
			}
			// Skip hidden dirs
			if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			if d.IsDir() {
				return nil
			}
			relPath, _ := filepath.Rel(basePath, path)
			matched, _ := filepath.Match(pattern, d.Name())
			if !matched {
				// Try matching against relative path for ** patterns
				matched, _ = filepath.Match(pattern, relPath)
			}
			if matched {
				results = append(results, relPath)
			}
			return nil
		})

		if len(results) == 0 {
			return &ToolResult{Content: "No files matched"}, nil
		}
		output := strings.Join(results, "\n")
		if len(results) >= maxGlobResults {
			output += fmt.Sprintf("\n... (showing first %d results)", maxGlobResults)
		}
		return &ToolResult{Content: output}, nil
	}
}

func isBinaryExt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".exe", ".bin", ".so", ".dylib", ".dll", ".o", ".a",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".webp",
		".pdf", ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z",
		".mp3", ".mp4", ".avi", ".mov", ".wav", ".flac",
		".wasm", ".class", ".pyc", ".pyo":
		return true
	}
	return false
}

// makeFetchHandler creates a handler for the fetch tool (HTTP requests).
func makeFetchHandler() ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		url, _ := input["url"].(string)
		if url == "" {
			return &ToolResult{Content: "Error: url is required"}, nil
		}
		method, _ := input["method"].(string)
		if method == "" {
			method = "GET"
		}

		reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, method, url, nil)
		if err != nil {
			return &ToolResult{Content: fmt.Sprintf("Error creating request: %v", err)}, nil
		}

		if headers, ok := input["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return &ToolResult{Content: fmt.Sprintf("Error: %v", err)}, nil
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
		if err != nil {
			return &ToolResult{Content: fmt.Sprintf("Error reading response: %v", err)}, nil
		}

		result := fmt.Sprintf("HTTP %d %s\n\n%s", resp.StatusCode, resp.Status, string(body))
		if len(result) > 10000 {
			result = result[:10000] + "\n... (truncated)"
		}
		return &ToolResult{Content: result}, nil
	}
}

// makeCalculatorHandler creates a handler for the calculator tool.
func makeCalculatorHandler() ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		expression, _ := input["expression"].(string)
		if expression == "" {
			return &ToolResult{Content: "Error: expression is required"}, nil
		}

		result, err := evaluateExpression(expression)
		if err != nil {
			return &ToolResult{Content: fmt.Sprintf("Error: %v", err)}, nil
		}

		return &ToolResult{Content: fmt.Sprintf("%s = %v", expression, result)}, nil
	}
}

func evaluateExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	expr = strings.ReplaceAll(expr, " ", "")

	if strings.Contains(expr, "+") && !strings.HasPrefix(expr, "+") {
		parts := strings.SplitN(expr, "+", 2)
		a, err1 := strconv.ParseFloat(parts[0], 64)
		b, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil {
			return a + b, nil
		}
	}
	if strings.Contains(expr, "-") && !strings.HasPrefix(expr, "-") {
		parts := strings.SplitN(expr, "-", 2)
		a, err1 := strconv.ParseFloat(parts[0], 64)
		b, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil {
			return a - b, nil
		}
	}
	if strings.Contains(expr, "*") {
		parts := strings.SplitN(expr, "*", 2)
		a, err1 := strconv.ParseFloat(parts[0], 64)
		b, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil {
			return a * b, nil
		}
	}
	if strings.Contains(expr, "/") {
		parts := strings.SplitN(expr, "/", 2)
		a, err1 := strconv.ParseFloat(parts[0], 64)
		b, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil {
			if b == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return a / b, nil
		}
	}

	return strconv.ParseFloat(expr, 64)
}

// makeDatetimeHandler creates a handler for the datetime tool.
func makeDatetimeHandler() ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
		format, _ := input["format"].(string)
		if format == "" {
			format = "RFC3339"
		}
		timezone, _ := input["timezone"].(string)
		if timezone == "" {
			timezone = "Local"
		}

		loc, err := time.LoadLocation(timezone)
		if err != nil {
			loc = time.Local
		}

		now := time.Now().In(loc)

		var result string
		switch format {
		case "RFC3339":
			result = now.Format(time.RFC3339)
		case "RFC1123":
			result = now.Format(time.RFC1123)
		case "Unix":
			result = fmt.Sprintf("%d", now.Unix())
		case "UnixMilli":
			result = fmt.Sprintf("%d", now.UnixMilli())
		case "Date":
			result = now.Format("2006-01-02")
		case "Time":
			result = now.Format("15:04:05")
		case "DateTime":
			result = now.Format("2006-01-02 15:04:05")
		default:
			result = now.Format(format)
		}

		return &ToolResult{Content: result}, nil
	}
}
