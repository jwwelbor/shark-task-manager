package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditService provides filesystem write operations for the web viewer.
// It is the write-side counterpart to ViewerService.File / FileByPath.
// Kept separate per ADR-E27-003: ViewerService is explicitly read-only.
type EditService struct {
	projectRoot string
}

// NewEditService creates a new EditService rooted at projectRoot.
func NewEditService(projectRoot string) *EditService {
	return &EditService{projectRoot: projectRoot}
}

// WriteFileResult is returned on successful writes.
type WriteFileResult struct {
	Path         string `json:"path"`
	BytesWritten int    `json:"bytes_written"`
}

// WriteFile writes content to the file at the given relative path within
// the project root.
//
// Security: rejects absolute paths and paths that resolve outside projectRoot
// (same canonicalization as ViewerService.FileByPath).
//
// Atomicity: writes to path+".tmp", then os.Rename. Removes .tmp on failure.
//
// Returns SecurityError if the path escapes the project root.
func (s *EditService) WriteFile(ctx context.Context, relPath string, content string) (*WriteFileResult, error) {
	// Step 1: Reject absolute paths immediately.
	if filepath.IsAbs(relPath) {
		return nil, &SecurityError{Path: relPath}
	}

	// Step 2: Resolve project root to an absolute path.
	absRoot, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("edit service: failed to resolve project root: %w", err)
	}

	// Step 3: Join root + relPath to get absolute target.
	absPath := filepath.Join(absRoot, relPath)

	// Step 4: Canonicalize root via EvalSymlinks.
	rootCanon, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("edit service: failed to canonicalize project root: %w", err)
	}

	// Step 4b: Canonicalize the target path via EvalSymlinks.
	// For write operations the target file may not exist yet, so we evaluate
	// the parent directory and re-attach the base name.
	targetCanon, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("edit service: failed to canonicalize file path: %w", err)
		}
		// File doesn't exist yet — evaluate parent directory instead.
		parentCanon, parentErr := filepath.EvalSymlinks(filepath.Dir(absPath))
		if parentErr != nil {
			return nil, fmt.Errorf("edit service: failed to canonicalize parent directory: %w", parentErr)
		}
		targetCanon = filepath.Join(parentCanon, filepath.Base(absPath))
	}

	// Step 5: Containment check — resolved target must be within rootCanon.
	prefix := rootCanon + string(os.PathSeparator)
	if targetCanon != rootCanon && !strings.HasPrefix(targetCanon, prefix) {
		return nil, &SecurityError{Path: targetCanon}
	}

	// Step 6: Atomic write — write to a unique temp file in the same directory,
	// then os.Rename to the target.  Using os.CreateTemp with a pattern that
	// includes the base name keeps the temp file on the same filesystem (required
	// for Rename atomicity) while avoiding collisions between concurrent callers.
	dir := filepath.Dir(targetCanon)
	base := filepath.Base(targetCanon)
	tmpFile, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return nil, fmt.Errorf("edit service: failed to create temp file for %q: %w", targetCanon, err)
	}
	tmpPath := tmpFile.Name()

	// Write content then close before rename.
	_, writeErr := tmpFile.WriteString(content)
	closeErr := tmpFile.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("edit service: failed to write to temp file %q: %w", tmpPath, writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("edit service: failed to close temp file %q: %w", tmpPath, closeErr)
	}

	if err := os.Rename(tmpPath, targetCanon); err != nil {
		// Rename failed — clean up the .tmp file.
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("edit service: failed to rename %q to %q: %w", tmpPath, targetCanon, err)
	}

	// Compute the relative path from root to the canonical target for the result.
	relResult, err := filepath.Rel(rootCanon, targetCanon)
	if err != nil {
		// Fallback to the input path if Rel fails (should not happen).
		relResult = relPath
	}

	return &WriteFileResult{
		Path:         relResult,
		BytesWritten: len(content),
	}, nil
}
