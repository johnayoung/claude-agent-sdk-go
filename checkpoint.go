package claude

import (
	"context"
	"errors"
)

// ErrCheckpointingDisabled is returned when RewindFiles is called without file checkpointing enabled.
var ErrCheckpointingDisabled = errors.New("claude: file checkpointing is not enabled")

// ErrNoSession is returned when a control method is called before a session has been established.
var ErrNoSession = errors.New("claude: no session ID available for rewind")

// RewindFiles sends a rewind_files control request to the CLI, restoring file state
// to the checkpoint associated with the given user message ID. File checkpointing
// must be enabled via WithFileCheckpointing and a session must already be established.
func (c *Client) RewindFiles(ctx context.Context, userMessageID string) error {
	c.mu.Lock()
	if !c.opts.FileCheckpointing {
		c.mu.Unlock()
		return ErrCheckpointingDisabled
	}
	c.mu.Unlock()

	tr, err := c.startControlTransport(ctx)
	if err != nil {
		return err
	}
	defer tr.Close()

	reqID := generateControlRequestID("rewind")
	payload := rewindFilesPayload{
		Subtype:       "rewind_files",
		UserMessageID: userMessageID,
	}
	if err := sendControlRequest(tr, reqID, payload); err != nil {
		return err
	}

	_, err = awaitControlResponseTyped[struct{}](ctx, tr, reqID, "rewind")
	return err
}

type rewindFilesPayload struct {
	Subtype       string `json:"subtype"`
	UserMessageID string `json:"user_message_id"`
}
