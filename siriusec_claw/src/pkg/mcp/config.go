package mcp

import (
	"github.com/siriusec/siriusec_claw/pkg/config"
)

// ServerSpec represents a resolved MCP server specification.
type ServerSpec struct {
	Name       string            `json:"name"`
	Enabled    bool              `json:"enabled"`
	Transport  string            `json:"transport"` // "stdio", "sse", "http"
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	URL        string            `json:"url,omitempty"`
	ToolPrefix string            `json:"toolPrefix,omitempty"`
}

// BuildServerSpecs converts MCP config entries to resolved server specs.
func BuildServerSpecs(mcpCfg *config.McpConfig) []ServerSpec {
	if mcpCfg == nil || mcpCfg.Servers == nil {
		return nil
	}

	var specs []ServerSpec
	for name, entry := range mcpCfg.Servers {
		enabled := true
		if entry.Enabled != nil {
			enabled = *entry.Enabled
		}
		if !enabled {
			continue
		}

		spec := ServerSpec{
			Name:       name,
			Enabled:    enabled,
			ToolPrefix: entry.ToolPrefix,
			Env:        entry.Env,
		}

		if entry.Command != "" {
			spec.Transport = "stdio"
			spec.Command = entry.Command
			spec.Args = entry.Args
		} else if entry.URL != "" {
			spec.Transport = "sse"
			spec.URL = entry.URL
		} else if entry.Service != "" {
			spec.Transport = "stdio"
			spec.Command, spec.Args = resolveServiceCommand(entry.Service, entry.ServiceURL)
		}

		specs = append(specs, spec)
	}
	return specs
}

// ListServerNames returns the names of all enabled MCP servers.
func ListServerNames(mcpCfg *config.McpConfig) []string {
	specs := BuildServerSpecs(mcpCfg)
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		names = append(names, s.Name)
	}
	return names
}

func resolveServiceCommand(service, serviceURL string) (string, []string) {
	// Map well-known service names to their npx commands
	switch service {
	case "prometheus":
		args := []string{"-y", "prometheus-mcp-server"}
		if serviceURL != "" {
			args = append(args, "--url", serviceURL)
		}
		return "npx", args
	case "grafana":
		args := []string{"-y", "grafana-mcp-server"}
		if serviceURL != "" {
			args = append(args, "--url", serviceURL)
		}
		return "npx", args
	default:
		return "npx", []string{"-y", service + "-mcp-server"}
	}
}
