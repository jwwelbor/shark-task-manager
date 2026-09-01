package gaterun

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

func newRunDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir, err := RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	return dir
}

func TestCreateResult_CreatesOnceWithOwnerOnlyMode(t *testing.T) {
	dir := newRunDir(t)
	created, err := CreateResult(dir, []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	if !created {
		t.Error("created = false, want true on first call")
	}
	path := filepath.Join(dir, resultFileName)
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat result.json: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("result.json mode = %o, want 0600", perm)
	}
}

func TestCreateResult_NoLeftoverTempFile(t *testing.T) {
	dir := newRunDir(t)
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != resultFileName {
			t.Errorf("unexpected leftover entry after CreateResult: %s", e.Name())
		}
	}
}

func TestCreateResult_IdenticalReplayIsIdempotent(t *testing.T) {
	dir := newRunDir(t)
	data := []byte(`{"a":1}`)
	if _, err := CreateResult(dir, data); err != nil {
		t.Fatalf("first create: %v", err)
	}
	created, err := CreateResult(dir, data)
	if err != nil {
		t.Fatalf("identical replay: want nil error, got %v", err)
	}
	if created {
		t.Error("created = true on identical replay, want false (idempotent)")
	}
}

func TestCreateResult_ConflictingReplayRejectedWithoutOverwrite(t *testing.T) {
	dir := newRunDir(t)
	original := []byte(`{"a":1}`)
	if _, err := CreateResult(dir, original); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := CreateResult(dir, []byte(`{"a":2}`))
	if err == nil {
		t.Fatal("conflicting replay: want error, got nil")
	}
	if !IsConflict(err) {
		t.Errorf("conflicting replay error = %v, want *ConflictError", err)
	}
	// The accepted bytes must be untouched.
	got, exists, rerr := ReadResult(dir)
	if rerr != nil || !exists {
		t.Fatalf("ReadResult after conflict: exists=%v err=%v", exists, rerr)
	}
	if string(got) != string(original) {
		t.Errorf("result.json bytes changed after rejected conflict: got %s, want %s", got, original)
	}
}

func TestCreateResult_RejectsSymlinkTarget(t *testing.T) {
	dir := newRunDir(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(dir, resultFileName)
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err == nil {
		t.Fatal("CreateResult over symlinked target: want error, got nil")
	}
}

func TestCreateResult_RejectsNonRegularExistingTarget(t *testing.T) {
	dir := newRunDir(t)
	subdir := filepath.Join(dir, resultFileName)
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err == nil {
		t.Fatal("CreateResult over a directory target: want error, got nil")
	}
}

func TestCreateResult_RejectsOversizedRead(t *testing.T) {
	dir := newRunDir(t)
	// Write a pre-existing result.json larger than the bound directly (not
	// through CreateResult, to exercise the EEXIST/existing-target path).
	path := filepath.Join(dir, resultFileName)
	big := make([]byte, maxSidecarBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(path, big, 0o600); err != nil {
		t.Fatalf("write oversized file: %v", err)
	}
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err == nil {
		t.Fatal("CreateResult against oversized existing target: want error, got nil")
	}
}

func TestCreateResult_RejectsEmptyData(t *testing.T) {
	dir := newRunDir(t)
	if _, err := CreateResult(dir, nil); err == nil {
		t.Fatal("CreateResult with empty data: want error, got nil")
	}
}

func TestCreateResult_RaceFirstWriterWins(t *testing.T) {
	dir := newRunDir(t)
	const n = 20
	var wg sync.WaitGroup
	results := make([]bool, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			created, err := CreateResult(dir, []byte(`{"race":true}`))
			results[i] = created
			errs[i] = err
		}(i)
	}
	wg.Wait()

	createdCount := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d: unexpected error for identical racing payload: %v", i, errs[i])
		}
		if results[i] {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Errorf("createdCount = %d, want exactly 1 (first-writer-wins)", createdCount)
	}
}

