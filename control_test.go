package claude_test

import (
	"context"
	"encoding/json"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest"
	"github.com/johnayoung/claude-agent-sdk-go/hooks"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

func findSentControlResponse(sent [][]byte, requestID string) map[string]any {
	for _, line := range sent {
		var msg map[string]any
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		if msg["type"] != "control_response" {
			continue
		}
		respBody, _ := msg["response"].(map[string]any)
		if respBody["request_id"] == requestID {
			return msg
		}
	}
	return nil
}

func TestControlRequest_PermissionAllow(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-1","request":{"subtype":"can_use_tool","tool_name":"bash","input":{"command":"rm -rf /"},"permission_suggestions":null,"blocked_path":null,"tool_use_id":"toolu_01"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	var called bool
	var calledToolName string
	var calledInput map[string]any

	canUseTool := func(toolName string, input map[string]any, ctx permission.ToolContext) (permission.Decision, error) {
		called = true
		calledToolName = toolName
		calledInput = input
		return permission.Allow("test allows"), nil
	}

	var messages []claude.Message
	for msg, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
		claude.WithCanUseTool(canUseTool),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		messages = append(messages, msg)
	}

	if !called {
		t.Fatal("CanUseTool was not called")
	}
	if calledToolName != "bash" {
		t.Errorf("unexpected tool_name: %q", calledToolName)
	}
	if calledInput["command"] != "rm -rf /" {
		t.Errorf("unexpected input: %v", calledInput)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message (result), got %d", len(messages))
	}
	if _, ok := messages[0].(*claude.ResultMessage); !ok {
		t.Fatalf("expected *ResultMessage, got %T", messages[0])
	}

	resp := findSentControlResponse(tr.Sent(), "req-1")
	if resp == nil {
		t.Fatal("no control_response for req-1 found")
	}
	respBody, _ := resp["response"].(map[string]any)
	if respBody["subtype"] != "success" {
		t.Errorf("unexpected response subtype: %v", respBody["subtype"])
	}
}

func TestControlRequest_PermissionDeny(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-2","request":{"subtype":"can_use_tool","tool_name":"bash","input":{"command":"danger"},"permission_suggestions":null,"blocked_path":null,"tool_use_id":"toolu_02"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	canUseTool := func(toolName string, input map[string]any, ctx permission.ToolContext) (permission.Decision, error) {
		return permission.Deny("not allowed"), nil
	}

	for msg, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
		claude.WithCanUseTool(canUseTool),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = msg
	}

	resp := findSentControlResponse(tr.Sent(), "req-2")
	if resp == nil {
		t.Fatal("no control_response for req-2 found")
	}
	respBody, _ := resp["response"].(map[string]any)
	innerResp, _ := respBody["response"].(map[string]any)
	if innerResp["behavior"] != "deny" {
		t.Errorf("expected deny behavior, got: %v", innerResp["behavior"])
	}
	if innerResp["message"] != "not allowed" {
		t.Errorf("unexpected deny message: %v", innerResp["message"])
	}
}

func TestControlRequest_DefaultAllow_NilCanUseTool(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-3","request":{"subtype":"can_use_tool","tool_name":"read","input":{},"permission_suggestions":null,"blocked_path":null,"tool_use_id":"toolu_03"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for msg, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = msg
	}

	resp := findSentControlResponse(tr.Sent(), "req-3")
	if resp == nil {
		t.Fatal("no control_response for req-3 found")
	}
	respBody, _ := resp["response"].(map[string]any)
	innerResp, _ := respBody["response"].(map[string]any)
	if innerResp["behavior"] != "allow" {
		t.Errorf("expected default allow, got: %v", innerResp["behavior"])
	}
}

func TestControlRequest_Interrupt(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-int","request":{"subtype":"interrupt"}}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
	)

	for _, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
	) {
		if err != nil && err != context.Canceled {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	resp := findSentControlResponse(tr.Sent(), "req-int")
	if resp == nil {
		t.Fatal("no control_response for req-int found")
	}
}

func TestControlRequest_NotYieldedToCaller(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-hidden","request":{"subtype":"mcp_message","server_name":"test","message":{}}}`
	assistantMsg := `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}],"model":"claude-sonnet-4-5"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(assistantMsg),
		[]byte(resultMsg),
	)

	var messages []claude.Message
	for msg, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		messages = append(messages, msg)
	}

	for _, msg := range messages {
		if msg.MessageType() == "control_request" {
			t.Fatal("control_request was yielded to caller")
		}
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 messages (assistant + result), got %d", len(messages))
	}
}

func TestControlRequest_HookCallback_Stop(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-stop","request":{"subtype":"hook_callback","callback_id":"cb-1","input":{"hook_event_name":"stop","reason":"end_turn","session_id":"sess-1"}}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	var callCount int
	var firstReason string
	h := hooks.New()
	h.OnStop(func(ctx context.Context, input *hooks.StopInput) (*hooks.StopOutput, error) {
		callCount++
		if callCount == 1 {
			firstReason = input.Reason
		}
		return &hooks.StopOutput{}, nil
	})

	for _, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
		claude.WithHooks(h),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if callCount == 0 {
		t.Fatal("stop hook was not dispatched")
	}
	if firstReason != "end_turn" {
		t.Errorf("unexpected reason from control request: %q", firstReason)
	}
}

func TestControlRequest_HookCallback_UserPromptSubmit(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-ups","request":{"subtype":"hook_callback","callback_id":"cb-2","input":{"hook_event_name":"user_prompt_submit","prompt":"hello world","session_id":"sess-1"}}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	var calledPrompt string
	h := hooks.New()
	h.OnUserPromptSubmit(func(ctx context.Context, input *hooks.UserPromptSubmitInput) (*hooks.UserPromptSubmitOutput, error) {
		calledPrompt = input.Prompt
		return &hooks.UserPromptSubmitOutput{}, nil
	})

	for _, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
		claude.WithHooks(h),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if calledPrompt != "hello world" {
		t.Errorf("unexpected prompt: %q", calledPrompt)
	}
}

func TestControlRequest_HookCallback_PreCompact(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-pc","request":{"subtype":"hook_callback","callback_id":"cb-3","input":{"hook_event_name":"pre_compact","session_id":"sess-1","message_count":42}}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	var calledCount int
	h := hooks.New()
	h.OnPreCompact(func(ctx context.Context, input *hooks.PreCompactInput) (*hooks.PreCompactOutput, error) {
		calledCount = input.MessageCount
		return &hooks.PreCompactOutput{}, nil
	})

	for _, err := range claude.Query(context.Background(), "test",
		claude.WithTransport(tr),
		claude.WithHooks(h),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if calledCount != 42 {
		t.Errorf("unexpected message_count: %d", calledCount)
	}
}
