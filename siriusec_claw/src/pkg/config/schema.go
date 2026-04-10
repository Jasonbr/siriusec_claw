// Package config defines the full SiriuSec Claw configuration schema.
// Compatible with ~/.siriusec_claw/siriusec_claw.json.
package config

import (
	"encoding/json"
	"strings"
)

// ClawConfig is the root configuration structure.

type ClawConfig struct {
	Meta        *ConfigMeta        `json:"meta,omitempty"`
	Auth        *AuthConfig        `json:"auth,omitempty"`
	Env         *EnvConfig         `json:"env,omitempty"`
	Wizard      *WizardConfig      `json:"wizard,omitempty"`
	Diagnostics *DiagnosticsConfig `json:"diagnostics,omitempty"`
	Logging     *LoggingConfig     `json:"logging,omitempty"`
	Update      *UpdateConfig      `json:"update,omitempty"`
	Browser     *BrowserConfig     `json:"browser,omitempty"`
	UI          *UIConfig          `json:"ui,omitempty"`
	Skills      *SkillsConfig      `json:"skills,omitempty"`
	Plugins     *PluginsConfig     `json:"plugins,omitempty"`
	Models      *ModelsConfig      `json:"models,omitempty"`
	NodeHost    *NodeHostConfig    `json:"nodeHost,omitempty"`
	Agents      *AgentsConfig      `json:"agents,omitempty"`
	Tools       *ToolsConfig       `json:"tools,omitempty"`
	Bindings    []AgentBinding     `json:"bindings,omitempty"`
	Broadcast   *BroadcastConfig   `json:"broadcast,omitempty"`
	Audio       *AudioConfig       `json:"audio,omitempty"`
	Messages    *MessagesConfig    `json:"messages,omitempty"`
	Commands    *CommandsConfig    `json:"commands,omitempty"`
	Approvals   *ApprovalsConfig   `json:"approvals,omitempty"`
	Session     *SessionConfig     `json:"session,omitempty"`
	Web         *WebConfig         `json:"web,omitempty"`
	Channels    *ChannelsConfig    `json:"channels,omitempty"`
	Cron        *CronConfig        `json:"cron,omitempty"`
	Hooks       *HooksConfig       `json:"hooks,omitempty"`
	Discovery   *DiscoveryConfig   `json:"discovery,omitempty"`
	CanvasHost  *CanvasHostConfig  `json:"canvasHost,omitempty"`
	Talk        *TalkConfig        `json:"talk,omitempty"`
	Gateway     *GatewayConfig     `json:"gateway,omitempty"`
	Memory      *MemoryConfig      `json:"memory,omitempty"`
	Mcp         *McpConfig         `json:"mcp,omitempty"`
	Security    *SecurityConfig    `json:"security,omitempty"`
	Marketplace *MarketplaceConfig `json:"marketplace,omitempty"`
}

// --- Marketplace ---

type MarketplaceConfig struct {
	URL   string `json:"url,omitempty"`   // Marketplace URL, default: https://openocta.ai
	Token string `json:"token,omitempty"` // Auth token for private marketplace
}

// --- Meta ---

type ConfigMeta struct {
	LastTouchedVersion string `json:"lastTouchedVersion,omitempty"`
	LastTouchedAt      string `json:"lastTouchedAt,omitempty"`
}

// --- Logging ---

type LoggingConfig struct {
	Level           *string  `json:"level,omitempty"`
	File            string   `json:"file,omitempty"`
	ConsoleLevel    *string  `json:"consoleLevel,omitempty"`
	ConsoleStyle    *string  `json:"consoleStyle,omitempty"`
	RedactSensitive *string  `json:"redactSensitive,omitempty"`
	RedactPatterns  []string `json:"redactPatterns,omitempty"`
}

// --- Auth ---

type AuthConfig struct {
	Profiles  map[string]AuthProfileConfig `json:"profiles,omitempty"`
	Order     map[string][]string          `json:"order,omitempty"`
	Cooldowns *AuthCooldownsConfig         `json:"cooldowns,omitempty"`
}

type AuthProfileConfig struct {
	Provider string `json:"provider"`
	Mode     string `json:"mode"`
	Email    string `json:"email,omitempty"`
}

type AuthCooldownsConfig struct {
	BillingBackoffHours           *int           `json:"billingBackoffHours,omitempty"`
	BillingBackoffHoursByProvider map[string]int `json:"billingBackoffHoursByProvider,omitempty"`
	BillingMaxHours               *int           `json:"billingMaxHours,omitempty"`
	FailureWindowHours            *int           `json:"failureWindowHours,omitempty"`
}