func TestCreateResult_RaceConflictingPayloadsRejectAllButOne(t *testing.T) {
	dir := newRunDir(t)
	const n = 10
	var wg sync.WaitGroup
	created := make([]bool, n)
	conflicted := make([]bool, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			payload := []byte(`{"race":` + string(rune('0'+i)) + `}`)
			c, err := CreateResult(dir, payload)
			created[i] = c
			conflicted[i] = err != nil && IsConflict(err)
		}(i)
	}
	wg.Wait()

	createdCount := 0
	for i := 0; i < n; i++ {
		if created[i] {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Errorf("createdCount = %d, want exactly 1 across conflicting concurrent payloads", createdCount)
	}
}

func TestWriteOperationState_AtomicReplaceAndReadBack(t *testing.T) {
	dir := newRunDir(t)
	if err := WriteOperationState(dir, []byte(`{"phase":"start"}`)); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteOperationState(dir, []byte(`{"phase":"done"}`)); err != nil {
		t.Fatalf("second write (replace): %v", err)
	}
	got, exists, err := ReadOperationState(dir)
	if err != nil || !exists {
		t.Fatalf("ReadOperationState: exists=%v err=%v", exists, err)
	}
	if string(got) != `{"phase":"done"}` {
		t.Errorf("operation-state.json = %s, want the latest replace", got)
	}

	path := filepath.Join(dir, operationStateFileName)
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("operation-state.json mode = %o, want 0600", perm)
	}

	// No leftover temp files after a clean replace.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != resultFileName && e.Name() != operationStateFileName {
			t.Errorf("unexpected leftover entry after atomic replace: %s", e.Name())
		}
	}
}

func TestWriteOperationState_RejectsSymlinkTarget(t *testing.T) {
	dir := newRunDir(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(dir, operationStateFileName)
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := WriteOperationState(dir, []byte(`{"phase":"x"}`)); err == nil {
		t.Fatal("WriteOperationState over symlinked target: want error, got nil")
	}
}

// TestReadResult_RejectsSymlinkTarget closes the TOCTOU class the write path
// already defeats (see TestCreateResult_RejectsSymlinkTarget): a symlink
// planted at result.json must never be transparently followed by a read.
// Before the fix, readRegularBounded performed a separate os.Lstat followed
// by a plain os.Open, which *does* follow symlinks — a same-UID concurrent
// process could swap the lstat'd target for a symlink in the window between
// the two syscalls and have it silently dereferenced. This test uses a
// symlink planted before the read (rather than a live race) to deterministically
// prove the read path rejects it via a no-follow open, per the guidance that a
// true race test would be flaky.
func TestReadResult_RejectsSymlinkTarget(t *testing.T) {
	dir := newRunDir(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(dir, resultFileName)
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, _, err := ReadResult(dir); err == nil {
		t.Fatal("ReadResult over symlinked target: want error, got nil")
	} else {
		var unsafeErr *UnsafePathError
		if !errors.As(err, &unsafeErr) {
			t.Errorf("ReadResult over symlinked target error = %v, want *UnsafePathError", err)
		}
	}
}

// TestReadOperationState_RejectsSymlinkTarget is the operation-state.json
// counterpart of TestReadResult_RejectsSymlinkTarget — see that test's
// comment for the defect this closes.
func TestReadOperationState_RejectsSymlinkTarget(t *testing.T) {
	dir := newRunDir(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(dir, operationStateFileName)
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, _, err := ReadOperationState(dir); err == nil {
		t.Fatal("ReadOperationState over symlinked target: want error, got nil")
	} else {
		var unsafeErr *UnsafePathError
		if !errors.As(err, &unsafeErr) {
			t.Errorf("ReadOperationState over symlinked target error = %v, want *UnsafePathError", err)
		}
	}
}

// TestReadResult_RejectsFIFOTarget guards against the no-follow open
// blocking indefinitely on a FIFO opened for read with no writer connected
// (a plain os.OpenFile(O_RDONLY) on a FIFO blocks until a writer appears).
// The fix must open with O_NONBLOCK so this returns promptly with an
// UnsafePathError instead of hanging the test (and, in production, the
// calling command).
func TestReadResult_RejectsFIFOTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are not available on windows")
	}
	dir := newRunDir(t)
	path := filepath.Join(dir, resultFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, _, err := ReadResult(dir); err == nil {
			t.Error("ReadResult over a FIFO target: want error, got nil")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ReadResult over a FIFO target blocked instead of returning an error")
	}
}

// TestReadResult_SymlinkSwapRaceNeverFollowsLink is the actual TOCTOU
// reproduction: the two static "rejects symlink" tests above only prove the
// read path catches a symlink that is *already* at result.json when the read
// starts — which the buggy Lstat-then-Open code also caught, since Lstat
// alone rejects a link seen at rest. The real defect is the window between
// that Lstat and the later Open: a same-UID concurrent process can swap the
// lstat'd regular file for a symlink to secretPath after the Lstat check
// passes but before Open runs, and a plain os.Open (which follows symlinks)
// silently dereferences it. This test races a goroutine that continuously
// swaps result.json between a regular file and a symlink to an outside
// "secret" file against a tight ReadResult loop; against the pre-fix code it
// reliably observes the secret content within the budget below, and against
// the fixed no-follow-open code it must never observe it.
func TestReadResult_SymlinkSwapRaceNeverFollowsLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink-swap race targets the POSIX no-follow-open fix")
	}
	dir := newRunDir(t)
	path := filepath.Join(dir, resultFileName)

	secretDir := t.TempDir()
	secretPath := filepath.Join(secretDir, "secret.json")
	secret := []byte(`{"secret":true}`)
	if err := os.WriteFile(secretPath, secret, 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	legit := []byte(`{"legit":true}`)
	if err := os.WriteFile(path, legit, 0o600); err != nil {
		t.Fatalf("write initial legit result.json: %v", err)
	}

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		linkTmp := path + ".linktmp"
		regTmp := path + ".regtmp"
		for {
			select {
			case <-stop:
				return
			default:
			}
			// Swap in a symlink to the secret file.
			_ = os.Remove(linkTmp)
			if err := os.Symlink(secretPath, linkTmp); err == nil {
				_ = os.Rename(linkTmp, path)
			}
			// Swap back to a legit regular file.
			if err := os.WriteFile(regTmp, legit, 0o600); err == nil {
				_ = os.Rename(regTmp, path)
			}
		}
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, exists, err := ReadResult(dir)
		if err == nil && exists && bytes.Contains(data, []byte("secret")) {
			t.Fatal("ReadResult returned the symlink target's content: TOCTOU symlink follow observed")
		}
	}
}

