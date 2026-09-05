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

// fakeStoredNote is one note "persisted" by fakeNoteRecorder.AddNoteWithMetadata,
// enough for fakeNoteRecorder.ListNotes to filter and return it the way
// *services.NoteService.ListNotes would.
type fakeStoredNote struct {
	entityType string
	entityKey  string
	noteType   string
	content    string
	metadata   string
}

// fakeNoteRecorder is a NoteRecorder test double that records each call's
// arguments, tracks the maximum number of concurrently in-flight
// AddNoteWithMetadata calls (so a concurrency test can observe whether
// RegisterRun's lock actually serialized real racing goroutines without the
// test itself imposing any serialization), and — added by task
// T-E34-F08-014 — stores successfully-inserted notes so ListNotes can
// return them, letting tests exercise RegisterRun's existing-note lookup
// (existingRegistrationNote) without a real database. Tests in the same
// package may also seed/mutate `notes` directly to construct a conflicting
// or tampered pre-existing note.
type fakeNoteRecorder struct {
	mu           sync.Mutex
	calls        int
	inFlight     int
	maxInFlight  int
	lastContent  string
	lastMetadata string
	err          error
	listErr      error
	notes        []fakeStoredNote
}

func (f *fakeNoteRecorder) AddNoteWithMetadata(_ context.Context, entityType models.EntityType, entityKey string, noteType string, content string, _ string, metadata string) (*models.EntityNote, error) {
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
	if err == nil {
		f.notes = append(f.notes, fakeStoredNote{
			entityType: string(entityType),
			entityKey:  entityKey,
			noteType:   noteType,
			content:    content,
			metadata:   metadata,
		})
	}
	f.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return &models.EntityNote{EntityType: entityType, NoteType: models.NoteType(noteType), Content: content}, nil
}

// ListNotes returns the notes stored so far matching entityType/entityKey,
// filtered to noteTypes when non-empty — mirroring
// *services.NoteService.ListNotes's own filtering behavior closely enough
// for RegisterRun's existingRegistrationNote lookup.
func (f *fakeNoteRecorder) ListNotes(_ context.Context, entityType models.EntityType, entityKey string, noteTypes []string) ([]*models.EntityNote, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.listErr != nil {
		return nil, f.listErr
	}

	wantTypes := make(map[string]bool, len(noteTypes))
	for _, t := range noteTypes {
		wantTypes[t] = true
	}

	var out []*models.EntityNote
	for _, n := range f.notes {
		if n.entityType != string(entityType) || n.entityKey != entityKey {
			continue
		}
		if len(wantTypes) > 0 && !wantTypes[n.noteType] {
			continue
		}
		metadata := n.metadata
		out = append(out, &models.EntityNote{
			EntityType: entityType,
			NoteType:   models.NoteType(n.noteType),
			Content:    n.content,
			Metadata:   &metadata,
		})
	}
	return out, nil
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
//
// It also covers T-E34-F08-014 AC-T1's idempotent-retry requirement under
// real concurrency, not just sequential retries: only the first goroutine to
// hold the lock ever finds no existing note and calls AddNoteWithMetadata;
// every later goroutine's existingRegistrationNote lookup (also taken under
// the same lock) finds the note the first goroutine just inserted and
// returns it without a second insert. Before T-E34-F08-014 this assertion
// was `recorder.calls != goroutines` with a comment marking dedup as this
// task's scope — now that RegisterRun dedups, exactly one insert is correct.
func TestRegisterRun_ConcurrentAttemptsSerialize(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run, event, head := setUpRegisteredRun(t, "E80", "E80-F01", "commit-y")
	recorder := &fakeNoteRecorder{}

	const goroutines = 6
	var start, done sync.WaitGroup
	errs := make([]error, goroutines)
	notes := make([]*models.EntityNote, goroutines)
	start.Add(1)
	done.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer done.Done()
			start.Wait() // barrier: release all goroutines together
			note, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
			notes[i] = note
			errs[i] = err
		}()
	}
	start.Done() // release the barrier
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: RegisterRun error: %v", i, err)
		}
		if notes[i] == nil {
			t.Fatalf("goroutine %d: expected a non-nil note", i)
		}
	}

	if recorder.maxInFlight > 1 {
		t.Fatalf("registration lock did not serialize concurrent attempts: observed %d concurrent note inserts, want at most 1", recorder.maxInFlight)
	}
	if recorder.calls != 1 {
		t.Fatalf("expected exactly 1 note-insert call across %d concurrent attempts (idempotent dedup, T-E34-F08-014 AC-T1), got %d", goroutines, recorder.calls)
	}
}

