// Package postgres is a Postgres-backed claude.SessionStore reference adapter.
//
// This is a reference implementation demonstrating that the
// claude.SessionStore interface generalizes to a relational backend. It is
// not shipped as part of the SDK; copy it into your project and adapt as
// needed (add migrations, partitioning, retention sweeps, etc.). This
// mirrors the Python SDK's postgres_session_store.py.
//
// Schema (one row per transcript entry; seq orders entries within a key):
//
//	CREATE TABLE IF NOT EXISTS claude_session_store (
//	  project_key text   NOT NULL,
//	  session_id  text   NOT NULL,
//	  subpath     text   NOT NULL DEFAULT '',
//	  seq         bigserial,
//	  entry       jsonb  NOT NULL,
//	  mtime       bigint NOT NULL,
//	  PRIMARY KEY (project_key, session_id, subpath, seq)
//	);
//	CREATE INDEX IF NOT EXISTS claude_session_store_list_idx
//	  ON claude_session_store (project_key, session_id) WHERE subpath = '';
//
// The empty string is the subpath sentinel for the main transcript so the
// composite primary key is total (Postgres treats NULL as distinct in PKs).
//
// JSONB key ordering: entries are stored as jsonb, which reorders object
// keys on read-back. This is explicitly allowed by the SessionStore
// contract (Load requires deep-equal, not byte-equal, returns).
//
// Retention: this adapter never deletes rows on its own. Add a scheduled
// DELETE ... WHERE mtime < $cutoff (or table partitioning by mtime) to
// expire transcripts according to your compliance requirements.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

// Conservative identifier guard for the table name. The name is interpolated
// into DDL/DML (pgx cannot parameterize identifiers), so reject anything
// that isn't a plain [A-Za-z_][A-Za-z0-9_]* to rule out injection.
var identRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const defaultTable = "claude_session_store"

// SessionStore is a Postgres-backed claude.SessionStore.
//
// One row per transcript entry. Append is a single multi-row INSERT;
// Load is SELECT entry ... ORDER BY seq. The pgx pool is safe for
// concurrent use.
type SessionStore struct {
	pool  *pgxpool.Pool
	table string
}

// New creates a Postgres-backed SessionStore.
//
// pool must be a pre-configured *pgxpool.Pool (caller controls DSN, auth,
// pool sizing, etc.). A pool is required so the adapter can be shared
// across concurrent batcher flushes.
//
// table defaults to "claude_session_store" when empty. Must match
// [A-Za-z_][A-Za-z0-9_]* — it is interpolated directly into SQL
// (identifiers cannot be parameterized).
func New(pool *pgxpool.Pool, table string) (*SessionStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres session store: pool is required")
	}
	if table == "" {
		table = defaultTable
	}
	if !identRE.MatchString(table) {
		return nil, fmt.Errorf("postgres session store: table %q must match [A-Za-z_][A-Za-z0-9_]*", table)
	}
	return &SessionStore{pool: pool, table: table}, nil
}

// CreateSchema creates the table and listing index if absent. Idempotent.
//
// Call once at startup (or run the equivalent migration out-of-band).
// The partial index on subpath = '' keeps ListSessions cheap without
// indexing every subagent row.
func (s *SessionStore) CreateSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
			project_key text   NOT NULL,
			session_id  text   NOT NULL,
			subpath     text   NOT NULL DEFAULT '',
			seq         bigserial,
			entry       jsonb  NOT NULL,
			mtime       bigint NOT NULL,
			PRIMARY KEY (project_key, session_id, subpath, seq)
		);
		CREATE INDEX IF NOT EXISTS %[1]s_list_idx
			ON %[1]s (project_key, session_id) WHERE subpath = '';
	`, s.table))
	return err
}

// Append mirrors a batch of transcript entries. Single round-trip multi-row
// INSERT via unnest() so the whole batch lands in one statement (atomic,
// ordered by array position via WITH ORDINALITY, one bigserial draw per row).
func (s *SessionStore) Append(ctx context.Context, key claude.SessionKey, entries []claude.SessionStoreEntry) error {
	if len(entries) == 0 {
		return nil
	}
	payload := make([]string, len(entries))
	for i, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("postgres session store: marshal entry %d: %w", i, err)
		}
		payload[i] = string(data)
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (project_key, session_id, subpath, entry, mtime)
		SELECT $1, $2, $3, e::jsonb, $5
		FROM unnest($4::text[]) WITH ORDINALITY AS t(e, ord)
		ORDER BY ord
	`, s.table),
		key.ProjectKey, key.SessionID, key.Subpath, payload, time.Now().UnixMilli())
	return err
}

// Load returns all entries for a key in append order. Returns nil, nil when
// the session is unknown.
func (s *SessionStore) Load(ctx context.Context, key claude.SessionKey) ([]claude.SessionStoreEntry, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT entry FROM %s
		WHERE project_key = $1 AND session_id = $2 AND subpath = $3
		ORDER BY seq
	`, s.table), key.ProjectKey, key.SessionID, key.Subpath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []claude.SessionStoreEntry
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e claude.SessionStoreEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, fmt.Errorf("postgres session store: unmarshal row: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ListSessions enumerates main-transcript sessions for a project. Subagent
// rows (subpath <> '') are excluded to match InMemorySessionStore semantics.
func (s *SessionStore) ListSessions(ctx context.Context, projectKey string) ([]claude.SessionStoreListEntry, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT session_id, MAX(mtime) AS mtime FROM %s
		WHERE project_key = $1 AND subpath = ''
		GROUP BY session_id
	`, s.table), projectKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []claude.SessionStoreListEntry
	for rows.Next() {
		var sid string
		var mtime int64
		if err := rows.Scan(&sid, &mtime); err != nil {
			return nil, err
		}
		out = append(out, claude.SessionStoreListEntry{SessionID: sid, Mtime: mtime})
	}
	return out, rows.Err()
}

// Delete removes a session. If Subpath is set, only that subpath's rows
// are deleted; otherwise every row for (project_key, session_id) is
// removed (cascade).
func (s *SessionStore) Delete(ctx context.Context, key claude.SessionKey) error {
	if key.Subpath != "" {
		_, err := s.pool.Exec(ctx, fmt.Sprintf(`
			DELETE FROM %s
			WHERE project_key = $1 AND session_id = $2 AND subpath = $3
		`, s.table), key.ProjectKey, key.SessionID, key.Subpath)
		return err
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE project_key = $1 AND session_id = $2
	`, s.table), key.ProjectKey, key.SessionID)
	return err
}

// ListSubkeys returns all non-empty subpaths under a session.
func (s *SessionStore) ListSubkeys(ctx context.Context, key claude.SessionKey) ([]string, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT subpath FROM %s
		WHERE project_key = $1 AND session_id = $2 AND subpath <> ''
	`, s.table), key.ProjectKey, key.SessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var sp string
		if err := rows.Scan(&sp); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

var _ claude.SessionStore = (*SessionStore)(nil)
