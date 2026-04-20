package claude_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

var (
	result1JSON = []byte(`{"type":"result","subtype":"success","result":"done","cost_usd":0.001,"duration_ms":100,"is_error":false,"session_id":"s1","num_turns":1,"total_input_tokens":10,"total_output_tokens":5}`)
	result2JSON = []byte(`{"type":"result","subtype":"success","result":"done","cost_usd":0.001,"duration_ms":100,"is_error":false,"session_id":"s1","num_turns":2,"total_input_tokens":20,"total_output_tokens":10}`)
)

// multiTransport serves successive batches of JSON lines, one batch per Start/Close cycle.
type multiTransport struct {
	batches [][][]byte
	idx     int
	pos     int
	mu      sync.Mutex
}

func (t *multiTransport) Start(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pos = 0
	return nil
}
func (t *multiTransport) Send(_ []byte) error { return nil }
func (t *multiTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.idx++
	return nil
}
func (t *multiTransport) Receive() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.idx >= len(t.batches) || t.pos >= len(t.batches[t.idx]) {
		return nil, io.EOF
	}
	line := t.batches[t.idx][t.pos]
	t.pos++
	return line, nil
}

// readyTransport blocks in Receive() until the context from Start() is cancelled,
// and signals via Started when Start() is first called.
type readyTransport struct {
	Started chan struct{}
	ctx     context.Context
	once    sync.Once
	mu      sync.Mutex
	closed  bool
}

func newReadyTransport() *readyTransport {
	return &readyTransport{Started: make(chan struct{})}
}

func (t *readyTransport) Start(ctx context.Context) error {
	t.ctx = ctx
	t.once.Do(func() { close(t.Started) })
	return nil
}
func (t *readyTransport) Send(_ []byte) error { return nil }
func (t *readyTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}
func (t *readyTransport) Receive() ([]byte, error) {
	<-t.ctx.Done()
	return nil, io.EOF
}

func TestClient_SequentialQueries(t *testing.T) {
	tr := &multiTransport{
		batches: [][][]byte{
			{assistantJSON, result1JSON},
			{assistantJSON, result2JSON},
		},
	}

	ctx := context.Background()
	client, err := claude.NewClient(ctx, agent.WithTransport(tr))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	for i, prompt := range []string{"hello", "follow-up"} {
		var count int
		for msg, err := range client.Query(ctx, prompt) {
			if err != nil {
				t.Fatalf("query %d: unexpected error: %v", i+1, err)
			}
			_ = msg
			count++
		}
		if count != 2 {
			t.Fatalf("query %d: expected 2 messages, got %d", i+1, count)
		}
	}
}

func TestClient_ConcurrentQueryReturnsError(t *testing.T) {
	rt := newReadyTransport()
	ctx := context.Background()
	client, err := claude.NewClient(ctx, agent.WithTransport(rt))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	go func() {
		for _, _ = range client.Query(ctx, "blocking") {
		}
	}()

	<-rt.Started

	var gotErr error
	for _, err := range client.Query(ctx, "concurrent") {
		gotErr = err
		break
	}
	if !errors.Is(gotErr, claude.ErrClientBusy) {
		t.Fatalf("expected ErrClientBusy, got %v", gotErr)
	}
}

func TestClient_CloseTerminatesTransport(t *testing.T) {
	rt := newReadyTransport()
	ctx := context.Background()
	client, err := claude.NewClient(ctx, agent.WithTransport(rt))
	if err != nil {
		t.Fatal(err)
	}

	var gotErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, err := range client.Query(ctx, "hello") {
			if err != nil {
				gotErr = err
				return
			}
		}
	}()

	<-rt.Started
	client.Close()
	<-done

	if gotErr != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", gotErr)
	}
	rt.mu.Lock()
	closed := rt.closed
	rt.mu.Unlock()
	if !closed {
		t.Fatal("transport not closed after client close")
	}
}

func TestClient_WorksWithCustomTransport(t *testing.T) {
	tr := &seqTransport{lines: [][]byte{assistantJSON, resultJSON}}
	ctx := context.Background()
	client, err := claude.NewClient(ctx, agent.WithTransport(tr))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var msgs []agent.Message
	for msg, err := range client.Query(ctx, "hello") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if _, ok := msgs[1].(*agent.ResultMessage); !ok {
		t.Errorf("last message should be ResultMessage, got %T", msgs[1])
	}
}

func TestClient_ClosedClientReturnsError(t *testing.T) {
	tr := &seqTransport{lines: [][]byte{resultJSON}}
	ctx := context.Background()
	client, err := claude.NewClient(ctx, agent.WithTransport(tr))
	if err != nil {
		t.Fatal(err)
	}
	client.Close()

	var gotErr error
	for _, err := range client.Query(ctx, "hello") {
		gotErr = err
		break
	}
	if !errors.Is(gotErr, claude.ErrClientClosed) {
		t.Fatalf("expected ErrClientClosed, got %v", gotErr)
	}
}
