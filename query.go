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

	"github.com/johnayoung/claude-agent-sdk-go/hooks"
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
			args := buildQueryArgs(prompt, o)
			trOpts := []transport.Option{transport.WithCLIPath(o.CLIPath)}
			if o.WorkingDir != "" {
				trOpts = append(trOpts, transport.WithWorkingDir(o.WorkingDir))
			}
			tr = transport.New(args, trOpts...)
		}

		if err := tr.Start(ctx); err != nil {
			yield(nil, err)
			return
		}
		defer tr.Close()

		for {
			line, err := tr.Receive()
			if err != nil {
				if err == io.EOF {
					if ctx.Err() != nil {
						yield(nil, ctx.Err())
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

			dispatchHooks(ctx, o, msg)

			if !yield(msg, nil) {
				return
			}

			if _, ok := msg.(*ResultMessage); ok {
				return
			}
		}
	}
}

func dispatchHooks(ctx context.Context, o *Options, msg Message) {
	if o.Hooks == nil {
		return
	}
	switch m := msg.(type) {
	case *AssistantMessage:
		for _, block := range m.Content {
			switch b := block.(type) {
			case *TextBlock:
				o.Hooks.DispatchModelResponse(ctx, &hooks.ModelResponseInput{
					Response: b.Text,
				})
			case *ToolUseBlock:
				var input map[string]any
				_ = json.Unmarshal(b.Input, &input)
				o.Hooks.DispatchPreToolUse(ctx, &hooks.PreToolUseInput{
					ToolName:  b.Name,
					ToolInput: input,
				})
			case *ToolResultBlock:
				o.Hooks.DispatchPostToolUse(ctx, &hooks.PostToolUseInput{
					ToolName:   "",
					ToolOutput: b.Content,
					IsError:    b.IsError,
				})
			}
		}
	case *ResultMessage:
		o.Hooks.DispatchStop(ctx, &hooks.StopInput{
			Reason:    m.Subtype,
			SessionID: m.SessionID,
		})
	case *TaskNotificationMessage:
		o.Hooks.DispatchNotificationArrived(ctx, &hooks.NotificationArrivedInput{
			Title:   string(m.Status),
			Message: m.Summary,
		})
	case *TaskStartedMessage:
		o.Hooks.DispatchSessionStarted(ctx, &hooks.SessionStartedInput{
			SessionID: m.SessionID,
		})
	}
}

func buildQueryArgs(prompt string, o *Options) []string {
	args := []string{
		"--print", prompt,
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
	}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.FallbackModel != "" {
		args = append(args, "--fallback-model", o.FallbackModel)
	}
	if o.SystemPrompt != "" {
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
			args = append(args, "--thinking", fmt.Sprintf("enabled:%d", o.Thinking.BudgetTokens))
		case "disabled":
			args = append(args, "--thinking", "disabled")
		case "adaptive":
			args = append(args, "--thinking", "adaptive")
		}
	}
	if o.MaxThinkingTokens > 0 {
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
	if o.PermissionPromptTool != "" {
		args = append(args, "--permission-prompt-tool", o.PermissionPromptTool)
	}
	if len(o.Tools) > 0 {
		args = append(args, "--tools", strings.Join(o.Tools, ","))
	}
	if len(o.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(o.AllowedTools, ","))
	}
	if len(o.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(o.DisallowedTools, ","))
	}
	if len(o.MCPServers) > 0 {
		mcpCfg := buildMCPConfig(o.MCPServers)
		if data, err := json.Marshal(mcpCfg); err == nil {
			args = append(args, "--mcp-config", string(data))
		}
	}
	if o.FileCheckpointing {
		args = append(args, "--enable-file-checkpointing")
	}
	if len(o.Betas) > 0 {
		args = append(args, "--betas", strings.Join(o.Betas, ","))
	}
	if len(o.Skills) > 0 {
		args = append(args, "--skills", strings.Join(o.Skills, ","))
	}
	if len(o.SettingSources) > 0 {
		args = append(args, "--setting-sources", strings.Join(o.SettingSources, ","))
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
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func buildMCPConfig(servers []MCPServerConfig) map[string]mcpServerEntry {
	cfg := make(map[string]mcpServerEntry, len(servers))
	for _, s := range servers {
		entry := mcpServerEntry{
			Type:    string(s.Type),
			Command: s.Command,
			Args:    s.Args,
			URL:     s.URL,
			Env:     s.Env,
		}
		cfg[s.Name] = entry
	}
	return cfg
}
