package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/agent"
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

// --- agents.list ---

func AgentsListHandler(opts HandlerOpts) error {
	cfg := opts.Context.Config
	var agentsList []map[string]interface{}

	// Always include the "main" agent
	mainAgent := map[string]interface{}{
		"id":      "main",
		"label":   "Main Agent",
		"builtin": true,
	}
	if cfg != nil && cfg.Agents != nil && cfg.Agents.Defaults != nil {
		if ref := extractModelRefFromInterface(cfg.Agents.Defaults.Model); ref != "" {
			mainAgent["model"] = ref
		}
		if cfg.Agents.Defaults.Workspace != "" {
			mainAgent["workspace"] = cfg.Agents.Defaults.Workspace
		}
	}
	agentsList = append(agentsList, mainAgent)

	// Add configured agents
	if cfg != nil && cfg.Agents != nil && cfg.Agents.List != nil {
		for _, a := range cfg.Agents.List {
			if a.ID == "main" {
				continue
			}
			entry := map[string]interface{}{
				"id": a.ID,
			}
			if ref := extractModelRefFromInterface(a.Model); ref != "" {
				entry["model"] = ref
			}
			if a.Workspace != "" {
				entry["workspace"] = a.Workspace
			}
			agentsList = append(agentsList, entry)
		}
	}

	opts.Respond(true, map[string]interface{}{
		"agents": agentsList,
	}, nil, nil)
	return nil
}

// --- agents.create ---

func AgentsCreateHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		id = uuid.New().String()[:8]
	}

	env := envGetter()
	agentDir := agent.ResolveAgentDir(id, env)
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	// Create agent metadata file
	meta := map[string]interface{}{
		"id":        id,
		"createdAt": time.Now().UnixMilli(),
	}
	if label := stringParam(opts.Params, "label", ""); label != "" {
		meta["label"] = label
	}
	if model := stringParam(opts.Params, "model", ""); model != "" {
		meta["model"] = model
	}

	data, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(agentDir, "agent.json")
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok": true,
		"id": id,
	}, nil, nil)
	return nil
}

// --- agents.update ---

func AgentsUpdateHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	env := envGetter()
	agentDir := agent.ResolveAgentDir(id, env)
	metaPath := filepath.Join(agentDir, "agent.json")

	var meta map[string]interface{}
	if data, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(data, &meta)
	}
	if meta == nil {
		meta = map[string]interface{}{"id": id}
	}

	if label, ok := opts.Params["label"]; ok {
		meta["label"] = label
	}
	if model, ok := opts.Params["model"]; ok {
		meta["model"] = model
	}
	meta["updatedAt"] = time.Now().UnixMilli()

	if err := os.MkdirAll(agentDir, 0755); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"ok": true, "id": id}, nil, nil)
	return nil
}

// --- agents.delete ---

func AgentsDeleteHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" || id == "main" {
		opts.Respond(false, nil, errInvalidParams("cannot delete main agent"), nil)
		return nil
	}

	env := envGetter()
	agentDir := agent.ResolveAgentDir(id, env)
	if err := os.RemoveAll(agentDir); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"ok": true, "id": id, "deleted": true}, nil, nil)
	return nil
}

// --- agents.files.list ---

func AgentsFilesListHandler(opts HandlerOpts) error {
	agentID := stringParam(opts.Params, "agentId", "main")
	env := envGetter()
	filesDir := agent.ResolveAgentFilesDir(agentID, env)

	entries, err := os.ReadDir(filesDir)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(true, map[string]interface{}{"files": []interface{}{}}, nil, nil)
			return nil
		}
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	var files []map[string]interface{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":    e.Name(),
			"size":    info.Size(),
			"modTime": info.ModTime().UnixMilli(),
		})
	}

	opts.Respond(true, map[string]interface{}{"files": files}, nil, nil)
	return nil
}

// --- agents.files.get ---

func AgentsFilesGetHandler(opts HandlerOpts) error {
	agentID := stringParam(opts.Params, "agentId", "main")
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	filePath := filepath.Join(agent.ResolveAgentFilesDir(agentID, env), filepath.Base(name))
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(true, map[string]interface{}{"exists": false}, nil, nil)
			return nil
		}
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"exists":  true,
		"name":    name,
		"content": string(data),
	}, nil, nil)
	return nil
}

// --- agents.files.set ---

func AgentsFilesSetHandler(opts HandlerOpts) error {
	agentID := stringParam(opts.Params, "agentId", "main")
	name := stringParam(opts.Params, "name", "")
	content := stringParam(opts.Params, "content", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	filesDir := agent.ResolveAgentFilesDir(agentID, env)
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	filePath := filepath.Join(filesDir, filepath.Base(name))
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"ok": true, "name": name}, nil, nil)
	return nil
}

// --- models.list ---

func ModelsListHandler(opts HandlerOpts) error {
	cfg := opts.Context.Config

	type modelEntry struct {
		Provider string `json:"provider"`
		ID       string `json:"id"`
		Name     string `json:"name,omitempty"`
	}

	// Default models
	models := []modelEntry{
		{Provider: "anthropic", ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5"},
		{Provider: "anthropic", ID: "claude-opus-4-20250514", Name: "Claude Opus 4"},
		{Provider: "anthropic", ID: "claude-haiku-3-5-20241022", Name: "Claude 3.5 Haiku"},
		{Provider: "openai", ID: "gpt-4o", Name: "GPT-4o"},
		{Provider: "openai", ID: "gpt-4o-mini", Name: "GPT-4o Mini"},
		{Provider: "deepseek", ID: "deepseek-chat", Name: "DeepSeek V3"},
		{Provider: "deepseek", ID: "deepseek-reasoner", Name: "DeepSeek R1"},
	}

	// Add models from config
	if cfg != nil && cfg.Models != nil && cfg.Models.Providers != nil {
		for providerName, provider := range cfg.Models.Providers {
			if provider.Models != nil {
				for _, m := range provider.Models {
					models = append(models, modelEntry{
						Provider: providerName,
						ID:       m.ID,
						Name:     m.Name,
					})
				}
			}
		}
	}

	// Resolve current default
	defaultProvider, defaultModel := "anthropic", "claude-sonnet-4-5-20250929"
	if cfg != nil && cfg.Agents != nil && cfg.Agents.Defaults != nil && cfg.Agents.Defaults.Model != nil {
		if ref := extractModelRefFromInterface(cfg.Agents.Defaults.Model); ref != "" {
			defaultProvider, defaultModel = agent.ResolveModelRef(ref)
		}
	}

	opts.Respond(true, map[string]interface{}{
		"models": models,
		"default": map[string]interface{}{
			"provider": defaultProvider,
			"model":    defaultModel,
		},
	}, nil, nil)
	return nil
}

// --- Helper: resolve stateDir from env ---

func resolveStateDir() string {
	return paths.ResolveStateDir(os.Getenv)
}

// --- Helper: resolve config ---

func loadConfigForHandlers() *config.ClawConfig {
	cfg, _ := config.Load(os.Getenv)
	return cfg
}

// extractModelRefFromInterface extracts model reference string from interface{}.
func extractModelRefFromInterface(model interface{}) string {
	if model == nil {
		return ""
	}
	switch v := model.(type) {
	case string:
		return v
	case map[string]interface{}:
		if primary, ok := v["primary"]; ok {
			if s, ok := primary.(string); ok {
				return s
			}
		}
	}
	return ""
}
