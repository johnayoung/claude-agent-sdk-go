package hooks

import (
	"context"
	"path"
)

// HookEvent identifies which lifecycle event occurred.
type HookEvent string

const (
	EventPreToolUse          HookEvent = "pre_tool_use"
	EventPostToolUse         HookEvent = "post_tool_use"
	EventPostToolUseFailure  HookEvent = "post_tool_use_failure"
	EventModelResponse       HookEvent = "model_response"
	EventNotificationArrived HookEvent = "notification_arrived"
	EventStop                HookEvent = "stop"
	EventSubagentStarted     HookEvent = "subagent_started"
	EventSubagentStopped     HookEvent = "subagent_stopped"
	EventSessionStarted      HookEvent = "session_started"
	EventSessionStopped      HookEvent = "session_stopped"
	EventUserPromptSubmit    HookEvent = "user_prompt_submit"
	EventPermissionRequest   HookEvent = "permission_request"
	EventPreCompact          HookEvent = "pre_compact"
	EventError               HookEvent = "error"
)

// PreToolUseInput carries context for a tool call before it executes.
type PreToolUseInput struct {
	ToolName  string
	ToolInput map[string]any
	SessionID string
}

// PreToolUseOutput allows modifying or blocking the tool call.
type PreToolUseOutput struct {
	// ToolInput overrides the input passed to the tool; nil means no change.
	ToolInput map[string]any
	Block     bool
	Reason    string
}

// PostToolUseInput carries context for a tool call after it executes.
type PostToolUseInput struct {
	ToolName   string
	ToolInput  map[string]any
	ToolOutput string
	IsError    bool
	SessionID  string
}

// PostToolUseOutput is returned after a PostToolUse handler runs.
type PostToolUseOutput struct {
	SuppressOutput bool
}

// PostToolUseFailureInput carries context for a tool call that produced an error.
type PostToolUseFailureInput struct {
	ToolName   string
	ToolInput  map[string]any
	ToolOutput string
	Error      string
	SessionID  string
}

// PostToolUseFailureOutput is returned after a PostToolUseFailure handler runs.
type PostToolUseFailureOutput struct {
	SuppressOutput bool
}

// ModelResponseInput carries a text response from the model.
type ModelResponseInput struct {
	Response  string
	SessionID string
}

// ModelResponseOutput is returned after a ModelResponse handler runs.
type ModelResponseOutput struct{}

// NotificationArrivedInput carries a task notification.
type NotificationArrivedInput struct {
	Title     string
	Message   string
	SessionID string
}

// NotificationArrivedOutput is returned after a NotificationArrived handler runs.
type NotificationArrivedOutput struct{}

// StopInput carries the reason the agent stopped.
type StopInput struct {
	Reason    string
	SessionID string
}

// StopOutput is returned after a Stop handler runs.
type StopOutput struct {
	StopReason string
}

// UserPromptSubmitInput carries context when a user submits a prompt.
type UserPromptSubmitInput struct {
	Prompt    string
	SessionID string
}

// UserPromptSubmitOutput allows modifying or blocking the user prompt.
type UserPromptSubmitOutput struct {
	Prompt        string
	Block         bool
	Reason        string
	SystemMessage string
}

// PermissionRequestInput carries context for a permission request.
type PermissionRequestInput struct {
	ToolName  string
	ToolInput map[string]any
	SessionID string
}

// PermissionRequestOutput allows hooks to make permission decisions.
type PermissionRequestOutput struct {
	Decision string // "allow" or "deny"
	Reason   string
}

// PreCompactInput carries context before message compaction.
type PreCompactInput struct {
	SessionID    string
	MessageCount int
}

// PreCompactOutput allows controlling compaction behavior.
type PreCompactOutput struct {
	Block  bool
	Reason string
}

// SubagentStartedInput carries context when a subagent starts.
type SubagentStartedInput struct {
	AgentID   string
	SessionID string
}

// SubagentStartedOutput is returned after a SubagentStarted handler runs.
type SubagentStartedOutput struct{}

