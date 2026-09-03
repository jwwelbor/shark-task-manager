//go:build !windows

package commands

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestReadBoundedEnvelopeFile_RejectsFIFOTarget proves --apply-result
// pointing at a FIFO with no writer connected returns an error promptly
// instead of hanging the CLI process. A 5s deadline bounds the test itself
// in case the fix regresses.
//
// syscall.Mkfifo has no Windows implementation, so this test is split out of
// run_apply_result_symlink_fifo_test.go (which also covers the
// cross-platform symlink case) into its own POSIX-only file rather than
// making that shared file fail go vet/go build on GOOS=windows.
func TestReadBoundedEnvelopeFile_RejectsFIFOTarget(t *testing.T) {
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
