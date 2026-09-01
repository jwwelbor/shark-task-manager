//go:build windows

package gaterun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// This file is the Windows fallback for the fd-relative primitives
// fsio_nofollow_unix.go implements via openat/linkat/renameat/unlinkat/
// fstatat. The standard library exposes no portable no-follow or
// directory-relative ("*at") primitives on this platform, so each operation
// here falls back to a plain path join off dh's directory name plus the
// ordinary os.* path-based call — the same approach, and the same residual
// TOCTOU gap, this package has always documented for Windows (see
// dirhandle_windows.go and the existing leaf-file fallback below).

func joinAt(dh *os.File, name string) string {
	return filepath.Join(dh.Name(), name)
}

// openRegularNoFollowAt is the Windows fallback: it falls back to a separate
// Lstat-then-Open, leaving the TOCTOU window this package's Unix build
// closes (see fsio_nofollow_unix.go) unaddressed on Windows; symlink
// creation there requires elevated privileges by default, and the existing
// test suite skips the symlink-swap-race regression test on windows for the
// same reason.
func openRegularNoFollowAt(dh *os.File, name string) (*os.File, error) {
	path := joinAt(dh, name)
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

func createExclAt(dh *os.File, name string, mode os.FileMode) (*os.File, error) {
	path := joinAt(dh, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode) // #nosec G304 -- path is joined from a validated run dir.
	if err != nil {
		return nil, fmt.Errorf("gaterun: create %s: %w", path, err)
	}
	return f, nil
}

func linkAt(dh *os.File, oldname, newname string) error {
	if err := os.Link(joinAt(dh, oldname), joinAt(dh, newname)); err != nil {
		return fmt.Errorf("gaterun: link %s to %s: %w", oldname, newname, err)
	}
	return nil
}

func renameAt(dh *os.File, oldname, newname string) error {
	if err := os.Rename(joinAt(dh, oldname), joinAt(dh, newname)); err != nil {
		return fmt.Errorf("gaterun: rename %s to %s: %w", oldname, newname, err)
	}
	return nil
}

func removeAt(dh *os.File, name string) error {
	if err := os.Remove(joinAt(dh, name)); err != nil {
		return fmt.Errorf("gaterun: remove %s: %w", name, err)
	}
	return nil
}

func existingTargetKindAt(dh *os.File, name string) (exists, isSymlink, isRegular bool, err error) {
	fi, statErr := os.Lstat(joinAt(dh, name))
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return false, false, false, nil
		}
		return false, false, false, fmt.Errorf("gaterun: stat %s: %w", name, statErr)
	}
	return true, fi.Mode()&os.ModeSymlink != 0, fi.Mode().IsRegular(), nil
}
