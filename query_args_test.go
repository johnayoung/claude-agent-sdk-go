package claude

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
)

func TestBuildQueryArgs_InputFormat(t *testing.T) {
	args := buildQueryArgs("hello", &Options{})
	assertFlag(t, args, "--input-format", "stream-json")
}

func TestBuildQueryArgs_TaskBudget(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		TaskBudget: &TaskBudget{Total: 5000},
	})
	assertFlag(t, args, "--task-budget", "5000")
}

func TestBuildQueryArgs_PermissionPromptTool(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		PermissionPromptTool: "my_tool",
	})
	assertFlag(t, args, "--permission-prompt-tool", "my_tool")
}

func TestBuildQueryArgs_SessionID_NewSession(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SessionID: "sess-123",
	})
	assertFlag(t, args, "--session-id", "sess-123")
	assertNoFlag(t, args, "--resume")
}

func TestBuildQueryArgs_SessionID_Resume(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SessionID:            "sess-123",
		ContinueConversation: true,
	})
	assertFlag(t, args, "--resume", "sess-123")
	assertNoFlag(t, args, "--session-id")
}

func TestBuildQueryArgs_SessionID_Fork(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SessionID:   "sess-123",
		ForkSession: true,
	})
	assertFlag(t, args, "--resume", "sess-123")
	assertHasFlag(t, args, "--fork-session")
}

func TestBuildQueryArgs_AddDirs(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		AddDirs: []string{"/tmp/a", "/tmp/b"},
	})
	indices := allFlagIndices(args, "--add-dir")
	if len(indices) != 2 {
		t.Fatalf("expected 2 --add-dir flags, got %d", len(indices))
	}
	if args[indices[0]+1] != "/tmp/a" || args[indices[1]+1] != "/tmp/b" {
		t.Fatalf("unexpected --add-dir values: %v", args)
	}
}

func TestBuildQueryArgs_IncludePartialMessages(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		IncludePartialMsgs: true,
	})
	assertHasFlag(t, args, "--include-partial-messages")
}

func TestBuildQueryArgs_ForkSession(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		ForkSession: true,
	})
	assertHasFlag(t, args, "--fork-session")
}

func TestBuildQueryArgs_SessionMirror(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SessionStore: &InMemorySessionStore{},
	})
	assertHasFlag(t, args, "--session-mirror")
}

func TestBuildQueryArgs_PluginDir(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		Plugins: []SdkPluginConfig{
			{Type: "local", Path: "/path/to/plugin1"},
			{Type: "local", Path: "/path/to/plugin2"},
		},
	})
	indices := allFlagIndices(args, "--plugin-dir")
	if len(indices) != 2 {
		t.Fatalf("expected 2 --plugin-dir flags, got %d", len(indices))
	}
	if args[indices[0]+1] != "/path/to/plugin1" || args[indices[1]+1] != "/path/to/plugin2" {
		t.Fatalf("unexpected --plugin-dir values: %v", args)
	}
}

func TestBuildQueryArgs_MaxThinkingTokens(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		MaxThinkingTokens: 8192,
	})
	assertFlag(t, args, "--max-thinking-tokens", "8192")
}

func TestBuildQueryArgs_ThinkingEnabled(t *testing.T) {
	cfg := ThinkingEnabled(4096)
	args := buildQueryArgs("hello", &Options{Thinking: &cfg})
	assertFlag(t, args, "--max-thinking-tokens", "4096")
	assertNoFlag(t, args, "--thinking")
}

func TestBuildQueryArgs_ThinkingAdaptive(t *testing.T) {
	cfg := ThinkingAdaptive()
	args := buildQueryArgs("hello", &Options{Thinking: &cfg})
	assertFlag(t, args, "--thinking", "adaptive")
	assertNoFlag(t, args, "--max-thinking-tokens")
}

func TestBuildQueryArgs_ThinkingDisabled(t *testing.T) {
	cfg := ThinkingDisabled()
	args := buildQueryArgs("hello", &Options{Thinking: &cfg})
	assertFlag(t, args, "--thinking", "disabled")
}

func TestBuildQueryArgs_SystemPromptAlwaysSent(t *testing.T) {
	args := buildQueryArgs("hello", &Options{})
	assertFlag(t, args, "--system-prompt", "")
}

func TestBuildQueryArgs_SystemPromptFile(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SystemPromptSource: &SystemPromptSource{Type: "file", Path: "/tmp/prompt.md"},
	})
	assertFlag(t, args, "--system-prompt-file", "/tmp/prompt.md")
	assertNoFlag(t, args, "--system-prompt")
}

func TestBuildQueryArgs_AppendSystemPrompt(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SystemPromptSource: &SystemPromptSource{Type: "preset", Append: "extra instructions"},
	})
	assertFlag(t, args, "--append-system-prompt", "extra instructions")
	assertNoFlag(t, args, "--system-prompt")
}

func TestBuildQueryArgs_ToolsPreset(t *testing.T) {
	args := buildQueryArgs("hello", &Options{ToolsPreset: "default"})
	assertFlag(t, args, "--tools", "default")
}

func TestBuildQueryArgs_ToolsEmptyList(t *testing.T) {
	args := buildQueryArgs("hello", &Options{Tools: []string{}})
	assertFlag(t, args, "--tools", "")
}

func TestBuildQueryArgs_SettingSourcesEqualForm(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		SettingSources: []string{"user", "project"},
	})
	assertHasFlag(t, args, "--setting-sources=user,project")
}

func TestBuildQueryArgs_JsonSchema(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{}}
	args := buildQueryArgs("hello", &Options{
		OutputFormat: schema,
	})
	idx := slices.Index(args, "--json-schema")
	if idx < 0 {
		t.Fatal("--json-schema flag not found")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(args[idx+1]), &decoded); err != nil {
		t.Fatalf("--json-schema value not valid JSON: %v", err)
	}
}

