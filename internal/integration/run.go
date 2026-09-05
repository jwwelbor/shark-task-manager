// Package integration holds epic-run state for E34-F08's integration-event
// log: the once-per-epic-run base-commit capture (this file), and — added by
// later tasks in this feature — the per-feature-completion event log and the
// accumulated integration candidate. This is deliberately not part of
// internal/sharkdata: it is run-scoped runtime state (like this repo's
// existing `.shark/runs/<run-id>/` convention), not bundle content.
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// runDirMode / runFileMode match this repo's existing run-artifact
// convention (internal/runner/transcript.go: 0o755 dir / 0o644 file).
const (
	runDirMode  os.FileMode = 0o755
	runFileMode os.FileMode = 0o644
)

// IntegrationRun captures the base commit for one epic's integration-review
// run. It is written once per epic by CaptureBase and never overwritten
// afterward: every later integration-event write reads BaseCommit from this
// same record, so the epic's cumulative diff view has one stable origin.
//
// Spec reference: spec.md REQ-F-004.
type IntegrationRun struct {
	EpicRunID  string    `json:"epic_run_id"`
	EpicKey    string    `json:"epic_key"`
	BaseCommit string    `json:"base_commit"`
	CreatedAt  time.Time `json:"created_at"`
}

// CorruptRunError indicates a run record exists on disk at Path but could
// not be parsed as JSON. CaptureBase returns this rather than silently
// creating a second run when the existing file is present but unreadable
// (spec.md AC-3 / task T-E34-F08-004 AC-T3).
type CorruptRunError struct {
	Path string
	Err  error
}

// Error implements the error interface.
func (e *CorruptRunError) Error() string {
	return fmt.Sprintf("integration: run record at %s is corrupt: %v", e.Path, e.Err)
}

// Unwrap exposes the underlying parse error for errors.Is/errors.As.
func (e *CorruptRunError) Unwrap() error {
	return e.Err
}

// CaptureBase captures the base commit for epicKey's integration run, once.
// The first call for a given epicKey creates and persists a new
// IntegrationRun — a fresh EpicRunID and the repository's current HEAD
// commit as BaseCommit. Every later call for the same epicKey, including a
// call racing the first one from a concurrent goroutine or process, is a
// no-op that returns the identical, already-persisted record; BaseCommit is
// never recomputed or overwritten (spec.md REQ-F-004, AC-3).
//
// Concurrency: CaptureBase never relies on an in-process mutex to decide a
// winner, so it is safe whether or not competing callers share a process.
// Each caller that finds no existing record independently prepares its own
// candidate, writes it to a private temp file, then races the others to
// publish it by hardlinking that temp file onto the shared run-record path.
// Hardlink creation is atomic and fails when the destination already exists,
// so exactly one candidate is ever published; every other caller discards
// its own candidate and reads back the winner's record, which — because a
// caller only ever links a fully-written temp file — is always complete.
//
// A run file that exists but fails to parse returns a *CorruptRunError
// rather than being treated as "no run yet" (AC-T3): CaptureBase must never
// paper over a broken record by creating a second one beside it.
func CaptureBase(epicKey string) (*IntegrationRun, error) {
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	path := runRecordPath(projectRoot, epicKey)

	existing, err := readRun(path)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	baseCommit, err := currentHeadCommit(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("integration: resolve base commit for epic %s: %w", epicKey, err)
	}

	candidate := &IntegrationRun{
		EpicRunID:  uuid.New().String(),
		EpicKey:    epicKey,
		BaseCommit: baseCommit,
		CreatedAt:  time.Now().UTC(),
	}

	return publishRun(path, candidate)
}

// runRecordPath is the per-epic run-record path, keyed by epicKey alone so
// CaptureBase can discover an existing run before any EpicRunID is known.
// Deliberately separate from `.shark/runs/<epic-run-id>/...`, which later
// integration-event/candidate files key by the EpicRunID this record hands
// out.
func runRecordPath(projectRoot, epicKey string) string {
	return filepath.Join(projectRoot, ".shark", "integration", epicKey, "run.json")
}

// readRun reads and parses the run record at path. It returns (nil, nil)
// when no file exists yet, (run, nil) when the file parses successfully,
// and (nil, *CorruptRunError) when the file exists but is not valid JSON —
// distinguishing "no run yet" from "a run exists but its file is broken" so
// CaptureBase never silently creates a second run on top of a broken one.
func readRun(path string) (*IntegrationRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("integration: read run record at %s: %w", path, err)
	}

	var run IntegrationRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, &CorruptRunError{Path: path, Err: err}
	}
	return &run, nil
}

// publishRun writes candidate to a private temp file, then races to publish
// it onto path via os.Link (atomic create-if-absent on POSIX filesystems).
// If another caller published first, publishRun discards candidate and
// returns the winner's already-complete record instead.
func publishRun(path string, candidate *IntegrationRun) (*IntegrationRun, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create run directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("integration: marshal run record: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.%s.%d.tmp", path, candidate.EpicRunID, os.Getpid())
	if err := os.WriteFile(tmpPath, data, runFileMode); err != nil {
		return nil, fmt.Errorf("integration: write temp run record: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := os.Link(tmpPath, path); err != nil {
		if os.IsExist(err) {
			// Another caller published first. Its file is guaranteed
			// complete: a caller only ever links a fully-written temp file
			// onto path, never writes into path directly.
			winner, readErr := readRun(path)
			if readErr != nil {
				return nil, readErr
			}
			if winner == nil {
				return nil, fmt.Errorf("integration: run record at %s vanished after a concurrent publish", path)
			}
			return winner, nil
		}
		return nil, fmt.Errorf("integration: publish run record at %s: %w", path, err)
	}

	return candidate, nil
}

// currentHeadCommit resolves the repository's current HEAD commit hash by
// running `git rev-parse HEAD` inside projectRoot. This becomes a new run's
// immutable BaseCommit — never recomputed by a later CaptureBase call for
// the same epic.
func currentHeadCommit(projectRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
