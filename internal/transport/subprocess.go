package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

var _ Transport = (*SubprocessTransport)(nil)

const scannerBufSize = 1 << 20 // 1 MiB — large messages from Claude can exceed the default 64 KiB

// Option configures a SubprocessTransport.
type Option func(*SubprocessTransport)

// WithCLIPath sets an explicit path to the claude CLI binary, bypassing PATH discovery.
func WithCLIPath(path string) Option {
	return func(t *SubprocessTransport) {
		t.cliPath = path
	}
}

// WithWorkingDir sets the working directory for the subprocess.
func WithWorkingDir(dir string) Option {
	return func(t *SubprocessTransport) {
		t.workingDir = dir
	}
}

// SubprocessTransport manages a Claude CLI subprocess via stdin/stdout JSON lines.
type SubprocessTransport struct {
	cliPath    string
	workingDir string
	args       []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

// New returns a SubprocessTransport that will invoke the claude CLI with the given args.
func New(args []string, opts ...Option) *SubprocessTransport {
	t := &SubprocessTransport{args: args}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Start discovers the CLI binary and launches the subprocess. Context cancellation
// will terminate the subprocess.
func (t *SubprocessTransport) Start(ctx context.Context) error {
	cliPath := t.cliPath
	if cliPath == "" {
		p, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude CLI not found in PATH (%s): %w", os.Getenv("PATH"), err)
		}
		cliPath = p
	}

	t.cmd = exec.CommandContext(ctx, cliPath, t.args...)
	if t.workingDir != "" {
		t.cmd.Dir = t.workingDir
	}

	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return err
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)
	t.stdout = scanner

	return t.cmd.Start()
}

// Send writes a single JSON line to the subprocess stdin.
func (t *SubprocessTransport) Send(line []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	buf := make([]byte, len(line)+1)
	copy(buf, line)
	buf[len(line)] = '\n'
	_, err := t.stdin.Write(buf)
	return err
}

// Receive blocks until the next JSON line is available from stdout.
// Returns io.EOF when the subprocess exits or stdout is closed.
func (t *SubprocessTransport) Receive() ([]byte, error) {
	if !t.stdout.Scan() {
		if err := t.stdout.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	raw := t.stdout.Bytes()
	line := make([]byte, len(raw))
	copy(line, raw)
	return line, nil
}

// Close closes stdin and waits for the subprocess to exit.
func (t *SubprocessTransport) Close() error {
	if t.stdin != nil {
		_ = t.stdin.Close()
	}
	if t.cmd != nil {
		return t.cmd.Wait()
	}
	return nil
}
