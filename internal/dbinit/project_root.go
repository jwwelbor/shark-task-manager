// Package dbinit provides cloud-aware database initialization that can be used
// by both cmd/shark (CLI) and cmd/server (HTTP viewer) without importing internal/cli.
package dbinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/projectroot"
)

// findProjectRoot walks up the directory tree from startDir to find the project root.
// It looks for markers with different priorities:
//  1. .sharkconfig.json (STRONGEST - always preferred)
//  2. shark-tasks.db (STRONG - used if no .sharkconfig.json found)
//  3. .git/ directory (WEAK - used if no stronger markers found)
//
// Returns the project root directory, or startDir if no markers found.
// It delegates to internal/projectroot so database initialization shares the
// same marker-validation contract as the CLI and integration packages.
func findProjectRoot(startDir string) (string, error) {
	return findProjectRootFrom(startDir, "")
}

// findProjectRootFrom is findProjectRoot with an optional ceiling directory.
// ceiling, when non-empty, stops the upward walk at that directory (inclusive).
// This is used in tests to prevent the walk from escaping the temp directory
// tree and picking up markers from the host environment.
func findProjectRootFrom(startDir, ceiling string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return projectroot.FindProjectRootFrom(startDir, ceiling)
}
