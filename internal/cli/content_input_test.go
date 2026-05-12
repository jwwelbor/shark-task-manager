package cli

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// socketPair wraps the two ends of a Unix-domain socketpair so that the test
// can hand `os.Stdin` something that is non-TTY, non-pipe, non-regular — the
// exact shape Claude Code's bash wrapper provides — and still clean up.
type socketPair struct {
	localFile  *os.File
	remoteFile *os.File
}

func (p *socketPair) close() {
	_ = p.localFile.Close()
	_ = p.remoteFile.Close()
}

func socketPairForTest(t *testing.T) (*socketPair, error) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	return &socketPair{
		localFile:  os.NewFile(uintptr(fds[0]), "stdin-socket-local"),
		remoteFile: os.NewFile(uintptr(fds[1]), "stdin-socket-remote"),
	}, nil
}

// timeoutAfterShort returns a channel that fires after a short duration —
// long enough that a fast in-process call comfortably finishes, short enough
// that a regression (blocking io.ReadAll on a never-closing socket) fails
// the test promptly instead of hanging the suite.
func timeoutAfterShort() <-chan time.Time {
	return time.After(500 * time.Millisecond)
}

// TestResolveContentInput_FlagOnly verifies that a --content flag value is
// returned when stdin is a TTY (the default in tests). Stdin piping is awkward
// to simulate inside the test process, so the flag path is the practical check
// for `ResolveContentInput`.
func TestResolveContentInput_FlagOnly(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("content", "", "")
	if err := cmd.Flags().Set("content", "hello body"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	got, err := ResolveContentInput(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello body" {
		t.Errorf("expected %q, got %q", "hello body", got)
	}
}

// TestResolveContentInput_NoFlag returns "" when the --content flag is empty
// and stdin is a TTY.
func TestResolveContentInput_NoFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("content", "", "")

	got, err := ResolveContentInput(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestResolveContentInput_FlagNotRegistered tolerates commands that haven't
// declared a --content flag — the function falls through to "".
func TestResolveContentInput_FlagNotRegistered(t *testing.T) {
	cmd := &cobra.Command{}
	got, err := ResolveContentInput(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestResolveContentInput_StdinSocket guards against the failure mode that
// motivated the pipe/regular-file gating: when stdin is a Unix-domain socket
// (the shape Claude Code's bash wrapper hands us), ResolveContentInput must
// not call io.ReadAll on it — that would block until the wrapper closes the
// socket, which it never does. The function should fall through to the
// --content flag instead.
func TestResolveContentInput_StdinSocket(t *testing.T) {
	pair, err := socketPairForTest(t)
	if err != nil {
		t.Skipf("socketpair unavailable on this platform: %v", err)
	}
	defer pair.close()

	origStdin := os.Stdin
	os.Stdin = pair.localFile
	defer func() { os.Stdin = origStdin }()

	cmd := &cobra.Command{}
	cmd.Flags().String("content", "", "")
	if err := cmd.Flags().Set("content", "flag-wins"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	done := make(chan struct {
		val string
		err error
	}, 1)
	go func() {
		v, e := ResolveContentInput(cmd)
		done <- struct {
			val string
			err error
		}{v, e}
	}()

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("unexpected error: %v", result.err)
		}
		if result.val != "flag-wins" {
			t.Errorf("expected %q (flag fallback), got %q — implies socket stdin was read", "flag-wins", result.val)
		}
	case <-timeoutAfterShort():
		t.Fatal("ResolveContentInput blocked on a Unix socket stdin — the pipe/regular-file gate is not in effect")
	}
}
