package claude_test

import (
	"context"
	"io"
	"sync"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

// seqTransport is a Transporter that serves pre-loaded JSON lines then returns io.EOF.
type seqTransport struct {
	lines  [][]byte
	pos    int
	closed bool
	mu     sync.Mutex
}

func (t *seqTransport) Start(_ context.Context) error { return nil }
func (t *seqTransport) Send(_ []byte) error           { return nil }
func (t *seqTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}
func (t *seqTransport) Receive() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pos >= len(t.lines) {
		return nil, io.EOF
	}
	line := t.lines[t.pos]
	t.pos++
	return line, nil
}

// ctxTransport blocks Receive() until the context from Start() is cancelled.
type ctxTransport struct {
	ctx    context.Context
	closed bool
	mu     sync.Mutex
}

func (t *ctxTransport) Start(ctx context.Context) error {
	t.ctx = ctx
	return nil
}
func (t *ctxTransport) Send(_ []byte) error { return nil }
func (t *ctxTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}
func (t *ctxTransport) Receive() ([]byte, error) {
	<-t.ctx.Done()
	return nil, io.EOF
}

var (
	assistantJSON = []byte(`{"type":"assistant","role":"assistant","content":[{"type":"text","text":"Hello!"}]}`)
	systemJSON    = []byte(`{"type":"system","content":"initialized"}`)
	userJSON      = []byte(`{"type":"user","role":"user","content":[{"type":"text","text":"Hi"}]}`)
	resultJSON    = []byte(`{"type":"result","subtype":"success","result":"done","cost_usd":0.001,"duration_ms":100,"is_error":false,"session_id":"s1","num_turns":1,"total_input_tokens":10,"total_output_tokens":5}`)
)

func TestQuery_YieldsMessages(t *testing.T) {
	tr := &seqTransport{lines: [][]byte{assistantJSON, resultJSON}}
	var msgs []agent.Message
	for msg, err := range claude.Query(context.Background(), "hello", agent.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if _, ok := msgs[0].(*agent.AssistantMessage); !ok {
		t.Errorf("msgs[0]: want *AssistantMessage, got %T", msgs[0])
	}
	if _, ok := msgs[1].(*agent.ResultMessage); !ok {
		t.Errorf("msgs[1]: want *ResultMessage, got %T", msgs[1])
	}
}

func TestQuery_AllMessageTypes(t *testing.T) {
	tr := &seqTransport{lines: [][]byte{systemJSON, assistantJSON, userJSON, resultJSON}}
	var msgs []agent.Message
	for msg, err := range claude.Query(context.Background(), "hello", agent.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if _, ok := msgs[0].(*agent.SystemMessage); !ok {
		t.Errorf("msgs[0]: want *SystemMessage, got %T", msgs[0])
	}
	if _, ok := msgs[1].(*agent.AssistantMessage); !ok {
		t.Errorf("msgs[1]: want *AssistantMessage, got %T", msgs[1])
	}
	if _, ok := msgs[2].(*agent.UserMessage); !ok {
		t.Errorf("msgs[2]: want *UserMessage, got %T", msgs[2])
	}
	if _, ok := msgs[3].(*agent.ResultMessage); !ok {
		t.Errorf("msgs[3]: want *ResultMessage, got %T", msgs[3])
	}
}

func TestQuery_EarlyBreakCleansUpTransport(t *testing.T) {
	tr := &seqTransport{lines: [][]byte{assistantJSON, assistantJSON, resultJSON}}
	count := 0
	for msg, err := range claude.Query(context.Background(), "hello", agent.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = msg
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected 1 message before break, got %d", count)
	}
	tr.mu.Lock()
	closed := tr.closed
	tr.mu.Unlock()
	if !closed {
		t.Fatal("transport not closed after early break")
	}
}

func TestQuery_ContextCancellationPropagates(t *testing.T) {
	bt := &ctxTransport{}
	ctx, cancel := context.WithCancel(context.Background())

	var gotErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, err := range claude.Query(ctx, "hello", agent.WithTransport(bt)) {
			if err != nil {
				gotErr = err
				return
			}
		}
	}()

	cancel()
	<-done

	if gotErr != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", gotErr)
	}
	bt.mu.Lock()
	closed := bt.closed
	bt.mu.Unlock()
	if !closed {
		t.Fatal("transport not closed after context cancellation")
	}
}

func TestQuery_ResultMessageTerminatesStream(t *testing.T) {
	// Extra lines after result should not be yielded.
	tr := &seqTransport{lines: [][]byte{resultJSON, assistantJSON}}
	var msgs []agent.Message
	for msg, err := range claude.Query(context.Background(), "hello", agent.WithTransport(tr)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (only result), got %d", len(msgs))
	}
}
