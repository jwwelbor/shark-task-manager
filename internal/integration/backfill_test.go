package integration

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Task T-E34-F08-007 covers test-plan.md TC-009's service-level decision
// table for integration.Backfill: dry-run, non-dry-run, re-registration,
// and four malformed-input variants — each variant asserting zero new
// files/notes (AC-T4's validate-fully-before-first-write). Per TC-009's
// Caller-Path Contract, these tests drive the real integration.Backfill
// entrypoint against a real temp directory; the only test double is
// fakeNoteRecorder (defined in run_test.go, same package), standing in for
// this package's one DB-backed dependency exactly as T-E34-F08-012's own
// RegisterRun tests already do — not a mock of integration.Backfill or the
// filesystem.

// addTestCommit creates a new commit in dir (an existing git repo, as set
// up by chdirProjectRoot/initTestGitRepo) and returns its full hash. Used
// to give a Backfill test a second, real, reachable commit distinct from
// the repo's initial HEAD, for --base.
func addTestCommit(t *testing.T, dir, filename, content string) string {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", filename)
	run("commit", "-q", "-m", "add "+filename)

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// validBackfillEvents builds a two-entry, internally-consistent events
// array for epicRunID: each entry's EventID is the real digest
// deriveEventID computes from its own epic run/feature key/commit, exactly
// as a well-formed --events-file would carry.
func validBackfillEvents(epicRunID string) []IntegrationEvent {
	e1 := IntegrationEvent{FeatureKey: "E90-F01", FeatureCommit: "feature-commit-1", TrackedPaths: []string{"a.go"}}
	e1.EventID = deriveEventID(epicRunID, e1.FeatureKey, e1.FeatureCommit)
	e2 := IntegrationEvent{FeatureKey: "E90-F02", FeatureCommit: "feature-commit-2", UntrackedPaths: []string{"b.txt"}}
	e2.EventID = deriveEventID(epicRunID, e2.FeatureKey, e2.FeatureCommit)
	return []IntegrationEvent{e1, e2}
}

// countFilesUnder recursively counts regular files under root (root itself
// need not exist — a missing directory counts as zero files), for
// before/after directory-listing diffs.
func countFilesUnder(t *testing.T, root string) int {
	t.Helper()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return 0
	}
	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return count
}

// TestBackfill_DryRun_WritesNothing covers TC-009 subtest (a) / AC-T1:
// --dry-run against a valid, complete base/events input performs every
// validation but writes nothing to disk and creates no note; the returned
// candidate describes what a real run would create.
func TestBackfill_DryRun_WritesNothing(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)
	shark := filepath.Join(dir, ".shark")
	before := countFilesUnder(t, shark)

	const epicRunID = "run-dry"
	events := validBackfillEvents(epicRunID)
	recorder := &fakeNoteRecorder{}

	candidate, err := Backfill(context.Background(), recorder, "E90", epicRunID, headCommit, events, true, "test-agent")
	if err != nil {
		t.Fatalf("dry-run Backfill: %v", err)
	}
	if candidate == nil {
		t.Fatal("expected a non-nil candidate describing what dry-run would create")
	}
	if candidate.BaseCommit != headCommit {
		t.Errorf("candidate.BaseCommit = %q, want %q", candidate.BaseCommit, headCommit)
	}
	if len(candidate.EventIDs) != len(events) {
		t.Errorf("candidate.EventIDs has %d entries, want %d", len(candidate.EventIDs), len(events))
	}
	if candidate.Digest == "" {
		t.Error("candidate.Digest is empty")
	}

	after := countFilesUnder(t, shark)
	if after != before {
		t.Fatalf("dry-run wrote files: before=%d after=%d", before, after)
	}
	if recorder.calls != 0 {
		t.Fatalf("dry-run must not create a note: got %d AddNoteWithMetadata calls", recorder.calls)
	}

	// The dry-run report is only meaningful if it actually matches what a
	// real call would produce — a digest field that isn't empty proves
	// nothing about *which* candidate it describes. Since dry-run wrote
	// nothing above, this directory is still pristine: a real, non-dry-run
	// call against the identical input must fold to the exact same digest
	// simulateBackfillCandidate predicted, or the "reports what it would
	// create" promise is false.
	real, err := Backfill(context.Background(), recorder, "E90", epicRunID, headCommit, events, false, "test-agent")
	if err != nil {
		t.Fatalf("non-dry-run Backfill after dry-run: %v", err)
	}
	if real.Digest != candidate.Digest {
		t.Fatalf("dry-run predicted digest %q, real run produced %q — dry-run report does not match reality", candidate.Digest, real.Digest)
	}
}

