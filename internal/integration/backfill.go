// Package integration — see run.go's package doc.
package integration

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// maxBackfillEvents bounds the size of a single Backfill call's events
// array (task T-E34-F08-007 AC-T4's "bounded array size" input check). No
// numeric bound is prescribed by spec.md itself; this is a generous ceiling
// for a one-time manual bootstrap of a single epic's already-completed
// features (an epic realistically has, at most, a few dozen features), not
// a value with any other significance.
const maxBackfillEvents = 500

// BackfillValidationError indicates Backfill rejected its base commit or
// events input during validation, before anything was written to disk
// (task T-E34-F08-007 AC-T4: validate fully before the first write). It is
// a distinct type — not a plain fmt.Errorf — so a caller can distinguish
// "your input was rejected" from an unexpected I/O failure, mirroring this
// package's existing typed-error convention (CorruptRunError,
// CandidateConflictError, RegistrationLockTimeoutError).
type BackfillValidationError struct {
	Reason string
}

// Error implements the error interface.
func (e *BackfillValidationError) Error() string {
	return fmt.Sprintf("integration: backfill input rejected: %s", e.Reason)
}

// Backfill registers an integration run for an epic that was already active
// before this feature shipped — i.e., one with no pre-execution
// IntegrationRun record — by performing the identical
// capture-then-append-then-register sequence CaptureBase/RecordEvent/
// UpdateCandidate/RegisterRun perform in steady state (spec.md "Key
// technical decisions" #2: "Backfill shares the steady-state write path").
// There is no second, independently-maintained write path: the run record
// is published via the same readRun/publishRun mechanism CaptureBase uses
// (backfillRun below), each event is written via the real, exported
// RecordEvent, each is folded into the candidate via the real, exported
// UpdateCandidate, and the closing epic reference note is written via the
// real, exported RegisterRun — the same functions the cascading `active`
// step and T-E34-F08-008's wiring call in steady state.
//
// base is the caller-supplied base commit for the epic's pre-existing
// history; unlike CaptureBase, Backfill never resolves this from the
// repository's current HEAD — an epic already active before this feature
// shipped predates this feature's own HEAD-at-first-dispatch capture point,
// so the caller (ultimately a human operator via `shark integration
// backfill --base=...`) must name it explicitly.
//
// Every one of base's reachability, events' internal consistency (no
// duplicate EventID, each entry's EventID matching the digest derived from
// its own epic run/feature key/commit), the bounded array-size limit, and
// the "not already registered" check completes — read-only, no write of
// any kind — before Backfill ever touches disk (AC-T4:
// validate-fully-before-first-write). dryRun stops right there. A
// non-dryRun call that passes every check writes exactly one
// IntegrationRun, one IntegrationEvent per input entry, one
// IntegrationCandidate, and one epic `--type=reference` note (AC-T1). A
// second Backfill attempt against an epic that already carries a
// registration note is rejected with zero mutation regardless of whether
// its inputs would otherwise match (AC-T2) — once registered, steady-state
// capture owns the epic, not a second backfill call.
//
// recorder/createdBy exist for the same reason RegisterRun itself takes
// them: this package never depends on internal/services (run.go's package
// doc), so the one DB-backed side effect a caller-side NoteRecorder
// implementation performs is injected here rather than looked up
// internally. This is a deliberate, minor widening of spec.md's literally
// listed `Backfill(epicKey, epicRunID, base string, events
// []IntegrationEvent, dryRun bool) (*IntegrationCandidate, error)` shape:
// without a recorder, Backfill would have no way to satisfy AC-6/AC-T1's
// "creates ... one epic reference note" requirement while still honoring
// run.go's own no-internal_services-dependency rule.
func Backfill(ctx context.Context, recorder NoteRecorder, epicKey, epicRunID, base string, events []IntegrationEvent, dryRun bool, createdBy string) (*IntegrationCandidate, error) {
	if recorder == nil {
		return nil, fmt.Errorf("integration: Backfill requires a non-nil NoteRecorder")
	}
	if strings.TrimSpace(epicKey) == "" {
		return nil, fmt.Errorf("integration: Backfill requires a non-empty epic key")
	}
	if strings.TrimSpace(epicRunID) == "" {
		return nil, fmt.Errorf("integration: Backfill requires a non-empty epic run ID")
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	// Validate fully before the first write (AC-T4). Every check in this
	// block is read-only: git rev-parse/cat-file, in-memory slice/map
	// inspection, and reads of already-existing files. None of them writes
	// or creates anything.
	if err := verifyCommitReachable(projectRoot, base); err != nil {
		return nil, err
	}
	if err := validateBackfillEvents(epicRunID, events); err != nil {
		return nil, err
	}
	if err := checkExistingRun(projectRoot, epicKey, epicRunID, base); err != nil {
		return nil, err
	}
	existingNote, _, err := existingRegistrationNote(ctx, recorder, epicKey)
	if err != nil {
		return nil, err
	}
	if existingNote != nil {
		return nil, &RegistrationConflictError{
			EpicKey: epicKey,
			Reason:  "epic already has a registered integration run; backfill only applies to an epic with no existing registration",
		}
	}

	if dryRun {
		return simulateBackfillCandidate(epicRunID, base, events)
	}

	run, err := backfillRun(projectRoot, epicKey, epicRunID, base)
	if err != nil {
		return nil, err
	}

	var (
		candidate *IntegrationCandidate
		lastEvent *IntegrationEvent
	)
	for i := range events {
		input := events[i]
		recorded, err := RecordEvent(epicRunID, input.FeatureKey, input.FeatureCommit, input.TrackedPaths, input.UntrackedPaths)
		if err != nil {
			return nil, fmt.Errorf("integration: backfill record event %d (%s): %w", i, input.FeatureKey, err)
		}
		candidate, err = UpdateCandidate(epicRunID, recorded)
		if err != nil {
			return nil, fmt.Errorf("integration: backfill update candidate for event %d (%s): %w", i, input.FeatureKey, err)
		}
		lastEvent = recorded
	}

	if _, err := RegisterRun(ctx, recorder, run, candidate, lastEvent, createdBy); err != nil {
		return nil, err
	}

	return candidate, nil
}

// verifyCommitReachable rejects base as a *BackfillValidationError unless it
// resolves to a real, reachable commit inside projectRoot's repository
// (`git cat-file -e <base>^{commit}`, which checks object existence and
// type without printing or otherwise materializing its content).
func verifyCommitReachable(projectRoot, base string) error {
	if strings.TrimSpace(base) == "" {
		return &BackfillValidationError{Reason: "--base must not be empty"}
	}
	cmd := exec.Command("git", "cat-file", "-e", base+"^{commit}")
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		return &BackfillValidationError{Reason: fmt.Sprintf("--base %q is not a reachable commit: %v", base, err)}
	}
	return nil
}

