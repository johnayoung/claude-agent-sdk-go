package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestFilePathToSessionKey_WithProjectsDir(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		projectsDir string
		wantKey     SessionKey
	}{
		{
			name:        "main transcript",
			filePath:    "/home/user/.claude/projects/abc123/session-xyz.jsonl",
			projectsDir: "/home/user/.claude/projects",
			wantKey:     SessionKey{ProjectKey: "abc123", SessionID: "session-xyz"},
		},
		{
			name:        "subagent transcript",
			filePath:    "/home/user/.claude/projects/abc123/session-xyz/subagents/agent-001.jsonl",
			projectsDir: "/home/user/.claude/projects",
			wantKey:     SessionKey{ProjectKey: "abc123", SessionID: "session-xyz", Subpath: "subagents/agent-001"},
		},
		{
			name:        "deeply nested subagent",
			filePath:    "/home/user/.claude/projects/proj/sess/subagents/deep/agent-abc.jsonl",
			projectsDir: "/home/user/.claude/projects",
			wantKey:     SessionKey{ProjectKey: "proj", SessionID: "sess", Subpath: "subagents/deep/agent-abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := filePathToSessionKey(tt.filePath, tt.projectsDir)
			if key == nil {
				t.Fatal("expected non-nil key")
			}
			if *key != tt.wantKey {
				t.Errorf("got %+v, want %+v", *key, tt.wantKey)
			}
		})
	}
}

func TestFilePathToSessionKey_FallbackHash(t *testing.T) {
	key := filePathToSessionKey("/some/random/path/session.jsonl", "")
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.SessionID == "" {
		t.Error("expected non-empty SessionID")
	}
	if key.ProjectKey != "" {
		t.Error("expected empty ProjectKey for fallback")
	}
	// Verify deterministic
	key2 := filePathToSessionKey("/some/random/path/session.jsonl", "")
	if key.SessionID != key2.SessionID {
		t.Error("expected deterministic key")
	}
}

func TestFilePathToSessionKey_OutsideProjectsDir(t *testing.T) {
	key := filePathToSessionKey("/other/path/session.jsonl", "/home/user/.claude/projects")
	if key == nil {
		t.Fatal("expected non-nil key (should fall back to hash)")
	}
	// Should fall back to hash-based since path is outside projectsDir
	if key.ProjectKey != "" {
		t.Error("expected empty ProjectKey for fallback path")
	}
}

func TestParseTranscriptMirror(t *testing.T) {
	line := []byte(`{"type":"transcript_mirror","filePath":"/tmp/test.jsonl","entries":[{"type":"user","uuid":"abc"}]}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mirror, ok := msg.(*transcriptMirrorMessage)
	if !ok {
		t.Fatalf("expected *transcriptMirrorMessage, got %T", msg)
	}
	if mirror.FilePath != "/tmp/test.jsonl" {
		t.Errorf("got filePath %q, want %q", mirror.FilePath, "/tmp/test.jsonl")
	}
	if len(mirror.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(mirror.Entries))
	}
}

func TestHandleTranscriptMirror_Success(t *testing.T) {
	store := NewInMemorySessionStore()
	entries := []json.RawMessage{
		json.RawMessage(`{"type":"user","uuid":"u1","timestamp":"2025-01-01T00:00:00Z"}`),
		json.RawMessage(`{"type":"assistant","uuid":"a1","timestamp":"2025-01-01T00:00:01Z"}`),
	}
	msg := &transcriptMirrorMessage{
		FilePath: "/home/user/.claude/projects/proj1/sess1.jsonl",
		Entries:  entries,
	}

	errMsg := handleTranscriptMirror(context.Background(), store, "/home/user/.claude/projects", msg)
	if errMsg != nil {
		t.Fatalf("unexpected error message: %v", errMsg.Error)
	}

	key := SessionKey{ProjectKey: "proj1", SessionID: "sess1"}
	loaded, err := store.Load(context.Background(), key)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("got %d entries, want 2", len(loaded))
	}
	if loaded[0].UUID != "u1" {
		t.Errorf("got uuid %q, want %q", loaded[0].UUID, "u1")
	}
}

func TestHandleTranscriptMirror_EmptyEntries(t *testing.T) {
	store := NewInMemorySessionStore()
	msg := &transcriptMirrorMessage{
		FilePath: "/tmp/test.jsonl",
		Entries:  nil,
	}
	errMsg := handleTranscriptMirror(context.Background(), store, "", msg)
	if errMsg != nil {
		t.Fatal("expected nil for empty entries")
	}
}

func TestHandleTranscriptMirror_StoreError(t *testing.T) {
	store := &failingSessionStore{}
	entries := []json.RawMessage{
		json.RawMessage(`{"type":"user","uuid":"u1"}`),
	}
	msg := &transcriptMirrorMessage{
		FilePath: "/tmp/test.jsonl",
		Entries:  entries,
	}

	errMsg := handleTranscriptMirror(context.Background(), store, "", msg)
	if errMsg == nil {
		t.Fatal("expected error message")
	}
	if errMsg.Error == "" {
		t.Error("expected non-empty error string")
	}
	if errMsg.Key == nil {
		t.Error("expected non-nil key in error message")
	}
}

func TestTranscriptMirrorNotYieldedToCallers(t *testing.T) {
	line := []byte(`{"type":"transcript_mirror","filePath":"/tmp/test.jsonl","entries":[{"type":"user","uuid":"u1"}]}`)
	msg, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.MessageType() != "transcript_mirror" {
		t.Fatalf("expected transcript_mirror, got %s", msg.MessageType())
	}
	// Verify it's the unexported type - callers cannot type-assert to it
	if _, ok := msg.(*transcriptMirrorMessage); !ok {
		t.Fatal("expected *transcriptMirrorMessage")
	}
}

type failingSessionStore struct{}

func (s *failingSessionStore) Append(_ context.Context, _ SessionKey, _ []SessionStoreEntry) error {
	return fmt.Errorf("disk full")
}
func (s *failingSessionStore) Load(_ context.Context, _ SessionKey) ([]SessionStoreEntry, error) {
	return nil, nil
}
func (s *failingSessionStore) Delete(_ context.Context, _ SessionKey) error { return nil }
func (s *failingSessionStore) ListSessions(_ context.Context, _ string) ([]SessionStoreListEntry, error) {
	return nil, nil
}
func (s *failingSessionStore) ListSubkeys(_ context.Context, _ SessionKey) ([]string, error) {
	return nil, nil
}
