// Package integration holds epic-run state for E34-F08's integration-event
// log: the once-per-epic-run base-commit capture (this file), and — added by
// later tasks in this feature — the per-feature-completion event log and the
// accumulated integration candidate. This is deliberately not part of
// internal/sharkdata: it is run-scoped runtime state (like this repo's
// existing `.shark/runs/<run-id>/` convention), not bundle content.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/projectroot"
)

// runDirMode / runFileMode match this repo's existing run-artifact
// convention (internal/runner/transcript.go: 0o755 dir / 0o644 file).
const (
	runDirMode  os.FileMode = 0o755
	runFileMode os.FileMode = 0o644
)

// CaptureCreatedBy identifies the automated actor recorded on durable notes
// this package's callers write about integration-capture events and
// failures (e.g. FeatureService's RegisterRun/failure-note calls, `shark
// next`'s epic-cascade capture-failure note). Exported so both
// internal/services and internal/cli/commands share one symbol instead of
// each keeping its own copy of the same literal.
const CaptureCreatedBy = "shark-integration-capture"

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
	projectRoot, err := projectroot.FindProjectRoot()
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

// GetRun returns epicKey's already-captured IntegrationRun, or (nil, nil) if
// none has been captured yet. Unlike CaptureBase, GetRun never creates a
// run — it is the read-only "has this epic's integration run already
// begun" check used by callers outside the epic active step's cascade
// action (internal/cli/commands/next.go's resolveCascade, which owns
// CaptureBase's call). FeatureService.TransitionStatus (T-E34-F08-008)
// uses this to decide whether a terminal feature transition should record
// an IntegrationEvent: no run yet means the epic never entered its active
// cascade (or this feature's own transition raced ahead of it), so there is
// nothing to record against.
func GetRun(epicKey string) (*IntegrationRun, error) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}
	return readRun(runRecordPath(projectRoot, epicKey))
}

// CurrentCommit resolves the repository's current HEAD commit hash — the
// same resolution CaptureBase uses for a new run's BaseCommit. Exported for
// FeatureService.TransitionStatus's terminal-transition RecordEvent call
// site (T-E34-F08-008), which needs "the feature's commit" without
// resolving a project root itself.
func CurrentCommit() (string, error) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return "", fmt.Errorf("integration: resolve project root: %w", err)
	}
	return currentHeadCommit(projectRoot)
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

// RecordKindIntegrationCandidateRoot is the reference note's record_kind
// value (architecture.md "Epic integration candidate identity"), exported
// so later readers of this note — T-E34-F08-014's crash-restart repair and
// T-E34-F08-010's integration_review closure check — share one literal
// rather than each redefining it.
const RecordKindIntegrationCandidateRoot = "integration-candidate-root"

// RegistrationNoteContent is the registration reference note's content
// payload: the epic run ID, base commit, and head digest architecture.md's
// "Epic integration candidate identity" section names.
type RegistrationNoteContent struct {
	RecordKind string `json:"record_kind"`
	EpicRunID  string `json:"epic_run_id"`
	BaseCommit string `json:"base_commit"`
	HeadDigest string `json:"head_digest"`
}

// RegistrationNoteMetadata is the registration reference note's
// structured-metadata payload — architecture.md: "the coordinator...
// insert[s] the note with that [suboperation] ID in metadata."
type RegistrationNoteMetadata struct {
	SuboperationID string `json:"suboperation_id"`
}

// NoteRecorder is the subset of *services.NoteService's behavior
// RegisterRun delegates note creation and lookup to. This mirrors
// services.ImpactNoteRecorder's approach for I-04: a package-local,
// consumer-side interface satisfied structurally by *services.NoteService
// without either package importing the other, so RegisterRun never
// persists a note directly and internal/integration never depends on
// internal/services. ListNotes was added by task T-E34-F08-014 so
// RegisterRun can discover an existing registration note before deciding
// whether this attempt is a first-time registration, a repair, an
// idempotent no-op, or a conflict (AC-T1/AC-T2) — its signature matches
// *services.NoteService.ListNotes exactly.
type NoteRecorder interface {
	AddNoteWithMetadata(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string, metadata string) (*models.EntityNote, error)
	ListNotes(ctx context.Context, entityType models.EntityType, entityKey string, noteTypes []string) ([]*models.EntityNote, error)
}

