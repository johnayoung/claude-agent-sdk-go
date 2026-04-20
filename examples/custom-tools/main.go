// custom-tools demonstrates registering in-process tools via the MCP SDK server.
// The Tool interface is implemented locally and served to the Claude CLI without
// a separate subprocess -- the SDK handles the MCP protocol internally.
//
// Run:
//
//	go run ./examples/custom-tools
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/mcp"
)

// upperTool is a simple MCP tool that converts text to uppercase.
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
	result := strings.ToUpper(text)
	return json.Marshal(result)
}

func main() {
	ctx := context.Background()

	server := mcp.NewMCPServer(upperTool{})

	for msg, err := range claude.Query(ctx,
		"Use the to_upper tool to convert 'hello world' to uppercase, then tell me the result.",
		claude.WithMCPServers(claude.MCPServerConfig{
			Name: server.Name,
			Type: claude.MCPServerTypeSDK,
		}),
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
			fmt.Printf("\ncost: $%.6f\n", m.TotalCostUSD)
		}
	}
}
