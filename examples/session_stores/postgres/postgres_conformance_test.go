package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest/sessionstoretest"
	pgstore "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/postgres"
)

// TestConformance runs the shared SessionStore conformance suite against a
// live Postgres backend. Skipped unless SESSION_STORE_POSTGRES_URL is set.
//
// Example:
//
//	docker run -d --rm -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine
//	SESSION_STORE_POSTGRES_URL='postgres://postgres:postgres@localhost:5432/postgres' \
//	    go test -v ./...
//
// Each subtest gets its own ephemeral table, so concurrent runs and CI
// against a shared cluster don't interfere.
func TestConformance(t *testing.T) {
	dsn := os.Getenv("SESSION_STORE_POSTGRES_URL")
	if dsn == "" {
		t.Skip("SESSION_STORE_POSTGRES_URL not set; skipping live Postgres conformance")
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(pingCtx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(pingCtx); err != nil {
		t.Fatalf("postgres unreachable: %v", err)
	}

	sessionstoretest.Run(t, func(t *testing.T) claude.SessionStore {
		table := "sstest_" + randomHex(t, 6)
		store, err := pgstore.New(pool, table)
		if err != nil {
			t.Fatalf("pgstore.New: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := store.CreateSchema(ctx); err != nil {
			t.Fatalf("CreateSchema: %v", err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := pool.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table)); err != nil {
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
