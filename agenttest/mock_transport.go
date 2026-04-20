// Package agenttest provides test utilities for the claude-agent-sdk-go module.
// It exports MockTransport and message builder/assertion helpers for use in tests.
package agenttest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

// MockTransport replays a fixed sequence of messages as JSON lines.
// It implements agent.Transporter and is safe for single-goroutine use.
type MockTransport struct {
	lines [][]byte
	idx   int
	mu    sync.Mutex
	sent  [][]byte
}

// NewMockTransport creates a MockTransport that replays the given messages in order.
// Returns an error if any message cannot be serialized to the wire format.
func NewMockTransport(messages ...agent.Message) (*MockTransport, error) {
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

// Start satisfies agent.Transporter; always succeeds.
func (m *MockTransport) Start(_ context.Context) error { return nil }

// Close satisfies agent.Transporter; always succeeds.
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
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type wireContentMessage struct {
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Content []wireContentBlock `json:"content"`
}

type wireSystem struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type wireResult struct {
	Type        string  `json:"type"`
	Subtype     string  `json:"subtype"`
	Result      string  `json:"result"`
	CostUSD     float64 `json:"cost_usd"`
	DurationMS  int64   `json:"duration_ms"`
	IsError     bool    `json:"is_error"`
	SessionID   string  `json:"session_id"`
	NumTurns    int     `json:"num_turns"`
	TotalInput  int     `json:"total_input_tokens"`
	TotalOutput int     `json:"total_output_tokens"`
}

type wireTaskStarted struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

type wireTaskProgress struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type wireTaskNotification struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func marshalMessage(msg agent.Message) ([]byte, error) {
	switch v := msg.(type) {
	case *agent.AssistantMessage:
		w := wireContentMessage{Type: "assistant", Role: v.Role, Content: blocksToWire(v.Content)}
		return json.Marshal(w)
	case *agent.UserMessage:
		w := wireContentMessage{Type: "user", Role: v.Role, Content: blocksToWire(v.Content)}
		return json.Marshal(w)
	case *agent.SystemMessage:
		return json.Marshal(wireSystem{Type: "system", Content: v.Content})
	case *agent.ResultMessage:
		return json.Marshal(wireResult{
			Type:        "result",
			Subtype:     v.Subtype,
			Result:      v.Result,
			CostUSD:     v.CostUSD,
			DurationMS:  v.DurationMS,
			IsError:     v.IsError,
			SessionID:   v.SessionID,
			NumTurns:    v.NumTurns,
			TotalInput:  v.TotalInput,
			TotalOutput: v.TotalOutput,
		})
	case *agent.TaskStarted:
		return json.Marshal(wireTaskStarted{Type: "task_started", SessionID: v.SessionID})
	case *agent.TaskProgress:
		return json.Marshal(wireTaskProgress{Type: "task_progress", Message: v.Message})
	case *agent.TaskNotification:
		return json.Marshal(wireTaskNotification{Type: "task_notification", Title: v.Title, Message: v.Message})
	default:
		return nil, fmt.Errorf("unsupported message type %T", msg)
	}
}

func blocksToWire(blocks []agent.ContentBlock) []wireContentBlock {
	out := make([]wireContentBlock, 0, len(blocks))
	for _, b := range blocks {
		switch v := b.(type) {
		case *agent.TextBlock:
			out = append(out, wireContentBlock{Type: "text", Text: v.Text})
		case *agent.ThinkingBlock:
			out = append(out, wireContentBlock{Type: "thinking", Thinking: v.Thinking})
		case *agent.ToolUseBlock:
			out = append(out, wireContentBlock{Type: "tool_use", ID: v.ID, Name: v.Name, Input: v.Input})
		case *agent.ToolResultBlock:
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
