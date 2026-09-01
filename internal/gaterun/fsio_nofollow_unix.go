//go:build !windows

package gaterun

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// This file provides the fd-relative primitives every file operation in
// fsio.go and lock.go performs against an already-opened, no-follow-verified
// run-directory handle (see openRunDirNoFollow in dirhandle_unix.go). Using
// openat/linkat/renameat/unlinkat/fstatat relative to that one descriptor,
// instead of a fresh os.Lstat-then-path-reopen for every operation, is what
// makes the no-follow guarantee hold all the way to the real filesystem
// call: there is no separate check-then-reuse-by-pathname step for any of
// these operations to race against.

// openRegularNoFollowAt opens name, relative to dh, for reading with
// O_NOFOLLOW, then verifies via fstat on the already-open descriptor that
// the target is a regular file. Because the safety check runs on the open
// fd rather than a separate stat-then-open, there is no TOCTOU window in
// which a concurrent same-UID process can swap the target between the check
// and the read.
//
// O_NONBLOCK is included so that opening a FIFO for read returns
// immediately (POSIX: O_RDONLY|O_NONBLOCK on a FIFO never blocks waiting for
// a writer) rather than hanging the caller; it has no effect on reads from a
// regular file, which is the only target this function accepts.
func openRegularNoFollowAt(dh *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(dh.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, &UnsafePathError{Path: name, Reason: "refusing to follow symlink"}
		}
		if errors.Is(err, unix.ENOENT) {
			return nil, fmt.Errorf("gaterun: open %s: %w", name, os.ErrNotExist)
		}
		return nil, fmt.Errorf("gaterun: open %s: %w", name, err)
	}
	f := os.NewFile(uintptr(fd), name)

	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("gaterun: fstat %s: %w", name, err)
	}
	if !fi.Mode().IsRegular() {
		_ = f.Close()
		return nil, &UnsafePathError{Path: name, Reason: "target is not a regular file"}
	}
	return f, nil
}

// createExclAt creates name relative to dh with O_CREAT|O_EXCL|O_WRONLY,
// failing if it already exists (regardless of what it is).
func createExclAt(dh *os.File, name string, mode os.FileMode) (*os.File, error) {
	fd, err := unix.Openat(int(dh.Fd()), name, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC, uint32(mode.Perm()))
	if err != nil {
		if errors.Is(err, unix.EEXIST) {
			return nil, fmt.Errorf("gaterun: create %s: %w", name, os.ErrExist)
		}
		return nil, fmt.Errorf("gaterun: create %s: %w", name, err)
	}
	return os.NewFile(uintptr(fd), name), nil
}

// linkAt hard-links oldname to newname, both relative to dh.
func linkAt(dh *os.File, oldname, newname string) error {
	dfd := int(dh.Fd())
	if err := unix.Linkat(dfd, oldname, dfd, newname, 0); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("gaterun: link %s: %w", newname, os.ErrExist)
		}
		return fmt.Errorf("gaterun: link %s to %s: %w", oldname, newname, err)
	}
	return nil
}

// renameAt atomically renames oldname to newname, both relative to dh.
func renameAt(dh *os.File, oldname, newname string) error {
	dfd := int(dh.Fd())
	if err := unix.Renameat(dfd, oldname, dfd, newname); err != nil {
		return fmt.Errorf("gaterun: rename %s to %s: %w", oldname, newname, err)
	}
	return nil
}

// removeAt unlinks name relative to dh. Unlike a path-based remove, unlink
// never dereferences the final path component even when it is a symlink, so
// this needs no separate no-follow guard.
func removeAt(dh *os.File, name string) error {
	if err := unix.Unlinkat(int(dh.Fd()), name, 0); err != nil {
		return fmt.Errorf("gaterun: remove %s: %w", name, err)
	}
	return nil
}

// existingTargetKindAt reports, for name relative to dh, whether it exists
// and — if so — whether it is a symlink or a regular file, via a single
// no-follow fstatat call. It never opens or reads the target.
func existingTargetKindAt(dh *os.File, name string) (exists, isSymlink, isRegular bool, err error) {
	var st unix.Stat_t
	statErr := unix.Fstatat(int(dh.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW)
	if statErr != nil {
		if errors.Is(statErr, unix.ENOENT) {
			return false, false, false, nil
		}
		return false, false, false, fmt.Errorf("gaterun: stat %s: %w", name, statErr)
	}
	fileType := uint32(st.Mode) & unix.S_IFMT //nolint:gosec // st.Mode is a small unsigned stat-mode field on every unix target.
	return true, fileType == unix.S_IFLNK, fileType == unix.S_IFREG, nil
}
