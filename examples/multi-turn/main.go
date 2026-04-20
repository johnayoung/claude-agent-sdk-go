// multi-turn demonstrates Client, which maintains a conversation session across
// multiple Query calls. Each call automatically resumes the previous session.
//
// Run:
//
//	go run ./examples/multi-turn
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

	// NewClient resolves the claude CLI path; returns an error if not found.
	client, err := claude.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	prompts := []string{
		"My name is Alex. Remember it.",
		"What is my name?",
		"Say goodbye using my name.",
	}

	for _, prompt := range prompts {
		fmt.Printf("You: %s\n", prompt)
		fmt.Print("Claude: ")

		// client.Query resumes the prior session automatically after the first turn.
		for msg, err := range client.Query(ctx, prompt) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
				os.Exit(1)
			}

			switch m := msg.(type) {
			case *agent.AssistantMessage:
				for _, block := range m.Content {
					if text, ok := block.(*agent.TextBlock); ok {
						fmt.Print(text.Text)
					}
				}
			case *agent.ResultMessage:
				fmt.Printf("\n[session: %s]\n\n", m.SessionID)
			}
		}
	}
}
