// custom-tools demonstrates registering in-process tools via the MCP SDK server.
// The Tool interface is implemented locally and served to the Claude CLI without
// a separate subprocess — the SDK handles the MCP protocol internally.
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
	"github.com/johnayoung/claude-agent-sdk-go/agent"
	"github.com/johnayoung/claude-agent-sdk-go/mcp"
)

// upperTool is a simple MCP tool that converts text to uppercase.
type upperTool struct{}

func (upperTool) Name() string        { return "to_upper" }
func (upperTool) Description() string { return "Converts a string to uppercase" }

// InputSchema returns a JSON Schema describing the tool's accepted parameters.
func (upperTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"text": {"type": "string", "description": "Text to convert"}
		},
		"required": ["text"]
	}`)
}

// Run is called by the SDK when the Claude CLI invokes this tool.
func (upperTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	text, _ := input["text"].(string)
	result := strings.ToUpper(text)
	return json.Marshal(result)
}

func main() {
	ctx := context.Background()

	// NewMCPServer wraps in-process Tool implementations so the Claude CLI can call them.
	server := mcp.NewMCPServer(upperTool{})

	// WithMCPServers registers the server; the CLI discovers tools from it automatically.
	for msg, err := range claude.Query(ctx,
		"Use the to_upper tool to convert 'hello world' to uppercase, then tell me the result.",
		agent.WithMCPServers(agent.MCPServerConfig{
			Name: server.Name,
			Type: agent.MCPServerTypeSDK,
		}),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		switch m := msg.(type) {
		case *agent.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *agent.TextBlock:
					fmt.Println(b.Text)
				case *agent.ToolUseBlock:
					fmt.Printf("[tool call: %s]\n", b.Name)
				}
			}
		case *agent.ResultMessage:
			fmt.Printf("\ncost: $%.6f\n", m.CostUSD)
		}
	}
}
