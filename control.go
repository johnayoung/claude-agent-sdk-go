package claude

import "encoding/json"

// ControlRequestMessage wraps an incoming control_request from the CLI.
// It implements Message so parseLine can return it, but it is NOT yielded to callers.
type ControlRequestMessage struct {
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
}

func (m *ControlRequestMessage) MessageType() string { return "control_request" }

// controlRequestSubtype extracts the subtype field from a raw request payload.
type controlRequestSubtype struct {
	Subtype string `json:"subtype"`
}

// --- Request subtypes ---

type ControlPermissionRequest struct {
	Subtype               string         `json:"subtype"`
	ToolName              string         `json:"tool_name"`
	Input                 map[string]any `json:"input"`
	PermissionSuggestions []any          `json:"permission_suggestions"`
	BlockedPath           *string        `json:"blocked_path"`
	ToolUseID             string         `json:"tool_use_id"`
	AgentID               string         `json:"agent_id,omitempty"`
}

type ControlHookCallbackRequest struct {
	Subtype    string `json:"subtype"`
	CallbackID string `json:"callback_id"`
	Input      any    `json:"input"`
	ToolUseID  *string `json:"tool_use_id"`
}

type ControlInterruptRequest struct {
	Subtype string `json:"subtype"`
}

type ControlSetPermissionModeRequest struct {
	Subtype string `json:"subtype"`
	Mode    string `json:"mode"`
}

type ControlMcpMessageRequest struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"server_name"`
	Message    any    `json:"message"`
}

type ControlRewindFilesRequest struct {
	Subtype       string `json:"subtype"`
	UserMessageID string `json:"user_message_id"`
}

type ControlMcpReconnectRequest struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
}

type ControlMcpToggleRequest struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
	Enabled    bool   `json:"enabled"`
}

type ControlStopTaskRequest struct {
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
}

// --- Response types ---

type ControlResponse struct {
	Subtype   string `json:"subtype"`
	RequestID string `json:"request_id"`
	Response  any    `json:"response"`
}

type ControlErrorResponse struct {
	Subtype   string `json:"subtype"`
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
}

type SDKControlResponse struct {
	Type     string `json:"type"`
	Response any    `json:"response"`
}

// --- Permission response payload ---

type permissionAllowResponse struct {
	Behavior           string `json:"behavior"`
	UpdatedInput       any    `json:"updated_input,omitempty"`
	UpdatedPermissions any    `json:"updated_permissions,omitempty"`
}

type permissionDenyResponse struct {
	Behavior  string `json:"behavior"`
	Message   string `json:"message,omitempty"`
	Interrupt bool   `json:"interrupt,omitempty"`
}
