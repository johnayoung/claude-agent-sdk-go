package claude

import (
	"encoding/json"
	"fmt"
)

type envelope struct {
	Type string `json:"type"`
}

func parseLine(line []byte) (Message, error) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return nil, &JSONDecodeError{RawLine: string(line), Err: err}
	}

	var data map[string]any
	if err := json.Unmarshal(line, &data); err != nil {
		return nil, &JSONDecodeError{RawLine: string(line), Err: err}
	}

	switch env.Type {
	case "user":
		return parseUserMessage(data)
	case "assistant":
		return parseAssistantMessage(data)
	case "system":
		return parseSystemMessage(data, line)
	case "result":
		return parseResultMessage(line)
	case "stream_event":
		return parseStreamEvent(line)
	case "rate_limit_event":
		var m RateLimitEvent
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &MessageParseError{TypeField: env.Type, RawJSON: string(line), Err: err}
		}
		return &m, nil
	case "transcript_mirror":
		return parseTranscriptMirror(line)
	case "control_request":
		return parseControlRequest(line)
	case "control_response":
		return parseControlResponse(line)
	default:
		// Forward compatibility: skip unknown types
		return nil, nil
	}
}

func parseUserMessage(data map[string]any) (Message, error) {
	msg := &UserMessage{}

	msg.UUID, _ = data["uuid"].(string)
	msg.ParentToolUseID, _ = data["parent_tool_use_id"].(string)

	if tur, ok := data["tool_use_result"].(map[string]any); ok {
		msg.ToolUseResult = tur
	}

	messageData, ok := data["message"].(map[string]any)
	if !ok {
		return nil, &MessageParseError{TypeField: "user", RawJSON: fmt.Sprintf("%v", data), Err: fmt.Errorf("missing 'message' field")}
	}

	content := messageData["content"]
	switch c := content.(type) {
	case string:
		msg.Content = []ContentBlock{&TextBlock{Type: "text", Text: c}}
	case []any:
		blocks, err := parseContentBlocksFromSlice(c)
		if err != nil {
			return nil, &MessageParseError{TypeField: "user", RawJSON: fmt.Sprintf("%v", data), Err: err}
		}
		msg.Content = blocks
	}

	return msg, nil
}

func parseAssistantMessage(data map[string]any) (Message, error) {
	msg := &AssistantMessage{}

	msg.ParentToolUseID, _ = data["parent_tool_use_id"].(string)
	if errStr, ok := data["error"].(string); ok {
		msg.Error = AssistantMessageError(errStr)
	}
	msg.SessionID, _ = data["session_id"].(string)
	msg.UUID, _ = data["uuid"].(string)

	messageData, ok := data["message"].(map[string]any)
	if !ok {
		return nil, &MessageParseError{TypeField: "assistant", RawJSON: fmt.Sprintf("%v", data), Err: fmt.Errorf("missing 'message' field")}
	}

	msg.Model, _ = messageData["model"].(string)
	msg.MessageID, _ = messageData["id"].(string)
	msg.StopReason, _ = messageData["stop_reason"].(string)

	if usage, ok := messageData["usage"].(map[string]any); ok {
		msg.Usage = usage
	}

	contentRaw, ok := messageData["content"].([]any)
	if !ok {
		return nil, &MessageParseError{TypeField: "assistant", RawJSON: fmt.Sprintf("%v", data), Err: fmt.Errorf("missing 'message.content' field")}
	}

	blocks, err := parseContentBlocksFromSlice(contentRaw)
	if err != nil {
		return nil, &MessageParseError{TypeField: "assistant", RawJSON: fmt.Sprintf("%v", data), Err: err}
	}
	msg.Content = blocks

	return msg, nil
}

