package claude

import "context"

type mcpReconnectPayload struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
}

type mcpTogglePayload struct {
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
	Enabled    bool   `json:"enabled"`
}

type mcpStatusPayload struct {
	Subtype string `json:"subtype"`
}

// ReconnectMCPServer sends a control request to reconnect the named MCP server.
func (c *Client) ReconnectMCPServer(ctx context.Context, serverName string) error {
	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return err
	}
	defer tr.Close()

	reqID := generateControlRequestID("mcp_reconnect")
	payload := mcpReconnectPayload{Subtype: "mcp_reconnect", ServerName: serverName}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return err
	}

	_, err = awaitControlResponseTyped[struct{}](ctx, tr, reqID, "mcp_reconnect")
	return err
}

// ToggleMCPServer sends a control request to enable or disable the named MCP server.
func (c *Client) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return err
	}
	defer tr.Close()

	reqID := generateControlRequestID("mcp_toggle")
	payload := mcpTogglePayload{Subtype: "mcp_toggle", ServerName: serverName, Enabled: enabled}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return err
	}

	_, err = awaitControlResponseTyped[struct{}](ctx, tr, reqID, "mcp_toggle")
	return err
}

// GetMCPStatus sends a control request to retrieve the status of all MCP servers.
func (c *Client) GetMCPStatus(ctx context.Context) (*McpStatusResponse, error) {
	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return nil, err
	}
	defer tr.Close()

	reqID := generateControlRequestID("get_mcp_status")
	payload := mcpStatusPayload{Subtype: "get_mcp_status"}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return nil, err
	}

	return awaitControlResponseTyped[McpStatusResponse](ctx, tr, reqID, "get_mcp_status")
}
