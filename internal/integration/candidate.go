// Package integration — see run.go's package doc.
package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/projectroot"
)

// maxUpdateCandidateAttempts bounds UpdateCandidate to one write attempt
// plus exactly one retry on a CAS conflict (spec.md Durable unresolved
// decision Q-F08-01 / task T-E34-F08-006 AC-T4): a conflict that persists
// through the retry is reported to the caller rather than looping a third
// time.
const maxUpdateCandidateAttempts = 2

// currentPathDigestSchemaVersion marks candidates whose path-digest
// inventory was computed by the current candidate writer. A zero value is
// deliberately reserved for pre-B076 candidates, where the fields did not
// exist and an omitted empty map cannot prove that a clean inventory was
// captured.
const currentPathDigestSchemaVersion = 1

// candidateClaimPollInterval / candidateClaimTimeout bound the wait for a
// competing writer that has claimed, but not yet published, the transition
// away from a candidate digest. That state is not a stale-write conflict: the
// current candidate still has the expected digest, so consuming the caller's
// one retry would make two legitimate concurrent completions flaky. A claim
// that remains in-flight until the timeout is still reported as the typed
// conflict rather than being retried forever.
var (
	candidateClaimPollInterval = 5 * time.Millisecond
	candidateClaimTimeout      = 10 * time.Second
)

// IntegrationCandidate holds the single, atomic accumulated view of an
// epic's integration run: every IntegrationEvent recorded so far, folded
// into one file at .shark/runs/<epic-run-id>/integration-candidate.json.
// TrackedPathDigests and UntrackedPathDigests record a sha256 hex digest,
// keyed by path, for every dirty tracked file and untracked candidate path
// found in the real working tree at build time (architecture.md "Epic
// integration candidate identity"; task T-E34-F08-016 AC-T1) — closing the
// REQ-F-004/REQ-F-005 inventory gap the integration review (T-E34-F08-010)
// reads for its full-diff/path-digest inventory check. Digest is a sha256
// hex digest over the struct's canonical JSON with Digest itself excluded
// (matching I-03's existing guard-digest convention — no new hashing
// scheme), and is what UpdateCandidate's compare-and-swap write path checks
// against; it covers TrackedPathDigests/UntrackedPathDigests too, so a
// working-tree change between builds changes the digest like any other
// field.
//
// Spec reference: spec.md REQ-F-004, task T-E34-F08-006 AC-T1.
type IntegrationCandidate struct {
	EpicRunID               string            `json:"epic_run_id"`
	BaseCommit              string            `json:"base_commit"`
	HeadCommit              string            `json:"head_commit"`
	EventIDs                []string          `json:"event_ids"`
	PathDigestSchemaVersion int               `json:"path_digest_schema_version,omitempty"`
	TrackedPathDigests      map[string]string `json:"tracked_path_digests,omitempty"`
	UntrackedPathDigests    map[string]string `json:"untracked_path_digests,omitempty"`
	Digest                  string            `json:"digest"`
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

// candidateClaimAcquiredTestHook and candidateClaimContendedTestHook let the
// concurrency regression deterministically hold a winning claim while a
// second real UpdateCandidate call observes it. Both are nil in production.
var (
	candidateClaimAcquiredTestHook  func()
	candidateClaimContendedTestHook func()
)

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
func UpdateCandidate(ctx context.Context, epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) {
	if newEvent == nil {
		return nil, fmt.Errorf("integration: UpdateCandidate requires a non-nil event")
	}

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	path := candidatePath(projectRoot, epicRunID)

	var lastErr error
	for attempt := 0; attempt < maxUpdateCandidateAttempts; attempt++ {
		candidate, err := attemptUpdateCandidate(ctx, projectRoot, path, epicRunID, newEvent)
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

// GetCandidate returns epicRunID's current on-disk IntegrationCandidate, or
// (nil, nil) if no candidate has been published yet — the read-only
// counterpart to UpdateCandidate's compare-and-swap write, mirroring
// GetRun's relationship to CaptureBase (run.go).
//
// Added by T-E34-F08-008's UAT rework round 2, Finding 2:
// FeatureService.recordIntegrationEventForTerminalTransition
// (internal/services/feature_service.go) uses this to decide — from the
// actual persisted candidate state rather than a wall-clock heuristic on
// the just-recorded event's RecordedAt — whether a completion retry still
// needs to fold its event, and whether a first-head candidate is still
// missing its registration note. See that function's doc comment for the
// full state-based reconciliation this enables.
func GetCandidate(ctx context.Context, epicRunID string) (*IntegrationCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("integration: get candidate: %w", err)
	}
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}
	candidate, _, err := readCandidate(candidatePath(projectRoot, epicRunID))
	return candidate, err
}

// MigrateCandidatePathDigestsAuthorized is the migration entrypoint for a
// candidate transition is claimed and immediately before archival/publication,
// caller that owns an external lease. A non-dry-run call must provide the
// authorizer; this keeps durable migration writes behind an explicit lease
// boundary instead of exposing an unauthenticated compatibility path.
func MigrateCandidatePathDigestsAuthorized(ctx context.Context, epicKey, epicRunID string, dryRun bool, authorize func(context.Context) error) (*IntegrationCandidate, error) {
	if !dryRun && authorize == nil {
		return nil, fmt.Errorf("integration: migrate candidate: authorization callback is required for writes")
	}
	return migrateCandidatePathDigests(ctx, epicKey, epicRunID, dryRun, authorize)
}

func migrateCandidatePathDigests(ctx context.Context, epicKey, epicRunID string, dryRun bool, authorize func(context.Context) error) (*IntegrationCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("integration: migrate candidate: %w", err)
	}
	epicKey = strings.TrimSpace(epicKey)
	if err := models.ValidateEpicKey(epicKey); err != nil {
		return nil, fmt.Errorf("integration: migrate candidate: invalid epic key %q: %w", epicKey, err)
	}
	epicRunID = strings.TrimSpace(epicRunID)
	if err := ValidateEpicRunID(epicRunID); err != nil {
		return nil, fmt.Errorf("integration: migrate candidate: %w", err)
	}
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: migrate candidate: resolve project root: %w", err)
	}

	for attempt := 0; attempt < maxUpdateCandidateAttempts; attempt++ {
		candidate, err := attemptMigrateCandidate(ctx, projectRoot, epicKey, epicRunID, dryRun, authorize)
		if err == nil {
			return candidate, nil
		}
		var conflict *CandidateConflictError
		if !errors.As(err, &conflict) || dryRun {
			return nil, err
		}
	}
	return nil, &CandidateConflictError{Path: candidatePath(projectRoot, epicRunID)}
}

