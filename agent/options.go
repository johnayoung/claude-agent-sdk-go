package agent

import (
	"context"
	"os/exec"

	"github.com/johnayoung/claude-agent-sdk-go/hooks"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

// PermissionMode aliases permission.Mode for convenience.
type PermissionMode = permission.Mode

const (
	PermissionModeDefault           = permission.ModeDefault
	PermissionModeAcceptEdits       = permission.ModeAcceptEdits
	PermissionModePlan              = permission.ModePlan
	PermissionModeBypassPermissions = permission.ModeBypassPermissions
	PermissionModeDontAsk           = permission.ModeDontAsk
	PermissionModeAuto              = permission.ModeAuto
)

// CanUseToolFunc aliases permission.CanUseToolFunc for convenience.
type CanUseToolFunc = permission.CanUseToolFunc

// Transporter is the communication layer interface. transport.SubprocessTransport satisfies this.
type Transporter interface {
	Start(ctx context.Context) error
	Send(line []byte) error
	Receive() ([]byte, error)
	Close() error
}

// MCPServerType identifies the protocol used by an MCP server.
type MCPServerType string

const (
	MCPServerTypeStdio MCPServerType = "stdio"
	MCPServerTypeSSE   MCPServerType = "sse"
	MCPServerTypeHTTP  MCPServerType = "http"
	MCPServerTypeSDK   MCPServerType = "sdk"
)

// MCPServerConfig holds configuration for an MCP server connection.
type MCPServerConfig struct {
	Name    string
	Type    MCPServerType
	Command string
	Args    []string
	URL     string
	Env     map[string]string
}

// Options holds all configurable parameters for Query and Client.
type Options struct {
	SystemPrompt   string
	MaxTurns       int
	CLIPath        string
	Transport      Transporter
	PermissionMode PermissionMode
	CanUseTool     CanUseToolFunc
	Hooks          *hooks.Hooks
	MCPServers     []MCPServerConfig
	WorkingDir     string
}

// Option is a functional option that configures Options.
type Option func(*Options)

// WithSystemPrompt sets the system prompt prepended to every conversation.
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) { o.SystemPrompt = prompt }
}

// WithMaxTurns limits the number of agentic turns (0 = unlimited).
func WithMaxTurns(n int) Option {
	return func(o *Options) { o.MaxTurns = n }
}

// WithCLIPath sets an explicit path to the claude CLI binary, bypassing PATH discovery.
func WithCLIPath(path string) Option {
	return func(o *Options) { o.CLIPath = path }
}

// WithTransport replaces the default subprocess transport with a custom implementation.
func WithTransport(t Transporter) Option {
	return func(o *Options) { o.Transport = t }
}

// WithPermissionMode sets the permission handling mode for tool execution.
func WithPermissionMode(mode PermissionMode) Option {
	return func(o *Options) { o.PermissionMode = mode }
}

// WithCanUseTool registers a callback that approves or denies individual tool calls.
func WithCanUseTool(fn CanUseToolFunc) Option {
	return func(o *Options) { o.CanUseTool = fn }
}

// WithHooks attaches a hook registry to receive agent lifecycle events.
func WithHooks(h *hooks.Hooks) Option {
	return func(o *Options) { o.Hooks = h }
}

// WithMCPServers appends MCP server configurations used by the agent.
func WithMCPServers(servers ...MCPServerConfig) Option {
	return func(o *Options) { o.MCPServers = append(o.MCPServers, servers...) }
}

// WithWorkingDir sets the working directory for the CLI subprocess.
func WithWorkingDir(dir string) Option {
	return func(o *Options) { o.WorkingDir = dir }
}

// NewOptions builds an Options by applying the given options over defaults.
func NewOptions(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	applyDefaults(o)
	return o
}

// applyDefaults fills zero-value fields with sensible defaults.
func applyDefaults(o *Options) {
	if o.CLIPath == "" {
		if p, err := exec.LookPath("claude"); err == nil {
			o.CLIPath = p
		}
	}
	if o.PermissionMode == "" {
		o.PermissionMode = PermissionModeDefault
	}
}
