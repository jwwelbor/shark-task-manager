// Package integration — see run.go's package doc.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/projectroot"
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

// epicRunIDPattern is an allowlist for epicRunID: letters, digits,
// underscore, and hyphen only, starting with a letter or digit, up to 128
// characters. Deliberately excludes "." (and therefore "..") and path
// separators ("/", "\") — epicRunID is joined directly onto a filesystem
// path by eventRecordPath/candidatePath/registrationLockPath/
// replacementRecordPath (event.go, candidate.go, lock.go, history.go) and
// interpolated directly into a temp-file name by publishRun (run.go:
// fmt.Sprintf("%s.%s.%d.tmp", path, candidate.EpicRunID, os.Getpid())) — a
// second site the defect-class sweep found matching the same shape (a
// string-built path/filename, not only filepath.Join). An allowlist here
// (mirroring internal/models/validation.go's regex-allowlist convention for
// other entity keys) is what keeps a caller-supplied value like
// "../../etc/passwd" from ever reaching any of those, rather than a
// blocklist that has to anticipate every traversal spelling. A generated
// UUID (CaptureBase's uuid.New().String(), e.g.
// "550e8400-e29b-41d4-a716-446655440000" — lowercase hex and hyphens only)
// always satisfies this pattern, so CaptureBase's own steady-state flow is
// unaffected by this allowlist.
var epicRunIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

