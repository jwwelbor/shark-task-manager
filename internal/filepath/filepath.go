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

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

var (
	// taskKeyPattern validates task keys: T-{epic}-{feature}-{sequence}
	// Example: T-E04-F05-001
	taskKeyPattern = regexp.MustCompile(`^T-([A-Z0-9]+)-([A-Z0-9]+)-(\d{3})$`)
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

// FindProjectRoot delegates to cli.FindProjectRoot for consistent project root discovery.
// It looks for .sharkconfig.json, shark-tasks.db, or .git directory.
//
// This ensures that the filepath package uses the same project root logic as the rest of the application.
func FindProjectRoot() (string, error) {
	return cli.FindProjectRoot()
}

// ResetProjectRootCache is a no-op for backward compatibility with tests.
// The actual cache is managed by cli.FindProjectRoot.
func ResetProjectRootCache() {
	// No-op: cli package doesn't expose cache reset
	// This is OK because tests use separate working directories
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
