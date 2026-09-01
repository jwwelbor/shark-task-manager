//go:build !windows

package gaterun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// openRunDirNoFollow re-derives, from a trusted projectRoot anchor, a
// no-follow-verified open directory handle for runDir (a path previously
// returned by RunDir: <projectRoot>/.shark/runs/<runID>). Every ancestor
// path component this package owns (.shark, runs, <runID>) is opened with
// O_NOFOLLOW|O_DIRECTORY directly off the descriptor for the level above it,
// so a concurrent same-UID process swapping any of those three components
// for a symlink — whether before this call or between two calls, e.g. after
// RunDir's own ensureRealDir check ran in an earlier process invocation —
// is refused here rather than silently followed. Every file operation in
// this package performs its actual work relative to the returned handle
// (fd-relative openat/linkat/renameat/unlinkat/fstatat), so the no-follow
// guarantee holds all the way to the real open/create/rename, not just at a
// preceding, separately-reusable-by-pathname check (the defect class this
// closes).
//
// projectRoot itself (the anchor) is opened by path and may traverse
// symlinks — it is the caller-supplied trust boundary RunDir has always
// accepted (RunDir's own ensureRealDir never verified projectRoot, only the
// three components beneath it), not a new gap introduced here.
func openRunDirNoFollow(runDir string) (*os.File, error) {
	runsDir := filepath.Dir(runDir)
	sharkDir := filepath.Dir(runsDir)
	projectRoot := filepath.Dir(sharkDir)

	runIDName := filepath.Base(runDir)
	runsName := filepath.Base(runsDir)
	sharkName := filepath.Base(sharkDir)
	if sharkName != ".shark" || runsName != "runs" {
		return nil, fmt.Errorf("gaterun: %q is not a recognized run directory", runDir)
	}
	if err := ValidateRunID(runIDName); err != nil {
		return nil, fmt.Errorf("gaterun: %q is not a recognized run directory: %w", runDir, err)
	}

	anchor, err := os.Open(projectRoot) // #nosec G304 -- projectRoot is the caller-trusted anchor RunDir has always accepted.
	if err != nil {
		return nil, fmt.Errorf("gaterun: open project root %s: %w", projectRoot, err)
	}
	defer func() { _ = anchor.Close() }()

	sharkFd, err := openatDirNoFollow(int(anchor.Fd()), sharkDir, sharkName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(sharkFd) }()

	runsFd, err := openatDirNoFollow(sharkFd, runsDir, runsName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(runsFd) }()

	leafFd, err := openatDirNoFollow(runsFd, runDir, runIDName)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(leafFd), runDir), nil
}

// openatDirNoFollow opens name as a directory relative to parentFd,
// rejecting rather than following a symlink at that path component.
// displayPath is used only for error messages.
func openatDirNoFollow(parentFd int, displayPath, name string) (int, error) {
	fd, err := unix.Openat(parentFd, name, unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		// O_NOFOLLOW|O_DIRECTORY against a symlink reports ELOOP on some
		// unix kernels but ENOTDIR on Linux specifically (the unfollowed
		// symlink itself is never a directory, and Linux's O_DIRECTORY
		// check runs before symlink resolution would otherwise report
		// ELOOP) — both mean the same thing here: the path component is not
		// a real directory we can safely descend into, whether because it
		// is a symlink or because it is some other non-directory entry.
		if errors.Is(err, unix.ELOOP) || errors.Is(err, unix.ENOTDIR) {
			return 0, &UnsafePathError{Path: displayPath, Reason: "refusing to follow symlink or non-directory ancestor"}
		}
		return 0, fmt.Errorf("gaterun: open %s: %w", displayPath, err)
	}
	return fd, nil
}
