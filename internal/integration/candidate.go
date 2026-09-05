// Package integration — see run.go's package doc.
package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// maxUpdateCandidateAttempts bounds UpdateCandidate to one write attempt
// plus exactly one retry on a CAS conflict (spec.md Durable unresolved
// decision Q-F08-01 / task T-E34-F08-006 AC-T4): a conflict that persists
// through the retry is reported to the caller rather than looping a third
// time.
const maxUpdateCandidateAttempts = 2

// IntegrationCandidate holds the single, atomic accumulated view of an
// epic's integration run: every IntegrationEvent recorded so far, folded
// into one file at .shark/runs/<epic-run-id>/integration-candidate.json.
// Digest is a sha256 hex digest over the struct's canonical JSON with
// Digest itself excluded (matching I-03's existing guard-digest
// convention — no new hashing scheme), and is what UpdateCandidate's
// compare-and-swap write path checks against.
//
// Spec reference: spec.md REQ-F-004, task T-E34-F08-006 AC-T1.
type IntegrationCandidate struct {
	EpicRunID  string   `json:"epic_run_id"`
	BaseCommit string   `json:"base_commit"`
	HeadCommit string   `json:"head_commit"`
	EventIDs   []string `json:"event_ids"`
	Digest     string   `json:"digest"`
}

// CandidateConflictError indicates UpdateCandidate's compare-and-swap write
// was rejected because the on-disk candidate's digest no longer matched the
// digest the writer expected when it began building its update — another
// writer published a change in between. The writer already retried once
// (Q-F08-01's adopted policy) before this error is returned; the on-disk
// file is left byte-identical to whatever the winning writer last
// published (task T-E34-F08-006 AC-T3).
type CandidateConflictError struct {
	Path string
}

// Error implements the error interface.
func (e *CandidateConflictError) Error() string {
	return fmt.Sprintf("integration: candidate at %s changed concurrently (stale digest); retry exhausted", e.Path)
}

// updateCandidateTestHook, when non-nil, is invoked exactly once per
// UpdateCandidate attempt, between that attempt's initial read and its
// claim-then-publish write. Production code never sets it (nil by default,
// so it costs nothing outside tests). A test uses it to deterministically
// publish a concurrent change while the current attempt is "paused" at
// this point, forcing a real, reproducible CAS conflict instead of relying
// on goroutine-timing luck (test-plan.md TC-008's Caller-Path Contract: the
// hook exercises UpdateCandidate's real read-claim-write-retry path
// unmodified around the pause point — it is not a mock of production
// logic, and tests must not bypass it by hand-constructing a stale digest
// against a lower-level function).
var updateCandidateTestHook func()

