package parse

import (
	"encoding/json"
	"fmt"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

type envelope struct {
	Type string `json:"type"`
}

type rawContentMessage struct {
	Role    string            `json:"role"`
	Content []json.RawMessage `json:"content"`
}

type rawBlock struct {
	Type string `json:"type"`
}

// ParseLine parses a single newline-delimited JSON line into a typed Message.
// Returns JSONDecodeError for malformed JSON and MessageParseError for unknown types.
func ParseLine(line []byte) (agent.Message, error) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return nil, &agent.JSONDecodeError{RawLine: string(line), Err: err}
	}

	switch env.Type {
	case "user":
		return parseUserMessage(line)
	case "assistant":
		return parseAssistantMessage(line)
	case "system":
		var m agent.SystemMessage
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &agent.MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "result":
		var m agent.ResultMessage
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &agent.MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "task_started":
		var m agent.TaskStarted
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &agent.MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "task_progress":
		var m agent.TaskProgress
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &agent.MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "task_notification":
		var m agent.TaskNotification
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &agent.MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	default:
		return nil, &agent.MessageParseError{
			TypeField: env.Type,
			RawJSON:   string(line),
			Err:       fmt.Errorf("unknown message type %q", env.Type),
		}
	}
}

func parseUserMessage(line []byte) (agent.Message, error) {
	var raw rawContentMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, &agent.MessageParseError{TypeField: "user", RawJSON: string(line), Err: err}
	}
	blocks, err := parseContentBlocks(raw.Content)
	if err != nil {
		return nil, &agent.MessageParseError{TypeField: "user", RawJSON: string(line), Err: err}
	}
	return &agent.UserMessage{Role: raw.Role, Content: blocks}, nil
}

func parseAssistantMessage(line []byte) (agent.Message, error) {
	var raw rawContentMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, &agent.MessageParseError{TypeField: "assistant", RawJSON: string(line), Err: err}
	}
	blocks, err := parseContentBlocks(raw.Content)
	if err != nil {
		return nil, &agent.MessageParseError{TypeField: "assistant", RawJSON: string(line), Err: err}
	}
	return &agent.AssistantMessage{Role: raw.Role, Content: blocks}, nil
}

func parseContentBlocks(raws []json.RawMessage) ([]agent.ContentBlock, error) {
	blocks := make([]agent.ContentBlock, 0, len(raws))
	for _, raw := range raws {
		var rb rawBlock
		if err := json.Unmarshal(raw, &rb); err != nil {
			return nil, err
		}
		switch rb.Type {
		case "text":
			var b agent.TextBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		case "thinking":
			var b agent.ThinkingBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		case "tool_use":
			var b agent.ToolUseBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		case "tool_result":
			var b agent.ToolResultBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		}
		// unknown content block types are silently skipped
	}
	return blocks, nil
}
