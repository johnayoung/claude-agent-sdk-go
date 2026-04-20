package agent

import "fmt"

// CLINotFoundError indicates the Claude CLI binary could not be found.
type CLINotFoundError struct {
	SearchPath string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("claude CLI not found in PATH: %s", e.SearchPath)
}

// ProcessError indicates the Claude CLI process exited with an error.
type ProcessError struct {
	ExitCode int
	Stderr   string
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("claude process exited with code %d: %s", e.ExitCode, e.Stderr)
}

// JSONDecodeError indicates a failure to parse a JSON line from the CLI output.
type JSONDecodeError struct {
	RawLine string
	Err     error
}

func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("failed to decode JSON: %v", e.Err)
}

func (e *JSONDecodeError) Unwrap() error {
	return e.Err
}

// MessageParseError indicates a failure to parse a decoded JSON object into a typed message.
type MessageParseError struct {
	TypeField string
	RawJSON   string
	Err       error
}

func (e *MessageParseError) Error() string {
	return fmt.Sprintf("failed to parse message of type %q: %v", e.TypeField, e.Err)
}

func (e *MessageParseError) Unwrap() error {
	return e.Err
}