// RegistrationConflictError indicates RegisterRun found existing
// registration state for EpicKey that cannot be reconciled with this
// attempt without operator intervention: a different EpicRunID already
// registered while nonterminal, a matching EpicRunID whose recorded head
// digest conflicts with this attempt's, or a registered head that no
// longer resolves to valid retained bytes (absent, truncated, or
// tampered). Task T-E34-F08-014 AC-T1/AC-T2: RegisterRun fails closed in
// every one of these cases rather than guessing or reconstructing state —
// "the parent cannot acknowledge active entry or dispatch a feature until
// head and note reconcile" (architecture.md "Epic integration candidate
// identity").
type RegistrationConflictError struct {
	EpicKey string
	Reason  string
}

// Error implements the error interface.
func (e *RegistrationConflictError) Error() string {
	return fmt.Sprintf("integration: registration for epic %s cannot reconcile: %s", e.EpicKey, e.Reason)
}

// DeriveRegistrationSuboperationID derives the registration suboperation
// ID as the hex-encoded SHA-256 of epicKey+epicRunID+baseCommit+
// firstHeadDigest concatenated (architecture.md "Epic integration
// candidate identity": "The registration suboperation ID is SHA-256 over
// the epic key, epic run ID, base, and first head digest"). Deterministic:
// an exact retry with identical inputs derives the identical ID —
// T-E34-F08-014's repair-vs-conflict decision is built on this stability.
func DeriveRegistrationSuboperationID(epicKey, epicRunID, baseCommit, firstHeadDigest string) string {
	sum := sha256.Sum256([]byte(epicKey + epicRunID + baseCommit + firstHeadDigest))
	return hex.EncodeToString(sum[:])
}

