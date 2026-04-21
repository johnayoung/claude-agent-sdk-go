package claude_test

import (
	"context"
	"encoding/json"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest"
)

func TestRewindFiles_CheckpointingDisabled(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.RewindFiles(context.Background(), "msg-123")
	if err != claude.ErrCheckpointingDisabled {
		t.Fatalf("expected ErrCheckpointingDisabled, got: %v", err)
	}
}

func TestRewindFiles_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
		claude.WithFileCheckpointing(),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.RewindFiles(context.Background(), "msg-123")
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestRewindFiles_ClientClosed(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
		claude.WithFileCheckpointing(),
	)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	err = client.RewindFiles(context.Background(), "msg-123")
	if err != claude.ErrClientClosed {
		t.Fatalf("expected ErrClientClosed, got: %v", err)
	}
}

func TestRewindFiles_Success(t *testing.T) {
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-abc","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines([]byte(resultMsg))

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
		claude.WithFileCheckpointing(),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Establish session via a query
	for msg, err := range client.Query(context.Background(), "hello") {
		if err != nil {
			t.Fatal(err)
		}
		_ = msg
	}
	if client.SessionID() != "sess-abc" {
		t.Fatalf("expected session sess-abc, got %q", client.SessionID())
	}

	// Test RewindFiles with a fresh client that uses a mock transport
	// that dynamically matches request IDs.
	rewindTr := &rewindMockTransport{}

	client2, err := claude.NewClient(context.Background(),
		claude.WithTransport(rewindTr),
		claude.WithFileCheckpointing(),
	)
	if err != nil {
		t.Fatal(err)
	}
	// Establish session — init response + result
	rewindTr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-abc","num_turns":0}`),
	}
	for _, err := range client2.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	// Now set up for rewind: transport returns a success control_response
	rewindTr.receiveLines = [][]byte{}
	rewindTr.receiveIdx = 0
	rewindTr.matchRequestID = true

	err = client2.RewindFiles(context.Background(), "user-msg-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the control request was sent
	sent := rewindTr.Sent()
	var foundRewind bool
	for _, line := range sent {
		var msg map[string]any
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		if msg["type"] == "control_request" {
			req, _ := msg["request"].(map[string]any)
			if req["subtype"] == "rewind_files" && req["user_message_id"] == "user-msg-456" {
				foundRewind = true
			}
		}
	}
	if !foundRewind {
		t.Fatal("rewind_files control request was not sent")
	}
}

func TestRewindFiles_ErrorResponse(t *testing.T) {
	rewindTr := &rewindMockTransport{matchRequestID: true}
	rewindTr.errorMessage = "checkpoint not found"

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(rewindTr),
		claude.WithFileCheckpointing(),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Establish session — init response + result
	rewindTr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-xyz","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	// Set up error response for rewind
	rewindTr.receiveLines = [][]byte{}
	rewindTr.receiveIdx = 0

	err = client.RewindFiles(context.Background(), "nonexistent-msg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "claude: rewind failed: checkpoint not found" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestControlRequest_RewindFiles_FromCLI(t *testing.T) {
	controlReq := `{"type":"control_request","request_id":"req-rw","request":{"subtype":"rewind_files","user_message_id":"msg-789"}}`
	resultMsg := `{"type":"result","subtype":"success","result":"done","duration_ms":100,"duration_api_ms":50,"is_error":false,"session_id":"sess-1","num_turns":1}`

	tr := agenttest.NewMockTransportFromLines(
		[]byte(controlReq),
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

	// Verify control_request is not yielded to caller
	for _, msg := range messages {
		if msg.MessageType() == "control_request" {
			t.Fatal("control_request should not be yielded")
		}
	}

	// Verify success response was sent back for the rewind request
	sent := tr.Sent()
	var foundRewindResp bool
	for _, line := range sent {
		var resp map[string]any
		if json.Unmarshal(line, &resp) != nil {
			continue
		}
		if resp["type"] != "control_response" {
			continue
		}
		respBody, _ := resp["response"].(map[string]any)
		if respBody["request_id"] == "req-rw" {
			foundRewindResp = true
			if respBody["subtype"] != "success" {
				t.Errorf("expected success, got: %v", respBody["subtype"])
			}
		}
	}
	if !foundRewindResp {
		t.Fatal("no control_response for req-rw found in sent messages")
	}
}

// rewindMockTransport is a test transport that dynamically matches request IDs
// in control responses. This allows testing the RewindFiles round-trip without
// knowing the generated request ID in advance.
type rewindMockTransport struct {
	receiveLines   [][]byte
	receiveIdx     int
	sent           [][]byte
	matchRequestID bool
	errorMessage   string
}

func (m *rewindMockTransport) Start(_ context.Context) error { return nil }
func (m *rewindMockTransport) Close() error                  { return nil }

func (m *rewindMockTransport) Send(line []byte) error {
	cp := make([]byte, len(line))
	copy(cp, line)
	m.sent = append(m.sent, cp)

	if m.matchRequestID {
		var msg map[string]any
		if json.Unmarshal(line, &msg) == nil && msg["type"] == "control_request" {
			reqID, _ := msg["request_id"].(string)
			if reqID != "" {
				subtype := "success"
				if m.errorMessage != "" {
					subtype = "error"
				}
				resp := map[string]any{
					"subtype":    subtype,
					"request_id": reqID,
				}
				if m.errorMessage != "" {
					resp["error"] = m.errorMessage
				}
				envelope := map[string]any{
					"type":     "control_response",
					"response": resp,
				}
				data, _ := json.Marshal(envelope)
				m.receiveLines = append(m.receiveLines, data)
			}
		}
	}
	return nil
}

func (m *rewindMockTransport) Receive() ([]byte, error) {
	if m.receiveIdx >= len(m.receiveLines) {
		return nil, nil
	}
	line := m.receiveLines[m.receiveIdx]
	m.receiveIdx++
	return line, nil
}

func (m *rewindMockTransport) Sent() [][]byte {
	result := make([][]byte, len(m.sent))
	for i, b := range m.sent {
		cp := make([]byte, len(b))
		copy(cp, b)
		result[i] = cp
	}
	return result
}
