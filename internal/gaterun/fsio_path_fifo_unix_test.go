//go:build !windows

package gaterun

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestReadBoundedRegularFile_RejectsFIFOTarget guards against the no-follow
// open blocking indefinitely on a FIFO opened for read with no writer
// connected (a plain os.Open on a FIFO blocks until a writer appears). The
// fix opens with O_NONBLOCK so this returns promptly with an error instead
// of hanging the caller (and, in production, the CLI process). A 5s
// deadline bounds the test itself in case the fix regresses.
//
// syscall.Mkfifo has no Windows implementation, so this test — split out of
// fsio_path_test.go, which otherwise runs on every platform — is POSIX-only
// and lives behind its own build constraint rather than making the shared
// test file fail go vet/go build on GOOS=windows.
func TestReadBoundedRegularFile_RejectsFIFOTarget(t *testing.T) {
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
