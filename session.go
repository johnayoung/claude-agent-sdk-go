package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// SessionKey identifies a session in a SessionStore.
type SessionKey struct {
	ProjectKey string `json:"project_key"`
	SessionID  string `json:"session_id"`
	Subpath    string `json:"subpath,omitempty"`
}

// SessionStoreEntry represents a single JSONL transcript line observed by a
// SessionStore adapter.
//
// The concrete shape is the CLI's on-disk transcript format — a large
// discriminated union whose exhaustive schema is internal to the CLI. This
// type is a minimal structural supertype: Type/UUID/Timestamp are surfaced
// for convenience, and every other field the CLI emits is preserved verbatim
// in Extra. Adapters must round-trip entries without loss — the
// json.Marshal → storage → json.Unmarshal cycle is the only required
// invariant.
//
// This mirrors the Python SDK's SessionStoreEntry TypedDict with pass-through
// extras.
type SessionStoreEntry struct {
	Type      string
	UUID      string
	Timestamp string

	// Extra holds every top-level key in the wire payload other than type,
	// uuid, and timestamp, preserved as raw JSON so adapters can round-trip
	// CLI-emitted content (message bodies, tool calls, etc.) without knowing
	// the evolving CLI schema. Nil for entries constructed in code with only
	// the documented fields.
	Extra map[string]json.RawMessage
}

// MarshalJSON emits a flat JSON object that merges Type/UUID/Timestamp with
// Extra. Type is always emitted; UUID and Timestamp use omitempty semantics.
// Keys in Extra never shadow the documented fields.
func (e SessionStoreEntry) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage, len(e.Extra)+3)
	for k, v := range e.Extra {
		if k == "type" || k == "uuid" || k == "timestamp" {
			// Documented fields always take precedence over Extra duplicates.
			continue
		}
		m[k] = v
	}
	typeRaw, err := json.Marshal(e.Type)
	if err != nil {
		return nil, err
	}
	m["type"] = typeRaw
	if e.UUID != "" {
		uuidRaw, err := json.Marshal(e.UUID)
		if err != nil {
			return nil, err
		}
		m["uuid"] = uuidRaw
	}
	if e.Timestamp != "" {
		tsRaw, err := json.Marshal(e.Timestamp)
		if err != nil {
			return nil, err
		}
		m["timestamp"] = tsRaw
	}
	return json.Marshal(m)
}

// UnmarshalJSON extracts the documented fields into Type/UUID/Timestamp and
// preserves everything else in Extra. Returns an error only if the payload
// is not a JSON object.
func (e *SessionStoreEntry) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if raw, ok := m["type"]; ok {
		if err := json.Unmarshal(raw, &e.Type); err != nil {
			return fmt.Errorf("SessionStoreEntry.type: %w", err)
		}
		delete(m, "type")
	} else {
		e.Type = ""
	}
	if raw, ok := m["uuid"]; ok {
		if err := json.Unmarshal(raw, &e.UUID); err != nil {
			return fmt.Errorf("SessionStoreEntry.uuid: %w", err)
		}
		delete(m, "uuid")
	} else {
		e.UUID = ""
	}
	if raw, ok := m["timestamp"]; ok {
		if err := json.Unmarshal(raw, &e.Timestamp); err != nil {
			return fmt.Errorf("SessionStoreEntry.timestamp: %w", err)
		}
		delete(m, "timestamp")
	} else {
		e.Timestamp = ""
	}
	if len(m) == 0 {
		e.Extra = nil
	} else {
		e.Extra = m
	}
	return nil
}

// SessionStoreListEntry is a summary of a stored session.
type SessionStoreListEntry struct {
	SessionID string `json:"session_id"`
	Mtime     int64  `json:"mtime"`
}

// SessionStore is the interface for custom session persistence backends.
type SessionStore interface {
	Append(ctx context.Context, key SessionKey, entries []SessionStoreEntry) error
	Load(ctx context.Context, key SessionKey) ([]SessionStoreEntry, error)
	Delete(ctx context.Context, key SessionKey) error
	ListSessions(ctx context.Context, projectKey string) ([]SessionStoreListEntry, error)
	ListSubkeys(ctx context.Context, key SessionKey) ([]string, error)
}

// InMemorySessionStore is a SessionStore backed by an in-process map.
type InMemorySessionStore struct {
	mu sync.RWMutex
	// data maps the composite key (project/session[/subpath]) to its entries.
	data map[string][]SessionStoreEntry
	// mtimes tracks the last-write time per main session (project/session).
	// Only main-transcript appends update mtime — ListSessions filters to main.
	mtimes map[string]int64
	// clock is strictly monotonic within this instance so same-ms appends
	// stay distinguishable.
	clockMu sync.Mutex
	lastMs  int64
}

// NewInMemorySessionStore creates an empty in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		data:   make(map[string][]SessionStoreEntry),
		mtimes: make(map[string]int64),
	}
}

func sessionStoreKey(key SessionKey) string {
	k := key.ProjectKey + "/" + key.SessionID
	if key.Subpath != "" {
		k += "/" + key.Subpath
	}
	return k
}

