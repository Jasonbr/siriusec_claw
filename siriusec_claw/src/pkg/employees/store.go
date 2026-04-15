package employees

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// Manifest represents a digital employee definition.
type Manifest struct {
	ID          string                           `json:"id"`
	Name        string                           `json:"name"`
	Description string                           `json:"description"`
	Prompt      string                           `json:"prompt"`
	Enabled     bool                             `json:"enabled"`
	CreatedAt   int64                            `json:"createdAt"`
	Builtin     bool                             `json:"builtin"`
	SkillIDs    []string                         `json:"skillIds,omitempty"`
	McpServers  map[string]config.McpServerEntry `json:"mcpServers,omitempty"`
	Env         map[string]string                `json:"env,omitempty"` // Environment variables for this employee
	Type        string                           `json:"type,omitempty"`
	From        string                           `json:"from,omitempty"` // "local" or "remote"
}

// Summary is a lightweight employee listing item.
type Summary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Builtin     bool   `json:"builtin"`
	Type        string `json:"type,omitempty"`
}

// ResolveEmployeesDir returns the employees directory path.
func ResolveEmployeesDir(env func(string) string) string {
	return filepath.Join(paths.ResolveStateDir(env), "employees")
}

// ResolveManifestPath returns the manifest file path for an employee.
func ResolveManifestPath(id string, env func(string) string) string {
	return filepath.Join(ResolveEmployeesDir(env), id, "manifest.json")
}

// ListSummaries returns summaries of all employees.
func ListSummaries(env func(string) string) ([]Summary, error) {
	dir := ResolveEmployeesDir(env)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Summary{}, nil
		}
		return nil, err
	}

	var summaries []Summary
	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		m, err := LoadManifest(de.Name(), env)
		if err != nil {
			continue
		}
		summaries = append(summaries, Summary{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Enabled:     m.Enabled,
			Builtin:     m.Builtin,
			Type:        m.Type,
		})
	}
	return summaries, nil
}

// LoadManifest reads an employee's manifest from disk.
func LoadManifest(id string, env func(string) string) (*Manifest, error) {
	path := ResolveManifestPath(id, env)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	m.ID = id
	return &m, nil
}

// SaveManifest writes an employee's manifest to disk.
func SaveManifest(m *Manifest, env func(string) string) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = time.Now().UnixMilli()
	}
	dir := filepath.Join(ResolveEmployeesDir(env), m.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600)
}

// DeleteEmployee removes an employee and its directory.
func DeleteEmployee(id string, env func(string) string) error {
	dir := filepath.Join(ResolveEmployeesDir(env), id)
	return os.RemoveAll(dir)
}

var githubRepoRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)`)

// InstallFromGitHub downloads and installs an employee from a GitHub repository.
func InstallFromGitHub(ctx context.Context, repoURL string, env func(string) string) (*Manifest, error) {
	matches := githubRepoRe.FindStringSubmatch(repoURL)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid GitHub URL: %s (expected github.com/owner/repo)", repoURL)
	}
	owner, repo := matches[1], matches[2]
	repo = strings.TrimSuffix(repo, ".git")

	// Extract subpath if present
	subPath := ""
	treeParts := strings.SplitN(repoURL, "/tree/", 2)
	if len(treeParts) == 2 {
		branchAndPath := treeParts[1]
		slashIdx := strings.Index(branchAndPath, "/")
		if slashIdx >= 0 {
			subPath = branchAndPath[slashIdx+1:]
		}
	}

	// Download zipball
	zipURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", zipURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}

	return InstallFromZip(repo, zipData, env, subPath)
}

// InstallFromUpload installs an employee from an EMPLOYEE.md file content.
func InstallFromUpload(name string, mdContent []byte, env func(string) string) (*Manifest, error) {
	manifest := &Manifest{
		ID:          uuid.New().String(),
		Name:        name,
		Description: "",
		Prompt:      string(mdContent),
		Enabled:     true,
		CreatedAt:   time.Now().UnixMilli(),
		From:        "upload",
	}

	// Try to extract name from frontmatter if present
	if name == "" {
		manifest.Name = "Imported Employee"
	}

	if err := SaveManifest(manifest, env); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	return manifest, nil
}

// InstallFromZip extracts and installs an employee from a zip file.
func InstallFromZip(name string, zipData []byte, env func(string) string, subPath ...string) (*Manifest, error) {
	// Extract zip to temp dir
	tmpDir, err := os.MkdirTemp("", "employee-install-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}

	// Find the root directory name in zip
	var rootDir string
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() && rootDir == "" {
			rootDir = f.Name
			break
		}
	}

	// Extract all files
	for _, f := range zipReader.File {
		path := filepath.Join(tmpDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("create dir: %w", err)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry: %w", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read zip entry: %w", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, fmt.Errorf("write file: %w", err)
		}
	}

	// Look for EMPLOYEE.md or manifest.json
	searchDir := filepath.Join(tmpDir, rootDir)
	if len(subPath) > 0 && subPath[0] != "" {
		searchDir = filepath.Join(searchDir, subPath[0])
	}

	// Try to load manifest.json or config.json first
	for _, filename := range []string{"manifest.json", "config.json"} {
		manifestPath := filepath.Join(searchDir, filename)
		if data, err := os.ReadFile(manifestPath); err == nil {
			var manifest Manifest
			if err := json.Unmarshal(data, &manifest); err == nil {
				manifest.Enabled = true
				manifest.From = "github"
				if manifest.ID == "" {
					manifest.ID = uuid.New().String()
				}
				if manifest.Name == "" {
					manifest.Name = name
				}
				if err := SaveManifest(&manifest, env); err != nil {
					return nil, fmt.Errorf("save manifest: %w", err)
				}
				return &manifest, nil
			}
		}
	}

	// Try EMPLOYEE.md
	mdPath := filepath.Join(searchDir, "EMPLOYEE.md")
	if data, err := os.ReadFile(mdPath); err == nil {
		manifest := &Manifest{
			ID:          uuid.New().String(),
			Name:        name,
			Description: "",
			Prompt:      string(data),
			Enabled:     true,
			CreatedAt:   time.Now().UnixMilli(),
			From:        "github",
		}
		if name == "" {
			manifest.Name = "Imported Employee"
		}
		if err := SaveManifest(manifest, env); err != nil {
			return nil, fmt.Errorf("save manifest: %w", err)
		}
		return manifest, nil
	}

	return nil, fmt.Errorf("no EMPLOYEE.md or manifest.json found in archive")
}
