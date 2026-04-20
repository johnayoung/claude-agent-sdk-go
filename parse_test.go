package claude

import (
	"errors"
	"testing"
)

func TestParseLine_SystemMessage(t *testing.T) {
	line := []byte(`{"type":"system","subtype":"start"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*SystemMessage)
	if !ok {
		t.Fatalf("expected *SystemMessage, got %T", msg)
	}
	if m.Subtype != "start" {
		t.Errorf("unexpected subtype: %q", m.Subtype)
	}
}

func TestParseLine_AssistantMessage(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello!"},{"type":"thinking","thinking":"Let me think...","signature":"sig-1"}],"model":"claude-opus-4-5"}}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*AssistantMessage)
	if !ok {
		t.Fatalf("expected *AssistantMessage, got %T", msg)
	}
	if len(m.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(m.Content))
	}
	text, ok := m.Content[0].(*TextBlock)
	if !ok {
		t.Fatalf("expected *TextBlock, got %T", m.Content[0])
	}
	if text.Text != "Hello!" {
		t.Errorf("unexpected text: %q", text.Text)
	}
	think, ok := m.Content[1].(*ThinkingBlock)
	if !ok {
		t.Fatalf("expected *ThinkingBlock, got %T", m.Content[1])
	}
	if think.Thinking != "Let me think..." {
		t.Errorf("unexpected thinking: %q", think.Thinking)
	}
	if think.Signature != "sig-1" {
		t.Errorf("unexpected signature: %q", think.Signature)
	}
	if m.Model != "claude-opus-4-5" {
		t.Errorf("unexpected model: %q", m.Model)
	}
}

func TestParseLine_AssistantMessage_WithFields(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}],"model":"claude-sonnet-4-5","id":"msg_01abc","stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5}},"session_id":"sess-1","uuid":"uuid-1","error":"","parent_tool_use_id":"toolu_01"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*AssistantMessage)
	if m.MessageID != "msg_01abc" {
		t.Errorf("unexpected message_id: %q", m.MessageID)
	}
	if m.StopReason != "end_turn" {
		t.Errorf("unexpected stop_reason: %q", m.StopReason)
	}
	if m.SessionID != "sess-1" {
		t.Errorf("unexpected session_id: %q", m.SessionID)
	}
	if m.UUID != "uuid-1" {
		t.Errorf("unexpected uuid: %q", m.UUID)
	}
	if m.ParentToolUseID != "toolu_01" {
		t.Errorf("unexpected parent_tool_use_id: %q", m.ParentToolUseID)
	}
	if m.Usage == nil {
		t.Fatal("expected non-nil usage")
	}
}

func TestParseLine_UserMessage(t *testing.T) {
	line := []byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu_1","content":"result","is_error":false}]}}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*UserMessage)
	if !ok {
		t.Fatalf("expected *UserMessage, got %T", msg)
	}
	if len(m.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(m.Content))
	}
	tr, ok := m.Content[0].(*ToolResultBlock)
	if !ok {
		t.Fatalf("expected *ToolResultBlock, got %T", m.Content[0])
	}
	if tr.ToolUseID != "tu_1" {
		t.Errorf("unexpected tool_use_id: %q", tr.ToolUseID)
	}
}

func TestParseLine_UserMessage_WithUUID(t *testing.T) {
	line := []byte(`{"type":"user","uuid":"msg-abc","parent_tool_use_id":"toolu_01","message":{"content":[{"type":"text","text":"hi"}]}}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*UserMessage)
	if m.UUID != "msg-abc" {
		t.Errorf("unexpected uuid: %q", m.UUID)
	}
	if m.ParentToolUseID != "toolu_01" {
		t.Errorf("unexpected parent_tool_use_id: %q", m.ParentToolUseID)
	}
}

func TestParseLine_ToolUseBlock(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"bash","input":{"command":"ls"}}],"model":"claude-opus-4-5"}}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*AssistantMessage)
	tu, ok := m.Content[0].(*ToolUseBlock)
	if !ok {
		t.Fatalf("expected *ToolUseBlock, got %T", m.Content[0])
	}
	if tu.Name != "bash" {
		t.Errorf("unexpected name: %q", tu.Name)
	}
}

