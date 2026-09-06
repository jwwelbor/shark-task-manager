package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/integration"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// deriveEventIDForTest replicates integration.deriveEventID's unexported
// derivation (spec.md REQ-F-004: first 16 hex chars of
// sha256(epicRunID+featureKey+featureCommit)) so this fixture's EventID is
// internally consistent without exporting a test-only seam from
// internal/integration for a single CLI fixture.
func deriveEventIDForTest(epicRunID, featureKey, featureCommit string) string {
	sum := sha256.Sum256([]byte(epicRunID + featureKey + featureCommit))
	return hex.EncodeToString(sum[:])[:16]
}

// Task T-E34-F08-015 covers test-plan.md TC-010: the CLI-layer proof that
// `shark integration backfill`'s claim/session authorization check (which
// only this layer owns — integration.Backfill's own signature has no
// session parameter) is actually wired into runIntegrationBackfill, not
// just implemented somewhere unreachable. Per TC-010's Caller-Path
// Contract, only the claim-lookup seam (integrationClaimLookup) is mocked;
// integration.Backfill itself is the real, unmocked function, and the
// events-file/base-commit fixtures below are internally valid so that an
// implementation which forgot to enforce the claim check would actually
// reach Backfill and write real files — which the before/after file count
// below would then catch.

// initIntegrationTestGitRepo creates a real, minimal git repo in dir and
// returns its HEAD commit — a reachable --base value real enough that a
// wiring bug (skipping the claim check) would let integration.Backfill
// proceed past its own base-reachability validation.
func initIntegrationTestGitRepo(t *testing.T, dir string) string {
	t.Helper()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	run("add", "seed.txt")
	run("commit", "-q", "-m", "seed")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// writeIntegrationEventsFile writes a single internally-consistent
// IntegrationEvent (its EventID matches the digest Backfill itself would
// derive from epicRunID/FeatureKey/FeatureCommit) to a temp JSON file and
// returns its path — a fully valid --events-file, so nothing about the
// events file itself gives TC-010's rejection a reason to fire.
func writeIntegrationEventsFile(t *testing.T, dir, epicRunID string) string {
	t.Helper()

	event := integration.IntegrationEvent{
		FeatureKey:    "E90-F01",
		FeatureCommit: "feature-commit-1",
		TrackedPaths:  []string{"a.go"},
		RecordedAt:    time.Now().UTC(),
	}
	event.EventID = deriveEventIDForTest(epicRunID, event.FeatureKey, event.FeatureCommit)

	data, err := json.Marshal([]integration.IntegrationEvent{event})
	if err != nil {
		t.Fatalf("marshal events fixture: %v", err)
	}

	path := filepath.Join(dir, "events.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write events fixture: %v", err)
	}
	return path
}

// countFilesUnderForTest recursively counts regular files under root (a
// missing directory counts as zero), for TC-010's before/after
// directory-listing diff.
func countFilesUnderForTest(t *testing.T, root string) int {
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

// buildIntegrationBackfillTestCmd returns a fresh *cobra.Command carrying
// the same flags integrationBackfillCmd registers, populated for a single
// TC-010 invocation. A fresh command (not the shared package-level
// integrationBackfillCmd) avoids cross-test flag-state pollution, matching
// this file's buildCmdWithTagFlag precedent in entity_tag_error_path_test.go.
func buildIntegrationBackfillTestCmd(t *testing.T, epicRunID, base, eventsFile, session string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "backfill", SilenceErrors: true, SilenceUsage: true}
	cmd.Flags().String("epic-run-id", "", "")
	cmd.Flags().String("base", "", "")
	cmd.Flags().String("events-file", "", "")
	cmd.Flags().String("session", "", "")
	cmd.Flags().Bool("dry-run", false, "")

	set := func(name, value string) {
		t.Helper()
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	set("epic-run-id", epicRunID)
	set("base", base)
	set("events-file", eventsFile)
	set("session", session)
	return cmd
}

// withIntegrationClaimLookup overrides the integrationClaimLookup seam for
// the duration of the test, restoring the original on cleanup.
func withIntegrationClaimLookup(t *testing.T, fn func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error)) {
	t.Helper()
	orig := integrationClaimLookup
	integrationClaimLookup = fn
	t.Cleanup(func() { integrationClaimLookup = orig })
}

// fakeCLINoteRecorder is a minimal integration.NoteRecorder test double —
// this file's own copy of internal/integration's fakeNoteRecorder shape
// (T-E34-F08-007's tests), needed here because that type is unexported in a
// different package. It stands in for the CLI's one DB-backed dependency
// (*services.NoteService), never for internal/integration itself: the
// positive-path test below still drives the real, unmocked
// integration.Backfill against a real temp git repo, matching TC-010's
// Caller-Path Contract ("do not mock internal/integration package
// functions in this test — only the claim lookup is mocked").
type fakeCLINoteRecorder struct {
	calls int
	notes []*models.EntityNote
}

