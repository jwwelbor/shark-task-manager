// This file covers code-review round-6's finding: impact.go's --impact-file
// read used a plain os.ReadFile, buffering the entire file into memory
// before gateresult.ValidateChangeImpactSet ever applied its field-level
// bounds — the same defect class T-E34-F05-004's rework closed for
// --apply-result (run_apply_result_symlink_fifo_test.go). A plain
// os.ReadFile also silently follows a symlink target and can hang
// indefinitely on a FIFO with no writer connected. These tests prove both
// are now rejected via the shared gaterun.ReadBoundedRegularFile helper
// (fsio_path.go).
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

// TestReadBoundedImpactFile_RejectsSymlinkTarget proves --impact-file
// pointing at a symlink is rejected rather than transparently followed to
// whatever file it targets.
func TestReadBoundedImpactFile_RejectsSymlinkTarget(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "sensitive.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(t.TempDir(), "impact.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := readBoundedImpactFile(link)
	if err == nil {
		t.Fatal("readBoundedImpactFile over a symlinked --impact-file: want error, got nil")
	}
	var unsafeErr *gaterun.UnsafePathError
	if !errors.As(err, &unsafeErr) {
		t.Errorf("readBoundedImpactFile over a symlinked --impact-file error = %v, want *gaterun.UnsafePathError", err)
	}
}

// TestReadBoundedImpactFile_RejectsFIFOTarget proves --impact-file pointing
// at a FIFO with no writer connected returns an error promptly instead of
// hanging the CLI process. A 5s deadline bounds the test itself in case the
// fix regresses.
func TestReadBoundedImpactFile_RejectsFIFOTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFOs are not available on windows")
	}
	path := filepath.Join(t.TempDir(), "impact.json")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := readBoundedImpactFile(path); err == nil {
			t.Error("readBoundedImpactFile over a FIFO --impact-file: want error, got nil")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readBoundedImpactFile over a FIFO --impact-file blocked instead of returning an error")
	}
}
