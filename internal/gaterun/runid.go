package gaterun

import (
	"fmt"
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
//
// The platform-specific ancestor-verification strategy lives in
// ensureRunDirTree (runid_unix.go's openat(O_NOFOLLOW|O_DIRECTORY) chain, or
// runid_windows.go's documented Lstat-based fallback).
func RunDir(projectRoot, runID string) (string, error) {
	if projectRoot == "" {
		return "", fmt.Errorf("gaterun: project root must not be empty")
	}
	if err := ValidateRunID(runID); err != nil {
		return "", err
	}

	leaf, err := ensureRunDirTree(projectRoot, runID)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(leaf)
	if err != nil {
		return "", fmt.Errorf("gaterun: resolve absolute run dir: %w", err)
	}
	return abs, nil
}
