package claude

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/johnayoung/claude-agent-sdk-go/internal/transport"
)

// ErrCheckpointingDisabled is returned when RewindFiles is called without file checkpointing enabled.
var ErrCheckpointingDisabled = errors.New("claude: file checkpointing is not enabled")

// ErrNoSession is returned when RewindFiles is called before a session has been established.
var ErrNoSession = errors.New("claude: no session ID available for rewind")

// RewindFiles sends a rewind_files control request to the CLI, restoring file state
// to the checkpoint associated with the given user message ID. File checkpointing
// must be enabled via WithFileCheckpointing and a session must already be established.
func (c *Client) RewindFiles(ctx context.Context, userMessageID string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClientClosed
	}
	if c.busy {
		c.mu.Unlock()
		return ErrClientBusy
	}
	if !c.opts.FileCheckpointing {
		c.mu.Unlock()
		return ErrCheckpointingDisabled
	}
	sessionID := c.sessionID
	c.mu.Unlock()

	if sessionID == "" {
		return ErrNoSession
	}

	var tr Transporter
	if c.opts.Transport != nil {
		tr = c.opts.Transport
	} else {
		args := buildRewindArgs(c.opts, sessionID)
		trOpts := []transport.Option{transport.WithCLIPath(c.opts.CLIPath)}
		if c.opts.WorkingDir != "" {
			trOpts = append(trOpts, transport.WithWorkingDir(c.opts.WorkingDir))
		}
		if env := buildEnv(c.opts); len(env) > 0 {
			trOpts = append(trOpts, transport.WithEnv(env))
		}
		if c.opts.Stderr != nil {
			trOpts = append(trOpts, transport.WithStderrCallback(c.opts.Stderr))
		}
		tr = transport.New(args, trOpts...)
	}

	if err := tr.Start(ctx); err != nil {
		return fmt.Errorf("claude: rewind transport start: %w", err)
	}
	defer tr.Close()

	if err := sendInitializeRequest(tr, c.opts); err != nil {
		return fmt.Errorf("claude: rewind initialize: %w", err)
	}

	reqID := generateRequestID()
	if err := sendRewindFilesRequest(tr, reqID, userMessageID); err != nil {
		return fmt.Errorf("claude: rewind send: %w", err)
	}

	return awaitControlResponse(ctx, tr, reqID)
}

func buildRewindArgs(o *Options, sessionID string) []string {
	args := []string{
		"--print", " ",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
		"--resume", sessionID,
		"--continue",
		"--max-turns", "1",
	}
	if o.PermissionMode != "" && o.PermissionMode != PermissionModeDefault {
		args = append(args, "--permission-mode", string(o.PermissionMode))
	}
	return args
}

type sdkControlRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	Request   any    `json:"request"`
}

type rewindFilesPayload struct {
	Subtype       string `json:"subtype"`
	UserMessageID string `json:"user_message_id"`
}

func sendRewindFilesRequest(tr Transporter, reqID, userMessageID string) error {
	req := sdkControlRequest{
		Type:      "control_request",
		RequestID: reqID,
		Request: rewindFilesPayload{
			Subtype:       "rewind_files",
			UserMessageID: userMessageID,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return tr.Send(data)
}

func awaitControlResponse(ctx context.Context, tr Transporter, reqID string) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line, err := tr.Receive()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("claude: transport closed before rewind response received")
			}
			return fmt.Errorf("claude: rewind receive: %w", err)
		}

		var envelope struct {
			Type     string `json:"type"`
			Response json.RawMessage `json:"response"`
		}
		if json.Unmarshal(line, &envelope) != nil {
			continue
		}
		if envelope.Type != "control_response" {
			continue
		}

		var resp struct {
			Subtype   string `json:"subtype"`
			RequestID string `json:"request_id"`
			Error     string `json:"error,omitempty"`
		}
		if json.Unmarshal(envelope.Response, &resp) != nil {
			continue
		}
		if resp.RequestID != reqID {
			continue
		}

		if resp.Subtype == "error" {
			return fmt.Errorf("claude: rewind failed: %s", resp.Error)
		}
		return nil
	}
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "req_rewind_" + hex.EncodeToString(b)
}
