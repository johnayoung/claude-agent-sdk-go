# Feature: Claude Agent SDK for Go

## Summary

Feature-perfect idiomatic Go port of the official Claude Agent SDK (Python). Provides a complete Go library for building agents powered by Claude Code, including streaming queries, bidirectional client sessions, hooks, custom MCP tools, permission management, and session persistence.

## Background

The official Python SDK (github.com/anthropics/claude-agent-sdk-python) is the reference implementation. This port targets Go developers who need programmatic access to Claude Code's agent capabilities. The SDK communicates with a locally-installed Claude Code CLI process via a transport layer.

Public repository: github.com/johnayoung/claude-agent-sdk-go

## Scope

### In Scope

- `query()` equivalent: one-shot streaming queries returning iter.Seq2[Message, error]
- `ClaudeSDKClient` equivalent: bidirectional interactive client with full lifecycle
- Hooks system: all 10 hook events with typed handler functions
- Permission management: allow/deny callbacks, permission modes, permission updates
- Custom MCP tools: Tool interface + MCP server creation (stdio, SSE, HTTP, SDK types)
- Session management: list, get, fork, rename, tag, delete sessions
- Session store: SessionStore interface + InMemorySessionStore
- Transport layer: public Transport interface, default subprocess implementation
- All message types: User, Assistant, System, Result, TaskStarted/Progress/Notification
- All content blocks: Text, Thinking, ToolUse, ToolResult
- Error hierarchy: typed errors matching Python's exception classes
- Testing subpackage: mock transport, test helpers
- Context usage and MCP status queries
- Agent definitions and subagent support
- Sandbox configuration types
- Rate limit types and events

### Out of Scope

- Bundling/downloading the Claude Code CLI binary (require pre-installed)
- Python's `_bundled/` directory equivalent
- Build scripts (build_wheel.py equivalent)
- Python-specific type markers (py.typed)
- Transcript mirror batching (internal optimization, implement if needed later)

## Requirements

### Functional Requirements

1. **FR-1**: One-shot query function that streams messages from Claude Code
   - Acceptance: `for msg, err := range agent.Query(ctx, "prompt", opts...) { }` works and yields all message types

2. **FR-2**: Bidirectional client supporting multi-turn conversations
   - Acceptance: Client can send queries, receive responses, send follow-ups, and interrupt

3. **FR-3**: Typed hook system covering all 10 hook events
   - Acceptance: Each hook event has its own input/output types and handler signature; hooks execute at correct lifecycle points

4. **FR-4**: Permission management with custom callbacks
   - Acceptance: `WithCanUseTool(func)` receives tool name + input, returns allow/deny decision; permission updates propagate to CLI

5. **FR-5**: Custom MCP tool definition via Tool interface
   - Acceptance: Implement Tool interface, register with NewMCPServer(), tools are callable by Claude

6. **FR-6**: Session management as standalone functions
   - Acceptance: `session.List(ctx, store)`, `session.Fork(ctx, store, id)` etc. work independently of client

7. **FR-7**: Public Transport interface with default subprocess implementation
   - Acceptance: Custom Transport implementations can be injected; default uses os/exec to manage CLI process

8. **FR-8**: Testing subpackage with mock utilities
   - Acceptance: `testing.NewMockTransport(messages...)` enables unit testing without CLI

9. **FR-9**: Configuration via functional options
   - Acceptance: `agent.NewClient(ctx, agent.WithSystemPrompt("..."), agent.WithMaxTurns(5))` compiles and configures correctly

10. **FR-10**: CLI discovery via PATH with option override
    - Acceptance: Finds `claude` in PATH; `WithCLIPath("/custom")` overrides; returns CLINotFoundError if missing

### Non-Functional Requirements

- **Performance**: Streaming should add minimal overhead over raw CLI output parsing
- **Security**: No shell injection in CLI argument construction; sanitize all inputs passed to subprocess
- **Compatibility**: Go 1.23+ minimum; standard library preferred over third-party deps where practical
- **Documentation**: Godoc comments on all exported types and functions; runnable examples in _test files

## Behavior Specification

### Happy Path

