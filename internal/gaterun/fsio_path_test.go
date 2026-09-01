package gaterun

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// TestReadBoundedRegularFile_RejectsSymlinkTarget guards
// ReadBoundedRegularFile — the bare-path counterpart of readRegularBounded
// used by CLI-flag-supplied files (--apply-result, --impact-file) — against
// transparently following a symlink to a file outside the caller's intended
// scope.
func TestReadBoundedRegularFile_RejectsSymlinkTarget(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if _, err := ReadBoundedRegularFile(link, 1024); err == nil {
		t.Fatal("ReadBoundedRegularFile over a symlinked target: want error, got nil")
	} else {
		var unsafeErr *UnsafePathError
		if !errors.As(err, &unsafeErr) {
			t.Errorf("ReadBoundedRegularFile over a symlinked target error = %v, want *UnsafePathError", err)
		}
	}
}

// TestReadBoundedRegularFile_RejectsFIFOTarget guards against the no-follow
// open blocking indefinitely on a FIFO opened for read with no writer
// connected (a plain os.Open on a FIFO blocks until a writer appears). The
// fix opens with O_NONBLOCK so this returns promptly with an error instead
// of hanging the caller (and, in production, the CLI process). A 5s
// deadline bounds the test itself in case the fix regresses.
func TestReadBoundedRegularFile_RejectsFIFOTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are not available on windows")
	}
	path := filepath.Join(t.TempDir(), "envelope.json")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := ReadBoundedRegularFile(path, 1024); err == nil {
			t.Error("ReadBoundedRegularFile over a FIFO target: want error, got nil")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ReadBoundedRegularFile over a FIFO target blocked instead of returning an error")
	}
}

// TestReadBoundedRegularFile_OrdinaryFileRoundTrips is the sanity check that
// a well-formed regular file is read back byte-for-byte within the bound.
func TestReadBoundedRegularFile_OrdinaryFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "envelope.json")
	want := []byte(`{"a":1}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	got, err := ReadBoundedRegularFile(path, 1024)
	if err != nil {
		t.Fatalf("ReadBoundedRegularFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestReadBoundedRegularFile_RejectsOversized proves the maxBytes bound is
// enforced by ReadBoundedRegularFile itself.
func TestReadBoundedRegularFile_RejectsOversized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.WriteFile(path, make([]byte, 11), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	if _, err := ReadBoundedRegularFile(path, 10); err == nil {
		t.Fatal("ReadBoundedRegularFile with an oversized file: want error, got nil")
	}
}
