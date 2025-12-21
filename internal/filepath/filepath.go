// Package filepath provides utilities for resolving task file paths.
//
// Task files are organized in a feature-based structure:
//
//	docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md
//
// Files NEVER move - they stay in their feature folder regardless of status changes.
// Status is tracked in the database and YAML frontmatter, not folder location.
package filepath

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// taskKeyPattern validates task keys: T-{epic}-{feature}-{sequence}
	// Example: T-E04-F05-001
	taskKeyPattern = regexp.MustCompile(`^T-([A-Z0-9]+)-([A-Z0-9]+)-(\d{3})$`)

	// projectRoot caches the discovered project root to avoid repeated filesystem searches
	projectRoot string
)

// GetTaskFilePath returns the absolute file path for a task based on its epic, feature, and task key.
//
// Example:
//
//	GetTaskFilePath("E04-task-mgmt-cli-core", "F05-file-path-management", "T-E04-F05-001")
//	Returns: /home/user/project/docs/plan/E04-task-mgmt-cli-core/F05-file-path-management/tasks/T-E04-F05-001.md
//
// The path is deterministic - given the same inputs, it always returns the same path.
// The file may or may not exist; this function only constructs the path.
func GetTaskFilePath(epicKey, featureKey, taskKey string) (string, error) {
	// Validate task key format
	if !IsValidTaskKey(taskKey) {
		return "", fmt.Errorf("invalid task key: %s (expected format: T-{epic}-{feature}-{seq}, e.g., T-E04-F05-001)", taskKey)
	}

	// Find project root
	root, err := FindProjectRoot()
	if err != nil {
		return "", err
	}

	// Construct path: {root}/docs/plan/{epic}/{feature}/tasks/{taskKey}.md
	path := filepath.Join(root, "docs", "plan", epicKey, featureKey, "tasks", taskKey+".md")

	return path, nil
}

// GetTasksDirectory returns the tasks directory path for a feature.
//
// Example:
//
//	GetTasksDirectory("E04-task-mgmt-cli-core", "F05-file-path-management")
//	Returns: /home/user/project/docs/plan/E04-task-mgmt-cli-core/F05-file-path-management/tasks
func GetTasksDirectory(epicKey, featureKey string) (string, error) {
	root, err := FindProjectRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(root, "docs", "plan", epicKey, featureKey, "tasks"), nil
}

// CreateTasksDirectory creates the tasks directory for a feature if it doesn't exist.
// This is idempotent - calling it multiple times is safe.
//
// Returns the absolute path to the created (or existing) directory.
func CreateTasksDirectory(epicKey, featureKey string) (string, error) {
	dir, err := GetTasksDirectory(epicKey, featureKey)
	if err != nil {
		return "", err
	}

	// Create directory with parents (mkdir -p behavior)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tasks directory %s: %w", dir, err)
	}

	return dir, nil
}

// FindProjectRoot searches for the project root by looking for .git directory or go.mod file.
// It starts from the current working directory and walks up the directory tree.
//
// The result is cached after the first successful search.
func FindProjectRoot() (string, error) {
	// Return cached value if available
	if projectRoot != "" {
		return projectRoot, nil
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up directory tree looking for .git or go.mod
	dir := cwd
	for {
		// Check for .git directory
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			projectRoot = dir
			return projectRoot, nil
		}

		// Check for go.mod file
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			projectRoot = dir
			return projectRoot, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root of filesystem without finding project markers
			return "", fmt.Errorf("project root not found (no .git or go.mod found in any parent directory)")
		}
		dir = parent
	}
}

// ResetProjectRootCache clears the cached project root.
// This is primarily useful for testing.
func ResetProjectRootCache() {
	projectRoot = ""
}

// IsValidTaskKey validates that a task key matches the expected format.
// Valid format: T-{epic}-{feature}-{sequence}
// Example: T-E04-F05-001
func IsValidTaskKey(taskKey string) bool {
	return taskKeyPattern.MatchString(taskKey)
}

// ParseTaskKey extracts the epic, feature, and sequence components from a task key.
// Returns error if the task key is invalid.
//
// Example:
//
//	ParseTaskKey("T-E04-F05-001") returns ("E04", "F05", "001", nil)
func ParseTaskKey(taskKey string) (epic, feature, sequence string, err error) {
	matches := taskKeyPattern.FindStringSubmatch(taskKey)
	if matches == nil {
		return "", "", "", fmt.Errorf("invalid task key: %s (expected format: T-{epic}-{feature}-{seq})", taskKey)
	}

	return matches[1], matches[2], matches[3], nil
}

// ValidateFilePath checks if a file path matches the expected pattern for a task file.
// This is useful for validation commands.
func ValidateFilePath(filePath, taskKey string) error {
	// Extract epic and feature from task key
	_, _, _, err := ParseTaskKey(taskKey)
	if err != nil {
		return err
	}

	// Normalize path separators for consistent checking
	normalizedPath := filepath.ToSlash(filePath)

	// Check that path contains the expected components in order
	expectedParts := []string{"docs/plan", "/tasks/", taskKey + ".md"}

	for _, part := range expectedParts {
		if !strings.Contains(normalizedPath, part) {
			return fmt.Errorf("file path %s doesn't match expected pattern for task %s (missing: %s)", filePath, taskKey, part)
		}
	}

	return nil
}
