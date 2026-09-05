package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TC-006 (spec.md AC-3 / task T-E34-F08-004 AC-T1/AC-T2/AC-T3): CaptureBase
// idempotent capture, a concurrent race, and a corrupted-run-file edge case.
// Per the test plan's Caller-Path Contract for TC-006, these tests drive
// real file I/O against a real temp `.shark/` directory — no filesystem
// mock, no in-memory run-record stub.

// initTestGitRepo initializes a minimal real git repository at dir with one
// commit and returns that commit's full hash. CaptureBase resolves
// BaseCommit via `git rev-parse HEAD`, so exercising it for real (rather
// than faking a commit hash) is required by TC-006's "must be real file
// I/O" contract.
func initTestGitRepo(t *testing.T, dir string) string {
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

// chdirProjectRoot creates a temp git repository, chdirs the test process
// into it (t.Chdir restores the original working directory automatically at
// test cleanup), and returns the repo directory and its HEAD commit.
// CaptureBase resolves its project root via cli.FindProjectRoot(), which
// walks up from the process's working directory — this is the seam TC-006
// uses to point CaptureBase at a real temp `.shark/` tree instead of this
// repository's own.
func chdirProjectRoot(t *testing.T) (dir, headCommit string) {
	t.Helper()
	dir = t.TempDir()
	headCommit = initTestGitRepo(t, dir)
	t.Chdir(dir)
	return dir, headCommit
}

// TestCaptureBase_SequentialIdempotent covers TC-006 subtest (a) / AC-T1:
// calling CaptureBase twice, sequentially, for the same epic key returns an
// identical BaseCommit/EpicRunID both times, and exactly one run record
// exists on disk afterward.
func TestCaptureBase_SequentialIdempotent(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)

	first, err := CaptureBase("E99")
	if err != nil {
		t.Fatalf("first CaptureBase: %v", err)
	}
	if first.BaseCommit != headCommit {
		t.Fatalf("BaseCommit = %q, want %q", first.BaseCommit, headCommit)
	}
	if first.EpicRunID == "" {
		t.Fatal("EpicRunID is empty")
	}

	second, err := CaptureBase("E99")
	if err != nil {
		t.Fatalf("second CaptureBase: %v", err)
	}
	if second.BaseCommit != first.BaseCommit {
		t.Fatalf("BaseCommit changed on second call: got %q, want %q", second.BaseCommit, first.BaseCommit)
	}
	if second.EpicRunID != first.EpicRunID {
		t.Fatalf("EpicRunID changed on second call: got %q, want %q", second.EpicRunID, first.EpicRunID)
	}

	assertExactlyOneRunFile(t, dir, "E99")
}

// TestCaptureBase_ConcurrentRace covers TC-006 subtest (b) / AC-T2: several
// goroutines call CaptureBase for the same epic key simultaneously (started
// via a WaitGroup barrier, not serialized by a test-side mutex, so the real
// file writes actually race), yet every goroutine observes the same
// BaseCommit/EpicRunID and exactly one run record exists on disk once all
// goroutines have joined.
func TestCaptureBase_ConcurrentRace(t *testing.T) {
	dir, headCommit := chdirProjectRoot(t)

	const goroutines = 8
	var (
		start sync.WaitGroup
		done  sync.WaitGroup
		mu    sync.Mutex
		runs  = make([]*IntegrationRun, goroutines)
		errs  = make([]error, goroutines)
	)
	start.Add(1)
	done.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer done.Done()
			start.Wait() // barrier: release all goroutines together
			run, err := CaptureBase("E99")
			mu.Lock()
			runs[i] = run
			errs[i] = err
			mu.Unlock()
		}()
	}
	start.Done() // release the barrier
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: CaptureBase error: %v", i, err)
		}
	}

	first := runs[0]
	if first.BaseCommit != headCommit {
		t.Fatalf("BaseCommit = %q, want %q", first.BaseCommit, headCommit)
	}
	for i, run := range runs {
		if run.BaseCommit != first.BaseCommit {
			t.Errorf("goroutine %d: BaseCommit = %q, want %q", i, run.BaseCommit, first.BaseCommit)
		}
		if run.EpicRunID != first.EpicRunID {
			t.Errorf("goroutine %d: EpicRunID = %q, want %q", i, run.EpicRunID, first.EpicRunID)
		}
	}

	assertExactlyOneRunFile(t, dir, "E99")
}

