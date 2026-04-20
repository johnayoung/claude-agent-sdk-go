package claude

import "encoding/json"

// Message is the common interface for all message types received from the Claude CLI.
type Message interface {
	MessageType() string
}

type UserMessage struct {
	Content          []ContentBlock         `json:"content"`
	UUID             string                 `json:"uuid,omitempty"`
	ParentToolUseID  string                 `json:"parent_tool_use_id,omitempty"`
	SessionID        string                 `json:"session_id,omitempty"`
	ToolUseResult    map[string]any `json:"tool_use_result,omitempty"`
}

func (m *UserMessage) MessageType() string { return "user" }

type AssistantMessage struct {
	Content         []ContentBlock         `json:"content"`
	Model           string                 `json:"model"`
	ParentToolUseID string                 `json:"parent_tool_use_id,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Usage           map[string]any `json:"usage,omitempty"`
	MessageID       string                 `json:"message_id,omitempty"`
	StopReason      string                 `json:"stop_reason,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	UUID            string                 `json:"uuid,omitempty"`
}

func (m *AssistantMessage) MessageType() string { return "assistant" }

type SystemMessage struct {
	Subtype string                 `json:"subtype"`
	Data    map[string]any `json:"data,omitempty"`
}

func (m *SystemMessage) MessageType() string { return "system" }

type TaskStartedMessage struct {
	SystemMessage
	TaskID      string `json:"task_id"`
	Description string `json:"description"`
	UUID        string `json:"uuid"`
	SessionID   string `json:"session_id"`
	ToolUseID   string `json:"tool_use_id,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
}

func (m *TaskStartedMessage) MessageType() string { return "system" }

type TaskProgressMessage struct {
	SystemMessage
	TaskID       string                 `json:"task_id"`
	Description  string                 `json:"description"`
	Usage        map[string]any `json:"usage"`
	UUID         string                 `json:"uuid"`
	SessionID    string                 `json:"session_id"`
	ToolUseID    string                 `json:"tool_use_id,omitempty"`
	LastToolName string                 `json:"last_tool_name,omitempty"`
}

func (m *TaskProgressMessage) MessageType() string { return "system" }

type TaskNotificationMessage struct {
	SystemMessage
	TaskID     string                 `json:"task_id"`
	Status     string                 `json:"status"`
	OutputFile string                 `json:"output_file"`
	Summary    string                 `json:"summary"`
	UUID       string                 `json:"uuid"`
	SessionID  string                 `json:"session_id"`
	ToolUseID  string                 `json:"tool_use_id,omitempty"`
	Usage      map[string]any `json:"usage,omitempty"`
}

func (m *TaskNotificationMessage) MessageType() string { return "system" }

type MirrorErrorMessage struct {
	SystemMessage
	Key   string `json:"key,omitempty"`
	Error string `json:"error,omitempty"`
}

func (m *MirrorErrorMessage) MessageType() string { return "system" }

type ResultMessage struct {
	Subtype           string                 `json:"subtype"`
	DurationMS        int64                  `json:"duration_ms"`
	DurationAPIMS     int64                  `json:"duration_api_ms"`
	IsError           bool                   `json:"is_error"`
	NumTurns          int                    `json:"num_turns"`
	SessionID         string                 `json:"session_id"`
	StopReason        string                 `json:"stop_reason,omitempty"`
	TotalCostUSD      float64                `json:"total_cost_usd,omitempty"`
	Usage             map[string]any `json:"usage,omitempty"`
	Result            string                 `json:"result,omitempty"`
	StructuredOutput  json.RawMessage        `json:"structured_output,omitempty"`
	ModelUsage        map[string]any `json:"model_usage,omitempty"`
	PermissionDenials []any          `json:"permission_denials,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
	UUID              string                 `json:"uuid,omitempty"`
}

func (m *ResultMessage) MessageType() string { return "result" }

type StreamEvent struct {
	UUID            string                 `json:"uuid"`
	SessionID       string                 `json:"session_id"`
	Event           map[string]any `json:"event"`
	ParentToolUseID string                 `json:"parent_tool_use_id,omitempty"`
}

func (m *StreamEvent) MessageType() string { return "stream_event" }

type RateLimitEvent struct {
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
	UUID          string        `json:"uuid"`
	SessionID     string        `json:"session_id"`
}

func (m *RateLimitEvent) MessageType() string { return "rate_limit_event" }
