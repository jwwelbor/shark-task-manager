package utils

import (
	"fmt"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/slug"
)

// PathBuilder resolves file paths for epics, features, and tasks based on custom folder path inheritance
type PathBuilder struct {
	projectRoot string
}

// NewPathBuilder creates a new PathBuilder instance with the given project root
func NewPathBuilder(projectRoot string) *PathBuilder {
	return &PathBuilder{
		projectRoot: projectRoot,
	}
}

// ResolveEpicPath resolves the file path for an epic.md file
// Precedence:
// 1. filename (if non-nil) - exact override
// 2. customFolderPath (if non-nil) - use custom base path
// 3. default - docs/plan/<epicKey>/epic.md
func (pb *PathBuilder) ResolveEpicPath(epicKey string, filename *string, customFolderPath *string) (string, error) {
	// Precedence 1: Explicit filename override
	if filename != nil && *filename != "" {
		return *filename, nil
	}

	// Build the base directory
	var baseDir string

	// Precedence 2: Custom folder path
	if customFolderPath != nil && *customFolderPath != "" {
		_, relPath, err := ValidateFolderPath(*customFolderPath, pb.projectRoot)
		if err != nil {
			return "", fmt.Errorf("custom folder path validation failed: %w", err)
		}
		baseDir = filepath.Join(pb.projectRoot, relPath, epicKey)
	} else {
		// Precedence 3: Default path
		baseDir = filepath.Join(pb.projectRoot, "docs", "plan", epicKey)
	}

	return filepath.Join(baseDir, "epic.md"), nil
}

// ResolveFeaturePath resolves the file path for a feature.md file
// Precedence:
// 1. filename (if non-nil) - exact override
// 2. featureCustomPath (if non-nil) - feature's own custom path
// 3. epicCustomPath (if non-nil) - inherit from epic
// 4. default - docs/plan/<epicKey>/<featureKey>/feature.md
func (pb *PathBuilder) ResolveFeaturePath(epicKey, featureKey string, filename *string, featureCustomPath, epicCustomPath *string) (string, error) {
	// Precedence 1: Explicit filename override
	if filename != nil && *filename != "" {
		return *filename, nil
	}

	// Build the base directory
	var baseDir string

	// Precedence 2: Feature's own custom path
	if featureCustomPath != nil && *featureCustomPath != "" {
		_, relPath, err := ValidateFolderPath(*featureCustomPath, pb.projectRoot)
		if err != nil {
			return "", fmt.Errorf("feature custom folder path validation failed: %w", err)
		}
		baseDir = filepath.Join(pb.projectRoot, relPath, featureKey)
	} else if epicCustomPath != nil && *epicCustomPath != "" {
		// Precedence 3: Inherit from epic
		_, relPath, err := ValidateFolderPath(*epicCustomPath, pb.projectRoot)
		if err != nil {
			return "", fmt.Errorf("epic custom folder path validation failed: %w", err)
		}
		baseDir = filepath.Join(pb.projectRoot, relPath, epicKey, featureKey)
	} else {
		// Precedence 4: Default path
		baseDir = filepath.Join(pb.projectRoot, "docs", "plan", epicKey, featureKey)
	}

	return filepath.Join(baseDir, "feature.md"), nil
}

// ResolveTaskPath resolves the file path for a task.md file with optional slug from title
// Precedence:
// 1. filename (if non-nil) - exact override
// 2. featureCustomPath (if non-nil) - inherit from feature
// 3. epicCustomPath (if non-nil) - inherit from epic
// 4. default - docs/plan/<epicKey>/<featureKey>/tasks/<taskKey>[-<slug>].md
//
// If taskTitle is provided and non-empty, a slug will be generated and appended to the filename:
// - Without title: T-E04-F01-001.md
// - With title:    T-E04-F01-001-some-task-description.md
func (pb *PathBuilder) ResolveTaskPath(epicKey, featureKey, taskKey, taskTitle string, filename *string, featureCustomPath, epicCustomPath *string) (string, error) {
	// Precedence 1: Explicit filename override
	if filename != nil && *filename != "" {
		return *filename, nil
	}

	// Build the base directory
	var baseDir string

	// Precedence 2: Feature's custom path
	if featureCustomPath != nil && *featureCustomPath != "" {
		_, relPath, err := ValidateFolderPath(*featureCustomPath, pb.projectRoot)
		if err != nil {
			return "", fmt.Errorf("feature custom folder path validation failed: %w", err)
		}
		baseDir = filepath.Join(pb.projectRoot, relPath, featureKey, "tasks")
	} else if epicCustomPath != nil && *epicCustomPath != "" {
		// Precedence 3: Inherit from epic
		_, relPath, err := ValidateFolderPath(*epicCustomPath, pb.projectRoot)
		if err != nil {
			return "", fmt.Errorf("epic custom folder path validation failed: %w", err)
		}
		baseDir = filepath.Join(pb.projectRoot, relPath, epicKey, featureKey, "tasks")
	} else {
		// Precedence 4: Default path
		baseDir = filepath.Join(pb.projectRoot, "docs", "plan", epicKey, featureKey, "tasks")
	}

	// Generate filename with slug if title provided
	filename_str := slug.GenerateFilename(taskKey, taskTitle)

	return filepath.Join(baseDir, filename_str), nil
}
