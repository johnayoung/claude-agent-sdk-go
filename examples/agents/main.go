// agents demonstrates defining and using custom sub-agents with specific tools,
// prompts, and models.
//
// Run:
//
//	go run ./examples/agents
package main

import (
	"context"
	"fmt"
	"os"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func codeReviewerExample(ctx context.Context) {
	fmt.Println("=== Code Reviewer Agent Example ===")

	agents := map[string]claude.AgentDefinition{
		"code-reviewer": {
			Description: "Reviews code for best practices and potential issues",
			Prompt: "You are a code reviewer. Analyze code for bugs, performance issues, " +
				"security vulnerabilities, and adherence to best practices. " +
				"Provide constructive feedback.",
			Tools: []string{"Read", "Grep"},
			Model: "sonnet",
		},
	}

	for msg, err := range claude.Query(ctx,
		"Use the code-reviewer agent to review the code in query.go",
		claude.WithAgents(agents),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		printMessage(msg)
	}
	fmt.Println()
}

func documentationWriterExample(ctx context.Context) {
	fmt.Println("=== Documentation Writer Agent Example ===")

	agents := map[string]claude.AgentDefinition{
		"doc-writer": {
			Description: "Writes comprehensive documentation",
			Prompt: "You are a technical documentation expert. Write clear, comprehensive " +
				"documentation with examples. Focus on clarity and completeness.",
			Tools: []string{"Read", "Write", "Edit"},
			Model: "sonnet",
		},
	}

	for msg, err := range claude.Query(ctx,
		"Use the doc-writer agent to explain what AgentDefinition is used for",
		claude.WithAgents(agents),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		printMessage(msg)
	}
	fmt.Println()
}

func multipleAgentsExample(ctx context.Context) {
	fmt.Println("=== Multiple Agents Example ===")

	agents := map[string]claude.AgentDefinition{
		"analyzer": {
			Description: "Analyzes code structure and patterns",
			Prompt:      "You are a code analyzer. Examine code structure, patterns, and architecture.",
			Tools:       []string{"Read", "Grep", "Glob"},
		},
		"tester": {
			Description: "Creates and runs tests",
			Prompt:      "You are a testing expert. Write comprehensive tests and ensure code quality.",
			Tools:       []string{"Read", "Write", "Bash"},
			Model:       "sonnet",
		},
	}

	for msg, err := range claude.Query(ctx,
		"Use the analyzer agent to find all Go files in the examples/ directory",
		claude.WithAgents(agents),
		claude.WithSettingSources("user", "project"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		printMessage(msg)
	}
	fmt.Println()
}

func printMessage(msg claude.Message) {
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

func main() {
	ctx := context.Background()

	codeReviewerExample(ctx)
	documentationWriterExample(ctx)
	multipleAgentsExample(ctx)
}
