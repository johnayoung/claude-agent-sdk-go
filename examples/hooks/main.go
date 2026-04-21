// hooks demonstrates various hook patterns using the SDK's hooks package.
// Adapted from the Python SDK's hooks.py example.
//
// Usage:
//
//	go run ./examples/hooks              - List examples
//	go run ./examples/hooks all          - Run all examples
//	go run ./examples/hooks PreToolUse   - Run a specific example
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/hooks"
)

func displayMessage(msg claude.Message) {
	switch m := msg.(type) {
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

// checkBashCommand blocks bash commands containing forbidden patterns.
func checkBashCommand(_ context.Context, input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
	if input.ToolName != "Bash" {
		return &hooks.PreToolUseOutput{}, nil
	}

	command, _ := input.ToolInput["command"].(string)
	blockPatterns := []string{"foo.sh"}

	for _, pattern := range blockPatterns {
		if strings.Contains(command, pattern) {
			log.Printf("Blocked command: %s", command)
			return &hooks.PreToolUseOutput{
				Block:  true,
				Reason: fmt.Sprintf("Command contains invalid pattern: %s", pattern),
			}, nil
		}
	}

	return &hooks.PreToolUseOutput{}, nil
}

// addCustomInstructions injects context when a user prompt is submitted.
func addCustomInstructions(_ context.Context, _ *hooks.UserPromptSubmitInput) (*hooks.UserPromptSubmitOutput, error) {
	return &hooks.UserPromptSubmitOutput{
		SystemMessage: "My favorite color is hot pink",
	}, nil
}

// reviewToolOutput provides warnings when tool output contains errors.
func reviewToolOutput(_ context.Context, input *hooks.PostToolUseInput) (*hooks.PostToolUseOutput, error) {
	if input.IsError || strings.Contains(strings.ToLower(input.ToolOutput), "error") {
		log.Printf("Tool %s produced an error - consider checking command syntax", input.ToolName)
		return &hooks.PostToolUseOutput{
			SystemMessage:     "The command produced an error",
			Reason:            "Tool execution failed - consider checking the command syntax",
			AdditionalContext: "The command encountered an error. You may want to try a different approach.",
		}, nil
	}
	return &hooks.PostToolUseOutput{}, nil
}

// strictApprovalHook blocks writes to files with "important" in the name,
// and explicitly allows everything else.
func strictApprovalHook(_ context.Context, input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
	if input.ToolName == "Write" {
		filePath, _ := input.ToolInput["file_path"].(string)
		if strings.Contains(strings.ToLower(filePath), "important") {
			log.Printf("Blocked Write to: %s", filePath)
			return &hooks.PreToolUseOutput{
				Block:         true,
				Reason:        "Security policy blocks writes to important files",
				SystemMessage: "Write operation blocked by security policy",
			}, nil
		}
	}
	return &hooks.PreToolUseOutput{
		PermissionDecision: "allow",
		Reason:             "Tool passed security checks",
	}, nil
}

// stopOnErrorHook halts execution when critical errors appear in tool output.
func stopOnErrorHook(_ context.Context, input *hooks.PostToolUseInput) (*hooks.PostToolUseOutput, error) {
	if strings.Contains(strings.ToLower(input.ToolOutput), "critical") {
		log.Println("Critical error detected - stopping execution")
		f := false
		return &hooks.PostToolUseOutput{
			Continue:      &f,
			StopReason:    "Critical error detected in tool output - execution halted for safety",
			SystemMessage: "Execution stopped due to critical error",
		}, nil
	}
	return &hooks.PostToolUseOutput{}, nil
}

func examplePreToolUse(ctx context.Context) {
	fmt.Println("=== PreToolUse Example ===")
	fmt.Println("This example demonstrates how PreToolUse can block some bash commands but not others.")
	fmt.Println()

	h := hooks.New()
	h.OnPreToolUse("Bash", checkBashCommand)

	// Test 1: Command with forbidden pattern (will be blocked)
	fmt.Println("Test 1: Trying a command that our PreToolUse hook should block...")
	fmt.Println("User: Run the bash command: ./foo.sh --help")
	for msg, err := range claude.Query(ctx,
		"Run the bash command: ./foo.sh --help",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Test 2: Safe command that should work
	fmt.Println("Test 2: Trying a command that our PreToolUse hook should allow...")
	fmt.Println("User: Run the bash command: echo 'Hello from hooks example!'")
	for msg, err := range claude.Query(ctx,
		"Run the bash command: echo 'Hello from hooks example!'",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleUserPromptSubmit(ctx context.Context) {
	fmt.Println("=== UserPromptSubmit Example ===")
	fmt.Println("This example shows how a UserPromptSubmit hook can add context.")
	fmt.Println()

	h := hooks.New()
	h.OnUserPromptSubmit(addCustomInstructions)

	fmt.Println("User: What's my favorite color?")
	for msg, err := range claude.Query(ctx,
		"What's my favorite color?",
		claude.WithHooks(h),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func examplePostToolUse(ctx context.Context) {
	fmt.Println("=== PostToolUse Example ===")
	fmt.Println("This example shows how PostToolUse can review tool output.")
	fmt.Println()

	h := hooks.New()
	h.OnPostToolUse("Bash", reviewToolOutput)

	fmt.Println("User: Run a command that will produce an error: ls /nonexistent_directory")
	for msg, err := range claude.Query(ctx,
		"Run this command: ls /nonexistent_directory",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleDecisionFields(ctx context.Context) {
	fmt.Println("=== Permission Decision Example ===")
	fmt.Println("This example shows how to use PreToolUse to allow/deny with pattern matching.")
	fmt.Println()

	h := hooks.New()
	h.OnPreToolUse("Write", strictApprovalHook)

	// Test 1: Write to a file with "important" in the name (should be blocked)
	fmt.Println("Test 1: Trying to write to important_config.txt (should be blocked)...")
	fmt.Println("User: Write 'test' to important_config.txt")
	for msg, err := range claude.Query(ctx,
		"Write the text 'test data' to a file called important_config.txt",
		claude.WithHooks(h),
		claude.WithAllowedTools("Write", "Bash"),
		claude.WithModel("sonnet"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Test 2: Write to a regular file (should be approved)
	fmt.Println("Test 2: Trying to write to regular_file.txt (should be approved)...")
	fmt.Println("User: Write 'test' to regular_file.txt")
	for msg, err := range claude.Query(ctx,
		"Write the text 'test data' to a file called regular_file.txt",
		claude.WithHooks(h),
		claude.WithAllowedTools("Write", "Bash"),
		claude.WithModel("sonnet"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleContinueControl(ctx context.Context) {
	fmt.Println("=== Continue/Stop Control Example ===")
	fmt.Println("This example shows how to use Continue=false with StopReason to halt execution.")
	fmt.Println()

	h := hooks.New()
	h.OnPostToolUse("Bash", stopOnErrorHook)

	fmt.Println("User: Run a command that outputs 'CRITICAL ERROR'")
	for msg, err := range claude.Query(ctx,
		"Run this bash command: echo 'CRITICAL ERROR: system failure'",
		claude.WithHooks(h),
		claude.WithAllowedTools("Bash"),
	) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		displayMessage(msg)
	}

	fmt.Println()
}

func main() {
	examples := map[string]func(context.Context){
		"PreToolUse":       examplePreToolUse,
		"UserPromptSubmit": exampleUserPromptSubmit,
		"PostToolUse":      examplePostToolUse,
		"DecisionFields":   exampleDecisionFields,
		"ContinueControl":  exampleContinueControl,
	}

	order := []string{
		"PreToolUse",
		"UserPromptSubmit",
		"PostToolUse",
		"DecisionFields",
		"ContinueControl",
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./examples/hooks <example_name>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		for _, name := range order {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println("\nExample descriptions:")
		fmt.Println("  PreToolUse       - Block commands using PreToolUse hook")
		fmt.Println("  UserPromptSubmit - Add context at prompt submission")
		fmt.Println("  PostToolUse      - Review tool output and log warnings")
		fmt.Println("  DecisionFields   - Use PreToolUse to allow/deny writes by file path")
		fmt.Println("  ContinueControl  - Halt execution with continue=false and stopReason")
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
