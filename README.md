# claude-agent-sdk-go

[![CI](https://github.com/johnayoung/claude-agent-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/johnayoung/claude-agent-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/johnayoung/claude-agent-sdk-go.svg)](https://pkg.go.dev/github.com/johnayoung/claude-agent-sdk-go)

Go SDK for building Claude-powered agents via the Claude CLI.

## Install

```sh
go get github.com/johnayoung/claude-agent-sdk-go@latest
```

Requires Go 1.23+ and the [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated.

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"os"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	for msg, err := range claude.Query(ctx, "What is 2+2? Reply in one sentence.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if m, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Println(text.Text)
				}
			}
		}
	}
}
```

## Packages

| Import | Purpose |
| --- | --- |
| `claude "github.com/johnayoung/claude-agent-sdk-go"` | Core types, `Query`, `NewClient` |
| `github.com/johnayoung/claude-agent-sdk-go/hooks` | Lifecycle event hooks |
| `github.com/johnayoung/claude-agent-sdk-go/permission` | Tool permission callbacks |
| `github.com/johnayoung/claude-agent-sdk-go/mcp` | MCP tool interface and server configs |
| `github.com/johnayoung/claude-agent-sdk-go/agenttest` | Test utilities (mock transport, assertions) |

## Examples

| Example                                | Description                                     |
| -------------------------------------- | ----------------------------------------------- |
| [basic-query](examples/basic-query/)   | Single-turn query with streamed response        |
| [multi-turn](examples/multi-turn/)     | Multi-turn conversation with session resumption |
| [custom-tools](examples/custom-tools/) | In-process MCP tool implementation              |

## License

MIT
