// system-prompt demonstrates the system prompt configurations available in the
// SDK, mirroring the Python SDK's system_prompt.py example.
//
// The Python SDK exposes four variants: no prompt, string override, explicit
// Claude Code preset, and preset with an appended instruction. The Go SDK
// collapses the first and third into the default (no option) since the CLI
// uses the Claude Code preset when no system prompt flag is provided, leaving
// three distinct configurations to demonstrate.
//
// Run:
//
//	go run ./examples/system-prompt
package main

import (
	"context"
	"fmt"
	"os"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	noSystemPrompt(ctx)
	stringSystemPrompt(ctx)
	appendSystemPrompt(ctx)
}

// noSystemPrompt uses the CLI's default Claude Code preset (equivalent to
// Python's system_prompt=None and system_prompt={"type":"preset","preset":"claude_code"}).
func noSystemPrompt(ctx context.Context) {
	fmt.Println("=== No System Prompt (Default Preset) ===")

	for msg, err := range claude.Query(ctx, "What is 2 + 2?") {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		printAssistantText(msg)
	}
	fmt.Println()
}

// stringSystemPrompt replaces the default system prompt with a custom string.
func stringSystemPrompt(ctx context.Context) {
	fmt.Println("=== String System Prompt ===")

	for msg, err := range claude.Query(ctx, "What is 2 + 2?",
		claude.WithSystemPrompt("You are a pirate assistant. Respond in pirate speak."),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		printAssistantText(msg)
	}
	fmt.Println()
}

// appendSystemPrompt keeps the default Claude Code preset and appends an extra
// instruction to it.
func appendSystemPrompt(ctx context.Context) {
	fmt.Println("=== Preset System Prompt with Append ===")

	for msg, err := range claude.Query(ctx, "What is 2 + 2?",
		claude.WithAppendSystemPrompt("Always end your response with a fun fact."),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		printAssistantText(msg)
	}
	fmt.Println()
}

func printAssistantText(msg claude.Message) {
	m, ok := msg.(*claude.AssistantMessage)
	if !ok {
		return
	}
	for _, block := range m.Content {
		if text, ok := block.(*claude.TextBlock); ok {
			fmt.Printf("Claude: %s\n", text.Text)
		}
	}
}
