package mcp

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
"strings"
"time"

"github.com/siriusec/siriusec_claw/pkg/config"
"github.com/siriusec/siriusec_claw/pkg/paths"
)

// EnvGetter is a function that returns environment variable values
type EnvGetter = config.EnvGetter

// ServerManifest represents metadata for a managed MCP server
type ServerManifest struct {
Name        string                 `json:"name"`
Description string                 `json:"description"`
Author      string                 `json:"author"`
Homepage    string                 `json:"homepage"`
Source      string                 `json:"source"`
InstalledAt int64                  `json:"installedAt"`
UpdatedAt   int64                  `json:"updatedAt,omitempty"`
Config      *config.McpServerEntry `json:"config,omitempty"`
}

// ListServers returns all MCP servers from config
func ListServers(env EnvGetter) (map[string]config.McpServerEntry, error) {
cfg, err := config.Load(env)
if err != nil {
return nil, err
}
if cfg.Mcp == nil || cfg.Mcp.Servers == nil {
return make(map[string]config.McpServerEntry), nil
}
return cfg.Mcp.Servers, nil
}

// GetServer returns a specific MCP server entry
func GetServer(name string, env EnvGetter) (*config.McpServerEntry, bool) {
servers, err := ListServers(env)
if err != nil {
return nil, false
}
entry, ok := servers[name]
return &entry, ok
}

// AddServer adds or updates an MCP server in config
func AddServer(name string, entry config.McpServerEntry, env EnvGetter) error {
cfg, err := config.Load(env)
if err != nil {
cfg = &config.ClawConfig{}
}
if cfg.Mcp == nil {
cfg.Mcp = &config.McpConfig{}
}
if cfg.Mcp.Servers == nil {
cfg.Mcp.Servers = make(map[string]config.McpServerEntry)
}
cfg.Mcp.Servers[name] = entry

stateDir := paths.ResolveStateDir(env)
configPath := paths.ResolveConfigPath(env, stateDir)
return config.Save(configPath, cfg)
}

// RemoveServer removes an MCP server from config
func RemoveServer(name string, env EnvGetter) error {
cfg, err := config.Load(env)
if err != nil {
return err
}
if cfg.Mcp == nil || cfg.Mcp.Servers == nil {
return nil
}
delete(cfg.Mcp.Servers, name)

stateDir := paths.ResolveStateDir(env)
configPath := paths.ResolveConfigPath(env, stateDir)
return config.Save(configPath, cfg)
}

// managedServersDir returns the directory for managed server manifests
func managedServersDir(env EnvGetter) string {
return filepath.Join(paths.ResolveStateDir(env), "mcp-servers")
}

// LoadManagedManifest loads the manifest for a managed MCP server
func LoadManagedManifest(name string, env EnvGetter) (*ServerManifest, error) {
manifestPath := filepath.Join(managedServersDir(env), name+".json")
data, err := os.ReadFile(manifestPath)
if err != nil {
return nil, err
}
var manifest ServerManifest
if err := json.Unmarshal(data, &manifest); err != nil {
return nil, err
}
return &manifest, nil
}

// SaveManagedManifest saves the manifest for a managed MCP server
func SaveManagedManifest(name string, manifest *ServerManifest, env EnvGetter) error {
dir := managedServersDir(env)
if err := os.MkdirAll(dir, 0755); err != nil {
return err
}
manifestPath := filepath.Join(dir, name+".json")
data, err := json.MarshalIndent(manifest, "", "  ")
if err != nil {
return err
}
return os.WriteFile(manifestPath, data, 0644)
}

// DeleteManagedServer removes a managed MCP server manifest
func DeleteManagedServer(name string, env EnvGetter) error {
manifestPath := filepath.Join(managedServersDir(env), name+".json")
return os.Remove(manifestPath)
}

