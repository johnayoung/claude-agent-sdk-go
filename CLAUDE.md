# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Workflow

- **Build**: `go build ./...`
- **Lint**: `go vet ./...`
- **Test**: `go test ./...` or target a single test with `go test -run TestName ./path/to/package`

## Project Organization

The main package resides in the repo root with several key modules:

- **Core functionality**: `client.go` provides multi-turn session management, while `query.go` handles single requests
- **Support files**: `types.go` and `message.go` contain type definitions; `internal/transport/` houses subprocess CLI communication and `parse.go` handles message deserialization
- **Subpackages**: `hooks/` (lifecycle callbacks), `permission/` (tool approval), `mcp/` (MCP tool/server config), `agenttest/` (mock transport and test helpers)