// --- Env ---

type EnvConfig struct {
	ShellEnv *ShellEnvConfig              `json:"shellEnv,omitempty"`
	Vars     map[string]string            `json:"vars,omitempty"`
	ModelEnv map[string]map[string]string `json:"modelEnv,omitempty"`
}

type ShellEnvConfig struct {
	Enabled   *bool `json:"enabled,omitempty"`
	TimeoutMs *int  `json:"timeoutMs,omitempty"`
}

// --- Wizard ---

type WizardConfig struct {
	LastRunAt      string `json:"lastRunAt,omitempty"`
	LastRunVersion string `json:"lastRunVersion,omitempty"`
	LastRunCommit  string `json:"lastRunCommit,omitempty"`
	LastRunCommand string `json:"lastRunCommand,omitempty"`
	LastRunMode    string `json:"lastRunMode,omitempty"`
}

// --- Gateway ---

type GatewayConfig struct {
	Port           *int                    `json:"port,omitempty"`
	Mode           *string                 `json:"mode,omitempty"`
	Bind           *string                 `json:"bind,omitempty"`
	CustomBindHost *string                 `json:"customBindHost,omitempty"`
	ControlUI      *GatewayControlUIConfig `json:"controlUi,omitempty"`
	Auth           *GatewayAuthConfig      `json:"auth,omitempty"`
	Tailscale      *GatewayTailscaleConfig `json:"tailscale,omitempty"`
	Remote         *GatewayRemoteConfig    `json:"remote,omitempty"`
	Reload         *GatewayReloadConfig    `json:"reload,omitempty"`
	Tls            *GatewayTlsConfig       `json:"tls,omitempty"`
	Http           *GatewayHttpConfig      `json:"http,omitempty"`
	Nodes          *GatewayNodesConfig     `json:"nodes,omitempty"`
	TrustedProxies []string                `json:"trustedProxies,omitempty"`
	LlmTrace       *GatewayLlmTraceConfig  `json:"llmTrace,omitempty"`
}

type GatewayControlUIConfig struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	BasePath       *string  `json:"basePath,omitempty"`
	Root           *string  `json:"root,omitempty"`
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
}

type GatewayAuthConfig struct {
	Mode     *string `json:"mode,omitempty"`
	Token    string  `json:"token,omitempty"`
	Password string  `json:"password,omitempty"`
}

type GatewayTailscaleConfig struct {
	Mode        *string `json:"mode,omitempty"`
	ResetOnExit *bool   `json:"resetOnExit,omitempty"`
}

type GatewayRemoteConfig struct {
	URL            *string `json:"url,omitempty"`
	Transport      *string `json:"transport,omitempty"`
	Token          *string `json:"token,omitempty"`
	Password       *string `json:"password,omitempty"`
	TlsFingerprint *string `json:"tlsFingerprint,omitempty"`
	SshTarget      *string `json:"sshTarget,omitempty"`
	SshIdentity    *string `json:"sshIdentity,omitempty"`
}

type GatewayReloadConfig struct {
	Mode       *string `json:"mode,omitempty"`
	DebounceMs *int    `json:"debounceMs,omitempty"`
}

type GatewayTlsConfig struct {
	Enabled      *bool   `json:"enabled,omitempty"`
	AutoGenerate *bool   `json:"autoGenerate,omitempty"`
	CertPath     *string `json:"certPath,omitempty"`
	KeyPath      *string `json:"keyPath,omitempty"`
	CaPath       *string `json:"caPath,omitempty"`
}

type GatewayHttpConfig struct {
	Endpoints *GatewayHttpEndpointsConfig `json:"endpoints,omitempty"`
}

type GatewayHttpEndpointsConfig struct {
	ChatCompletions *GatewayHttpChatCompletionsConfig `json:"chatCompletions,omitempty"`
	Responses       *GatewayHttpResponsesConfig       `json:"responses,omitempty"`
}

type GatewayHttpChatCompletionsConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type GatewayHttpResponsesConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type GatewayNodesConfig struct {
	Browser       *GatewayNodesBrowserConfig `json:"browser,omitempty"`
	AllowCommands []string                   `json:"allowCommands,omitempty"`
	DenyCommands  []string                   `json:"denyCommands,omitempty"`
}

