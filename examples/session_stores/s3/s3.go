// Package s3 is an S3-backed claude.SessionStore reference adapter.
//
// This is a reference implementation — copy it into your own project and
// adapt as needed. It mirrors the Python SDK's s3_session_store.py.
//
// Transcripts are stored as JSONL part files:
//
//	s3://{bucket}/{prefix}{project_key}/{session_id}/part-{epochMs13}-{rand6}.jsonl
//
// Each Append writes a new part; Load lists, sorts, and concatenates them.
// The 13-digit zero-padded epoch-ms prefix means lexical key order equals
// chronological order. A per-instance monotonic millisecond counter orders
// same-instance same-ms appends; the random hex suffix disambiguates
// concurrent instances.
//
// Retention: this adapter never deletes objects on its own. Configure an
// S3 lifecycle policy on the bucket/prefix to expire transcripts
// according to your compliance requirements.
package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	claude "github.com/johnayoung/claude-agent-sdk-go"
)

// Bounded-parallel GetObject so Load isn't N*RTT serial.
const loadConcurrency = 16

var partMtimeRE = regexp.MustCompile(`/part-(\d{13})-[0-9a-f]{6}\.jsonl$`)

// SessionStore is an S3-backed claude.SessionStore.
//
// Append = PutObject of a new part file; Load = ListObjectsV2 + sort +
// bounded-parallel GetObject + concat. Monotonic ms orders same-instance
// same-ms appends; the rand suffix disambiguates instances.
type SessionStore struct {
	client *awss3.Client
	bucket string
	prefix string

	clockMu sync.Mutex
	lastMs  int64
}

// New creates an S3-backed SessionStore.
//
// client must be a pre-configured *s3.Client (caller controls region,
// credentials, endpoint, etc.). Non-empty prefix is normalized to end
// in exactly one "/"; an empty prefix produces no leading separator.
func New(client *awss3.Client, bucket, prefix string) *SessionStore {
	p := strings.TrimRight(prefix, "/")
	if p != "" {
		p += "/"
	}
	return &SessionStore{client: client, bucket: bucket, prefix: p}
}

// keyPrefix returns the directory prefix for a session (or subpath).
// Always ends in "/".
func (s *SessionStore) keyPrefix(key claude.SessionKey) string {
	parts := []string{key.ProjectKey, key.SessionID}
	if key.Subpath != "" {
		parts = append(parts, key.Subpath)
	}
	return s.prefix + strings.Join(parts, "/") + "/"
}

// projectPrefix returns the directory prefix for a project. Always ends in "/".
func (s *SessionStore) projectPrefix(projectKey string) string {
	return s.prefix + projectKey + "/"
}

// nextPartName produces a fixed-width epoch-ms + random-hex filename so
// lexical sort equals chronological order. lastMs + 1 keeps same-instance
// same-ms appends deterministic; the random suffix disambiguates instances.
func (s *SessionStore) nextPartName() (string, error) {
	s.clockMu.Lock()
	defer s.clockMu.Unlock()
	now := time.Now().UnixMilli()
	if now <= s.lastMs {
		now = s.lastMs + 1
	}
	s.lastMs = now
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("part-%013d-%s.jsonl", now, hex.EncodeToString(buf)), nil
}

// Append mirrors a batch of transcript entries as a single JSONL part file.
func (s *SessionStore) Append(ctx context.Context, key claude.SessionKey, entries []claude.SessionStoreEntry) error {
	if len(entries) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for i, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("s3 session store: marshal entry %d: %w", i, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	name, err := s.nextPartName()
	if err != nil {
		return fmt.Errorf("s3 session store: generate part name: %w", err)
	}
	objectKey := s.keyPrefix(key) + name
	_, err = s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/x-ndjson"),
	})
	return err
}

// Load returns all entries for a key in append order. Returns nil, nil
// when the session is unknown. Malformed lines are logged and skipped.
func (s *SessionStore) Load(ctx context.Context, key claude.SessionKey) ([]claude.SessionStoreEntry, error) {
	prefix := s.keyPrefix(key)

	// List part files directly under this prefix only. Without Delimiter,
	// S3 recurses into subpaths (e.g. subagents/*), so a main-transcript
	// Load would mix in subagent entries — diverging from
	// InMemorySessionStore's exact-key semantics and corrupting resume.
	var keys []string
	var contToken *string
	for {
		result, err := s.client.ListObjectsV2(ctx, &awss3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			Delimiter:         aws.String("/"),
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, err
		}
		for _, obj := range result.Contents {
			if obj.Key == nil {
				continue
			}
			k := *obj.Key
			// Guard against S3-compatibles that ignore Delimiter: keep
			// only direct children.
			if strings.Contains(k[len(prefix):], "/") {
				continue
			}
			keys = append(keys, k)
		}
		if result.NextContinuationToken == nil {
			break
		}
		contToken = result.NextContinuationToken
	}
	if len(keys) == 0 {
		return nil, nil
	}
	// 13-digit epochMs prefix is fixed-width, so lexical == chronological.
	sort.Strings(keys)

	// Bounded-parallel GetObject (serial is N*RTT); preserves sorted-key
	// order via slot-indexed result slice.
	bodies := make([]string, len(keys))
	errs := make([]error, len(keys))
	sem := make(chan struct{}, loadConcurrency)
	var wg sync.WaitGroup
	for i, k := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, k string) {
			defer wg.Done()
			defer func() { <-sem }()
			out, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    aws.String(k),
			})
			if err != nil {
				errs[i] = err
				return
			}
			defer out.Body.Close()
			data, err := io.ReadAll(out.Body)
			if err != nil {
				errs[i] = err
				return
			}
			bodies[i] = string(data)
		}(i, k)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	var all []claude.SessionStoreEntry
	for _, body := range bodies {
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			var e claude.SessionStoreEntry
			if err := json.Unmarshal([]byte(trimmed), &e); err != nil {
				log.Printf("s3 session store: skipping malformed line: %v", err)
				continue
			}
			all = append(all, e)
		}
	}
	if len(all) == 0 {
		return nil, nil
	}
	return all, nil
}

