package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"
	"strconv"
	"strings"

	"github.com/johnayoung/claude-agent-sdk-go/internal/transport"
)

// Query launches the Claude CLI with prompt and streams back messages.
// The iterator terminates after a ResultMessage or on context cancellation.
// Transport is always cleaned up, even on early break.
func Query(ctx context.Context, prompt string, opts ...Option) iter.Seq2[Message, error] {
	return func(yield func(Message, error) bool) {
		o := NewOptions(opts)

		var tr Transporter
		if o.Transport != nil {
			tr = o.Transport
		} else {
			if o.CLIPath == "" {
				yield(nil, &CLINotFoundError{SearchPath: os.Getenv("PATH")})
				return
			}
			args := buildSDKArgs(o)
			trOpts := []transport.Option{transport.WithCLIPath(o.CLIPath)}
			if o.WorkingDir != "" {
				trOpts = append(trOpts, transport.WithWorkingDir(o.WorkingDir))
			}
			if env := buildEnv(o); len(env) > 0 {
				trOpts = append(trOpts, transport.WithEnv(env))
			}
			if o.Stderr != nil {
				trOpts = append(trOpts, transport.WithStderrCallback(o.Stderr))
			}
			tr = transport.New(args, trOpts...)
		}

		qCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		if err := tr.Start(qCtx); err != nil {
			yield(nil, err)
			return
		}
		defer tr.Close()

		if err := sendInitializeRequest(tr, o); err != nil {
			yield(nil, err)
			return
		}

		if err := waitForInitResponse(tr, qCtx); err != nil {
			yield(nil, err)
			return
		}

		if err := sendUserPrompt(tr, prompt); err != nil {
			yield(nil, err)
			return
		}

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

			msg, parseErr := parseLine(line)
			if parseErr != nil {
				if !yield(nil, parseErr) {
					return
				}
				continue
			}

			if msg == nil {
				continue
			}

			if _, ok := msg.(*controlResponseMessage); ok {
				continue
			}

			if mirror, ok := msg.(*transcriptMirrorMessage); ok {
				if o.SessionStore != nil {
					if errMsg := handleTranscriptMirror(qCtx, o.SessionStore, o.ProjectsDir, mirror); errMsg != nil {
						if !yield(errMsg, nil) {
							return
						}
					}
				}
				continue
			}

			if cr, ok := msg.(*ControlRequestMessage); ok {
				if err := handleControlRequest(qCtx, tr, o, cr, cancel); err != nil {
					if !yield(nil, err) {
						return
					}
				}
				continue
			}

			if !yield(msg, nil) {
				return
			}

			if _, ok := msg.(*ResultMessage); ok {
				return
			}
		}
	}
}

func buildQueryArgs(prompt string, o *Options) []string {
	args := []string{
		"--print", prompt,
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
	}
	return appendCommonArgs(args, o)
}

func buildSDKArgs(o *Options) []string {
	args := []string{
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
	}
	return appendCommonArgs(args, o)
}

func appendCommonArgs(args []string, o *Options) []string {
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.FallbackModel != "" {
		args = append(args, "--fallback-model", o.FallbackModel)
	}
	if o.SystemPromptSource != nil {
		switch o.SystemPromptSource.Type {
		case "file":
			args = append(args, "--system-prompt-file", o.SystemPromptSource.Path)
		case "preset":
			if o.SystemPromptSource.Append != "" {
				args = append(args, "--append-system-prompt", o.SystemPromptSource.Append)
			}
		}
	} else {
		args = append(args, "--system-prompt", o.SystemPrompt)
	}
	if o.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(o.MaxTurns))
	}
	if o.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", o.MaxBudgetUSD))
	}
	if o.TaskBudget != nil && o.TaskBudget.Total > 0 {
		args = append(args, "--task-budget", strconv.Itoa(o.TaskBudget.Total))
	}
	if o.Effort != "" {
		args = append(args, "--effort", o.Effort)
	}
	if o.Thinking != nil {
		switch o.Thinking.Type {
		case "enabled":
			args = append(args, "--max-thinking-tokens", strconv.Itoa(o.Thinking.BudgetTokens))
		case "disabled":
			args = append(args, "--thinking", "disabled")
		case "adaptive":
			args = append(args, "--thinking", "adaptive")
		}
	} else if o.MaxThinkingTokens > 0 {
		args = append(args, "--max-thinking-tokens", strconv.Itoa(o.MaxThinkingTokens))
	}
	if o.SessionID != "" {
		if o.ContinueConversation || o.ForkSession {
			args = append(args, "--resume", o.SessionID)
		} else {
			args = append(args, "--session-id", o.SessionID)
		}
	}
	if o.ContinueConversation {
		args = append(args, "--continue")
	}
	if o.ForkSession {
		args = append(args, "--fork-session")
	}
	if o.PermissionMode != "" && o.PermissionMode != PermissionModeDefault {
		args = append(args, "--permission-mode", string(o.PermissionMode))
	}
	if o.CanUseTool != nil && o.PermissionPromptTool == "" {
		args = append(args, "--permission-prompt-tool", "stdio")
	} else if o.PermissionPromptTool != "" {
		args = append(args, "--permission-prompt-tool", o.PermissionPromptTool)
	}
	if o.ToolsPreset != "" {
		args = append(args, "--tools", o.ToolsPreset)
	} else if o.Tools != nil {
		args = append(args, "--tools", strings.Join(o.Tools, ","))
	}
	if len(o.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(o.AllowedTools, ","))
	}
	if len(o.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(o.DisallowedTools, ","))
	}
	if len(o.MCPServers) > 0 {
		wrapper := buildMCPConfigWrapper(o.MCPServers)
		if data, err := json.Marshal(wrapper); err == nil {
			args = append(args, "--mcp-config", string(data))
		}
	}
	if len(o.Betas) > 0 {
		args = append(args, "--betas", strings.Join(o.Betas, ","))
	}
	if len(o.Skills) > 0 {
		args = append(args, "--skills", strings.Join(o.Skills, ","))
	}
	if len(o.SettingSources) > 0 {
		args = append(args, "--setting-sources="+strings.Join(o.SettingSources, ","))
	}
	for _, dir := range o.AddDirs {
		args = append(args, "--add-dir", dir)
	}
	if o.IncludePartialMsgs {
		args = append(args, "--include-partial-messages")
	}
	if o.SessionStore != nil {
		args = append(args, "--session-mirror")
	}
	for _, plugin := range o.Plugins {
		if plugin.Path != "" {
			args = append(args, "--plugin-dir", plugin.Path)
		}
	}
	if o.OutputFormat != nil {
		if data, err := json.Marshal(o.OutputFormat); err == nil {
			args = append(args, "--json-schema", string(data))
		}
	}
	if o.Settings != "" {
		args = append(args, "--settings", o.Settings)
	}
	for flag, val := range o.ExtraArgs {
		if val == nil {
			args = append(args, flag)
		} else {
			args = append(args, flag, *val)
		}
	}
	return args
}

