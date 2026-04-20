package claude

import (
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

