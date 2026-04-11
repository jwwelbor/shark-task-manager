// Package dbinit provides cloud-aware database initialization that can be used
// by both cmd/shark (CLI) and cmd/server (HTTP viewer) without importing internal/cli.
package dbinit

import (
	"fmt"
	"os"
	"path/filepath"
)

// findProjectRoot walks up the directory tree from startDir to find the project root.
// It looks for markers with different priorities:
//  1. .sharkconfig.json (STRONGEST - always preferred)
//  2. shark-tasks.db (STRONG - used if no .sharkconfig.json found)
//  3. .git/ directory (WEAK - used if no stronger markers found)
//
// Returns the project root directory, or startDir if no markers found.
// This logic mirrors internal/cli.FindProjectRoot but is duplicated here
// to avoid a circular import chain: cmd/server → internal/dbinit → internal/cli → cobra.
func findProjectRoot(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Make startDir absolute.
	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Track the best marker found during the upward search.
	var foundConfig string // .sharkconfig.json (highest priority)
	var foundDB string     // shark-tasks.db (medium priority)
	var foundGit string    // .git directory (lowest priority)

	currentDir := startDir

	for {
		if foundConfig == "" {
			if _, err := os.Stat(filepath.Join(currentDir, ".sharkconfig.json")); err == nil {
				foundConfig = currentDir
			}
		}

		if foundDB == "" {
			if _, err := os.Stat(filepath.Join(currentDir, "shark-tasks.db")); err == nil {
				foundDB = currentDir
			}
		}

		if foundGit == "" {
			if _, err := os.Stat(filepath.Join(currentDir, ".git")); err == nil {
				foundGit = currentDir
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	if foundConfig != "" {
		return foundConfig, nil
	}
	if foundDB != "" {
		return foundDB, nil
	}
	if foundGit != "" {
		return foundGit, nil
	}

	// No markers found — fall back to the start directory.
	return startDir, nil
}
