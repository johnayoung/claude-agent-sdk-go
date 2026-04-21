package claude_test

import (
	"context"
	"encoding/json"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest"
)

func TestControlRequest_McpReconnect_FromCLI(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-mr","request":{"subtype":"mcp_reconnect","serverName":"test-server"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for msg, err := range claude.Query(context.Background(), "test", claude.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.MessageType() == "control_request" {
			t.Fatal("control_request should not be yielded")
		}
	}

	sent := tr.Sent()
	var foundResp bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-mr" {
			foundResp = true
			if respBody["subtype"] != "success" {
				t.Errorf("expected success, got: %v", respBody["subtype"])
			}
		}
	}
	if !foundResp {
		t.Fatal("no control_response for req-mr found")
	}
}

func TestControlRequest_McpReconnect_Malformed(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-bad","request":"not-json-object"}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for _, err := range claude.Query(context.Background(), "test", claude.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	sent := tr.Sent()
	var foundError bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-bad" && respBody["subtype"] == "error" {
			foundError = true
		}
	}
	if !foundError {
		t.Fatal("expected error response for malformed request")
	}
}

func TestControlRequest_McpToggle_FromCLI(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-mt","request":{"subtype":"mcp_toggle","serverName":"my-server","enabled":false}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for _, err := range claude.Query(context.Background(), "test", claude.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	sent := tr.Sent()
	var foundResp bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-mt" && respBody["subtype"] == "success" {
			foundResp = true
		}
	}
	if !foundResp {
		t.Fatal("no success control_response for req-mt found")
	}
}

func TestControlRequest_McpToggle_Malformed(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-mt-bad","request":"invalid"}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for _, err := range claude.Query(context.Background(), "test", claude.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	sent := tr.Sent()
	var foundError bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-mt-bad" && respBody["subtype"] == "error" {
			foundError = true
		}
	}
	if !foundError {
		t.Fatal("expected error response for malformed mcp_toggle request")
	}
}

func TestControlRequest_StopTask_FromCLI(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-st","request":{"subtype":"stop_task","task_id":"task-xyz"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for _, err := range claude.Query(context.Background(), "test", claude.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	sent := tr.Sent()
	var foundResp bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-st" && respBody["subtype"] == "success" {
			foundResp = true
		}
	}
	if !foundResp {
		t.Fatal("no success control_response for req-st found")
	}
}

func TestControlRequest_StopTask_Malformed(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-st-bad","request":"bad"}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
		[]byte(resultMsg),
	)

	for _, err := range claude.Query(context.Background(), "test", claude.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	sent := tr.Sent()
	var foundError bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-st-bad" && respBody["subtype"] == "error" {
			foundError = true
		}
	}
	if !foundError {
		t.Fatal("expected error response for malformed stop_task request")
	}
}
