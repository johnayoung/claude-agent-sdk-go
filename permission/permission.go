package permission

// Mode controls how the agent handles tool permission requests.
type Mode string

const (
	ModeDefault           Mode = "default"
	ModeAcceptEdits       Mode = "acceptEdits"
	ModePlan              Mode = "plan"
	ModeBypassPermissions Mode = "bypassPermissions"
	ModeDontAsk           Mode = "dontAsk"
	ModeAuto              Mode = "auto"
)

// Behavior represents a permission rule behavior.
type Behavior string

const (
	BehaviorAllow Behavior = "allow"
	BehaviorDeny  Behavior = "deny"
	BehaviorAsk   Behavior = "ask"
)

// UpdateDestination identifies where a permission update is persisted.
type UpdateDestination string

const (
	DestUserSettings    UpdateDestination = "userSettings"
	DestProjectSettings UpdateDestination = "projectSettings"
	DestLocalSettings   UpdateDestination = "localSettings"
	DestSession         UpdateDestination = "session"
)

// RuleValue identifies a specific tool and optional constraint.
type RuleValue struct {
	ToolName    string `json:"tool_name"`
	RuleContent string `json:"rule_content,omitempty"`
}

// Update describes a mutation to the permission state.
type Update struct {
	Type        string            `json:"type"`
	Rules       []RuleValue       `json:"rules,omitempty"`
	Behavior    Behavior          `json:"behavior,omitempty"`
	Mode        Mode              `json:"mode,omitempty"`
	Directories []string          `json:"directories,omitempty"`
	Destination UpdateDestination `json:"destination,omitempty"`
}

// ToolContext carries contextual information for a permission decision.
type ToolContext struct {
	Signal      any      `json:"-"`
	Suggestions []Update `json:"suggestions,omitempty"`
	ToolUseID   string   `json:"tool_use_id,omitempty"`
	AgentID     string   `json:"agent_id,omitempty"`
}

// Decision is the result of a CanUseToolFunc call.
type Decision struct {
	allowed          bool
	reason           string
	interrupt        bool
	updatedInput     map[string]any
	updatedPerms     []Update
}

// Allow returns a Decision that permits tool execution.
func Allow(reason string) Decision {
	return Decision{allowed: true, reason: reason}
}

// AllowWithUpdates permits execution and applies permission updates.
func AllowWithUpdates(reason string, input map[string]any, perms []Update) Decision {
	return Decision{allowed: true, reason: reason, updatedInput: input, updatedPerms: perms}
}

// Deny returns a Decision that blocks tool execution.
func Deny(reason string) Decision {
	return Decision{allowed: false, reason: reason}
}

// DenyWithInterrupt blocks execution and interrupts the current task.
func DenyWithInterrupt(reason string) Decision {
	return Decision{allowed: false, reason: reason, interrupt: true}
}

// Allowed reports whether the decision permits execution.
func (d Decision) Allowed() bool { return d.allowed }

// Reason returns the human-readable explanation for the decision.
func (d Decision) Reason() string { return d.reason }

// Interrupt reports whether the denial should interrupt the running task.
func (d Decision) Interrupt() bool { return d.interrupt }

// UpdatedInput returns overridden tool input, or nil for no change.
func (d Decision) UpdatedInput() map[string]any { return d.updatedInput }

// UpdatedPermissions returns permission updates to apply, or nil.
func (d Decision) UpdatedPermissions() []Update { return d.updatedPerms }

// CanUseToolFunc gates tool execution. Return Allow to permit, Deny to block.
type CanUseToolFunc func(toolName string, input map[string]any, ctx ToolContext) (Decision, error)