func mainSessionKey(projectKey, sessionID string) string {
	return projectKey + "/" + sessionID
}

func (s *InMemorySessionStore) monotonicNowMs() int64 {
	s.clockMu.Lock()
	defer s.clockMu.Unlock()
	now := time.Now().UnixMilli()
	if now <= s.lastMs {
		now = s.lastMs + 1
	}
	s.lastMs = now
	return now
}

func (s *InMemorySessionStore) Append(_ context.Context, key SessionKey, entries []SessionStoreEntry) error {
	if len(entries) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := sessionStoreKey(key)
	s.data[k] = append(s.data[k], entries...)
	if key.Subpath == "" {
		// Only main-transcript appends bump the session index.
		s.mtimes[mainSessionKey(key.ProjectKey, key.SessionID)] = s.monotonicNowMs()
	}
	return nil
}

func (s *InMemorySessionStore) Load(_ context.Context, key SessionKey) ([]SessionStoreEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, ok := s.data[sessionStoreKey(key)]
	if !ok {
		return nil, nil
	}
	out := make([]SessionStoreEntry, len(entries))
	copy(out, entries)
	return out, nil
}

// Delete removes a session (or a specific subpath). Deleting the main
// session (empty Subpath) cascades to every subpath under
// (ProjectKey, SessionID); deleting a specific subpath is exact-key only.
func (s *InMemorySessionStore) Delete(_ context.Context, key SessionKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key.Subpath != "" {
		delete(s.data, sessionStoreKey(key))
		return nil
	}
	// Cascade: delete the main transcript plus every key under
	// {project}/{session}/.
	mainKey := sessionStoreKey(key)
	prefix := mainKey + "/"
	delete(s.data, mainKey)
	for k := range s.data {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(s.data, k)
		}
	}
	delete(s.mtimes, mainSessionKey(key.ProjectKey, key.SessionID))
	return nil
}

// ListSessions returns main-transcript sessions only (subagent subpaths are
// excluded). Mtime reflects the most recent main-transcript Append.
func (s *InMemorySessionStore) ListSessions(_ context.Context, projectKey string) ([]SessionStoreListEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := projectKey + "/"
	var result []SessionStoreListEntry
	for k := range s.data {
		if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		rest := k[len(prefix):]
		// Main-transcript keys have no additional '/' after the session_id.
		for i := 0; i < len(rest); i++ {
			if rest[i] == '/' {
				rest = ""
				break
			}
		}
		if rest == "" {
			continue
		}
		result = append(result, SessionStoreListEntry{
			SessionID: rest,
			Mtime:     s.mtimes[mainSessionKey(projectKey, rest)],
		})
	}
	return result, nil
}

func (s *InMemorySessionStore) ListSubkeys(_ context.Context, key SessionKey) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := key.ProjectKey + "/" + key.SessionID + "/"
	var result []string
	for k := range s.data {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, k[len(prefix):])
		}
	}
	return result, nil
}

// --- Session management functions (CLI-backed) ---

func runCLISession(cliPath string, args ...string) ([]byte, error) {
	if cliPath == "" {
		var err error
		cliPath, err = exec.LookPath("claude")
		if err != nil {
			return nil, &CLINotFoundError{SearchPath: "PATH"}
		}
	}
	cmd := exec.Command(cliPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, &ProcessError{ExitCode: ee.ExitCode(), Stderr: string(ee.Stderr)}
		}
		return nil, err
	}
	return out, nil
}

// ListSessions returns all sessions for the given working directory.
func ListSessions(ctx context.Context, cliPath, cwd string) ([]SDKSessionInfo, error) {
	args := []string{"sessions", "list", "--output-format", "json"}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	out, err := runCLISession(cliPath, args...)
	if err != nil {
		return nil, err
	}
	var sessions []SDKSessionInfo
	if err := json.Unmarshal(out, &sessions); err != nil {
		return nil, fmt.Errorf("claude: failed to parse sessions list: %w", err)
	}
	return sessions, nil
}

// GetSessionInfo returns metadata for a specific session.
func GetSessionInfo(ctx context.Context, cliPath, sessionID string) (*SDKSessionInfo, error) {
	out, err := runCLISession(cliPath, "sessions", "info", sessionID, "--output-format", "json")
	if err != nil {
		return nil, err
	}
	var info SDKSessionInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("claude: failed to parse session info: %w", err)
	}
	return &info, nil
}

// GetSessionMessages returns the message transcript for a session.
func GetSessionMessages(ctx context.Context, cliPath, sessionID string) ([]SessionMessage, error) {
	out, err := runCLISession(cliPath, "sessions", "messages", sessionID, "--output-format", "json")
	if err != nil {
		return nil, err
	}
	var messages []SessionMessage
	if err := json.Unmarshal(out, &messages); err != nil {
		return nil, fmt.Errorf("claude: failed to parse session messages: %w", err)
	}
	return messages, nil
}

