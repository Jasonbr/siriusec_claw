package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"time"

	"github.com/siriusec/siriusec_claw/embed"
	"github.com/siriusec/siriusec_claw/pkg/agent/skills"
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/employees"
)

// --- skills.status ---

func SkillsStatusHandler(opts HandlerOpts) error {
	env := envGetter()
	workspaceDir := ""
	if opts.Context.Config != nil && opts.Context.Config.Agents != nil && opts.Context.Config.Agents.Defaults != nil {
		workspaceDir = opts.Context.Config.Agents.Defaults.Workspace
	}

	// Load embedded bundled skills
	bundledFS, _ := embed.SkillsFS()
	entries, err := skills.LoadWorkspaceEntriesWithFS(bundledFS, env, workspaceDir, nil)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	status := skills.GetStatus(entries)

	// Enrich with metadata and manifest info
	enriched := make([]map[string]interface{}, 0, len(entries))
	for i, e := range entries {
		item := map[string]interface{}{
			"name":        status[i].Name,
			"source":      status[i].Source,
			"enabled":     status[i].Enabled,
			"filePath":    status[i].FilePath,
			"hasMetadata": status[i].HasMetadata,
		}
		if e.Metadata != nil {
			if e.Metadata.Emoji != "" {
				item["emoji"] = e.Metadata.Emoji
			}
			if e.Metadata.Homepage != "" {
				item["homepage"] = e.Metadata.Homepage
			}
			if e.Metadata.Desc != "" {
				item["description"] = e.Metadata.Desc
			}
			if e.Metadata.APIConfig != nil {
				item["apiConfig"] = e.Metadata.APIConfig
			}
		}
		// Try to read manifest for managed skills
		if e.Source == "managed" {
			if m, err := skills.ReadManifest(e.BaseDir); err == nil {
				item["sourceURL"] = m.SourceURL
				item["installedAt"] = m.InstalledAt
				item["updatedAt"] = m.UpdatedAt
				item["installSource"] = m.Source
			}
		}
		enriched = append(enriched, item)
	}

	opts.Respond(true, map[string]interface{}{
		"skills": enriched,
		"count":  len(enriched),
	}, nil, nil)
	return nil
}

// --- skills.getDoc ---

func SkillsGetDocHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	workspaceDir := ""
	if opts.Context.Config != nil && opts.Context.Config.Agents != nil && opts.Context.Config.Agents.Defaults != nil {
		workspaceDir = opts.Context.Config.Agents.Defaults.Workspace
	}

	bundledFS, _ := embed.SkillsFS()
	entries, _ := skills.LoadWorkspaceEntriesWithFS(bundledFS, env, workspaceDir, nil)
	for _, e := range entries {
		if e.Name == name {
			content := string(e.EmbeddedContent)
			if content == "" && e.FilePath != "" {
				if data, err := os.ReadFile(e.FilePath); err == nil {
					content = string(data)
				}
			}
			opts.Respond(true, map[string]interface{}{
				"name":     e.Name,
				"source":   e.Source,
				"filePath": e.FilePath,
				"content":  content,
			}, nil, nil)
			return nil
		}
	}

	opts.Respond(false, nil, errInvalidParams("skill not found: "+name), nil)
	return nil
}

// --- skills.install ---