// InstallFromGitHub installs an MCP server from a GitHub repository
func InstallFromGitHub(ctx context.Context, url string, env EnvGetter) (*ServerManifest, error) {
// Parse GitHub URL and construct raw content URL
mcpURL := strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
if !strings.Contains(mcpURL, "/blob/") && !strings.Contains(mcpURL, "/tree/") {
mcpURL = strings.TrimSuffix(mcpURL, "/") + "/main/MCP.md"
}

req, err := http.NewRequestWithContext(ctx, "GET", mcpURL, nil)
if err != nil {
return nil, err
}

resp, err := http.DefaultClient.Do(req)
if err != nil {
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
mcpURL = strings.Replace(mcpURL, "/main/", "/master/", 1)
req, _ = http.NewRequestWithContext(ctx, "GET", mcpURL, nil)
resp, err = http.DefaultClient.Do(req)
if err != nil {
return nil, err
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("MCP.md not found in repository")
}
}

content, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

return InstallFromUpload("", content, env)
}

// InstallFromUpload installs an MCP server from uploaded content (MCP.md)
func InstallFromUpload(name string, content []byte, env EnvGetter) (*ServerManifest, error) {
manifest, entry, err := parseMCPMarkdown(content, name)
if err != nil {
return nil, err
}

if err := AddServer(manifest.Name, *entry, env); err != nil {
return nil, err
}

manifest.Source = "upload"
manifest.InstalledAt = time.Now().UnixMilli()
if err := SaveManagedManifest(manifest.Name, manifest, env); err != nil {
return nil, err
}

return manifest, nil
}

// InstallFromZip installs an MCP server from a zip file
func InstallFromZip(name string, zipContent []byte, env EnvGetter) (*ServerManifest, error) {
reader, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
if err != nil {
return nil, fmt.Errorf("failed to read zip: %w", err)
}

var mcpContent []byte
for _, file := range reader.File {
if strings.EqualFold(file.Name, "MCP.md") || strings.HasSuffix(file.Name, "/MCP.md") {
rc, err := file.Open()
if err != nil {
return nil, err
}
mcpContent, err = io.ReadAll(rc)
rc.Close()
if err != nil {
return nil, err
}
break
}
}

if mcpContent == nil {
return nil, fmt.Errorf("MCP.md not found in zip file")
}

manifest, err := InstallFromUpload(name, mcpContent, env)
if err != nil {
return nil, err
}

manifest.Source = "zip"
SaveManagedManifest(manifest.Name, manifest, env)

return manifest, nil
}

// parseMCPMarkdown parses MCP.md content and extracts configuration
func parseMCPMarkdown(content []byte, defaultName string) (*ServerManifest, *config.McpServerEntry, error) {
contentStr := string(content)

manifest := &ServerManifest{
Name:        defaultName,
InstalledAt: time.Now().UnixMilli(),
}
entry := &config.McpServerEntry{}

if strings.HasPrefix(contentStr, "---") {
endIdx := strings.Index(contentStr[3:], "---")
if endIdx > 0 {
frontmatter := contentStr[3 : endIdx+3]
lines := strings.Split(frontmatter, "\n")
for _, line := range lines {
line = strings.TrimSpace(line)
if line == "" || strings.HasPrefix(line, "#") {
continue
}

if idx := strings.Index(line, ":"); idx > 0 {
key := strings.TrimSpace(line[:idx])
value := strings.TrimSpace(line[idx+1:])
value = strings.Trim(value, "\"'")

switch key {
case "name":
manifest.Name = value
case "description":
manifest.Description = value
case "author":
manifest.Author = value
case "homepage":
manifest.Homepage = value
case "command":
entry.Command = value
case "toolPrefix":
entry.ToolPrefix = value
}
}
}
}
}

if manifest.Name == "" {
manifest.Name = defaultName
if manifest.Name == "" {
manifest.Name = "mcp-server-" + fmt.Sprintf("%d", time.Now().Unix())
}
}

if entry.Command == "" {
if idx := strings.Index(contentStr, "```bash"); idx > 0 {
blockStart := idx + 7
blockEnd := strings.Index(contentStr[blockStart:], "```")
if blockEnd > 0 {
cmdBlock := strings.TrimSpace(contentStr[blockStart : blockStart+blockEnd])
lines := strings.Split(cmdBlock, "\n")
for _, line := range lines {
line = strings.TrimSpace(line)
if line != "" && !strings.HasPrefix(line, "#") {
parts := strings.Fields(line)
if len(parts) > 0 {
entry.Command = parts[0]
if len(parts) > 1 {
entry.Args = parts[1:]
}
break
}
}
}
}
}
}

enabled := true
entry.Enabled = &enabled

return manifest, entry, nil
}
