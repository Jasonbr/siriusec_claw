package bootstrap

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/siriusec/siriusec_claw/pkg/agent/prompt"
)

// BootstrapFile defines a workspace file that can be injected into the system prompt.
type BootstrapFile struct {
	// Filename is the expected file name (e.g. "AGENTS.md").
	Filename string
	// Availability controls in which prompt mode this file is injected.
	Availability prompt.SectionAvailability
	// Required means a missing-file marker is injected when the file is absent.
	Required bool
}

// DefaultBootstrapFiles returns the standard set of bootstrap files.
func DefaultBootstrapFiles() []BootstrapFile {
	return []BootstrapFile{
		{Filename: "AGENTS.md", Availability: prompt.Always, Required: false},
		{Filename: "SOUL.md", Availability: prompt.FullOnly, Required: false},
		{Filename: "TOOLS.md", Availability: prompt.Always, Required: false},
		{Filename: "IDENTITY.md", Availability: prompt.FullOnly, Required: false},
		{Filename: "USER.md", Availability: prompt.FullOnly, Required: false},
		{Filename: "MEMORY.md", Availability: prompt.FullOnly, Required: false},
	}
}

// MinimalBootstrapFiles returns the reduced set for sub-agents.
func MinimalBootstrapFiles() []BootstrapFile {
	return []BootstrapFile{
		{Filename: "AGENTS.md", Availability: prompt.Always, Required: false},
		{Filename: "TOOLS.md", Availability: prompt.Always, Required: false},
	}
}

// Injector loads bootstrap files and injects them into the prompt builder.
type Injector struct {
	workspaceDir string
	maxChars     int // per-file max chars
	totalMax     int // total max chars across all files
	files        []BootstrapFile
}

// NewInjector creates a bootstrap file injector.
func NewInjector(workspaceDir string, maxChars, totalMax int, files []BootstrapFile) *Injector {
	if maxChars <= 0 {
		maxChars = 20000
	}
	if totalMax <= 0 {
		totalMax = 150000
	}
	if files == nil {
		files = DefaultBootstrapFiles()
	}
	return &Injector{
		workspaceDir: workspaceDir,
		maxChars:     maxChars,
		totalMax:     totalMax,
		files:        files,
	}
}

// Load reads all bootstrap files from the workspace directory.
// Returns a map of filename to content, and a list of truncated file names.
func (in *Injector) Load() (map[string]string, []string, error) {
	contents := make(map[string]string)
	var truncated []string
	totalChars := 0

	for _, bf := range in.files {
		path := filepath.Join(in.workspaceDir, bf.Filename)
		data, err := os.ReadFile(path)
		if err != nil {
			// File doesn't exist - skip silently
			if os.IsNotExist(err) {
				continue
			}
			// Other errors: skip but log
			continue
		}

		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}

		// Per-file truncation
		if len(content) > in.maxChars {
			content = content[:in.maxChars] + "\n... [truncated]"
			truncated = append(truncated, bf.Filename)
		}

		// Total budget check
		if totalChars+len(content) > in.totalMax {
			remaining := in.totalMax - totalChars
			if remaining > 100 {
				content = content[:remaining] + "\n... [truncated due to total budget]"
				truncated = append(truncated, bf.Filename)
			} else {
				truncated = append(truncated, bf.Filename)
				continue
			}
		}

		contents[bf.Filename] = content
		totalChars += len(content)
	}

	return contents, truncated, nil
}

// Inject loads bootstrap files and registers a ProjectContextSection in the builder.
func (in *Injector) Inject(builder *prompt.PromptBuilder) error {
	contents, truncated, err := in.Load()
	if err != nil {
		return err
	}
	builder.Register(prompt.ProjectContextSection(contents, truncated))
	return nil
}

// ListAvailable returns which bootstrap files exist in the workspace.
func (in *Injector) ListAvailable() []BootstrapFileStatus {
	var result []BootstrapFileStatus
	for _, bf := range in.files {
		path := filepath.Join(in.workspaceDir, bf.Filename)
		info, err := os.Stat(path)
		exists := err == nil && !info.IsDir()
		var size int
		if exists {
			size = int(info.Size())
		}
		result = append(result, BootstrapFileStatus{
			Filename:     bf.Filename,
			Exists:       exists,
			Size:         size,
			Availability: string(bf.Availability),
		})
	}
	return result
}

// BootstrapFileStatus describes the state of a single bootstrap file.
type BootstrapFileStatus struct {
	Filename     string `json:"filename"`
	Exists       bool   `json:"exists"`
	Size         int    `json:"size"`
	Availability string `json:"availability"`
}