func SkillsInstallHandler(opts HandlerOpts) error {
	source := stringParam(opts.Params, "source", "")
	if source == "" {
		opts.Respond(false, nil, errInvalidParams("source required (github, upload, or zip)"), nil)
		return nil
	}

	env := envGetter()
	var result *skills.InstallResult
	var err error

	switch source {
	case "github":
		url := stringParam(opts.Params, "url", "")
		if url == "" {
			opts.Respond(false, nil, errInvalidParams("url required for github source"), nil)
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		result, err = skills.InstallFromGitHub(ctx, url, env)

	case "upload":
		contentB64 := stringParam(opts.Params, "content", "")
		if contentB64 == "" {
			opts.Respond(false, nil, errInvalidParams("content required for upload source (base64 encoded)"), nil)
			return nil
		}
		content, decodeErr := base64.StdEncoding.DecodeString(contentB64)
		if decodeErr != nil {
			opts.Respond(false, nil, errInvalidParams("invalid base64 content: "+decodeErr.Error()), nil)
			return nil
		}
		name := stringParam(opts.Params, "name", "")
		result, err = skills.InstallFromUpload(name, content, env)

	case "zip":
		contentB64 := stringParam(opts.Params, "content", "")
		if contentB64 == "" {
			opts.Respond(false, nil, errInvalidParams("content required for zip source (base64 encoded zip file)"), nil)
			return nil
		}
		content, decodeErr := base64.StdEncoding.DecodeString(contentB64)
		if decodeErr != nil {
			opts.Respond(false, nil, errInvalidParams("invalid base64 content: "+decodeErr.Error()), nil)
			return nil
		}
		name := stringParam(opts.Params, "name", "")
		result, err = skills.InstallFromZip(name, content, env)

	default:
		opts.Respond(false, nil, errInvalidParams("invalid source: "+source), nil)
		return nil
	}

	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"name":     result.Name,
		"skillDir": result.SkillDir,
		"source":   result.Source,
		"metadata": result.Metadata,
	}, nil, nil)
	return nil
}

// --- skills.delete ---

func SkillsDeleteHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	if err := skills.DeleteSkill(name, env); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":      true,
		"name":    name,
		"deleted": true,
	}, nil, nil)
	return nil
}

// --- skills.update ---

func SkillsUpdateHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := skills.UpdateSkill(ctx, name, env)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"name":     result.Name,
		"skillDir": result.SkillDir,
		"source":   result.Source,
		"metadata": result.Metadata,
	}, nil, nil)
	return nil
}

// --- skills.bins ---

func SkillsBinsHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	workspaceDir := ""
	if opts.Context.Config != nil && opts.Context.Config.Agents != nil && opts.Context.Config.Agents.Defaults != nil {
		workspaceDir = opts.Context.Config.Agents.Defaults.Workspace
	}

	bundledFS, _ := embed.SkillsFS()
	entries, _ := skills.LoadWorkspaceEntriesWithFS(bundledFS, env, workspaceDir, nil)
	var targetSkill *skills.Entry
	for i := range entries {
		if entries[i].Name == name {
			targetSkill = &entries[i]
			break
		}
	}

	if targetSkill == nil {
		opts.Respond(false, nil, errInvalidParams("skill not found: "+name), nil)
		return nil
	}

	binStatus := skills.CheckBins(targetSkill.Metadata.Requires)

	opts.Respond(true, map[string]interface{}{
		"ok":   true,
		"name": name,
		"bins": binStatus,
	}, nil, nil)
	return nil
}

// --- skills.getConfig ---

func SkillsGetConfigHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	// Reload config from file to get the latest values
	env := envGetter()
	cfg, err := config.Load(env)
	if err != nil {
		cfg = opts.Context.Config // Fall back to in-memory config
	}

	// Get skill config from the loaded config
	var skillCfg map[string]interface{}
	if cfg != nil && cfg.Skills != nil && cfg.Skills.Entries != nil {
		if c, ok := cfg.Skills.Entries[name]; ok {
			skillCfg = map[string]interface{}{
				"enabled": c.Enabled,
				"apiKey":  c.APIKey,
				"env":     c.Env,
				"config":  c.Config,
			}
		}
	}

	if skillCfg == nil {
		skillCfg = map[string]interface{}{}
	}

	opts.Respond(true, map[string]interface{}{
		"ok":     true,
		"name":   name,
		"config": skillCfg,
	}, nil, nil)
	return nil
}

// --- skills.setConfig ---

func SkillsSetConfigHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	// Build skill config from params
	cfg := config.SkillConfig{}
	if v, ok := opts.Params["enabled"]; ok {
		if b, ok := v.(bool); ok {
			cfg.Enabled = &b
		}
	}
	if v, ok := opts.Params["apiKey"]; ok {
		if s, ok := v.(string); ok {
			cfg.APIKey = s
		}
	}
	if v, ok := opts.Params["env"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			cfg.Env = make(map[string]string)
			for k, val := range m {
				if s, ok := val.(string); ok {
					cfg.Env[k] = s
				}
			}
		}
	}
	if v, ok := opts.Params["config"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			cfg.Config = m
		}
	}

	// Update config file
	env := envGetter()
	if err := config.SetSkillConfig(name, cfg, env); err != nil {
		opts.Respond(false, nil, errInternal("failed to save config: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":     true,
		"name":   name,
		"config": cfg,
	}, nil, nil)
	return nil
}

