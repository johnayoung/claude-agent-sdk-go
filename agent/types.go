package agent

// RateLimitInfo contains rate limit status from the API.
type RateLimitInfo struct {
	RequestsLimit     int   `json:"requests_limit"`
	RequestsRemaining int   `json:"requests_remaining"`
	RequestsReset     int64 `json:"requests_reset"`
	TokensLimit       int   `json:"tokens_limit"`
	TokensRemaining   int   `json:"tokens_remaining"`
	TokensReset       int64 `json:"tokens_reset"`
}

// ContextUsage tracks token usage within a conversation.
type ContextUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read_tokens"`
	CacheWrite   int `json:"cache_write_tokens"`
}

// McpServerStatus represents the connection status of an MCP server.
type McpServerStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	NumTools  int    `json:"num_tools"`
	ToolNames []string `json:"tool_names,omitempty"`
}

// AgentDefinition describes a sub-agent configuration.
type AgentDefinition struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Model        string            `json:"model,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Tools        []string          `json:"tools,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SandboxConfig describes sandbox execution settings.
type SandboxConfig struct {
	Type       string   `json:"type"`
	AllowNet   bool     `json:"allow_net"`
	AllowRead  []string `json:"allow_read,omitempty"`
	AllowWrite []string `json:"allow_write,omitempty"`
	AllowExec  []string `json:"allow_exec,omitempty"`
}
