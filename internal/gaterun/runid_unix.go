//go:build !windows

package gaterun

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// ensureRunDirTree creates (if missing) and verifies, entirely via
// openat(O_NOFOLLOW|O_DIRECTORY) relative descriptors, the .shark/runs/
// <runID> directory chain under projectRoot. Every ancestor component this
// package owns is opened no-follow immediately after it is created (or
// found) — there is no separate "verify this is a real directory" step
// whose result is later reused by reopening the same path by name, so a
// concurrent same-UID process cannot swap any of these components for a
// symlink in a window this function leaves open. This is the same
// descriptor-chain guarantee openRunDirNoFollow (dirhandle_unix.go) gives
// every later file operation against an already-existing run directory.
func ensureRunDirTree(projectRoot, runID string) (string, error) {
	anchor, err := os.Open(projectRoot) // #nosec G304 -- projectRoot is the caller-trusted anchor RunDir has always accepted.
	if err != nil {
		return "", fmt.Errorf("gaterun: open project root %s: %w", projectRoot, err)
	}
	defer func() { _ = anchor.Close() }()

	sharkDir := filepath.Join(projectRoot, ".shark")
	sharkFd, err := ensureDirNoFollowAt(int(anchor.Fd()), ".shark", 0o755, sharkDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = unix.Close(sharkFd) }()

	runsDir := filepath.Join(sharkDir, "runs")
	runsFd, err := ensureDirNoFollowAt(sharkFd, "runs", 0o755, runsDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = unix.Close(runsFd) }()

	leaf := filepath.Join(runsDir, runID)
	leafFd, err := ensureDirNoFollowAt(runsFd, runID, 0o700, leaf)
	if err != nil {
		return "", err
	}
	defer func() { _ = unix.Close(leafFd) }()

	// Owner-only per REQ-NF-001, enforced unconditionally (not only at
	// creation) so a leaf directory created earlier by another subsystem
	// (e.g. the run-liveness recorder) is tightened, not trusted as-is.
	// Fchmod operates on the already-open, no-follow-verified descriptor —
	// never a path — so, unlike a path-based os.Chmod(leaf, ...), it cannot
	// be tricked into chmod'ing whatever leaf happens to resolve to at the
	// moment of the call.
	if err := unix.Fchmod(leafFd, 0o700); err != nil {
		return "", fmt.Errorf("gaterun: chmod run dir %s: %w", leaf, err)
	}
	return leaf, nil
}

// ensureDirNoFollowAt opens name as a directory relative to parentFd,
// creating it with mode when absent, and rejecting rather than following a
// symlink at that path component. displayPath is used only for error
// messages. The post-create re-open (not the pre-create existence check) is
// what the caller trusts as its no-follow verification.
func ensureDirNoFollowAt(parentFd int, name string, mode os.FileMode, displayPath string) (int, error) {
	fd, err := openatDirNoFollow(parentFd, displayPath, name)
	if err == nil {
		return fd, nil
	}
	var unsafeErr *UnsafePathError
	if errors.As(err, &unsafeErr) {
		return 0, err
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	if mkErr := unix.Mkdirat(parentFd, name, uint32(mode.Perm())); mkErr != nil && !errors.Is(mkErr, fs.ErrExist) {
		// Tolerate a concurrent creator; the re-open below re-verifies.
		return 0, fmt.Errorf("gaterun: create dir %s: %w", displayPath, mkErr)
	}
	return openatDirNoFollow(parentFd, displayPath, name)
}
