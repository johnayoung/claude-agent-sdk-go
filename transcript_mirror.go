package claude

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// transcriptMirrorMessage is an internal message type that is intercepted by the
// query loops and never yielded to callers.
type transcriptMirrorMessage struct {
	FilePath string            `json:"filePath"`
	Entries  []json.RawMessage `json:"entries"`
}

func (m *transcriptMirrorMessage) MessageType() string { return "transcript_mirror" }

func parseTranscriptMirror(line []byte) (Message, error) {
	var msg transcriptMirrorMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, &MessageParseError{TypeField: "transcript_mirror", RawJSON: string(line), Err: err}
	}
	return &msg, nil
}

// filePathToSessionKey derives a SessionKey from a transcript mirror file path.
//
// When projectsDir is provided, uses the Python SDK's path-based approach:
//   - Main transcript:    <projectsDir>/<projectKey>/<sessionID>.jsonl
//   - Subagent transcript: <projectsDir>/<projectKey>/<sessionID>/subagents/<agent>.jsonl
//
// When projectsDir is empty or the path is not relative to it, falls back to a
// hash-based key using the normalized path.
func filePathToSessionKey(filePath, projectsDir string) *SessionKey {
	if projectsDir != "" {
		if key := filePathToSessionKeyFromDir(filePath, projectsDir); key != nil {
			return key
		}
	}

	normalized := filepath.Clean(filePath)
	base := filepath.Base(normalized)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	h := sha256.Sum256([]byte(normalized))
	pathHash := hex.EncodeToString(h[:])[:16]

	return &SessionKey{
		SessionID: fmt.Sprintf("%s-%s", nameWithoutExt, pathHash),
	}
}

func filePathToSessionKeyFromDir(filePath, projectsDir string) *SessionKey {
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return nil
	}
	absDir, err := filepath.Abs(projectsDir)
	if err != nil {
		return nil
	}

	rel, err := filepath.Rel(absDir, absFile)
	if err != nil {
		return nil
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil
	}

	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return nil
	}

	projectKey := parts[0]
	second := parts[1]

	// Main transcript: <projectKey>/<sessionID>.jsonl
	if len(parts) == 2 && strings.HasSuffix(second, ".jsonl") {
		return &SessionKey{
			ProjectKey: projectKey,
			SessionID:  strings.TrimSuffix(second, ".jsonl"),
		}
	}

	// Subagent transcript: <projectKey>/<sessionID>/subagents/.../<agent>.jsonl
	if len(parts) >= 4 {
		subpathParts := parts[2:]
		last := subpathParts[len(subpathParts)-1]
		if strings.HasSuffix(last, ".jsonl") {
			subpathParts[len(subpathParts)-1] = strings.TrimSuffix(last, ".jsonl")
		}
		return &SessionKey{
			ProjectKey: projectKey,
			SessionID:  second,
			Subpath:    strings.Join(subpathParts, "/"),
		}
	}

	return nil
}

// handleTranscriptMirror persists a transcript mirror message to the session store.
// Returns a *MirrorErrorMessage if persistence fails, or nil on success.
func handleTranscriptMirror(ctx context.Context, store SessionStore, projectsDir string, msg *transcriptMirrorMessage) *MirrorErrorMessage {
	if len(msg.Entries) == 0 {
		return nil
	}

	key := filePathToSessionKey(msg.FilePath, projectsDir)

	entries := make([]SessionStoreEntry, 0, len(msg.Entries))
	for _, raw := range msg.Entries {
		var entry SessionStoreEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			entries = append(entries, SessionStoreEntry{})
			continue
		}
		entries = append(entries, entry)
	}

	if err := store.Append(ctx, *key, entries); err != nil {
		return &MirrorErrorMessage{
			SystemMessage: SystemMessage{
				Subtype: "mirror_error",
				Data:    map[string]any{"type": "system", "subtype": "mirror_error"},
			},
			Key:   key,
			Error: fmt.Sprintf("failed to mirror transcript for %s: %v", msg.FilePath, err),
		}
	}
	return nil
}
