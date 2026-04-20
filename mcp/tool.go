package mcp

import (
	"context"
	"encoding/json"
)

// Tool defines the interface for an MCP-compatible tool that can be served to the Claude CLI.
type Tool interface {
	Name() string
	Description() string
	InputSchema() json.RawMessage
	Run(ctx context.Context, input map[string]any) (json.RawMessage, error)
}
