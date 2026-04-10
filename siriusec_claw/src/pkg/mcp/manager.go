package mcp

import (
	"encoding/json"
	"os"

	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// ServerManifest represents a managed MCP server with metadata.
type ServerManifest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Author      string                 `json:"author,omitempty"`
	Homepage    string                 `json:"homepage,omitempty"`
	SourceURL   string                 `json:"sourceUrl,omitempty"`
	InstalledAt int64                  `json:"installedAt,omitempty"`
	UpdatedAt   int64                  `json:"updatedAt,omitempty"`
	Source      string                 `json:"source"` // "upload", "github", "marketplace"
	Config      *config.McpServerEntry `json:"config"`
}

// ManagedServersDir returns the managed MCP servers directory.
func ManagedServersDir(env func(string) string) string {
	stateDir := ""
	if env != nil {
		stateDir = env("SIRIUSEC_CLAW_STATE_DIR")
	}
	if stateDir == "" {
		home := ""
		if env != nil {
			home = env("HOME")
		}
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		stateDir = home + "/.siriusec_claw"
	}
	return stateDir + "/mcp_servers"
}

// AddServer adds an MCP server to the config file.
func AddServer(name string, entry config.McpServerEntry, env func(string) string) error {
	if env == nil {
		env = os.Getenv
	}

	// Load existing config
	cfg, err := config.Load(env)
	if err != nil {
		cfg = &config.ClawConfig{}
	}

	// Initialize MCP config if needed
	if cfg.Mcp == nil {
		cfg.Mcp = &config.McpConfig{}
	}
	if cfg.Mcp.Servers == nil {
		cfg.Mcp.Servers = make(map[string]config.McpServerEntry)
	}

	// Add/update server
	cfg.Mcp.Servers[name] = entry

	// Save config
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveCanonicalConfigPath(env, stateDir)
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	return config.Save(configPath, cfg)
}

// RemoveServer removes an MCP server from the config file.
func RemoveServer(name string, env func(string) string) error {
	if env == nil {
		env = os.Getenv
	}

	cfg, err := config.Load(env)
	if err != nil {
		return err
	}

	if cfg.Mcp == nil || cfg.Mcp.Servers == nil {
		return nil
	}

	delete(cfg.Mcp.Servers, name)

	// Save config
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveCanonicalConfigPath(env, stateDir)
	return config.Save(configPath, cfg)
}

// ListServers returns all configured MCP servers.
func ListServers(env func(string) string) (map[string]config.McpServerEntry, error) {
	if env == nil {
		env = os.Getenv
	}

	cfg, err := config.Load(env)
	if err != nil {
		return nil, err
	}

	if cfg.Mcp == nil || cfg.Mcp.Servers == nil {
		return map[string]config.McpServerEntry{}, nil
	}

	return cfg.Mcp.Servers, nil
}

// GetServer returns a specific MCP server config.
func GetServer(name string, env func(string) string) (*config.McpServerEntry, bool) {
	if env == nil {
		env = os.Getenv
	}

	cfg, err := config.Load(env)
	if err != nil {
		return nil, false
	}

	if cfg.Mcp == nil || cfg.Mcp.Servers == nil {
		return nil, false
	}

	entry, ok := cfg.Mcp.Servers[name]
	if !ok {
		return nil, false
	}
	return &entry, true
}

// SaveManagedManifest saves a managed MCP server manifest.
func SaveManagedManifest(name string, manifest *ServerManifest, env func(string) string) error {
	dir := ManagedServersDir(env) + "/" + name
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dir+"/manifest.json", data, 0644)
}

// LoadManagedManifest loads a managed MCP server manifest.
func LoadManagedManifest(name string, env func(string) string) (*ServerManifest, error) {
	path := ManagedServersDir(env) + "/" + name + "/manifest.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m ServerManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteManagedServer removes a managed MCP server.
func DeleteManagedServer(name string, env func(string) string) error {
	dir := ManagedServersDir(env) + "/" + name
	return os.RemoveAll(dir)
}