// RegisterRun performs task T-E34-F08-012's AC-T2 registration-note write
// sequence for run: holding the run-scoped registration lock for the
// entire sequence, it fsyncs the event and head files this registration is
// binding, derives the deterministic registration suboperation ID from
// run's epic key/run ID/base commit and firstHead's digest, then inserts
// the idempotent epic reference note carrying RecordKindIntegrationCandidateRoot
// and that suboperation ID in metadata (architecture.md "Epic integration
// candidate identity").
//
// firstHead is the digest of the first IntegrationCandidate head ever
// published for run — the caller supplies it because this function owns
// only the registration *mechanism*, not when to invoke it: deciding that
// moment (after the first UpdateCandidate call) is T-E34-F08-008's cascade-
// wiring scope, and re-invoking it on a restarted parent is
// T-E34-F08-014's crash-restart-repair scope.
//
// This function's fsync-then-insert sequence covers the write path for a
// first-time registration, a repair of a missing note, and an idempotent
// retry once a note already exists. Before ever fsyncing or inserting,
// RegisterRun first looks for an existing registration note for run.EpicKey
// (existingRegistrationNote) and decides which of those four outcomes
// applies (T-E34-F08-014 AC-T1/AC-T2):
//
//   - no existing note: proceed with fsync + insert below — this covers both
//     a genuine first-time registration and an exact retry repairing a note
//     that a crash left missing after the head was already durable.
//   - an existing note for a *different* EpicRunID: reject
//     (*RegistrationConflictError) — a second nonterminal candidate for the
//     same epic (AC-T2). This package has no notion of "terminal" beyond a
//     note's mere presence: nothing in this package ever removes or
//     supersedes a registration note, so any note found here is, by
//     construction, still nonterminal from RegisterRun's point of view.
//   - an existing note for the *same* EpicRunID but a different HeadDigest:
//     reject — conflicting bytes fail closed without reconstruction (AC-T1).
//   - an existing note for the same EpicRunID and the same HeadDigest: this
//     attempt is already reconciled, *provided* the referenced head still
//     resolves to valid retained bytes (resolveRetainedHead) — live or
//     archived, not tampered or truncated. If it does not, that is exactly
//     "a note whose referenced head is absent/corrupt" and fails closed
//     even though the note's own bytes match (AC-T1). Otherwise this is a
//     pure idempotent no-op: the existing note is returned without a second
//     insert.
func RegisterRun(ctx context.Context, recorder NoteRecorder, run *IntegrationRun, firstHead *IntegrationCandidate, event *IntegrationEvent, createdBy string) (*models.EntityNote, error) {
	if recorder == nil {
		return nil, fmt.Errorf("integration: RegisterRun requires a non-nil NoteRecorder")
	}
	if run == nil {
		return nil, fmt.Errorf("integration: RegisterRun requires a non-nil run")
	}
	if firstHead == nil {
		return nil, fmt.Errorf("integration: RegisterRun requires a non-nil firstHead")
	}
	if event == nil {
		return nil, fmt.Errorf("integration: RegisterRun requires a non-nil event")
	}

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("integration: resolve project root: %w", err)
	}

	lock, err := AcquireRegistrationLock(projectRoot, run.EpicRunID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release() }()

	existingNote, existingContent, err := existingRegistrationNote(ctx, recorder, run.EpicKey)
	if err != nil {
		return nil, err
	}
	if existingNote != nil {
		if existingContent.EpicRunID != run.EpicRunID {
			return nil, &RegistrationConflictError{
				EpicKey: run.EpicKey,
				Reason: fmt.Sprintf(
					"epic run %s is already registered and nonterminal; refusing to register a second run %s",
					existingContent.EpicRunID, run.EpicRunID,
				),
			}
		}
		if existingContent.HeadDigest != firstHead.Digest {
			return nil, &RegistrationConflictError{
				EpicKey: run.EpicKey,
				Reason: fmt.Sprintf(
					"existing registration note for run %s references head %s, which conflicts with this attempt's head %s",
					run.EpicRunID, existingContent.HeadDigest, firstHead.Digest,
				),
			}
		}
		if _, err := resolveRetainedHead(projectRoot, run.EpicRunID, existingContent.HeadDigest); err != nil {
			return nil, &RegistrationConflictError{
				EpicKey: run.EpicKey,
				Reason:  fmt.Sprintf("registered head %s does not reconcile: %v", existingContent.HeadDigest, err),
			}
		}
		return existingNote, nil
	}

	// No existing note: either a first-time registration or an exact retry
	// repairing a note a crash left missing. Either way, the head this note
	// is about to reference must itself resolve to valid retained bytes
	// before RegisterRun ever inserts a note pointing at it.
	if _, err := resolveRetainedHead(projectRoot, run.EpicRunID, firstHead.Digest); err != nil {
		return nil, &RegistrationConflictError{
			EpicKey: run.EpicKey,
			Reason:  fmt.Sprintf("head %s cannot be registered: %v", firstHead.Digest, err),
		}
	}

	// AC-T2: "events and head are fsynced before the note is inserted."
	// The event and candidate files are already fully written by RecordEvent
	// /UpdateCandidate before RegisterRun is ever called; this step
	// guarantees their contents are durable on disk before the note — the
	// record that later readers use to discover this run — can be found.
	// This does not fsync the containing directory entries (the other half
	// of making an os.Link/os.Rename durable across a crash); that is a
	// named gap this task leaves open rather than guessed at here for a
	// repo with Windows-conditional tests.
	eventPath := eventRecordPath(projectRoot, run.EpicRunID, event.EventID)
	if err := fsyncFile(eventPath); err != nil {
		return nil, fmt.Errorf("integration: fsync event file before registration: %w", err)
	}
	headPath := candidatePath(projectRoot, run.EpicRunID)
	if err := fsyncFile(headPath); err != nil {
		return nil, fmt.Errorf("integration: fsync head file before registration: %w", err)
	}

	// registerRunTestHook, when set, lets a test simulate a crash at exactly
	// this point: the event and head are already durable (fsynced above,
	// candidate head already replaced by an earlier UpdateCandidate call),
	// but the note has not yet been acknowledged. Production code never sets
	// it. See its own doc comment below for why one seam here covers every
	// "failure after fsync, before note ack" scenario test-plan.md TC-017
	// names (after event fsync, after archived-head fsync, after
	// candidate-head replacement) rather than one hook per scenario: a
	// restart after any of them always resumes from this same point.
	if registerRunTestHook != nil {
		if err := registerRunTestHook(); err != nil {
			return nil, err
		}
	}

	suboperationID := DeriveRegistrationSuboperationID(run.EpicKey, run.EpicRunID, run.BaseCommit, firstHead.Digest)

	metadata, err := json.Marshal(RegistrationNoteMetadata{SuboperationID: suboperationID})
	if err != nil {
		return nil, fmt.Errorf("integration: marshal registration metadata: %w", err)
	}
	content, err := json.Marshal(RegistrationNoteContent{
		RecordKind: RecordKindIntegrationCandidateRoot,
		EpicRunID:  run.EpicRunID,
		BaseCommit: run.BaseCommit,
		HeadDigest: firstHead.Digest,
	})
	if err != nil {
		return nil, fmt.Errorf("integration: marshal registration content: %w", err)
	}

	note, err := recorder.AddNoteWithMetadata(ctx, models.EntityTypeEpic, run.EpicKey, string(models.NoteTypeReference), string(content), createdBy, string(metadata))
	if err != nil {
		return nil, fmt.Errorf("integration: insert registration note for epic %s: %w", run.EpicKey, err)
	}
	return note, nil
}