type GatewayNodesBrowserConfig struct {
	Mode *string `json:"mode,omitempty"`
	Node *string `json:"node,omitempty"`
}

type GatewayLlmTraceConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// --- Security ---

type SecurityConfig struct {
	Sandbox       *SandboxConfig          `json:"sandbox,omitempty"`
	Validator     *SandboxValidatorConfig `json:"validator,omitempty"`
	ApprovalQueue *SandboxApprovalQueue   `json:"approvalQueue,omitempty"`
	CommandPolicy *CommandPolicyConfig    `json:"commandPolicy,omitempty"`
	Preset        *string                 `json:"preset,omitempty"`
}

type SandboxConfig struct {
	Enabled       *bool                 `json:"enabled,omitempty"`
	AllowedPaths  []string              `json:"allowedPaths,omitempty"`
	NetworkAllow  []string              `json:"networkAllow,omitempty"`
	Root          *string               `json:"root,omitempty"`
	ResourceLimit *SandboxResourceLimit `json:"resourceLimit,omitempty"`
	ApprovalStore *string               `json:"approvalStore,omitempty"`
}

type SandboxResourceLimit struct {
	MaxCPUPercent  *float64 `json:"maxCpuPercent,omitempty"`
	MaxMemoryBytes *uint64  `json:"maxMemoryBytes,omitempty"`
	MaxDiskBytes   *uint64  `json:"maxDiskBytes,omitempty"`
}

type SandboxValidatorConfig struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	BanCommands    []string `json:"banCommands,omitempty"`
	BanArguments   []string `json:"banArguments,omitempty"`
	BanFragments   []string `json:"banFragments,omitempty"`
	MaxLength      *int     `json:"maxLength,omitempty"`
	SecretPatterns []string `json:"secretPatterns,omitempty"`
}

type SandboxApprovalQueue struct {
	Enabled         *bool    `json:"enabled,omitempty"`
	TimeoutSeconds  *int     `json:"timeoutSeconds,omitempty"`
	BlockOnApproval *bool    `json:"blockOnApproval,omitempty"`
	Allow           []string `json:"allow,omitempty"`
	Ask             []string `json:"ask,omitempty"`
	Deny            []string `json:"deny,omitempty"`
}

type CommandPolicyConfig struct {
	Enabled        *bool               `json:"enabled,omitempty"`
	DefaultPolicy  *string             `json:"defaultPolicy,omitempty"`
	Rules          []CommandPolicyRule `json:"rules,omitempty"`
	Deny           []string            `json:"deny,omitempty"`
	Ask            []string            `json:"ask,omitempty"`
	Allow          []string            `json:"allow,omitempty"`
	BanArguments   []string            `json:"banArguments,omitempty"`
	MaxLength      *int                `json:"maxLength,omitempty"`
	SecretPatterns []string            `json:"secretPatterns,omitempty"`
}

type CommandPolicyRule struct {
	Action  string `json:"action"`
	Pattern string `json:"pattern"`
	Type    string `json:"type"`
}

// --- MCP ---

type McpConfig struct {
	Servers            map[string]McpServerEntry `json:"servers,omitempty"`
	TimeoutSeconds     *int                      `json:"timeoutSeconds,omitempty"`
	ToolTimeoutSeconds *int                      `json:"toolTimeoutSeconds,omitempty"`
}

type McpServerEntry struct {
	Enabled    *bool             `json:"enabled,omitempty"`
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	URL        string            `json:"url,omitempty"`
	Service    string            `json:"service,omitempty"`
	ServiceURL string            `json:"serviceUrl,omitempty"`
	ToolPrefix string            `json:"toolPrefix,omitempty"`
}

// --- Models ---

type ModelsConfig struct {
	Mode      string                   `json:"mode,omitempty"`
	Providers map[string]ModelProvider `json:"providers,omitempty"`
}