// SubagentStoppedInput carries context when a subagent stops.
type SubagentStoppedInput struct {
	AgentID   string
	SessionID string
	Result    string
}

// SubagentStoppedOutput is returned after a SubagentStopped handler runs.
type SubagentStoppedOutput struct{}

// SessionStartedInput carries context when a session starts.
type SessionStartedInput struct {
	SessionID string
}

// SessionStartedOutput is returned after a SessionStarted handler runs.
type SessionStartedOutput struct{}

// SessionStoppedInput carries context when a session stops.
type SessionStoppedInput struct {
	SessionID string
}

// SessionStoppedOutput is returned after a SessionStopped handler runs.
type SessionStoppedOutput struct{}

// ErrorInput carries an error from the agent runtime.
type ErrorInput struct {
	Err       error
	SessionID string
}

// ErrorOutput is returned after an Error handler runs.
type ErrorOutput struct{}

// HookContext carries ambient state available to all hook callbacks.
type HookContext struct {
	Signal any
}

// --- Wire-format types (JSON protocol between SDK and CLI) ---

// BaseHookInput contains fields common to all hook inputs on the wire.
type BaseHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode,omitempty"`
}

// PreToolUseHookWireInput is the JSON wire format for PreToolUse hook events.
type PreToolUseHookWireInput struct {
	BaseHookInput
	HookEventName string         `json:"hook_event_name"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolUseID     string         `json:"tool_use_id"`
}

// PostToolUseHookWireInput is the JSON wire format for PostToolUse hook events.
type PostToolUseHookWireInput struct {
	BaseHookInput
	HookEventName string         `json:"hook_event_name"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolResponse  any            `json:"tool_response"`
	ToolUseID     string         `json:"tool_use_id"`
}

// PostToolUseFailureHookWireInput is the JSON wire format for PostToolUseFailure hook events.
type PostToolUseFailureHookWireInput struct {
	BaseHookInput
	HookEventName string         `json:"hook_event_name"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolUseID     string         `json:"tool_use_id"`
	Error         string         `json:"error"`
	IsInterrupt   bool           `json:"is_interrupt,omitempty"`
}

// UserPromptSubmitHookWireInput is the JSON wire format for UserPromptSubmit hook events.
type UserPromptSubmitHookWireInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
}

// StopHookWireInput is the JSON wire format for Stop hook events.
type StopHookWireInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// SubagentStopHookWireInput is the JSON wire format for SubagentStop hook events.
type SubagentStopHookWireInput struct {
	BaseHookInput
	HookEventName       string `json:"hook_event_name"`
	StopHookActive      bool   `json:"stop_hook_active"`
	AgentID             string `json:"agent_id"`
	AgentTranscriptPath string `json:"agent_transcript_path"`
	AgentType           string `json:"agent_type"`
}

// PreCompactHookWireInput is the JSON wire format for PreCompact hook events.
type PreCompactHookWireInput struct {
	BaseHookInput
	HookEventName      string `json:"hook_event_name"`
	Trigger            string `json:"trigger"`
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

// NotificationHookWireInput is the JSON wire format for Notification hook events.
type NotificationHookWireInput struct {
	BaseHookInput
	HookEventName    string `json:"hook_event_name"`
	Message          string `json:"message"`
	Title            string `json:"title,omitempty"`
	NotificationType string `json:"notification_type"`
}

// SubagentStartHookWireInput is the JSON wire format for SubagentStart hook events.
type SubagentStartHookWireInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	AgentID       string `json:"agent_id"`
	AgentType     string `json:"agent_type"`
}

// PermissionRequestHookWireInput is the JSON wire format for PermissionRequest hook events.
type PermissionRequestHookWireInput struct {
	BaseHookInput
	HookEventName        string         `json:"hook_event_name"`
	ToolName             string         `json:"tool_name"`
	ToolInput            map[string]any `json:"tool_input"`
	PermissionSuggestions []any         `json:"permission_suggestions,omitempty"`
}

// --- Hook-specific output types (returned in hookSpecificOutput field) ---

// PreToolUseHookSpecificOutput is the hook-specific output for PreToolUse events.
type PreToolUseHookSpecificOutput struct {
	HookEventName          string         `json:"hookEventName"`
	PermissionDecision     string         `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string       `json:"permissionDecisionReason,omitempty"`
	UpdatedInput           map[string]any `json:"updatedInput,omitempty"`
	AdditionalContext      string         `json:"additionalContext,omitempty"`
}

