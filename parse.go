package claude

import (
	"encoding/json"
	"fmt"
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

func parseLine(line []byte) (Message, error) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return nil, &JSONDecodeError{RawLine: string(line), Err: err}
	}

	switch env.Type {
	case "user":
		return parseUserMessage(line)
	case "assistant":
		return parseAssistantMessage(line)
	case "system":
		var m SystemMessage
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "result":
		var m ResultMessage
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "task_started":
		var m TaskStarted
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "task_progress":
		var m TaskProgress
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "task_notification":
		var m TaskNotification
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "rate_limit":
		var m RateLimitEvent
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	default:
		return nil, &MessageParseError{
			TypeField: env.Type,
			RawJSON:   string(line),
			Err:       fmt.Errorf("unknown message type %q", env.Type),
		}
	}
}

func parseUserMessage(line []byte) (Message, error) {
	var raw rawContentMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, &MessageParseError{TypeField: "user", RawJSON: string(line), Err: err}
	}
	blocks, err := parseContentBlocks(raw.Content)
	if err != nil {
		return nil, &MessageParseError{TypeField: "user", RawJSON: string(line), Err: err}
	}
	return &UserMessage{Role: raw.Role, Content: blocks}, nil
}

func parseAssistantMessage(line []byte) (Message, error) {
	var raw rawContentMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, &MessageParseError{TypeField: "assistant", RawJSON: string(line), Err: err}
	}
	blocks, err := parseContentBlocks(raw.Content)
	if err != nil {
		return nil, &MessageParseError{TypeField: "assistant", RawJSON: string(line), Err: err}
	}
	return &AssistantMessage{Role: raw.Role, Content: blocks}, nil
}

func parseContentBlocks(raws []json.RawMessage) ([]ContentBlock, error) {
	blocks := make([]ContentBlock, 0, len(raws))
	for _, raw := range raws {
		var rb rawBlock
		if err := json.Unmarshal(raw, &rb); err != nil {
			return nil, err
		}
		switch rb.Type {
		case "text":
			var b TextBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		case "thinking":
			var b ThinkingBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		case "tool_use":
			var b ToolUseBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		case "tool_result":
			var b ToolResultBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return nil, err
			}
			blocks = append(blocks, &b)
		}
	}
	return blocks, nil
}
