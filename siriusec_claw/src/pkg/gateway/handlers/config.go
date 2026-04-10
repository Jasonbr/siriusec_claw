package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// LoadConfigSnapshot reads the current config file and returns a snapshot.
func LoadConfigSnapshot(env config.EnvGetter) (*ConfigSnapshot, error) {
	if env == nil {
		env = config.DefaultEnv
	}
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)

	snap := &ConfigSnapshot{
		Path: configPath,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			snap.Exists = false
			return snap, nil
		}
		return nil, err
	}
	snap.Exists = true
	snap.Raw = string(data)
	snap.Hash = fmt.Sprintf("%x", sha256.Sum256(data))

	var cfg config.ClawConfig
	if len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err == nil {
			snap.Parsed = &cfg
			// Extract gateway token
			if cfg.Gateway != nil && cfg.Gateway.Auth != nil {
				snap.GatewayToken = cfg.Gateway.Auth.Token
			}
		}
	}
	return snap, nil
}

// GetExpectedGatewayToken extracts the expected gateway token from a config snapshot loader.
func GetExpectedGatewayToken(loader func() (*ConfigSnapshot, error)) string {
	if loader == nil {
		return ""
	}
	snap, err := loader()
	if err != nil || snap == nil {
		return ""
	}
	return snap.GatewayToken
}

// ConfigGetHandler returns the current config snapshot.
func ConfigGetHandler(opts HandlerOpts) error {
	if opts.Context.LoadConfigSnapshot == nil {
		opts.Respond(false, nil, errNotConfigured("config loader"), nil)
		return nil
	}
	snap, err := opts.Context.LoadConfigSnapshot()
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}
	opts.Respond(true, snap, nil, nil)
	return nil
}

// ConfigEnvHandler returns environment-related config info.
func ConfigEnvHandler(opts HandlerOpts) error {
	env := config.DefaultEnv
	stateDir := paths.ResolveStateDir(env)

	result := map[string]interface{}{
		"stateDir":   stateDir,
		"configPath": paths.ResolveConfigPath(env, stateDir),
	}
	opts.Respond(true, result, nil, nil)
	return nil
}

// ConfigSetHandler overwrites the config file with new content.
func ConfigSetHandler(opts HandlerOpts) error {
	if opts.Params == nil {
		opts.Respond(false, nil, errInvalidParams("params required"), nil)
		return nil
	}

	rawConfig, ok := opts.Params["config"]
	if !ok {
		opts.Respond(false, nil, errInvalidParams("config field required"), nil)
		return nil
	}

	data, err := json.MarshalIndent(rawConfig, "", "  ")
	if err != nil {
		opts.Respond(false, nil, errInternal("failed to marshal config"), nil)
		return nil
	}

	env := config.DefaultEnv
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	// Reload config into context
	cfg, err := config.Load(env)
	if err == nil && opts.Context != nil {
		opts.Context.Config = cfg
	}

	opts.Respond(true, map[string]interface{}{"ok": true}, nil, nil)
	return nil
}

// ConfigPatchHandler merges a patch into the current config.
func ConfigPatchHandler(opts HandlerOpts) error {
	if opts.Params == nil {
		opts.Respond(false, nil, errInvalidParams("params required"), nil)
		return nil
	}

	env := config.DefaultEnv
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)

	// Load current config
	currentData, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	var current map[string]interface{}
	if len(currentData) > 0 {
		_ = json.Unmarshal(currentData, &current)
	}
	if current == nil {
		current = make(map[string]interface{})
	}

	// Get patch from params
	patch, ok := opts.Params["patch"].(map[string]interface{})
	if !ok {
		opts.Respond(false, nil, errInvalidParams("patch field must be an object"), nil)
		return nil
	}

	// Merge patch into current (shallow)
	for k, v := range patch {
		current[k] = v
	}

	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		opts.Respond(false, nil, errInternal("failed to marshal config"), nil)
		return nil
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	// Reload config into context
	cfg, loadErr := config.Load(env)
	if loadErr == nil && opts.Context != nil {
		opts.Context.Config = cfg
	}

	opts.Respond(true, map[string]interface{}{"ok": true}, nil, nil)
	return nil
}

// ConfigSchemaHandler returns config schema info.
func ConfigSchemaHandler(opts HandlerOpts) error {
	opts.Respond(true, map[string]interface{}{
		"version": opts.Context.Version,
		"schema":  "siriusec_claw configuration schema",
	}, nil, nil)
	return nil
}
