package claude

import "encoding/json"

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

// AgentDefinition describes a sub-agent configuration.
type AgentDefinition struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Prompt          string            `json:"prompt,omitempty"`
	Model           string            `json:"model,omitempty"`
	Instructions    string            `json:"instructions,omitempty"`
	Tools           []string          `json:"tools,omitempty"`
	DisallowedTools []string          `json:"disallowed_tools,omitempty"`
	Skills          []string          `json:"skills,omitempty"`
	Memory          string            `json:"memory,omitempty"`
	MCPServers      []string          `json:"mcp_servers,omitempty"`
	InitialPrompt   string            `json:"initial_prompt,omitempty"`
	MaxTurns        int               `json:"max_turns,omitempty"`
	Background      bool              `json:"background,omitempty"`
	Effort          string            `json:"effort,omitempty"`
	PermissionMode  string            `json:"permission_mode,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// SandboxConfig describes sandbox execution settings.
type SandboxConfig struct {
	Type       string   `json:"type"`
	AllowNet   bool     `json:"allow_net"`
	AllowRead  []string `json:"allow_read,omitempty"`
	AllowWrite []string `json:"allow_write,omitempty"`
	AllowExec  []string `json:"allow_exec,omitempty"`
}
