package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Entry represents a loaded skill.
type Entry struct {
	Name            string            `json:"name"`
	Source          string            `json:"source"`   // "builtin", "managed", "workspace"
	FilePath        string            `json:"filePath"` // path to SKILL.md
	BaseDir         string            `json:"baseDir"`
	Metadata        *Metadata         `json:"metadata,omitempty"`
	Frontmatter     map[string]string `json:"frontmatter,omitempty"`
	EmbeddedContent []byte            `json:"-"`
}

// Metadata is parsed from the SKILL.md frontmatter.
type Metadata struct {
	SkillKey   string        `json:"skillKey,omitempty" yaml:"skill-key"`
	PrimaryEnv string        `json:"primaryEnv,omitempty" yaml:"primary-env"`
	Always     *bool         `json:"always,omitempty" yaml:"always"`
	OS         []string      `json:"os,omitempty" yaml:"os"`
	Requires   *Requires     `json:"requires,omitempty" yaml:"requires"`
	Install    []InstallSpec `json:"install,omitempty" yaml:"install"`
	Emoji      string        `json:"emoji,omitempty" yaml:"emoji"`
	Homepage   string        `json:"homepage,omitempty" yaml:"homepage"`
	Name       string        `json:"name,omitempty" yaml:"name"`
	Desc       string        `json:"description,omitempty" yaml:"description"`
	APIConfig  *APIConfig    `json:"apiConfig,omitempty" yaml:"apiConfig"`
}

// Requires defines skill dependencies.
type Requires struct {
	Bins []string `json:"bins,omitempty" yaml:"bins"`
	Envs []string `json:"envs,omitempty" yaml:"envs"`
}

// APIConfig defines the API configuration a skill requires.
type APIConfig struct {
	// Endpoint fields
	URL         string `json:"url,omitempty" yaml:"url"`                 // API endpoint URL
	URLLabel    string `json:"urlLabel,omitempty" yaml:"urlLabel"`       // Label for URL field (default: "API URL")
	URLRequired bool   `json:"urlRequired,omitempty" yaml:"urlRequired"` // Whether URL is required

	// Auth type: "none", "token", "basic", "apikey"
	AuthType string `json:"authType,omitempty" yaml:"authType"`

	// Token auth
	TokenLabel string `json:"tokenLabel,omitempty" yaml:"tokenLabel"` // Label for token field

	// Basic auth
	UsernameLabel string `json:"usernameLabel,omitempty" yaml:"usernameLabel"`
	PasswordLabel string `json:"passwordLabel,omitempty" yaml:"passwordLabel"`

	// API Key auth
	APIKeyLabel  string `json:"apiKeyLabel,omitempty" yaml:"apiKeyLabel"`
	APIKeyHeader string `json:"apiKeyHeader,omitempty" yaml:"apiKeyHeader"` // Header name for API key

	// Additional config fields
	ExtraFields []APIConfigField `json:"extraFields,omitempty" yaml:"extraFields"`
}

// APIConfigField defines an additional configuration field.
type APIConfigField struct {
	Name        string   `json:"name" yaml:"name"`               // Field name (used as key in config)
	Label       string   `json:"label" yaml:"label"`             // Display label
	Type        string   `json:"type" yaml:"type"`               // Field type: "text", "password", "number", "boolean", "select"
	Required    bool     `json:"required" yaml:"required"`       // Whether field is required
	Default     string   `json:"default" yaml:"default"`         // Default value
	Placeholder string   `json:"placeholder" yaml:"placeholder"` // Placeholder text
	Options     []string `json:"options" yaml:"options"`         // Options for select type
	Description string   `json:"description" yaml:"description"` // Help text
}

// InstallSpec describes how to install a skill.
type InstallSpec struct {
	Platform string `json:"platform,omitempty" yaml:"platform"`
	Command  string `json:"command" yaml:"command"`
}

// StatusEntry is the status of a single skill for the skills.status response.
type StatusEntry struct {
	Name        string     `json:"name"`
	Source      string     `json:"source"`
	Enabled     bool       `json:"enabled"`
	FilePath    string     `json:"filePath"`
	HasMetadata bool       `json:"hasMetadata"`
	APIConfig   *APIConfig `json:"apiConfig,omitempty"`
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// SanitizeName normalizes a skill name to valid format.
func SanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)
	name = strings.Trim(name, "-")
	if len(name) > 64 {
		name = name[:64]
	}
	if name == "" {
		name = "skill"
	}
	return name
}

// LoadWorkspaceEntries discovers and loads skills from multiple directories.
// Priority: extra dirs < bundled (embedded) < managed < workspace (highest).
func LoadWorkspaceEntries(env func(string) string, workspaceDir string, extraDirs []string) ([]Entry, error) {
	return LoadWorkspaceEntriesWithFS(nil, env, workspaceDir, extraDirs)
}

