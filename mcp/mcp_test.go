package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/johnayoung/claude-agent-sdk-go/mcp"
)

type echoTool struct{}

func (e echoTool) Name() string        { return "echo" }
func (e echoTool) Description() string { return "Echoes the input back" }
func (e echoTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`)
}
func (e echoTool) Run(_ context.Context, input map[string]any) (json.RawMessage, error) {
	return json.Marshal(input)
}

func TestToolInterface(t *testing.T) {
	var _ mcp.Tool = echoTool{}
	tool := echoTool{}
	if tool.Name() != "echo" {
		t.Errorf("expected Name() == echo, got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
	schema := tool.InputSchema()
	if !json.Valid(schema) {
		t.Error("InputSchema() must return valid JSON")
	}
	out, err := tool.Run(context.Background(), map[string]any{"message": "hi"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !json.Valid(out) {
		t.Error("Run() output must be valid JSON")
	}
}

func TestNewMCPServer(t *testing.T) {
	srv := mcp.NewMCPServer(echoTool{})
	if srv == nil {
		t.Fatal("NewMCPServer returned nil")
	}
	if srv.ServerType() != "sdk" {
		t.Errorf("expected ServerType sdk, got %q", srv.ServerType())
	}
	if len(srv.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(srv.Tools))
	}
}

func TestSDKServerConfigMarshal(t *testing.T) {
	srv := mcp.NewMCPServer(echoTool{})
	data, err := json.Marshal(srv)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if m["type"] != "sdk" {
		t.Errorf("expected type=sdk, got %v", m["type"])
	}
	tools, ok := m["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Errorf("expected 1 tool in JSON, got %v", m["tools"])
	}
}

func TestStdioServerConfig(t *testing.T) {
	cfg := mcp.StdioServerConfig{Command: "my-mcp-server", Args: []string{"--port", "8080"}}
	if cfg.ServerType() != "stdio" {
		t.Errorf("expected stdio, got %q", cfg.ServerType())
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	if m["type"] != "stdio" {
		t.Errorf("expected type=stdio, got %v", m["type"])
	}
	if m["command"] != "my-mcp-server" {
		t.Errorf("unexpected command: %v", m["command"])
	}
}

func TestSSEServerConfig(t *testing.T) {
	cfg := mcp.SSEServerConfig{URL: "http://localhost:3000/sse"}
	if cfg.ServerType() != "sse" {
		t.Errorf("expected sse, got %q", cfg.ServerType())
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	if m["type"] != "sse" {
		t.Errorf("expected type=sse, got %v", m["type"])
	}
	if m["url"] != "http://localhost:3000/sse" {
		t.Errorf("unexpected url: %v", m["url"])
	}
}

func TestHTTPServerConfig(t *testing.T) {
	cfg := mcp.HTTPServerConfig{URL: "http://localhost:3000/mcp"}
	if cfg.ServerType() != "http" {
		t.Errorf("expected http, got %q", cfg.ServerType())
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	if m["type"] != "http" {
		t.Errorf("expected type=http, got %v", m["type"])
	}
}

func TestServerConfigInterface(t *testing.T) {
	configs := []mcp.ServerConfig{
		mcp.StdioServerConfig{Command: "cmd"},
		mcp.SSEServerConfig{URL: "http://host/sse"},
		mcp.HTTPServerConfig{URL: "http://host/mcp"},
		mcp.NewMCPServer(),
	}
	types := []string{"stdio", "sse", "http", "sdk"}
	for i, cfg := range configs {
		if cfg.ServerType() != types[i] {
			t.Errorf("config[%d]: expected type %q, got %q", i, types[i], cfg.ServerType())
		}
		data, err := cfg.MarshalJSON()
		if err != nil {
			t.Errorf("config[%d] MarshalJSON error: %v", i, err)
		}
		if !json.Valid(data) {
			t.Errorf("config[%d] produced invalid JSON", i)
		}
	}
}
