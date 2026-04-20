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
	tools := make([]toolSchema, len(c.Tools))
	for i, t := range c.Tools {
		tools[i] = toolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}
	return json.Marshal(struct {
		Type  string       `json:"type"`
		Name  string       `json:"name,omitempty"`
		Tools []toolSchema `json:"tools"`
	}{Type: "sdk", Name: c.Name, Tools: tools})
}

type toolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// NewMCPServer creates an SDKServerConfig wrapping the given Tool implementations.
func NewMCPServer(tools ...Tool) *SDKServerConfig {
	return &SDKServerConfig{Tools: tools}
}
