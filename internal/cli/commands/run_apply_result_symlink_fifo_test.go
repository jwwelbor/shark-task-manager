// This file covers code-review round-6's finding: readBoundedEnvelopeFile
// claimed parity with internal/gaterun's readRegularBounded but omitted its
// actual safety mechanism (a no-follow open plus an fstat regular-file
// check), using a plain os.Open instead. That means --apply-result=<symlink>
// silently followed the link, and --apply-result=<fifo> could hang the CLI
// process indefinitely (io.LimitReader's size bound never helps here — it
// still blocks reading from an open FIFO with no writer, below the limit).
// These tests prove both are now rejected via the shared
// gaterun.ReadBoundedRegularFile helper (fsio_path.go), the same no-follow +
// fstat protection gaterun's own sidecar reads use.
package commands

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

// TestReadBoundedEnvelopeFile_RejectsSymlinkTarget proves --apply-result
// pointing at a symlink is rejected rather than transparently followed to
// whatever file it targets.
func TestReadBoundedEnvelopeFile_RejectsSymlinkTarget(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "sensitive.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := readBoundedEnvelopeFile(link)
	if err == nil {
		t.Fatal("readBoundedEnvelopeFile over a symlinked --apply-result: want error, got nil")
	}
	var unsafeErr *gaterun.UnsafePathError
	if !errors.As(err, &unsafeErr) {
		t.Errorf("readBoundedEnvelopeFile over a symlinked --apply-result error = %v, want *gaterun.UnsafePathError", err)
	}
}

// TestReadBoundedEnvelopeFile_RejectsFIFOTarget proves --apply-result
// pointing at a FIFO with no writer connected returns an error promptly
// instead of hanging the CLI process. A 5s deadline bounds the test itself
// in case the fix regresses.
func TestReadBoundedEnvelopeFile_RejectsFIFOTarget(t *testing.T) {
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
		if _, err := readBoundedEnvelopeFile(path); err == nil {
			t.Error("readBoundedEnvelopeFile over a FIFO --apply-result: want error, got nil")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readBoundedEnvelopeFile over a FIFO --apply-result blocked instead of returning an error")
	}
}