// ListSubagents returns sub-agent sessions within a parent session.
func ListSubagents(ctx context.Context, cliPath, sessionID string) ([]SDKSessionInfo, error) {
	out, err := runCLISession(cliPath, "sessions", "subagents", sessionID, "--output-format", "json")
	if err != nil {
		return nil, err
	}
	var agents []SDKSessionInfo
	if err := json.Unmarshal(out, &agents); err != nil {
		return nil, fmt.Errorf("claude: failed to parse subagents: %w", err)
	}
	return agents, nil
}

// GetSubagentMessages returns transcript messages for a specific sub-agent.
func GetSubagentMessages(ctx context.Context, cliPath, sessionID, subagentID string) ([]SessionMessage, error) {
	out, err := runCLISession(cliPath, "sessions", "messages", sessionID, "--subagent", subagentID, "--output-format", "json")
	if err != nil {
		return nil, err
	}
	var messages []SessionMessage
	if err := json.Unmarshal(out, &messages); err != nil {
		return nil, fmt.Errorf("claude: failed to parse subagent messages: %w", err)
	}
	return messages, nil
}

// RenameSession sets a custom title for a session.
func RenameSession(ctx context.Context, cliPath, sessionID, title string) error {
	_, err := runCLISession(cliPath, "sessions", "rename", sessionID, title)
	return err
}

// TagSession associates a tag with a session.
func TagSession(ctx context.Context, cliPath, sessionID, tag string) error {
	_, err := runCLISession(cliPath, "sessions", "tag", sessionID, tag)
	return err
}

// DeleteSession removes a session.
func DeleteSession(ctx context.Context, cliPath, sessionID string) error {
	_, err := runCLISession(cliPath, "sessions", "delete", sessionID)
	return err
}

// ForkSession creates a copy of a session at the current state.
func ForkSession(ctx context.Context, cliPath, sessionID string) (*ForkSessionResult, error) {
	out, err := runCLISession(cliPath, "sessions", "fork", sessionID, "--output-format", "json")
	if err != nil {
		return nil, err
	}
	var result ForkSessionResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("claude: failed to parse fork result: %w", err)
	}
	return &result, nil
}

// ProjectKeyForDirectory returns the project key that the CLI uses for the given directory.
func ProjectKeyForDirectory(cliPath, dir string) (string, error) {
	out, err := runCLISession(cliPath, "project-key", "--cwd", dir)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// --- Store-backed session functions ---

// ListSessionsFromStore lists sessions using a SessionStore backend.
func ListSessionsFromStore(ctx context.Context, store SessionStore, projectKey string) ([]SessionStoreListEntry, error) {
	return store.ListSessions(ctx, projectKey)
}

// GetSessionInfoFromStore returns session info by loading entries from a SessionStore.
func GetSessionInfoFromStore(ctx context.Context, store SessionStore, projectKey, sessionID string) ([]SessionStoreEntry, error) {
	return store.Load(ctx, SessionKey{ProjectKey: projectKey, SessionID: sessionID})
}

// GetSessionMessagesFromStore returns all messages for a session from a SessionStore.
func GetSessionMessagesFromStore(ctx context.Context, store SessionStore, projectKey, sessionID string) ([]SessionStoreEntry, error) {
	return store.Load(ctx, SessionKey{ProjectKey: projectKey, SessionID: sessionID})
}

// ListSubagentsFromStore lists sub-agent subkeys within a session from a SessionStore.
func ListSubagentsFromStore(ctx context.Context, store SessionStore, projectKey, sessionID string) ([]string, error) {
	return store.ListSubkeys(ctx, SessionKey{ProjectKey: projectKey, SessionID: sessionID})
}

// GetSubagentMessagesFromStore returns messages for a sub-agent from a SessionStore.
func GetSubagentMessagesFromStore(ctx context.Context, store SessionStore, projectKey, sessionID, subagentID string) ([]SessionStoreEntry, error) {
	return store.Load(ctx, SessionKey{ProjectKey: projectKey, SessionID: sessionID, Subpath: subagentID})
}

// RenameSessionViaStore is not supported by the store interface; it requires CLI access.
func RenameSessionViaStore(ctx context.Context, cliPath, sessionID, title string) error {
	return RenameSession(ctx, cliPath, sessionID, title)
}

// TagSessionViaStore is not supported by the store interface; it requires CLI access.
func TagSessionViaStore(ctx context.Context, cliPath, sessionID, tag string) error {
	return TagSession(ctx, cliPath, sessionID, tag)
}

// DeleteSessionViaStore deletes a session from a SessionStore.
func DeleteSessionViaStore(ctx context.Context, store SessionStore, projectKey, sessionID string) error {
	return store.Delete(ctx, SessionKey{ProjectKey: projectKey, SessionID: sessionID})
}

// ForkSessionViaStore is not supported by the store interface; it requires CLI access.
func ForkSessionViaStore(ctx context.Context, cliPath, sessionID string) (*ForkSessionResult, error) {
	return ForkSession(ctx, cliPath, sessionID)
}