func TestReadResult_NotExists(t *testing.T) {
	dir := newRunDir(t)
	_, exists, err := ReadResult(dir)
	if err != nil {
		t.Fatalf("ReadResult on empty dir: %v", err)
	}
	if exists {
		t.Error("exists = true for a run dir with no result.json")
	}
}

func TestCreateResult_RejectsFIFOTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are not available on windows")
	}
	dir := newRunDir(t)
	path := filepath.Join(dir, resultFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err == nil {
		t.Fatal("CreateResult over a FIFO target: want error, got nil")
	}
}

func TestWriteOperationState_RejectsFIFOTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are not available on windows")
	}
	dir := newRunDir(t)
	path := filepath.Join(dir, operationStateFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	if err := WriteOperationState(dir, []byte(`{"phase":"x"}`)); err == nil {
		t.Fatal("WriteOperationState over a FIFO target: want error, got nil")
	}
}

func TestWriteOperationState_RejectsOversizedData(t *testing.T) {
	dir := newRunDir(t)
	big := make([]byte, maxSidecarBytes+1)
	if err := WriteOperationState(dir, big); err == nil {
		t.Fatal("WriteOperationState with oversized data: want error, got nil")
	}
}

func TestWriteOperationState_RejectsDirectoryTarget(t *testing.T) {
	dir := newRunDir(t)
	path := filepath.Join(dir, operationStateFileName)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := WriteOperationState(dir, []byte(`{"phase":"x"}`)); err == nil {
		t.Fatal("WriteOperationState over a directory target: want error, got nil")
	}
}

func TestReadOperationState_NotExists(t *testing.T) {
	dir := newRunDir(t)
	_, exists, err := ReadOperationState(dir)
	if err != nil {
		t.Fatalf("ReadOperationState on empty dir: %v", err)
	}
	if exists {
		t.Error("exists = true for a run dir with no operation-state.json")
	}
}
