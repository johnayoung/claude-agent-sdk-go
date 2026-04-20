package agenttest

import (
	"encoding/json"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

// NewTextMessage returns an AssistantMessage with a single TextBlock.
func NewTextMessage(text string) *agent.AssistantMessage {
	return &agent.AssistantMessage{
		Role: "assistant",
		Content: []agent.ContentBlock{
			&agent.TextBlock{Type: "text", Text: text},
		},
	}
}

// NewToolUseMessage returns an AssistantMessage with a single ToolUseBlock.
func NewToolUseMessage(id, name string, input any) *agent.AssistantMessage {
	raw, _ := json.Marshal(input)
	return &agent.AssistantMessage{
		Role: "assistant",
		Content: []agent.ContentBlock{
			&agent.ToolUseBlock{Type: "tool_use", ID: id, Name: name, Input: raw},
		},
	}
}

// NewResultMessage returns a ResultMessage with the given result text and session ID.
func NewResultMessage(result, sessionID string) *agent.ResultMessage {
	return &agent.ResultMessage{
		Subtype:   "success",
		Result:    result,
		SessionID: sessionID,
	}
}