// ValidateEpicRunID rejects epicRunID as a *BackfillValidationError unless
// it matches epicRunIDPattern. Exported so every caller-facing entry point
// that accepts an externally-supplied epic run ID validates it the same
// way: Backfill calls this below before epicRunID ever reaches a
// filesystem path, and the CLI layer (`shark integration backfill
// --epic-run-id=...`, T-E34-F08-015, internal/cli/commands/
// integration_cmd.go) should call this same function on the raw
// --epic-run-id flag value before invoking Backfill at all, rather than
// maintaining a second, independent character allowlist for the identical
// defect class (code-review kickback on T-E34-F08-007/T-E34-F08-015: "an
// externally-supplied identifier is used to construct a filesystem path
// with no format allowlist, enabling path traversal outside
// .shark/runs/").
//
// T-E34-F08-015's own <epic-key> arg needs the equivalent fix, but through
// a different, already-existing function: integration_cmd.go's current gate
// (`DetectEntityType(epicKey) != "epic"`, integration_cmd.go:96) accepts a
// *slugged* epic key (e.g. "E07-user-management") because DetectEntityType
// only rejects empty "-"-separated segments, not their characters — so a
// value like "E07-..-..-..-secrets" still passes it. Backfill below now
// requires a bare epic key (models.ValidateEpicKey, "^E\\d{2}$"), so
// T-E34-F08-015 should replace that CLI-layer gate with
// models.ValidateEpicKey(epicKey) on the normalized (upper-cased,
// trimmed) arg — calling models.ValidateEpicKey directly, the same
// validator Backfill calls just below, not a new wrapper in this package.
func ValidateEpicRunID(epicRunID string) error {
	if !epicRunIDPattern.MatchString(epicRunID) {
		return &BackfillValidationError{Reason: fmt.Sprintf(
			"--epic-run-id %q is invalid: must match %s (letters, digits, underscore, hyphen only; no path separators or \".\")",
			epicRunID, epicRunIDPattern.String(),
		)}
	}
	return nil
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
// non-dryRun call through BackfillAuthorized that passes every check writes exactly one
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
// run.go's own no-internal_services-dependency rule. Durable writes additionally
// require BackfillAuthorized so the caller's lease remains explicit.
func Backfill(ctx context.Context, recorder NoteRecorder, epicKey, epicRunID, base string, events []IntegrationEvent, dryRun bool, createdBy string) (*IntegrationCandidate, error) {
	if !dryRun {
		return nil, fmt.Errorf("integration: backfill: authorization callback is required for writes")
	}
	return backfill(ctx, recorder, epicKey, epicRunID, base, events, dryRun, createdBy, nil)
}

// BackfillAuthorized performs backfill with an explicit lease authorization
// callback. The callback is checked immediately before each write phase so a
// long validation or event fold cannot silently continue after the caller's
// session has been lost.
func BackfillAuthorized(ctx context.Context, recorder NoteRecorder, epicKey, epicRunID, base string, events []IntegrationEvent, dryRun bool, createdBy string, authorize func(context.Context) error) (*IntegrationCandidate, error) {
	if !dryRun && authorize == nil {
		return nil, fmt.Errorf("integration: backfill: authorization callback is required for writes")
	}
	return backfill(ctx, recorder, epicKey, epicRunID, base, events, dryRun, createdBy, authorize)
}

func backfill(ctx context.Context, recorder NoteRecorder, epicKey, epicRunID, base string, events []IntegrationEvent, dryRun bool, createdBy string, authorize func(context.Context) error) (*IntegrationCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("integration: backfill: %w", err)
	}
	if recorder == nil {
		return nil, fmt.Errorf("integration: Backfill requires a non-nil NoteRecorder")
	}
	// epicKey and epicRunID are both externally supplied (CLI args) and both
	// flow, unmodified, into filesystem paths this package builds
	// (runRecordPath keys by epicKey; eventRecordPath/candidatePath/
	// registrationLockPath key by epicRunID) — so both are validated against
	// a strict allowlist before either is used for anything, per this
	// function's own validate-fully-before-first-write contract (AC-T4).
	// Reuses internal/models.ValidateEpicKey (this package already imports
	// internal/models for note types) rather than inventing a second,
	// independently-maintained epic-key pattern — a bare epic key like "E07"
	// is required, so a slugged form such as "E07-user-management" (whose
	// slug segment has no character allowlist at the CLI layer) is rejected
	// here rather than reaching runRecordPath unsanitized.
	epicKey = strings.TrimSpace(epicKey)
	if err := models.ValidateEpicKey(epicKey); err != nil {
		return nil, &BackfillValidationError{Reason: fmt.Sprintf("epic key %q is invalid: %v", epicKey, err)}
	}
	epicRunID = strings.TrimSpace(epicRunID)
	if err := ValidateEpicRunID(epicRunID); err != nil {
		return nil, err
	}

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	// Validate fully before the first write (AC-T4). Every check in this
	// block is read-only: git rev-parse/cat-file, in-memory slice/map
	// inspection, and reads of already-existing files. None of them writes
	// or creates anything.
	if err := verifyCommitReachable(ctx, projectRoot, base); err != nil {
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
		return simulateBackfillCandidate(ctx, projectRoot, epicRunID, base, events)
	}
	// The ctx.Err() check must come before ensureBackfillManifest: that call
	// can publish the recovery manifest to disk (publishBackfillManifest), a
	// genuine write, so it must not run past the point this function
	// documents as "before first write" (AC-T4).
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("integration: backfill before first write: %w", err)
	}
	if authorize != nil {
		if err := authorize(ctx); err != nil {
			return nil, fmt.Errorf("integration: backfill before first write: %w", err)
		}
	}
	if err := reconcileBackfillManifestAndCandidate(projectRoot, epicKey, epicRunID, base, events); err != nil {
		return nil, err
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
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("integration: backfill before event %d: %w", i, err)
		}
		if authorize != nil {
			if err := authorize(ctx); err != nil {
				return nil, fmt.Errorf("integration: backfill before event %d: %w", i, err)
			}
		}
		input := events[i]
		recorded, err := RecordEvent(epicRunID, input.FeatureKey, input.FeatureCommit, input.TrackedPaths, input.UntrackedPaths)
		if err != nil {
			return nil, fmt.Errorf("integration: backfill record event %d (%s): %w", i, input.FeatureKey, err)
		}
		if !sameBackfillEvent(recorded, input, epicRunID) {
			return nil, &RegistrationConflictError{EpicKey: epicKey, Reason: fmt.Sprintf("retained event %s does not match the authorized backfill manifest", input.EventID)}
		}
		candidate, err = UpdateCandidate(ctx, epicRunID, recorded)
		if err != nil {
			return nil, fmt.Errorf("integration: backfill update candidate for event %d (%s): %w", i, input.FeatureKey, err)
		}
		lastEvent = recorded
	}
	if err := validateCompletedBackfillCandidate(epicKey, candidate, events); err != nil {
		return nil, err
	}
	if authorize != nil {
		if err := authorize(ctx); err != nil {
			return nil, fmt.Errorf("integration: backfill before registration: %w", err)
		}
	}

	if _, err := RegisterRun(ctx, recorder, run, candidate, lastEvent, createdBy); err != nil {
		return nil, err
	}

	return candidate, nil
}

