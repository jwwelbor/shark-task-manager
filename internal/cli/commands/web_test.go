package commands

import (
	"fmt"
	"net"
	"testing"
)

// TestFindFreePort verifies that findFreePort returns a port within the
// requested range and that the port is actually connectable.
func TestFindFreePort(t *testing.T) {
	const start = 17777
	const end = start + 13

	port, err := findFreePort(start, end)
	if err != nil {
		t.Fatalf("findFreePort(%d, %d) unexpected error: %v", start, end, err)
	}

	if port < start || port > end {
		t.Errorf("port %d outside range [%d, %d]", port, start, end)
	}

	// Verify the returned port can actually be bound.
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Errorf("returned port %d is not bindable: %v", port, err)
	} else {
		_ = ln.Close()
	}
}

// TestFindFreePort_InvalidRange verifies that findFreePort returns an error
// immediately when start > end (the loop never executes and falls through to
// the error return).
func TestFindFreePort_InvalidRange(t *testing.T) {
	_, err := findFreePort(8080, 8079)
	if err == nil {
		t.Error("expected error when start > end, got nil")
	}
}

// TestFindFreePort_AllBusy verifies that an error is returned when every
// port in the range is already in use.
func TestFindFreePort_AllBusy(t *testing.T) {
	// Bind a listener on an OS-assigned port, then restrict the range to just
	// that port so findFreePort has nowhere to go.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind test listener: %v", err)
	}
	defer ln.Close()

	busyPort := ln.Addr().(*net.TCPAddr).Port

	_, err = findFreePort(busyPort, busyPort)
	if err == nil {
		t.Errorf("expected error when all ports are busy, got nil")
	}
}

// TestBrowserCommand_OSDispatch verifies the OS-dispatch logic of
// browserCommand. It checks the constructed *exec.Cmd's argv without
// invoking Start(), so running tests does not actually launch any browsers.
func TestBrowserCommand_OSDispatch(t *testing.T) {
	const url = "http://127.0.0.1:7777"

	cases := []struct {
		goos     string
		wantArgs []string
	}{
		{"linux", []string{"xdg-open", url}},
		{"darwin", []string{"open", url}},
		{"windows", []string{"cmd", "/c", "start", url}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.goos, func(t *testing.T) {
			cmd, err := browserCommand(tc.goos, url)
			if err != nil {
				t.Fatalf("browserCommand(%q) unexpected error: %v", tc.goos, err)
			}
			if cmd == nil {
				t.Fatalf("browserCommand(%q) returned nil cmd", tc.goos)
			}
			if len(cmd.Args) != len(tc.wantArgs) {
				t.Fatalf("browserCommand(%q) args = %v, want %v", tc.goos, cmd.Args, tc.wantArgs)
			}
			for i, want := range tc.wantArgs {
				if cmd.Args[i] != want {
					t.Errorf("browserCommand(%q) args[%d] = %q, want %q", tc.goos, i, cmd.Args[i], want)
				}
			}
		})
	}
}

// TestBrowserCommand_UnsupportedOS verifies that browserCommand returns an
// "unsupported OS" error for non-supported GOOS values.
func TestBrowserCommand_UnsupportedOS(t *testing.T) {
	for _, goos := range []string{"plan9", "freebsd"} {
		goos := goos
		t.Run(goos, func(t *testing.T) {
			cmd, err := browserCommand(goos, "http://127.0.0.1:7777")
			if err == nil {
				t.Errorf("browserCommand(%q) expected error for unsupported OS, got nil", goos)
				return
			}
			if cmd != nil {
				t.Errorf("browserCommand(%q) returned non-nil cmd alongside error", goos)
			}
			want := fmt.Sprintf("unsupported OS: %s", goos)
			if err.Error() != want {
				t.Errorf("browserCommand(%q) error = %q, want %q", goos, err.Error(), want)
			}
		})
	}
}
