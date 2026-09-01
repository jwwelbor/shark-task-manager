//go:build windows

package gaterun

import (
	"fmt"
	"os"
)

// openRegularNoFollow is the Windows fallback: the standard library exposes
// no portable O_NOFOLLOW-equivalent open flag on this platform, so this
// falls back to a separate Lstat-then-Open. This keeps the TOCTOU window
// this package's Linux/macOS build closes (see fsio_nofollow_unix.go)
// unaddressed on Windows; symlink creation there requires elevated
// privileges by default, and the existing test suite skips the
// symlink-swap-race regression test on windows for the same reason.
func openRegularNoFollow(path string) (*os.File, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("gaterun: stat %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, &UnsafePathError{Path: path, Reason: "refusing to follow symlink"}
	}
	if !fi.Mode().IsRegular() {
		return nil, &UnsafePathError{Path: path, Reason: "target is not a regular file"}
	}

	f, err := os.Open(path) // #nosec G304 -- path is joined from a validated run dir, and is Lstat-verified above.
	if err != nil {
		return nil, fmt.Errorf("gaterun: open %s: %w", path, err)
	}
	return f, nil
}