type ModelProvider struct {
	BaseURL string            `json:"baseUrl"`
	APIKey  string            `json:"apiKey,omitempty"`
	Auth    *string           `json:"auth,omitempty"`
	API     *string           `json:"api,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Models  []ModelDefinition `json:"models"`
}

type ModelDefinition struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	API           *string          `json:"api,omitempty"`
	Reasoning     bool             `json:"reasoning"`
	Input         []string         `json:"input"`
	Cost          *ModelCostConfig `json:"cost,omitempty"`
	ContextWindow *int             `json:"contextWindow,omitempty"`
	MaxTokens     *int             `json:"maxTokens,omitempty"`
}

type ModelCostConfig struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// --- Agents ---

type AgentsConfig struct {
	Defaults *AgentDefaultsConfig `json:"defaults,omitempty"`
	List     []AgentConfig        `json:"list,omitempty"`
}

type AgentDefaultsConfig struct {
	Model           *AgentModelListConfig `json:"model,omitempty"`
	Workspace       string                `json:"workspace,omitempty"`
	RepoRoot        string                `json:"repoRoot,omitempty"`
	ContextTokens   *int                  `json:"contextTokens,omitempty"`
	ThinkingDefault *string               `json:"thinkingDefault,omitempty"`
	VerboseDefault  *string               `json:"verboseDefault,omitempty"`
	TimeoutSeconds  *int                  `json:"timeoutSeconds,omitempty"`
	Heartbeat       *AgentHeartbeatConfig `json:"heartbeat,omitempty"`
	MaxConcurrent   *int                  `json:"maxConcurrent,omitempty"`
	Subagents       *AgentSubagentsConfig `json:"subagents,omitempty"`
	MemorySearch    *MemorySearchConfig   `json:"memorySearch,omitempty"`
}

type AgentModelListConfig struct {
	Primary   *string  `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

type AgentHeartbeatConfig struct {
	Every   *string `json:"every,omitempty"`
	Model   *string `json:"model,omitempty"`
	Session *string `json:"session,omitempty"`
	Prompt  *string `json:"prompt,omitempty"`
}

type AgentSubagentsConfig struct {
	MaxConcurrent *int        `json:"maxConcurrent,omitempty"`
	Model         interface{} `json:"model,omitempty"`
	Thinking      *string     `json:"thinking,omitempty"`
}

type AgentConfig struct {
	ID        string      `json:"id"`
	Default   *bool       `json:"default,omitempty"`
	Name      string      `json:"name,omitempty"`
	Workspace string      `json:"workspace,omitempty"`
	AgentDir  string      `json:"agentDir,omitempty"`
	Model     interface{} `json:"model,omitempty"`
	Skills    []string    `json:"skills,omitempty"`
}

type AgentBinding struct {
	AgentID string            `json:"agentId"`
	Match   AgentBindingMatch `json:"match"`
}

type AgentBindingMatch struct {
	Channel   string  `json:"channel"`
	AccountID *string `json:"accountId,omitempty"`
}

// --- Skills ---

type SkillsConfig struct {
	AllowBundled []string               `json:"allowBundled,omitempty"`
	Load         *SkillsLoadConfig      `json:"load,omitempty"`
	Install      *SkillsInstallConfig   `json:"install,omitempty"`
	Entries      map[string]SkillConfig `json:"entries,omitempty"`
}

type SkillsLoadConfig struct {
	ExtraDirs       []string `json:"extraDirs,omitempty"`
	Watch           *bool    `json:"watch,omitempty"`
	WatchDebounceMs *int     `json:"watchDebounceMs,omitempty"`
}

type SkillsInstallConfig struct {
	PreferBrew  *bool   `json:"preferBrew,omitempty"`
	NodeManager *string `json:"nodeManager,omitempty"`
}

type SkillConfig struct {
	Enabled *bool                  `json:"enabled,omitempty"`
	APIKey  string                 `json:"apiKey,omitempty"`
	Env     map[string]string      `json:"env,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// --- Plugins ---

type PluginsConfig struct {
	Enabled *bool                  `json:"enabled,omitempty"`
	Allow   []string               `json:"allow,omitempty"`
	Deny    []string               `json:"deny,omitempty"`
	Entries map[string]interface{} `json:"entries,omitempty"`
}

// --- Tools ---

type ToolsConfig struct {
	Profile *string          `json:"profile,omitempty"`
	Allow   []string         `json:"allow,omitempty"`
	Deny    []string         `json:"deny,omitempty"`
	Web     *ToolsWebConfig  `json:"web,omitempty"`
	Exec    *ToolsExecConfig `json:"exec,omitempty"`
}

type ToolsWebConfig struct {
	Search *ToolsWebSearchConfig `json:"search,omitempty"`
	Fetch  *ToolsWebFetchConfig  `json:"fetch,omitempty"`
}

type ToolsWebSearchConfig struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	Provider   *string `json:"provider,omitempty"`
	APIKey     *string `json:"apiKey,omitempty"`
	MaxResults *int    `json:"maxResults,omitempty"`
}

type ToolsWebFetchConfig struct {
	Enabled        *bool `json:"enabled,omitempty"`
	MaxChars       *int  `json:"maxChars,omitempty"`
	TimeoutSeconds *int  `json:"timeoutSeconds,omitempty"`
}

type ToolsExecConfig struct {
	Host     *string  `json:"host,omitempty"`
	Security *string  `json:"security,omitempty"`
	SafeBins []string `json:"safeBins,omitempty"`
}

// --- Channels ---

type ChannelsConfig struct {
	Defaults *ChannelDefaultsConfig `json:"defaults,omitempty"`
	WhatsApp map[string]interface{} `json:"whatsapp,omitempty"`
	Telegram map[string]interface{} `json:"telegram,omitempty"`
	Discord  map[string]interface{} `json:"discord,omitempty"`
	Slack    map[string]interface{} `json:"slack,omitempty"`
	DingTalk map[string]interface{} `json:"dingtalk,omitempty"`
	Feishu   map[string]interface{} `json:"feishu,omitempty"`
	QQ       map[string]interface{} `json:"qq,omitempty"`
	WeWork   map[string]interface{} `json:"wework,omitempty"`
	Weixin   map[string]interface{} `json:"weixin,omitempty"`
	Plugins  map[string]interface{} `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for ChannelsConfig.
func (c *ChannelsConfig) UnmarshalJSON(data []byte) error {
	type ChannelsConfigAlias ChannelsConfig
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	alias := (*ChannelsConfigAlias)(c)
	if err := json.Unmarshal(data, alias); err != nil {
		return err
	}
	if c.Plugins == nil {
		c.Plugins = make(map[string]interface{})
	}
	knownChannels := map[string]bool{
		"defaults": true, "whatsapp": true, "telegram": true, "discord": true,
		"slack": true, "dingtalk": true, "feishu": true, "qq": true,
		"wework": true, "weixin": true,
	}
	for key, value := range raw {
		keyLower := strings.ToLower(key)
		if !knownChannels[keyLower] {
			var cfg interface{}
			if err := json.Unmarshal(value, &cfg); err == nil {
				c.Plugins[keyLower] = cfg
			}
		}
	}
	return nil
}

// GetChannelConfig returns the configuration for a channel by name.
func (c *ChannelsConfig) GetChannelConfig(name string) map[string]interface{} {
	if c == nil {
		return nil
	}
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "whatsapp":
		return c.WhatsApp
	case "telegram":
		return c.Telegram
	case "discord":
		return c.Discord
	case "slack":
		return c.Slack
	case "dingtalk":
		return c.DingTalk
	case "feishu":
		return c.Feishu
	case "qq":
		return c.QQ
	case "wework":
		return c.WeWork
	case "weixin":
		return c.Weixin
	default:
		if c.Plugins != nil {
			if cfg, ok := c.Plugins[name].(map[string]interface{}); ok {
				return cfg
			}
		}
		return nil
	}
}

type ChannelDefaultsConfig struct {
	GroupPolicy *string `json:"groupPolicy,omitempty"`
}

// --- Cron ---

type CronConfig struct {
	Enabled           *bool   `json:"enabled,omitempty"`
	Store             *string `json:"store,omitempty"`
	MaxConcurrentRuns *int    `json:"maxConcurrentRuns,omitempty"`
}

// --- Hooks ---

type HooksConfig struct {
	Enabled      *bool               `json:"enabled,omitempty"`
	Path         *string             `json:"path,omitempty"`
	Token        *string             `json:"token,omitempty"`
	MaxBodyBytes *int                `json:"maxBodyBytes,omitempty"`
	Presets      []string            `json:"presets,omitempty"`
	Mappings     []HookMappingConfig `json:"mappings,omitempty"`
}

type HookMappingConfig struct {
	ID              *string           `json:"id,omitempty"`
	Match           *HookMappingMatch `json:"match,omitempty"`
	Action          *string           `json:"action,omitempty"`
	WakeMode        *string           `json:"wakeMode,omitempty"`
	SessionKey      *string           `json:"sessionKey,omitempty"`
	MessageTemplate *string           `json:"messageTemplate,omitempty"`
	Channel         *string           `json:"channel,omitempty"`
	To              *string           `json:"to,omitempty"`
	TimeoutSeconds  *int              `json:"timeoutSeconds,omitempty"`
}

type HookMappingMatch struct {
	Path   *string `json:"path,omitempty"`
	Source *string `json:"source,omitempty"`
}

// --- Memory ---

type MemoryConfig struct {
	Backend   *string `json:"backend,omitempty"`
	Citations *string `json:"citations,omitempty"`
}

type MemorySearchConfig struct {
	Enabled    *bool    `json:"enabled,omitempty"`
	Sources    []string `json:"sources,omitempty"`
	ExtraPaths []string `json:"extraPaths,omitempty"`
	Provider   *string  `json:"provider,omitempty"`
	Model      *string  `json:"model,omitempty"`
}

// --- Session ---

type SessionConfig struct {
	Scope          *string               `json:"scope,omitempty"`
	DmScope        *string               `json:"dmScope,omitempty"`
	ResetTriggers  []string              `json:"resetTriggers,omitempty"`
	IdleMinutes    *int                  `json:"idleMinutes,omitempty"`
	Reset          *SessionResetConfig   `json:"reset,omitempty"`
	Store          *string               `json:"store,omitempty"`
	MainKey        *string               `json:"mainKey,omitempty"`
	SessionHistory *SessionHistoryConfig `json:"sessionHistory,omitempty"`
}

type SessionResetConfig struct {
	Mode        *string `json:"mode,omitempty"`
	AtHour      *int    `json:"atHour,omitempty"`
	IdleMinutes *int    `json:"idleMinutes,omitempty"`
}

type SessionHistoryConfig struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	MaxMessages        *int     `json:"maxMessages,omitempty"`
	Roles              []string `json:"roles,omitempty"`
	LoadFromTranscript *bool    `json:"loadFromTranscript,omitempty"`
}

