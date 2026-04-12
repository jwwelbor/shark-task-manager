package commands

import (
	"fmt"
	"net"
	"runtime"
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

// TestOpenBrowser_CommandConstruction verifies the OS-dispatch logic of
// openBrowserForOS.  Supported OS strings (linux, darwin, windows) must not
// return the "unsupported OS" error; they may return an exec error when the
// native binary is unavailable (e.g. testing Windows paths on Linux).
// Unsupported OS strings (plan9, freebsd, …) must always return an error
// whose message contains "unsupported OS".
func TestOpenBrowser_CommandConstruction(t *testing.T) {
	supportedOSes := []string{"linux", "darwin", "windows"}
	unsupportedOSes := []string{"plan9", "freebsd"}

	for _, goos := range supportedOSes {
		goos := goos
		t.Run(goos, func(t *testing.T) {
			err := openBrowserForOS(goos, "http://127.0.0.1:7777")
			// Supported OSes must not return the "unsupported OS" sentinel.
			// An exec error (binary not found, no display) is acceptable in CI.
			if err != nil && err.Error() == fmt.Sprintf("unsupported OS: %s", goos) {
				t.Errorf("openBrowserForOS(%q) returned unsupported-OS error on a known platform: %v", goos, err)
			}
		})
	}

	for _, goos := range unsupportedOSes {
		goos := goos
		t.Run(goos, func(t *testing.T) {
			err := openBrowserForOS(goos, "http://127.0.0.1:7777")
			if err == nil {
				t.Errorf("openBrowserForOS(%q) expected error for unsupported OS, got nil", goos)
				return
			}
			want := fmt.Sprintf("unsupported OS: %s", goos)
			if err.Error() != want {
				t.Errorf("openBrowserForOS(%q) error = %q, want %q", goos, err.Error(), want)
			}
		})
	}
}

// TestOpenBrowser_CurrentOS confirms that openBrowser does not immediately
// return an error on the current platform (the command may fail to start a
// browser in a headless environment, but it should at least resolve the
// command name without an "unsupported OS" error).
func TestOpenBrowser_CurrentOS(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		// openBrowser may fail to launch a browser in CI (xdg-open not found,
		// no display, etc.) but it must NOT return the "unsupported OS" error.
		err := openBrowser("http://127.0.0.1:7777")
		if err != nil {
			// A start failure (e.g. exec: "xdg-open": not found) is acceptable
			// in headless environments.  Only fail if we get an "unsupported OS"
			// error.
			if err.Error() == fmt.Sprintf("unsupported OS: %s", runtime.GOOS) {
				t.Errorf("unexpected unsupported-OS error on %s: %v", runtime.GOOS, err)
			}
		}
	default:
		t.Skipf("skipping on %s — not a supported platform", runtime.GOOS)
	}
}
