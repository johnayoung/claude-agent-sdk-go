package claude

import "context"

// SessionKey identifies a session in a SessionStore.
type SessionKey struct {
	ProjectKey string
	SessionID  string
	Subpath    string
}

// SessionStore is the interface for custom session persistence backends.
type SessionStore interface {
	Append(ctx context.Context, key SessionKey, entries [][]byte) error
	Load(ctx context.Context, key SessionKey) ([][]byte, error)
	Delete(ctx context.Context, key SessionKey) error
	ListSessions(ctx context.Context, projectKey string) ([]string, error)
	ListSubkeys(ctx context.Context, key SessionKey) ([]string, error)
}
