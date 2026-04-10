package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// ResolveAgentWorkspaceDir returns the workspace directory for a given agent.
// Priority: agent-specific config > defaults config > state dir default.
func ResolveAgentWorkspaceDir(agentID string, agentWorkspace, defaultWorkspace string, env func(string) string) string {
	if agentWorkspace != "" {
		return expandPath(agentWorkspace, env)
	}
	if defaultWorkspace != "" {
		return expandPath(defaultWorkspace, env)
	}
	stateDir := paths.ResolveStateDir(env)
	if agentID == "" || agentID == "main" {
		return filepath.Join(stateDir, "workspace")
	}
	return filepath.Join(stateDir, "agents", agentID, "workspace")
}

// ResolveAgentDir returns the agent configuration directory.
func ResolveAgentDir(agentID string, env func(string) string) string {
	stateDir := paths.ResolveStateDir(env)
	if agentID == "" || agentID == "main" {
		return filepath.Join(stateDir, "agents", "main")
	}
	return filepath.Join(stateDir, "agents", agentID)
}

// ResolveAgentFilesDir returns the directory for agent-specific files.
func ResolveAgentFilesDir(agentID string, env func(string) string) string {
	return filepath.Join(ResolveAgentDir(agentID, env), "files")
}

// DefaultPlatform returns the current platform identifier.
func DefaultPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func expandPath(p string, env func(string) string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~") {
		home := env("HOME")
		if home == "" {
			home = env("USERPROFILE")
		}
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
