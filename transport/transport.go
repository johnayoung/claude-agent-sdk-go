package transport

import "context"

// Transport defines the communication layer between the SDK and the Claude CLI.
type Transport interface {
	// Start initializes the transport with the given context. Context cancellation
	// must terminate any underlying subprocess or connection.
	Start(ctx context.Context) error

	// Send writes a single JSON line to the transport.
	Send(line []byte) error

	// Receive blocks until a JSON line is available and returns it.
	// Returns io.EOF when the transport is closed or the subprocess exits.
	Receive() ([]byte, error)

	// Close shuts down the transport gracefully.
	Close() error
}
