package redis_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest/sessionstoretest"
	redisstore "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/redis"
)

// TestConformance runs the shared SessionStore conformance suite against a
// live Redis backend. Skipped unless SESSION_STORE_REDIS_URL is set.
//
// Example:
//
//	docker run -d -p 6379:6379 redis:7-alpine
//	SESSION_STORE_REDIS_URL=redis://localhost:6379/0 \
//	    go test -v ./...
//
// Each subtest uses a unique key prefix so parallel/sequential runs don't
// see each other's data — no FLUSHDB needed.
func TestConformance(t *testing.T) {
	url := os.Getenv("SESSION_STORE_REDIS_URL")
	if url == "" {
		t.Skip("SESSION_STORE_REDIS_URL not set; skipping live Redis conformance")
	}

	opts, err := goredis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse SESSION_STORE_REDIS_URL: %v", err)
	}
	client := goredis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		t.Fatalf("redis unreachable at %s: %v", url, err)
	}

	sessionstoretest.Run(t, func(t *testing.T) claude.SessionStore {
		prefix := "sstest-" + randomHex(t, 6) + ":"
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			deleteByPrefix(ctx, t, client, prefix)
		})
		return redisstore.New(client, prefix)
	})
}

func deleteByPrefix(ctx context.Context, t *testing.T, client *goredis.Client, prefix string) {
	t.Helper()
	iter := client.Scan(ctx, 0, prefix+"*", 500).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 500 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				t.Logf("redis cleanup DEL: %v", err)
			}
			keys = keys[:0]
		}
	}
	if err := iter.Err(); err != nil {
		t.Logf("redis cleanup scan: %v", err)
	}
	if len(keys) > 0 {
		if err := client.Del(ctx, keys...).Err(); err != nil {
			t.Logf("redis cleanup DEL (tail): %v", err)
		}
	}
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b)
}
