//go:build windows

package gaterun

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// openRunDirNoFollow is the Windows counterpart of dirhandle_unix.go's
// openat(O_NOFOLLOW|O_DIRECTORY) chain: every ancestor path component this
// package owns (.shark, runs, <runID>) is opened relative to the descriptor
// for the level above it via NtCreateFile with FILE_OPEN_REPARSE_POINT (see
// ntfs_windows.go), and verified — via GetFileInformationByHandle on that
// same handle, never a separate by-path Lstat — to be a real, non-reparse-
// point directory. A concurrent same-user process swapping any of those
// three components for a symlink or junction is refused here rather than
// followed, closing the TOCTOU window the package's previous Windows
// fallback (plain os.Open by path) left open (TD-187 / UAT-3-2). Every file
// operation in this package then performs its actual work relative to the
// returned handle, so the no-follow guarantee holds all the way to the real
// open/create/rename — not just at a preceding, separately-reusable-by-
// pathname check.
//
// projectRoot itself (the anchor) is opened by path and may traverse
// symlinks — it is the caller-supplied trust boundary RunDir has always
// accepted, not a new gap introduced here (same as the Unix build).
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
	anchorHandle := windows.Handle(anchor.Fd())

	sharkHandle, err := openDirRelNoFollow(anchorHandle, sharkName, false, false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = windows.CloseHandle(sharkHandle) }()

	runsHandle, err := openDirRelNoFollow(sharkHandle, runsName, false, false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = windows.CloseHandle(runsHandle) }()

	leafHandle, err := openDirRelNoFollow(runsHandle, runIDName, false, true)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(leafHandle), runDir), nil
}
