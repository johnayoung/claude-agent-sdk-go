// quick-start mirrors the Python SDK's quick_start.py example.
//
// Run:
//
//	go run ./examples/quick-start
package main

import (
	"context"
	"fmt"
	"os"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	basicExample(ctx)
	withOptionsExample(ctx)
	withToolsExample(ctx)
}

func basicExample(ctx context.Context) {
	fmt.Println("=== Basic Example ===")

	for msg, err := range claude.Query(ctx, "What is 2 + 2?") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if m, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", text.Text)
				}
			}
		}
	}
	fmt.Println()
}

func withOptionsExample(ctx context.Context) {
	fmt.Println("=== With Options Example ===")

	for msg, err := range claude.Query(ctx, "Explain what Go is in one sentence.",
		claude.WithSystemPrompt("You are a helpful assistant that explains things simply."),
		claude.WithMaxTurns(1),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if m, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", text.Text)
				}
			}
		}
	}
	fmt.Println()
}

func withToolsExample(ctx context.Context) {
	fmt.Println("=== With Tools Example ===")

	for msg, err := range claude.Query(ctx, "Create a file called hello.txt with 'Hello, World!' in it",
		claude.WithAllowedTools("Read", "Write"),
		claude.WithSystemPrompt("You are a helpful file assistant."),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", text.Text)
				}
			}
		case *claude.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}
	fmt.Println()
}
