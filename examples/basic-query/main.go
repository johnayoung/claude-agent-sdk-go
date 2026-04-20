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
)

func main() {
	ctx := context.Background()

	for msg, err := range claude.Query(ctx, "What is 2+2? Reply in one sentence.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Println(text.Text)
				}
			}
		case *claude.ResultMessage:
			fmt.Printf("\nsession: %s  turns: %d  cost: $%.6f\n",
				m.SessionID, m.NumTurns, m.CostUSD)
		}
	}
}
