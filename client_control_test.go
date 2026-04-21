package claude_test

import (
	"context"
	"encoding/json"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest"
)

func TestGetContextUsage_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetContextUsage(context.Background())
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestGetContextUsage_ClientClosed(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	_, err = client.GetContextUsage(context.Background())
	if err != claude.ErrClientClosed {
		t.Fatalf("expected ErrClientClosed, got: %v", err)
	}
}

func TestGetContextUsage_Success(t *testing.T) {
	tr := &typedResponseMockTransport{}

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-ctx","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0
	tr.responsePayload = `{"totalTokens":5000,"maxTokens":200000,"percentage":2.5,"model":"claude-opus-4-5","categories":[{"name":"system","tokens":1000,"color":"blue"}]}`

	usage, err := client.GetContextUsage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.TotalTokens != 5000 {
		t.Errorf("unexpected TotalTokens: %d", usage.TotalTokens)
	}
	if usage.MaxTokens != 200000 {
		t.Errorf("unexpected MaxTokens: %d", usage.MaxTokens)
	}
	if usage.Model != "claude-opus-4-5" {
		t.Errorf("unexpected Model: %q", usage.Model)
	}
	if len(usage.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(usage.Categories))
	}
	if usage.Categories[0].Name != "system" {
		t.Errorf("unexpected category name: %q", usage.Categories[0].Name)
	}
}

func TestGetServerInfo_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetServerInfo(context.Background())
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestGetServerInfo_Success(t *testing.T) {
	tr := &typedResponseMockTransport{}

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-si","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0
	tr.responsePayload = `{"outputStyle":"streaming","availableOutputStyles":["streaming","text"]}`

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.OutputStyle != "streaming" {
		t.Errorf("unexpected OutputStyle: %q", info.OutputStyle)
	}
	if len(info.AvailableOutputStyles) != 2 {
		t.Fatalf("expected 2 output styles, got %d", len(info.AvailableOutputStyles))
	}
}

func TestStopTask_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.StopTask(context.Background(), "task-123")
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestStopTask_ClientClosed(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	err = client.StopTask(context.Background(), "task-123")
	if err != claude.ErrClientClosed {
		t.Fatalf("expected ErrClientClosed, got: %v", err)
	}
}

func TestStopTask_Success(t *testing.T) {
	tr := &rewindMockTransport{}

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-st","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0
	tr.matchRequestID = true

	err = client.StopTask(context.Background(), "task-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sent := tr.Sent()
	var found bool
	for _, line := range sent {
		var msg map[string]any
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		if msg["type"] == "control_request" {
			req, _ := msg["request"].(map[string]any)
			if req["subtype"] == "stop_task" && req["task_id"] == "task-abc" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("stop_task control request was not sent with correct payload")
	}
}

func TestStopTask_Error(t *testing.T) {
	tr := &rewindMockTransport{matchRequestID: true}
	tr.errorMessage = "task not found"

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-st","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0

	err = client.StopTask(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
