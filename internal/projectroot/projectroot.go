// Package projectroot resolves the Shark project root by walking up the
// directory tree from a starting directory, looking for .sharkconfig.json,
// shark-tasks.db, or a valid .git marker (in that priority order).
//
// This logic originally lived only in internal/cli (FindProjectRoot /
// findProjectRootFrom). It was extracted here so a package that must not
// depend on internal/cli can still resolve the project root without
// creating an import cycle: internal/cli imports internal/services (service
// accessors, error helpers), so internal/integration importing internal/cli
// for this one helper would make internal/services -> internal/integration
// -> internal/cli -> internal/services a cycle the moment a services-layer
// caller (FeatureService.TransitionStatus, T-E34-F08-008) calls into
// internal/integration directly. internal/projectroot depends on nothing
// internal, so both internal/cli and internal/integration can depend on it
// without any cycle.
//
// internal/cli.FindProjectRoot and its package-private findProjectRootFrom
// test seam now delegate here; their signatures and behavior are unchanged.
package projectroot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindProjectRoot walks up the directory tree from the current working
// directory to find the project root. See FindProjectRootFrom for the
// search algorithm.
func FindProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return FindProjectRootFrom(wd, "")
}

// FindProjectRootFrom walks up the directory tree from startDir looking for
// project markers in priority order:
// 1. .sharkconfig.json (highest)
// 2. shark-tasks.db
// 3. .git/
//
// The full tree is always scanned so that a .sharkconfig.json in a parent
// directory wins over a .git in a closer ancestor. Returns startDir if no
// markers are found anywhere in the tree.
//
// ceiling, when non-empty, stops the search at that directory (inclusive).
// This is used in tests to prevent the walk from escaping the temp directory
// tree and picking up markers from the host environment.
func FindProjectRootFrom(startDir, ceiling string) (string, error) {
	var foundConfig, foundDB, foundGit string

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
					// A .git file is a worktree pointer and must contain a
					// "gitdir: <path>" line to be accepted. This rejects
					// stray/empty/garbage .git files (B054).
					if data, readErr := os.ReadFile(gitDir); readErr == nil {
						if strings.HasPrefix(strings.TrimSpace(string(data)), "gitdir:") {
							foundGit = currentDir
						}
					}
				}
			}
		}

		if ceiling != "" && currentDir == ceiling {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
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
	return startDir, nil
}
