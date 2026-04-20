package agenttest

import (
	"fmt"
	"testing"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

// AssertTextContent asserts that msg is an AssistantMessage whose first TextBlock equals want.
func AssertTextContent(t testing.TB, msg agent.Message, want string) {
	t.Helper()
	am, ok := msg.(*agent.AssistantMessage)
	if !ok {
		t.Fatalf("AssertTextContent: expected *agent.AssistantMessage, got %T", msg)
	}
	for _, b := range am.Content {
		if tb, ok := b.(*agent.TextBlock); ok {
			if tb.Text != want {
				t.Fatalf("AssertTextContent: got %q, want %q", tb.Text, want)
			}
			return
		}
	}
	t.Fatalf("AssertTextContent: no TextBlock found in message")
}

// AssertToolUse asserts that msg is an AssistantMessage containing a ToolUseBlock with the given name.
func AssertToolUse(t testing.TB, msg agent.Message, name string) *agent.ToolUseBlock {
	t.Helper()
	am, ok := msg.(*agent.AssistantMessage)
	if !ok {
		t.Fatalf("AssertToolUse: expected *agent.AssistantMessage, got %T", msg)
	}
	for _, b := range am.Content {
		if tu, ok := b.(*agent.ToolUseBlock); ok && tu.Name == name {
			return tu
		}
	}
	t.Fatalf("AssertToolUse: no ToolUseBlock with name %q found", name)
	return nil
}

// AssertResult asserts that msg is a ResultMessage and returns it.
func AssertResult(t testing.TB, msg agent.Message) *agent.ResultMessage {
	t.Helper()
	rm, ok := msg.(*agent.ResultMessage)
	if !ok {
		t.Fatalf("AssertResult: expected *agent.ResultMessage, got %T", msg)
	}
	return rm
}

// AssertNoError fails t if err is non-nil.
func AssertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// CollectMessages drains messages from a channel into a slice, stopping on error.
// Use this with the slice returned by collecting from Query iterators in tests.
func CollectMessages(messages []agent.Message, errs []error) ([]agent.Message, error) {
	for _, err := range errs {
		if err != nil {
			return messages, err
		}
	}
	return messages, nil
}

// MustNewMockTransport calls NewMockTransport and panics on error.
// Intended for use in test setup where panicking is acceptable.
func MustNewMockTransport(messages ...agent.Message) *MockTransport {
	tr, err := NewMockTransport(messages...)
	if err != nil {
		panic(fmt.Sprintf("agenttest.MustNewMockTransport: %v", err))
	}
	return tr
}