// LoadWorkspaceEntriesWithFS discovers and loads skills including embedded bundled skills.
func LoadWorkspaceEntriesWithFS(bundledFS fs.FS, env func(string) string, workspaceDir string, extraDirs []string) ([]Entry, error) {
	stateDir := ""
	if env != nil {
		stateDir = os.Getenv("SIRIUSEC_CLAW_STATE_DIR")
		if stateDir == "" {
			home := env("HOME")
			if home == "" {
				home, _ = os.UserHomeDir()
			}
			stateDir = filepath.Join(home, ".siriusec_claw")
		}
	}

	var entries []Entry
	seen := make(map[string]int) // name → index in entries (for priority override)

	// 1. Extra directories (lowest priority)
	for _, dir := range extraDirs {
		loaded := loadSkillsFromDir(dir, "extra")
		for _, e := range loaded {
			if idx, ok := seen[e.Name]; ok {
				entries[idx] = e
			} else {
				seen[e.Name] = len(entries)
				entries = append(entries, e)
			}
		}
	}

	// 2. Bundled skills (from embedded FS or env var)
	if bundledFS != nil {
		loaded := loadSkillsFromEmbedFS(bundledFS, "builtin")
		for _, e := range loaded {
			if idx, ok := seen[e.Name]; ok {
				entries[idx] = e
			} else {
				seen[e.Name] = len(entries)
				entries = append(entries, e)
			}
		}
	} else {
		bundledDir := os.Getenv("SIRIUSEC_CLAW_BUNDLED_SKILLS_DIR")
		if bundledDir != "" {
			loaded := loadSkillsFromDir(bundledDir, "bundled")
			for _, e := range loaded {
				if idx, ok := seen[e.Name]; ok {
					entries[idx] = e
				} else {
					seen[e.Name] = len(entries)
					entries = append(entries, e)
				}
			}
		}
	}

	// 3. Managed skills (~/.siriusec_claw/skills)
	if stateDir != "" {
		managedDir := filepath.Join(stateDir, "skills")
		loaded := loadSkillsFromDir(managedDir, "managed")
		for _, e := range loaded {
			if idx, ok := seen[e.Name]; ok {
				entries[idx] = e
			} else {
				seen[e.Name] = len(entries)
				entries = append(entries, e)
			}
		}
	}

	// 4. Workspace skills (highest priority)
	if workspaceDir != "" {
		wsSkillsDir := filepath.Join(workspaceDir, "skills")
		loaded := loadSkillsFromDir(wsSkillsDir, "workspace")
		for _, e := range loaded {
			if idx, ok := seen[e.Name]; ok {
				entries[idx] = e
			} else {
				seen[e.Name] = len(entries)
				entries = append(entries, e)
			}
		}
	}

	return entries, nil
}

// LoadSpecificSkills loads only the skills with the specified IDs.
// It searches through all skill sources (workspace, managed, bundled) for the requested skills.
func LoadSpecificSkills(bundledFS fs.FS, env func(string) string, workspaceDir string, skillIDs []string) ([]Entry, error) {
	if len(skillIDs) == 0 {
		return []Entry{}, nil
	}

	// Create a set of requested skill IDs for quick lookup
	requested := make(map[string]bool)
	for _, id := range skillIDs {
		requested[SanitizeName(id)] = true
	}

	// Load all available skills first
	allSkills, err := LoadWorkspaceEntriesWithFS(bundledFS, env, workspaceDir, nil)
	if err != nil {
		return nil, err
	}

	// Filter to only requested skills
	var filtered []Entry
	for _, skill := range allSkills {
		if requested[skill.Name] {
			filtered = append(filtered, skill)
		}
	}

	return filtered, nil
}

// loadSkillsFromDir scans a directory for SKILL.md files.
func loadSkillsFromDir(dir, source string) []Entry {
	if dir == "" {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var entries []Entry
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, de := range dirEntries {
		if !de.IsDir() {
			// Check if it's a SKILL.md file directly
			if strings.EqualFold(de.Name(), "SKILL.md") {
				name := SanitizeName(filepath.Base(dir))
				content, err := os.ReadFile(filepath.Join(dir, de.Name()))
				if err != nil {
					continue
				}
				meta, body := ParseFrontmatter(content)
				if meta != nil && meta.Name != "" {
					name = SanitizeName(meta.Name)
				}
				entries = append(entries, Entry{
					Name:            name,
					Source:          source,
					FilePath:        filepath.Join(dir, de.Name()),
					BaseDir:         dir,
					Metadata:        meta,
					EmbeddedContent: body,
				})
			}
			continue
		}
		// Look for SKILL.md inside subdirectory
		skillFile := filepath.Join(dir, de.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		name := SanitizeName(de.Name())
		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		meta, body := ParseFrontmatter(content)
		if meta != nil && meta.Name != "" {
			name = SanitizeName(meta.Name)
		}
		entries = append(entries, Entry{
			Name:            name,
			Source:          source,
			FilePath:        skillFile,
			BaseDir:         filepath.Join(dir, de.Name()),
			Metadata:        meta,
			EmbeddedContent: body,
		})
	}

	return entries
}

// loadSkillsFromEmbedFS loads skills from an embedded file system.
func loadSkillsFromEmbedFS(fsys fs.FS, source string) []Entry {
	if fsys == nil {
		return nil
	}

	var entries []Entry
	dirEntries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil
	}

	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		skillFile := filepath.Join(de.Name(), "SKILL.md")
		content, err := fs.ReadFile(fsys, skillFile)
		if err != nil {
			continue
		}
		name := SanitizeName(de.Name())
		meta, body := ParseFrontmatter(content)
		if meta != nil && meta.Name != "" {
			name = SanitizeName(meta.Name)
		}
		entries = append(entries, Entry{
			Name:            name,
			Source:          source,
			FilePath:        skillFile, // virtual path in embed FS
			BaseDir:         de.Name(),
			Metadata:        meta,
			EmbeddedContent: body,
		})
	}

	return entries
}

// GetStatus returns status entries for all loaded skills.
func GetStatus(entries []Entry) []StatusEntry {
	result := make([]StatusEntry, 0, len(entries))
	for _, e := range entries {
		se := StatusEntry{
			Name:        e.Name,
			Source:      e.Source,
			Enabled:     true,
			FilePath:    e.FilePath,
			HasMetadata: e.Metadata != nil,
		}
		if e.Metadata != nil && e.Metadata.APIConfig != nil {
			se.APIConfig = e.Metadata.APIConfig
		}
		result = append(result, se)
	}
	return result
}
