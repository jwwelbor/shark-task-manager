//go:build !windows

package gaterun

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// openRegularNoFollow opens path for reading with O_NOFOLLOW, then verifies
// via fstat on the already-open descriptor that the target is a regular
// file. Because the safety check runs on the open fd rather than a separate
// stat-then-open of the path, there is no TOCTOU window in which a
// concurrent same-UID process can swap the target between the check and the
// read (unlike a plain os.Lstat followed by os.Open, where os.Open silently
// follows any symlink planted after the Lstat observed a regular file).
//
// O_NONBLOCK is included so that opening a FIFO for read returns
// immediately (POSIX: O_RDONLY|O_NONBLOCK on a FIFO never blocks waiting for
// a writer) rather than hanging the caller; it has no effect on reads from a
// regular file, which is the only target this function accepts.
func openRegularNoFollow(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 -- path is joined from a validated run dir; O_NOFOLLOW makes this call itself the safety check.
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, &UnsafePathError{Path: path, Reason: "refusing to follow symlink"}
		}
		return nil, fmt.Errorf("gaterun: open %s: %w", path, err)
	}

	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("gaterun: fstat %s: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		_ = f.Close()
		return nil, &UnsafePathError{Path: path, Reason: "target is not a regular file"}
	}
	return f, nil
}