func TestParseLine_ResultMessage(t *testing.T) {
	line := []byte(`{"type":"result","subtype":"success","result":"done","total_cost_usd":0.01,"duration_ms":1000,"duration_api_ms":500,"is_error":false,"session_id":"sid","num_turns":3}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatalf("expected *ResultMessage, got %T", msg)
	}
	if m.SessionID != "sid" {
		t.Errorf("unexpected session_id: %q", m.SessionID)
	}
	if m.NumTurns != 3 {
		t.Errorf("unexpected num_turns: %d", m.NumTurns)
	}
	if m.TotalCostUSD != 0.01 {
		t.Errorf("unexpected total_cost_usd: %f", m.TotalCostUSD)
	}
	if m.DurationAPIMS != 500 {
		t.Errorf("unexpected duration_api_ms: %d", m.DurationAPIMS)
	}
}

func TestParseLine_ResultMessage_WithModelUsage(t *testing.T) {
	line := []byte(`{"type":"result","subtype":"success","duration_ms":3000,"duration_api_ms":2000,"is_error":false,"num_turns":1,"session_id":"sid","modelUsage":{"claude-sonnet-4-5":{"inputTokens":3,"outputTokens":24}},"permission_denials":[],"uuid":"uuid-r","errors":["err1","err2"]}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*ResultMessage)
	if m.ModelUsage == nil {
		t.Fatal("expected non-nil model_usage")
	}
	if m.UUID != "uuid-r" {
		t.Errorf("unexpected uuid: %q", m.UUID)
	}
	if len(m.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(m.Errors))
	}
	if m.Errors[0] != "err1" {
		t.Errorf("unexpected error[0]: %q", m.Errors[0])
	}
}

