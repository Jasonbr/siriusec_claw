package handlers

import (
	"time"

	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/mcp"
)

// --- mcp.list ---

func MCPListHandler(opts HandlerOpts) error {
	env := envGetter()
	servers, err := mcp.ListServers(env)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	// Convert to list format with metadata
	result := make([]map[string]interface{}, 0)
	for name, entry := range servers {
		item := map[string]interface{}{
			"name":    name,
			"enabled": entry.Enabled == nil || *entry.Enabled,
		}
		if entry.Command != "" {
			item["transport"] = "stdio"
			item["command"] = entry.Command
			item["args"] = entry.Args
		}
		if entry.URL != "" {
			item["transport"] = "sse"
			item["url"] = entry.URL
		}
		if entry.Service != "" {
			item["service"] = entry.Service
		}
		if entry.ToolPrefix != "" {
			item["toolPrefix"] = entry.ToolPrefix
		}

		// Check for managed manifest
		if manifest, err := mcp.LoadManagedManifest(name, env); err == nil {
			item["description"] = manifest.Description
			item["author"] = manifest.Author
			item["source"] = manifest.Source
			item["installedAt"] = manifest.InstalledAt
		}

		result = append(result, item)
	}

	opts.Respond(true, map[string]interface{}{
		"servers": result,
		"count":   len(result),
	}, nil, nil)
	return nil
}

// --- mcp.get ---

func MCPGetHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	entry, ok := mcp.GetServer(name, env)
	if !ok {
		opts.Respond(false, nil, errInvalidParams("MCP server not found: "+name), nil)
		return nil
	}

	result := map[string]interface{}{
		"name":    name,
		"enabled": entry.Enabled == nil || *entry.Enabled,
	}
	if entry.Command != "" {
		result["transport"] = "stdio"
		result["command"] = entry.Command
		result["args"] = entry.Args
	}
	if entry.URL != "" {
		result["transport"] = "sse"
		result["url"] = entry.URL
	}
	if entry.Env != nil {
		result["env"] = entry.Env
	}
	if entry.ToolPrefix != "" {
		result["toolPrefix"] = entry.ToolPrefix
	}

	// Load managed manifest for metadata
	if manifest, err := mcp.LoadManagedManifest(name, env); err == nil {
		result["description"] = manifest.Description
		result["author"] = manifest.Author
		result["homepage"] = manifest.Homepage
		result["source"] = manifest.Source
		result["installedAt"] = manifest.InstalledAt
	}

	opts.Respond(true, map[string]interface{}{
		"server": result,
	}, nil, nil)
	return nil
}

// --- mcp.add ---

func MCPAddHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	entry := config.McpServerEntry{}

	// Parse transport type
	transport := stringParam(opts.Params, "transport", "stdio")
	if transport == "stdio" {
		entry.Command = stringParam(opts.Params, "command", "")
		if entry.Command == "" {
			opts.Respond(false, nil, errInvalidParams("command required for stdio transport"), nil)
			return nil
		}
		if args, ok := opts.Params["args"].([]interface{}); ok {
			for _, a := range args {
				if s, ok := a.(string); ok {
					entry.Args = append(entry.Args, s)
				}
			}
		}
	} else if transport == "sse" || transport == "http" {
		entry.URL = stringParam(opts.Params, "url", "")
		if entry.URL == "" {
			opts.Respond(false, nil, errInvalidParams("url required for sse transport"), nil)
			return nil
		}
	}

	// Optional fields
	if v, ok := opts.Params["enabled"].(bool); ok {
		entry.Enabled = &v
	}
	if env, ok := opts.Params["env"].(map[string]interface{}); ok {
		entry.Env = make(map[string]string)
		for k, v := range env {
			if s, ok := v.(string); ok {
				entry.Env[k] = s
			}
		}
	}
	entry.ToolPrefix = stringParam(opts.Params, "toolPrefix", "")

	env := envGetter()
	if err := mcp.AddServer(name, entry, env); err != nil {
		opts.Respond(false, nil, errInternal("failed to add MCP server: "+err.Error()), nil)
		return nil
	}

	// Save manifest with metadata
	manifest := &mcp.ServerManifest{
		Name:        name,
		Description: stringParam(opts.Params, "description", ""),
		Author:      stringParam(opts.Params, "author", ""),
		Homepage:    stringParam(opts.Params, "homepage", ""),
		Source:      "upload",
		InstalledAt: time.Now().UnixMilli(),
		Config:      &entry,
	}
	mcp.SaveManagedManifest(name, manifest, env)

	opts.Respond(true, map[string]interface{}{
		"ok":     true,
		"name":   name,
		"server": entry,
	}, nil, nil)
	return nil
}

// --- mcp.update ---

func MCPUpdateHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	existing, ok := mcp.GetServer(name, env)
	if !ok {
		opts.Respond(false, nil, errInvalidParams("MCP server not found: "+name), nil)
		return nil
	}

	// Update fields
	if v, ok := opts.Params["enabled"].(bool); ok {
		existing.Enabled = &v
	}
	if cmd := stringParam(opts.Params, "command", ""); cmd != "" {
		existing.Command = cmd
	}
	if url := stringParam(opts.Params, "url", ""); url != "" {
		existing.URL = url
	}
	if args, ok := opts.Params["args"].([]interface{}); ok {
		existing.Args = nil
		for _, a := range args {
			if s, ok := a.(string); ok {
				existing.Args = append(existing.Args, s)
			}
		}
	}
	if envMap, ok := opts.Params["env"].(map[string]interface{}); ok {
		existing.Env = make(map[string]string)
		for k, v := range envMap {
			if s, ok := v.(string); ok {
				existing.Env[k] = s
			}
		}
	}
	if toolPrefix := stringParam(opts.Params, "toolPrefix", ""); toolPrefix != "" {
		existing.ToolPrefix = toolPrefix
	}

	if err := mcp.AddServer(name, *existing, env); err != nil {
		opts.Respond(false, nil, errInternal("failed to update MCP server: "+err.Error()), nil)
		return nil
	}

	// Update manifest
	if manifest, err := mcp.LoadManagedManifest(name, env); err == nil {
		manifest.UpdatedAt = time.Now().UnixMilli()
		if desc := stringParam(opts.Params, "description", ""); desc != "" {
			manifest.Description = desc
		}
		mcp.SaveManagedManifest(name, manifest, env)
	}

	opts.Respond(true, map[string]interface{}{
		"ok":     true,
		"name":   name,
		"server": existing,
	}, nil, nil)
	return nil
}

// --- mcp.delete ---

func MCPDeleteHandler(opts HandlerOpts) error {
	name := stringParam(opts.Params, "name", "")
	if name == "" {
		opts.Respond(false, nil, errInvalidParams("name required"), nil)
		return nil
	}

	env := envGetter()
	if err := mcp.RemoveServer(name, env); err != nil {
		opts.Respond(false, nil, errInternal("failed to delete MCP server: "+err.Error()), nil)
		return nil
	}

	// Also remove managed manifest
	mcp.DeleteManagedServer(name, env)

	opts.Respond(true, map[string]interface{}{
		"ok":      true,
		"name":    name,
		"deleted": true,
	}, nil, nil)
	return nil
}
