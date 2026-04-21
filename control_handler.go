package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/johnayoung/claude-agent-sdk-go/hooks"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

// handleControlRequest processes an incoming control request and sends a response.
// Returns an error if sending the response fails.
// If cancelFn is non-nil and an interrupt is requested, it will be called.
func handleControlRequest(ctx context.Context, tr Transporter, o *Options, msg *ControlRequestMessage, cancelFn context.CancelFunc) error {
	var sub controlRequestSubtype
	if err := json.Unmarshal(msg.Request, &sub); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse request subtype: %v", err))
	}

	switch sub.Subtype {
	case "can_use_tool":
		return handlePermissionRequest(ctx, tr, o, msg)
	case "hook_callback":
		return handleHookCallback(ctx, tr, o, msg)
	case "interrupt":
		return handleInterrupt(tr, msg, cancelFn)
	case "set_permission_mode":
		return handleSetPermissionMode(tr, o, msg)
	case "rewind_files":
		return handleRewindFiles(tr, msg)
	case "mcp_message":
		return handleMcpMessage(ctx, tr, o, msg)
	case "mcp_reconnect":
		return handleMcpReconnect(tr, msg)
	case "mcp_toggle":
		return handleMcpToggle(tr, msg)
	case "stop_task":
		return handleStopTask(tr, msg)
	default:
		return sendControlSuccess(tr, msg.RequestID, nil)
	}
}

func handlePermissionRequest(ctx context.Context, tr Transporter, o *Options, msg *ControlRequestMessage) error {
	var req ControlPermissionRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse permission request: %v", err))
	}

	if o.CanUseTool == nil {
		resp := permissionAllowResponse{Behavior: "allow"}
		return sendControlSuccess(tr, msg.RequestID, resp)
	}

	toolCtx := permission.ToolContext{
		ToolUseID: req.ToolUseID,
		AgentID:   req.AgentID,
	}
	if req.PermissionSuggestions != nil {
		suggestions := make([]permission.Update, 0, len(req.PermissionSuggestions))
		for _, s := range req.PermissionSuggestions {
			if raw, err := json.Marshal(s); err == nil {
				var u permission.Update
				if json.Unmarshal(raw, &u) == nil {
					suggestions = append(suggestions, u)
				}
			}
		}
		toolCtx.Suggestions = suggestions
	}

	decision, err := o.CanUseTool(req.ToolName, req.Input, toolCtx)
	if err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("can_use_tool callback error: %v", err))
	}

	if decision.Allowed() {
		resp := permissionAllowResponse{Behavior: "allow"}
		if decision.UpdatedInput() != nil {
			resp.UpdatedInput = decision.UpdatedInput()
		}
		if decision.UpdatedPermissions() != nil {
			resp.UpdatedPermissions = decision.UpdatedPermissions()
		}
		return sendControlSuccess(tr, msg.RequestID, resp)
	}

	resp := permissionDenyResponse{
		Behavior:  "deny",
		Message:   decision.Reason(),
		Interrupt: decision.Interrupt(),
	}
	return sendControlSuccess(tr, msg.RequestID, resp)
}

