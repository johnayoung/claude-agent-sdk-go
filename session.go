package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
)

// SessionKey identifies a session in a SessionStore.
type SessionKey struct {
	ProjectKey string `json:"project_key"`
	SessionID  string `json:"session_id"`
	Subpath    string `json:"subpath,omitempty"`
}

// SessionStoreEntry represents a single entry persisted in a session store.
type SessionStoreEntry struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
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
	mu   sync.RWMutex
	data map[string][]SessionStoreEntry
}

// NewInMemorySessionStore creates an empty in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{data: make(map[string][]SessionStoreEntry)}
}

func sessionStoreKey(key SessionKey) string {
	k := key.ProjectKey + "/" + key.SessionID
	if key.Subpath != "" {
		k += "/" + key.Subpath
	}
	return k
}

func (s *InMemorySessionStore) Append(_ context.Context, key SessionKey, entries []SessionStoreEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := sessionStoreKey(key)
	s.data[k] = append(s.data[k], entries...)
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

func (s *InMemorySessionStore) Delete(_ context.Context, key SessionKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionStoreKey(key))
	return nil
}

func (s *InMemorySessionStore) ListSessions(_ context.Context, projectKey string) ([]SessionStoreListEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := projectKey + "/"
	var result []SessionStoreListEntry
	seen := make(map[string]bool)
	for k := range s.data {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			rest := k[len(prefix):]
			sid := rest
			for i := 0; i < len(rest); i++ {
				if rest[i] == '/' {
					sid = rest[:i]
					break
				}
			}
			if !seen[sid] {
				seen[sid] = true
				result = append(result, SessionStoreListEntry{SessionID: sid})
			}
		}
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