// TestBackfill_NonDryRun_CreatesExactlyOneOfEach covers TC-009 subtest (b) /
// AC-T1: a non-dry-run call with the same valid input creates exactly one
// IntegrationRun file, one IntegrationEvent file per input entry, one
// IntegrationCandidate file, and one epic reference note.
func TestBackfill_NonDryRun_CreatesExactlyOneOfEach(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)

	const epicRunID = "run-live"
	events := validBackfillEvents(epicRunID)
	recorder := &fakeNoteRecorder{}

	candidate, err := Backfill(context.Background(), recorder, "E91", epicRunID, headCommit, events, false, "test-agent")
	if err != nil {
		t.Fatalf("Backfill: %v", err)
	}
	if candidate == nil {
		t.Fatal("expected a non-nil candidate")
	}

	// Exactly one IntegrationRun file for E91.
	if _, err := os.Stat(runRecordPath(dir, "E91")); err != nil {
		t.Fatalf("run record not found: %v", err)
	}

	// Exactly one IntegrationEvent file per input entry.
	for _, ev := range events {
		path := eventRecordPath(dir, epicRunID, ev.EventID)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("event file for %s not found at %s: %v", ev.FeatureKey, path, err)
		}
	}
	eventsDir := filepath.Dir(eventRecordPath(dir, epicRunID, "x"))
	if got := countFilesUnder(t, eventsDir); got != len(events) {
		t.Fatalf("expected exactly %d event files in %s, found %d", len(events), eventsDir, got)
	}

	// Exactly one IntegrationCandidate file (the live file, not the
	// intermediate-archive mechanism candidate.go already owns).
	if _, err := os.Stat(candidatePath(dir, epicRunID)); err != nil {
		t.Fatalf("candidate file not found: %v", err)
	}

	// Exactly one epic reference note.
	if recorder.calls != 1 {
		t.Fatalf("expected exactly 1 note-insert call, got %d", recorder.calls)
	}
	if len(recorder.notes) != 1 {
		t.Fatalf("expected exactly 1 stored note, got %d", len(recorder.notes))
	}
	if recorder.notes[0].entityKey != "E91" {
		t.Errorf("note entityKey = %q, want %q", recorder.notes[0].entityKey, "E91")
	}
	if recorder.notes[0].noteType != "reference" {
		t.Errorf("note noteType = %q, want %q", recorder.notes[0].noteType, "reference")
	}
}

// TestBackfill_SecondAttemptAgainstRegisteredEpic_Rejected covers TC-009
// subtest (c) / AC-T2: a second backfill call against an epic that already
// has a registered run is rejected with zero mutation, whether the retry
// supplies the identical input or a genuinely conflicting one.
func TestBackfill_SecondAttemptAgainstRegisteredEpic_Rejected(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)
	shark := filepath.Join(dir, ".shark")

	const epicRunID = "run-second"
	events := validBackfillEvents(epicRunID)
	recorder := &fakeNoteRecorder{}

	if _, err := Backfill(context.Background(), recorder, "E92", epicRunID, headCommit, events, false, "test-agent"); err != nil {
		t.Fatalf("first Backfill: %v", err)
	}

	filesAfterFirst := countFilesUnder(t, shark)
	notesAfterFirst := len(recorder.notes)

	t.Run("identical retry", func(t *testing.T) {
		_, err := Backfill(context.Background(), recorder, "E92", epicRunID, headCommit, events, false, "test-agent")
		if err == nil {
			t.Fatal("expected the second backfill attempt to be rejected")
		}
		var conflict *RegistrationConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
		}
		if got := countFilesUnder(t, shark); got != filesAfterFirst {
			t.Fatalf("second attempt (identical input) mutated files: before=%d after=%d", filesAfterFirst, got)
		}
		if len(recorder.notes) != notesAfterFirst {
			t.Fatalf("second attempt (identical input) created a note: before=%d after=%d", notesAfterFirst, len(recorder.notes))
		}
	})

	t.Run("conflicting run id", func(t *testing.T) {
		otherRunID := "run-second-conflicting"
		otherEvents := validBackfillEvents(otherRunID)
		_, err := Backfill(context.Background(), recorder, "E92", otherRunID, headCommit, otherEvents, false, "test-agent")
		if err == nil {
			t.Fatal("expected the conflicting-run-id attempt to be rejected")
		}
		var conflict *RegistrationConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
		}
		if got := countFilesUnder(t, shark); got != filesAfterFirst {
			t.Fatalf("conflicting attempt mutated files: before=%d after=%d", filesAfterFirst, got)
		}
		if len(recorder.notes) != notesAfterFirst {
			t.Fatalf("conflicting attempt created a note: before=%d after=%d", notesAfterFirst, len(recorder.notes))
		}
	})
}