type backfillManifest struct {
	EpicKey    string             `json:"epic_key"`
	EpicRunID  string             `json:"epic_run_id"`
	BaseCommit string             `json:"base_commit"`
	Events     []IntegrationEvent `json:"events"`
	Digest     string             `json:"digest"`
}

func backfillManifestPath(projectRoot, epicKey string) string {
	return filepath.Join(projectRoot, ".shark", "integration", epicKey, "backfill-manifest.json")
}

// reconcileBackfillManifestAndCandidate publishes (or verifies) the recovery
// manifest for a resumed Backfill attempt, then validates any retained
// candidate against it. Both steps' error branches are contained here so
// Backfill's own body does not carry their decision count directly.
func reconcileBackfillManifestAndCandidate(projectRoot, epicKey, epicRunID, base string, events []IntegrationEvent) error {
	if err := ensureBackfillManifest(projectRoot, epicKey, epicRunID, base, events); err != nil {
		return err
	}
	return validateRetainedBackfillCandidate(projectRoot, epicKey, epicRunID, base, events)
}

func ensureBackfillManifest(projectRoot, epicKey, epicRunID, base string, events []IntegrationEvent) error {
	path := backfillManifestPath(projectRoot, epicKey)
	want := backfillManifest{EpicKey: epicKey, EpicRunID: epicRunID, BaseCommit: base, Events: normalizeBackfillEvents(events)}
	digest, err := digestBackfillManifest(want)
	if err != nil {
		return err
	}
	want.Digest = digest
	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		return fmt.Errorf("integration: marshal backfill manifest: %w", err)
	}
	present, err := matchingBackfillManifest(path, data, epicKey)
	if err != nil {
		return err
	}
	if present {
		return nil
	}
	if stateExists(projectRoot, epicKey, epicRunID) {
		return &RegistrationConflictError{EpicKey: epicKey, Reason: "legacy partial backfill state has no manifest"}
	}
	return publishBackfillManifest(path, data, projectRoot, epicKey, epicRunID, base, events)
}

func matchingBackfillManifest(path string, want []byte, epicKey string) (bool, error) {
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("integration: read backfill manifest: %w", err)
	}
	if string(existing) != string(want) {
		return false, &RegistrationConflictError{EpicKey: epicKey, Reason: "backfill manifest does not match retry input"}
	}
	return true, nil
}

func publishBackfillManifest(path string, data []byte, projectRoot, epicKey, epicRunID, base string, events []IntegrationEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		return fmt.Errorf("integration: create backfill manifest directory: %w", err)
	}
	tmp := fmt.Sprintf("%s.%d-%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := writeTempFileSynced(tmp, data); err != nil {
		return fmt.Errorf("integration: write backfill manifest: %w", err)
	}
	defer os.Remove(tmp)
	won, err := atomicLinkPublish(tmp, path)
	if err != nil {
		return fmt.Errorf("integration: publish backfill manifest: %w", err)
	}
	if !won {
		return ensureBackfillManifest(projectRoot, epicKey, epicRunID, base, events)
	}
	return nil
}

// normalizeBackfillEvents returns a copy of events sorted by EventID with
// RecordedAt zeroed, so two calls carrying the same logical event set (in
// different order, or regenerated with a fresh timestamp) produce the same
// manifest identity. RecordedAt is not part of an event's identity —
// deriveEventID never depends on it — so it must not gate a legitimate
// retry.
func normalizeBackfillEvents(events []IntegrationEvent) []IntegrationEvent {
	normalized := make([]IntegrationEvent, len(events))
	copy(normalized, events)
	for i := range normalized {
		normalized[i].RecordedAt = time.Time{}
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].EventID < normalized[j].EventID })
	return normalized
}