type mcpServerEntry struct {
	Type    string            `json:"type"`
	Name    string            `json:"name,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type mcpConfigWrapper struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

func buildMCPConfigWrapper(servers []MCPServerConfig) mcpConfigWrapper {
	cfg := make(map[string]mcpServerEntry, len(servers))
	for _, s := range servers {
		entry := mcpServerEntry{
			Type: string(s.Type),
		}
		if s.Type == MCPServerTypeSDK {
			entry.Name = s.Name
		} else {
			entry.Command = s.Command
			entry.Args = s.Args
			entry.URL = s.URL
			entry.Env = s.Env
		}
		cfg[s.Name] = entry
	}
	return mcpConfigWrapper{MCPServers: cfg}
}

const sdkVersion = "0.1.0"

func buildEnv(o *Options) map[string]string {
	env := make(map[string]string, len(o.Env)+3)
	env["CLAUDE_CODE_ENTRYPOINT"] = "sdk-go"
	env["CLAUDE_AGENT_SDK_VERSION"] = sdkVersion
	if o.FileCheckpointing {
		env["CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING"] = "true"
	}
	for k, v := range o.Env {
		env[k] = v
	}
	return env
}

type initializePayload struct {
	Subtype                string                     `json:"subtype"`
	Hooks                  json.RawMessage            `json:"hooks"`
	Agents                 map[string]AgentDefinition `json:"agents,omitempty"`
	ExcludeDynamicSections *bool                      `json:"excludeDynamicSections,omitempty"`
	Skills                 []string                   `json:"skills,omitempty"`
}

type controlRequestEnvelope struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	Request   any    `json:"request"`
}

var requestCounter uint64

func nextRequestID() string {
	requestCounter++
	return fmt.Sprintf("req_%d_%08x", requestCounter, requestCounter)
}

func sendInitializeRequest(tr Transporter, o *Options) error {
	payload := initializePayload{
		Subtype: "initialize",
		Hooks:   json.RawMessage("null"),
	}

	if o.Agents != nil {
		payload.Agents = o.Agents
	}
	if o.ExcludeDynamicSections != nil {
		payload.ExcludeDynamicSections = o.ExcludeDynamicSections
	}
	if len(o.Skills) > 0 && o.Skills[0] != "all" {
		payload.Skills = o.Skills
	}

	envelope := controlRequestEnvelope{
		Type:      "control_request",
		RequestID: nextRequestID(),
		Request:   payload,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return tr.Send(data)
}

type userPromptMessage struct {
	Type            string         `json:"type"`
	SessionID       string         `json:"session_id"`
	Message         userMsgContent `json:"message"`
	ParentToolUseID *string        `json:"parent_tool_use_id"`
}

type userMsgContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func waitForInitResponse(tr Transporter, ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line, err := tr.Receive()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("CLI exited before sending initialize response")
			}
			return err
		}
		msg, parseErr := parseLine(line)
		if parseErr != nil {
			continue
		}
		if _, ok := msg.(*controlResponseMessage); ok {
			return nil
		}
	}
}

func sendUserPrompt(tr Transporter, prompt string) error {
	msg := userPromptMessage{
		Type:      "user",
		SessionID: "",
		Message:   userMsgContent{Role: "user", Content: prompt},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return tr.Send(data)
}
