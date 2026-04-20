package claude

import (
	"context"
	"os/exec"

	"github.com/johnayoung/claude-agent-sdk-go/hooks"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

// ThinkingConfig controls extended thinking behavior.
type ThinkingConfig struct {
	Type         string // "adaptive", "enabled", "disabled"
	BudgetTokens int    // only used when Type is "enabled"
}

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

// Transporter is the communication layer interface. The default implementation
// uses subprocess communication with the Claude CLI.
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
	// Core
	Model                string
	FallbackModel        string
	SystemPrompt         string
	MaxTurns             int
	MaxBudgetUSD         float64
	Effort               string // "low", "medium", "high", "max"
	Thinking             *ThinkingConfig
	SessionID            string
	ContinueConversation bool

	// Tools
	Tools           []string
	AllowedTools    []string
	DisallowedTools []string

	// MCP
	MCPServers []MCPServerConfig

	// Permissions
	PermissionMode PermissionMode
	CanUseTool     CanUseToolFunc

	// Agents
	Agents map[string]AgentDefinition

	// Hooks
	Hooks *hooks.Hooks

	// Environment
	CLIPath            string
	Transport          Transporter
	WorkingDir         string
	Sandbox            *SandboxConfig
	FileCheckpointing  bool
	Betas              []string
	Skills             []string
	SettingSources     []string
	IncludePartialMsgs bool
}

// Option is a functional option that configures Options.
type Option func(*Options)

// WithModel sets the Claude model to use (e.g. "sonnet", "opus", "haiku").
func WithModel(model string) Option {
	return func(o *Options) { o.Model = model }
}

// WithFallbackModel sets a fallback model if the primary is unavailable.
func WithFallbackModel(model string) Option {
	return func(o *Options) { o.FallbackModel = model }
}

// WithSystemPrompt sets the system prompt prepended to every conversation.
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) { o.SystemPrompt = prompt }
}

// WithMaxTurns limits the number of agentic turns (0 = unlimited).
func WithMaxTurns(n int) Option {
	return func(o *Options) { o.MaxTurns = n }
}

// WithMaxBudgetUSD sets a spending limit checked after each API call.
func WithMaxBudgetUSD(budget float64) Option {
	return func(o *Options) { o.MaxBudgetUSD = budget }
}

// WithEffort sets the reasoning effort level ("low", "medium", "high", "max").
func WithEffort(effort string) Option {
	return func(o *Options) { o.Effort = effort }
}

// WithThinking configures extended thinking behavior.
func WithThinking(cfg ThinkingConfig) Option {
	return func(o *Options) { o.Thinking = &cfg }
}

// WithSessionID explicitly sets a session ID to resume.
func WithSessionID(id string) Option {
	return func(o *Options) { o.SessionID = id }
}

// WithContinueConversation resumes the most recent conversation.
func WithContinueConversation() Option {
	return func(o *Options) { o.ContinueConversation = true }
}

// WithTools sets the list of tool names or presets (e.g. "claude_code").
func WithTools(tools ...string) Option {
	return func(o *Options) { o.Tools = tools }
}

// WithAllowedTools sets an allowlist of specific tool names.
func WithAllowedTools(tools ...string) Option {
	return func(o *Options) { o.AllowedTools = tools }
}

// WithDisallowedTools sets a blocklist of specific tool names.
func WithDisallowedTools(tools ...string) Option {
	return func(o *Options) { o.DisallowedTools = tools }
}

// WithMCPServers appends MCP server configurations used by the agent.
func WithMCPServers(servers ...MCPServerConfig) Option {
	return func(o *Options) { o.MCPServers = append(o.MCPServers, servers...) }
}

// WithPermissionMode sets the permission handling mode for tool execution.
func WithPermissionMode(mode PermissionMode) Option {
	return func(o *Options) { o.PermissionMode = mode }
}

// WithCanUseTool registers a callback that approves or denies individual tool calls.
func WithCanUseTool(fn CanUseToolFunc) Option {
	return func(o *Options) { o.CanUseTool = fn }
}

// WithAgents registers sub-agent definitions keyed by name.
func WithAgents(agents map[string]AgentDefinition) Option {
	return func(o *Options) { o.Agents = agents }
}

// WithHooks attaches a hook registry to receive agent lifecycle events.
func WithHooks(h *hooks.Hooks) Option {
	return func(o *Options) { o.Hooks = h }
}

// WithCLIPath sets an explicit path to the claude CLI binary, bypassing PATH discovery.
func WithCLIPath(path string) Option {
	return func(o *Options) { o.CLIPath = path }
}

// WithTransport replaces the default subprocess transport with a custom implementation.
func WithTransport(t Transporter) Option {
	return func(o *Options) { o.Transport = t }
}

// WithWorkingDir sets the working directory for the CLI subprocess.
func WithWorkingDir(dir string) Option {
	return func(o *Options) { o.WorkingDir = dir }
}

// WithSandbox configures sandbox execution settings.
func WithSandbox(cfg SandboxConfig) Option {
	return func(o *Options) { o.Sandbox = &cfg }
}

// WithFileCheckpointing enables automatic file state checkpointing.
func WithFileCheckpointing() Option {
	return func(o *Options) { o.FileCheckpointing = true }
}

// WithBetas enables beta feature flags.
func WithBetas(betas ...string) Option {
	return func(o *Options) { o.Betas = betas }
}

// WithSkills enables specific skills or "all".
func WithSkills(skills ...string) Option {
	return func(o *Options) { o.Skills = skills }
}

// WithSettingSources controls which settings files are loaded (e.g. "user", "project", "local").
func WithSettingSources(sources ...string) Option {
	return func(o *Options) { o.SettingSources = sources }
}

// WithIncludePartialMessages enables streaming partial content blocks.
func WithIncludePartialMessages() Option {
	return func(o *Options) { o.IncludePartialMsgs = true }
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
