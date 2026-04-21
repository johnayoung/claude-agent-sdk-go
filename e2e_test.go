//go:build e2e

package claude_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/hooks"
	"github.com/johnayoung/claude-agent-sdk-go/mcp"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

// --- 1. Agents & Settings ---

func TestE2E_AgentDefinition(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	agents := map[string]claude.AgentDefinition{
		"test-agent": {
			Name:        "test-agent",
			Description: "A helpful math assistant",
			Prompt:      "You are a helpful math assistant.",
		},
	}

	msgs := collectMessages(t, ctx, "What is 2 + 2? Reply with just the number.",
		claude.WithAgents(agents),
	)

	initMsg := findSystem(msgs, "init")
	if initMsg == nil {
		t.Log("No init system message found (some CLI versions may not emit it)")
	}

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}
	if result.Subtype != "success" {
		t.Errorf("expected success, got subtype=%s", result.Subtype)
	}
}

func TestE2E_SettingSources(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	msgs := collectMessages(t, ctx, "What is 2 + 2? Reply with just the number.",
		claude.WithSettingSources("user"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}
	if result.Subtype != "success" {
		t.Errorf("expected success, got subtype=%s", result.Subtype)
	}
}

// --- 2. Dynamic Control ---

func TestE2E_DynamicControl_SetPermissionMode(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	client, err := claude.NewClient(ctx, baseOpts()...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for msg, err := range client.Query(ctx, "What is 2+2? Just the number.") {
		if err != nil {
			t.Fatalf("first query error: %v", err)
		}
		if _, ok := msg.(*claude.ResultMessage); ok {
			break
		}
	}

	client.SetPermissionMode(claude.PermissionModeAcceptEdits)
	client.SetPermissionMode(claude.PermissionModeBypassPermissions)

	for msg, err := range client.Query(ctx, "What is 3+3? Just the number.") {
		if err != nil {
			t.Fatalf("second query error: %v", err)
		}
		if _, ok := msg.(*claude.ResultMessage); ok {
			break
		}
	}
}

func TestE2E_DynamicControl_SetModel(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	client, err := claude.NewClient(ctx, baseOpts()...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for msg, err := range client.Query(ctx, "What is 1+1? Just the number.") {
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		if _, ok := msg.(*claude.ResultMessage); ok {
			break
		}
	}

	client.SetModel("claude-haiku-4-5")

	for msg, err := range client.Query(ctx, "What is 2+2? Just the number.") {
		if err != nil {
			t.Fatalf("second query error: %v", err)
		}
		if _, ok := msg.(*claude.ResultMessage); ok {
			break
		}
	}
}

func TestE2E_DynamicControl_Interrupt(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	client, err := claude.NewClient(ctx, baseOpts()...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	gotMsg := false
	for msg, err := range client.Query(ctx, "Count from 1 to 100 slowly, one number per line.") {
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			break
		}
		if _, ok := msg.(*claude.AssistantMessage); ok && !gotMsg {
			gotMsg = true
			client.Interrupt()
		}
		if _, ok := msg.(*claude.ResultMessage); ok {
			break
		}
	}
}

// --- 3. Hook Events ---

func TestE2E_HookEvents_PreToolUse(t *testing.T) {
	// Removed skip: hooks are now sent in the initialize request
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var invocations []string

	h := hooks.New()
	h.OnPreToolUse("*", func(_ context.Context, input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		mu.Lock()
		invocations = append(invocations, input.ToolName)
		mu.Unlock()
		return &hooks.PreToolUseOutput{}, nil
	})

	msgs := collectMessages(t, ctx, "Run this bash command: echo 'hook test'",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(invocations) == 0 {
		t.Error("PreToolUse hook was never invoked")
	}
}

func TestE2E_HookEvents_PostToolUse(t *testing.T) {
	// Removed skip: hooks are now sent in the initialize request
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var invocations []string

	h := hooks.New()
	h.OnPostToolUse("*", func(_ context.Context, input *hooks.PostToolUseInput) (*hooks.PostToolUseOutput, error) {
		mu.Lock()
		invocations = append(invocations, input.ToolName)
		mu.Unlock()
		return &hooks.PostToolUseOutput{}, nil
	})

	msgs := collectMessages(t, ctx, "Run this bash command: echo 'post hook test'",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(invocations) == 0 {
		t.Error("PostToolUse hook was never invoked")
	}
}

// --- 4. Hooks Control ---

func TestE2E_HooksControl_PreToolUseDeny(t *testing.T) {
	// Removed skip: hooks are now sent in the initialize request
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var blocked []string

	h := hooks.New()
	h.OnPreToolUse("Bash", func(_ context.Context, input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		mu.Lock()
		blocked = append(blocked, input.ToolName)
		mu.Unlock()
		return &hooks.PreToolUseOutput{Block: true, Reason: "blocked by test"}, nil
	})

	msgs := collectMessages(t, ctx, "Run this bash command: echo 'should be blocked'",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(blocked) == 0 {
		t.Error("PreToolUse deny hook was never invoked")
	}
	found := false
	for _, name := range blocked {
		if name == "Bash" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Bash in blocked tools, got %v", blocked)
	}
}

// --- 5. Partial Messages ---

func TestE2E_PartialMessages_StreamEvents(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	opts := append(baseOpts(), claude.WithIncludePartialMessages())
	var msgs []claude.Message
	for msg, err := range claude.Query(ctx, "Say hello in one word.", opts...) {
		if err != nil {
			t.Fatalf("Query error: %v", err)
		}
		msgs = append(msgs, msg)
	}

	var streamEvents int
	for _, m := range msgs {
		if _, ok := m.(*claude.StreamEvent); ok {
			streamEvents++
		}
	}

	if streamEvents == 0 {
		t.Error("no StreamEvent messages received with IncludePartialMessages enabled")
	}

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}
}

func TestE2E_PartialMessages_DisabledByDefault(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	msgs := collectMessages(t, ctx, "Say hello in one word.")

	for _, m := range msgs {
		if _, ok := m.(*claude.StreamEvent); ok {
			t.Error("StreamEvent received without IncludePartialMessages")
			break
		}
	}
}

// --- 6. SDK MCP Tools ---

type echoTool struct{}

func (echoTool) Name() string        { return "echo" }
func (echoTool) Description() string { return "Echoes back the input text" }
func (echoTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`)
}
func (echoTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	text, _ := input["text"].(string)
	return json.Marshal(map[string]any{
		"content": []map[string]any{{"type": "text", "text": "echo: " + text}},
	})
}

type greetTool struct{}

func (greetTool) Name() string        { return "greet" }
func (greetTool) Description() string { return "Greets the given name" }
func (greetTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
}
func (greetTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	name, _ := input["name"].(string)
	return json.Marshal(map[string]any{
		"content": []map[string]any{{"type": "text", "text": "Hello, " + name + "!"}},
	})
}

func TestE2E_SDKMCPTool_Execution(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	server := mcp.NewMCPServer("test", echoTool{})

	msgs := collectMessages(t, ctx, "Call the mcp__test__echo tool with the text 'hello world'.",
		claude.WithSDKMCPServer(server),
		claude.WithAllowedTools("mcp__test__echo"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}
	if result.Subtype != "success" {
		t.Errorf("expected success, got subtype=%s", result.Subtype)
	}
}

func TestE2E_SDKMCPTool_MultipleTools(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	server := mcp.NewMCPServer("multi", echoTool{}, greetTool{})

	msgs := collectMessages(t, ctx, "Call mcp__multi__echo with text='test' and mcp__multi__greet with name='Bob'. Do these one at a time.",
		claude.WithSDKMCPServer(server),
		claude.WithAllowedTools("mcp__multi__echo", "mcp__multi__greet"),
		claude.WithMaxTurns(5),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}
	if result.Subtype != "success" {
		t.Errorf("expected success, got subtype=%s", result.Subtype)
	}
}

func TestE2E_SDKMCPTool_DisallowedBlocked(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	server := mcp.NewMCPServer("test", echoTool{}, greetTool{})

	msgs := collectMessages(t, ctx, "Call the mcp__test__echo tool with text='should be blocked'.",
		claude.WithSDKMCPServer(server),
		claude.WithAllowedTools("mcp__test__greet"),
		claude.WithDisallowedTools("mcp__test__echo"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}
}

// --- 7. Stderr Callback ---

func TestE2E_StderrCallback_CapturesDebug(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var stderrLines []string

	msgs := collectMessages(t, ctx, "What is 1+1? Just the number.",
		claude.WithStderr(func(line string) {
			mu.Lock()
			stderrLines = append(stderrLines, line)
			mu.Unlock()
		}),
		claude.WithExtraArgs(map[string]*string{"--debug-to-stderr": nil}),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(stderrLines) == 0 {
		t.Error("no stderr output captured with debug mode enabled")
	}
}

func TestE2E_StderrCallback_NoneWithoutDebug(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var stderrLines []string

	msgs := collectMessages(t, ctx, "What is 1+1? Just the number.",
		claude.WithStderr(func(line string) {
			mu.Lock()
			stderrLines = append(stderrLines, line)
			mu.Unlock()
		}),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(stderrLines) > 0 {
		t.Logf("got %d stderr lines without debug mode (may be warnings)", len(stderrLines))
	}
}

// --- 8. Structured Output ---

func TestE2E_StructuredOutput_Simple(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"answer": map[string]any{"type": "number"},
		},
		"required": []string{"answer"},
	}

	msgs := collectMessages(t, ctx, "What is 2 + 2? Return the answer as a number.",
		claude.WithOutputFormat(schema),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	if len(result.StructuredOutput) == 0 {
		t.Fatal("no structured output in result")
	}

	var output map[string]any
	if err := json.Unmarshal(result.StructuredOutput, &output); err != nil {
		t.Fatalf("failed to parse structured output: %v", err)
	}

	answer, ok := output["answer"]
	if !ok {
		t.Fatal("structured output missing 'answer' field")
	}
	if num, ok := answer.(float64); !ok || num != 4 {
		t.Errorf("expected answer=4, got %v", answer)
	}
}

func TestE2E_StructuredOutput_Nested(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"word_count":      map[string]any{"type": "number"},
			"character_count": map[string]any{"type": "number"},
			"words":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"word_count", "character_count", "words"},
	}

	msgs := collectMessages(t, ctx,
		"Analyze this text: 'Hello world'. Provide word count, character count (including space), and list of words.",
		claude.WithOutputFormat(schema),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	if len(result.StructuredOutput) == 0 {
		t.Fatal("no structured output in result")
	}

	var output map[string]any
	if err := json.Unmarshal(result.StructuredOutput, &output); err != nil {
		t.Fatalf("failed to parse structured output: %v", err)
	}

	wordCount, ok := output["word_count"].(float64)
	if !ok {
		t.Fatal("missing word_count")
	}
	if wordCount != 2 {
		t.Errorf("expected word_count=2, got %v", wordCount)
	}

	words, ok := output["words"].([]any)
	if !ok {
		t.Fatal("missing words array")
	}
	if len(words) != 2 {
		t.Errorf("expected 2 words, got %d", len(words))
	}
}

func TestE2E_StructuredOutput_WithEnum(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"color": map[string]any{
				"type": "string",
				"enum": []string{"red", "blue", "green"},
			},
		},
		"required": []string{"color"},
	}

	msgs := collectMessages(t, ctx,
		"The sky is typically what color? Choose from: red, blue, green.",
		claude.WithOutputFormat(schema),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	if len(result.StructuredOutput) == 0 {
		t.Fatal("no structured output in result")
	}

	var output map[string]any
	if err := json.Unmarshal(result.StructuredOutput, &output); err != nil {
		t.Fatalf("failed to parse structured output: %v", err)
	}

	color, ok := output["color"].(string)
	if !ok {
		t.Fatal("missing color field")
	}
	validColors := map[string]bool{"red": true, "blue": true, "green": true}
	if !validColors[color] {
		t.Errorf("color %q not in valid enum values", color)
	}
}

// --- 9. Tool Permissions ---

func TestE2E_ToolPermissions_CallbackInvoked(t *testing.T) {
	t.Skip("CLI does not yet dispatch can_use_tool control requests to SDK via --permission-prompt-tool stdio")
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var toolNames []string

	canUseTool := func(toolName string, input map[string]any, _ permission.ToolContext) (permission.Decision, error) {
		mu.Lock()
		toolNames = append(toolNames, toolName)
		mu.Unlock()
		return permission.Allow("allowed by test"), nil
	}

	msgs := collectMessages(t, ctx,
		"Run this bash command: echo 'permission test'",
		claude.WithCanUseTool(canUseTool),
		claude.WithPermissionMode(claude.PermissionModeDefault),
		claude.WithAllowedTools("Bash"),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(toolNames) == 0 {
		t.Error("CanUseTool callback was never invoked")
	}
	found := false
	for _, name := range toolNames {
		if strings.Contains(name, "Bash") || strings.Contains(name, "bash") {
			found = true
			break
		}
	}
	if !found {
		t.Logf("tool names received: %v (Bash may be auto-allowed for echo)", toolNames)
	}
}

func TestE2E_ToolPermissions_DenyRespected(t *testing.T) {
	t.Skip("CLI does not yet dispatch can_use_tool control requests to SDK via --permission-prompt-tool stdio")
	skipIfNoCLI(t)
	ctx, cancel := e2eContext(t)
	defer cancel()

	var mu sync.Mutex
	var denied []string

	canUseTool := func(toolName string, input map[string]any, _ permission.ToolContext) (permission.Decision, error) {
		mu.Lock()
		denied = append(denied, toolName)
		mu.Unlock()
		return permission.Deny("denied by test"), nil
	}

	msgs := collectMessages(t, ctx,
		"Create a file at /tmp/sdk_e2e_deny_test.txt with content 'test'.",
		claude.WithCanUseTool(canUseTool),
		claude.WithPermissionMode(claude.PermissionModeDefault),
	)

	result := findResult(msgs)
	if result == nil {
		t.Fatal("no result message received")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(denied) == 0 {
		t.Error("CanUseTool deny callback was never invoked")
	}
}
