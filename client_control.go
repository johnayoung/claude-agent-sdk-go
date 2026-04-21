package claude

import "context"

// ServerInfoResponse contains information about the Claude CLI server.
type ServerInfoResponse struct {
	Commands             map[string]any   `json:"commands,omitempty"`
	OutputStyle          string           `json:"outputStyle,omitempty"`
	AvailableOutputStyles []string        `json:"availableOutputStyles,omitempty"`
}

type contextUsagePayload struct {
	Subtype string `json:"subtype"`
}

type serverInfoPayload struct {
	Subtype string `json:"subtype"`
}

type stopTaskPayload struct {
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
}

// GetContextUsage sends a control request to retrieve context window usage.
func (c *Client) GetContextUsage(ctx context.Context) (*ContextUsageResponse, error) {
	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return nil, err
	}
	defer tr.Close()

	reqID := generateControlRequestID("get_context_usage")
	payload := contextUsagePayload{Subtype: "get_context_usage"}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return nil, err
	}

	return awaitControlResponseTyped[ContextUsageResponse](ctx, tr, reqID, "get_context_usage")
}

// GetServerInfo sends a control request to retrieve server information.
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfoResponse, error) {
	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return nil, err
	}
	defer tr.Close()

	reqID := generateControlRequestID("get_server_info")
	payload := serverInfoPayload{Subtype: "get_server_info"}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return nil, err
	}

	return awaitControlResponseTyped[ServerInfoResponse](ctx, tr, reqID, "get_server_info")
}

// StopTask sends a control request to stop a running background task.
func (c *Client) StopTask(ctx context.Context, taskID string) error {
	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return err
	}
	defer tr.Close()

	reqID := generateControlRequestID("stop_task")
	payload := stopTaskPayload{Subtype: "stop_task", TaskID: taskID}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return err
	}

	_, err = awaitControlResponseTyped[struct{}](ctx, tr, reqID, "stop_task")
	return err
}
