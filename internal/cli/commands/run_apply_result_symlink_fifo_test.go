// This file covers code-review round-6's finding: readBoundedEnvelopeFile
// claimed parity with internal/gaterun's readRegularBounded but omitted its
// actual safety mechanism (a no-follow open plus an fstat regular-file
// check), using a plain os.Open instead. That means --apply-result=<symlink>
// silently followed the link. This test proves it is now rejected via the
// shared gaterun.ReadBoundedRegularFile helper (fsio_path.go), the same
// no-follow + fstat protection gaterun's own sidecar reads use.
//
// The companion FIFO regression (--apply-result=<fifo> hanging the CLI
// process) lives in run_apply_result_fifo_unix_test.go: syscall.Mkfifo has
// no Windows implementation, so it is split into its own POSIX-only file
// rather than making this file (which also covers the cross-platform
// symlink case) fail go vet/go build on GOOS=windows.
package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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