// PostToolUseHookSpecificOutput is the hook-specific output for PostToolUse events.
type PostToolUseHookSpecificOutput struct {
	HookEventName      string `json:"hookEventName"`
	AdditionalContext  string `json:"additionalContext,omitempty"`
	UpdatedMCPToolOutput any  `json:"updatedMCPToolOutput,omitempty"`
}

// PostToolUseFailureHookSpecificOutput is the hook-specific output for PostToolUseFailure events.
type PostToolUseFailureHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// NotificationHookSpecificOutput is the hook-specific output for Notification events.
type NotificationHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// SubagentStartHookSpecificOutput is the hook-specific output for SubagentStart events.
type SubagentStartHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// PermissionRequestHookSpecificOutput is the hook-specific output for PermissionRequest events.
type PermissionRequestHookSpecificOutput struct {
	HookEventName string         `json:"hookEventName"`
	Decision      map[string]any `json:"decision"`
}

// HookJSONOutput is the JSON envelope returned by hook processes.
type HookJSONOutput struct {
	Continue           *bool          `json:"continue,omitempty"`
	SuppressOutput     bool           `json:"suppressOutput,omitempty"`
	StopReason         string         `json:"stopReason,omitempty"`
	Decision           string         `json:"decision,omitempty"`
	SystemMessage      string         `json:"systemMessage,omitempty"`
	Reason             string         `json:"reason,omitempty"`
	HookSpecificOutput any            `json:"hookSpecificOutput,omitempty"`
	Async              bool           `json:"async,omitempty"`
	AsyncTimeout       int            `json:"asyncTimeout,omitempty"`
}

// HookMatcher pairs a pattern with callbacks and a timeout.
type HookMatcher struct {
	Matcher string   `json:"matcher"`
	Hooks   []any    `json:"hooks"`
	Timeout *float64 `json:"timeout,omitempty"`
}

// Handler function types — one per event.

type PreToolUseHandler func(ctx context.Context, input *PreToolUseInput) (*PreToolUseOutput, error)
type PostToolUseHandler func(ctx context.Context, input *PostToolUseInput) (*PostToolUseOutput, error)
type PostToolUseFailureHandler func(ctx context.Context, input *PostToolUseFailureInput) (*PostToolUseFailureOutput, error)
type ModelResponseHandler func(ctx context.Context, input *ModelResponseInput) (*ModelResponseOutput, error)
type NotificationArrivedHandler func(ctx context.Context, input *NotificationArrivedInput) (*NotificationArrivedOutput, error)
type StopHandler func(ctx context.Context, input *StopInput) (*StopOutput, error)
type SubagentStartedHandler func(ctx context.Context, input *SubagentStartedInput) (*SubagentStartedOutput, error)
type SubagentStoppedHandler func(ctx context.Context, input *SubagentStoppedInput) (*SubagentStoppedOutput, error)
type SessionStartedHandler func(ctx context.Context, input *SessionStartedInput) (*SessionStartedOutput, error)
type SessionStoppedHandler func(ctx context.Context, input *SessionStoppedInput) (*SessionStoppedOutput, error)
type UserPromptSubmitHandler func(ctx context.Context, input *UserPromptSubmitInput) (*UserPromptSubmitOutput, error)
type PermissionRequestHandler func(ctx context.Context, input *PermissionRequestInput) (*PermissionRequestOutput, error)
type PreCompactHandler func(ctx context.Context, input *PreCompactInput) (*PreCompactOutput, error)
type ErrorHandler func(ctx context.Context, input *ErrorInput) (*ErrorOutput, error)

type preToolUseEntry struct {
	pattern string
	handler PreToolUseHandler
}

type postToolUseEntry struct {
	pattern string
	handler PostToolUseHandler
}

