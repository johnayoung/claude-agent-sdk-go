# SDK Parity Audit Plan

Reference: https://github.com/anthropics/claude-agent-sdk-python

Work through each phase sequentially. For each, fetch the relevant Python SDK source, compare against our Go implementation, and fix gaps before moving to the next.

---

## Phase 1: Public API Surface Parity [DONE]

**Goal:** Every exported class/function/type in the Python SDK has a Go equivalent.

**Changes made:**

Types added (`types.go`):
- `TaskNotificationStatus`, `AssistantMessageError` — typed string constants
- `SdkBeta`, `SettingSource` — typed string aliases with constants
- `TaskUsage`, `TaskBudget` — structured types (replace `map[string]any`)
- `SDKSessionInfo`, `SessionMessage`, `ForkSessionResult` — session management
- `SdkPluginConfig` — plugin config
- `McpToolAnnotations`, `McpToolInfo`, `McpServerInfo`, `McpServerConnectionStatus`, `McpServerStatus`, `McpStatusResponse` — MCP status
- `ContextUsageCategory`, `ContextUsageResponse` — context usage
- `SandboxNetworkConfig`, `SandboxIgnoreViolations` — aligned `SandboxConfig` with Python SDK structure
- `AgentDefinition` — aligned JSON tags with Python SDK (`disallowedTools`, `mcpServers` as `[]any`, pointer types for optional ints/bools)

Types added (`permission/permission.go`):
- `Behavior`, `UpdateDestination`, `RuleValue`, `Update`, `ToolContext` — full permission model
- `AllowWithUpdates()`, `DenyWithInterrupt()` — richer decision constructors
- `Decision.Interrupt()`, `UpdatedInput()`, `UpdatedPermissions()` — new accessors
- `CanUseToolFunc` signature expanded to accept `ToolContext`

Types added (`errors.go`):
- `CLIConnectionError` — connection failure error

Types added (`session.go`):
- `SessionStoreEntry`, `SessionStoreListEntry` — store entry types
- `InMemorySessionStore` — in-process `SessionStore` implementation
- `SessionStore` interface updated to use typed entries

Functions added (`session.go`):
- `ListSessions`, `GetSessionInfo`, `GetSessionMessages` — session queries
- `ListSubagents`, `GetSubagentMessages` — sub-agent queries
- `RenameSession`, `TagSession`, `DeleteSession`, `ForkSession` — session mutations
- `ProjectKeyForDirectory` — project key lookup

Options added (`options.go`):
- `PermissionUpdate`, `ToolPermissionContext` — top-level aliases
- Fields: `User`, `Env`, `ExtraArgs`, `OutputFormat`, `TaskBudget`, `AddDirs`, `Settings`, `MaxThinkingTokens`, `PermissionPromptTool`, `ForkSession`, `MaxBufferSize`, `Stderr`, `Plugins`, `LoadTimeoutMS`, `SessionStore`
- `With*` constructors for all new fields

Message updates (`message.go`):
- `AssistantMessage.Error` → `AssistantMessageError` type
- `TaskProgressMessage.Usage` → `TaskUsage` type
- `TaskNotificationMessage.Status` → `TaskNotificationStatus` type
- `TaskNotificationMessage.Usage` → `*TaskUsage` (pointer, optional)

**Key files to compare:**
- Python: `__init__.py`, `types.py`
- Go: `message.go`, `types.go`, `content.go`, `options.go`, `query.go`, `client.go`

---

## Phase 2: CLI Argument Parity

**Goal:** Every CLI flag the Python SDK can pass, we can pass too.

**Steps:**
1. Fetch `src/claude_agent_sdk/_internal/transport/subprocess_cli.py`
2. Find where CLI args are constructed (look for `--print`, `--output-format`, etc.)
3. Also check `src/claude_agent_sdk/_internal/query.py` for query-specific args
4. Compare against our `buildQueryArgs()` in `query.go`
5. Flag missing or incorrectly-named flags

**Key files to compare:**
- Python: `transport/subprocess_cli.py`, `query.py`
- Go: `query.go` (buildQueryArgs), `options.go`

---

## Phase 3: Message Type Field Parity

**Goal:** Every field on every message type matches the wire format exactly.

**Steps:**
1. Fetch `src/claude_agent_sdk/types.py`
2. For each dataclass, compare fields against our Go struct:
   - `UserMessage`
   - `AssistantMessage`
   - `SystemMessage` and subtypes (`TaskStartedMessage`, `TaskProgressMessage`, `TaskNotificationMessage`, `MirrorErrorMessage`)
   - `ResultMessage`
   - `StreamEvent`
   - `RateLimitEvent` / `RateLimitInfo`
   - Content blocks (`TextBlock`, `ThinkingBlock`, `ToolUseBlock`, `ToolResultBlock`)
3. Verify JSON tags match camelCase/snake_case wire conventions
4. Verify optional vs required semantics (pointer types in Go for nullable fields)

**Key files to compare:**
- Python: `types.py`
- Go: `message.go`, `types.go`, `content.go`

---

## Phase 4: Behavioral Parity

**Goal:** Logic and control flow matches the reference SDK.

**Steps:**
1. **Session resume/continuation:** Compare `_internal/sessions.py` and `_internal/session_resume.py` against our `client.go`
2. **Error handling:** Compare `_errors.py` against our `errors.go` — what errors exist, when are they raised vs. swallowed
3. **Hook/event dispatch:** Compare what messages trigger which callbacks in the Python SDK's client vs. our `dispatchHooks()` in `query.go`
4. **Permission mode handling:** Compare how the Python SDK handles permission callbacks vs. our `permission/` package
5. **Unknown message handling:** Python returns None and continues; verify we return nil and skip

**Key files to compare:**
- Python: `_internal/client.py`, `_internal/sessions.py`, `_internal/session_resume.py`, `_errors.py`
- Go: `client.go`, `query.go`, `errors.go`, `permission/`

---

## Phase 5: Missing Features

**Goal:** Identify entire capabilities present in the Python SDK that we haven't implemented.

**Steps:**
1. **Session store interface:** `_internal/session_store.py` — custom persistence backends
2. **Structured output:** `ResultMessage.structured_output` — JSON schema-validated outputs
3. **Transcript mirroring:** `_internal/transcript_mirror_batcher.py` — real-time transcript streaming
4. **Control protocol:** bidirectional messages between SDK and CLI (control_request/control_response)
5. **File checkpointing / rewind:** `rewind_files()` support via UserMessage.uuid
6. **Session mutations:** `_internal/session_mutations.py` — modifying session state

For each, determine:
- Is it critical for basic SDK functionality?
- Is it used by the examples/common use cases?
- Should we implement it now or defer?

**Key files:**
- Python: `_internal/session_store.py`, `_internal/transcript_mirror_batcher.py`, `_internal/session_mutations.py`, `_internal/client.py` (control protocol)
