package parse

import (
	"errors"
	"testing"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

func TestParseLine_SystemMessage(t *testing.T) {
	line := []byte(`{"type":"system","content":"You are a helpful assistant."}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.SystemMessage)
	if !ok {
		t.Fatalf("expected *agent.SystemMessage, got %T", msg)
	}
	if m.Content != "You are a helpful assistant." {
		t.Errorf("unexpected content: %q", m.Content)
	}
}

func TestParseLine_AssistantMessage(t *testing.T) {
	line := []byte(`{"type":"assistant","role":"assistant","content":[{"type":"text","text":"Hello!"},{"type":"thinking","thinking":"Let me think..."}]}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.AssistantMessage)
	if !ok {
		t.Fatalf("expected *agent.AssistantMessage, got %T", msg)
	}
	if len(m.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(m.Content))
	}
	text, ok := m.Content[0].(*agent.TextBlock)
	if !ok {
		t.Fatalf("expected *agent.TextBlock, got %T", m.Content[0])
	}
	if text.Text != "Hello!" {
		t.Errorf("unexpected text: %q", text.Text)
	}
	think, ok := m.Content[1].(*agent.ThinkingBlock)
	if !ok {
		t.Fatalf("expected *agent.ThinkingBlock, got %T", m.Content[1])
	}
	if think.Thinking != "Let me think..." {
		t.Errorf("unexpected thinking: %q", think.Thinking)
	}
}

func TestParseLine_UserMessage(t *testing.T) {
	line := []byte(`{"type":"user","role":"user","content":[{"type":"tool_result","tool_use_id":"tu_1","content":"result","is_error":false}]}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.UserMessage)
	if !ok {
		t.Fatalf("expected *agent.UserMessage, got %T", msg)
	}
	if len(m.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(m.Content))
	}
	tr, ok := m.Content[0].(*agent.ToolResultBlock)
	if !ok {
		t.Fatalf("expected *agent.ToolResultBlock, got %T", m.Content[0])
	}
	if tr.ToolUseID != "tu_1" {
		t.Errorf("unexpected tool_use_id: %q", tr.ToolUseID)
	}
}

func TestParseLine_ToolUseBlock(t *testing.T) {
	line := []byte(`{"type":"assistant","role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"bash","input":{"command":"ls"}}]}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*agent.AssistantMessage)
	tu, ok := m.Content[0].(*agent.ToolUseBlock)
	if !ok {
		t.Fatalf("expected *agent.ToolUseBlock, got %T", m.Content[0])
	}
	if tu.Name != "bash" {
		t.Errorf("unexpected name: %q", tu.Name)
	}
}

func TestParseLine_ResultMessage(t *testing.T) {
	line := []byte(`{"type":"result","subtype":"success","result":"done","cost_usd":0.01,"duration_ms":1000,"is_error":false,"session_id":"sid","num_turns":3,"total_input_tokens":100,"total_output_tokens":50}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.ResultMessage)
	if !ok {
		t.Fatalf("expected *agent.ResultMessage, got %T", msg)
	}
	if m.SessionID != "sid" {
		t.Errorf("unexpected session_id: %q", m.SessionID)
	}
	if m.NumTurns != 3 {
		t.Errorf("unexpected num_turns: %d", m.NumTurns)
	}
}

func TestParseLine_TaskStarted(t *testing.T) {
	line := []byte(`{"type":"task_started","session_id":"abc123"}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.TaskStarted)
	if !ok {
		t.Fatalf("expected *agent.TaskStarted, got %T", msg)
	}
	if m.SessionID != "abc123" {
		t.Errorf("unexpected session_id: %q", m.SessionID)
	}
}

func TestParseLine_TaskProgress(t *testing.T) {
	line := []byte(`{"type":"task_progress","message":"Working..."}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.TaskProgress)
	if !ok {
		t.Fatalf("expected *agent.TaskProgress, got %T", msg)
	}
	if m.Message != "Working..." {
		t.Errorf("unexpected message: %q", m.Message)
	}
}

func TestParseLine_TaskNotification(t *testing.T) {
	line := []byte(`{"type":"task_notification","title":"Alert","message":"Something happened"}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*agent.TaskNotification)
	if !ok {
		t.Fatalf("expected *agent.TaskNotification, got %T", msg)
	}
	if m.Title != "Alert" || m.Message != "Something happened" {
		t.Errorf("unexpected fields: %+v", m)
	}
}

func TestParseLine_MalformedJSON(t *testing.T) {
	line := []byte(`not json at all`)
	_, err := ParseLine(line)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	var decErr *agent.JSONDecodeError
	if !errors.As(err, &decErr) {
		t.Fatalf("expected JSONDecodeError, got %T: %v", err, err)
	}
	if decErr.RawLine != "not json at all" {
		t.Errorf("unexpected RawLine: %q", decErr.RawLine)
	}
}

func TestParseLine_UnknownType(t *testing.T) {
	line := []byte(`{"type":"something_new","data":"x"}`)
	_, err := ParseLine(line)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	var parseErr *agent.MessageParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected MessageParseError, got %T: %v", err, err)
	}
	if parseErr.TypeField != "something_new" {
		t.Errorf("unexpected TypeField: %q", parseErr.TypeField)
	}
}

func TestParseLine_UnknownContentBlockSkipped(t *testing.T) {
	line := []byte(`{"type":"assistant","role":"assistant","content":[{"type":"text","text":"hi"},{"type":"future_block","data":"x"}]}`)
	msg, err := ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*agent.AssistantMessage)
	if len(m.Content) != 1 {
		t.Errorf("expected 1 block (unknown skipped), got %d", len(m.Content))
	}
}

func TestParseLine_EmptyType(t *testing.T) {
	line := []byte(`{"type":""}`)
	_, err := ParseLine(line)
	var parseErr *agent.MessageParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected MessageParseError for empty type, got %T: %v", err, err)
	}
}
