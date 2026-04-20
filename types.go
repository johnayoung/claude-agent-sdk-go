package claude

import "encoding/json"

// TaskNotificationStatus represents the terminal status of a background task.
type TaskNotificationStatus string

const (
	TaskNotificationCompleted TaskNotificationStatus = "completed"
	TaskNotificationFailed    TaskNotificationStatus = "failed"
	TaskNotificationStopped   TaskNotificationStatus = "stopped"
)

// AssistantMessageError categorizes errors surfaced in assistant messages.
type AssistantMessageError string

const (
	AssistantErrorAuthFailed     AssistantMessageError = "authentication_failed"
	AssistantErrorBilling        AssistantMessageError = "billing_error"
	AssistantErrorRateLimit      AssistantMessageError = "rate_limit"
	AssistantErrorInvalidRequest AssistantMessageError = "invalid_request"
	AssistantErrorServer         AssistantMessageError = "server_error"
	AssistantErrorUnknown        AssistantMessageError = "unknown"
)

// RateLimitStatus represents the rate limit state.
type RateLimitStatus string

const (
	RateLimitAllowed        RateLimitStatus = "allowed"
	RateLimitAllowedWarning RateLimitStatus = "allowed_warning"
	RateLimitRejected       RateLimitStatus = "rejected"
)

// RateLimitType identifies which rate limit window applies.
type RateLimitType string

const (
	RateLimitFiveHour    RateLimitType = "five_hour"
	RateLimitSevenDay    RateLimitType = "seven_day"
	RateLimitSevenDayOps RateLimitType = "seven_day_opus"
	RateLimitSevenDaySon RateLimitType = "seven_day_sonnet"
	RateLimitOverage     RateLimitType = "overage"
)

// SdkBeta identifies SDK beta feature flags.
type SdkBeta = string

const (
	BetaContext1M SdkBeta = "context-1m-2025-08-07"
)

// SettingSource identifies which settings file to load.
type SettingSource = string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// RateLimitInfo contains rate limit status emitted by the CLI.
type RateLimitInfo struct {
	Status                RateLimitStatus `json:"status"`
	ResetsAt              *int64          `json:"resetsAt,omitempty"`
	RateLimitType         RateLimitType   `json:"rateLimitType,omitempty"`
	Utilization           *float64        `json:"utilization,omitempty"`
	OverageStatus         RateLimitStatus `json:"overageStatus,omitempty"`
	OverageResetsAt       *int64          `json:"overageResetsAt,omitempty"`
	OverageDisabledReason string          `json:"overageDisabledReason,omitempty"`
	Raw                   json.RawMessage `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling to preserve the full raw dict.
func (r *RateLimitInfo) UnmarshalJSON(data []byte) error {
	type Alias RateLimitInfo
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = RateLimitInfo(a)
	r.Raw = make(json.RawMessage, len(data))
	copy(r.Raw, data)
	return nil
}

// TaskUsage tracks resource consumption for a background task.
type TaskUsage struct {
	TotalTokens int   `json:"total_tokens"`
	ToolUses    int   `json:"tool_uses"`
	DurationMS  int64 `json:"duration_ms"`
}

// TaskBudget constrains resource usage for a query.
type TaskBudget struct {
	Total int `json:"total"`
}

// AgentDefinition describes a sub-agent configuration.
type AgentDefinition struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Prompt          string            `json:"prompt,omitempty"`
	Model           string            `json:"model,omitempty"`
	Tools           []string          `json:"tools,omitempty"`
	DisallowedTools []string          `json:"disallowedTools,omitempty"`
	Skills          []string          `json:"skills,omitempty"`
	Memory          string            `json:"memory,omitempty"`
	MCPServers      []any             `json:"mcpServers,omitempty"`
	InitialPrompt   string            `json:"initialPrompt,omitempty"`
	MaxTurns        *int              `json:"maxTurns,omitempty"`
	Background      *bool             `json:"background,omitempty"`
	Effort          string            `json:"effort,omitempty"`
	PermissionMode  string            `json:"permissionMode,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// SandboxNetworkConfig controls network access within the sandbox.
type SandboxNetworkConfig struct {
	AllowUnixSockets    []string `json:"allowUnixSockets,omitempty"`
	AllowAllUnixSockets bool     `json:"allowAllUnixSockets,omitempty"`
	AllowLocalBinding   bool     `json:"allowLocalBinding,omitempty"`
	HTTPProxyPort       int      `json:"httpProxyPort,omitempty"`
	SOCKSProxyPort      int      `json:"socksProxyPort,omitempty"`
}

// SandboxIgnoreViolations specifies violation types to suppress.
type SandboxIgnoreViolations struct {
	File    []string `json:"file,omitempty"`
	Network []string `json:"network,omitempty"`
}

// SandboxConfig describes sandbox execution settings.
type SandboxConfig struct {
	Enabled                   *bool                    `json:"enabled,omitempty"`
	AutoAllowBashIfSandboxed  *bool                    `json:"autoAllowBashIfSandboxed,omitempty"`
	ExcludedCommands          []string                 `json:"excludedCommands,omitempty"`
	AllowUnsandboxedCommands  *bool                    `json:"allowUnsandboxedCommands,omitempty"`
	Network                   *SandboxNetworkConfig    `json:"network,omitempty"`
	IgnoreViolations          *SandboxIgnoreViolations `json:"ignoreViolations,omitempty"`
	EnableWeakerNestedSandbox *bool                    `json:"enableWeakerNestedSandbox,omitempty"`
}

// SdkPluginConfig identifies a local SDK plugin.
type SdkPluginConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// SDKSessionInfo describes a stored session.
type SDKSessionInfo struct {
	SessionID    string `json:"session_id"`
	Summary      string `json:"summary"`
	LastModified int64  `json:"last_modified"`
	FileSize     *int64 `json:"file_size,omitempty"`
	CustomTitle  string `json:"custom_title,omitempty"`
	FirstPrompt  string `json:"first_prompt,omitempty"`
	GitBranch    string `json:"git_branch,omitempty"`
	Cwd          string `json:"cwd,omitempty"`
	Tag          string `json:"tag,omitempty"`
	CreatedAt    *int64 `json:"created_at,omitempty"`
}

// SessionMessage represents a single message within a session transcript.
type SessionMessage struct {
	Type            string `json:"type"`
	UUID            string `json:"uuid"`
	SessionID       string `json:"session_id"`
	Message         any    `json:"message"`
	ParentToolUseID string `json:"parent_tool_use_id,omitempty"`
}

// ForkSessionResult holds the output of a session fork operation.
type ForkSessionResult struct {
	SessionID string `json:"session_id"`
}

// McpToolAnnotations describes behavioral annotations for an MCP tool.
type McpToolAnnotations struct {
	ReadOnly    *bool `json:"readOnly,omitempty"`
	Destructive *bool `json:"destructive,omitempty"`
	OpenWorld   *bool `json:"openWorld,omitempty"`
}

// McpToolInfo describes a tool provided by an MCP server.
type McpToolInfo struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Annotations *McpToolAnnotations `json:"annotations,omitempty"`
}