// fsyncFile opens path read-write and calls Sync on it, guaranteeing its
// already-written contents are durable before RegisterRun proceeds to
// insert the registration note. Opened O_RDWR rather than read-only:
// File.Sync ultimately calls FlushFileBuffers on Windows, which requires a
// write-capable handle (this repo has Windows-conditional tests
// elsewhere, so this is worth getting right rather than assuming Linux).
func fsyncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("integration: open %s for fsync: %w", path, err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("integration: fsync %s: %w", path, err)
	}
	return nil
}

// registerRunTestHook, when non-nil, is invoked exactly once per RegisterRun
// attempt that reaches the note-insert step: immediately after fsyncing the
// event and head files this registration is binding, but before ever
// calling the note recorder. Production code never sets it (nil by
// default), mirroring candidate.go's updateCandidateTestHook/archiveTestHook
// seam convention. Unlike those hooks, this one may return an error: a test
// uses that to simulate a crash landing at exactly this point — fsync
// already durable, but the note never acknowledged — without hand-
// constructing a note or bypassing RegisterRun's real fsync-then-insert
// path (test-plan.md TC-017's Caller-Path Contract).
var registerRunTestHook func() error

// existingRegistrationNote returns the single epic-level reference note
// carrying RecordKindIntegrationCandidateRoot for epicKey, if any, along
// with its parsed RegistrationNoteContent. It returns (nil, nil, nil) when
// no such note exists yet — the ordinary "nothing registered yet" case that
// covers both a genuine first-time registration and an exact retry
// repairing a note a crash left missing (task T-E34-F08-014 AC-T1). A note
// of the right type whose content does not parse as RegistrationNoteContent
// is not ours and is skipped rather than treated as a match. More than one
// matching note is itself an unreconciled state — this package never
// expects to create two — and is reported as a *RegistrationConflictError
// rather than picking one arbitrarily.
func existingRegistrationNote(ctx context.Context, recorder NoteRecorder, epicKey string) (*models.EntityNote, *RegistrationNoteContent, error) {
	notes, err := recorder.ListNotes(ctx, models.EntityTypeEpic, epicKey, []string{string(models.NoteTypeReference)})
	if err != nil {
		return nil, nil, fmt.Errorf("integration: list existing registration notes for epic %s: %w", epicKey, err)
	}

	var (
		match        *models.EntityNote
		matchContent *RegistrationNoteContent
		matchCount   int
	)
	for _, note := range notes {
		var content RegistrationNoteContent
		if err := json.Unmarshal([]byte(note.Content), &content); err != nil {
			continue // not a registration note in this content shape — not ours
		}
		if content.RecordKind != RecordKindIntegrationCandidateRoot {
			continue
		}
		matchCount++
		match = note
		matchContent = &content
	}

	if matchCount == 0 {
		return nil, nil, nil
	}
	if matchCount > 1 {
		return nil, nil, &RegistrationConflictError{
			EpicKey: epicKey,
			Reason:  fmt.Sprintf("%d registration notes found, expected at most one", matchCount),
		}
	}
	return match, matchContent, nil
}