func attemptMigrateCandidate(ctx context.Context, projectRoot, epicKey, epicRunID string, dryRun bool, authorize func(context.Context) error) (*IntegrationCandidate, error) {
	_, current, currentBytes, err := readLegacyCandidateForMigration(projectRoot, epicKey, epicRunID)
	if err != nil {
		return nil, err
	}
	if current.PathDigestSchemaVersion == currentPathDigestSchemaVersion {
		return current, nil
	}
	if current.PathDigestSchemaVersion > currentPathDigestSchemaVersion {
		return nil, fmt.Errorf("integration: migrate candidate: unsupported path-digest schema version %d", current.PathDigestSchemaVersion)
	}

	tracked, untracked, err := computeDirtyPathDigests(ctx, projectRoot)
	if err != nil {
		return nil, err
	}
	next := *current
	next.PathDigestSchemaVersion = currentPathDigestSchemaVersion
	next.TrackedPathDigests = tracked
	next.UntrackedPathDigests = untracked
	next.Digest, err = computeDigest(next)
	if err != nil {
		return nil, err
	}
	if dryRun {
		return &next, nil
	}
	if err := publishMigratedCandidate(ctx, candidatePath(projectRoot, epicRunID), current, currentBytes, &next, authorize); err != nil {
		return nil, err
	}
	return &next, nil
}

func readLegacyCandidateForMigration(projectRoot, epicKey, epicRunID string) (*IntegrationRun, *IntegrationCandidate, []byte, error) {
	run, err := readMigrationRun(projectRoot, epicKey, epicRunID)
	if err != nil {
		return nil, nil, nil, err
	}
	current, currentBytes, err := readMigrationCandidate(projectRoot, run, epicRunID)
	if err != nil {
		return nil, nil, nil, err
	}
	computed, err := computeDigest(*current)
	if err != nil {
		return nil, nil, nil, err
	}
	if computed != current.Digest {
		return nil, nil, nil, fmt.Errorf("integration: migrate candidate: candidate digest is invalid")
	}
	return run, current, currentBytes, nil
}

