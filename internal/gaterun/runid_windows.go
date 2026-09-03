//go:build windows

package gaterun

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// ensureRunDirTree is the Windows counterpart of runid_unix.go's
// openat(O_NOFOLLOW|O_DIRECTORY) descriptor-chain implementation: each
// ancestor component is opened-or-created (FILE_OPEN_IF) and verified real
// via NtCreateFile relative to the handle for the level above it (see
// ensureDirRelNoFollow / ntfs_windows.go), never via a separate
// Lstat-then-path-reuse step. This closes the ancestor-directory TOCTOU
// window the package's previous Windows fallback (Lstat, then Mkdir or
// os.Chmod against the same path string) left open (TD-187 / UAT-3-2).
//
// Unlike the pre-fix fallback, this does not issue a trailing path-based
// os.Chmod(leaf, ...) to reinforce owner-only mode: Windows has no
// permission-bit analogue of POSIX 0700 that os.Chmod actually enforces
// (it only toggles the read-only DOS attribute), so that call was already
// a no-op for the security property it claimed to provide while
// reintroducing exactly the same by-path TOCTOU this function closes for
// every other step. The leaf directory's real-directory / no-follow
// guarantee comes entirely from the verified handle chain above.
func ensureRunDirTree(projectRoot, runID string) (string, error) {
	anchor, err := os.Open(projectRoot) // #nosec G304 -- projectRoot is the caller-trusted anchor RunDir has always accepted.
	if err != nil {
		return "", fmt.Errorf("gaterun: open project root %s: %w", projectRoot, err)
	}
	defer func() { _ = anchor.Close() }()
	anchorHandle := windows.Handle(anchor.Fd())

	sharkDir := filepath.Join(projectRoot, ".shark")
	sharkHandle, err := ensureDirRelNoFollow(anchorHandle, ".shark", sharkDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(sharkHandle) }()

	runsDir := filepath.Join(sharkDir, "runs")
	runsHandle, err := ensureDirRelNoFollow(sharkHandle, "runs", runsDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(runsHandle) }()

	leaf := filepath.Join(runsDir, runID)
	leafHandle, err := ensureDirRelNoFollow(runsHandle, runID, leaf)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(leafHandle) }()

	return leaf, nil
}

// ensureDirRelNoFollow opens name as a directory relative to parent,
// creating it when absent, and verifies — via the resulting handle, never a
// separate Lstat — that it is a real (non-reparse-point) directory.
// displayPath is used only for error messages.
func ensureDirRelNoFollow(parent windows.Handle, name, displayPath string) (windows.Handle, error) {
	h, err := openDirRelNoFollow(parent, name, true, false)
	if err != nil {
		return 0, fmt.Errorf("gaterun: create dir %s: %w", displayPath, err)
	}
	return h, nil
}
