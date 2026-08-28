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
// This logic mirrors internal/cli.findProjectRootFrom (including the .git
// content validation added for B054) but is duplicated here to avoid a
// circular import chain: cmd/server → internal/dbinit → internal/cli → cobra.
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
			gitDir := filepath.Join(currentDir, ".git")
			if info, err := os.Stat(gitDir); err == nil {
				if info.IsDir() {
					// A .git directory is only a valid marker if it looks like
					// a real git repo (has a HEAD file or an objects/ dir).
					// This rejects stray/empty .git directories (B054).
					_, headErr := os.Stat(filepath.Join(gitDir, "HEAD"))
					_, objectsErr := os.Stat(filepath.Join(gitDir, "objects"))
					if headErr == nil || objectsErr == nil {
						foundGit = currentDir
					}
				} else {
					// A .git file is a worktree pointer (contains "gitdir: <path>")
					// and is always accepted as-is.
					foundGit = currentDir
				}
			}
		}

		if ceiling != "" && currentDir == ceiling {
			break
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