// TestRegisterRun_ExactRetry_RepairsMissingNoteThenIdempotent covers AC-T1's
// core wording: "exact retry (matching head, no note) repairs only the
// missing note." The first attempt fails after the head is already durable
// but before the note is ever acknowledged (recorder.err simulates the
// note-insert step itself failing/crashing — fsync already happened, so
// this models "failure-injected ... before note ack" without a hand-
// constructed note). The retry ("restart") finds no note yet and repairs by
// inserting exactly one. A further retry, now that the note exists and
// matches, must be a pure idempotent no-op — not a second insert.
func TestRegisterRun_ExactRetry_RepairsMissingNoteThenIdempotent(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run, event, head := setUpRegisteredRun(t, "E93", "E93-F01", "commit-1")
	recorder := &fakeNoteRecorder{err: errors.New("simulated note-ack failure")}

	if _, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent"); err == nil {
		t.Fatal("expected the simulated note-ack failure to propagate")
	}
	if recorder.calls != 1 {
		t.Fatalf("expected 1 attempted insert (which failed), got %d", recorder.calls)
	}
	if len(recorder.notes) != 0 {
		t.Fatalf("a failed insert must not be stored: got %d stored notes", len(recorder.notes))
	}

	// "Restart": the note-ack failure is gone now. Exact retry with a
	// matching head and no note repairs only the missing note.
	recorder.err = nil
	repaired, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if err != nil {
		t.Fatalf("repair retry: %v", err)
	}
	if repaired == nil {
		t.Fatal("expected a non-nil repaired note")
	}
	if recorder.calls != 2 {
		t.Fatalf("expected exactly 2 total AddNoteWithMetadata calls (1 failed + 1 repair), got %d", recorder.calls)
	}
	if len(recorder.notes) != 1 {
		t.Fatalf("expected exactly 1 stored note after repair, got %d", len(recorder.notes))
	}

	// A further retry, now that the note already exists and matches, must
	// not insert a second note.
	again, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if err != nil {
		t.Fatalf("idempotent retry after repair: %v", err)
	}
	if again == nil {
		t.Fatal("expected a non-nil note on the idempotent retry")
	}
	if recorder.calls != 2 {
		t.Fatalf("idempotent retry must not insert a second note: got %d total calls, want 2", recorder.calls)
	}
}

// TestRegisterRun_FailureInjectedBeforeNoteAck_RestartRepairs drives the
// same repair scenario through registerRunTestHook instead of a recorder
// error, directly covering test-plan.md TC-017's Caller-Path Contract
// ("failure-injected via ... test hooks ... after candidate-head
// replacement before note ack"): the hook fires after both the event and
// head files are already fsynced (durable), simulating the process dying at
// exactly that point, before RegisterRun ever calls the note recorder.
func TestRegisterRun_FailureInjectedBeforeNoteAck_RestartRepairs(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run, event, head := setUpRegisteredRun(t, "E94", "E94-F01", "commit-1")

	injected := errors.New("simulated crash: fsync done, note never acknowledged")
	registerRunTestHook = func() error { return injected }
	t.Cleanup(func() { registerRunTestHook = nil })

	recorder := &fakeNoteRecorder{}
	_, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if !errors.Is(err, injected) {
		t.Fatalf("expected the injected failure to propagate, got %v", err)
	}
	if recorder.calls != 0 {
		t.Fatalf("the note recorder must never be called before the injected hook fires: got %d calls", recorder.calls)
	}

	// "Restart": disable the injected failure and retry.
	registerRunTestHook = nil
	note, err := RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if err != nil {
		t.Fatalf("retry after simulated restart: %v", err)
	}
	if note == nil {
		t.Fatal("expected a non-nil repaired note")
	}
	if recorder.calls != 1 {
		t.Fatalf("expected exactly 1 note-insert call after the restart repair, got %d", recorder.calls)
	}
}

// TestRegisterRun_ConflictingHeadForSameRun_FailsClosed covers AC-T1's
// "conflicting bytes ... fails closed without reconstruction": an existing
// registration note for the same EpicRunID but a different HeadDigest (as
// if two different heads were somehow registered for one run) is rejected,
// and RegisterRun never attempts to insert a corrective note.
func TestRegisterRun_ConflictingHeadForSameRun_FailsClosed(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run, event, head := setUpRegisteredRun(t, "E95", "E95-F01", "commit-1")

	recorder := &fakeNoteRecorder{}
	conflictingContent, err := json.Marshal(RegistrationNoteContent{
		RecordKind: RecordKindIntegrationCandidateRoot,
		EpicRunID:  run.EpicRunID,
		BaseCommit: run.BaseCommit,
		HeadDigest: "a-different-head-digest-entirely",
	})
	if err != nil {
		t.Fatalf("marshal conflicting content: %v", err)
	}
	recorder.notes = append(recorder.notes, fakeStoredNote{
		entityType: string(models.EntityTypeEpic),
		entityKey:  run.EpicKey,
		noteType:   string(models.NoteTypeReference),
		content:    string(conflictingContent),
	})

	_, err = RegisterRun(context.Background(), recorder, run, head, event, "test-agent")
	if err == nil {
		t.Fatal("expected a conflict error for a mismatched head digest on the same run")
	}
	var conflict *RegistrationConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
	}
	if recorder.calls != 0 {
		t.Fatalf("a conflict must never attempt to insert a note: got %d calls", recorder.calls)
	}
}

