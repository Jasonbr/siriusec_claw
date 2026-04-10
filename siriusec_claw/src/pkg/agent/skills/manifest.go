package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// InstallManifest stores metadata about how a skill was installed.
type InstallManifest struct {
	Name        string `json:"name"`
	Source      string `json:"source"` // "github", "upload", "url"
	SourceURL   string `json:"sourceURL,omitempty"`
	InstalledAt string `json:"installedAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ReadManifest reads an install manifest from a skill directory.
func ReadManifest(skillDir string) (*InstallManifest, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m InstallManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// WriteManifest writes an install manifest to a skill directory.
func WriteManifest(skillDir string, m *InstallManifest) error {
	if m.InstalledAt == "" {
		m.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	}
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, "manifest.json"), data, 0644)
}