// UpdateCandidate folds newEvent into epicRunID's IntegrationCandidate,
// creating the candidate file if it does not exist yet. The update is
// applied via a compare-and-swap: read the current candidate (if any),
// build the next candidate on top of it, and claim the transition away
// from the digest just read by atomically hardlinking the new content onto
// a claim file named for that expected-prior digest (attemptUpdateCandidate)
// — publish only if that claim is won. A lost claim means another writer
// changed the candidate in between; UpdateCandidate re-reads and retries
// exactly once before reporting a typed *CandidateConflictError (spec.md
// Durable unresolved decision Q-F08-01; task T-E34-F08-006 AC-T4).
//
// This claim-based construction is a deliberate strengthening of spec.md's
// prose description of the CAS ("read current file, verify its digest
// matches, write only if it matches"): a literal read-verify-then-rename
// implementation has a race window — two writers can both observe a
// matching digest before either has published, and the later rename
// silently discards the earlier one's event. Naming the claim by the prior
// digest closes that window (os.Link's atomic create-if-absent behavior is
// the actual compare-and-swap primitive, not the digest reread) while
// staying entirely lock-free and local to this file — it does not touch,
// duplicate, or anticipate the separate run-scoped registration-note lock
// T-E34-F08-012 owns in lock.go.
func UpdateCandidate(epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) {
	if newEvent == nil {
		return nil, fmt.Errorf("integration: UpdateCandidate requires a non-nil event")
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	path := candidatePath(projectRoot, epicRunID)

	var lastErr error
	for attempt := 0; attempt < maxUpdateCandidateAttempts; attempt++ {
		candidate, err := attemptUpdateCandidate(projectRoot, path, epicRunID, newEvent)
		if err == nil {
			return candidate, nil
		}
		var conflict *CandidateConflictError
		if !errors.As(err, &conflict) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

// attemptUpdateCandidate performs one read-build-claim-publish cycle of
// UpdateCandidate's compare-and-swap: it reads the current candidate (or
// nil if none exists), builds the next candidate on top of it, writes that
// next candidate to a temp file with an fsync before publish (task AC-T2),
// then claims the transition away from the digest it read at the start of
// this attempt by hardlinking that temp file onto
// claimPath(expectedDigest) — an atomic, create-if-absent operation
// (os.Link fails with EEXIST if the destination already exists, exactly
// like run.go/event.go's own publish pattern). Exactly one writer can ever
// win a given expectedDigest's claim; every other writer racing the same
// expectedDigest loses the Link and returns a typed *CandidateConflictError
// without ever touching the on-disk candidate (task AC-T3) — a genuine
// compare-and-swap, not a read-then-write race window (see UpdateCandidate's
// doc comment for why the earlier reread-then-rename shape lost an event
// under real concurrency and was replaced by this one).
func attemptUpdateCandidate(projectRoot, path, epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) {
	current, err := readCandidate(path)
	if err != nil {
		return nil, err
	}

	expectedDigest := ""
	if current != nil {
		expectedDigest = current.Digest
	}

	next, err := buildNextCandidate(projectRoot, current, epicRunID, newEvent)
	if err != nil {
		return nil, err
	}

	digest, err := computeDigest(*next)
	if err != nil {
		return nil, err
	}
	next.Digest = digest

	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("integration: marshal candidate: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create candidate directory %s: %w", dir, err)
	}
	claimsDir := candidateClaimsDir(path)
	if err := os.MkdirAll(claimsDir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create candidate claims directory %s: %w", claimsDir, err)
	}

	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, runFileMode)
	if err != nil {
		return nil, fmt.Errorf("integration: create temp candidate file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("integration: write temp candidate file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("integration: fsync temp candidate file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("integration: close temp candidate file: %w", err)
	}

	if updateCandidateTestHook != nil {
		updateCandidateTestHook()
	}

	// The compare-and-swap: claim the transition away from expectedDigest.
	// os.Link is atomic create-if-absent, so exactly one concurrent writer
	// racing the same expectedDigest ever wins this claim — no reread, no
	// window between "check" and "write" for a second writer to slip
	// through (claim files are retained forever, never removed, so a
	// slot can never be reopened for reuse by a stale writer).
	claimPath := filepath.Join(claimsDir, claimFileName(expectedDigest))
	if err := os.Link(tmpPath, claimPath); err != nil {
		_ = os.Remove(tmpPath)
		if os.IsExist(err) {
			return nil, &CandidateConflictError{Path: path}
		}
		return nil, fmt.Errorf("integration: claim candidate transition at %s: %w", claimPath, err)
	}

	// Only the claim's winner ever reaches this rename: a losing writer
	// returned above without renaming, so the file on disk after a
	// rejected attempt is exactly what the winning writer last published
	// (task AC-T3).
	if err := os.Rename(tmpPath, path); err != nil {
		return nil, fmt.Errorf("integration: publish candidate at %s: %w", path, err)
	}

	return next, nil
}

// candidateClaimsDir is where attemptUpdateCandidate's compare-and-swap
// claim files live, one per prior-digest transition ever attempted for the
// candidate at path.
func candidateClaimsDir(path string) string {
	return filepath.Join(filepath.Dir(path), "integration-candidate-claims")
}

// claimFileName names the claim file for a transition away from
// expectedDigest. "" (the transition from "no candidate yet") gets a fixed
// name rather than an empty filename.
func claimFileName(expectedDigest string) string {
	if expectedDigest == "" {
		return "from-empty.claim"
	}
	return expectedDigest + ".claim"
}

// buildNextCandidate builds the candidate that results from folding
// newEvent into current (nil for a brand-new candidate). EventIDs is the
// union of current's EventIDs and newEvent.EventID, sorted for a
// deterministic (and therefore digest-stable) on-disk order. BaseCommit is
// carried forward unchanged from current when present; for a brand-new
// candidate it is looked up from the epic's IntegrationRun record, if one
// has been captured yet — a candidate update never fails just because no
// run record exists (e.g. a caller driving RecordEvent/UpdateCandidate
// without ever calling CaptureBase for this run), since enforcing that
// invariant is integration_review's closure-check job (spec.md REQ-F-005),
// not this function's.
func buildNextCandidate(projectRoot string, current *IntegrationCandidate, epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) {
	next := &IntegrationCandidate{EpicRunID: epicRunID, HeadCommit: newEvent.FeatureCommit}

	ids := map[string]struct{}{newEvent.EventID: {}}
	if current != nil {
		next.BaseCommit = current.BaseCommit
		for _, id := range current.EventIDs {
			ids[id] = struct{}{}
		}
	} else {
		baseCommit, err := lookupBaseCommit(projectRoot, epicRunID)
		if err != nil {
			return nil, err
		}
		next.BaseCommit = baseCommit
	}

	eventIDs := make([]string, 0, len(ids))
	for id := range ids {
		eventIDs = append(eventIDs, id)
	}
	sort.Strings(eventIDs)
	next.EventIDs = eventIDs

	return next, nil
}

// lookupBaseCommit finds the BaseCommit CaptureBase recorded for the epic
// that owns epicRunID, by scanning the per-epic run records under
// .shark/integration/*/run.json for the one whose EpicRunID matches. It
// returns ("", nil) — not an error — when no matching run record exists
// yet or when a non-matching record on disk is unreadable/corrupt: this
// lookup is best-effort scaffolding for a brand-new candidate's BaseCommit,
// not the authoritative fail-closed check on a missing/broken base that
// spec.md's REQ-F-004/REQ-F-005 assign to integration_review.
func lookupBaseCommit(projectRoot, epicRunID string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(projectRoot, ".shark", "integration", "*", "run.json"))
	if err != nil {
		return "", fmt.Errorf("integration: scan run records: %w", err)
	}
	for _, m := range matches {
		run, err := readRun(m)
		if err != nil || run == nil {
			continue
		}
		if run.EpicRunID == epicRunID {
			return run.BaseCommit, nil
		}
	}
	return "", nil
}

// computeDigest computes candidate's digest: the sha256 hex digest of its
// canonical JSON serialization with Digest itself cleared first (task
// AC-T1). EventIDs is expected to already be sorted by the caller so the
// serialization — and therefore the digest — is stable regardless of the
// order events were folded in.
func computeDigest(candidate IntegrationCandidate) (string, error) {
	candidate.Digest = ""
	data, err := json.Marshal(candidate)
	if err != nil {
		return "", fmt.Errorf("integration: marshal candidate for digest: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// candidatePath is the per-epic-run candidate path.
func candidatePath(projectRoot, epicRunID string) string {
	return filepath.Join(projectRoot, ".shark", "runs", epicRunID, "integration-candidate.json")
}

// readCandidate reads and parses the candidate record at path. It returns
// (nil, nil) when no file exists yet, (candidate, nil) when the file parses
// successfully, and (nil, err) when the file exists but is not valid JSON —
// mirroring readRun/readEvent's "no file yet" vs. "file exists but is
// broken" distinction.
func readCandidate(path string) (*IntegrationCandidate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("integration: read candidate at %s: %w", path, err)
	}

	var candidate IntegrationCandidate
	if err := json.Unmarshal(data, &candidate); err != nil {
		return nil, fmt.Errorf("integration: candidate at %s is corrupt: %w", path, err)
	}
	return &candidate, nil
}
