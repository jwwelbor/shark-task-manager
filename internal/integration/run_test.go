package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
