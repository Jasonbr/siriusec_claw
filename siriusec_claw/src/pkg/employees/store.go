package employees

import (
	"encoding/json"
	"os"
	"path/filepath"
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