func readMigrationRun(projectRoot, epicKey, epicRunID string) (*IntegrationRun, error) {
	run, err := readRun(runRecordPath(projectRoot, epicKey))
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("integration: migrate candidate: no integration run registered for %s", epicKey)
	}
	if run.EpicKey != epicKey || run.EpicRunID != epicRunID {
		return nil, fmt.Errorf("integration: migrate candidate: registered run identity does not match epic %s and run %s", epicKey, epicRunID)
	}
	return run, nil
}

func readMigrationCandidate(projectRoot string, run *IntegrationRun, epicRunID string) (*IntegrationCandidate, []byte, error) {
	current, currentBytes, err := readCandidate(candidatePath(projectRoot, epicRunID))
	if err != nil {
		return nil, nil, err
	}
	if current == nil {
		return nil, nil, fmt.Errorf("integration: migrate candidate: no candidate exists for run %s", epicRunID)
	}
	if current.EpicRunID != epicRunID || current.BaseCommit != run.BaseCommit {
		return nil, nil, fmt.Errorf("integration: migrate candidate: candidate identity does not match registered run %s", epicRunID)
	}
	return current, currentBytes, nil
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
//
// Winning the claim is necessary but not sufficient to spend it permanently:
// a deferred rollback immediately below the claim removes it (and its
// backing temp file) unless this attempt goes on to actually publish (the
// final rename succeeds). This is what keeps a later, genuinely transient
// failure — archiving or the rename itself — from permanently occupying
// expectedDigest's transition slot: the only way any future attempt (this
// call's own retry, or an independent later call) could ever lose the same
// claim's os.Link is if this attempt actually finished publishing under it.
func attemptUpdateCandidate(ctx context.Context, projectRoot, path, epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) {
	current, currentBytes, err := readCandidate(path)
	if err != nil {
		return nil, err
	}

	expectedDigest := ""
	if current != nil {
		expectedDigest = current.Digest
		// EventIDs is the durable set-union identity for a fold. Retrying an
		// event that is already present must not rebuild it with that event as
		// HeadCommit: a newer event may have advanced the head meanwhile.
		if candidateHasEvent(current, newEvent.EventID) {
			return current, nil
		}
	}

	next, err := buildNextCandidate(ctx, projectRoot, current, epicRunID, newEvent)
	if err != nil {
		return nil, err
	}

	digest, err := computeDigest(*next)
	if err != nil {
		return nil, err
	}
	next.Digest = digest
	// Folding an event already present in the candidate is idempotent. Do
	// not claim or publish an unchanged transition: a claim keyed by the
	// unchanged digest would permanently occupy the slot needed by the next
	// distinct event.
	if current != nil && next.Digest == current.Digest {
		return current, nil
	}

	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("integration: marshal candidate: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return nil, fmt.Errorf("integration: create candidate directory %s: %w", dir, err)
	}
	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := writeTempFileSynced(tmpPath, data); err != nil {
		return nil, fmt.Errorf("integration: write synced temp candidate file: %w", err)
	}

	if updateCandidateTestHook != nil {
		updateCandidateTestHook()
	}

	if err := publishCandidateTransition(ctx, path, current, currentBytes, tmpPath, expectedDigest, nil); err != nil {
		return nil, err
	}

	return next, nil
}

// publishMigratedCandidate publishes a replacement for an existing legacy
// candidate through the same claim/archive/rename sequence as ordinary
// candidate updates. Keeping migration on this path preserves the archived
// predecessor and prevents a stale migration from overwriting a newer head.
func publishMigratedCandidate(ctx context.Context, path string, current *IntegrationCandidate, currentBytes []byte, next *IntegrationCandidate, authorize func(context.Context) error) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("integration: marshal migrated candidate: %w", err)
	}
	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := writeTempFileSynced(tmpPath, data); err != nil {
		return fmt.Errorf("integration: write migrated candidate: %w", err)
	}
	return publishCandidateTransition(ctx, path, current, currentBytes, tmpPath, current.Digest, authorize)
}

