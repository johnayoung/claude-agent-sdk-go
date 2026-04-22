package claude_test

import (
	"testing"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest/sessionstoretest"
)

// TestInMemorySessionStore_Conformance exercises the in-tree store against
// the shared contract so it stays in lockstep with the third-party adapters.
func TestInMemorySessionStore_Conformance(t *testing.T) {
	sessionstoretest.Run(t, func(t *testing.T) claude.SessionStore {
		return claude.NewInMemorySessionStore()
	})
}