func (f *fakeCLINoteRecorder) AddNoteWithMetadata(_ context.Context, entityType models.EntityType, _ string, noteType string, content string, _ string, metadata string) (*models.EntityNote, error) {
	f.calls++
	note := &models.EntityNote{EntityType: entityType, NoteType: models.NoteType(noteType), Content: content, Metadata: &metadata}
	f.notes = append(f.notes, note)
	return note, nil
}

func (f *fakeCLINoteRecorder) ListNotes(_ context.Context, _ models.EntityType, _ string, noteTypes []string) ([]*models.EntityNote, error) {
	wantTypes := make(map[string]bool, len(noteTypes))
	for _, t := range noteTypes {
		wantTypes[t] = true
	}
	var out []*models.EntityNote
	for _, n := range f.notes {
		if len(wantTypes) > 0 && !wantTypes[string(n.NoteType)] {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

// withIntegrationNoteRecorder overrides the integrationNoteRecorder seam
// for the duration of the test, restoring the original on cleanup.
func withIntegrationNoteRecorder(t *testing.T, recorder integration.NoteRecorder) {
	t.Helper()
	orig := integrationNoteRecorder
	integrationNoteRecorder = func(ctx context.Context) (integration.NoteRecorder, error) {
		return recorder, nil
	}
	t.Cleanup(func() { integrationNoteRecorder = orig })
}

// TestRunIntegrationBackfill_NoActiveClaim_RejectsWithZeroMutation covers
// TC-010 subtest (a): no active claim exists on <epic-key> at all.
// runIntegrationBackfill must reject before ever calling integration.Backfill,
// leaving the epic's .shark/integration tree exactly as it was.
func TestRunIntegrationBackfill_NoActiveClaim_RejectsWithZeroMutation(t *testing.T) {
	dir := t.TempDir()
	initIntegrationTestGitRepo(t, dir)
	t.Chdir(dir)

	const epicKey = "E90"
	const epicRunID = "run-tc010-a"
	eventsFile := writeIntegrationEventsFile(t, dir, epicRunID)
	shark := filepath.Join(dir, ".shark")
	before := countFilesUnderForTest(t, shark)

	withIntegrationClaimLookup(t, func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
		if entityType != "epic" || entityKey != epicKey {
			t.Fatalf("unexpected claim lookup: entityType=%q entityKey=%q", entityType, entityKey)
		}
		return nil, nil // no active claim
	})

	headCommit := currentHeadForTest(t, dir)
	cmd := buildIntegrationBackfillTestCmd(t, epicRunID, headCommit, eventsFile, "session-abc")

	err := runIntegrationBackfill(cmd, []string{epicKey})
	if err == nil {
		t.Fatal("expected rejection when no active claim exists, got nil error")
	}

	after := countFilesUnderForTest(t, shark)
	if after != before {
		t.Fatalf("expected zero mutation on rejection: before=%d after=%d", before, after)
	}
}

// TestRunIntegrationBackfill_SessionMismatch_RejectsWithZeroMutation covers
// TC-010 subtest (b): an active claim exists on <epic-key> but its session
// id does not match --session. runIntegrationBackfill must reject naming
// the mismatch, leaving the epic's .shark/integration tree unchanged.
func TestRunIntegrationBackfill_SessionMismatch_RejectsWithZeroMutation(t *testing.T) {
	dir := t.TempDir()
	initIntegrationTestGitRepo(t, dir)
	t.Chdir(dir)

	const epicKey = "E90"
	const epicRunID = "run-tc010-b"
	eventsFile := writeIntegrationEventsFile(t, dir, epicRunID)
	shark := filepath.Join(dir, ".shark")
	before := countFilesUnderForTest(t, shark)

	withIntegrationClaimLookup(t, func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
		if entityType != "epic" || entityKey != epicKey {
			t.Fatalf("unexpected claim lookup: entityType=%q entityKey=%q", entityType, entityKey)
		}
		return &models.EntityClaim{
			EntityType: "epic",
			EntityKey:  epicKey,
			ClaimedBy:  "someone-else",
			SessionID:  "a-different-session",
		}, nil
	})

	headCommit := currentHeadForTest(t, dir)
	cmd := buildIntegrationBackfillTestCmd(t, epicRunID, headCommit, eventsFile, "session-abc")

	err := runIntegrationBackfill(cmd, []string{epicKey})
	if err == nil {
		t.Fatal("expected rejection when --session does not match the active claim, got nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "session") {
		t.Fatalf("expected error to name the session mismatch, got: %v", err)
	}

	after := countFilesUnderForTest(t, shark)
	if after != before {
		t.Fatalf("expected zero mutation on rejection: before=%d after=%d", before, after)
	}
}

// TestRunIntegrationBackfill_MatchingClaim_WiresToRealBackfill is the TC-010
// wiring proof named by this feature's Caller-Path Contract: "a CLI wrapper
// that never actually calls integration.Backfill (dead wiring) would still
// pass TC-009 (service-level) but fail TC-010's end-to-end file assertions."
// A matching claim lets runIntegrationBackfill proceed past its own
// authorization check into the real, unmocked integration.Backfill against
// a real temp git repo; this test then asserts the real files
// integration.Backfill's non-dry-run path creates actually exist, and that
// the fake note recorder recorded exactly one reference note. Deleting the
// integration.Backfill call (replacing it with a fabricated candidate)
// makes this test fail, which the two rejection subtests above cannot.
func TestRunIntegrationBackfill_MatchingClaim_WiresToRealBackfill(t *testing.T) {
	dir := t.TempDir()
	initIntegrationTestGitRepo(t, dir)
	t.Chdir(dir)

	const epicKey = "E90"
	const epicRunID = "run-tc010-wiring"
	const session = "session-abc"
	eventsFile := writeIntegrationEventsFile(t, dir, epicRunID)
	headCommit := currentHeadForTest(t, dir)

	withIntegrationClaimLookup(t, func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
		if entityType != "epic" || entityKey != epicKey {
			t.Fatalf("unexpected claim lookup: entityType=%q entityKey=%q", entityType, entityKey)
		}
		return &models.EntityClaim{
			EntityType: "epic",
			EntityKey:  epicKey,
			ClaimedBy:  "agent-1",
			SessionID:  session,
		}, nil
	})
	recorder := &fakeCLINoteRecorder{}
	withIntegrationNoteRecorder(t, recorder)

	cmd := buildIntegrationBackfillTestCmd(t, epicRunID, headCommit, eventsFile, session)

	if err := runIntegrationBackfill(cmd, []string{epicKey}); err != nil {
		t.Fatalf("runIntegrationBackfill: %v", err)
	}

	runRecordPath := filepath.Join(dir, ".shark", "integration", epicKey, "run.json")
	if _, err := os.Stat(runRecordPath); err != nil {
		t.Errorf("expected run record at %s: %v", runRecordPath, err)
	}
	eventDir := filepath.Join(dir, ".shark", "runs", epicRunID, "integration-events")
	if n := countFilesUnderForTest(t, eventDir); n != 1 {
		t.Errorf("expected exactly one integration event file, got %d", n)
	}
	candidatePath := filepath.Join(dir, ".shark", "runs", epicRunID, "integration-candidate.json")
	if _, err := os.Stat(candidatePath); err != nil {
		t.Errorf("expected candidate file at %s: %v", candidatePath, err)
	}

	if recorder.calls != 1 {
		t.Fatalf("expected exactly one AddNoteWithMetadata call, got %d", recorder.calls)
	}
	if got := string(recorder.notes[0].NoteType); got != "reference" {
		t.Errorf("expected a reference note, got note type %q", got)
	}
}

// TestRunIntegrationBackfill_MalformedEventsFile_RejectsWithZeroMutation
// covers TC-009(d)'s "--events-file is not valid JSON" variant at the CLI
// layer: integration.Backfill's own signature (spec.md) accepts only an
// already-parsed []IntegrationEvent, so this case can only be rejected by
// runIntegrationBackfill itself, before Backfill is ever called. The claim
// here matches --session, so a rejection can only be attributed to the
// malformed JSON, not the authorization check.
func TestRunIntegrationBackfill_MalformedEventsFile_RejectsWithZeroMutation(t *testing.T) {
	dir := t.TempDir()
	initIntegrationTestGitRepo(t, dir)
	t.Chdir(dir)

	const epicKey = "E90"
	const epicRunID = "run-tc009-d"
	const session = "session-abc"
	headCommit := currentHeadForTest(t, dir)

	eventsFile := filepath.Join(dir, "malformed-events.json")
	if err := os.WriteFile(eventsFile, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write malformed events fixture: %v", err)
	}
	shark := filepath.Join(dir, ".shark")
	before := countFilesUnderForTest(t, shark)

	withIntegrationClaimLookup(t, func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
		return &models.EntityClaim{
			EntityType: "epic",
			EntityKey:  epicKey,
			ClaimedBy:  "agent-1",
			SessionID:  session,
		}, nil
	})
	recorder := &fakeCLINoteRecorder{}
	withIntegrationNoteRecorder(t, recorder)

	cmd := buildIntegrationBackfillTestCmd(t, epicRunID, headCommit, eventsFile, session)

	err := runIntegrationBackfill(cmd, []string{epicKey})
	if err == nil {
		t.Fatal("expected rejection for a malformed --events-file, got nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "json") {
		t.Fatalf("expected error to name the JSON parse failure, got: %v", err)
	}

	after := countFilesUnderForTest(t, shark)
	if after != before {
		t.Fatalf("expected zero mutation on rejection: before=%d after=%d", before, after)
	}
	if recorder.calls != 0 {
		t.Fatalf("expected zero note calls on rejection, got %d", recorder.calls)
	}
}

// TestRunIntegrationBackfill_MalformedEpicRunID_RejectsWithZeroMutation
// covers the code-review kickback on this task (defect class shared with
// T-E34-F08-007: "--epic-run-id checked for non-empty only, not format").
// A non-empty --epic-run-id containing a path separator and ".." must be
// rejected by runIntegrationBackfill itself — via the same
// integration.ValidateEpicRunID allowlist Backfill applies — before the
// claim lookup or integration.Backfill ever run, leaving the epic's
// .shark/integration tree untouched. The claim-lookup stub calls t.Fatal if
// invoked, proving the rejection happens strictly before that seam, not
// merely before Backfill.
func TestRunIntegrationBackfill_MalformedEpicRunID_RejectsWithZeroMutation(t *testing.T) {
	dir := t.TempDir()
	initIntegrationTestGitRepo(t, dir)
	t.Chdir(dir)

	const epicKey = "E90"
	const malformedEpicRunID = "../../etc/passwd"
	const session = "session-abc"
	headCommit := currentHeadForTest(t, dir)
	eventsFile := writeIntegrationEventsFile(t, dir, "run-placeholder")
	shark := filepath.Join(dir, ".shark")
	before := countFilesUnderForTest(t, shark)

	withIntegrationClaimLookup(t, func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
		t.Fatal("claim lookup must not be reached for a malformed --epic-run-id")
		return nil, nil
	})

	cmd := buildIntegrationBackfillTestCmd(t, malformedEpicRunID, headCommit, eventsFile, session)

	err := runIntegrationBackfill(cmd, []string{epicKey})
	if err == nil {
		t.Fatal("expected rejection for a malformed --epic-run-id, got nil error")
	}
	if !strings.Contains(err.Error(), "epic-run-id") {
		t.Fatalf("expected error to name --epic-run-id, got: %v", err)
	}
	var validationErr *integration.BackfillValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected a *integration.BackfillValidationError, got %T: %v", err, err)
	}

	after := countFilesUnderForTest(t, shark)
	if after != before {
		t.Fatalf("expected zero mutation on rejection: before=%d after=%d", before, after)
	}
}