func TestParseLine_TaskStarted(t *testing.T) {
	line := []byte(`{"type":"system","subtype":"task_started","task_id":"task-abc","description":"Working","uuid":"uuid-1","session_id":"session-1","tool_use_id":"toolu_01","task_type":"background"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*TaskStartedMessage)
	if !ok {
		t.Fatalf("expected *TaskStartedMessage, got %T", msg)
	}
	if m.TaskID != "task-abc" {
		t.Errorf("unexpected task_id: %q", m.TaskID)
	}
	if m.SessionID != "session-1" {
		t.Errorf("unexpected session_id: %q", m.SessionID)
	}
	if m.ToolUseID != "toolu_01" {
		t.Errorf("unexpected tool_use_id: %q", m.ToolUseID)
	}
	if m.TaskType != "background" {
		t.Errorf("unexpected task_type: %q", m.TaskType)
	}
	// Verify it's also a SystemMessage
	if m.Subtype != "task_started" {
		t.Errorf("unexpected subtype: %q", m.Subtype)
	}
}

func TestParseLine_TaskProgress(t *testing.T) {
	line := []byte(`{"type":"system","subtype":"task_progress","task_id":"task-abc","description":"Halfway","usage":{"total_tokens":1234,"tool_uses":5,"duration_ms":9876},"uuid":"uuid-2","session_id":"session-1","last_tool_name":"Read"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*TaskProgressMessage)
	if !ok {
		t.Fatalf("expected *TaskProgressMessage, got %T", msg)
	}
	if m.Description != "Halfway" {
		t.Errorf("unexpected description: %q", m.Description)
	}
	if m.LastToolName != "Read" {
		t.Errorf("unexpected last_tool_name: %q", m.LastToolName)
	}
}

func TestParseLine_TaskNotification(t *testing.T) {
	line := []byte(`{"type":"system","subtype":"task_notification","task_id":"task-abc","status":"completed","output_file":"/tmp/out.md","summary":"All done","uuid":"uuid-3","session_id":"session-1"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*TaskNotificationMessage)
	if !ok {
		t.Fatalf("expected *TaskNotificationMessage, got %T", msg)
	}
	if m.Status != "completed" {
		t.Errorf("unexpected status: %q", m.Status)
	}
	if m.Summary != "All done" {
		t.Errorf("unexpected summary: %q", m.Summary)
	}
}

func TestParseLine_RateLimitEvent(t *testing.T) {
	line := []byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","resetsAt":1700000000,"rateLimitType":"five_hour","utilization":0.85},"uuid":"uuid-rl","session_id":"sess-1"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*RateLimitEvent)
	if !ok {
		t.Fatalf("expected *RateLimitEvent, got %T", msg)
	}
	if m.UUID != "uuid-rl" {
		t.Errorf("unexpected uuid: %q", m.UUID)
	}
	if m.RateLimitInfo.Status != RateLimitAllowedWarning {
		t.Errorf("unexpected status: %q", m.RateLimitInfo.Status)
	}
	if m.RateLimitInfo.ResetsAt == nil || *m.RateLimitInfo.ResetsAt != 1700000000 {
		t.Errorf("unexpected resets_at: %v", m.RateLimitInfo.ResetsAt)
	}
	if m.RateLimitInfo.RateLimitType != RateLimitFiveHour {
		t.Errorf("unexpected rate_limit_type: %q", m.RateLimitInfo.RateLimitType)
	}
	if m.RateLimitInfo.Utilization == nil || *m.RateLimitInfo.Utilization != 0.85 {
		t.Errorf("unexpected utilization: %v", m.RateLimitInfo.Utilization)
	}
	if m.RateLimitInfo.Raw == nil {
		t.Error("expected non-nil raw")
	}
}

func TestParseLine_RateLimitEvent_Rejected(t *testing.T) {
	line := []byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1700003600,"rateLimitType":"seven_day","overageStatus":"rejected","overageDisabledReason":"out_of_credits"},"uuid":"uuid-rl2","session_id":"sess-1"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*RateLimitEvent)
	if m.RateLimitInfo.Status != RateLimitRejected {
		t.Errorf("unexpected status: %q", m.RateLimitInfo.Status)
	}
	if m.RateLimitInfo.OverageStatus != RateLimitRejected {
		t.Errorf("unexpected overage_status: %q", m.RateLimitInfo.OverageStatus)
	}
	if m.RateLimitInfo.OverageDisabledReason != "out_of_credits" {
		t.Errorf("unexpected overage_disabled_reason: %q", m.RateLimitInfo.OverageDisabledReason)
	}
}

func TestParseLine_StreamEvent(t *testing.T) {
	line := []byte(`{"type":"stream_event","uuid":"uuid-se","session_id":"sess-1","event":{"type":"content_block_delta","delta":{"text":"hi"}}}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := msg.(*StreamEvent)
	if !ok {
		t.Fatalf("expected *StreamEvent, got %T", msg)
	}
	if m.UUID != "uuid-se" {
		t.Errorf("unexpected uuid: %q", m.UUID)
	}
	if m.Event == nil {
		t.Error("expected non-nil event")
	}
}

func TestParseLine_MalformedJSON(t *testing.T) {
	line := []byte(`not json at all`)
	_, err := parseLine(line)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	var decErr *JSONDecodeError
	if !errors.As(err, &decErr) {
		t.Fatalf("expected JSONDecodeError, got %T: %v", err, err)
	}
	if decErr.RawLine != "not json at all" {
		t.Errorf("unexpected RawLine: %q", decErr.RawLine)
	}
}

func TestParseLine_UnknownType(t *testing.T) {
	line := []byte(`{"type":"something_new","data":"x"}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error for unknown type: %v", err)
	}
	if msg != nil {
		t.Fatalf("expected nil message for unknown type, got %T", msg)
	}
}

func TestParseLine_UnknownContentBlockSkipped(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"},{"type":"future_block","data":"x"}],"model":"claude-opus-4-5"}}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*AssistantMessage)
	if len(m.Content) != 1 {
		t.Errorf("expected 1 block (unknown skipped), got %d", len(m.Content))
	}
}

func TestParseLine_EmptyType(t *testing.T) {
	line := []byte(`{"type":""}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != nil {
		t.Fatalf("expected nil for empty type, got %T", msg)
	}
}
