package agenttest_test

import (
	"context"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest"
)

func TestMockTransport_ReplayMessages(t *testing.T) {
	text := agenttest.NewTextMessage("hello")
	result := agenttest.NewResultMessage("done", "sess-1")

	tr, err := agenttest.NewMockTransport(text, result)
	if err != nil {
		t.Fatalf("NewMockTransport: %v", err)
	}

	if err := tr.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer tr.Close()

	line1, err := tr.Receive()
	agenttest.AssertNoError(t, err)
	if line1 == nil {
		t.Fatal("expected non-nil line")
	}

	line2, err := tr.Receive()
	agenttest.AssertNoError(t, err)
	if line2 == nil {
		t.Fatal("expected non-nil line")
	}

	_, eofErr := tr.Receive()
	if eofErr == nil {
		t.Fatal("expected io.EOF after messages exhausted")
	}
}

func TestMockTransport_ImplementsTransporter(t *testing.T) {
	tr := agenttest.MustNewMockTransport()
	var _ claude.Transporter = tr
}

func TestMockTransport_Send(t *testing.T) {
	tr := agenttest.MustNewMockTransport()
	_ = tr.Send([]byte(`{"type":"ping"}`))
	sent := tr.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent line, got %d", len(sent))
	}
}

func TestNewTextMessage(t *testing.T) {
	msg := agenttest.NewTextMessage("greetings")
	if msg.Role != "assistant" {
		t.Errorf("expected role assistant, got %q", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(msg.Content))
	}
	tb, ok := msg.Content[0].(*claude.TextBlock)
	if !ok {
		t.Fatalf("expected *claude.TextBlock, got %T", msg.Content[0])
	}
	if tb.Text != "greetings" {
		t.Errorf("expected text %q, got %q", "greetings", tb.Text)
	}
}

func TestNewToolUseMessage(t *testing.T) {
	msg := agenttest.NewToolUseMessage("tu_1", "bash", map[string]string{"command": "ls"})
	tu, ok := msg.Content[0].(*claude.ToolUseBlock)
	if !ok {
		t.Fatalf("expected *claude.ToolUseBlock, got %T", msg.Content[0])
	}
	if tu.Name != "bash" {
		t.Errorf("expected name bash, got %q", tu.Name)
	}
	if tu.ID != "tu_1" {
		t.Errorf("expected id tu_1, got %q", tu.ID)
	}
}

func TestNewResultMessage(t *testing.T) {
	msg := agenttest.NewResultMessage("all done", "session-abc")
	if msg.Result != "all done" {
		t.Errorf("unexpected result: %q", msg.Result)
	}
	if msg.SessionID != "session-abc" {
		t.Errorf("unexpected session_id: %q", msg.SessionID)
	}
}

func TestAssertTextContent(t *testing.T) {
	msg := agenttest.NewTextMessage("expected text")
	agenttest.AssertTextContent(t, msg, "expected text")
}

func TestAssertToolUse(t *testing.T) {
	msg := agenttest.NewToolUseMessage("id1", "read_file", nil)
	tu := agenttest.AssertToolUse(t, msg, "read_file")
	if tu == nil {
		t.Fatal("expected non-nil ToolUseBlock")
	}
}

func TestAssertResult(t *testing.T) {
	msg := agenttest.NewResultMessage("result", "sid")
	rm := agenttest.AssertResult(t, msg)
	if rm.SessionID != "sid" {
		t.Errorf("unexpected session_id: %q", rm.SessionID)
	}
}

func TestMockTransport_RoundTrip(t *testing.T) {
	text := agenttest.NewTextMessage("round trip")
	result := agenttest.NewResultMessage("ok", "s1")

	tr := agenttest.MustNewMockTransport(text, result)
	_ = tr.Start(context.Background())
	defer tr.Close()

	line1, _ := tr.Receive()
	line2, _ := tr.Receive()

	if line1 == nil || line2 == nil {
		t.Fatal("expected two non-nil lines")
	}
}
