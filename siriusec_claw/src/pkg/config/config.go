package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// EnvGetter returns an environment variable value by name.
type EnvGetter func(string) string

// DefaultEnv uses os.Getenv.
func DefaultEnv(key string) string {
	return os.Getenv(key)
}

// DefaultGatewayToken is the default gateway.auth.token for new configs.
const DefaultGatewayToken = "siriusec_claw_default_token_2026"

// Load reads and parses the config file.
// Returns default config if file does not exist or is empty.
func Load(env EnvGetter) (*ClawConfig, error) {
	if env == nil {
		env = DefaultEnv
	}
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ClawConfig{}, nil
		}
		return nil, err
	}

	var cfg ClawConfig
	if len(data) == 0 {
		return &cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to the given path.
func Save(path string, cfg *ClawConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// EnsureDefaultConfig ensures the config file exists; if not, creates directory and writes defaults.
func EnsureDefaultConfig(env EnvGetter) error {
	if env == nil {
		env = DefaultEnv
	}
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveCanonicalConfigPath(env, stateDir)

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	modeToken := "token"
	modeLocal := "local"
	bindLoopback := "loopback"
	port := 18900

	cfg := &ClawConfig{
		Meta: &ConfigMeta{
			LastTouchedVersion: "0.0.1",
			LastTouchedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		},
		Gateway: &GatewayConfig{
			Port: &port,
			Mode: &modeLocal,
			Bind: &bindLoopback,
			Auth: &GatewayAuthConfig{
				Mode:  &modeToken,
				Token: DefaultGatewayToken,
			},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

// SetSkillConfig updates the configuration for a specific skill.
func SetSkillConfig(name string, skillCfg SkillConfig, env EnvGetter) error {
	if env == nil {
		env = DefaultEnv
	}

	// Load existing config
	cfg, err := Load(env)
	if err != nil {
		return err
	}

	// Initialize skills config if needed
	if cfg.Skills == nil {
		cfg.Skills = &SkillsConfig{}
	}
	if cfg.Skills.Entries == nil {
		cfg.Skills.Entries = make(map[string]SkillConfig)
	}

	// Update the skill config
	cfg.Skills.Entries[name] = skillCfg

	// Save the config
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveCanonicalConfigPath(env, stateDir)

	// Ensure directory exists
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	return Save(configPath, cfg)
}

// GetSkillConfig retrieves the configuration for a specific skill.
func GetSkillConfig(name string, env EnvGetter) (*SkillConfig, bool) {
	if env == nil {
		env = DefaultEnv
	}

	cfg, err := Load(env)
	if err != nil {
		return nil, false
	}

	if cfg.Skills == nil || cfg.Skills.Entries == nil {
		return nil, false
	}

	skillCfg, ok := cfg.Skills.Entries[name]
	if !ok {
		return nil, false
	}
	return &skillCfg, true
}

// GetMarketplaceConfig retrieves the marketplace configuration.
func GetMarketplaceConfig(env EnvGetter) *MarketplaceConfig {
	if env == nil {
		env = DefaultEnv
	}

	cfg, err := Load(env)
	if err != nil {
		return nil
	}

	if cfg.Marketplace == nil {
		return nil
	}
	return cfg.Marketplace
}

// SetMarketplaceConfig updates the marketplace configuration.
func SetMarketplaceConfig(marketCfg *MarketplaceConfig, env EnvGetter) error {
	if env == nil {
		env = DefaultEnv
	}

	// Load existing config
	cfg, err := Load(env)
	if err != nil {
		return err
	}

	cfg.Marketplace = marketCfg

	// Save the config
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveCanonicalConfigPath(env, stateDir)

	// Ensure directory exists
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	return Save(configPath, cfg)
}