func digestBackfillManifest(manifest backfillManifest) (string, error) {
	manifest.Digest = ""
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("integration: marshal backfill manifest digest: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func stateExists(projectRoot, epicKey, epicRunID string) bool {
	for _, path := range []string{runRecordPath(projectRoot, epicKey), candidatePath(projectRoot, epicRunID)} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	entries, err := os.ReadDir(filepath.Dir(eventRecordPath(projectRoot, epicRunID, "x")))
	return err == nil && len(entries) > 0
}

func validateRetainedBackfillCandidate(projectRoot, epicKey, epicRunID, base string, events []IntegrationEvent) error {
	candidate, _, err := readCandidate(candidatePath(projectRoot, epicRunID))
	if err != nil || candidate == nil {
		return err
	}
	if candidate.EpicRunID != epicRunID || candidate.BaseCommit != base {
		return &RegistrationConflictError{EpicKey: epicKey, Reason: "retained candidate does not match authorized backfill manifest"}
	}
	digest, err := computeDigest(*candidate)
	if err != nil || digest != candidate.Digest {
		return &RegistrationConflictError{EpicKey: epicKey, Reason: "retained candidate digest is invalid"}
	}
	allowed := make(map[string]bool, len(events))
	for _, event := range events {
		allowed[event.EventID] = true
	}
	seen := make(map[string]bool, len(candidate.EventIDs))
	for _, id := range candidate.EventIDs {
		if seen[id] {
			return &RegistrationConflictError{EpicKey: epicKey, Reason: "retained candidate contains duplicate event IDs"}
		}
		seen[id] = true
		if !allowed[id] {
			return &RegistrationConflictError{EpicKey: epicKey, Reason: "retained candidate contains an event outside the authorized manifest"}
		}
	}
	return nil
}

func validateCompletedBackfillCandidate(epicKey string, candidate *IntegrationCandidate, events []IntegrationEvent) error {
	if candidate == nil || len(candidate.EventIDs) != len(events) {
		return &RegistrationConflictError{EpicKey: epicKey, Reason: "backfill candidate does not contain the complete authorized event set"}
	}
	want := make(map[string]bool, len(events))
	for _, event := range events {
		want[event.EventID] = true
	}
	for _, id := range candidate.EventIDs {
		if !want[id] {
			return &RegistrationConflictError{EpicKey: epicKey, Reason: "backfill candidate contains an unauthorized event"}
		}
	}
	if candidate.HeadCommit != events[len(events)-1].FeatureCommit {
		return &RegistrationConflictError{EpicKey: epicKey, Reason: "backfill candidate head does not match the authorized final event"}
	}
	return nil
}

func sameBackfillEvent(recorded *IntegrationEvent, input IntegrationEvent, epicRunID string) bool {
	return recorded.EpicRunID == epicRunID && recorded.EventID == input.EventID && recorded.FeatureKey == input.FeatureKey && recorded.FeatureCommit == input.FeatureCommit && slices.Equal(recorded.TrackedPaths, input.TrackedPaths) && slices.Equal(recorded.UntrackedPaths, input.UntrackedPaths)
}

// verifyCommitReachable rejects base as a *BackfillValidationError unless it
// resolves to a real, reachable commit inside projectRoot's repository.
// The actual check is history.go's VerifyBaseReachable (no head yet at
// backfill time, so head is passed as "") — the same function
// AnalyzeHistory uses for the epic's own base once a run is underway, so
// there is one correctness argument for what counts as a reachable base,
// not two independently-maintained checks (task T-E34-F08-013's "shared
// with steady-state capture" scope, applying spec.md's "Key technical
// decisions" #2 principle to base-reachability rather than the write
// path that decision was originally written about).
func verifyCommitReachable(ctx context.Context, projectRoot, base string) error {
	if err := VerifyBaseReachable(ctx, projectRoot, base, ""); err != nil {
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
// Named gap: this only covers the run-record itself. Resuming a backfill
// that crashed after writing one or more events/candidate updates but
// before RegisterRun ran is now handled by the manifest recovery protocol
// above (ensureBackfillManifest/validateRetainedBackfillCandidate/
// sameBackfillEvent/validateCompletedBackfillCandidate) for the common
// crash windows: after the manifest is published, after the run record is
// written, and after each event is folded.
//
// One window remains a documented, fail-closed gap rather than an
// auto-recovered one: if a crash lands inside UpdateCandidate's own
// claim-then-publish transition (candidate.go), the claim file survives
// the crash with no record of its owning operation, so a retry that reaches
// that same digest transition again waits out candidateClaimTimeout and
// returns a plain *CandidateConflictError rather than a diagnosed,
// automatically-repaired resume. This is safe (no corruption, no silent
// wrong behavior) but not self-healing. Given Backfill is a one-time,
// human-operated bootstrap tool with a documented manual workaround (delete
// the run/event/candidate files for the epic and retry against a clean
// epic), closing this last window with a durable claim-ownership witness in
// candidate.go is deferred rather than built speculatively; see TD-212's
// research report for the design this would require if the tool's usage
// pattern ever changes enough to justify it.
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
		return checkBackfillRunIdentity(existing, epicKey, epicRunID, base)
	}

	candidate := &IntegrationRun{
		EpicRunID:  epicRunID,
		EpicKey:    epicKey,
		BaseCommit: base,
		CreatedAt:  time.Now().UTC(),
	}
	published, err := publishRun(path, candidate)
	if err != nil {
		return nil, err
	}
	// publishRun can lose the atomic-link race and return a concurrent
	// writer's already-published record instead of `candidate` — a record
	// this call never validated against (epicKey, epicRunID, base). Without
	// this check, Backfill would proceed to fold events onto a run it never
	// authorized.
	return checkBackfillRunIdentity(published, epicKey, epicRunID, base)
}

// checkBackfillRunIdentity rejects run as a *RegistrationConflictError
// unless it matches (epicKey, epicRunID, base) — the identity backfillRun's
// caller validated everything else against. Shared by both paths that can
// hand backfillRun a run record it did not itself just construct: an
// already-existing record, and a concurrent writer's winning publish.
func checkBackfillRunIdentity(run *IntegrationRun, epicKey, epicRunID, base string) (*IntegrationRun, error) {
	if run.EpicKey != epicKey || run.EpicRunID != epicRunID || run.BaseCommit != base {
		return nil, &RegistrationConflictError{
			EpicKey: epicKey,
			Reason:  "integration run changed while backfill was acquiring ownership",
		}
	}
	return run, nil
}

// simulateBackfillCandidate computes, without writing anything to disk, the
// IntegrationCandidate a non-dryRun Backfill call with the same
// (epicRunID, base, events) would produce — dryRun's "reports what it would
// create" requirement (AC-T1). It mirrors buildNextCandidate's folding rule
// (EventIDs is the sorted set of every entry's EventID; HeadCommit is the
// last-applied event's FeatureCommit) without touching any file, since a
// dry run must leave every sidecar and note byte-for-byte unchanged.
func simulateBackfillCandidate(ctx context.Context, projectRoot, epicRunID, base string, events []IntegrationEvent) (*IntegrationCandidate, error) {
	ids := make([]string, 0, len(events))
	var head string
	for _, ev := range events {
		ids = append(ids, ev.EventID)
		head = ev.FeatureCommit
	}
	sort.Strings(ids)

	tracked, untracked, err := computeDirtyPathDigests(ctx, projectRoot)
	if err != nil {
		return nil, err
	}

	candidate := &IntegrationCandidate{
		EpicRunID:               epicRunID,
		BaseCommit:              base,
		HeadCommit:              head,
		EventIDs:                ids,
		PathDigestSchemaVersion: currentPathDigestSchemaVersion,
		TrackedPathDigests:      tracked,
		UntrackedPathDigests:    untracked,
	}
	digest, err := computeDigest(*candidate)
	if err != nil {
		return nil, err
	}
	candidate.Digest = digest
	return candidate, nil
}
