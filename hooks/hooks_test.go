package hooks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/johnayoung/claude-agent-sdk-go/hooks"
)

func TestAllEventsHaveTypedStructs(t *testing.T) {
	// Compile-time check: instantiate all Input/Output types
	_ = &hooks.PreToolUseInput{}
	_ = &hooks.PreToolUseOutput{}
	_ = &hooks.PostToolUseInput{}
	_ = &hooks.PostToolUseOutput{}
	_ = &hooks.ModelResponseInput{}
	_ = &hooks.ModelResponseOutput{}
	_ = &hooks.NotificationArrivedInput{}
	_ = &hooks.NotificationArrivedOutput{}
	_ = &hooks.StopInput{}
	_ = &hooks.StopOutput{}
	_ = &hooks.SubagentStartedInput{}
	_ = &hooks.SubagentStartedOutput{}
	_ = &hooks.SubagentStoppedInput{}
	_ = &hooks.SubagentStoppedOutput{}
	_ = &hooks.SessionStartedInput{}
	_ = &hooks.SessionStartedOutput{}
	_ = &hooks.SessionStoppedInput{}
	_ = &hooks.SessionStoppedOutput{}
	_ = &hooks.ErrorInput{}
	_ = &hooks.ErrorOutput{}
}

func TestHookConstants(t *testing.T) {
	events := []hooks.HookEvent{
		hooks.EventPreToolUse,
		hooks.EventPostToolUse,
		hooks.EventModelResponse,
		hooks.EventNotificationArrived,
		hooks.EventStop,
		hooks.EventSubagentStarted,
		hooks.EventSubagentStopped,
		hooks.EventSessionStarted,
		hooks.EventSessionStopped,
		hooks.EventError,
	}
	if len(events) != 10 {
		t.Fatalf("expected 10 events, got %d", len(events))
	}
}

func TestDispatchPreToolUse_ToolInputModification(t *testing.T) {
	h := hooks.New()
	h.OnPreToolUse("*", func(ctx context.Context, in *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		return &hooks.PreToolUseOutput{
			ToolInput: map[string]any{"modified": true},
		}, nil
	})

	out, err := h.DispatchPreToolUse(context.Background(), &hooks.PreToolUseInput{
		ToolName:  "Bash",
		ToolInput: map[string]any{"cmd": "ls"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ToolInput["modified"] != true {
		t.Fatal("expected ToolInput to be modified")
	}
}

func TestDispatchPreToolUse_Block(t *testing.T) {
	h := hooks.New()
	h.OnPreToolUse("Bash", func(ctx context.Context, in *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		return &hooks.PreToolUseOutput{Block: true, Reason: "not allowed"}, nil
	})

	out, err := h.DispatchPreToolUse(context.Background(), &hooks.PreToolUseInput{ToolName: "Bash"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Block {
		t.Fatal("expected Block=true")
	}
	if out.Reason != "not allowed" {
		t.Fatalf("unexpected reason: %q", out.Reason)
	}
}

func TestDispatchPreToolUse_GlobMatching(t *testing.T) {
	h := hooks.New()
	called := false
	h.OnPreToolUse("Write*", func(ctx context.Context, in *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		called = true
		return &hooks.PreToolUseOutput{}, nil
	})

	// "Bash" should NOT match "Write*"
	_, err := h.DispatchPreToolUse(context.Background(), &hooks.PreToolUseInput{ToolName: "Bash"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("handler should not have been called for non-matching tool")
	}

	// "WriteFile" should match "Write*"
	_, err = h.DispatchPreToolUse(context.Background(), &hooks.PreToolUseInput{ToolName: "WriteFile"})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("handler should have been called for matching tool")
	}
}

func TestDispatchPreToolUse_ChainedModification(t *testing.T) {
	h := hooks.New()
	h.OnPreToolUse("*", func(ctx context.Context, in *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		return &hooks.PreToolUseOutput{ToolInput: map[string]any{"step": 1}}, nil
	})
	h.OnPreToolUse("*", func(ctx context.Context, in *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
		if in.ToolInput["step"] != 1 {
			t.Error("second handler did not receive modified input from first")
		}
		return &hooks.PreToolUseOutput{ToolInput: map[string]any{"step": 2}}, nil
	})

	out, err := h.DispatchPreToolUse(context.Background(), &hooks.PreToolUseInput{ToolName: "Bash"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ToolInput["step"] != 2 {
		t.Fatalf("expected final step=2, got %v", out.ToolInput["step"])
	}
}

func TestDispatchPostToolUse_GlobMatching(t *testing.T) {
	h := hooks.New()
	count := 0
	h.OnPostToolUse("Bash", func(ctx context.Context, in *hooks.PostToolUseInput) (*hooks.PostToolUseOutput, error) {
		count++
		return &hooks.PostToolUseOutput{}, nil
	})

	h.DispatchPostToolUse(context.Background(), &hooks.PostToolUseInput{ToolName: "Read"})
	if count != 0 {
		t.Fatal("handler called for non-matching tool")
	}
	h.DispatchPostToolUse(context.Background(), &hooks.PostToolUseInput{ToolName: "Bash"})
	if count != 1 {
		t.Fatal("handler not called for matching tool")
	}
}

func TestDispatchModelResponse(t *testing.T) {
	h := hooks.New()
	got := ""
	h.OnModelResponse(func(ctx context.Context, in *hooks.ModelResponseInput) (*hooks.ModelResponseOutput, error) {
		got = in.Response
		return &hooks.ModelResponseOutput{}, nil
	})
	h.DispatchModelResponse(context.Background(), &hooks.ModelResponseInput{Response: "hello"})
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

func TestDispatchError_PropagatesError(t *testing.T) {
	h := hooks.New()
	sentinel := errors.New("handler error")
	h.OnError(func(ctx context.Context, in *hooks.ErrorInput) (*hooks.ErrorOutput, error) {
		return nil, sentinel
	})
	_, err := h.DispatchError(context.Background(), &hooks.ErrorInput{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestAllDispatchMethods_NoHandlers(t *testing.T) {
	h := hooks.New()
	ctx := context.Background()

	if _, err := h.DispatchPreToolUse(ctx, &hooks.PreToolUseInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchPostToolUse(ctx, &hooks.PostToolUseInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchModelResponse(ctx, &hooks.ModelResponseInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchNotificationArrived(ctx, &hooks.NotificationArrivedInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchStop(ctx, &hooks.StopInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchSubagentStarted(ctx, &hooks.SubagentStartedInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchSubagentStopped(ctx, &hooks.SubagentStoppedInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchSessionStarted(ctx, &hooks.SessionStartedInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchSessionStopped(ctx, &hooks.SessionStoppedInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DispatchError(ctx, &hooks.ErrorInput{}); err != nil {
		t.Fatal(err)
	}
}
