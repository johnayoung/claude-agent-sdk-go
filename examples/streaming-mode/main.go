// streaming-mode demonstrates various patterns for building applications with
// the Claude SDK streaming interface. Adapted from the Python SDK's
// streaming_mode.py example.
//
// Usage:
//
//	go run ./examples/streaming-mode              - List examples
//	go run ./examples/streaming-mode all           - Run all examples
//	go run ./examples/streaming-mode basic         - Run a specific example
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func displayMessage(msg claude.Message) {
	switch m := msg.(type) {
	case *claude.UserMessage:
		for _, block := range m.Content {
			if text, ok := block.(*claude.TextBlock); ok {
				fmt.Printf("User: %s\n", text.Text)
			}
		}
	case *claude.AssistantMessage:
		for _, block := range m.Content {
			if text, ok := block.(*claude.TextBlock); ok {
				fmt.Printf("Claude: %s\n", text.Text)
			}
		}
	case *claude.ResultMessage:
		fmt.Println("Result ended")
	}
}

func exampleBasicStreaming(ctx context.Context) {
	fmt.Println("=== Basic Streaming Example ===")

	fmt.Println("User: What is 2+2?")
	for msg, err := range claude.Query(ctx, "What is 2+2? Reply in one sentence.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleMultiTurnConversation(ctx context.Context) {
	fmt.Println("=== Multi-Turn Conversation Example ===")

	client, err := claude.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("User: What's the capital of France?")
	for msg, err := range client.Query(ctx, "What's the capital of France? Reply in one sentence.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println("\nUser: What's the population of that city?")
	for msg, err := range client.Query(ctx, "What's the population of that city? Reply in one sentence.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleWithInterrupt(ctx context.Context) {
	fmt.Println("=== Interrupt Example ===")

	client, err := claude.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("User: Count from 1 to 100 slowly")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg, err := range client.Query(ctx, "Count from 1 to 100 slowly, with a brief pause between each number") {
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
				return
			}
			displayMessage(msg)
		}
	}()

	time.Sleep(2 * time.Second)
	fmt.Println("\n[After 2 seconds, sending interrupt...]")
	client.Interrupt()
	<-done

	fmt.Println("\nUser: Never mind, just tell me a quick joke")
	for msg, err := range client.Query(ctx, "Never mind, just tell me a quick joke. Reply briefly.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleManualMessageHandling(ctx context.Context) {
	fmt.Println("=== Manual Message Handling Example ===")

	knownLanguages := []string{"Python", "JavaScript", "Java", "C++", "Go", "Rust", "Ruby"}
	var languagesFound []string

	for msg, err := range claude.Query(ctx, "List 5 programming languages and their main use cases. Be concise.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", text.Text)
					for _, lang := range knownLanguages {
						if strings.Contains(text.Text, lang) && !slices.Contains(languagesFound, lang) {
							languagesFound = append(languagesFound, lang)
							fmt.Printf("  -> Found language: %s\n", lang)
						}
					}
				}
			}
		case *claude.ResultMessage:
			displayMessage(m)
			fmt.Printf("Total languages mentioned: %d\n", len(languagesFound))
		}
	}

	fmt.Println()
}

func exampleWithOptions(ctx context.Context) {
	fmt.Println("=== Custom Options Example ===")

	fmt.Println("User: What is the square root of 144?")
	for msg, err := range claude.Query(ctx,
		"What is the square root of 144? Reply in one sentence.",
		claude.WithSystemPrompt("You are a helpful math tutor. Always show your reasoning."),
		claude.WithAllowedTools("Bash", "Read"),
		claude.WithEnv(map[string]string{"MATH_MODE": "verbose"}),
		claude.WithMaxTurns(1),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleBashCommand(ctx context.Context) {
	fmt.Println("=== Bash Command Example ===")

	fmt.Println("User: Run a bash echo command")
	messageTypes := map[string]bool{}

	for msg, err := range claude.Query(ctx, "Run a bash echo command that says 'Hello from bash!'") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		messageTypes[msg.MessageType()] = true

		switch m := msg.(type) {
		case *claude.UserMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *claude.TextBlock:
					fmt.Printf("User: %s\n", b.Text)
				case *claude.ToolResultBlock:
					content := string(b.Content)
					if len(content) > 100 {
						content = content[:100] + "..."
					}
					fmt.Printf("Tool Result (id: %s): %s\n", b.ToolUseID, content)
				}
			}
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *claude.TextBlock:
					fmt.Printf("Claude: %s\n", b.Text)
				case *claude.ToolUseBlock:
					fmt.Printf("Tool Use: %s (id: %s)\n", b.Name, b.ID)
					if b.Name == "Bash" {
						var input map[string]any
						if json.Unmarshal(b.Input, &input) == nil {
							if cmd, ok := input["command"].(string); ok {
								fmt.Printf("  Command: %s\n", cmd)
							}
						}
					}
				}
			}
		case *claude.ResultMessage:
			fmt.Println("Result ended")
			if m.TotalCostUSD != nil {
				fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	types := make([]string, 0, len(messageTypes))
	for t := range messageTypes {
		types = append(types, t)
	}
	fmt.Printf("\nMessage types received: %s\n", strings.Join(types, ", "))

	fmt.Println()
}

