//go:build !windows

package commands

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestReadBoundedImpactFile_RejectsFIFOTarget proves --impact-file pointing
// at a FIFO with no writer connected returns an error promptly instead of
// hanging the CLI process. A 5s deadline bounds the test itself in case the
// fix regresses.
//
// syscall.Mkfifo has no Windows implementation, so this test is split out of
// impact_symlink_fifo_test.go (which also covers the cross-platform symlink
// case) into its own POSIX-only file rather than making that shared file
// fail go vet/go build on GOOS=windows.
func TestReadBoundedImpactFile_RejectsFIFOTarget(t *testing.T) {
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
