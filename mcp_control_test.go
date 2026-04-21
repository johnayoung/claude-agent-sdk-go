package claude_test

import (
	"context"
	"encoding/json"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest"
)

func TestReconnectMCPServer_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.ReconnectMCPServer(context.Background(), "my-server")
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestReconnectMCPServer_ClientClosed(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	err = client.ReconnectMCPServer(context.Background(), "my-server")
	if err != claude.ErrClientClosed {
		t.Fatalf("expected ErrClientClosed, got: %v", err)
	}
}

func TestReconnectMCPServer_Success(t *testing.T) {
	tr := &rewindMockTransport{}

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-mcp","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0
	tr.matchRequestID = true

	err = client.ReconnectMCPServer(context.Background(), "my-server")
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
			if req["subtype"] == "mcp_reconnect" && req["serverName"] == "my-server" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("mcp_reconnect control request was not sent")
	}
}

func TestReconnectMCPServer_Error(t *testing.T) {
	tr := &rewindMockTransport{matchRequestID: true}
	tr.errorMessage = "server not found"

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-mcp","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0

	err = client.ReconnectMCPServer(context.Background(), "unknown-server")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestToggleMCPServer_Success(t *testing.T) {
	tr := &rewindMockTransport{}

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-mcp","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0
	tr.matchRequestID = true

	err = client.ToggleMCPServer(context.Background(), "my-server", false)
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
			if req["subtype"] == "mcp_toggle" && req["serverName"] == "my-server" && req["enabled"] == false {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("mcp_toggle control request was not sent with correct payload")
	}
}

func TestToggleMCPServer_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.ToggleMCPServer(context.Background(), "my-server", true)
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestGetMCPStatus_NoSession(t *testing.T) {
	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(agenttest.NewMockTransportFromLines()),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetMCPStatus(context.Background())
	if err != claude.ErrNoSession {
		t.Fatalf("expected ErrNoSession, got: %v", err)
	}
}

func TestGetMCPStatus_Success(t *testing.T) {
	tr := &typedResponseMockTransport{}

	client, err := claude.NewClient(context.Background(),
		claude.WithTransport(tr),
	)
	if err != nil {
		t.Fatal(err)
	}

	tr.receiveLines = [][]byte{
		initJSON,
		[]byte(`{"type":"result","subtype":"success","result":"","duration_ms":1,"duration_api_ms":1,"is_error":false,"session_id":"sess-mcp","num_turns":0}`),
	}
	for _, err := range client.Query(context.Background(), "setup") {
		if err != nil {
			t.Fatal(err)
		}
	}

	tr.receiveLines = [][]byte{}
	tr.receiveIdx = 0
	tr.responsePayload = `{"mcpServers":[{"name":"test-server","status":"connected"}]}`

	status, err := client.GetMCPStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(status.MCPServers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(status.MCPServers))
	}
	if status.MCPServers[0].Name != "test-server" {
		t.Errorf("unexpected server name: %q", status.MCPServers[0].Name)
	}
	if status.MCPServers[0].Status != claude.McpStatusConnected {
		t.Errorf("unexpected status: %q", status.MCPServers[0].Status)
	}
}

// typedResponseMockTransport is a test transport that returns typed response payloads
// in control responses. It dynamically matches request IDs.
type typedResponseMockTransport struct {
	receiveLines    [][]byte
	receiveIdx      int
	sent            [][]byte
	responsePayload string
	errorMessage    string
}

func (m *typedResponseMockTransport) Start(_ context.Context) error { return nil }
func (m *typedResponseMockTransport) Close() error                  { return nil }

func (m *typedResponseMockTransport) Send(line []byte) error {
	cp := make([]byte, len(line))
	copy(cp, line)
	m.sent = append(m.sent, cp)

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
			if m.responsePayload != "" && m.errorMessage == "" {
				var payload any
				json.Unmarshal([]byte(m.responsePayload), &payload)
				resp["response"] = payload
			}
			envelope := map[string]any{
				"type":     "control_response",
				"response": resp,
			}
			data, _ := json.Marshal(envelope)
			m.receiveLines = append(m.receiveLines, data)
		}
	}
	return nil
}

func (m *typedResponseMockTransport) Receive() ([]byte, error) {
	if m.receiveIdx >= len(m.receiveLines) {
		return nil, nil
	}
	line := m.receiveLines[m.receiveIdx]
	m.receiveIdx++
	return line, nil
}

func (m *typedResponseMockTransport) Sent() [][]byte {
	result := make([][]byte, len(m.sent))
	for i, b := range m.sent {
		cp := make([]byte, len(b))
		copy(cp, b)
		result[i] = cp
	}
	return result
}