// TestCaptureBase_CorruptExistingRunReturnsTypedError covers TC-006's edge
// case / AC-T3: an existing run file that is not valid JSON must fail with a
// typed error, never a silent second run creation.
func TestCaptureBase_CorruptExistingRunReturnsTypedError(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	path := runRecordPath(dir, "E99")
	if err := os.MkdirAll(filepath.Dir(path), runDirMode); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), runFileMode); err != nil {
		t.Fatalf("write corrupt run file: %v", err)
	}

	_, err := CaptureBase("E99")
	if err == nil {
		t.Fatal("expected an error for a corrupt run file, got nil")
	}

	var corruptErr *CorruptRunError
	if !errors.As(err, &corruptErr) {
		t.Fatalf("expected *CorruptRunError, got %T: %v", err, err)
	}

	assertExactlyOneRunFile(t, dir, "E99")
}

// assertExactlyOneRunFile asserts exactly one file exists in epicKey's run
// directory under projectRoot.
func assertExactlyOneRunFile(t *testing.T, projectRoot, epicKey string) {
	t.Helper()
	dir := filepath.Dir(runRecordPath(projectRoot, epicKey))
	matches, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatalf("glob run dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one run file in %s, found %d: %v", dir, len(matches), matches)
	}
}

// Task T-E34-F08-012 covers test-plan.md TC-016's remaining registration
// assertions: the suboperation ID derivation and the fsync-before-note
// ordering RegisterRun performs under the run-scoped lock (AC-T2). The
// lock's own concurrent-serialization guarantee is covered directly in
// lock_test.go; the tests here focus on RegisterRun's own sequencing and
// content, using a fake NoteRecorder (not a mock of the filesystem or the
// lock — TC-016's forbidden mocks — but a stand-in for the DB-backed note
// write, per this feature's own test-plan framing that only the claim/
// session seam is DB-backed and mockable within this filesystem-only
// package).

// fakeNoteRecorder is a NoteRecorder test double that records each call's
// arguments and tracks the maximum number of concurrently in-flight calls,
// so a concurrency test can observe whether RegisterRun's lock actually
// serialized real racing goroutines without the test itself imposing any
// serialization.
type fakeNoteRecorder struct {
	mu           sync.Mutex
	calls        int
	inFlight     int
	maxInFlight  int
	lastContent  string
	lastMetadata string
	err          error
}

func (f *fakeNoteRecorder) AddNoteWithMetadata(_ context.Context, _ models.EntityType, _ string, _ string, content string, _ string, metadata string) (*models.EntityNote, error) {
	f.mu.Lock()
	f.calls++
	f.inFlight++
	if f.inFlight > f.maxInFlight {
		f.maxInFlight = f.inFlight
	}
	f.lastContent = content
	f.lastMetadata = metadata
	err := f.err
	f.mu.Unlock()

	time.Sleep(5 * time.Millisecond) // widen the race window

	f.mu.Lock()
	f.inFlight--
	f.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return &models.EntityNote{}, nil
}

// setUpRegisteredRun creates a real run, event, and candidate on disk
// (via CaptureBase/RecordEvent/UpdateCandidate) so RegisterRun's fsync
// step has real files to open, and returns them for a test to pass to
// RegisterRun.
func setUpRegisteredRun(t *testing.T, epicKey, featureKey, featureCommit string) (*IntegrationRun, *IntegrationEvent, *IntegrationCandidate) {
	t.Helper()

	run, err := CaptureBase(epicKey)
	if err != nil {
		t.Fatalf("CaptureBase: %v", err)
	}
	event, err := RecordEvent(run.EpicRunID, featureKey, featureCommit, nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	head, err := UpdateCandidate(run.EpicRunID, event)
	if err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}
	return run, event, head
}

// TestDeriveRegistrationSuboperationID_DeterministicAndDistinguishesInputs
// covers TC-016 / AC-T2's suboperation-ID formula: identical inputs derive
// an identical ID (T-E34-F08-014's repair path depends on this stability),
// and a different firstHeadDigest derives a different ID.
func TestDeriveRegistrationSuboperationID_DeterministicAndDistinguishesInputs(t *testing.T) {
	a := DeriveRegistrationSuboperationID("E01", "run-1", "base-1", "head-1")
	b := DeriveRegistrationSuboperationID("E01", "run-1", "base-1", "head-1")
	if a != b {
		t.Fatalf("DeriveRegistrationSuboperationID is not deterministic: %q != %q", a, b)
	}

	c := DeriveRegistrationSuboperationID("E01", "run-1", "base-1", "head-2")
	if a == c {
		t.Fatalf("expected a different suboperation ID for a different firstHeadDigest, got the same: %q", a)
	}
}