// publishCandidateTransition owns the shared claim, authorization, archive,
// and rename protocol used by ordinary updates and legacy migration. Keeping
// these steps in one path prevents a new publisher from accidentally losing
// rollback or archived-head guarantees.
func publishCandidateTransition(ctx context.Context, path string, current *IntegrationCandidate, currentBytes []byte, tmpPath, expectedDigest string, authorize func(context.Context) error) error {
	claimsDir := candidateClaimsDir(path)
	if err := os.MkdirAll(claimsDir, runDirMode); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("integration: create candidate claims directory %s: %w", claimsDir, err)
	}
	claimPath := filepath.Join(claimsDir, claimFileName(expectedDigest))
	if err := claimCandidateTransition(ctx, path, claimPath, tmpPath, expectedDigest); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = os.Remove(claimPath)
			_ = os.Remove(tmpPath)
		}
	}()
	if authorize != nil {
		if err := authorize(ctx); err != nil {
			return err
		}
	}
	if current != nil {
		if authorize != nil {
			if err := authorize(ctx); err != nil {
				return err
			}
		}
		if err := archiveCandidateHead(path, current.Digest, currentBytes); err != nil {
			return err
		}
		if archiveTestHook != nil {
			archiveTestHook()
		}
	}
	if authorize != nil {
		if err := authorize(ctx); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("integration: publish candidate at %s: %w", path, err)
	}
	published = true
	return nil
}

// claimCandidateTransition atomically claims the expected prior digest, or
// waits for the existing claimant to either publish or release its claim.
func claimCandidateTransition(ctx context.Context, path, claimPath, tmpPath, expectedDigest string) error {
	for {
		if err := os.Link(tmpPath, claimPath); err == nil {
			if candidateClaimAcquiredTestHook != nil {
				candidateClaimAcquiredTestHook()
			}
			return nil
		} else if !os.IsExist(err) {
			return fmt.Errorf("integration: claim candidate transition at %s: %w", claimPath, err)
		}
		if candidateClaimContendedTestHook != nil {
			candidateClaimContendedTestHook()
		}
		advanced, err := waitForCandidateClaim(ctx, path, claimPath, expectedDigest)
		if err != nil {
			return err
		}
		if advanced {
			return &CandidateConflictError{Path: path}
		}
	}
}

type candidateClaimState uint8

const (
	candidateClaimPending candidateClaimState = iota
	candidateClaimReleased
	candidateClaimAdvanced
)

// waitForCandidateClaim distinguishes a live claimant from a completed
// transition. It returns advanced when the current candidate has moved away
// from expectedDigest; released means an unpublished claimant removed its
// claim and the caller may try to claim the unchanged transition itself.
func waitForCandidateClaim(ctx context.Context, path, claimPath, expectedDigest string) (advanced bool, err error) {
	deadline := time.Now().Add(candidateClaimTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return false, fmt.Errorf("integration: wait for candidate claim at %s: %w", claimPath, err)
		}

		state, err := inspectCandidateClaim(path, claimPath, expectedDigest)
		if err != nil {
			return false, err
		}
		if state == candidateClaimReleased {
			return false, nil
		}
		if state == candidateClaimAdvanced {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, &CandidateConflictError{Path: path}
		}

		select {
		case <-ctx.Done():
			return false, fmt.Errorf("integration: wait for candidate claim at %s: %w", claimPath, ctx.Err())
		case <-time.After(candidateClaimPollInterval):
		}
	}
}

// inspectCandidateClaim classifies one observation without making state
// changes. A same-file claim is a completed legacy no-op and stays
// fail-closed for TD-212's crash-safe recovery work.
func inspectCandidateClaim(path, claimPath, expectedDigest string) (candidateClaimState, error) {
	current, _, err := readCandidate(path)
	if err != nil {
		return candidateClaimPending, err
	}
	if candidateDigest(current) != expectedDigest {
		return candidateClaimAdvanced, nil
	}
	claimInfo, err := os.Stat(claimPath)
	if os.IsNotExist(err) {
		return candidateClaimReleased, nil
	}
	if err != nil {
		return candidateClaimPending, fmt.Errorf("integration: stat candidate claim at %s: %w", claimPath, err)
	}
	candidateInfo, err := os.Stat(path)
	if err == nil && os.SameFile(candidateInfo, claimInfo) {
		return candidateClaimAdvanced, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return candidateClaimPending, fmt.Errorf("integration: stat candidate at %s: %w", path, err)
	}
	return candidateClaimPending, nil
}