// TestRegisterRun_SecondEpicRunIDWhileNonterminal_Rejected covers AC-T2: a
// second registration for a different EpicRunID for the same epic, while
// the first is still registered (nonterminal), is rejected. The two
// IntegrationRun/event/head sets are constructed independently — this test
// does not rely on CaptureBase's own single-run-per-epic idempotency, since
// that would make a second EpicRunID for the same epic impossible to
// produce in the first place; RecordEvent/UpdateCandidate accept any
// epicRunID directly.
func TestRegisterRun_SecondEpicRunIDWhileNonterminal_Rejected(t *testing.T) {
	_, _ = chdirProjectRoot(t)

	run1, event1, head1 := setUpRegisteredRun(t, "E96", "E96-F01", "commit-1")
	recorder := &fakeNoteRecorder{}

	if _, err := RegisterRun(context.Background(), recorder, run1, head1, event1, "test-agent"); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	if recorder.calls != 1 {
		t.Fatalf("expected 1 call after the first registration, got %d", recorder.calls)
	}

	run2 := &IntegrationRun{EpicRunID: "e96-second-run", EpicKey: "E96", BaseCommit: run1.BaseCommit}
	event2, err := RecordEvent(run2.EpicRunID, "E96-F02", "commit-2", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent for second run: %v", err)
	}
	head2, err := UpdateCandidate(run2.EpicRunID, event2)
	if err != nil {
		t.Fatalf("UpdateCandidate for second run: %v", err)
	}

	_, err = RegisterRun(context.Background(), recorder, run2, head2, event2, "test-agent")
	if err == nil {
		t.Fatal("expected the second EpicRunID's registration to be rejected while the first is nonterminal")
	}
	var conflict *RegistrationConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
	}
	if recorder.calls != 1 {
		t.Fatalf("the rejected second registration must not insert a note: got %d total calls, want 1", recorder.calls)
	}
}

