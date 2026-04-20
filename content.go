package claude

import "encoding/json"

// ContentBlock is the common interface for all content block types.
type ContentBlock interface {
	BlockType() string
}

type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (b *TextBlock) BlockType() string { return "text" }

type ThinkingBlock struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

func (b *ThinkingBlock) BlockType() string { return "thinking" }

type ToolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (b *ToolUseBlock) BlockType() string { return "tool_use" }

type ToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

func (b *ToolResultBlock) BlockType() string { return "tool_result" }
