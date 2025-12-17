package taskfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteTaskFile writes a task file with YAML frontmatter and markdown content.
// It creates parent directories if needed and uses atomic writes (temp file + rename).
//
// The file will be formatted as:
//
//	---
//	task_key: T-E04-F05-001
//	status: todo
//	title: Task title
//	---
//
//	# Task content
//	Markdown content here...
func WriteTaskFile(filePath string, taskFile *TaskFile) error {
	// Validate metadata before writing
	if err := taskFile.Metadata.Validate(); err != nil {
		return fmt.Errorf("invalid task metadata: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Generate file content
	content, err := FormatTaskFile(taskFile)
	if err != nil {
		return fmt.Errorf("failed to format task file: %w", err)
	}

	// Write atomically: write to temp file, then rename
	tempFile := filePath + ".tmp"

	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", tempFile, err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, filePath); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file to %s: %w", filePath, err)
	}

	return nil
}

// FormatTaskFile formats a TaskFile into a string with frontmatter and content.
// Returns the formatted string ready to be written to disk.
func FormatTaskFile(taskFile *TaskFile) (string, error) {
	// Marshal metadata to YAML
	yamlBytes, err := yaml.Marshal(&taskFile.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata to YAML: %w", err)
	}

	// Build file content
	var builder strings.Builder

	// Write frontmatter delimiter
	builder.WriteString("---\n")

	// Write YAML metadata
	builder.Write(yamlBytes)

	// Write closing delimiter
	builder.WriteString("---\n")

	// Write content (if present)
	if taskFile.Content != "" {
		// Ensure content starts with a newline after frontmatter
		if !strings.HasPrefix(taskFile.Content, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString(taskFile.Content)
	}

	// Ensure file ends with newline
	result := builder.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result, nil
}

// UpdateTaskMetadata updates an existing task file's metadata while preserving content.
// This is useful for status changes without rewriting the entire file.
func UpdateTaskMetadata(filePath string, updater func(*TaskMetadata) error) error {
	// Read existing file
	taskFile, err := ParseTaskFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Update metadata using provided function
	if err := updater(&taskFile.Metadata); err != nil {
		return fmt.Errorf("metadata update failed: %w", err)
	}

	// Validate updated metadata
	if err := taskFile.Metadata.Validate(); err != nil {
		return fmt.Errorf("invalid updated metadata: %w", err)
	}

	// Write updated file
	if err := WriteTaskFile(filePath, taskFile); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	return nil
}

// CreateTaskFile is a convenience function that creates a new task file with standard formatting.
func CreateTaskFile(filePath string, metadata TaskMetadata, content string) error {
	taskFile := &TaskFile{
		Metadata: metadata,
		Content:  content,
	}

	return WriteTaskFile(filePath, taskFile)
}

// UpdateFrontmatterField updates a specific field in the frontmatter while preserving other content.
// This is a convenience wrapper around UpdateTaskMetadata for simple single-field updates.
func UpdateFrontmatterField(filePath string, fieldName string, value interface{}) error {
	return UpdateTaskMetadata(filePath, func(metadata *TaskMetadata) error {
		switch fieldName {
		case "task_key":
			if v, ok := value.(string); ok {
				metadata.TaskKey = v
			} else {
				return fmt.Errorf("task_key must be a string")
			}
		case "title":
			if v, ok := value.(string); ok {
				metadata.Title = v
			} else {
				return fmt.Errorf("title must be a string")
			}
		case "description":
			if v, ok := value.(string); ok {
				metadata.Description = v
			} else {
				return fmt.Errorf("description must be a string")
			}
		default:
			return fmt.Errorf("unsupported field: %s", fieldName)
		}
		return nil
	})
}
