package claude

// Message is the common interface for all message types received from the Claude CLI.
type Message interface {
	MessageType() string
}

type UserMessage struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

func (m *UserMessage) MessageType() string { return "user" }

type AssistantMessage struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

func (m *AssistantMessage) MessageType() string { return "assistant" }

type SystemMessage struct {
	Content string `json:"content"`
}

func (m *SystemMessage) MessageType() string { return "system" }

type ResultMessage struct {
	Subtype          string  `json:"subtype"`
	Result           string  `json:"result"`
	CostUSD          float64 `json:"cost_usd"`
	CostInputUSD     float64 `json:"cost_input_usd"`
	CostOutputUSD    float64 `json:"cost_output_usd"`
	DurationMS       int64   `json:"duration_ms"`
	IsError          bool    `json:"is_error"`
	ErrorType        string  `json:"error_type,omitempty"`
	SessionID        string  `json:"session_id"`
	NumTurns         int     `json:"num_turns"`
	TotalInput       int     `json:"total_input_tokens"`
	TotalOutput      int     `json:"total_output_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens"`
	CacheWriteTokens int     `json:"cache_write_tokens"`
}

func (m *ResultMessage) MessageType() string { return "result" }

type TaskStarted struct {
	SessionID string `json:"session_id"`
}

func (m *TaskStarted) MessageType() string { return "task_started" }

type TaskProgress struct {
	Message string `json:"message"`
}

func (m *TaskProgress) MessageType() string { return "task_progress" }

type TaskNotification struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (m *TaskNotification) MessageType() string { return "task_notification" }

type RateLimitEvent struct {
	RequestsLimit     int   `json:"requests_limit"`
	RequestsRemaining int   `json:"requests_remaining"`
	RequestsReset     int64 `json:"requests_reset"`
	TokensLimit       int   `json:"tokens_limit"`
	TokensRemaining   int   `json:"tokens_remaining"`
	TokensReset       int64 `json:"tokens_reset"`
}

func (m *RateLimitEvent) MessageType() string { return "rate_limit" }
