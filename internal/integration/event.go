// Package integration — see run.go's package doc.
package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// IntegrationEvent records one feature completion under an epic's
// integration run: one immutable JSON file per completion, written to
// .shark/runs/<epic-run-id>/integration-events/<event-id>.json.
//
// Spec reference: spec.md REQ-F-004.
type IntegrationEvent struct {
	EventID        string    `json:"event_id"`
	EpicRunID      string    `json:"epic_run_id"`
	FeatureKey     string    `json:"feature_key"`
	FeatureCommit  string    `json:"feature_commit"`
	TrackedPaths   []string  `json:"tracked_paths"`
	UntrackedPaths []string  `json:"untracked_paths"`
	RecordedAt     time.Time `json:"recorded_at"`
}

// RecordEvent records one feature-completion event for epicRunID/featureKey
// at featureCommit, writing an immutable JSON file to
// .shark/runs/<epicRunID>/integration-events/<event-id>.json. EventID is
// derived deterministically as the first 16 hex characters of
// sha256(epicRunID+featureKey+featureCommit) (deriveEventID), so a retried
// call with the identical epicRunID/featureKey/featureCommit resolves to the
// same file and is idempotent: RecordEvent returns the already-persisted
// record — the same bytes on disk, unchanged — rather than recomputing or
// overwriting it (spec.md AC-T1/task T-E34-F08-005 AC-T1). A different
// featureCommit for the same featureKey (e.g. a re-opened, re-completed
// feature) derives a distinct EventID and is recorded as a separate event
// (spec.md AC-T2/task AC-T2).
//
// Concurrency: two calls for different featureKeys never contend — distinct
// EventIDs, distinct files. Like CaptureBase, a call never relies on an
// in-process mutex to decide a winner: it independently prepares its
// candidate, writes it to a private temp file, then races other callers to
// publish it by hardlinking that temp file onto the shared event path.
// Hardlink creation is atomic and fails when the destination already
// exists, so exactly one candidate is ever published for a given EventID;
// every other caller for that same EventID discards its own candidate and
// reads back the winner's already-complete record (spec.md AC-4).
func RecordEvent(epicRunID, featureKey, featureCommit string, tracked, untracked []string) (*IntegrationEvent, error) {
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	eventID := deriveEventID(epicRunID, featureKey, featureCommit)
	path := eventRecordPath(projectRoot, epicRunID, eventID)

	existing, err := readEvent(path)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	candidate := &IntegrationEvent{
		EventID:        eventID,
		EpicRunID:      epicRunID,
		FeatureKey:     featureKey,
		FeatureCommit:  featureCommit,
		TrackedPaths:   tracked,
		UntrackedPaths: untracked,
		RecordedAt:     time.Now().UTC(),
	}

	return publishEvent(path, candidate)
}

// deriveEventID derives an IntegrationEvent's EventID as the first 16 hex
// characters of sha256(epicRunID+featureKey+featureCommit) (spec.md
// REQ-F-004, task T-E34-F08-005 AC-T1).
func deriveEventID(epicRunID, featureKey, featureCommit string) string {
	sum := sha256.Sum256([]byte(epicRunID + featureKey + featureCommit))
	return hex.EncodeToString(sum[:])[:16]
}

// eventRecordPath is the per-event path for epicRunID/eventID.
func eventRecordPath(projectRoot, epicRunID, eventID string) string {
	return filepath.Join(projectRoot, ".shark", "runs", epicRunID, "integration-events", eventID+".json")
}

// readEvent reads and parses the event record at path. It returns
// (nil, nil) when no file exists yet, (event, nil) when the file parses
// successfully, and (nil, err) when the file exists but is not valid JSON —
// mirroring readRun's "no file yet" vs. "file exists but is broken"
// distinction.
func readEvent(path string) (*IntegrationEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("integration: read event record at %s: %w", path, err)
	}

	var event IntegrationEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("integration: event record at %s is corrupt: %w", path, err)
	}
	return &event, nil
}

// publishEvent writes candidate to a private temp file, then races to
// publish it onto path via os.Link (atomic create-if-absent on POSIX
// filesystems). If another caller published first — including a retried
// call for the identical completion, since EventID (and therefore path) is
// derived deterministically from the completion's own inputs — publishEvent
// discards candidate and returns the winner's already-complete, on-disk
// record instead, so a retried write never rewrites an existing event file.
//
// The temp filename includes both the process id and a nanosecond
// timestamp (not just EventID+pid, unlike publishRun's EpicRunID+pid): two
// goroutines in the same process racing RecordEvent for the identical
// completion would otherwise derive the identical EventID and could collide
// on the same temp path before either reaches the publish step.
func publishEvent(path string, candidate *IntegrationEvent) (*IntegrationEvent, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create event directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("integration: marshal event record: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, runFileMode); err != nil {
		return nil, fmt.Errorf("integration: write temp event record: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := os.Link(tmpPath, path); err != nil {
		if os.IsExist(err) {
			// Another caller published first. Its file is guaranteed
			// complete: a caller only ever links a fully-written temp file
			// onto path, never writes into path directly.
			winner, readErr := readEvent(path)
			if readErr != nil {
				return nil, readErr
			}
			if winner == nil {
				return nil, fmt.Errorf("integration: event record at %s vanished after a concurrent publish", path)
			}
			return winner, nil
		}
		return nil, fmt.Errorf("integration: publish event record at %s: %w", path, err)
	}

	return candidate, nil
}