func candidateDigest(candidate *IntegrationCandidate) string {
	if candidate == nil {
		return ""
	}
	return candidate.Digest
}

func candidateHasEvent(candidate *IntegrationCandidate, eventID string) bool {
	for _, id := range candidate.EventIDs {
		if id == eventID {
			return true
		}
	}
	return false
}

// archiveTestHook, when non-nil, is invoked exactly once per
// attemptUpdateCandidate call that has a prior head to archive, after that
// head's bytes have been durably written to
// integration-heads/<current.Digest>.json but before the rename that
// replaces the live candidate at path. Production code never sets it (nil
// by default). A single-threaded before/after comparison of final state
// alone cannot distinguish "archived, then replaced" from "replaced, then
// archived" — test-plan.md TC-016's Caller-Path Contract calls this out
// explicitly, so this hook lets a test observe the live candidate file
// still holding the prior (not-yet-replaced) bytes at the moment the
// archived copy already exists on disk.
var archiveTestHook func()

// candidateHeadsDir is where attemptUpdateCandidate's archived prior
// candidate heads live, one immutable file per historical head, named by
// that head's own Digest (its "record digest") and retained forever so a
// later reader can recompute any prior_record_digest purely from these
// bytes (architecture.md "Epic integration candidate identity"; task
// T-E34-F08-012 AC-T1).
func candidateHeadsDir(path string) string {
	return filepath.Join(filepath.Dir(path), "integration-heads")
}

