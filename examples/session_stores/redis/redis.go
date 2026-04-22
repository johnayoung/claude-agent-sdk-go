// Package redis is a Redis-backed claude.SessionStore reference adapter.
//
// This is a reference implementation demonstrating that the
// claude.SessionStore interface generalizes to a non-blob backend. It is
// not shipped as part of the SDK; copy it into your project and adapt as
// needed. This mirrors the Python SDK's redis_session_store.py.
//
// Key scheme (":" separator; project_key / session_id are opaque so
// collisions with the SDK's "/"-based project_key are avoided):
//
//	{prefix}{project_key}:{session_id}               list  - main transcript
//	{prefix}{project_key}:{session_id}:{subpath}     list  - subagent transcript
//	{prefix}{project_key}:{session_id}:__subkeys     set   - subpaths in this session
//	{prefix}{project_key}:__sessions                 zset  - session_id -> mtime(ms)
//
// Retention: this adapter never expires keys on its own. Configure Redis
// key expiration on your prefix or call Delete according to your
// compliance requirements.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

const (
	// Reserved subpath sentinel for the per-session subkey set.
	sentinelSubkeys = "__subkeys"
	// Reserved session_id sentinel for the per-project session index.
	sentinelSessions = "__sessions"
)

// SessionStore is a Redis-backed claude.SessionStore.
//
// Each Append is an RPUSH plus an index update in a single MULTI; Load is
// LRANGE 0 -1. The go-redis client is safe for concurrent use.
type SessionStore struct {
	client goredis.Cmdable
	prefix string

	clockMu sync.Mutex
	lastMs  int64
}

// New creates a Redis-backed SessionStore.
//
// The client must be configured by the caller (host, auth, TLS, etc.).
// Non-empty prefix is normalized to end in exactly one ":"; an empty
// prefix produces no leading separator.
func New(client goredis.Cmdable, prefix string) *SessionStore {
	p := strings.TrimRight(prefix, ":")
	if p != "" {
		p += ":"
	}
	return &SessionStore{client: client, prefix: p}
}

func (s *SessionStore) entryKey(key claude.SessionKey) string {
	parts := []string{key.ProjectKey, key.SessionID}
	if key.Subpath != "" {
		parts = append(parts, key.Subpath)
	}
	return s.prefix + strings.Join(parts, ":")
}

func (s *SessionStore) subkeysKey(key claude.SessionKey) string {
	return fmt.Sprintf("%s%s:%s:%s", s.prefix, key.ProjectKey, key.SessionID, sentinelSubkeys)
}

func (s *SessionStore) sessionsKey(projectKey string) string {
	return fmt.Sprintf("%s%s:%s", s.prefix, projectKey, sentinelSessions)
}

// monotonicNowMs returns a strictly-increasing epoch-ms within this instance
// so same-instance same-ms appends stay distinguishable in the session index.
func (s *SessionStore) monotonicNowMs() int64 {
	s.clockMu.Lock()
	defer s.clockMu.Unlock()
	now := time.Now().UnixMilli()
	if now <= s.lastMs {
		now = s.lastMs + 1
	}
	s.lastMs = now
	return now
}

// Append mirrors a batch of transcript entries to Redis.
func (s *SessionStore) Append(ctx context.Context, key claude.SessionKey, entries []claude.SessionStoreEntry) error {
	if len(entries) == 0 {
		return nil
	}
	values := make([]any, len(entries))
	for i, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("redis session store: marshal entry %d: %w", i, err)
		}
		values[i] = string(data)
	}
	pipe := s.client.TxPipeline()
	pipe.RPush(ctx, s.entryKey(key), values...)
	if key.Subpath != "" {
		pipe.SAdd(ctx, s.subkeysKey(key), key.Subpath)
	} else {
		// Only main-transcript appends bump the session index.
		pipe.ZAdd(ctx, s.sessionsKey(key.ProjectKey), goredis.Z{
			Score:  float64(s.monotonicNowMs()),
			Member: key.SessionID,
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Load returns all entries for a key in append order. Returns nil, nil when
// the session is unknown. Malformed lines are logged and skipped.
func (s *SessionStore) Load(ctx context.Context, key claude.SessionKey) ([]claude.SessionStoreEntry, error) {
	raw, err := s.client.LRange(ctx, s.entryKey(key), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]claude.SessionStoreEntry, 0, len(raw))
	for _, line := range raw {
		var e claude.SessionStoreEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			log.Printf("redis session store: skipping malformed entry: %v", err)
			continue
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ListSessions enumerates sessions for a project with modification times.
func (s *SessionStore) ListSessions(ctx context.Context, projectKey string) ([]claude.SessionStoreListEntry, error) {
	pairs, err := s.client.ZRangeWithScores(ctx, s.sessionsKey(projectKey), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]claude.SessionStoreListEntry, 0, len(pairs))
	for _, p := range pairs {
		sid, _ := p.Member.(string)
		out = append(out, claude.SessionStoreListEntry{
			SessionID: sid,
			Mtime:     int64(p.Score),
		})
	}
	return out, nil
}

// Delete removes a session (or a specific subpath). Deleting the main
// session cascades to all subpaths.
func (s *SessionStore) Delete(ctx context.Context, key claude.SessionKey) error {
	if key.Subpath != "" {
		pipe := s.client.TxPipeline()
		pipe.Del(ctx, s.entryKey(key))
		pipe.SRem(ctx, s.subkeysKey(key), key.Subpath)
		_, err := pipe.Exec(ctx)
		return err
	}
	subkeysK := s.subkeysKey(key)
	subpaths, err := s.client.SMembers(ctx, subkeysK).Result()
	if err != nil {
		return err
	}
	toDelete := make([]string, 0, 2+len(subpaths))
	toDelete = append(toDelete, s.entryKey(key), subkeysK)
	for _, sp := range subpaths {
		toDelete = append(toDelete, s.entryKey(claude.SessionKey{
			ProjectKey: key.ProjectKey,
			SessionID:  key.SessionID,
			Subpath:    sp,
		}))
	}
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, toDelete...)
	pipe.ZRem(ctx, s.sessionsKey(key.ProjectKey), key.SessionID)
	_, err = pipe.Exec(ctx)
	return err
}

// ListSubkeys returns all subpaths under a session.
func (s *SessionStore) ListSubkeys(ctx context.Context, key claude.SessionKey) ([]string, error) {
	return s.client.SMembers(ctx, s.subkeysKey(claude.SessionKey{
		ProjectKey: key.ProjectKey,
		SessionID:  key.SessionID,
	})).Result()
}

var _ claude.SessionStore = (*SessionStore)(nil)
