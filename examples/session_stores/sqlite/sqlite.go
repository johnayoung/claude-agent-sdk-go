// Package sqlite is a SQLite-backed claude.SessionStore reference adapter.
//
// This is a reference implementation demonstrating that the
// claude.SessionStore interface generalizes to an embedded relational
// backend. It is not shipped as part of the SDK; copy it into your project
// and adapt as needed (add migrations, retention sweeps, WAL-mode tuning,
// etc.).
//
// Schema (one row per transcript entry; seq orders entries within a key):
//
//	CREATE TABLE IF NOT EXISTS claude_session_store (
//	  seq         INTEGER PRIMARY KEY AUTOINCREMENT,
//	  project_key TEXT    NOT NULL,
//	  session_id  TEXT    NOT NULL,
//	  subpath     TEXT    NOT NULL DEFAULT '',
//	  entry       TEXT    NOT NULL,
//	  mtime       INTEGER NOT NULL
//	);
//	CREATE INDEX IF NOT EXISTS claude_session_store_lookup_idx
//	  ON claude_session_store (project_key, session_id, subpath, seq);
//	CREATE INDEX IF NOT EXISTS claude_session_store_list_idx
//	  ON claude_session_store (project_key, session_id) WHERE subpath = '';
//
// SQLite has only one PRIMARY KEY per table, and AUTOINCREMENT requires the
// PK to be a single INTEGER column. We therefore promote `seq` to PK and
// add an explicit composite lookup index — same query plan as the Postgres
// adapter's composite PK.
//
// The empty string is the subpath sentinel for the main transcript.
//
// Driver: any database/sql driver works. The conformance test uses
// modernc.org/sqlite (pure-Go) so it runs without cgo. Callers who want
// cgo can substitute github.com/mattn/go-sqlite3.
//
// Concurrency: SQLite serializes writers. For multi-goroutine workloads
// enable WAL mode (PRAGMA journal_mode=WAL) on the *sql.DB; otherwise set
// db.SetMaxOpenConns(1) to avoid "database is locked" errors under
// concurrent Append calls.
//
// Retention: this adapter never deletes rows on its own. Add a scheduled
// DELETE ... WHERE mtime < ? to expire transcripts according to your
// compliance requirements.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

// Conservative identifier guard for the table name. The name is interpolated
// into DDL/DML (database/sql cannot parameterize identifiers), so reject
// anything that isn't a plain [A-Za-z_][A-Za-z0-9_]* to rule out injection.
var identRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const defaultTable = "claude_session_store"

// SessionStore is a SQLite-backed claude.SessionStore.
//
// One row per transcript entry. Append is a single multi-row INSERT inside
// a transaction; Load is SELECT entry ... ORDER BY seq.
type SessionStore struct {
	db    *sql.DB
	table string
}

// New creates a SQLite-backed SessionStore.
//
// db must be a pre-opened *sql.DB (caller controls driver, DSN, pragmas,
// pool sizing, etc.).
//
// table defaults to "claude_session_store" when empty. Must match
// [A-Za-z_][A-Za-z0-9_]* — it is interpolated directly into SQL
// (identifiers cannot be parameterized).
func New(db *sql.DB, table string) (*SessionStore, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlite session store: db is required")
	}
	if table == "" {
		table = defaultTable
	}
	if !identRE.MatchString(table) {
		return nil, fmt.Errorf("sqlite session store: table %q must match [A-Za-z_][A-Za-z0-9_]*", table)
	}
	return &SessionStore{db: db, table: table}, nil
}

// CreateSchema creates the table and indexes if absent. Idempotent.
//
// Call once at startup (or run the equivalent migration out-of-band).
func (s *SessionStore) CreateSchema(ctx context.Context) error {
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %[1]s (
			seq         INTEGER PRIMARY KEY AUTOINCREMENT,
			project_key TEXT    NOT NULL,
			session_id  TEXT    NOT NULL,
			subpath     TEXT    NOT NULL DEFAULT '',
			entry       TEXT    NOT NULL,
			mtime       INTEGER NOT NULL
		)`, s.table),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %[1]s_lookup_idx
			ON %[1]s (project_key, session_id, subpath, seq)`, s.table),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %[1]s_list_idx
			ON %[1]s (project_key, session_id) WHERE subpath = ''`, s.table),
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// Append mirrors a batch of transcript entries. Wrapped in a single
// transaction so the whole batch lands atomically and seq values are
// contiguous in append order.
func (s *SessionStore) Append(ctx context.Context, key claude.SessionKey, entries []claude.SessionStoreEntry) error {
	if len(entries) == 0 {
		return nil
	}
	mtime := time.Now().UnixMilli()

	placeholders := make([]string, len(entries))
	args := make([]any, 0, len(entries)*5)
	for i, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("sqlite session store: marshal entry %d: %w", i, err)
		}
		placeholders[i] = "(?, ?, ?, ?, ?)"
		args = append(args, key.ProjectKey, key.SessionID, key.Subpath, string(data), mtime)
	}
	query := fmt.Sprintf(
		`INSERT INTO %s (project_key, session_id, subpath, entry, mtime) VALUES %s`,
		s.table, strings.Join(placeholders, ", "),
	)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Load returns all entries for a key in append order. Returns nil, nil
// when the session is unknown.
func (s *SessionStore) Load(ctx context.Context, key claude.SessionKey) ([]claude.SessionStoreEntry, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT entry FROM %s
		WHERE project_key = ? AND session_id = ? AND subpath = ?
		ORDER BY seq
	`, s.table), key.ProjectKey, key.SessionID, key.Subpath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []claude.SessionStoreEntry
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e claude.SessionStoreEntry
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, fmt.Errorf("sqlite session store: unmarshal row: %w", err)
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
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT session_id, MAX(mtime) AS mtime FROM %s
		WHERE project_key = ? AND subpath = ''
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
		_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
			DELETE FROM %s
			WHERE project_key = ? AND session_id = ? AND subpath = ?
		`, s.table), key.ProjectKey, key.SessionID, key.Subpath)
		return err
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE project_key = ? AND session_id = ?
	`, s.table), key.ProjectKey, key.SessionID)
	return err
}

// ListSubkeys returns all non-empty subpaths under a session.
func (s *SessionStore) ListSubkeys(ctx context.Context, key claude.SessionKey) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT subpath FROM %s
		WHERE project_key = ? AND session_id = ? AND subpath <> ''
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