// validateBackfillEvents rejects events as a *BackfillValidationError when:
// it is empty or exceeds maxBackfillEvents (the bounded-array-size check),
// two entries share an EventID (the no-duplicate-EventID check), or an
// entry's own EventID does not match the digest deriveEventID computes from
// that same entry's EpicRunID-context, FeatureKey, and FeatureCommit fields
// (the digest/path-match-against-named-commits check, task AC-T4) — the
// same self-consistency an entry would need for RecordEvent to later derive
// the identical EventID Backfill is about to write it under.
func validateBackfillEvents(epicRunID string, events []IntegrationEvent) error {
	if len(events) == 0 {
		return &BackfillValidationError{Reason: "--events-file must contain at least one entry"}
	}
	if len(events) > maxBackfillEvents {
		return &BackfillValidationError{Reason: fmt.Sprintf("--events-file exceeds the bounded array size limit of %d entries (got %d)", maxBackfillEvents, len(events))}
	}

	seen := make(map[string]struct{}, len(events))
	for i, ev := range events {
		if _, dup := seen[ev.EventID]; dup {
			return &BackfillValidationError{Reason: fmt.Sprintf("--events-file entry %d: duplicate EventID %q", i, ev.EventID)}
		}
		seen[ev.EventID] = struct{}{}

		expected := deriveEventID(epicRunID, ev.FeatureKey, ev.FeatureCommit)
		if ev.EventID != expected {
			return &BackfillValidationError{Reason: fmt.Sprintf(
				"--events-file entry %d: EventID %q does not match the digest derived from its own feature key/commit under this epic run (expected %q)",
				i, ev.EventID, expected,
			)}
		}
	}
	return nil
}