// --- employees.list ---

func EmployeesListHandler(opts HandlerOpts) error {
	env := envGetter()
	summaries, err := employees.ListSummaries(env)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"employees": summaries,
		"count":     len(summaries),
	}, nil, nil)
	return nil
}

// --- employees.get ---

func EmployeesGetHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	env := envGetter()
	manifest, err := employees.LoadManifest(id, env)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(false, nil, errInvalidParams("employee not found: "+id), nil)
			return nil
		}
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"employee": manifest,
	}, nil, nil)
	return nil
}

// --- employees.create ---

func EmployeesCreateHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	manifest := &employees.Manifest{
		Name:        name,
		Description: stringParam(opts.Params, "description", ""),
		Prompt:      stringParam(opts.Params, "prompt", ""),
		Enabled:     true,
		From:        "local",
	}

	env := envGetter()
	if err := employees.SaveManifest(manifest, env); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"id":       manifest.ID,
		"employee": manifest,
	}, nil, nil)
	return nil
}

// --- employees.delete ---

func EmployeesDeleteHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	env := envGetter()
	if err := employees.DeleteEmployee(id, env); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":      true,
		"id":      id,
		"deleted": true,
	}, nil, nil)
	return nil
}

// --- employees.update ---

func EmployeesUpdateHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	env := envGetter()
	manifest, err := employees.LoadManifest(id, env)
	if err != nil {
		opts.Respond(false, nil, errInvalidParams("employee not found: "+id), nil)
		return nil
	}

	// Update fields
	if name := stringParam(opts.Params, "name", ""); name != "" {
		manifest.Name = name
	}
	if desc := stringParam(opts.Params, "description", ""); desc != "" {
		manifest.Description = desc
	}
	if prompt := stringParam(opts.Params, "prompt", ""); prompt != "" {
		manifest.Prompt = prompt
	}
	if v, ok := opts.Params["enabled"].(bool); ok {
		manifest.Enabled = v
	}
	if skillIds, ok := opts.Params["skillIds"].([]interface{}); ok {
		manifest.SkillIDs = nil
		for _, s := range skillIds {
			if str, ok := s.(string); ok {
				manifest.SkillIDs = append(manifest.SkillIDs, str)
			}
		}
	}

	if err := employees.SaveManifest(manifest, env); err != nil {
		opts.Respond(false, nil, errInternal("failed to update employee: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"id":       id,
		"employee": manifest,
	}, nil, nil)
	return nil
}

// --- employees.upload ---

func EmployeesUploadHandler(opts HandlerOpts) error {
	contentB64 := stringParam(opts.Params, "content", "")
	if contentB64 == "" {
		opts.Respond(false, nil, errInvalidParams("content required (base64 encoded JSON)"), nil)
		return nil
	}

	content, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		opts.Respond(false, nil, errInvalidParams("invalid base64 content: "+err.Error()), nil)
		return nil
	}

	var manifest employees.Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		opts.Respond(false, nil, errInvalidParams("invalid employee JSON: "+err.Error()), nil)
		return nil
	}

	// Validate required fields
	if manifest.Name == "" {
		opts.Respond(false, nil, errInvalidParams("name required in employee manifest"), nil)
		return nil
	}

	// Set defaults
	if manifest.ID == "" {
		// Will be generated by SaveManifest
	}
	manifest.Enabled = true
	manifest.From = "upload"

	env := envGetter()
	if err := employees.SaveManifest(&manifest, env); err != nil {
		opts.Respond(false, nil, errInternal("failed to save employee: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":       true,
		"id":       manifest.ID,
		"employee": manifest,
	}, nil, nil)
	return nil
}