// TestBackfill_MalformedInput_ZeroMutation covers TC-009 subtest (d): four
// malformed-input variants, each rejected before any write (AC-T4). Each
// subtest runs against its own fresh temp directory/epic so a before/after
// diff of zero mutation is unambiguous.
func TestBackfill_MalformedInput_ZeroMutation(t *testing.T) {
	t.Run("unreachable base", func(t *testing.T) {
		dir, _ := chdirProjectRoot(t)
		shark := filepath.Join(dir, ".shark")

		const epicRunID = "run-bad-base"
		events := validBackfillEvents(epicRunID)
		recorder := &fakeNoteRecorder{}

		_, err := Backfill(context.Background(), recorder, "E93", epicRunID, "0000000000000000000000000000000000000dead", events, false, "test-agent")
		if err == nil {
			t.Fatal("expected an error for an unreachable --base commit")
		}
		var valErr *BackfillValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected *BackfillValidationError, got %T: %v", err, err)
		}
		if got := countFilesUnder(t, shark); got != 0 {
			t.Fatalf("unreachable-base attempt wrote %d files, want 0", got)
		}
		if recorder.calls != 0 {
			t.Fatalf("unreachable-base attempt created a note: %d calls", recorder.calls)
		}
	})

	t.Run("duplicate EventID", func(t *testing.T) {
		dir, headCommit := chdirProjectRoot(t)
		shark := filepath.Join(dir, ".shark")

		const epicRunID = "run-dup-event"
		events := validBackfillEvents(epicRunID)
		events[1].EventID = events[0].EventID // force a duplicate
		recorder := &fakeNoteRecorder{}

		_, err := Backfill(context.Background(), recorder, "E94", epicRunID, headCommit, events, false, "test-agent")
		if err == nil {
			t.Fatal("expected an error for a duplicate EventID")
		}
		var valErr *BackfillValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected *BackfillValidationError, got %T: %v", err, err)
		}
		if got := countFilesUnder(t, shark); got != 0 {
			t.Fatalf("duplicate-EventID attempt wrote %d files, want 0", got)
		}
		if recorder.calls != 0 {
			t.Fatalf("duplicate-EventID attempt created a note: %d calls", recorder.calls)
		}
	})

	t.Run("digest mismatch", func(t *testing.T) {
		dir, headCommit := chdirProjectRoot(t)
		shark := filepath.Join(dir, ".shark")

		const epicRunID = "run-digest-mismatch"
		events := validBackfillEvents(epicRunID)
		events[1].EventID = "not-the-real-digest-for-this-entry"
		recorder := &fakeNoteRecorder{}

		_, err := Backfill(context.Background(), recorder, "E95", epicRunID, headCommit, events, false, "test-agent")
		if err == nil {
			t.Fatal("expected an error for an EventID that doesn't match its own entry's derived digest")
		}
		var valErr *BackfillValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected *BackfillValidationError, got %T: %v", err, err)
		}
		if got := countFilesUnder(t, shark); got != 0 {
			t.Fatalf("digest-mismatch attempt wrote %d files, want 0", got)
		}
		if recorder.calls != 0 {
			t.Fatalf("digest-mismatch attempt created a note: %d calls", recorder.calls)
		}
	})

	t.Run("bounded array size exceeded", func(t *testing.T) {
		dir, headCommit := chdirProjectRoot(t)
		shark := filepath.Join(dir, ".shark")

		const epicRunID = "run-too-many-events"
		events := make([]IntegrationEvent, maxBackfillEvents+1)
		for i := range events {
			ev := IntegrationEvent{FeatureKey: "E96-F00", FeatureCommit: "commit-" + string(rune('a'+i%26)) + string(rune('0'+i/26))}
			ev.EventID = deriveEventID(epicRunID, ev.FeatureKey, ev.FeatureCommit)
			events[i] = ev
		}
		recorder := &fakeNoteRecorder{}

		_, err := Backfill(context.Background(), recorder, "E96", epicRunID, headCommit, events, false, "test-agent")
		if err == nil {
			t.Fatal("expected an error for an events array exceeding the bounded size limit")
		}
		var valErr *BackfillValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected *BackfillValidationError, got %T: %v", err, err)
		}
		if got := countFilesUnder(t, shark); got != 0 {
			t.Fatalf("oversized-array attempt wrote %d files, want 0", got)
		}
		if recorder.calls != 0 {
			t.Fatalf("oversized-array attempt created a note: %d calls", recorder.calls)
		}
	})
}

// TestBackfill_UsesRealAdditionalCommit exercises the reachable-base check's
// happy path against a second, freshly-created real commit (not just the
// repo's initial seed commit from chdirProjectRoot), so the reachability
// check is proven against more than a single fixed value.
func TestBackfill_UsesRealAdditionalCommit(t *testing.T) {
	dir, _ := chdirProjectRoot(t)
	second := addTestCommit(t, dir, "second.txt", "second")

	const epicRunID = "run-second-commit"
	events := validBackfillEvents(epicRunID)
	recorder := &fakeNoteRecorder{}

	candidate, err := Backfill(context.Background(), recorder, "E97", epicRunID, second, events, false, "test-agent")
	if err != nil {
		t.Fatalf("Backfill against a real second commit: %v", err)
	}
	if candidate.BaseCommit != second {
		t.Errorf("candidate.BaseCommit = %q, want %q", candidate.BaseCommit, second)
	}
}
