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

// Decision is the result of a CanUseToolFunc call.
type Decision struct {
	allowed bool
	reason  string
}

// Allow returns a Decision that permits tool execution.
func Allow(reason string) Decision {
	return Decision{allowed: true, reason: reason}
}

// Deny returns a Decision that blocks tool execution.
func Deny(reason string) Decision {
	return Decision{allowed: false, reason: reason}
}

// Allowed reports whether the decision permits execution.
func (d Decision) Allowed() bool { return d.allowed }

// Reason returns the human-readable explanation for the decision.
func (d Decision) Reason() string { return d.reason }

// CanUseToolFunc gates tool execution. Return Allow to permit, Deny to block.
type CanUseToolFunc func(toolName string, input map[string]any) (Decision, error)
