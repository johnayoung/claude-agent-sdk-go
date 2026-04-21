package claude

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/johnayoung/claude-agent-sdk-go/internal/transport"
)

// ControlRequestMessage wraps an incoming control_request from the CLI.
// It implements Message so parseLine can return it, but it is NOT yielded to callers.
type ControlRequestMessage struct {
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
}

func (m *ControlRequestMessage) MessageType() string { return "control_request" }

// controlRequestSubtype extracts the subtype field from a raw request payload.
type controlRequestSubtype struct {
	Subtype string `json:"subtype"`
}

type sdkControlRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	Request   any    `json:"request"`
}

// --- Request subtypes ---

type ControlPermissionRequest struct {
	Subtype               string         `json:"subtype"`
	ToolName              string         `json:"tool_name"`
	Input                 map[string]any `json:"input"`
	PermissionSuggestions []any          `json:"permission_suggestions"`
	BlockedPath           *string        `json:"blocked_path"`
	ToolUseID             string         `json:"tool_use_id"`
	AgentID               string         `json:"agent_id,omitempty"`
}

type ControlHookCallbackRequest struct {
	Subtype    string `json:"subtype"`
	CallbackID string `json:"callback_id"`
	Input      any    `json:"input"`
	ToolUseID  *string `json:"tool_use_id"`
}

type ControlInterruptRequest struct {
	Subtype string `json:"subtype"`
}

type ControlSetPermissionModeRequest struct {
	Subtype string `json:"subtype"`
	Mode    string `json:"mode"`
}

type ControlMcpMessageRequest struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"server_name"`
	Message    any    `json:"message"`
}

type ControlRewindFilesRequest struct {
	Subtype       string `json:"subtype"`
	UserMessageID string `json:"user_message_id"`
}

type ControlMcpReconnectRequest struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
}

type ControlMcpToggleRequest struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
	Enabled    bool   `json:"enabled"`
}

type ControlStopTaskRequest struct {
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
}

// --- Response types ---

type ControlResponse struct {
	Subtype   string `json:"subtype"`
	RequestID string `json:"request_id"`
	Response  any    `json:"response"`
}

type ControlErrorResponse struct {
	Subtype   string `json:"subtype"`
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
}

type SDKControlResponse struct {
	Type     string `json:"type"`
	Response any    `json:"response"`
}

// --- Permission response payload ---

type permissionAllowResponse struct {
	Behavior           string `json:"behavior"`
	UpdatedInput       any    `json:"updatedInput"`
	UpdatedPermissions any    `json:"updatedPermissions,omitempty"`
}

type permissionDenyResponse struct {
	Behavior  string `json:"behavior"`
	Message   string `json:"message,omitempty"`
	Interrupt bool   `json:"interrupt,omitempty"`
}

// --- Reusable control request helpers ---

func buildControlTransportArgs(o *Options, sessionID string) []string {
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

func generateControlRequestID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return "req_" + prefix + "_" + hex.EncodeToString(b)
}

func sendControlRequest(tr Transporter, reqID string, payload any) error {
	req := sdkControlRequest{
		Type:      "control_request",
		RequestID: reqID,
		Request:   payload,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return tr.Send(data)
}

func awaitControlResponseTyped[T any](ctx context.Context, tr Transporter, reqID, opName string) (*T, error) {
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line, err := tr.Receive()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("claude: transport closed before %s response received", opName)
			}
			return nil, fmt.Errorf("claude: %s receive: %w", opName, err)
		}

		var env struct {
			Type     string          `json:"type"`
			Response json.RawMessage `json:"response"`
		}
		if json.Unmarshal(line, &env) != nil {
			continue
		}
		if env.Type != "control_response" {
			continue
		}

		var resp struct {
			Subtype   string          `json:"subtype"`
			RequestID string          `json:"request_id"`
			Error     string          `json:"error,omitempty"`
			Response  json.RawMessage `json:"response,omitempty"`
		}
		if json.Unmarshal(env.Response, &resp) != nil {
			continue
		}
		if resp.RequestID != reqID {
			continue
		}

		if resp.Subtype == "error" {
			return nil, fmt.Errorf("claude: %s failed: %s", opName, resp.Error)
		}

		var result T
		if len(resp.Response) > 0 && string(resp.Response) != "null" {
			if err := json.Unmarshal(resp.Response, &result); err != nil {
				return nil, fmt.Errorf("claude: %s response unmarshal: %w", opName, err)
			}
		}
		return &result, nil
	}
}

// startControlTransport checks client preconditions (closed, busy, no session),
// creates a transport, starts it, and sends the initialize request.
// The caller must defer tr.Close() on the returned transport.
func (c *Client) startControlTransport(ctx context.Context) (Transporter, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrClientClosed
	}
	if c.busy {
		c.mu.Unlock()
		return nil, ErrClientBusy
	}
	sessionID := c.sessionID
	c.mu.Unlock()

	if sessionID == "" {
		return nil, ErrNoSession
	}

	var tr Transporter
	if c.opts.Transport != nil {
		tr = c.opts.Transport
	} else {
		args := buildControlTransportArgs(c.opts, sessionID)
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
		return nil, fmt.Errorf("claude: control transport start: %w", err)
	}

	if err := sendInitializeRequest(tr, c.opts); err != nil {
		tr.Close()
		return nil, fmt.Errorf("claude: control initialize: %w", err)
	}

	return tr, nil
}
