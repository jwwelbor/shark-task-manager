//go:build windows

package gaterun

import (
	"fmt"
	"os"
	"path/filepath"
)

// ensureRunDirTree is the Windows fallback for runid_unix.go's
// openat(O_NOFOLLOW|O_DIRECTORY) descriptor-chain implementation. The
// standard library exposes no portable O_NOFOLLOW-equivalent open flag on
// this platform, so each ancestor component is verified with a plain
// Lstat-then-path-reuse instead — leaving the same ancestor-directory TOCTOU
// window the Unix build closes unaddressed on Windows. This is a residual,
// documented gap (TD-181), consistent with this package's existing leaf-file
// Windows fallback (fsio_nofollow_windows.go) and its test suite, which
// skips the symlink-swap-race regression tests on windows for the same
// reason.
func ensureRunDirTree(projectRoot, runID string) (string, error) {
	sharkDir := filepath.Join(projectRoot, ".shark")
	if err := ensureRealDir(sharkDir, 0o755); err != nil {
		return "", err
	}
	runsDir := filepath.Join(sharkDir, "runs")
	if err := ensureRealDir(runsDir, 0o755); err != nil {
		return "", err
	}
	leaf := filepath.Join(runsDir, runID)
	if err := ensureRealDir(leaf, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(leaf, 0o700); err != nil {
		return "", fmt.Errorf("gaterun: chmod run dir %s: %w", leaf, err)
	}
	return leaf, nil
}

// ensureRealDir verifies path is a real (non-symlink) directory, creating it
// with mode when absent. It never follows a symlink at path, and rejects any
// existing non-directory target.
func ensureRealDir(path string, mode os.FileMode) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.Mkdir(path, mode); mkErr != nil {
				// Tolerate a concurrent creator; re-check below.
				if !os.IsExist(mkErr) {
					return fmt.Errorf("gaterun: create dir %s: %w", path, mkErr)
				}
			} else {
				return nil
			}
			fi, err = os.Lstat(path)
			if err != nil {
				return fmt.Errorf("gaterun: stat dir %s after create race: %w", path, err)
			}
		} else {
			return fmt.Errorf("gaterun: stat %s: %w", path, err)
		}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return &UnsafePathError{Path: path, Reason: "refusing to follow symlink"}
	}
	if !fi.IsDir() {
		return &UnsafePathError{Path: path, Reason: "exists and is not a directory"}
	}
	return nil
}