// resolveRetainedHead resolves headDigest's IntegrationCandidate bytes for
// epicRunID, checking the live candidate file first (if it currently holds
// that digest) and falling back to the archived copy T-E34-F08-012 writes
// to integration-heads/<digest>.json before ever replacing a head
// (architecture.md "Epic integration candidate identity": "every
// prior_record_digest is recomputable from retained bytes"). Whichever
// source resolves, resolveRetainedHead recomputes the digest from the
// resolved bytes and rejects a mismatch rather than trusting the source
// location alone — this single content-address check is what makes a head
// "absent/corrupt" (task T-E34-F08-014 AC-T1) fail closed, and it also
// catches a "reordering" attack that swaps two archived heads' file
// contents between each other's digest-named files: a swapped file's
// content no longer hashes to the name it was swapped onto.
func resolveRetainedHead(projectRoot, epicRunID, headDigest string) (*IntegrationCandidate, error) {
	livePath := candidatePath(projectRoot, epicRunID)
	if live, _, err := readCandidate(livePath); err == nil && live != nil && live.Digest == headDigest {
		if err := verifyRetainedDigest(*live, headDigest); err != nil {
			return nil, err
		}
		return live, nil
	}

	archivePath := filepath.Join(candidateHeadsDir(livePath), headDigest+".json")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("integration: head %s is absent — no live or archived record found", headDigest)
		}
		return nil, fmt.Errorf("integration: read archived head %s: %w", headDigest, err)
	}

	var archived IntegrationCandidate
	if err := json.Unmarshal(data, &archived); err != nil {
		return nil, fmt.Errorf("integration: archived head %s is corrupt (truncated or invalid JSON): %w", headDigest, err)
	}
	if err := verifyRetainedDigest(archived, headDigest); err != nil {
		return nil, err
	}
	return &archived, nil
}

// verifyRetainedDigest rejects candidate as corrupt if either its own
// stored Digest field or the digest recomputed from its content disagrees
// with wantDigest — the name resolveRetainedHead is trying to resolve this
// record under (a live or archived file's own claimed identity).
func verifyRetainedDigest(candidate IntegrationCandidate, wantDigest string) error {
	if candidate.Digest != wantDigest {
		return fmt.Errorf("integration: head %s is corrupt: stored digest field is %s", wantDigest, candidate.Digest)
	}
	recomputed, err := computeDigest(candidate)
	if err != nil {
		return fmt.Errorf("integration: recompute digest for head %s: %w", wantDigest, err)
	}
	if recomputed != wantDigest {
		return fmt.Errorf("integration: head %s is corrupt: content does not hash to its own digest (tampered or truncated)", wantDigest)
	}
	return nil
}
