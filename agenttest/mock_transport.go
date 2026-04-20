// Package agenttest provides test utilities for the claude-agent-sdk-go module.
// It exports MockTransport and message builder/assertion helpers for use in tests.
package agenttest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

// MockTransport replays a fixed sequence of messages as JSON lines.
// It implements claude.Transporter and is safe for single-goroutine use.
type MockTransport struct {
	lines [][]byte
	idx   int
	mu    sync.Mutex
	sent  [][]byte
}

// NewMockTransport creates a MockTransport that replays the given messages in order.
// Returns an error if any message cannot be serialized to the wire format.
func NewMockTransport(messages ...claude.Message) (*MockTransport, error) {
	lines := make([][]byte, 0, len(messages))
	for _, msg := range messages {
		line, err := marshalMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("agenttest: marshal %T: %w", msg, err)
		}
		lines = append(lines, line)
	}
	return &MockTransport{lines: lines}, nil
}

// Start satisfies claude.Transporter; always succeeds.
func (m *MockTransport) Start(_ context.Context) error { return nil }

// Close satisfies claude.Transporter; always succeeds.
func (m *MockTransport) Close() error { return nil }

// Send records the line so tests can inspect outgoing messages via Sent().
func (m *MockTransport) Send(line []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(line))
	copy(cp, line)
	m.sent = append(m.sent, cp)
	return nil
}

// Receive returns the next pre-loaded message line, or io.EOF when exhausted.
func (m *MockTransport) Receive() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx >= len(m.lines) {
		return nil, io.EOF
	}
	line := m.lines[m.idx]
	m.idx++
	return line, nil
}

// Sent returns a copy of all lines passed to Send, in order.
func (m *MockTransport) Sent() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.sent))
	for i, b := range m.sent {
		cp := make([]byte, len(b))
		copy(cp, b)
		result[i] = cp
	}
	return result
}

// ---- wire serialization -------------------------------------------------------

type wireContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type wireInnerMessage struct {
	Content []wireContentBlock     `json:"content"`
	Model   string                 `json:"model,omitempty"`
	ID      string                 `json:"id,omitempty"`
	Usage   map[string]any `json:"usage,omitempty"`
}

type wireAssistant struct {
	Type            string           `json:"type"`
	Message         wireInnerMessage `json:"message"`
	ParentToolUseID string           `json:"parent_tool_use_id,omitempty"`
	Error           string           `json:"error,omitempty"`
	SessionID       string           `json:"session_id,omitempty"`
	UUID            string           `json:"uuid,omitempty"`
}

type wireUser struct {
	Type            string           `json:"type"`
	Message         wireInnerMessage `json:"message"`
	ParentToolUseID string           `json:"parent_tool_use_id,omitempty"`
	UUID            string           `json:"uuid,omitempty"`
	SessionID       string           `json:"session_id,omitempty"`
}

type wireSystem struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

type wireResult struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	Result       string  `json:"result,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	DurationMS   int64   `json:"duration_ms"`
	DurationAPIM int64   `json:"duration_api_ms"`
	IsError      bool    `json:"is_error"`
	SessionID    string  `json:"session_id"`
	NumTurns     int     `json:"num_turns"`
}

type wireTaskStarted struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	TaskID      string `json:"task_id"`
	Description string `json:"description"`
	UUID        string `json:"uuid"`
	SessionID   string `json:"session_id"`
}

type wireTaskNotification struct {
	Type       string `json:"type"`
	Subtype    string `json:"subtype"`
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	OutputFile string `json:"output_file"`
	Summary    string `json:"summary"`
	UUID       string `json:"uuid"`
	SessionID  string `json:"session_id"`
}

func marshalMessage(msg claude.Message) ([]byte, error) {
	switch v := msg.(type) {
	case *claude.AssistantMessage:
		w := wireAssistant{
			Type:            "assistant",
			Message:         wireInnerMessage{Content: blocksToWire(v.Content), Model: v.Model},
			ParentToolUseID: v.ParentToolUseID,
			Error:           v.Error,
			SessionID:       v.SessionID,
			UUID:            v.UUID,
		}
		return json.Marshal(w)
	case *claude.UserMessage:
		w := wireUser{
			Type:            "user",
			Message:         wireInnerMessage{Content: blocksToWire(v.Content)},
			ParentToolUseID: v.ParentToolUseID,
			UUID:            v.UUID,
			SessionID:       v.SessionID,
		}
		return json.Marshal(w)
	case *claude.SystemMessage:
		return json.Marshal(wireSystem{Type: "system", Subtype: v.Subtype})
	case *claude.TaskStartedMessage:
		return json.Marshal(wireTaskStarted{
			Type: "system", Subtype: "task_started",
			TaskID: v.TaskID, Description: v.Description,
			UUID: v.UUID, SessionID: v.SessionID,
		})
	case *claude.TaskNotificationMessage:
		return json.Marshal(wireTaskNotification{
			Type: "system", Subtype: "task_notification",
			TaskID: v.TaskID, Status: v.Status,
			OutputFile: v.OutputFile, Summary: v.Summary,
			UUID: v.UUID, SessionID: v.SessionID,
		})
	case *claude.ResultMessage:
		return json.Marshal(wireResult{
			Type:         "result",
			Subtype:      v.Subtype,
			Result:       v.Result,
			TotalCostUSD: v.TotalCostUSD,
			DurationMS:   v.DurationMS,
			DurationAPIM: v.DurationAPIMS,
			IsError:      v.IsError,
			SessionID:    v.SessionID,
			NumTurns:     v.NumTurns,
		})
	default:
		return nil, fmt.Errorf("unsupported message type %T", msg)
	}
}

func blocksToWire(blocks []claude.ContentBlock) []wireContentBlock {
	out := make([]wireContentBlock, 0, len(blocks))
	for _, b := range blocks {
		switch v := b.(type) {
		case *claude.TextBlock:
			out = append(out, wireContentBlock{Type: "text", Text: v.Text})
		case *claude.ThinkingBlock:
			out = append(out, wireContentBlock{Type: "thinking", Thinking: v.Thinking, Signature: v.Signature})
		case *claude.ToolUseBlock:
			out = append(out, wireContentBlock{Type: "tool_use", ID: v.ID, Name: v.Name, Input: v.Input})
		case *claude.ToolResultBlock:
			out = append(out, wireContentBlock{
				Type:      "tool_result",
				ToolUseID: v.ToolUseID,
				Content:   v.Content,
				IsError:   v.IsError,
			})
		}
	}
	return out
}