type postToolUseFailureEntry struct {
	pattern string
	handler PostToolUseFailureHandler
}

type permissionRequestEntry struct {
	pattern string
	handler PermissionRequestHandler
}

// Hooks is a registry for agent lifecycle event handlers.
type Hooks struct {
	preToolUse          []preToolUseEntry
	postToolUse         []postToolUseEntry
	postToolUseFailure  []postToolUseFailureEntry
	modelResponse       []ModelResponseHandler
	notificationArrived []NotificationArrivedHandler
	stop                []StopHandler
	subagentStarted     []SubagentStartedHandler
	subagentStopped     []SubagentStoppedHandler
	sessionStarted      []SessionStartedHandler
	sessionStopped      []SessionStoppedHandler
	userPromptSubmit    []UserPromptSubmitHandler
	permissionRequest   []permissionRequestEntry
	preCompact          []PreCompactHandler
	errorHandlers       []ErrorHandler
}

// New returns an empty Hooks registry.
func New() *Hooks {
	return &Hooks{}
}

func sdkMatcher(pattern string) HookMatcher {
	return HookMatcher{Matcher: pattern, Hooks: []any{}}
}

// RegisteredEvents returns event entries suitable for the initialize request.
// Each entry maps an event name to its matchers (patterns).
func (h *Hooks) RegisteredEvents() map[string][]HookMatcher {
	events := make(map[string][]HookMatcher)

	for _, e := range h.preToolUse {
		events["PreToolUse"] = append(events["PreToolUse"], sdkMatcher(e.pattern))
	}
	for _, e := range h.postToolUse {
		events["PostToolUse"] = append(events["PostToolUse"], sdkMatcher(e.pattern))
	}
	for _, e := range h.postToolUseFailure {
		events["PostToolUseFailure"] = append(events["PostToolUseFailure"], sdkMatcher(e.pattern))
	}
	if len(h.modelResponse) > 0 {
		events["ModelResponse"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.notificationArrived) > 0 {
		events["NotificationArrived"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.stop) > 0 {
		events["Stop"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.subagentStarted) > 0 {
		events["SubagentStarted"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.subagentStopped) > 0 {
		events["SubagentStopped"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.sessionStarted) > 0 {
		events["SessionStarted"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.sessionStopped) > 0 {
		events["SessionStopped"] = []HookMatcher{sdkMatcher("")}
	}
	if len(h.userPromptSubmit) > 0 {
		events["UserPromptSubmit"] = []HookMatcher{sdkMatcher("")}
	}
	for _, e := range h.permissionRequest {
		events["PermissionRequest"] = append(events["PermissionRequest"], sdkMatcher(e.pattern))
	}
	if len(h.preCompact) > 0 {
		events["PreCompact"] = []HookMatcher{sdkMatcher("")}
	}

	return events
}

// OnPreToolUse registers a handler for tool calls whose name matches pattern (glob syntax).
func (h *Hooks) OnPreToolUse(pattern string, fn PreToolUseHandler) {
	h.preToolUse = append(h.preToolUse, preToolUseEntry{pattern: pattern, handler: fn})
}

// OnPostToolUse registers a handler for completed tool calls whose name matches pattern.
func (h *Hooks) OnPostToolUse(pattern string, fn PostToolUseHandler) {
	h.postToolUse = append(h.postToolUse, postToolUseEntry{pattern: pattern, handler: fn})
}

// OnModelResponse registers a handler for model text responses.
func (h *Hooks) OnModelResponse(fn ModelResponseHandler) {
	h.modelResponse = append(h.modelResponse, fn)
}

// OnNotificationArrived registers a handler for task notifications.
func (h *Hooks) OnNotificationArrived(fn NotificationArrivedHandler) {
	h.notificationArrived = append(h.notificationArrived, fn)
}

// OnStop registers a handler called when the agent stops.
func (h *Hooks) OnStop(fn StopHandler) {
	h.stop = append(h.stop, fn)
}

// OnSubagentStarted registers a handler called when a subagent starts.
func (h *Hooks) OnSubagentStarted(fn SubagentStartedHandler) {
	h.subagentStarted = append(h.subagentStarted, fn)
}

// OnSubagentStopped registers a handler called when a subagent stops.
func (h *Hooks) OnSubagentStopped(fn SubagentStoppedHandler) {
	h.subagentStopped = append(h.subagentStopped, fn)
}

// OnSessionStarted registers a handler called when a session starts.
func (h *Hooks) OnSessionStarted(fn SessionStartedHandler) {
	h.sessionStarted = append(h.sessionStarted, fn)
}

// OnSessionStopped registers a handler called when a session stops.
func (h *Hooks) OnSessionStopped(fn SessionStoppedHandler) {
	h.sessionStopped = append(h.sessionStopped, fn)
}

// OnPostToolUseFailure registers a handler for failed tool calls whose name matches pattern.
func (h *Hooks) OnPostToolUseFailure(pattern string, fn PostToolUseFailureHandler) {
	h.postToolUseFailure = append(h.postToolUseFailure, postToolUseFailureEntry{pattern: pattern, handler: fn})
}

// OnUserPromptSubmit registers a handler called when a user prompt is submitted.
func (h *Hooks) OnUserPromptSubmit(fn UserPromptSubmitHandler) {
	h.userPromptSubmit = append(h.userPromptSubmit, fn)
}

// OnPermissionRequest registers a handler for permission requests whose tool name matches pattern.
func (h *Hooks) OnPermissionRequest(pattern string, fn PermissionRequestHandler) {
	h.permissionRequest = append(h.permissionRequest, permissionRequestEntry{pattern: pattern, handler: fn})
}

// OnPreCompact registers a handler called before message compaction.
func (h *Hooks) OnPreCompact(fn PreCompactHandler) {
	h.preCompact = append(h.preCompact, fn)
}

// OnError registers a handler called when an error occurs.
func (h *Hooks) OnError(fn ErrorHandler) {
	h.errorHandlers = append(h.errorHandlers, fn)
}

// DispatchPreToolUse runs all matching PreToolUse handlers in registration order.
// Each handler receives the (possibly modified) input from the previous handler.
// If any handler blocks, the merged output is blocking.
func (h *Hooks) DispatchPreToolUse(ctx context.Context, input *PreToolUseInput) (*PreToolUseOutput, error) {
	merged := &PreToolUseOutput{}
	current := input
	for _, entry := range h.preToolUse {
		matched, err := path.Match(entry.pattern, input.ToolName)
		if err != nil || !matched {
			continue
		}
		out, err := entry.handler(ctx, current)
		if err != nil {
			return nil, err
		}
		if out != nil {
			if out.Block {
				merged.Block = true
				merged.Reason = out.Reason
			}
			if out.ToolInput != nil {
				merged.ToolInput = out.ToolInput
				current = &PreToolUseInput{
					ToolName:  input.ToolName,
					ToolInput: out.ToolInput,
					SessionID: input.SessionID,
				}
			}
		}
	}
	return merged, nil
}

// DispatchPostToolUse runs all matching PostToolUse handlers in registration order.
func (h *Hooks) DispatchPostToolUse(ctx context.Context, input *PostToolUseInput) (*PostToolUseOutput, error) {
	for _, entry := range h.postToolUse {
		matched, err := path.Match(entry.pattern, input.ToolName)
		if err != nil || !matched {
			continue
		}
		if _, err := entry.handler(ctx, input); err != nil {
			return nil, err
		}
	}
	return &PostToolUseOutput{}, nil
}

// DispatchModelResponse runs all ModelResponse handlers in registration order.
func (h *Hooks) DispatchModelResponse(ctx context.Context, input *ModelResponseInput) (*ModelResponseOutput, error) {
	for _, fn := range h.modelResponse {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &ModelResponseOutput{}, nil
}

// DispatchNotificationArrived runs all NotificationArrived handlers in registration order.
func (h *Hooks) DispatchNotificationArrived(ctx context.Context, input *NotificationArrivedInput) (*NotificationArrivedOutput, error) {
	for _, fn := range h.notificationArrived {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &NotificationArrivedOutput{}, nil
}

// DispatchStop runs all Stop handlers in registration order.
func (h *Hooks) DispatchStop(ctx context.Context, input *StopInput) (*StopOutput, error) {
	for _, fn := range h.stop {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &StopOutput{}, nil
}

// DispatchSubagentStarted runs all SubagentStarted handlers in registration order.
func (h *Hooks) DispatchSubagentStarted(ctx context.Context, input *SubagentStartedInput) (*SubagentStartedOutput, error) {
	for _, fn := range h.subagentStarted {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &SubagentStartedOutput{}, nil
}

// DispatchSubagentStopped runs all SubagentStopped handlers in registration order.
func (h *Hooks) DispatchSubagentStopped(ctx context.Context, input *SubagentStoppedInput) (*SubagentStoppedOutput, error) {
	for _, fn := range h.subagentStopped {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &SubagentStoppedOutput{}, nil
}

// DispatchSessionStarted runs all SessionStarted handlers in registration order.
func (h *Hooks) DispatchSessionStarted(ctx context.Context, input *SessionStartedInput) (*SessionStartedOutput, error) {
	for _, fn := range h.sessionStarted {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &SessionStartedOutput{}, nil
}

// DispatchSessionStopped runs all SessionStopped handlers in registration order.
func (h *Hooks) DispatchSessionStopped(ctx context.Context, input *SessionStoppedInput) (*SessionStoppedOutput, error) {
	for _, fn := range h.sessionStopped {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &SessionStoppedOutput{}, nil
}

// DispatchPostToolUseFailure runs all matching PostToolUseFailure handlers in registration order.
func (h *Hooks) DispatchPostToolUseFailure(ctx context.Context, input *PostToolUseFailureInput) (*PostToolUseFailureOutput, error) {
	for _, entry := range h.postToolUseFailure {
		matched, err := path.Match(entry.pattern, input.ToolName)
		if err != nil || !matched {
			continue
		}
		if _, err := entry.handler(ctx, input); err != nil {
			return nil, err
		}
	}
	return &PostToolUseFailureOutput{}, nil
}

// DispatchUserPromptSubmit runs all UserPromptSubmit handlers in registration order.
func (h *Hooks) DispatchUserPromptSubmit(ctx context.Context, input *UserPromptSubmitInput) (*UserPromptSubmitOutput, error) {
	merged := &UserPromptSubmitOutput{}
	for _, fn := range h.userPromptSubmit {
		out, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		if out != nil {
			if out.Block {
				merged.Block = true
				merged.Reason = out.Reason
			}
			if out.Prompt != "" {
				merged.Prompt = out.Prompt
			}
			if out.SystemMessage != "" {
				merged.SystemMessage = out.SystemMessage
			}
		}
	}
	return merged, nil
}

// DispatchPermissionRequest runs all matching PermissionRequest handlers in registration order.
func (h *Hooks) DispatchPermissionRequest(ctx context.Context, input *PermissionRequestInput) (*PermissionRequestOutput, error) {
	for _, entry := range h.permissionRequest {
		matched, err := path.Match(entry.pattern, input.ToolName)
		if err != nil || !matched {
			continue
		}
		out, err := entry.handler(ctx, input)
		if err != nil {
			return nil, err
		}
		if out != nil && out.Decision != "" {
			return out, nil
		}
	}
	return &PermissionRequestOutput{}, nil
}

// DispatchPreCompact runs all PreCompact handlers in registration order.
func (h *Hooks) DispatchPreCompact(ctx context.Context, input *PreCompactInput) (*PreCompactOutput, error) {
	for _, fn := range h.preCompact {
		out, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}
		if out != nil && out.Block {
			return out, nil
		}
	}
	return &PreCompactOutput{}, nil
}

// DispatchError runs all Error handlers in registration order.
func (h *Hooks) DispatchError(ctx context.Context, input *ErrorInput) (*ErrorOutput, error) {
	for _, fn := range h.errorHandlers {
		if _, err := fn(ctx, input); err != nil {
			return nil, err
		}
	}
	return &ErrorOutput{}, nil
}