1. User creates client with options: `client, err := agent.NewClient(ctx, opts...)`
2. Sends query: `client.Query(ctx, "analyze this code")`
3. Receives streaming messages via iter.Seq2 or channel on client
4. Messages include text blocks, tool use, tool results
5. Conversation completes with a ResultMessage

### Error Handling

| Error Condition            | Expected Behavior                                               |
| -------------------------- | --------------------------------------------------------------- |
| CLI not found in PATH      | Return `*CLINotFoundError` with searched paths                  |
| CLI process exits non-zero | Return `*ProcessError` with exit code and stderr                |
| Malformed JSON from CLI    | Return `*JSONDecodeError` with raw line and wrapped parse error |
| Message parse failure      | Return `*MessageParseError` with raw data                       |
| Context cancelled          | Stop reading from process, propagate context.Canceled           |
| Hook returns error         | Propagate error to caller, include hook event context           |

### Edge Cases

| Case                                       | Expected Behavior                                                   |
| ------------------------------------------ | ------------------------------------------------------------------- |
| Empty prompt string                        | Pass through to CLI (let CLI validate)                              |
| Nil options                                | Use zero-value defaults (no system prompt, default permission mode) |
| Multiple concurrent queries on same client | Return error; client is single-session                              |
| Transport connection lost mid-stream       | Return error on next message read                                   |
| Hook modifies tool input                   | Modified input forwarded to CLI via hook output                     |
| MCP server fails to connect                | Reported via McpServerStatus with error field                       |

## Technical Context

### Package Structure

```
github.com/johnayoung/claude-agent-sdk-go/
  go.mod
  agent/          -- Core: Query(), Client, Options, Message types
  hooks/          -- Hook types, matchers, event definitions
  mcp/            -- Tool interface, MCP server configs, NewMCPServer()
  session/        -- Session management functions, SessionStore interface
  transport/      -- Transport interface, subprocess implementation
  permission/     -- Permission types, callbacks, updates
  testing/        -- Mock transport, test helpers, message builders
  internal/       -- Shared internal utilities (message parsing, etc.)
  examples/       -- Runnable example programs
```

### Integration Points

- Claude Code CLI: subprocess communication via stdin/stdout JSON lines
- MCP protocol: SDK MCP servers communicate via stdio transport
- File system: session store reads/writes, working directory management

### Relevant Existing Code

- Python reference: github.com/anthropics/claude-agent-sdk-python (all of src/claude_agent_sdk/)
- Python types.py: Complete type system to port
- Python client.py: Client lifecycle and method signatures
- Python query.py: One-shot query pattern
- Python _internal/transport/: CLI subprocess communication protocol

## Decisions Log

| Decision          | Choice                                    | Rationale                                       |
| ----------------- | ----------------------------------------- | ----------------------------------------------- |
| Scope             | Full port of all public API               | User wants feature parity                       |
| API style         | Idiomatic Go                              | Go consumers expect Go conventions              |
| Go version        | 1.23+                                     | Enables iter.Seq for streaming, latest features |
| Streaming         | iter.Seq2[Message, error]                 | Native Go 1.23 iterators, composable with range |
| MCP tools         | Tool interface + struct                   | Type-safe, testable, idiomatic                  |
| Configuration     | Functional options                        | Extensible, zero-value safe                     |
| Hooks             | Typed handler funcs per event             | Compile-time type safety                        |
| CLI binary        | Require pre-installed, PATH lookup        | Go libs don't bundle binaries                   |
| Transport         | Public interface                          | Enables testing and custom implementations      |
| Testing           | Dedicated subpackage                      | Follows net/http/httptest pattern               |
| Errors            | Typed error structs                       | errors.As() inspection with context             |
| Sessions          | Standalone functions in session pkg       | Decoupled from client lifecycle                 |
| Module path       | github.com/johnayoung/claude-agent-sdk-go | Matches repo name                               |
| Package structure | Single module, multi-package              | Clean separation of concerns                    |

## Open Questions

None - all requirements resolved through discovery.

## Next Steps

Run `/task 00001-FEATURE-go-agent-sdk` to generate implementation tasks from this spec.
