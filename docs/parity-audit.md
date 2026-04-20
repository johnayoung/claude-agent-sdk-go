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

Second pass additions:
- `ThinkingAdaptive()`, `ThinkingEnabled()`, `ThinkingDisabled()` — constructors matching Python's discriminated union variants
- `permission.ResultAllow`, `permission.ResultDeny` — explicit result types; top-level aliases `PermissionResultAllow`, `PermissionResultDeny`
- `ClaudeSDKError` — base error type for all SDK errors
- `ToolAnnotations`, `SdkMcpTool` — generic tool annotation types
- `SessionListSubkeysKey` — helper key type for subkey enumeration
- `hooks.HookContext` — ambient state for hook callbacks
- Hook wire-format types (`hooks`): `BaseHookInput`, `PreToolUseHookWireInput`, `PostToolUseHookWireInput`, `PostToolUseFailureHookWireInput`, `UserPromptSubmitHookWireInput`, `StopHookWireInput`, `SubagentStopHookWireInput`, `PreCompactHookWireInput`, `NotificationHookWireInput`, `SubagentStartHookWireInput`, `PermissionRequestHookWireInput`
- Hook-specific output types (`hooks`): `PreToolUseHookSpecificOutput`, `PostToolUseHookSpecificOutput`, `PostToolUseFailureHookSpecificOutput`, `NotificationHookSpecificOutput`, `SubagentStartHookSpecificOutput`, `PermissionRequestHookSpecificOutput`
- `hooks.HookJSONOutput`, `hooks.HookMatcher` — hook configuration types
- Store-backed session functions: `ListSessionsFromStore`, `GetSessionInfoFromStore`, `GetSessionMessagesFromStore`, `ListSubagentsFromStore`, `GetSubagentMessagesFromStore`, `DeleteSessionViaStore`, `RenameSessionViaStore`, `TagSessionViaStore`, `ForkSessionViaStore`

**Key files to compare:**
- Python: `__init__.py`, `types.py`
- Go: `message.go`, `types.go`, `content.go`, `options.go`, `query.go`, `client.go`

---

## Phase 2: CLI Argument Parity [DONE]

**Goal:** Every CLI flag the Python SDK can pass, we can pass too.

**Changes made:**