// TestRunIntegrationBackfill_NonCanonicalEpicKey_RejectsBeforeClaimLookup
// covers the same kickback's <epic-key> half: integration.Backfill now
// requires a bare epic key (models.ValidateEpicKey, "^E\\d{2}$"), so this
// CLI layer's gate must reject a slugged key like "E90-user-management"
// itself rather than relying on DetectEntityType's looser "-"-segment check,
// which accepts it. Rejection must happen before the claim lookup and
// before any flags are even read, leaving the epic's .shark/integration
// tree untouched.
func TestRunIntegrationBackfill_NonCanonicalEpicKey_RejectsBeforeClaimLookup(t *testing.T) {
	dir := t.TempDir()
	initIntegrationTestGitRepo(t, dir)
	t.Chdir(dir)

	const slugEpicKey = "E90-user-management"
	const epicRunID = "run-noncanonical-key"
	const session = "session-abc"
	headCommit := currentHeadForTest(t, dir)
	eventsFile := writeIntegrationEventsFile(t, dir, epicRunID)
	shark := filepath.Join(dir, ".shark")
	before := countFilesUnderForTest(t, shark)

	withIntegrationClaimLookup(t, func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
		t.Fatal("claim lookup must not be reached for a non-canonical epic key")
		return nil, nil
	})

	cmd := buildIntegrationBackfillTestCmd(t, epicRunID, headCommit, eventsFile, session)

	err := runIntegrationBackfill(cmd, []string{slugEpicKey})
	if err == nil {
		t.Fatal("expected rejection for a non-canonical (slugged) epic key, got nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "epic key") {
		t.Fatalf("expected error to name the invalid epic key, got: %v", err)
	}

	after := countFilesUnderForTest(t, shark)
	if after != before {
		t.Fatalf("expected zero mutation on rejection: before=%d after=%d", before, after)
	}
}

// currentHeadForTest resolves dir's current HEAD commit.
func currentHeadForTest(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}
