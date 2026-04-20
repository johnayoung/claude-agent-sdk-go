package agenttest

import (
	"encoding/json"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

// NewTextMessage returns an AssistantMessage with a single TextBlock.
func NewTextMessage(text string) *claude.AssistantMessage {
	return &claude.AssistantMessage{
		Role: "assistant",
		Content: []claude.ContentBlock{
			&claude.TextBlock{Type: "text", Text: text},
		},
	}
}

// NewToolUseMessage returns an AssistantMessage with a single ToolUseBlock.
func NewToolUseMessage(id, name string, input any) *claude.AssistantMessage {
	raw, _ := json.Marshal(input)
	return &claude.AssistantMessage{
		Role: "assistant",
		Content: []claude.ContentBlock{
			&claude.ToolUseBlock{Type: "tool_use", ID: id, Name: name, Input: raw},
		},
	}
}

// NewResultMessage returns a ResultMessage with the given result text and session ID.
func NewResultMessage(result, sessionID string) *claude.ResultMessage {
	return &claude.ResultMessage{
		Subtype:   "success",
		Result:    result,
		SessionID: sessionID,
	}
}
