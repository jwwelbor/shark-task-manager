package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Path validation error types
var (
	ErrAbsolutePath       = fmt.Errorf("path must be relative to project root")
	ErrPathTraversal      = fmt.Errorf("path contains '..' (path traversal not allowed)")
	ErrPathOutsideProject = fmt.Errorf("resolves outside project root")
	ErrEmptyPath          = fmt.Errorf("path is empty or whitespace only")
)

// ValidateFolderPath validates a custom folder path for security and correctness.
// It returns the absolute path, relative path, and any validation error.
func ValidateFolderPath(path, projectRoot string) (absPath string, relPath string, err error) {
	// Check for empty path
	if strings.TrimSpace(path) == "" {
		return "", "", ErrEmptyPath
	}

	// Check for absolute path (starts with /) BEFORE normalization
	if strings.HasPrefix(strings.TrimSpace(path), "/") {
		return "", "", fmt.Errorf("%w, got absolute path: %s", ErrAbsolutePath, path)
	}

	// Check for path traversal (..) as a path component BEFORE normalization
	// Split by path separator and check each component
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", "", fmt.Errorf("%w: %s", ErrPathTraversal, path)
		}
	}

	// Normalize the path (remove trailing slashes, clean ./ sequences)
	normalizedPath := filepath.Clean(path)

	// After normalization, check if we ended up with just "." (empty path)
	if normalizedPath == "." {
		return "", "", ErrEmptyPath
	}

	// Join with project root to get absolute path
	absPath = filepath.Join(projectRoot, normalizedPath)

	// Verify the absolute path is within projectRoot
	projectRootAbs := filepath.Clean(projectRoot)
	absPathNormalized := filepath.Clean(absPath)

	// Simple containment check: the absolute path must start with projectRoot/
	if !strings.HasPrefix(absPathNormalized, projectRootAbs) &&
		absPathNormalized != projectRootAbs {
		return "", "", fmt.Errorf("%w: %s", ErrPathOutsideProject, normalizedPath)
	}

	// Return absolute path, relative path, and no error
	return absPathNormalized, normalizedPath, nil
}
