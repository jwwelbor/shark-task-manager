package gaterun

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestAncestorSymlinkSwap_NeverFollowsLink is the ancestor-directory
// counterpart of TestReadResult_RejectsSymlinkTarget/TestCreateResult_
// RejectsSymlinkTarget (which only cover the *leaf* result.json /
// operation-state.json files, closed by the first T-E34-F05-002 rework
// round, commit c40f3b98). UAT's red-team found a broader, different
// instance of the same defect class: runid.go's ensureRealDir and fsio.go's
// path-reuse both verified a *directory* ancestor component (.shark,
// .shark/runs) via Lstat and then separately reused that same path string
// later (in RunDir's own chmod, and in every fsio.go entry point called
// against a directory RunDir had already returned in a prior call/process).
// A concurrent same-UID process can swap one of those ancestor components
// for a symlink in the window between the check and the reuse.
//
// This test reproduces that window directly: it builds a real run directory
// via RunDir (which performs its own ancestor checks and returns), then
// swaps the "runs" ancestor for a symlink to an attacker-controlled
// directory *after* RunDir's checks have already passed, and asserts that
// every entry point in this package that is later called against the same
// dir string refuses to follow the swapped ancestor rather than silently
// operating inside the attacker directory.
func TestAncestorSymlinkSwap_NeverFollowsLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ancestor no-follow descent targets the POSIX openat(O_NOFOLLOW|O_DIRECTORY) fix; windows keeps the documented TD-181 residual gap")
	}

	newSwappedDir := func(t *testing.T) (dir, attacker string) {
		t.Helper()
		root := t.TempDir()
		dir, err := RunDir(root, "run-1")
		if err != nil {
			t.Fatalf("RunDir: %v", err)
		}
		runsDir := filepath.Dir(dir) // .shark/runs — the ancestor directly above the leaf

		attacker = filepath.Join(t.TempDir(), "attacker")
		if err := os.Mkdir(attacker, 0o700); err != nil {
			t.Fatalf("mkdir attacker: %v", err)
		}

		// Swap the already-verified "runs" ancestor for a symlink to the
		// attacker directory, reproducing the window a concurrent process
		// could win between RunDir's ancestor check and a later call (in
		// this process or another) that reuses the same dir string.
		if err := os.RemoveAll(runsDir); err != nil {
			t.Fatalf("remove runs dir: %v", err)
		}
		if err := os.Symlink(attacker, runsDir); err != nil {
			t.Fatalf("symlink runs -> attacker: %v", err)
		}
		return dir, attacker
	}

	assertUnsafe := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("want error through swapped ancestor, got nil")
		}
		var unsafeErr *UnsafePathError
		if !errors.As(err, &unsafeErr) {
			t.Errorf("error = %v (%T), want *UnsafePathError", err, err)
		}
	}

	assertAttackerEmpty := func(t *testing.T, attacker string) {
		t.Helper()
		entries, err := os.ReadDir(attacker)
		if err != nil {
			t.Fatalf("readdir attacker: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("attacker dir has %d entries, want 0 — a write escaped through the swapped ancestor", len(entries))
		}
	}

	t.Run("CreateResult", func(t *testing.T) {
		dir, attacker := newSwappedDir(t)
		_, err := CreateResult(dir, []byte(`{"a":1}`))
		assertUnsafe(t, err)
		assertAttackerEmpty(t, attacker)
	})

	t.Run("ReadResult", func(t *testing.T) {
		dir, _ := newSwappedDir(t)
		_, _, err := ReadResult(dir)
		assertUnsafe(t, err)
	})

	t.Run("WriteOperationState", func(t *testing.T) {
		dir, attacker := newSwappedDir(t)
		err := WriteOperationState(dir, []byte(`{"phase":"x"}`))
		assertUnsafe(t, err)
		assertAttackerEmpty(t, attacker)
	})

	t.Run("ReadOperationState", func(t *testing.T) {
		dir, _ := newSwappedDir(t)
		_, _, err := ReadOperationState(dir)
		assertUnsafe(t, err)
	})

	t.Run("AcquireRunLock", func(t *testing.T) {
		dir, attacker := newSwappedDir(t)
		_, err := AcquireRunLock(dir, 100*time.Millisecond)
		assertUnsafe(t, err)
		assertAttackerEmpty(t, attacker)
	})
}

// TestCreateResult_AncestorSymlinkSwapRace_NeverFollowsLink is the live-race
// variant of TestAncestorSymlinkSwap_NeverFollowsLink, mirroring
// TestReadResult_SymlinkSwapRaceNeverFollowsLink's style: a goroutine
// continuously swaps the "runs" ancestor between a real directory and a
// symlink to an attacker directory while a tight CreateResult loop runs
// against the leaf beneath it. Against the pre-fix Lstat-then-path-reopen
// code this reliably lets a write land inside the attacker directory within
// the budget below; against the fixed openat(O_NOFOLLOW|O_DIRECTORY) descent
// it must never happen.
func TestCreateResult_AncestorSymlinkSwapRace_NeverFollowsLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ancestor no-follow descent targets the POSIX openat(O_NOFOLLOW|O_DIRECTORY) fix")
	}

	root := t.TempDir()
	dir, err := RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	runsDir := filepath.Dir(dir)
	runsParent := filepath.Dir(runsDir)
	runIDName := filepath.Base(dir)

	attacker := filepath.Join(t.TempDir(), "attacker")
	if err := os.Mkdir(attacker, 0o700); err != nil {
		t.Fatalf("mkdir attacker: %v", err)
	}

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		linkTmp := runsDir + ".linktmp"
		for {
			select {
			case <-stop:
				return
			default:
			}
			// Swap "runs" for a symlink to the attacker directory.
			_ = os.RemoveAll(linkTmp)
			if err := os.Symlink(attacker, linkTmp); err == nil {
				_ = os.RemoveAll(runsDir)
				_ = os.Rename(linkTmp, runsDir)
			}
			// Swap back to a legitimate "runs" directory with the run's
			// leaf directory recreated beneath it.
			_ = os.RemoveAll(runsDir)
			if err := os.MkdirAll(filepath.Join(runsParent, "runs", runIDName), 0o700); err == nil {
				_ = os.Rename(filepath.Join(runsParent, "runs"), runsDir)
			}
		}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = CreateResult(dir, []byte(`{"race":true}`))
		entries, err := os.ReadDir(attacker)
		if err == nil && len(entries) != 0 {
			t.Fatalf("attacker dir received %d entries: TOCTOU ancestor symlink follow observed", len(entries))
		}
	}
}
