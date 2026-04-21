// tool-permission-callback demonstrates using WithCanUseTool to control tool
// execution with custom permission logic. It registers an in-process MCP tool,
// then uses a permission callback that auto-allows, denies, and modifies input
// depending on the tool being invoked.
//
// Run:
//
//	go run ./examples/tool-permission-callback
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/mcp"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

type upperTool struct{}

func (upperTool) Name() string        { return "to_upper" }
func (upperTool) Description() string { return "Converts a string to uppercase" }

func (upperTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"text": {"type": "string", "description": "Text to convert"}
		},
		"required": ["text"]
	}`)
}

func (upperTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	text, _ := input["text"].(string)
	return json.Marshal(strings.ToUpper(text))
}

type reverseTool struct{}

func (reverseTool) Name() string        { return "reverse" }
func (reverseTool) Description() string { return "Reverses a string" }

func (reverseTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"text": {"type": "string", "description": "Text to reverse"}
		},
		"required": ["text"]
	}`)
}

func (reverseTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	text, _ := input["text"].(string)
	runes := []rune(text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return json.Marshal(string(runes))
}

type blockedTool struct{}

func (blockedTool) Name() string        { return "delete_everything" }
func (blockedTool) Description() string { return "Deletes everything (dangerous)" }

func (blockedTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"confirm": {"type": "boolean", "description": "Confirm deletion"}
		},
		"required": ["confirm"]
	}`)
}

func (blockedTool) Run(_ context.Context, _ map[string]any) (json.RawMessage, error) {
	return json.Marshal("this should never execute")
}

// toolBaseName extracts the short tool name from an MCP-prefixed name.
// MCP tools arrive as "mcp__<server>__<tool>"; built-in tools are bare names.
func toolBaseName(name string) string {
	if i := strings.LastIndex(name, "__"); i >= 0 {
		return name[i+2:]
	}
	return name
}

func permissionCallback(toolName string, input map[string]any, ctx permission.ToolContext) (permission.Decision, error) {
	baseName := toolBaseName(toolName)
	inputJSON, _ := json.Marshal(input)
	fmt.Printf("\n[permission] Tool: %s | Input: %s\n", toolName, string(inputJSON))

	// Auto-allow read-only / safe tools
	safeTools := map[string]bool{"Read": true, "Glob": true, "Grep": true}
	if safeTools[baseName] {
		fmt.Printf("[permission] ALLOW (safe tool)\n")
		return permission.Allow("read-only tool"), nil
	}

	// Deny a blocklisted tool
	if baseName == "delete_everything" {
		fmt.Printf("[permission] DENY (blocklisted)\n")
		return permission.Deny("tool is blocklisted for safety"), nil
	}

	// Modify input: when to_upper is called, append " (modified by permission callback)"
	// to the text input to demonstrate AllowWithUpdates
	if baseName == "to_upper" {
		text, _ := input["text"].(string)
		modified := map[string]any{"text": text + " (modified by permission callback)"}
		fmt.Printf("[permission] ALLOW with modified input: %v\n", modified)
		return permission.AllowWithUpdates("input modified for demonstration", modified, nil), nil
	}

	// Default: allow everything else
	fmt.Printf("[permission] ALLOW (default)\n")
	return permission.Allow("no restriction"), nil
}

func main() {
	ctx := context.Background()

	server := mcp.NewMCPServer("permission-tools", upperTool{}, reverseTool{}, blockedTool{})

	fmt.Println("=== Tool Permission Callback Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  1. WithPermissionMode -- setting PermissionModeDefault")
	fmt.Println("  2. WithCanUseTool -- custom permission callback")
	fmt.Println("  3. ResultAllow, ResultDeny, AllowWithUpdates return values")
	fmt.Println("  4. Input modification via AllowWithUpdates")
	fmt.Println()

	prompt := `You have three tools available: to_upper, reverse, and delete_everything.
Please do the following in order:
1. Use to_upper to convert "hello world" to uppercase
2. Use reverse to reverse "permission test"
3. Try to use delete_everything with confirm=true
Report what happened with each tool call.`

	for msg, err := range claude.Query(ctx, prompt,
		claude.WithSDKMCPServer(server),
		claude.WithPermissionMode(claude.PermissionModeDefault),
		claude.WithCanUseTool(permissionCallback),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *claude.TextBlock:
					fmt.Println(b.Text)
				case *claude.ToolUseBlock:
					fmt.Printf("[tool call: %s]\n", b.Name)
				}
			}
		case *claude.ResultMessage:
			var cost float64
			if m.TotalCostUSD != nil {
				cost = *m.TotalCostUSD
			}
			fmt.Printf("\ncost: $%.6f\n", cost)
		}
	}
}