// checkExistingRun rejects (epicKey, epicRunID, base) as a
// *RegistrationConflictError when epicKey already has a run record on disk
// whose EpicRunID or BaseCommit differs from this attempt's — a genuine
// conflict, not a retry of the same backfill. A matching existing record
// (identical EpicRunID and BaseCommit) is not rejected here: backfillRun
// below resolves that part of an exact retry idempotently, the same way
// CaptureBase resolves a repeated call for the same epic.
//
// Named gap: this only covers the run-record itself. A retry of a backfill
// that crashed *after* writing one or more events/candidate updates but
// *before* RegisterRun ever ran is not handled — UpdateCandidate's
// per-transition claim files (candidate.go) are retained forever, so
// re-folding an already-applied event reproduces a digest transition whose
// claim file already exists and returns a spurious *CandidateConflictError
// rather than resuming. T-E34-F08-014's crash-restart repair is scoped to
// RegisterRun's own fsync-then-note-insert step (run.go), not this
// earlier, multi-event backfill loop — fixing this would mean skipping
// already-applied events on retry, which is beyond this task's ACs. A
// crashed backfill must currently be retried against a clean epic (the
// run/event/candidate files removed by hand) rather than resumed in place.
func checkExistingRun(projectRoot, epicKey, epicRunID, base string) error {
	existing, err := readRun(runRecordPath(projectRoot, epicKey))
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	if existing.EpicRunID != epicRunID || existing.BaseCommit != base {
		return &RegistrationConflictError{
			EpicKey: epicKey,
			Reason: fmt.Sprintf(
				"epic %s already has a run record (run %s, base %s) that conflicts with this backfill attempt (run %s, base %s)",
				epicKey, existing.EpicRunID, existing.BaseCommit, epicRunID, base,
			),
		}
	}
	return nil
}

// backfillRun creates or resolves epicKey's IntegrationRun record for a
// Backfill call, sharing CaptureBase's exact publish mechanism
// (readRun/publishRun/runRecordPath) but supplying base explicitly instead
// of resolving it from the repository's current HEAD — Backfill's whole
// purpose is registering an epic whose base predates this feature's own
// HEAD-at-first-dispatch capture point, so it cannot use CaptureBase's
// git-HEAD-at-call-time semantics. The caller (Backfill) has already run
// checkExistingRun, so the only two outcomes reachable here are "publish a
// new record" and "return the already-matching record" (an exact-match
// retry of a backfill call that partially completed earlier).
func backfillRun(projectRoot, epicKey, epicRunID, base string) (*IntegrationRun, error) {
	path := runRecordPath(projectRoot, epicKey)

	existing, err := readRun(path)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	candidate := &IntegrationRun{
		EpicRunID:  epicRunID,
		EpicKey:    epicKey,
		BaseCommit: base,
		CreatedAt:  time.Now().UTC(),
	}
	return publishRun(path, candidate)
}

// simulateBackfillCandidate computes, without writing anything to disk, the
// IntegrationCandidate a non-dryRun Backfill call with the same
// (epicRunID, base, events) would produce — dryRun's "reports what it would
// create" requirement (AC-T1). It mirrors buildNextCandidate's folding rule
// (EventIDs is the sorted set of every entry's EventID; HeadCommit is the
// last-applied event's FeatureCommit) without touching any file, since a
// dry run must leave every sidecar and note byte-for-byte unchanged.
func simulateBackfillCandidate(epicRunID, base string, events []IntegrationEvent) (*IntegrationCandidate, error) {
	ids := make([]string, 0, len(events))
	var head string
	for _, ev := range events {
		ids = append(ids, ev.EventID)
		head = ev.FeatureCommit
	}
	sort.Strings(ids)

	candidate := &IntegrationCandidate{
		EpicRunID:  epicRunID,
		BaseCommit: base,
		HeadCommit: head,
		EventIDs:   ids,
	}
	digest, err := computeDigest(*candidate)
	if err != nil {
		return nil, err
	}
	candidate.Digest = digest
	return candidate, nil
}
