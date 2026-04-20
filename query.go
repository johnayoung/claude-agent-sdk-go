package claude

import (
	"context"
	"io"
	"iter"
	"os"
	"strconv"

	"github.com/johnayoung/claude-agent-sdk-go/agent"
	"github.com/johnayoung/claude-agent-sdk-go/internal/parse"
	"github.com/johnayoung/claude-agent-sdk-go/transport"
)

// Query launches the Claude CLI with prompt and streams back messages.
// The iterator terminates after a ResultMessage or on context cancellation.
// Transport is always cleaned up, even on early break.
func Query(ctx context.Context, prompt string, opts ...agent.Option) iter.Seq2[agent.Message, error] {
	return func(yield func(agent.Message, error) bool) {
		o := agent.NewOptions(opts)

		var tr agent.Transporter
		if o.Transport != nil {
			tr = o.Transport
		} else {
			if o.CLIPath == "" {
				yield(nil, &agent.CLINotFoundError{SearchPath: os.Getenv("PATH")})
				return
			}
			args := buildQueryArgs(prompt, o)
			tr = transport.New(args, transport.WithCLIPath(o.CLIPath))
		}

		if err := tr.Start(ctx); err != nil {
			yield(nil, err)
			return
		}
		defer tr.Close()

		for {
			line, err := tr.Receive()
			if err != nil {
				if err == io.EOF {
					if ctx.Err() != nil {
						yield(nil, ctx.Err())
					}
					return
				}
				yield(nil, err)
				return
			}

			msg, parseErr := parse.ParseLine(line)
			if parseErr != nil {
				if !yield(nil, parseErr) {
					return
				}
				continue
			}

			if !yield(msg, nil) {
				return
			}

			if _, ok := msg.(*agent.ResultMessage); ok {
				return
			}
		}
	}
}

func buildQueryArgs(prompt string, o *agent.Options) []string {
	args := []string{
		"--print", prompt,
		"--output-format", "stream-json",
	}
	if o.SystemPrompt != "" {
		args = append(args, "--system-prompt", o.SystemPrompt)
	}
	if o.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(o.MaxTurns))
	}
	if o.PermissionMode != "" && o.PermissionMode != agent.PermissionModeDefault {
		args = append(args, "--permission-mode", string(o.PermissionMode))
	}
	return args
}
