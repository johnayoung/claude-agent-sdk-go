package claude

import (
	"context"
	"errors"
	"io"
	"iter"
	"os"
	"sync"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
	"github.com/johnayoung/claude-agent-sdk-go/internal/parse"
	"github.com/johnayoung/claude-agent-sdk-go/transport"
)

// ErrClientBusy is returned when Query is called while another query is in progress.
var ErrClientBusy = errors.New("claude: client has a query in progress")

// ErrClientClosed is returned when Query is called on a closed Client.
var ErrClientClosed = errors.New("claude: client is closed")

// Client maintains a multi-turn conversation session with the Claude CLI.
// Each Query resumes the same session using the session ID from the previous result.
// Client is safe to use from a single goroutine; concurrent Query calls return ErrClientBusy.
type Client struct {
	opts      *agent.Options
	sessionID string
	mu        sync.Mutex
	busy      bool
	closed    bool
	cancelFn  context.CancelFunc
}

// NewClient creates a Client configured with the given options.
// Returns an error if the CLI binary cannot be found and no custom Transport is provided.
func NewClient(ctx context.Context, opts ...agent.Option) (*Client, error) {
	o := agent.NewOptions(opts)
	if o.Transport == nil && o.CLIPath == "" {
		return nil, &agent.CLINotFoundError{SearchPath: os.Getenv("PATH")}
	}
	return &Client{opts: o}, nil
}

// Query sends prompt to the Claude CLI and returns a streaming iterator over the response.
// If a previous Query produced a ResultMessage with a session ID, the conversation is resumed
// automatically via --resume. Returns ErrClientBusy or ErrClientClosed on misuse.
func (c *Client) Query(ctx context.Context, prompt string) iter.Seq2[agent.Message, error] {
	return func(yield func(agent.Message, error) bool) {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			yield(nil, ErrClientClosed)
			return
		}
		if c.busy {
			c.mu.Unlock()
			yield(nil, ErrClientBusy)
			return
		}
		c.busy = true
		sessionID := c.sessionID
		qCtx, cancel := context.WithCancel(ctx)
		c.cancelFn = cancel
		c.mu.Unlock()

		defer func() {
			cancel()
			c.mu.Lock()
			c.busy = false
			c.cancelFn = nil
			c.mu.Unlock()
		}()

		var tr agent.Transporter
		if c.opts.Transport != nil {
			tr = c.opts.Transport
		} else {
			args := buildClientArgs(prompt, c.opts, sessionID)
			tr = transport.New(args, transport.WithCLIPath(c.opts.CLIPath))
		}

		if err := tr.Start(qCtx); err != nil {
			yield(nil, err)
			return
		}
		defer tr.Close()

		for {
			line, err := tr.Receive()
			if err != nil {
				if err == io.EOF {
					if qCtx.Err() != nil {
						yield(nil, qCtx.Err())
					}
					return
				}
				yield(nil, err)
				return
			}

			msg, parseErr := parse.ParseLine(line)
			if parseErr != nil {
				if !yield(nil, parseErr) {
					return
				}
				continue
			}

			if result, ok := msg.(*agent.ResultMessage); ok {
				if result.SessionID != "" {
					c.mu.Lock()
					c.sessionID = result.SessionID
					c.mu.Unlock()
				}
				if !yield(msg, nil) {
					return
				}
				return
			}

			if !yield(msg, nil) {
				return
			}
		}
	}
}

// Close terminates the Client and interrupts any in-flight Query.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.cancelFn != nil {
		c.cancelFn()
	}
	return nil
}

func buildClientArgs(prompt string, o *agent.Options, sessionID string) []string {
	args := buildQueryArgs(prompt, o)
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	return args
}