// ListSessions enumerates main-transcript sessions for a project. mtime is
// derived from each part filename's 13-digit epochMs prefix; CommonPrefixes
// carry no timestamp so we list Contents directly.
func (s *SessionStore) ListSessions(ctx context.Context, projectKey string) ([]claude.SessionStoreListEntry, error) {
	prefix := s.projectPrefix(projectKey)
	sessions := make(map[string]int64)
	var contToken *string
	for {
		result, err := s.client.ListObjectsV2(ctx, &awss3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, err
		}
		for _, obj := range result.Contents {
			if obj.Key == nil {
				continue
			}
			k := *obj.Key
			rest := k[len(prefix):]
			slash := strings.Index(rest, "/")
			if slash == -1 {
				continue
			}
			// Main-transcript parts only (one level under session_id);
			// deeper keys are subagent parts and would surface phantom
			// session IDs / skew mtime.
			if strings.Contains(rest[slash+1:], "/") {
				continue
			}
			sid := rest[:slash]
			var mtime int64
			if m := partMtimeRE.FindStringSubmatch(k); m != nil {
				if n, err := strconv.ParseInt(m[1], 10, 64); err == nil {
					mtime = n
				}
			} else if obj.LastModified != nil {
				mtime = obj.LastModified.UnixMilli()
			}
			if mtime > sessions[sid] {
				sessions[sid] = mtime
			}
		}
		if result.NextContinuationToken == nil {
			break
		}
		contToken = result.NextContinuationToken
	}
	out := make([]claude.SessionStoreListEntry, 0, len(sessions))
	for sid, mtime := range sessions {
		out = append(out, claude.SessionStoreListEntry{SessionID: sid, Mtime: mtime})
	}
	return out, nil
}

// Delete removes a session (or a specific subpath).
//
// Deleting a main session cascades to every subpath under it; deleting a
// specific subpath is exact-key only.
func (s *SessionStore) Delete(ctx context.Context, key claude.SessionKey) error {
	prefix := s.keyPrefix(key)
	directOnly := key.Subpath != ""
	var contToken *string
	for {
		input := &awss3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: contToken,
		}
		if directOnly {
			input.Delimiter = aws.String("/")
		}
		result, err := s.client.ListObjectsV2(ctx, input)
		if err != nil {
			return err
		}
		var toDelete []s3types.ObjectIdentifier
		for _, obj := range result.Contents {
			if obj.Key == nil {
				continue
			}
			k := *obj.Key
			if directOnly && strings.Contains(k[len(prefix):], "/") {
				continue
			}
			toDelete = append(toDelete, s3types.ObjectIdentifier{Key: aws.String(k)})
		}
		if len(toDelete) > 0 {
			delResult, err := s.client.DeleteObjects(ctx, &awss3.DeleteObjectsInput{
				Bucket: aws.String(s.bucket),
				Delete: &s3types.Delete{Objects: toDelete, Quiet: aws.Bool(true)},
			})
			if err != nil {
				return err
			}
			if len(delResult.Errors) > 0 {
				parts := make([]string, 0, len(delResult.Errors))
				for _, e := range delResult.Errors {
					key := ""
					if e.Key != nil {
						key = *e.Key
					}
					code := ""
					if e.Code != nil {
						code = *e.Code
					}
					parts = append(parts, fmt.Sprintf("%s: %s", key, code))
				}
				return fmt.Errorf("s3 session store: delete failed for %d object(s): %s",
					len(delResult.Errors), strings.Join(parts, ", "))
			}
		}
		if result.NextContinuationToken == nil {
			break
		}
		contToken = result.NextContinuationToken
	}
	return nil
}

// ListSubkeys returns all non-empty subpaths under a session.
func (s *SessionStore) ListSubkeys(ctx context.Context, key claude.SessionKey) ([]string, error) {
	prefix := s.keyPrefix(claude.SessionKey{
		ProjectKey: key.ProjectKey,
		SessionID:  key.SessionID,
	})
	subkeys := make(map[string]struct{})
	var contToken *string
	for {
		result, err := s.client.ListObjectsV2(ctx, &awss3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, err
		}
		for _, obj := range result.Contents {
			if obj.Key == nil {
				continue
			}
			k := *obj.Key
			rel := k[len(prefix):]
			parts := strings.Split(rel, "/")
			if len(parts) < 2 {
				continue
			}
			subpath := strings.Join(parts[:len(parts)-1], "/")
			if subpath == "" {
				continue
			}
			// Defense-in-depth: drop ".."/"."/"" segments.
			valid := true
			for _, seg := range strings.Split(subpath, "/") {
				if seg == "" || seg == "." || seg == ".." {
					valid = false
					break
				}
			}
			if valid {
				subkeys[subpath] = struct{}{}
			}
		}
		if result.NextContinuationToken == nil {
			break
		}
		contToken = result.NextContinuationToken
	}
	out := make([]string, 0, len(subkeys))
	for sp := range subkeys {
		out = append(out, sp)
	}
	return out, nil
}

var _ claude.SessionStore = (*SessionStore)(nil)
