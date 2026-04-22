# claude-agent-sdk-go

[![CI](https://github.com/johnayoung/claude-agent-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/johnayoung/claude-agent-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/johnayoung/claude-agent-sdk-go.svg)](https://pkg.go.dev/github.com/johnayoung/claude-agent-sdk-go)

Go SDK for building AI agents powered by Claude Code. Wraps the Claude CLI as a subprocess and gives you a streaming iterator interface for single-turn queries, multi-turn conversations, custom tools, and sub-agents.

## Prerequisites

1. **Go 1.23+**
2. **Claude CLI** installed and authenticated -- see [Claude Code docs](https://docs.anthropic.com/en/docs/claude-code)

## Install

```sh
go get github.com/johnayoung/claude-agent-sdk-go@latest
```

## Quick start

### Single query

The simplest usage -- send a prompt, stream back the response:

```go
package main

import (
	"context"
	"fmt"
	"os"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func main() {
	for msg, err := range claude.Query(context.Background(), "Explain goroutines in two sentences.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if m, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Print(text.Text)
				}
			}
		}
	}
}
```

### Multi-turn conversation

`Client` keeps session state across calls so Claude remembers prior context:

```go
client, _ := claude.NewClient(context.Background())
defer client.Close()

for msg, err := range client.Query(ctx, "My name is Alex.") {
	// handle response...
}

// Claude remembers the name from the previous turn
for msg, err := range client.Query(ctx, "What's my name?") {
	// handle response...
}
```

### Custom tools (in-process MCP)

Define tools that Claude can call -- no external MCP server needed:

```go
type upperTool struct{}

func (upperTool) Name() string        { return "to_upper" }
func (upperTool) Description() string { return "Converts text to uppercase" }
func (upperTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`)
}
func (upperTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	text, _ := input["text"].(string)
	return json.Marshal(strings.ToUpper(text))
}

server := mcp.NewMCPServer("my-tools", upperTool{})

for msg, err := range claude.Query(ctx, "Uppercase 'hello world' using the to_upper tool",
	claude.WithSDKMCPServer(server),
) {
	// handle response...
}
```

### Sub-agents

Define specialized agents with their own tools, prompts, and models:

```go
agents := map[string]claude.AgentDefinition{
	"code-reviewer": {
		Description: "Reviews code for best practices",
		Prompt:      "You are a code reviewer. Analyze for bugs and security issues.",
		Tools:       []string{"Read", "Grep"},
		Model:       "sonnet",
	},
}

for msg, err := range claude.Query(ctx, "Use code-reviewer to review main.go",
	claude.WithAgents(agents),
) {
	// handle response...
}
```

## Features & Python SDK parity

This SDK targets feature parity with [`claude-agent-sdk-python`](https://github.com/anthropics/claude-agent-sdk-python).

**Legend:** Y = implemented, -- = not yet implemented, N/A = not applicable to Go

### Core query & streaming

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Single-turn query | `Query()` sends a prompt and returns a streaming iterator | Y | Y | [basic-query](examples/basic-query/) |
| Multi-turn client | `NewClient()` maintains session state across multiple queries | Y | Y | [multi-turn](examples/multi-turn/) |
| Streaming iterator | `iter.Seq2[Message, error]` yields messages as they arrive | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Partial messages | Stream incomplete content blocks for real-time UI | Y | Y | TODO |
| Interrupt / cancel | `Client.Interrupt()` stops in-flight query, keeps client usable | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Context timeout | Standard `context.WithTimeout` for deadline control | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Async runtimes (trio, IPython) | Alternative async event loops | N/A | Y | -- |

### Configuration

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Model selection | `WithModel("sonnet")` | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Fallback model | `WithFallbackModel()` when primary is unavailable | Y | Y | TODO |
| System prompt (string) | `WithSystemPrompt()` | Y | Y | [streaming-mode](examples/streaming-mode/) |
| System prompt (file) | `WithSystemPromptFile()` loads from disk | Y | Y | TODO |
| Append system prompt | `WithAppendSystemPrompt()` extends default prompt | Y | Y | TODO |
| Max turns | `WithMaxTurns()` limits agentic tool-use loops | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Max budget USD | `WithMaxBudgetUSD()` spending cap per query | Y | Y | TODO |
| Effort levels | `WithEffort("low" / "medium" / "high" / "max")` | Y | Y | TODO |
| Extended thinking | `WithThinking(ThinkingAdaptive())` or `ThinkingEnabled(n)` | Y | Y | TODO |
| Structured output | `WithOutputFormat()` JSON schema validation | Y | Y | TODO |
| Setting sources | `WithSettingSources("user", "project", "local")` | Y | Y | [agents](examples/agents/) |
| Working directory | `WithWorkingDir()` for CLI subprocess | Y | Y | TODO |
| Environment variables | `WithEnv()` passes env to CLI | Y | Y | TODO |
| Extra CLI args | `WithExtraArgs()` for unmapped flags | Y | Y | TODO |
| Stderr callback | `WithStderr()` captures CLI debug output | Y | Y | TODO |

### Tools & MCP

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| In-process MCP tools | Implement `mcp.Tool` interface, serve via `mcp.NewMCPServer()` | Y | Y | [custom-tools](examples/custom-tools/) |
| External MCP servers | `WithMCPServers()` for stdio, SSE, and HTTP servers | Y | Y | TODO |
| Tool allowlist | `WithAllowedTools()` pre-approves specific tools | Y | Y | TODO |
| Tool blocklist | `WithDisallowedTools()` blocks specific tools | Y | Y | TODO |
| MCP reconnect | `Client.ReconnectMCPServer()` reconnects a named server | Y | Y | [streaming-mode](examples/streaming-mode/) |
| MCP toggle | `Client.ToggleMCPServer()` enables/disables a server | Y | Y | [streaming-mode](examples/streaming-mode/) |
| MCP status | `Client.GetMCPStatus()` queries all server connection states | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Server tool blocks | Parse `ServerToolUseBlock` and `ServerToolResultBlock` from MCP invocations | Y | Y | [streaming-mode](examples/streaming-mode/) |
| `@tool` decorator | Python decorator for defining tools from functions | N/A | Y | -- |

### Permissions

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Permission modes | `WithPermissionMode()` -- Default, AcceptEdits, Plan, BypassPermissions, DontAsk, Auto | Y | Y | [tool-permission-callback](examples/tool-permission-callback/) |
| Permission callback | `WithCanUseTool(fn)` for custom approval logic | Y | Y | [tool-permission-callback](examples/tool-permission-callback/) |
| Allow / Deny decisions | `ResultAllow`, `ResultDeny`, `AllowWithUpdates` | Y | Y | [tool-permission-callback](examples/tool-permission-callback/) |
| Input modification | Modify tool inputs before execution via permission callback | Y | Y | [tool-permission-callback](examples/tool-permission-callback/) |

### Hooks & lifecycle

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| PreToolUse | Run logic before tool execution; can block or modify input | Y | Y | [hooks](examples/hooks/) |
| PostToolUse | Run logic after successful tool execution | Y | Y | [hooks](examples/hooks/) |
| PostToolUseFailure | Handle tool execution errors | Y | Y | TODO |
| ModelResponse | Intercept model text responses | Y | Y | TODO |
| UserPromptSubmit | Intercept/modify user prompts before sending | Y | Y | [hooks](examples/hooks/) |
| Stop | Custom stop logic with reason | Y | Y | TODO |
| SessionStarted / Stopped | Session lifecycle events | Y | Y | TODO |
| SubagentStarted / Stopped | Sub-agent lifecycle events | Y | Y | TODO |
| NotificationArrived | Task notification events | Y | Y | TODO |
| PermissionRequest | Custom permission decision handling | Y | Y | TODO |
| PreCompact | Intercept before message compaction | Y | Y | TODO |
| Error | Global error handler | Y | Y | TODO |
| Pattern matching | Glob patterns to target specific tools in hooks | Y | Y | [hooks](examples/hooks/) |

### Agents

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Agent definitions | `AgentDefinition` with description, prompt, tools, model | Y | Y | [agents](examples/agents/) |
| Multiple agents | Register multiple agents with distinct roles | Y | Y | [agents](examples/agents/) |
| Per-agent config | Each agent gets its own tools, model, effort, and permission mode | Y | Y | [agents](examples/agents/) |
| Filesystem-based agents | Load agent definitions from `.claude/agents/*.md` files | -- | Y | -- |

### Session management

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Resume by ID | `WithSessionID()` resumes a specific session | Y | Y | TODO |
| Continue last session | `WithContinueConversation()` resumes most recent | Y | Y | TODO |
| Fork session | `WithForkSession()` creates a branching copy | Y | Y | TODO |
| List sessions | `ListSessions()` enumerates sessions | Y | Y | TODO |
| Get session info | `GetSessionInfo()` fetches metadata | Y | Y | TODO |
| Rename session | `RenameSession()` updates title | Y | Y | TODO |
| Tag session | `TagSession()` assigns tags | Y | Y | TODO |
| Delete session | `DeleteSession()` permanently removes session | Y | Y | TODO |
| Subagent messages | `ListSubagents()` / `GetSubagentMessages()` | Y | Y | TODO |

### Session storage

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| SessionStore interface | Pluggable backend for session persistence | Y | Y | TODO |
| In-memory store | `NewInMemorySessionStore()` for development | Y | Y | TODO |
| Transcript mirroring | Automatic persistence when SessionStore is set | Y | Y | TODO |
| S3 store adapter | Reference adapter for AWS S3 | Y | Y | [session_stores/s3](examples/session_stores/s3/) |
| Redis store adapter | Reference adapter for Redis | Y | Y | [session_stores/redis](examples/session_stores/redis/) |
| PostgreSQL store adapter | Reference adapter for PostgreSQL | Y | Y | [session_stores/postgres](examples/session_stores/postgres/) |
| Conformance test suite | Validates store implementations against behavioral contracts | Y | Y | [agenttest/sessionstoretest](agenttest/sessionstoretest/) |

### Runtime control

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Context usage | `Client.GetContextUsage()` returns token breakdown | Y | Y | [streaming-mode](examples/streaming-mode/) |
| Server info | `Client.GetServerInfo()` returns CLI server metadata | Y | Y | TODO |
| Stop task | `Client.StopTask()` stops a running background task | Y | Y | TODO |

### File checkpointing

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Enable checkpointing | `WithFileCheckpointing()` snapshots file state | Y | Y | TODO |
| Rewind files | `Client.RewindFiles()` restores to checkpoint | Y | Y | TODO |

### Plugins

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Local plugins | `WithPlugins()` loads local SDK plugins | Y | Y | TODO |

### Testing utilities (`agenttest` package)

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| Mock transport | `NewMockTransport()` for unit testing without CLI | Y | Y | [agenttest](agenttest/) |
| Message builders | `NewTextMessage()`, `NewToolUseMessage()`, `NewResultMessage()` | Y | Y | [agenttest](agenttest/) |
| Assertions | `AssertTextContent()`, `AssertToolUse()`, `AssertResult()` | Y | Y | [agenttest](agenttest/) |

### Observability

| Feature | Description | Go | Python | Example |
| --- | --- | --- | --- | --- |
| OpenTelemetry tracing | Distributed tracing with trace context propagation | -- | Y | -- |
| Bundled CLI binary | SDK ships with Claude CLI for target platforms | N/A | Y | -- |

## Common options

Options are passed as functional arguments to `Query` or `NewClient`:

```go
claude.Query(ctx, prompt,
	claude.WithModel("opus"),
	claude.WithSystemPrompt("You are a helpful assistant."),
	claude.WithMaxTurns(3),
	claude.WithMaxBudgetUSD(0.50),
	claude.WithPermissionMode(claude.PermissionModeBypassPermissions),
)
```

| Option | Description |
| --- | --- |
| `WithModel(m)` | Model selection: `"sonnet"`, `"opus"`, `"haiku"` |
| `WithSystemPrompt(s)` | Custom system prompt |
| `WithMaxTurns(n)` | Limit agentic tool-use turns |
| `WithMaxBudgetUSD(n)` | Spending cap per query |
| `WithPermissionMode(m)` | Tool permission handling (`Default`, `AcceptEdits`, `BypassPermissions`, etc.) |
| `WithTools(t...)` | Explicit tool list |
| `WithAllowedTools(t...)` | Tool allowlist |
| `WithDisallowedTools(t...)` | Tool blocklist |
| `WithMCPServers(s...)` | External MCP server configs |
| `WithSDKMCPServer(s)` | In-process MCP tool server |
| `WithAgents(a)` | Sub-agent definitions |
| `WithThinking(cfg)` | Extended thinking (`ThinkingAdaptive()`, `ThinkingEnabled(n)`, `ThinkingDisabled()`) |
| `WithWorkingDir(d)` | CLI working directory |
| `WithEnv(e)` | Extra environment variables |

Full list in [options.go](options.go) and on [pkg.go.dev](https://pkg.go.dev/github.com/johnayoung/claude-agent-sdk-go).

## Message types

The streaming iterator yields these `Message` types:

| Type | When |
| --- | --- |
| `*AssistantMessage` | Claude's response text, tool calls, and server tool invocations |
| `*UserMessage` | Echoed user input and tool results |
| `*ResultMessage` | Final message with session ID, cost, and token usage |

Content blocks within messages:

| Block type | Description |
| --- | --- |
| `*TextBlock` | Text content |
| `*ThinkingBlock` | Extended thinking content |
| `*ToolUseBlock` | Tool invocation request |
| `*ToolResultBlock` | Tool execution result |
| `*ServerToolUseBlock` | MCP server tool invocation |
| `*ServerToolResultBlock` | MCP server tool result |

Extract text from an `AssistantMessage`:

```go
if m, ok := msg.(*claude.AssistantMessage); ok {
	for _, block := range m.Content {
		if text, ok := block.(*claude.TextBlock); ok {
			fmt.Print(text.Text)
		}
	}
}
```

## Packages

| Import | Purpose |
| --- | --- |
| `claude "github.com/johnayoung/claude-agent-sdk-go"` | Core: `Query`, `NewClient`, message types |
| `.../hooks` | Lifecycle event callbacks |
| `.../permission` | Tool approval callbacks |
| `.../mcp` | MCP tool interface and server configs |
| `.../agenttest` | Mock transport and test helpers |

## Examples

| Example | What it shows |
| --- | --- |
| [basic-query](examples/basic-query/) | Single-turn streaming query |
| [multi-turn](examples/multi-turn/) | Conversation with session resumption |
| [custom-tools](examples/custom-tools/) | In-process MCP tools |
| [agents](examples/agents/) | Sub-agent definitions |
| [hooks](examples/hooks/) | Lifecycle hooks: PreToolUse, PostToolUse, UserPromptSubmit, pattern matching |
| [streaming-mode](examples/streaming-mode/) | Streaming patterns, MCP status, context usage, server tool blocks |
| [tool-permission-callback](examples/tool-permission-callback/) | Permission callbacks: allow, deny, and input modification |

Run any example:

```sh
go run ./examples/basic-query
```

## License

MIT
