package mcp

import "encoding/json"

// ServerConfig is the common interface implemented by all MCP server configuration types.
type ServerConfig interface {
	ServerType() string
	// MarshalJSON produces CLI-compatible JSON for this server configuration.
	MarshalJSON() ([]byte, error)
}

// StdioServerConfig configures an MCP server launched as a subprocess communicating via stdio.
type StdioServerConfig struct {
	Name    string            `json:"name,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (c StdioServerConfig) ServerType() string { return "stdio" }

func (c StdioServerConfig) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type    string            `json:"type"`
		Command string            `json:"command"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
	}
	return json.Marshal(alias{Type: "stdio", Command: c.Command, Args: c.Args, Env: c.Env})
}

// SSEServerConfig configures an MCP server accessed via Server-Sent Events.
type SSEServerConfig struct {
	Name    string            `json:"name,omitempty"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (c SSEServerConfig) ServerType() string { return "sse" }

func (c SSEServerConfig) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers,omitempty"`
	}
	return json.Marshal(alias{Type: "sse", URL: c.URL, Headers: c.Headers})
}

// HTTPServerConfig configures an MCP server accessed via HTTP.
type HTTPServerConfig struct {
	Name    string            `json:"name,omitempty"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (c HTTPServerConfig) ServerType() string { return "http" }

func (c HTTPServerConfig) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers,omitempty"`
	}
	return json.Marshal(alias{Type: "http", URL: c.URL, Headers: c.Headers})
}

// SDKServerConfig wraps in-process Tool implementations to be served as an MCP server.
type SDKServerConfig struct {
	Name  string
	Tools []Tool
}

func (c SDKServerConfig) ServerType() string { return "sdk" }

func (c SDKServerConfig) MarshalJSON() ([]byte, error) {
	tools := make([]ToolSchema, len(c.Tools))
	for i, t := range c.Tools {
		tools[i] = ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}
	return json.Marshal(struct {
		Type  string       `json:"type"`
		Name  string       `json:"name,omitempty"`
		Tools []ToolSchema `json:"tools"`
	}{Type: "sdk", Name: c.Name, Tools: tools})
}

// ToolSchema describes a tool's name, description, and JSON Schema input.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// FindTool returns the Tool with the given name, or nil if not found.
func (c *SDKServerConfig) FindTool(name string) Tool {
	for _, t := range c.Tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// ListTools returns ToolSchema entries for all registered tools.
func (c *SDKServerConfig) ListTools() []ToolSchema {
	schemas := make([]ToolSchema, len(c.Tools))
	for i, t := range c.Tools {
		schemas[i] = ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}
	return schemas
}

// NewMCPServer creates an SDKServerConfig wrapping the given Tool implementations.
// The name identifies the server in the CLI MCP configuration. If empty, a default is used.
func NewMCPServer(name string, tools ...Tool) *SDKServerConfig {
	if name == "" {
		name = "sdk-tools"
	}
	return &SDKServerConfig{Name: name, Tools: tools}
}