func parseSystemMessage(data map[string]any, line []byte) (Message, error) {
	subtype, _ := data["subtype"].(string)
	if subtype == "" {
		return nil, &MessageParseError{TypeField: "system", RawJSON: string(line), Err: fmt.Errorf("missing 'subtype' field")}
	}

	dataMap := make(map[string]any)
	for k, v := range data {
		dataMap[k] = v
	}

	base := SystemMessage{Subtype: subtype, Data: dataMap}

	switch subtype {
	case "task_started":
		m := &TaskStartedMessage{SystemMessage: base}
		m.TaskID, _ = data["task_id"].(string)
		m.Description, _ = data["description"].(string)
		m.UUID, _ = data["uuid"].(string)
		m.SessionID, _ = data["session_id"].(string)
		m.ToolUseID, _ = data["tool_use_id"].(string)
		m.TaskType, _ = data["task_type"].(string)
		return m, nil

	case "task_progress":
		m := &TaskProgressMessage{SystemMessage: base}
		m.TaskID, _ = data["task_id"].(string)
		m.Description, _ = data["description"].(string)
		m.UUID, _ = data["uuid"].(string)
		m.SessionID, _ = data["session_id"].(string)
		m.ToolUseID, _ = data["tool_use_id"].(string)
		m.LastToolName, _ = data["last_tool_name"].(string)
		if usage, ok := data["usage"].(map[string]any); ok {
			m.Usage = parseTaskUsage(usage)
		}
		return m, nil

	case "task_notification":
		m := &TaskNotificationMessage{SystemMessage: base}
		m.TaskID, _ = data["task_id"].(string)
		if s, ok := data["status"].(string); ok {
			m.Status = TaskNotificationStatus(s)
		}
		m.OutputFile, _ = data["output_file"].(string)
		m.Summary, _ = data["summary"].(string)
		m.UUID, _ = data["uuid"].(string)
		m.SessionID, _ = data["session_id"].(string)
		m.ToolUseID, _ = data["tool_use_id"].(string)
		if usage, ok := data["usage"].(map[string]any); ok {
			tu := parseTaskUsage(usage)
			m.Usage = &tu
		}
		return m, nil

	case "mirror_error":
		m := &MirrorErrorMessage{SystemMessage: base}
		if keyMap, ok := data["key"].(map[string]any); ok {
			sk := &SessionKey{}
			sk.ProjectKey, _ = keyMap["project_key"].(string)
			sk.SessionID, _ = keyMap["session_id"].(string)
			sk.Subpath, _ = keyMap["subpath"].(string)
			m.Key = sk
		}
		m.Error, _ = data["error"].(string)
		return m, nil

	default:
		return &base, nil
	}
}

func parseResultMessage(line []byte) (Message, error) {
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, &MessageParseError{TypeField: "result", RawJSON: string(line), Err: err}
	}

	m := &ResultMessage{}
	m.Subtype, _ = raw["subtype"].(string)
	if v, ok := raw["duration_ms"].(float64); ok {
		m.DurationMS = int64(v)
	}
	if v, ok := raw["duration_api_ms"].(float64); ok {
		m.DurationAPIMS = int64(v)
	}
	m.IsError, _ = raw["is_error"].(bool)
	if v, ok := raw["num_turns"].(float64); ok {
		m.NumTurns = int(v)
	}
	m.SessionID, _ = raw["session_id"].(string)
	m.StopReason, _ = raw["stop_reason"].(string)
	if v, ok := raw["total_cost_usd"].(float64); ok {
		m.TotalCostUSD = &v
	}
	if v, ok := raw["usage"].(map[string]any); ok {
		m.Usage = v
	}
	m.Result, _ = raw["result"].(string)
	if v, ok := raw["structured_output"]; ok && v != nil {
		b, _ := json.Marshal(v)
		m.StructuredOutput = b
	}
	if v, ok := raw["model_usage"].(map[string]any); ok {
		m.ModelUsage = v
	}
	if v, ok := raw["permission_denials"].([]any); ok {
		m.PermissionDenials = v
	}
	if v, ok := raw["errors"].([]any); ok {
		m.Errors = make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				m.Errors = append(m.Errors, s)
			}
		}
	}
	m.UUID, _ = raw["uuid"].(string)

	return m, nil
}