func exampleErrorHandling(ctx context.Context) {
	fmt.Println("=== Error Handling Example ===")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fmt.Println("User: Count to 50 slowly")
	var msgCount int

	for msg, err := range claude.Query(ctx, "Count to 50 slowly, pausing between each number") {
		if err != nil {
			if ctx.Err() != nil {
				fmt.Printf("\nContext deadline exceeded after 5 seconds - demonstrating graceful handling\n")
				fmt.Printf("Received %d messages before timeout\n", msgCount)
			} else {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			break
		}

		msgCount++
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if text, ok := block.(*claude.TextBlock); ok {
					truncated := text.Text
					if len(truncated) > 50 {
						truncated = truncated[:50] + "..."
					}
					fmt.Printf("Claude: %s\n", truncated)
				}
			}
		case *claude.ResultMessage:
			displayMessage(m)
		}
	}

	fmt.Println()
}

func exampleMCPStatus(ctx context.Context) {
	fmt.Println("=== MCP Status Example ===")

	client, err := claude.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating client: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("User: Hello")
	for msg, err := range client.Query(ctx, "Hello, reply briefly.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting MCP status: %v\n", err)
		return
	}

	fmt.Printf("\nMCP Servers (%d):\n", len(status.MCPServers))
	for _, s := range status.MCPServers {
		fmt.Printf("  %s: %s", s.Name, s.Status)
		if s.Error != "" {
			fmt.Printf(" (error: %s)", s.Error)
		}
		fmt.Printf(" [%d tools]\n", len(s.Tools))
	}

	fmt.Println()
}

func exampleContextUsage(ctx context.Context) {
	fmt.Println("=== Context Usage Example ===")

	client, err := claude.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating client: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("User: Hello")
	for msg, err := range client.Query(ctx, "Hello, reply briefly.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	usage, err := client.GetContextUsage(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting context usage: %v\n", err)
		return
	}

	fmt.Printf("\nContext Usage: %d / %d tokens (%.1f%%)\n", usage.TotalTokens, usage.MaxTokens, usage.Percentage)
	fmt.Printf("Model: %s\n", usage.Model)
	for _, cat := range usage.Categories {
		fmt.Printf("  %s: %d tokens\n", cat.Name, cat.Tokens)
	}

	fmt.Println()
}

func exampleServerToolBlocks(ctx context.Context) {
	fmt.Println("=== Server Tool Block Handling Example ===")
	fmt.Println("(Demonstrating type assertions for ServerToolUseBlock and ServerToolResultBlock)")

	for msg, err := range claude.Query(ctx, "Use an MCP tool if one is available. Otherwise just say hello. Reply briefly.") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if am, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range am.Content {
				switch b := block.(type) {
				case *claude.TextBlock:
					fmt.Printf("Claude: %s\n", b.Text)
				case *claude.ToolUseBlock:
					fmt.Printf("Tool Use: %s (id: %s)\n", b.Name, b.ID)
				case *claude.ServerToolUseBlock:
					fmt.Printf("Server Tool Use: %s (id: %s)\n", b.Name, b.ID)
					fmt.Printf("  Input: %s\n", string(b.Input))
				case *claude.ServerToolResultBlock:
					fmt.Printf("Server Tool Result (tool_use_id: %s)\n", b.ToolUseID)
					fmt.Printf("  Content: %s\n", string(b.Content))
				}
			}
		}
	}

	fmt.Println()
}

func main() {
	examples := map[string]func(context.Context){
		"basic":              exampleBasicStreaming,
		"multi_turn":         exampleMultiTurnConversation,
		"interrupt":          exampleWithInterrupt,
		"manual_handling":    exampleManualMessageHandling,
		"options":            exampleWithOptions,
		"bash_command":       exampleBashCommand,
		"error_handling":     exampleErrorHandling,
		"mcp_status":         exampleMCPStatus,
		"context_usage":      exampleContextUsage,
		"server_tool_blocks": exampleServerToolBlocks,
	}

	order := []string{
		"basic",
		"multi_turn",
		"interrupt",
		"manual_handling",
		"options",
		"bash_command",
		"error_handling",
		"mcp_status",
		"context_usage",
		"server_tool_blocks",
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./examples/streaming-mode <example_name>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		for _, name := range order {
			fmt.Printf("  %s\n", name)
		}
		os.Exit(0)
	}

	ctx := context.Background()
	name := os.Args[1]

	if name == "all" {
		for _, n := range order {
			examples[n](ctx)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println()
		}
		return
	}

	fn, ok := examples[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown example %q\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		for _, n := range order {
			fmt.Printf("  %s\n", n)
		}
		os.Exit(1)
	}

	fn(ctx)
}
