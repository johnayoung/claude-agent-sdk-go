// basic-query demonstrates the simplest SDK usage: a single prompt, streamed response.
//
// Run:
//
//	go run ./examples/basic-query
package main

import (
	"context"
	"fmt"
	"os"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agent"
)

func main() {
	ctx := context.Background()

	// Query streams messages from the Claude CLI and terminates after ResultMessage.
	for msg, err := range claude.Query(ctx, "What is 2+2? Reply in one sentence.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		switch m := msg.(type) {
		case *agent.AssistantMessage:
			// AssistantMessage holds one or more content blocks; print text blocks.
			for _, block := range m.Content {
				if text, ok := block.(*agent.TextBlock); ok {
					fmt.Println(text.Text)
				}
			}
		case *agent.ResultMessage:
			fmt.Printf("\nsession: %s  turns: %d  cost: $%.6f\n",
				m.SessionID, m.NumTurns, m.CostUSD)
		}
	}
}