func parseStreamEvent(line []byte) (Message, error) {
	var m StreamEvent
	if err := json.Unmarshal(line, &m); err != nil {
		return nil, &MessageParseError{TypeField: "stream_event", RawJSON: string(line), Err: err}
	}
	return &m, nil
}

func parseTaskUsage(m map[string]any) TaskUsage {
	var u TaskUsage
	if v, ok := m["total_tokens"].(float64); ok {
		u.TotalTokens = int(v)
	}
	if v, ok := m["tool_uses"].(float64); ok {
		u.ToolUses = int(v)
	}
	if v, ok := m["duration_ms"].(float64); ok {
		u.DurationMS = int64(v)
	}
	return u
}

func parseControlRequest(line []byte) (Message, error) {
	var msg ControlRequestMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, &MessageParseError{TypeField: "control_request", RawJSON: string(line), Err: err}
	}
	return &msg, nil
}

// controlResponseMessage is an internal type for CLI responses to SDK control requests.
// It is NOT yielded to callers — the query loop intercepts it.
type controlResponseMessage struct {
	Response json.RawMessage `json:"response"`
}

func (m *controlResponseMessage) MessageType() string { return "control_response" }

func parseControlResponse(line []byte) (Message, error) {
	var msg controlResponseMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, &MessageParseError{TypeField: "control_response", RawJSON: string(line), Err: err}
	}
	return &msg, nil
}

func parseContentBlocksFromSlice(raw []any) ([]ContentBlock, error) {
	blocks := make([]ContentBlock, 0, len(raw))
	for _, item := range raw {
		blockMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		blockType, _ := blockMap["type"].(string)
		switch blockType {
		case "text":
			text, _ := blockMap["text"].(string)
			blocks = append(blocks, &TextBlock{Type: "text", Text: text})
		case "thinking":
			thinking, _ := blockMap["thinking"].(string)
			signature, _ := blockMap["signature"].(string)
			blocks = append(blocks, &ThinkingBlock{Type: "thinking", Thinking: thinking, Signature: signature})
		case "tool_use":
			id, _ := blockMap["id"].(string)
			name, _ := blockMap["name"].(string)
			var input json.RawMessage
			if inp, ok := blockMap["input"]; ok {
				input, _ = json.Marshal(inp)
			}
			blocks = append(blocks, &ToolUseBlock{Type: "tool_use", ID: id, Name: name, Input: input})
		case "tool_result":
			toolUseID, _ := blockMap["tool_use_id"].(string)
			var content json.RawMessage
			if c, ok := blockMap["content"]; ok && c != nil {
				content, _ = json.Marshal(c)
			}
			var isError *bool
			if ie, ok := blockMap["is_error"].(bool); ok {
				isError = &ie
			}
			blocks = append(blocks, &ToolResultBlock{Type: "tool_result", ToolUseID: toolUseID, Content: content, IsError: isError})
		case "server_tool_use":
			id, _ := blockMap["id"].(string)
			name, _ := blockMap["name"].(string)
			var input json.RawMessage
			if inp, ok := blockMap["input"]; ok {
				input, _ = json.Marshal(inp)
			}
			blocks = append(blocks, &ServerToolUseBlock{Type: "server_tool_use", ID: id, Name: name, Input: input})
		case "server_tool_result":
			toolUseID, _ := blockMap["tool_use_id"].(string)
			var content json.RawMessage
			if c, ok := blockMap["content"]; ok && c != nil {
				content, _ = json.Marshal(c)
			}
			blocks = append(blocks, &ServerToolResultBlock{Type: "server_tool_result", ToolUseID: toolUseID, Content: content})
		}
	}
	return blocks, nil
}
