# SessionStore reference adapters

> **Reference implementations for interface validation. Not packaged, not maintained as production code.**

Reference [`SessionStore`](../../session.go) implementations — copy into your project, install the backend client, adapt as needed.

Each example lives in its own Go module with a `replace` directive back to the parent SDK. This keeps the root `go.mod` free of heavyweight optional dependencies: `go build ./...` at the repo root never pulls Redis, pgx, or the AWS SDK.

## Running the examples locally

Each subdirectory is a self-contained module. To compile-check one:

```bash
cd examples/session_stores/redis
go build ./...
go vet ./...
```

The adapters are designed to be copied into your own project. Replace the `replace` directive with a concrete version of `github.com/johnayoung/claude-agent-sdk-go` once you pull the SDK as a dependency.

## Production checklist

These adapters are reference code. Before running one in production, work through the relevant items below.

### All adapters

- `Append` failures are logged and surfaced as mirror errors by the SDK; they never block the conversation. Monitor for these so silent mirror gaps don't go unnoticed.
- Load-test under your expected throughput before deploying.
- Plan retention (S3 lifecycle policies / Redis `EXPIRE` / Postgres scheduled `DELETE`) — the SDK never auto-deletes.

### S3

- Required IAM actions on the bucket/prefix: `s3:PutObject`, `s3:GetObject`, `s3:ListBucket`, `s3:DeleteObject`.
- Part-file ordering uses the client-side wall clock. Multiple writer instances with clock skew >1s may produce out-of-order `Load` results. Use NTP or a single writer per session.
- Configure S3 lifecycle policies for retention — the SDK never auto-deletes.

### Redis

- Set `maxmemory-policy noeviction` (or use a dedicated DB) — eviction will silently drop session data.
- Lists are unbounded; implement TTL via `EXPIRE` in a wrapper if needed.
- Redis Cluster: keys sharing a `{project_key}:{session_id}` prefix should hash to the same slot — wrap in `{...}` hash tags if using Cluster.
- If you derive `project_key` or `session_id` outside the SDK, ensure they cannot contain `:` (the key separator) — collisions would mix data across keys.

### Postgres

- Size the `pgxpool.Pool` ≥ expected concurrent sessions; don't share a pool with request-handler code that holds connections.
- `jsonb` reorders keys — contract-safe, but don't byte-compare entries.
- Add a retention job (`DELETE WHERE mtime < ...`) — the table grows unbounded.

---

## S3 — [`s3/s3.go`](s3/s3.go)

Stores transcripts as JSONL part files:

```
s3://{bucket}/{prefix}{project_key}/{session_id}/part-{epochMs13}-{rand6}.jsonl
```

Each `Append` writes a new part; `Load` lists, sorts, and concatenates them.

### Dependencies

```
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/service/s3
```

### Usage

```go
import (
    "context"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"

    claude "github.com/johnayoung/claude-agent-sdk-go"
    s3store "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/s3"
)

cfg, err := config.LoadDefaultConfig(context.Background())
if err != nil { /* ... */ }
client := s3.NewFromConfig(cfg)

store := s3store.New(client, "my-claude-sessions", "transcripts")

for msg, err := range claude.Query(ctx, "Hello!",
    claude.WithSessionStore(store),
) {
    // messages are mirrored to S3 as they stream
    _ = msg; _ = err
}
```

### Running live against MinIO

```bash
docker run -d -p 9000:9000 minio/minio server /data
docker run --rm --network host minio/mc \
    sh -c 'mc alias set local http://localhost:9000 minioadmin minioadmin && mc mb local/test'
```

Then configure the S3 client with `BaseEndpoint: "http://localhost:9000"` and a static credentials provider.

---

## Redis — [`redis/redis.go`](redis/redis.go)

Backed by [`redis/go-redis/v9`](https://github.com/redis/go-redis).

### Dependencies

```
github.com/redis/go-redis/v9
```

### Key scheme

```
{prefix}{project_key}:{session_id}               list  - main transcript entries (JSON each)
{prefix}{project_key}:{session_id}:{subpath}     list  - subagent transcript entries
{prefix}{project_key}:{session_id}:__subkeys     set   - subpaths under this session
{prefix}{project_key}:__sessions                 zset  - session_id -> mtime(ms)
```

Each `Append` is an `RPUSH` plus an index update in a single `MULTI`; `Load` is `LRANGE 0 -1`.

### Usage

```go
import (
    goredis "github.com/redis/go-redis/v9"

    claude "github.com/johnayoung/claude-agent-sdk-go"
    redisstore "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/redis"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
store := redisstore.New(client, "transcripts")

for msg, err := range claude.Query(ctx, "Hello!",
    claude.WithSessionStore(store),
) {
    _ = msg; _ = err
}
```

### Running live

```bash
docker run -d -p 6379:6379 redis:7-alpine
```

---

## Postgres — [`postgres/postgres.go`](postgres/postgres.go)

Backed by [`jackc/pgx/v5`](https://github.com/jackc/pgx), the native-asyncio Postgres driver.

### Dependencies

```
github.com/jackc/pgx/v5
```

### Schema

One row per transcript entry; `seq` (a `bigserial`) orders entries within a `(project_key, session_id, subpath)` key:

```sql
CREATE TABLE IF NOT EXISTS claude_session_store (
  project_key text   NOT NULL,
  session_id  text   NOT NULL,
  subpath     text   NOT NULL DEFAULT '',
  seq         bigserial,
  entry       jsonb  NOT NULL,
  mtime       bigint NOT NULL,
  PRIMARY KEY (project_key, session_id, subpath, seq)
);
CREATE INDEX IF NOT EXISTS claude_session_store_list_idx
  ON claude_session_store (project_key, session_id) WHERE subpath = '';
```

`Append` is a single multi-row `INSERT ... SELECT unnest(...)`; `Load` is `SELECT entry ... ORDER BY seq`. The `CreateSchema(ctx)` method is idempotent — call it once at startup or run the equivalent migration out-of-band.

### Usage

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"

    claude "github.com/johnayoung/claude-agent-sdk-go"
    postgresstore "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/postgres"
)

pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/db")
if err != nil { /* ... */ }
defer pool.Close()

store, err := postgresstore.New(pool, "") // "" -> default table name
if err != nil { /* ... */ }
if err := store.CreateSchema(ctx); err != nil { /* ... */ }

for msg, err := range claude.Query(ctx, "Hello!",
    claude.WithSessionStore(store),
) {
    _ = msg; _ = err
}
```

### JSONB key ordering

Entries are stored as `jsonb`, which reorders object keys on read-back (shorter keys first, then by byte order). This is explicitly allowed by the `SessionStore` contract — `Load` requires *deep-equal*, not *byte-equal*, returns. If you need byte-stable storage, switch the column to `json` or `text`.

### Running live

```bash
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine
```