// --- Misc top-level configs ---

type DiagnosticsConfig struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Flags   []string `json:"flags,omitempty"`
}

type UpdateConfig struct {
	Channel      *string `json:"channel,omitempty"`
	CheckOnStart *bool   `json:"checkOnStart,omitempty"`
}

type BrowserConfig struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	CdpURL         *string `json:"cdpUrl,omitempty"`
	ExecutablePath *string `json:"executablePath,omitempty"`
	Headless       *bool   `json:"headless,omitempty"`
}

type UIConfig struct {
	SeamColor *string `json:"seamColor,omitempty"`
}

type NodeHostConfig struct {
	BrowserProxy *NodeHostBrowserProxyConfig `json:"browserProxy,omitempty"`
}

type NodeHostBrowserProxyConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type BroadcastConfig struct {
	Strategy *string `json:"strategy,omitempty"`
}

type AudioConfig struct {
	Transcription *AudioTranscriptionConfig `json:"transcription,omitempty"`
}

type AudioTranscriptionConfig struct {
	Command        []string `json:"command"`
	TimeoutSeconds *int     `json:"timeoutSeconds,omitempty"`
}

type MessagesConfig struct {
	MessagePrefix  *string `json:"messagePrefix,omitempty"`
	ResponsePrefix *string `json:"responsePrefix,omitempty"`
	AckReaction    *string `json:"ackReaction,omitempty"`
}

type CommandsConfig struct {
	Native *bool `json:"native,omitempty"`
	Text   *bool `json:"text,omitempty"`
	Bash   *bool `json:"bash,omitempty"`
	Config *bool `json:"config,omitempty"`
}

type ApprovalsConfig struct {
	Exec *ExecApprovalConfig `json:"exec,omitempty"`
}

type ExecApprovalConfig struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Mode    *string `json:"mode,omitempty"`
}

type WebConfig struct {
	Enabled          *bool `json:"enabled,omitempty"`
	HeartbeatSeconds *int  `json:"heartbeatSeconds,omitempty"`
}

type DiscoveryConfig struct {
	Mdns *MdnsDiscoveryConfig `json:"mdns,omitempty"`
}

type MdnsDiscoveryConfig struct {
	Mode *string `json:"mode,omitempty"`
}

type CanvasHostConfig struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	Root       *string `json:"root,omitempty"`
	Port       *int    `json:"port,omitempty"`
	LiveReload *bool   `json:"liveReload,omitempty"`
}

type TalkConfig struct {
	VoiceID *string `json:"voiceId,omitempty"`
	ModelID *string `json:"modelId,omitempty"`
	APIKey  *string `json:"apiKey,omitempty"`
}