CLI argument fixes (`query.go`):
- `--system-prompt` now always sent (empty string when unset, matching Python's explicit None behavior)
- `--system-prompt-file` support via `SystemPromptSource{Type: "file"}`
- `--append-system-prompt` support via `SystemPromptSource{Type: "preset"}`
- `--tools` supports presets (e.g. "default") via `ToolsPreset` field
- `--tools ""` sent when `Tools` is explicitly `[]string{}` (empty slice vs nil)
- `--thinking enabled:N` replaced with `--max-thinking-tokens N` only (matching Python)
- `--setting-sources=X` now uses `=` form (matching Python)
- Removed `--enable-file-checkpointing` CLI flag (handled via env var)

Environment variables (`buildEnv` in `query.go`):
- `CLAUDE_CODE_ENTRYPOINT=sdk-go` — identifies SDK invocation
- `CLAUDE_AGENT_SDK_VERSION` — reports SDK version
- `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true` — when enabled
- User `Env` map merged in

Initialize request protocol (`sendInitializeRequest` in `query.go`):
- Sends `{"type": "initialize", ...}` via stdin after process starts
- Fields: `agents`, `excludeDynamicSections`, `skills`

Transport env support (`internal/transport/subprocess.go`):
- Added `WithEnv` option to set subprocess environment variables

New options (`options.go`):
- `SystemPromptSource` — file/preset system prompt variants
- `ToolsPreset` — named tool presets
- `ExcludeDynamicSections` — control dynamic section inclusion
- `WithSystemPromptFile()`, `WithAppendSystemPrompt()`, `WithToolsPreset()`, `WithExcludeDynamicSections()`

**Key files to compare:**
- Python: `transport/subprocess_cli.py`, `query.py`
- Go: `query.go` (buildQueryArgs, buildEnv, sendInitializeRequest), `options.go`

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

## Phase 4: Behavioral Parity [DONE]

**Goal:** Logic and control flow matches the reference SDK.

**Changes made:**

Control protocol (`control.go`, `control_handler.go`):
- `ControlRequestMessage` parsed from wire, never yielded to callers
- `handleControlRequest` dispatches by subtype: `can_use_tool`, `hook_callback`, `interrupt`, `set_permission_mode`, `mcp_message`, `rewind_files`, `mcp_reconnect`, `mcp_toggle`, `stop_task`
- Permission handler maps `Decision` to allow/deny with `updated_input`, `updated_permissions`, `message`, `interrupt`
- Nil `CanUseTool` defaults to allow
- Interrupt cancels context via `cancelFn()`
- All 13 hook event types dispatched with `session_id` propagation
- Error paths send `ControlErrorResponse`
- No goroutines — protocol is synchronous
- Removed redundant `dispatchHooks` local dispatch (hooks now flow exclusively through control protocol)

**Key files:**
- `control.go`, `control_handler.go`, `control_test.go`
- `query.go`, `client.go` (integration points)

---

## Phase 5: Session Store Persistence [TODO]

**Goal:** Wire up `SessionStore` so messages are persisted as they stream in during a query.

**Context:** `SessionStore` interface, `InMemorySessionStore`, and `--session-mirror` CLI flag all exist but the query loop doesn't actually persist messages.

**Steps:**
1. Fetch Python SDK's `_internal/client.py` — understand `_persist_message` behavior (what's stored, key structure, subpath conventions)
2. Add persistence call in both `query.go` and `client.go` query loops when `Options.SessionStore` is non-nil
3. Key by `(project_key, session_id, subpath)` matching Python conventions
4. Keep it synchronous — no goroutines

**Key files:**
- Python: `_internal/client.py` (`_persist_message`)
- Go: `query.go`, `client.go`, `session.go`

---

## Phase 6: Transcript Mirroring [TODO]

**Goal:** Real-time batched transcript streaming to external consumers.

**Steps:**
1. Fetch Python SDK's `_internal/transcript_mirror_batcher.py` — understand batching strategy, flush intervals, and wire format
2. Implement a `TranscriptMirror` interface or callback that receives batched messages
3. Integrate into the query loop (flush on result, periodic flush if needed)
4. Respect `Options.SessionStore` presence as the trigger

**Key files:**
- Python: `_internal/transcript_mirror_batcher.py`
- Go: new `mirror.go` or inline in `query.go`

---

## Phase 7: File Checkpointing and Rewind [TODO]

**Goal:** Support file checkpoint/restore when the CLI requests a rewind.

**Steps:**
1. Fetch Python SDK's file checkpointing logic — understand when checkpoints are created and how `rewind_files` control requests trigger restoration
2. Implement checkpoint creation (snapshot file state before tool execution)
3. Handle `rewind_files` control request: restore files to checkpoint identified by `user_message_id`
4. Gate behind `Options.FileCheckpointing` (already wires env var `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING`)

**Key files:**
- Python: `_internal/client.py` (checkpoint logic), control protocol handling
- Go: `control_handler.go` (currently no-ops `rewind_files`), new `checkpoint.go`

---

## Phase 8: End-to-End Integration Tests [TODO]

**Goal:** Verify complete flows work against the real CLI binary.

**Steps:**
1. Multi-turn conversation via `Client` with session resume
2. Permission deny/allow flow with tool re-invocation
3. Hook callback chain (pre_tool_use modifies input, post_tool_use observes output)
4. Interrupt mid-stream and verify clean shutdown
5. Session store round-trip: persist during query, read back via `GetSessionMessages`
6. Structured output with JSON schema validation

**Key files:**
- New `integration_test.go` (build-tagged `//go:build integration`)
