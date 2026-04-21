//go:build e2e

package claude_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

const (
	e2eModel      = "haiku"
	e2eMaxTurns   = 3
	e2eTimeout    = 120 * time.Second
)

func skipIfNoCLI(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found on PATH; skipping e2e test")
	}
}

func e2eContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), e2eTimeout)
}

func baseOpts() []claude.Option {
	return []claude.Option{
		claude.WithModel(e2eModel),
		claude.WithMaxTurns(e2eMaxTurns),
		claude.WithPermissionMode(claude.PermissionModeBypassPermissions),
	}
}

func collectMessages(t *testing.T, ctx context.Context, prompt string, opts ...claude.Option) []claude.Message {
	t.Helper()
	all := append(baseOpts(), opts...)
	t.Logf("sending prompt: %q", prompt)
	var msgs []claude.Message
	for msg, err := range claude.Query(ctx, prompt, all...) {
		if err != nil {
			t.Fatalf("Query error: %v", err)
		}
		t.Logf("received: %s", msg.MessageType())
		msgs = append(msgs, msg)
	}
	t.Logf("done: %d messages received", len(msgs))
	return msgs
}

func findResult(msgs []claude.Message) *claude.ResultMessage {
	for _, m := range msgs {
		if r, ok := m.(*claude.ResultMessage); ok {
			return r
		}
	}
	return nil
}

func findAssistant(msgs []claude.Message) *claude.AssistantMessage {
	for _, m := range msgs {
		if a, ok := m.(*claude.AssistantMessage); ok {
			return a
		}
	}
	return nil
}

func findSystem(msgs []claude.Message, subtype string) *claude.SystemMessage {
	for _, m := range msgs {
		if s, ok := m.(*claude.SystemMessage); ok && s.Subtype == subtype {
			return s
		}
	}
	return nil
}