// TestRegisterRun_InsertsNoteWithSuboperationIDAndContent covers TC-016 /
// AC-T2's content/metadata shape: the inserted note's metadata carries the
// deterministic suboperation ID, and its content carries record_kind, the
// run's EpicRunID/BaseCommit, and firstHead's Digest.
func TestRegisterRun_InsertsNoteWithSuboperationIDAndContent(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run, event, head := setUpRegisteredRun(t, "E78", "E78-F01", "commit-x")

	recorder := &fakeNoteRecorder{}
	note, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if err != nil {
		t.Fatalf("RegisterRun: %v", err)
	}
	if note == nil {
		t.Fatal("expected a non-nil note")
	}

	wantSubID := DeriveRegistrationSuboperationID(run.EpicKey, run.EpicRunID, run.BaseCommit, head.Digest)

	var metadata RegistrationNoteMetadata
	if err := json.Unmarshal([]byte(recorder.lastMetadata), &metadata); err != nil {
		t.Fatalf("note metadata is not valid JSON: %v", err)
	}
	if metadata.SuboperationID != wantSubID {
		t.Fatalf("metadata suboperation_id = %q, want %q", metadata.SuboperationID, wantSubID)
	}

	var content RegistrationNoteContent
	if err := json.Unmarshal([]byte(recorder.lastContent), &content); err != nil {
		t.Fatalf("note content is not valid JSON: %v", err)
	}
	if content.RecordKind != RecordKindIntegrationCandidateRoot {
		t.Fatalf("content record_kind = %q, want %q", content.RecordKind, RecordKindIntegrationCandidateRoot)
	}
	if content.EpicRunID != run.EpicRunID {
		t.Fatalf("content epic_run_id = %q, want %q", content.EpicRunID, run.EpicRunID)
	}
	if content.BaseCommit != run.BaseCommit {
		t.Fatalf("content base_commit = %q, want %q", content.BaseCommit, run.BaseCommit)
	}
	if content.HeadDigest != head.Digest {
		t.Fatalf("content head_digest = %q, want %q", content.HeadDigest, head.Digest)
	}
}

// TestRegisterRun_FailsIfSidecarFilesMissing covers AC-T2's ordering
// requirement from the other direction: if the event/head files RegisterRun
// is supposed to fsync do not actually exist on disk, it must fail before
// ever calling the note recorder — proving the fsync step is a real
// precondition, not a no-op ahead of the note insert.
func TestRegisterRun_FailsIfSidecarFilesMissing(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run := &IntegrationRun{EpicRunID: "run-missing-sidecars", EpicKey: "E79", BaseCommit: "deadbeef"}
	event := &IntegrationEvent{EventID: "nonexistent-event-id"}
	head := &IntegrationCandidate{Digest: "nonexistent-digest"}
	recorder := &fakeNoteRecorder{}

	_, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if err == nil {
		t.Fatal("expected an error when the event/head sidecar files do not exist on disk")
	}
	if recorder.calls != 0 {
		t.Fatalf("note must not be inserted when fsync fails before it: got %d calls, want 0", recorder.calls)
	}
}

// TestRegisterRun_ConcurrentAttemptsSerialize covers TC-016's registration-
// lock assertion end to end through RegisterRun itself (not just
// AcquireRegistrationLock directly, as lock_test.go covers): several
// goroutines call RegisterRun concurrently for the same run, and the fake
// recorder's max-in-flight counter must never exceed 1 — the run-scoped
// lock serializes the entire fsync-then-note-insert sequence.
func TestRegisterRun_ConcurrentAttemptsSerialize(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run, event, head := setUpRegisteredRun(t, "E80", "E80-F01", "commit-y")
	recorder := &fakeNoteRecorder{}

	const goroutines = 6
	var start, done sync.WaitGroup
	errs := make([]error, goroutines)
	start.Add(1)
	done.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer done.Done()
			start.Wait() // barrier: release all goroutines together
			_, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
			errs[i] = err
		}()
	}
	start.Done() // release the barrier
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: RegisterRun error: %v", i, err)
		}
	}

	if recorder.maxInFlight > 1 {
		t.Fatalf("registration lock did not serialize concurrent attempts: observed %d concurrent note inserts, want at most 1", recorder.maxInFlight)
	}
	if recorder.calls != goroutines {
		t.Fatalf("expected %d note-insert calls (this test does not check idempotent dedup — that is T-E34-F08-014's scope), got %d", goroutines, recorder.calls)
	}
}
