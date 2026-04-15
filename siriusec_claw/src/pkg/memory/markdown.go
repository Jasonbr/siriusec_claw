package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/logging"
)

var mdLog = logging.Sub("memory-md")

// MarkdownStore manages memory as Markdown files (dual-layer).
// Layer 1: Daily logs (memory/YYYY-MM-DD.md) - append-only
// Layer 2: Long-term memory (MEMORY.md) - manually curated
type MarkdownStore struct {
	memoryDir string
}

// NewMarkdownStore creates a Markdown-based memory store.
func NewMarkdownStore(env func(string) string) (*MarkdownStore, error) {
	dir := ResolveMemoryDir(env)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create memory dir: %w", err)
	}
	mdLog.Info("markdown memory store initialized dir=%s", dir)
	return &MarkdownStore{memoryDir: dir}, nil
}

// DailyLogPath returns the path for today's daily log.
func (ms *MarkdownStore) DailyLogPath(t time.Time) string {
	return filepath.Join(ms.memoryDir, t.Format("2006-01-02")+".md")
}

// MemoryIndexPath returns the path to the long-term MEMORY.md.
func (ms *MarkdownStore) MemoryIndexPath() string {
	return filepath.Join(ms.memoryDir, "MEMORY.md")
}

// AppendToDailyLog adds an entry to today's daily log file.
func (ms *MarkdownStore) AppendToDailyLog(title, content string, tags []string) error {
	now := time.Now()
	path := ms.DailyLogPath(now)

	// Build the entry
	var sb strings.Builder
	sb.WriteString("\n### ")
	sb.WriteString(now.Format("15:04:05"))
	sb.WriteString(" - ")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	sb.WriteString(content)
	sb.WriteString("\n")
	if len(tags) > 0 {
		sb.WriteString("\nTags: ")
		for i, t := range tags {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("#" + t)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n---\n")

	// Check if file exists to write header
	var f *os.File
	var err error
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("create daily log: %w", err)
		}
		header := fmt.Sprintf("# Daily Memory - %s\n\n> Auto-generated daily log. Append-only.\n", now.Format("2006-01-02"))
		f.WriteString(header)
	} else {
		f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("open daily log: %w", err)
		}
	}
	defer f.Close()

	_, err = f.WriteString(sb.String())
	if err != nil {
		return fmt.Errorf("write daily log: %w", err)
	}

	mdLog.Info("appended to daily log path=%s title=%s", path, title)
	return nil
}

// LoadRecentDailyLogs loads today's and yesterday's daily log content.
func (ms *MarkdownStore) LoadRecentDailyLogs() (map[string]string, error) {
	now := time.Now()
	result := make(map[string]string)

	days := []time.Time{now, now.AddDate(0, 0, -1)}
	for _, day := range days {
		path := ms.DailyLogPath(day)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := day.Format("2006-01-02")
		result[name] = string(data)
	}

	return result, nil
}

// LoadMemoryIndex loads the MEMORY.md long-term memory file.
func (ms *MarkdownStore) LoadMemoryIndex() (string, error) {
	path := ms.MemoryIndexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// SaveMemoryIndex writes to the MEMORY.md file.
func (ms *MarkdownStore) SaveMemoryIndex(content string) error {
	path := ms.MemoryIndexPath()
	return os.WriteFile(path, []byte(content), 0600)
}

// ListDailyLogs returns all daily log file names sorted by date.
func (ms *MarkdownStore) ListDailyLogs() ([]string, error) {
	entries, err := os.ReadDir(ms.memoryDir)
	if err != nil {
		return nil, err
	}

	var logs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".md") && name != "MEMORY.md" {
			// Validate date format YYYY-MM-DD.md
			base := strings.TrimSuffix(name, ".md")
			if _, err := time.Parse("2006-01-02", base); err == nil {
				logs = append(logs, name)
			}
		}
	}

	// Sort by name (date) ascending
	for i := 0; i < len(logs); i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[j] < logs[i] {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	return logs, nil
}

// ReadDailyLog reads a specific daily log by date string (YYYY-MM-DD).
func (ms *MarkdownStore) ReadDailyLog(date string) (string, error) {
	path := filepath.Join(ms.memoryDir, date+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadMarkdownFile reads any .md file in the memory directory.
func (ms *MarkdownStore) ReadMarkdownFile(filename string) (string, error) {
	// Security: only allow reading from memory dir
	clean := filepath.Clean(filename)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid filename: %s", filename)
	}
	path := filepath.Join(ms.memoryDir, clean)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AllMarkdownContent returns all markdown file contents for search indexing.
func (ms *MarkdownStore) AllMarkdownContent() (map[string]string, error) {
	result := make(map[string]string)

	// Walk the memory directory
	entries, err := os.ReadDir(ms.memoryDir)
	if err != nil {
		return result, err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(ms.memoryDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		result[e.Name()] = string(data)
	}

	return result, nil
}
