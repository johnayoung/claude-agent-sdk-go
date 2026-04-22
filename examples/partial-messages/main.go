// partial-messages demonstrates real-time partial message streaming via
// WithIncludePartialMessages.
//
// With this option enabled, the SDK emits *StreamEvent messages that carry
// raw Anthropic streaming events (message_start, content_block_delta,
// message_delta, message_stop, etc.) in addition to the usual SystemMessage,
// AssistantMessage, and ResultMessage. This example prints every message as
// it arrives — each rendered as JSON so pointers and RawMessage fields are
// readable — so you can inspect the full wire-level stream.
//
// Run:
//
//	go run ./examples/partial-messages
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func main() {
	prompt := "Think of three jokes, then tell one"

	fmt.Println("Partial Message Streaming Example")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Println(strings.Repeat("=", 50))

	ctx := context.Background()
	opts := []claude.Option{
		claude.WithIncludePartialMessages(),
		claude.WithMaxTurns(1),
	}

	for msg, err := range claude.Query(ctx, prompt, opts...) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		body, mErr := json.Marshal(msg)
		if mErr != nil {
			fmt.Fprintf(os.Stderr, "marshal error for %T: %v\n", msg, mErr)
			continue
		}
		fmt.Printf("%T %s\n", msg, body)
	}
}