func handleHookCallback(ctx context.Context, tr Transporter, o *Options, msg *ControlRequestMessage) error {
	if o.Hooks == nil {
		return sendControlSuccess(tr, msg.RequestID, nil)
	}

	var req ControlHookCallbackRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse hook_callback request: %v", err))
	}

	inputMap, ok := req.Input.(map[string]any)
	if !ok {
		return sendControlSuccess(tr, msg.RequestID, nil)
	}

	hookEventName, _ := inputMap["hook_event_name"].(string)

	sessionID, _ := inputMap["session_id"].(string)

	switch hookEventName {
	case "pre_tool_use":
		toolName, _ := inputMap["tool_name"].(string)
		toolInput, _ := inputMap["tool_input"].(map[string]any)
		out, err := o.Hooks.DispatchPreToolUse(ctx, &hooks.PreToolUseInput{
			ToolName:  toolName,
			ToolInput: toolInput,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "post_tool_use":
		toolName, _ := inputMap["tool_name"].(string)
		toolInput, _ := inputMap["tool_input"].(map[string]any)
		toolOutput, _ := inputMap["tool_response"].(string)
		isError, _ := inputMap["is_error"].(bool)
		out, err := o.Hooks.DispatchPostToolUse(ctx, &hooks.PostToolUseInput{
			ToolName:   toolName,
			ToolInput:  toolInput,
			ToolOutput: toolOutput,
			IsError:    isError,
			SessionID:  sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "post_tool_use_failure":
		toolName, _ := inputMap["tool_name"].(string)
		toolInput, _ := inputMap["tool_input"].(map[string]any)
		toolOutput, _ := inputMap["tool_response"].(string)
		errMsg, _ := inputMap["error"].(string)
		out, err := o.Hooks.DispatchPostToolUseFailure(ctx, &hooks.PostToolUseFailureInput{
			ToolName:   toolName,
			ToolInput:  toolInput,
			ToolOutput: toolOutput,
			Error:      errMsg,
			SessionID:  sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "model_response":
		response, _ := inputMap["response"].(string)
		out, err := o.Hooks.DispatchModelResponse(ctx, &hooks.ModelResponseInput{
			Response:  response,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "notification_arrived":
		title, _ := inputMap["title"].(string)
		message, _ := inputMap["message"].(string)
		out, err := o.Hooks.DispatchNotificationArrived(ctx, &hooks.NotificationArrivedInput{
			Title:     title,
			Message:   message,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "stop":
		reason, _ := inputMap["reason"].(string)
		out, err := o.Hooks.DispatchStop(ctx, &hooks.StopInput{
			Reason:    reason,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "subagent_started":
		agentID, _ := inputMap["agent_id"].(string)
		out, err := o.Hooks.DispatchSubagentStarted(ctx, &hooks.SubagentStartedInput{
			AgentID:   agentID,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "subagent_stopped":
		agentID, _ := inputMap["agent_id"].(string)
		result, _ := inputMap["result"].(string)
		out, err := o.Hooks.DispatchSubagentStopped(ctx, &hooks.SubagentStoppedInput{
			AgentID:   agentID,
			SessionID: sessionID,
			Result:    result,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "session_started":
		out, err := o.Hooks.DispatchSessionStarted(ctx, &hooks.SessionStartedInput{
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "session_stopped":
		out, err := o.Hooks.DispatchSessionStopped(ctx, &hooks.SessionStoppedInput{
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "user_prompt_submit":
		prompt, _ := inputMap["prompt"].(string)
		out, err := o.Hooks.DispatchUserPromptSubmit(ctx, &hooks.UserPromptSubmitInput{
			Prompt:    prompt,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "permission_request":
		toolName, _ := inputMap["tool_name"].(string)
		toolInput, _ := inputMap["tool_input"].(map[string]any)
		out, err := o.Hooks.DispatchPermissionRequest(ctx, &hooks.PermissionRequestInput{
			ToolName:  toolName,
			ToolInput: toolInput,
			SessionID: sessionID,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	case "pre_compact":
		var messageCount int
		if v, ok := inputMap["message_count"].(float64); ok {
			messageCount = int(v)
		}
		out, err := o.Hooks.DispatchPreCompact(ctx, &hooks.PreCompactInput{
			SessionID:    sessionID,
			MessageCount: messageCount,
		})
		if err != nil {
			return sendControlError(tr, msg.RequestID, err.Error())
		}
		return sendControlSuccess(tr, msg.RequestID, out)

	default:
		return sendControlSuccess(tr, msg.RequestID, nil)
	}
}

func handleInterrupt(tr Transporter, msg *ControlRequestMessage, cancelFn context.CancelFunc) error {
	if cancelFn != nil {
		cancelFn()
	}
	return sendControlSuccess(tr, msg.RequestID, nil)
}

func handleSetPermissionMode(tr Transporter, o *Options, msg *ControlRequestMessage) error {
	var req ControlSetPermissionModeRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse set_permission_mode: %v", err))
	}
	o.PermissionMode = PermissionMode(req.Mode)
	return sendControlSuccess(tr, msg.RequestID, nil)
}

func handleRewindFiles(tr Transporter, msg *ControlRequestMessage) error {
	var req ControlRewindFilesRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse rewind_files request: %v", err))
	}
	return sendControlSuccess(tr, msg.RequestID, nil)
}

func handleMcpReconnect(tr Transporter, msg *ControlRequestMessage) error {
	var req ControlMcpReconnectRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse mcp_reconnect request: %v", err))
	}
	return sendControlSuccess(tr, msg.RequestID, nil)
}

func handleMcpToggle(tr Transporter, msg *ControlRequestMessage) error {
	var req ControlMcpToggleRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse mcp_toggle request: %v", err))
	}
	return sendControlSuccess(tr, msg.RequestID, nil)
}

func handleStopTask(tr Transporter, msg *ControlRequestMessage) error {
	var req ControlStopTaskRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse stop_task request: %v", err))
	}
	return sendControlSuccess(tr, msg.RequestID, nil)
}

func sendControlSuccess(tr Transporter, requestID string, response any) error {
	resp := SDKControlResponse{
		Type: "control_response",
		Response: ControlResponse{
			Subtype:   "success",
			RequestID: requestID,
			Response:  response,
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return tr.Send(data)
}

func handleMcpMessage(ctx context.Context, tr Transporter, o *Options, msg *ControlRequestMessage) error {
	var req ControlMcpMessageRequest
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		return sendControlError(tr, msg.RequestID, fmt.Sprintf("failed to parse mcp_message: %v", err))
	}

	server := o.SDKMCPServers[req.ServerName]
	if server == nil {
		resp := map[string]any{"mcp_response": jsonrpcError(req.Message, -32600, "unknown SDK MCP server: "+req.ServerName)}
		return sendControlSuccess(tr, msg.RequestID, resp)
	}

	mcpMsg, ok := req.Message.(map[string]any)
	if !ok {
		resp := map[string]any{"mcp_response": jsonrpcError(req.Message, -32600, "invalid MCP message")}
		return sendControlSuccess(tr, msg.RequestID, resp)
	}

	method, _ := mcpMsg["method"].(string)
	id := mcpMsg["id"]
	params, _ := mcpMsg["params"].(map[string]any)

	var result any
	var rpcErr *jsonrpcErrorPayload

	switch method {
	case "initialize":
		result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": server.Name, "version": "1.0.0"},
		}
	case "notifications/initialized":
		result = map[string]any{}
	case "tools/list":
		tools := server.ListTools()
		toolEntries := make([]map[string]any, len(tools))
		for i, t := range tools {
			toolEntries[i] = map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": json.RawMessage(t.InputSchema),
			}
		}
		result = map[string]any{"tools": toolEntries}
	case "tools/call":
		name, _ := params["name"].(string)
		arguments, _ := params["arguments"].(map[string]any)
		tool := server.FindTool(name)
		if tool == nil {
			rpcErr = &jsonrpcErrorPayload{Code: -32602, Message: "unknown tool: " + name}
		} else {
			output, err := tool.Run(ctx, arguments)
			if err != nil {
				result = map[string]any{
					"content": []map[string]any{{"type": "text", "text": err.Error()}},
					"isError": true,
				}
			} else {
				result = map[string]any{
					"content": []map[string]any{{"type": "text", "text": string(output)}},
				}
			}
		}
	default:
		rpcErr = &jsonrpcErrorPayload{Code: -32601, Message: "method not found: " + method}
	}

	var mcpResponse any
	if rpcErr != nil {
		mcpResponse = map[string]any{"jsonrpc": "2.0", "id": id, "error": rpcErr}
	} else {
		mcpResponse = map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
	}

	resp := map[string]any{"mcp_response": mcpResponse}
	return sendControlSuccess(tr, msg.RequestID, resp)
}

type jsonrpcErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func jsonrpcError(msg any, code int, message string) map[string]any {
	var id any
	if m, ok := msg.(map[string]any); ok {
		id = m["id"]
	}
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   &jsonrpcErrorPayload{Code: code, Message: message},
	}
}

func sendControlError(tr Transporter, requestID string, errMsg string) error {
	resp := SDKControlResponse{
		Type: "control_response",
		Response: ControlErrorResponse{
			Subtype:   "error",
			RequestID: requestID,
			Error:     errMsg,
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return tr.Send(data)
}