func TestBuildQueryArgs_Settings(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		Settings: "/path/to/settings.json",
	})
	assertFlag(t, args, "--settings", "/path/to/settings.json")
}

func TestBuildQueryArgs_AllowedTools_CamelCase(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		AllowedTools: []string{"tool1", "tool2"},
	})
	assertFlag(t, args, "--allowedTools", "tool1,tool2")
	assertNoFlag(t, args, "--allowed-tools")
}

func TestBuildQueryArgs_DisallowedTools_CamelCase(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		DisallowedTools: []string{"tool1"},
	})
	assertFlag(t, args, "--disallowedTools", "tool1")
	assertNoFlag(t, args, "--disallowed-tools")
}

func TestBuildQueryArgs_Betas_CommaSeparated(t *testing.T) {
	args := buildQueryArgs("hello", &Options{
		Betas: []string{"feature1", "feature2"},
	})
	assertFlag(t, args, "--betas", "feature1,feature2")
	assertNoFlag(t, args, "--beta")
}

func TestBuildQueryArgs_ExtraArgs(t *testing.T) {
	val := "bar"
	args := buildQueryArgs("hello", &Options{
		ExtraArgs: map[string]*string{
			"--foo":     &val,
			"--verbose": nil,
		},
	})
	assertFlag(t, args, "--foo", "bar")
	// --verbose as extra arg (boolean) should appear at least twice (once from default, once from extra)
	count := 0
	for _, a := range args {
		if a == "--verbose" {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected --verbose from both default and ExtraArgs, got count=%d", count)
	}
}

func TestBuildEnv_Defaults(t *testing.T) {
	env := buildEnv(&Options{})
	if env["CLAUDE_CODE_ENTRYPOINT"] != "sdk-go" {
		t.Fatalf("expected CLAUDE_CODE_ENTRYPOINT=sdk-go, got %q", env["CLAUDE_CODE_ENTRYPOINT"])
	}
	if env["CLAUDE_AGENT_SDK_VERSION"] != sdkVersion {
		t.Fatalf("expected CLAUDE_AGENT_SDK_VERSION=%s, got %q", sdkVersion, env["CLAUDE_AGENT_SDK_VERSION"])
	}
	if _, ok := env["CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING"]; ok {
		t.Fatal("file checkpointing env should not be set when disabled")
	}
}

func TestBuildEnv_FileCheckpointing(t *testing.T) {
	env := buildEnv(&Options{FileCheckpointing: true})
	if env["CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING"] != "true" {
		t.Fatal("expected file checkpointing env var to be set")
	}
}

func TestBuildEnv_UserEnvMerged(t *testing.T) {
	env := buildEnv(&Options{Env: map[string]string{"MY_VAR": "hello"}})
	if env["MY_VAR"] != "hello" {
		t.Fatalf("expected MY_VAR=hello, got %q", env["MY_VAR"])
	}
	if env["CLAUDE_CODE_ENTRYPOINT"] != "sdk-go" {
		t.Fatal("SDK env vars should still be present")
	}
}

func TestSendInitializeRequest(t *testing.T) {
	var sent []byte
	tr := &mockTransport{sendFn: func(line []byte) error {
		sent = line
		return nil
	}}
	exclude := true
	o := &Options{
		Agents: map[string]AgentDefinition{
			"helper": {Model: "sonnet"},
		},
		Skills:                 []string{"search", "edit"},
		ExcludeDynamicSections: &exclude,
	}
	if err := sendInitializeRequest(tr, o); err != nil {
		t.Fatal(err)
	}
	var req map[string]any
	if err := json.Unmarshal(sent, &req); err != nil {
		t.Fatal(err)
	}
	if req["type"] != "initialize" {
		t.Fatalf("expected type=initialize, got %v", req["type"])
	}
	if req["excludeDynamicSections"] != true {
		t.Fatalf("expected excludeDynamicSections=true, got %v", req["excludeDynamicSections"])
	}
	skills := req["skills"].([]any)
	if len(skills) != 2 || skills[0] != "search" {
		t.Fatalf("unexpected skills: %v", skills)
	}
}

type mockTransport struct {
	sendFn func([]byte) error
}

func (m *mockTransport) Start(_ context.Context) error     { return nil }
func (m *mockTransport) Send(line []byte) error            { return m.sendFn(line) }
func (m *mockTransport) Receive() ([]byte, error)          { return nil, nil }
func (m *mockTransport) Close() error                      { return nil }

// helpers

func assertFlag(t *testing.T, args []string, flag, value string) {
	t.Helper()
	idx := slices.Index(args, flag)
	if idx < 0 {
		t.Fatalf("flag %s not found in args: %v", flag, args)
	}
	if idx+1 >= len(args) {
		t.Fatalf("flag %s has no value (at end of args)", flag)
	}
	if args[idx+1] != value {
		t.Fatalf("flag %s: want %q, got %q", flag, value, args[idx+1])
	}
}

func assertHasFlag(t *testing.T, args []string, flag string) {
	t.Helper()
	if !slices.Contains(args, flag) {
		t.Fatalf("flag %s not found in args: %v", flag, args)
	}
}

func assertNoFlag(t *testing.T, args []string, flag string) {
	t.Helper()
	if slices.Contains(args, flag) {
		t.Fatalf("flag %s should not be in args: %v", flag, args)
	}
}

func allFlagIndices(args []string, flag string) []int {
	var indices []int
	for i, a := range args {
		if a == flag {
			indices = append(indices, i)
		}
	}
	return indices
}

