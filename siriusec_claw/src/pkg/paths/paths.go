package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	stateDirname     = ".siriusec_claw"
	stateDirnameWin  = "siriusec_claw"
	configFilename   = "siriusec_claw.json"
	defaultPort      = 18900
)

// ResolveStateDir returns the state directory for mutable data.
// Override via SIRIUSEC_CLAW_STATE_DIR.
// Default: ~/.siriusec_claw on Linux/macOS, %APPDATA%\siriusec_claw on Windows.
func ResolveStateDir(env func(string) string) string {
	if override := strings.TrimSpace(env("SIRIUSEC_CLAW_STATE_DIR")); override != "" {
		return expandUserPath(override, env)
	}
	if runtime.GOOS == "windows" {
		appData := strings.TrimSpace(env("APPDATA"))
		if appData == "" {
			appData = strings.TrimSpace(env("LOCALAPPDATA"))
		}
		if appData == "" {
			home := resolveHomeDir(env)
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, stateDirnameWin)
	}
	home := resolveHomeDir(env)
	return filepath.Join(home, stateDirname)
}

// ResolveConfigPath returns the active config file path.
// Override via SIRIUSEC_CLAW_CONFIG_PATH.
// Default: $STATE_DIR/siriusec_claw.json
func ResolveConfigPath(env func(string) string, stateDir string) string {
	if override := strings.TrimSpace(env("SIRIUSEC_CLAW_CONFIG_PATH")); override != "" {
		return expandUserPath(override, env)
	}
	return filepath.Join(stateDir, configFilename)
}

// ResolveCanonicalConfigPath returns the canonical config path regardless of file existence.
func ResolveCanonicalConfigPath(env func(string) string, stateDir string) string {
	if override := strings.TrimSpace(env("SIRIUSEC_CLAW_CONFIG_PATH")); override != "" {
		return expandUserPath(override, env)
	}
	return filepath.Join(stateDir, configFilename)
}

// DefaultGatewayPort returns the default gateway listen port.
func DefaultGatewayPort() int {
	return defaultPort
}

// ResolveGatewayPort returns the gateway port from config or env.
func ResolveGatewayPort(portFromConfig *int, env func(string) string) int {
	if envRaw := strings.TrimSpace(env("SIRIUSEC_CLAW_GATEWAY_PORT")); envRaw != "" {
		var n int
		if _, err := fmt.Sscanf(envRaw, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	if portFromConfig != nil && *portFromConfig > 0 {
		return *portFromConfig
	}
	return defaultPort
}

// ResolveRunMode resolves the gateway run mode: "desktop" or "service".
// Priority: env > config > platform default.
func ResolveRunMode(env func(string) string, gatewayModeFromConfig *string) string {
	if env != nil {
		raw := strings.TrimSpace(env("SIRIUSEC_CLAW_RUN_MODE"))
		switch strings.ToLower(raw) {
		case "desktop":
			return "desktop"
		case "service":
			return "service"
		}
	}
	if gatewayModeFromConfig != nil {
		switch strings.ToLower(strings.TrimSpace(*gatewayModeFromConfig)) {
		case "desktop":
			return "desktop"
		case "service":
			return "service"
		case "local":
			return "desktop"
		case "remote":
			return "service"
		}
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return "desktop"
	default:
		return "service"
	}
}

// ResolveGatewayAddr returns the listen address for the given port and run mode.
func ResolveGatewayAddr(port int, runMode string) string {
	if strings.EqualFold(strings.TrimSpace(runMode), "desktop") {
		return fmt.Sprintf("127.0.0.1:%d", port)
	}
	return fmt.Sprintf(":%d", port)
}

func resolveHomeDir(env func(string) string) string {
	home := env("HOME")
	if home == "" {
		home = env("USERPROFILE")
	}
	if home != "" {
		return filepath.Clean(strings.TrimSpace(home))
	}
	dir, err := os.UserHomeDir()
	if err == nil {
		return dir
	}
	return "."
}

func expandUserPath(input string, env func(string) string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "~") {
		home := resolveHomeDir(env)
		return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(input, "~"), "/"))
	}
	if abs, err := filepath.Abs(input); err == nil {
		return abs
	}
	return input
}
