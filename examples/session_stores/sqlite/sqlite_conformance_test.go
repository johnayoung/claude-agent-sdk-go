package sqlite_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest/sessionstoretest"
	sqlitestore "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/sqlite"
)

// TestConformance runs the shared SessionStore conformance suite against a
// SQLite backend. Unlike the postgres/redis/s3 conformance tests it needs
// no env var — SQLite is embedded, so the test creates an ephemeral
// database under t.TempDir() and runs unconditionally.
//
// Each subtest gets its own ephemeral table so the 14 contracts don't
// observe each other's state.
func TestConformance(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Single connection avoids "database is locked" under the conformance
	// suite's mixed read/write workload without needing WAL setup here.
	db.SetMaxOpenConns(1)

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("sqlite unreachable: %v", err)
	}

	sessionstoretest.Run(t, func(t *testing.T) claude.SessionStore {
		table := "sstest_" + randomHex(t, 6)
		store, err := sqlitestore.New(db, table)
		if err != nil {
			t.Fatalf("sqlitestore.New: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := store.CreateSchema(ctx); err != nil {
			t.Fatalf("CreateSchema: %v", err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table)); err != nil {
				t.Logf("drop table %s: %v", table, err)
			}
		})
		return store
	})
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b)
}