// McpServerInfo identifies an MCP server implementation.
type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// McpServerConnectionStatus represents the connection state of an MCP server.
type McpServerConnectionStatus string

const (
	McpStatusConnected McpServerConnectionStatus = "connected"
	McpStatusFailed    McpServerConnectionStatus = "failed"
	McpStatusNeedsAuth McpServerConnectionStatus = "needs-auth"
	McpStatusPending   McpServerConnectionStatus = "pending"
	McpStatusDisabled  McpServerConnectionStatus = "disabled"
)

// McpServerStatus reports the runtime state of a single MCP server.
type McpServerStatus struct {
	Name       string                    `json:"name"`
	Status     McpServerConnectionStatus `json:"status"`
	ServerInfo *McpServerInfo            `json:"serverInfo,omitempty"`
	Error      string                    `json:"error,omitempty"`
	Config     map[string]any            `json:"config,omitempty"`
	Scope      string                    `json:"scope,omitempty"`
	Tools      []McpToolInfo             `json:"tools,omitempty"`
}

// McpStatusResponse is the response from querying MCP server status.
type McpStatusResponse struct {
	MCPServers []McpServerStatus `json:"mcpServers"`
}

// ContextUsageCategory describes token usage for a single context category.
type ContextUsageCategory struct {
	Name       string `json:"name"`
	Tokens     int    `json:"tokens"`
	Color      string `json:"color"`
	IsDeferred bool   `json:"isDeferred,omitempty"`
}

// ContextUsageResponse reports overall context window utilization.
type ContextUsageResponse struct {
	Categories               []ContextUsageCategory   `json:"categories"`
	TotalTokens              int                      `json:"totalTokens"`
	MaxTokens                int                      `json:"maxTokens"`
	RawMaxTokens             int                      `json:"rawMaxTokens"`
	Percentage               float64                  `json:"percentage"`
	Model                    string                   `json:"model"`
	IsAutoCompactEnabled     bool                     `json:"isAutoCompactEnabled"`
	MemoryFiles              []map[string]any         `json:"memoryFiles"`
	McpTools                 []map[string]any         `json:"mcpTools"`
	Agents                   []map[string]any         `json:"agents"`
	GridRows                 [][]map[string]any       `json:"gridRows"`
	AutoCompactThreshold     *int                     `json:"autoCompactThreshold,omitempty"`
	DeferredBuiltinTools     []map[string]any         `json:"deferredBuiltinTools,omitempty"`
	SystemTools              []map[string]any         `json:"systemTools,omitempty"`
	SystemPromptSections     []map[string]any         `json:"systemPromptSections,omitempty"`
	SlashCommands            map[string]any           `json:"slashCommands,omitempty"`
	Skills                   map[string]any           `json:"skills,omitempty"`
	MessageBreakdown         map[string]any           `json:"messageBreakdown,omitempty"`
	APIUsage                 map[string]any           `json:"apiUsage,omitempty"`
}
