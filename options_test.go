package claude

import (
	"context"
	"testing"

	"github.com/johnayoung/claude-agent-sdk-go/hooks"
	"github.com/johnayoung/claude-agent-sdk-go/permission"
)

// stubTransport satisfies the Transporter interface for testing.
type stubTransport struct{}

func (s *stubTransport) Start(_ context.Context) error { return nil }
func (s *stubTransport) Send(_ []byte) error           { return nil }
func (s *stubTransport) Receive() ([]byte, error)      { return nil, nil }
func (s *stubTransport) Close() error                  { return nil }

func TestZeroValueDefaults(t *testing.T) {
	o := NewOptions(nil)
	if o.PermissionMode != PermissionModeDefault {
		t.Errorf("expected default permission mode, got %q", o.PermissionMode)
	}
}

func TestWithSystemPrompt(t *testing.T) {
	o := NewOptions([]Option{WithSystemPrompt("be concise")})
	if o.SystemPrompt != "be concise" {
		t.Errorf("unexpected SystemPrompt: %q", o.SystemPrompt)
	}
}

func TestWithMaxTurns(t *testing.T) {
	o := NewOptions([]Option{WithMaxTurns(5)})
	if o.MaxTurns != 5 {
		t.Errorf("unexpected MaxTurns: %d", o.MaxTurns)
	}
}

func TestWithCLIPath(t *testing.T) {
	o := NewOptions([]Option{WithCLIPath("/usr/local/bin/claude")})
	if o.CLIPath != "/usr/local/bin/claude" {
		t.Errorf("unexpected CLIPath: %q", o.CLIPath)
	}
}

func TestWithTransport(t *testing.T) {
	st := &stubTransport{}
	o := NewOptions([]Option{WithTransport(st)})
	if o.Transport != st {
		t.Error("expected custom transport to be set")
	}
}

func TestWithPermissionMode(t *testing.T) {
	o := NewOptions([]Option{WithPermissionMode(PermissionModeAcceptEdits)})
	if o.PermissionMode != PermissionModeAcceptEdits {
		t.Errorf("unexpected PermissionMode: %q", o.PermissionMode)
	}
}

func TestWithCanUseTool(t *testing.T) {
	called := false
	fn := func(toolName string, input map[string]any, ctx permission.ToolContext) (permission.Decision, error) {
		called = true
		return permission.Allow("allowed"), nil
	}
	o := NewOptions([]Option{WithCanUseTool(fn)})
	if o.CanUseTool == nil {
		t.Fatal("expected CanUseTool to be set")
	}
	d, err := o.CanUseTool("bash", nil, permission.ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("CanUseTool callback was not invoked")
	}
	if !d.Allowed() {
		t.Error("expected Allow decision")
	}
}

func TestWithHooks(t *testing.T) {
	h := hooks.New()
	o := NewOptions([]Option{WithHooks(h)})
	if o.Hooks == nil {
		t.Error("expected Hooks to be set")
	}
}

func TestWithMCPServers(t *testing.T) {
	s1 := MCPServerConfig{Name: "tools", Type: MCPServerTypeStdio, Command: "/bin/tools"}
	s2 := MCPServerConfig{Name: "api", Type: MCPServerTypeHTTP, URL: "http://localhost:8080"}
	o := NewOptions([]Option{WithMCPServers(s1, s2)})
	if len(o.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(o.MCPServers))
	}
}

func TestWithWorkingDir(t *testing.T) {
	o := NewOptions([]Option{WithWorkingDir("/tmp/work")})
	if o.WorkingDir != "/tmp/work" {
		t.Errorf("unexpected WorkingDir: %q", o.WorkingDir)
	}
}

func TestOptionsComposable(t *testing.T) {
	o := NewOptions([]Option{
		WithSystemPrompt("you are helpful"),
		WithMaxTurns(10),
		WithPermissionMode(PermissionModePlan),
		WithWorkingDir("/tmp"),
	})
	if o.SystemPrompt != "you are helpful" {
		t.Error("SystemPrompt not set")
	}
	if o.MaxTurns != 10 {
		t.Error("MaxTurns not set")
	}
	if o.PermissionMode != PermissionModePlan {
		t.Error("PermissionMode not set")
	}
	if o.WorkingDir != "/tmp" {
		t.Error("WorkingDir not set")
	}
}

func TestWithMCPServersAccumulates(t *testing.T) {
	o := NewOptions([]Option{
		WithMCPServers(MCPServerConfig{Name: "a", Type: MCPServerTypeStdio}),
		WithMCPServers(MCPServerConfig{Name: "b", Type: MCPServerTypeSSE}),
	})
	if len(o.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(o.MCPServers))
	}
}
