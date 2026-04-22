#!/usr/bin/env bash
# Spin up local Redis, run the conformance suite against it, tear down.
set -euo pipefail

NAME=sstest-redis
PORT=16379

docker run -d --rm --name "$NAME" -p "$PORT:6379" redis:7-alpine >/dev/null
trap 'docker stop "$NAME" >/dev/null' EXIT

until docker exec "$NAME" redis-cli ping >/dev/null 2>&1; do sleep 0.1; done

cd "$(dirname "$0")"
SESSION_STORE_REDIS_URL="redis://localhost:$PORT/0" go test -v -run TestConformance ./...
