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

## Phase 3: Message Type Field Parity [DONE]

**Goal:** Every field on every message type matches the wire format exactly.

**Changes made:**

All message types audited against Python SDK `types.py` and aligned:
- `UserMessage` — `UUID`, `ParentToolUseID`, `ToolUseResult` fields
- `AssistantMessage` — `SessionID`, `UUID`, `Model`, `MessageID`, `StopReason`, `Usage`, `Error`
- `SystemMessage` subtypes — `TaskStartedMessage`, `TaskProgressMessage`, `TaskNotificationMessage`, `MirrorErrorMessage` with typed fields
- `ResultMessage` — `StructuredOutput`, `ModelUsage`, `PermissionDenials`, `Errors`, `UUID`
- `StreamEvent`, `RateLimitEvent` — full struct deserialization
- Content blocks — `ThinkingBlock.Signature`, `ToolResultBlock.IsError` as `*bool`
- JSON tags verified for camelCase/snake_case wire conventions
- Optional fields use pointer types where Python uses `Optional`

**Key files:**
- `message.go`, `types.go`, `content.go`, `parse.go`

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

## Phase 5: Transcript Mirroring & Session Persistence [DONE]

**Goal:** Intercept `transcript_mirror` messages from the CLI and persist entries to the configured `SessionStore`.

**Changes made:**

Transcript mirror handling (`transcript_mirror.go`):
- `transcriptMirrorMessage` — internal message type, never yielded to callers
- `parseTranscriptMirror` — deserializes wire format (`filePath` + `entries[]`)
- `filePathToSessionKey` — derives `SessionKey` from file path using projects dir structure
- `filePathToSessionKeyFromDir` — path-based key extraction for main transcripts (`<projectKey>/<sessionID>.jsonl`) and subagents (`<projectKey>/<sessionID>/subagents/<agent>.jsonl`)
- Hash fallback — when path is outside projects dir, uses `basename-sha256[:16]`
- `handleTranscriptMirror` — synchronous persistence to `SessionStore.Append`; returns `*MirrorErrorMessage` on failure

Integration (`query.go`, `client.go`):
- Both query loops intercept `*transcriptMirrorMessage` before yielding, call `handleTranscriptMirror` when `SessionStore` is set
- `--session-mirror` CLI flag emitted when `SessionStore != nil`

Options (`options.go`):
- `ProjectsDir` field + `WithProjectsDir()` — configures path resolution for transcript mirror

Parse (`parse.go`):
- `"transcript_mirror"` case added to `parseLine` switch

Tests (`transcript_mirror_test.go`):
- Path-based key derivation (main, subagent, deeply nested)
- Hash fallback (no projects dir, outside projects dir)
- Parsing round-trip
- Success, empty entries, store error persistence cases

**Key files:**
- `transcript_mirror.go`, `transcript_mirror_test.go`
- `query.go`, `client.go` (intercept points)
- `options.go` (`ProjectsDir`)

---

## Phase 6: File Checkpointing and Rewind [DONE]

**Goal:** Support file checkpoint/restore via CLI control protocol.

**Finding:** The Python SDK does NOT manage checkpoints SDK-side. All checkpointing is CLI-internal, triggered by `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true`. The SDK's role is (a) setting the env var and (b) sending/handling `rewind_files` control requests.

**Changes made:**

`checkpoint.go`:
- `Client.RewindFiles(ctx, userMessageID) error` — sends SDK→CLI control request to restore files to a checkpoint
- `ErrCheckpointingDisabled`, `ErrNoSession` — sentinel errors
- Internal helpers: `buildRewindArgs`, `sendRewindFilesRequest`, `awaitControlResponse`, `generateRequestID`
- Starts a subprocess connected to the session, sends the control request, waits for response

`control_handler.go`:
- Broke `rewind_files` out of the no-op batch into `handleRewindFiles` which parses the `ControlRewindFilesRequest` and ACKs

`checkpoint_test.go`:
- `TestRewindFiles_CheckpointingDisabled` — verifies guard
- `TestRewindFiles_NoSession` — verifies guard
- `TestRewindFiles_ClientClosed` — verifies guard
- `TestRewindFiles_Success` — round-trip via mock transport
- `TestRewindFiles_ErrorResponse` — error propagation
- `TestControlRequest_RewindFiles_FromCLI` — CLI→SDK handling

**Key files:**
- `checkpoint.go`, `checkpoint_test.go`
- `control_handler.go` (rewind_files handling)

---

## Phase 7: End-to-End Integration Tests [TODO]

**Goal:** Verify complete flows work against the real CLI binary, matching Python SDK e2e coverage.

**Reference:** https://github.com/anthropics/claude-agent-sdk-python/tree/main/e2e-tests

**Placement:** Idiomatic Go — tests live in `e2e_test.go` (or split per-domain) in the root package, gated by `//go:build e2e`. Run via `go test -tags=e2e -count=1 ./...`. Requires `ANTHROPIC_API_KEY` env var.

**Test domains (from Python SDK e2e-tests):**

1. **Agents & Settings** (`test_agents_and_settings.py`)
   - Agent definitions appear in init (streaming + query)
   - Large agents (~260KB) register successfully
   - Filesystem agent loading from `.claude/agents/*.md`
   - `SettingSources` filtering (user-only, project-included)

2. **Dynamic Control** (`test_dynamic_control.py`)
   - `Client.SetPermissionMode()` mid-session
   - `Client.SetModel()` mid-session
   - `Client.Interrupt()` during response

3. **Hook Events** (`test_hook_events.py`)
   - PreToolUse receives `tool_use_id`, returns `additionalContext` + allow
   - PostToolUse receives `tool_use_id`
   - Notification hook shape
   - Multiple hooks registered simultaneously

4. **Hooks Control** (`test_hooks.py`)
   - PreToolUse denies with reason
   - PostToolUse stops execution with `stopReason`
   - PostToolUse returns `additionalContext`

5. **Partial Messages** (`test_include_partial_messages.py`)
   - `WithIncludePartialMessages()` yields stream events (deltas, starts, stops)
   - Thinking deltas arrive incrementally
   - Disabled by default (no stream events)

6. **SDK MCP Tools** (`test_sdk_mcp_tools.py`)
   - SDK-defined MCP tool executes handler
   - `DisallowedTools` blocks, `AllowedTools` permits
   - Multiple tools callable in one session
   - Without explicit allow, tools are blocked

7. **Stderr Callback** (`test_stderr_callback.py`)
   - `WithStderr` captures debug output with `ExtraArgs{"--debug-to-stderr": nil}`
   - Without debug mode, no stderr output

8. **Structured Output** (`test_structured_output.py`)
   - `WithOutputFormat` produces `ResultMessage.StructuredOutput`
   - Nested objects/arrays, enum constraints
   - Works alongside tool use

9. **Tool Permissions** (`test_tool_permissions.py`)
   - `WithCanUseTool` callback invoked for non-read-only tools
   - Allow/deny decisions respected

**Key files:**
- `e2e_test.go` (build tag `//go:build e2e`)
- Helper `e2e_helpers_test.go` for shared setup (API key, skip logic, client factory)