// archiveCandidateHead durably writes headBytes — the exact, unmodified
// bytes read from disk for the head being replaced — to
// integration-heads/<headDigest>.json, fsynced before the caller's rename
// replaces the live candidate. Idempotent: a headDigest's content never
// changes, so if the archive file already exists (e.g. this exact
// transition was archived by an earlier attempt), archiving is a no-op
// rather than an error.
func archiveCandidateHead(candidatePath, headDigest string, headBytes []byte) error {
	dir := candidateHeadsDir(candidatePath)
	if err := os.MkdirAll(dir, runDirMode); err != nil {
		return fmt.Errorf("integration: create archived-head directory %s: %w", dir, err)
	}
	archivePath := filepath.Join(dir, headDigest+".json")

	tmpPath := fmt.Sprintf("%s.%d-%d.tmp", archivePath, os.Getpid(), time.Now().UnixNano())
	if err := writeTempFileSynced(tmpPath, headBytes); err != nil {
		return fmt.Errorf("integration: write synced temp archived-head file: %w", err)
	}
	defer func() {
		// The publish result is authoritative; this is best-effort temp cleanup.
		_ = os.Remove(tmpPath)
	}()

	if err := os.Link(tmpPath, archivePath); err != nil {
		if os.IsExist(err) {
			archivedBytes, readErr := os.ReadFile(archivePath)
			if readErr != nil {
				return fmt.Errorf("integration: verify existing archived head at %s: %w", archivePath, readErr)
			}
			if !bytes.Equal(archivedBytes, headBytes) {
				return fmt.Errorf("integration: existing archived head at %s does not match digest %s", archivePath, headDigest)
			}
			return nil
		}
		return fmt.Errorf("integration: publish archived head at %s: %w", archivePath, err)
	}
	return nil
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
func buildNextCandidate(ctx context.Context, projectRoot string, current *IntegrationCandidate, epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) {
	next := &IntegrationCandidate{
		EpicRunID:               epicRunID,
		HeadCommit:              newEvent.FeatureCommit,
		PathDigestSchemaVersion: currentPathDigestSchemaVersion,
	}

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

	tracked, untracked, err := computeDirtyPathDigests(ctx, projectRoot)
	if err != nil {
		return nil, err
	}
	next.TrackedPathDigests = tracked
	next.UntrackedPathDigests = untracked

	return next, nil
}

// sharkRuntimeDirPrefix is the shark runtime state directory this
// package's own event/candidate/lock files live under. computeDirtyPathDigests
// excludes it (and everything under it) from the dirty/untracked inventory:
// it is the review's own bookkeeping, not a candidate path under review,
// and including it would make the candidate self-referential — its own
// digest would depend on the very event/candidate files that computing it
// is in the middle of writing (temp-file names include a wall-clock
// timestamp, so an untracked ".shark/..." entry would make the digest
// non-deterministic run to run).
const sharkRuntimeDirPrefix = ".shark/"

// computeDirtyPathDigests walks projectRoot's real git working tree (`git
// status --porcelain=v1 -z`) and returns a sha256 hex digest, keyed by
// path, for every dirty tracked path (staged or worktree-modified relative
// to HEAD) and every untracked path currently sitting in the tree
// (test-plan.md TC-019's Caller-Path Contract: a real filesystem/git walk,
// never a hand-built file list — synthesizing digests from a caller-
// supplied path list would let the review's full-diff inventory silently
// diverge from what the working tree actually contains). A path whose
// content cannot be read (e.g. deleted in the worktree so there is nothing
// left to digest, or a directory entry from a fully-untracked directory)
// is omitted rather than erroring.
func computeDirtyPathDigests(ctx context.Context, projectRoot string) (tracked, untracked map[string]string, err error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v1", "-z", "-uall")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("integration: git status at %s: %w", projectRoot, err)
	}

	tracked = map[string]string{}
	untracked = map[string]string{}

	trimmed := strings.TrimSuffix(string(out), "\x00")
	if trimmed == "" {
		return tracked, untracked, nil
	}
	entries := strings.Split(trimmed, "\x00")

	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 {
			continue
		}
		x, y := entry[0], entry[1]
		path := entry[3:]

		// A rename/copy entry carries a second, null-separated "original
		// path" element immediately after it in `-z` output. This
		// candidate's inventory only records paths as they exist now, so
		// the original-path element is consumed (skipped) rather than
		// misparsed as its own status entry.
		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			i++
		}

		if path == strings.TrimSuffix(sharkRuntimeDirPrefix, "/") || strings.HasPrefix(path, sharkRuntimeDirPrefix) {
			continue
		}

		digest, ok, derr := digestWorkingTreeFile(projectRoot, path)
		if derr != nil {
			return nil, nil, derr
		}
		if !ok {
			continue
		}

		if x == '?' && y == '?' {
			untracked[path] = digest
		} else {
			tracked[path] = digest
		}
	}

	return tracked, untracked, nil
}

// digestWorkingTreeFile reads relPath (relative to projectRoot) from the
// working tree and returns its sha256 hex digest. ok is false, with a nil
// error, specifically when the path no longer exists on disk (a deleted
// path git status still reports) or is a directory (an untracked directory
// git status can report as one collapsed entry) — neither has file content
// to digest, and that is not itself a failure to compute one.
func digestWorkingTreeFile(projectRoot, relPath string) (digest string, ok bool, err error) {
	full := filepath.Join(projectRoot, relPath)
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("integration: stat dirty path %s: %w", relPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", false, fmt.Errorf("integration: refuse to hash symlink dirty path %s", relPath)
	}
	if info.IsDir() {
		return "", false, nil
	}

	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("integration: read dirty path %s: %w", relPath, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), true, nil
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

// readCandidate reads and parses the candidate record at path, returning
// the parsed candidate alongside the exact raw bytes read from disk (so a
// caller that needs to archive those bytes unchanged — attemptUpdateCandidate's
// AC-T1 archival step — never has to re-read the file and risk a second,
// possibly-different read racing a concurrent writer). It returns
// (nil, nil, nil) when no file exists yet, (candidate, data, nil) when the
// file parses successfully, and (nil, nil, err) when the file exists but is
// not valid JSON — mirroring readRun/readEvent's "no file yet" vs. "file
// exists but is broken" distinction.
func readCandidate(path string) (*IntegrationCandidate, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("integration: read candidate at %s: %w", path, err)
	}

	var candidate IntegrationCandidate
	if err := json.Unmarshal(data, &candidate); err != nil {
		return nil, nil, fmt.Errorf("integration: candidate at %s is corrupt: %w", path, err)
	}
	return &candidate, data, nil
}
