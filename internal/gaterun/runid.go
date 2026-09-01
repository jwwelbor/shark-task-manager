package gaterun

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// runIDPattern anchors the accepted run_id shape: it must start with an
// alphanumeric character (rejecting "." and ".." outright, since those never
// match) and contain only alphanumerics, "-", "_", and "." thereafter. No
// path separator of any platform is a member of this character class, so a
// valid run_id can never escape the .shark/runs/<run-id> directory it is
// joined into.
var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ValidateRunID rejects empty, oversized, path-escaping, or otherwise
// disallowed run_id values. It is the sole gate a run_id must pass before it
// is ever joined into a filesystem path by this package.
func ValidateRunID(runID string) error {
	if runID == "" {
		return fmt.Errorf("gaterun: run_id must not be empty")
	}
	if !runIDPattern.MatchString(runID) {
		return fmt.Errorf("gaterun: run_id %q contains disallowed characters or exceeds 128 bytes", runID)
	}
	return nil
}

// RunDir resolves, and ensures exists as a real (non-symlink) directory, the
// owner-only (0700) .shark/runs/<run-id> directory for runID under
// projectRoot. Every ancestor path component (.shark, .shark/runs) is
// verified real-directory / no-follow before use, but is created with a
// shared-convention 0755 mode when missing rather than tightened to 0700 —
// those directories are shared across every run, not owned by this one.
func RunDir(projectRoot, runID string) (string, error) {
	if projectRoot == "" {
		return "", fmt.Errorf("gaterun: project root must not be empty")
	}
	if err := ValidateRunID(runID); err != nil {
		return "", err
	}

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
	// Owner-only per REQ-NF-001, enforced unconditionally (not only at
	// creation) so a leaf directory created earlier by another subsystem
	// (e.g. the run-liveness recorder) is tightened, not trusted as-is.
	if err := os.Chmod(leaf, 0o700); err != nil {
		return "", fmt.Errorf("gaterun: chmod run dir %s: %w", leaf, err)
	}

	abs, err := filepath.Abs(leaf)
	if err != nil {
		return "", fmt.Errorf("gaterun: resolve absolute run dir: %w", err)
	}
	return abs, nil
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