// TestRegisterRun_ArchivedHeadRecomputableThenTamperTruncationReorderingFailClosed
// covers TC-017's "recomputable prior-head chain" happy path together with
// its tamper/truncation/reordering fail-closed requirements, all against
// the retained integration-heads/ archive T-E34-F08-012 writes.
//
// Setup folds three events into one run so the first two heads are archived
// (superseded) rather than live. The subtests below register against the
// oldest, fully-archived head — proving it is still recomputable purely
// from retained bytes — then corrupt that archive in three different ways
// and confirm each one fails closed with a *RegistrationConflictError
// rather than reconstructing anything. "Reordering" here means swapping two
// archived heads' file contents between each other's digest-named files:
// content-addressed storage makes this indistinguishable from tamper (the
// swapped content no longer hashes to the name it landed on), so it is
// covered by the same verifyRetainedDigest check exercised by the tamper
// subtest — a second, dedicated subtest documents that this specific attack
// shape is caught too, per test-plan.md TC-017's explicit "reordering"
// wording, without a second detection code path (Rule 2: simplicity).
func TestRegisterRun_ArchivedHeadRecomputableThenTamperTruncationReorderingFailClosed(t *testing.T) {
	dir, _ := chdirProjectRoot(t)

	const epicRunID = "run-e97-archive-chain"
	event1, err := RecordEvent(epicRunID, "E97-F01", "commit-1", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent 1: %v", err)
	}
	head1, err := UpdateCandidate(epicRunID, event1)
	if err != nil {
		t.Fatalf("UpdateCandidate 1: %v", err)
	}

	event2, err := RecordEvent(epicRunID, "E97-F02", "commit-2", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent 2: %v", err)
	}
	head2, err := UpdateCandidate(epicRunID, event2)
	if err != nil {
		t.Fatalf("UpdateCandidate 2: %v", err)
	}

	event3, err := RecordEvent(epicRunID, "E97-F03", "commit-3", nil, nil)
	if err != nil {
		t.Fatalf("RecordEvent 3: %v", err)
	}
	if _, err := UpdateCandidate(epicRunID, event3); err != nil {
		t.Fatalf("UpdateCandidate 3: %v", err)
	}

	run := &IntegrationRun{EpicRunID: epicRunID, EpicKey: "E97", BaseCommit: head1.BaseCommit}
	archiveDir := candidateHeadsDir(candidatePath(dir, epicRunID))
	head1ArchivePath := filepath.Join(archiveDir, head1.Digest+".json")
	head2ArchivePath := filepath.Join(archiveDir, head2.Digest+".json")

	t.Run("recomputable from archive", func(t *testing.T) {
		recorder := &fakeNoteRecorder{}
		note, err := RegisterRun(context.Background(), recorder, run, head1, event1, "test-agent")
		if err != nil {
			t.Fatalf("RegisterRun against a fully-archived (non-live) head: %v", err)
		}
		if note == nil {
			t.Fatal("expected a non-nil note")
		}
	})

	t.Run("tampered archive fails closed", func(t *testing.T) {
		original, err := os.ReadFile(head2ArchivePath)
		if err != nil {
			t.Fatalf("read archived head2 before tampering: %v", err)
		}
		t.Cleanup(func() {
			if err := os.WriteFile(head2ArchivePath, original, runFileMode); err != nil {
				t.Fatalf("restore tampered archive: %v", err)
			}
		})
		if err := os.WriteFile(head2ArchivePath, []byte(`{"epic_run_id":"tampered","digest":"`+head2.Digest+`"}`), runFileMode); err != nil {
			t.Fatalf("tamper archived head2: %v", err)
		}

		recorder := &fakeNoteRecorder{}
		_, err = RegisterRun(context.Background(), recorder, run, head2, event2, "test-agent")
		if err == nil {
			t.Fatal("expected tamper detection to fail closed")
		}
		var conflict *RegistrationConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
		}
		if recorder.calls != 0 {
			t.Fatalf("a tampered head must never reach the note recorder: got %d calls", recorder.calls)
		}
	})

	t.Run("truncated archive fails closed", func(t *testing.T) {
		original, err := os.ReadFile(head1ArchivePath)
		if err != nil {
			t.Fatalf("read archived head1 before truncating: %v", err)
		}
		t.Cleanup(func() {
			if err := os.WriteFile(head1ArchivePath, original, runFileMode); err != nil {
				t.Fatalf("restore truncated archive: %v", err)
			}
		})
		if err := os.WriteFile(head1ArchivePath, []byte(`{"epic_run`), runFileMode); err != nil {
			t.Fatalf("truncate archived head1: %v", err)
		}

		recorder := &fakeNoteRecorder{}
		_, err = RegisterRun(context.Background(), recorder, run, head1, event1, "test-agent")
		if err == nil {
			t.Fatal("expected truncation detection to fail closed")
		}
		var conflict *RegistrationConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
		}
		if recorder.calls != 0 {
			t.Fatalf("a truncated head must never reach the note recorder: got %d calls", recorder.calls)
		}
	})

	t.Run("reordered (swapped) archive contents fail closed", func(t *testing.T) {
		head1Bytes, err := os.ReadFile(head1ArchivePath)
		if err != nil {
			t.Fatalf("read archived head1: %v", err)
		}
		head2Bytes, err := os.ReadFile(head2ArchivePath)
		if err != nil {
			t.Fatalf("read archived head2: %v", err)
		}
		t.Cleanup(func() {
			_ = os.WriteFile(head1ArchivePath, head1Bytes, runFileMode)
			_ = os.WriteFile(head2ArchivePath, head2Bytes, runFileMode)
		})

		// Swap: head1's filename now holds head2's bytes and vice versa.
		if err := os.WriteFile(head1ArchivePath, head2Bytes, runFileMode); err != nil {
			t.Fatalf("swap onto head1 archive: %v", err)
		}
		if err := os.WriteFile(head2ArchivePath, head1Bytes, runFileMode); err != nil {
			t.Fatalf("swap onto head2 archive: %v", err)
		}

		for _, tc := range []struct {
			name string
			head *IntegrationCandidate
			evt  *IntegrationEvent
		}{
			{"head1 slot", head1, event1},
			{"head2 slot", head2, event2},
		} {
			t.Run(tc.name, func(t *testing.T) {
				recorder := &fakeNoteRecorder{}
				_, err := RegisterRun(context.Background(), recorder, run, tc.head, tc.evt, "test-agent")
				if err == nil {
					t.Fatal("expected reordering (swapped archive contents) to fail closed")
				}
				var conflict *RegistrationConflictError
				if !errors.As(err, &conflict) {
					t.Fatalf("expected *RegistrationConflictError, got %T: %v", err, err)
				}
				if recorder.calls != 0 {
					t.Fatalf("swapped archive content must never reach the note recorder: got %d calls", recorder.calls)
				}
			})
		}
	})
}
