// This file covers code-review round-6's finding: impact.go's --impact-file
// read used a plain os.ReadFile, buffering the entire file into memory
// before gateresult.ValidateChangeImpactSet ever applied its field-level
// bounds — the same defect class T-E34-F05-004's rework closed for
// --apply-result (run_apply_result_symlink_fifo_test.go). A plain
// os.ReadFile also silently follows a symlink target. This test proves that
// is now rejected via the shared gaterun.ReadBoundedRegularFile helper
// (fsio_path.go).
//
// The companion FIFO regression (--impact-file=<fifo> hanging the CLI
// process) lives in impact_fifo_unix_test.go: syscall.Mkfifo has no Windows
// implementation, so it is split into its own POSIX-only file rather than
// making this file (which also covers the cross-platform symlink case) fail
// go vet/go build on GOOS=windows.
package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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
