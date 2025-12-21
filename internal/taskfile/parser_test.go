package taskfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTaskFileContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantTaskKey string
		wantStatus  string
		wantTitle   string
	}{
		{
			name: "valid task file",
			content: `---
task_key: T-E04-F05-001
status: todo
title: Implement file path utilities
feature: E04-F05
created: 2025-12-16
---

# Task Description

This is the task content.
`,
			wantErr:     false,
			wantTaskKey: "T-E04-F05-001",
			wantStatus:  "todo",
			wantTitle:   "Implement file path utilities",
		},
		{
			name: "minimal valid task",
			content: `---
task_key: T-E01-F01-001
status: in_progress
title: Simple task
---
`,
			wantErr:     false,
			wantTaskKey: "T-E01-F01-001",
			wantStatus:  "in_progress",
			wantTitle:   "Simple task",
		},
		{
			name: "task with dependencies",
			content: `---
task_key: T-E04-F05-002
status: blocked
title: Advanced task
dependencies:
  - T-E04-F05-001
  - T-E04-F04-003
---

# Implementation

Details here.
`,
			wantErr:     false,
			wantTaskKey: "T-E04-F05-002",
			wantStatus:  "blocked",
			wantTitle:   "Advanced task",
		},
		{
			name:    "missing frontmatter",
			content: "# Just a heading\n\nNo frontmatter here",
			wantErr: true,
		},
		{
			name: "missing closing delimiter",
			content: `---
task_key: T-E04-F05-001
status: todo
title: Broken task
`,
			wantErr: true,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskFile, err := ParseTaskFileContent(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseTaskFileContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseTaskFileContent() unexpected error: %v", err)
				return
			}

			if taskFile.Metadata.TaskKey != tt.wantTaskKey {
				t.Errorf("TaskKey = %q, want %q", taskFile.Metadata.TaskKey, tt.wantTaskKey)
			}
			if taskFile.Metadata.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", taskFile.Metadata.Status, tt.wantStatus)
			}
			if taskFile.Metadata.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", taskFile.Metadata.Title, tt.wantTitle)
			}
		})
	}
}

func TestParseTaskFile(t *testing.T) {
	// Create temp file for testing
	tempDir := t.TempDir()
	taskFilePath := filepath.Join(tempDir, "T-E04-F05-001.md")

	content := `---
task_key: T-E04-F05-001
status: todo
title: Test task
description: A test task for parsing
priority: 5
---

# Task Description

This is the task content with **markdown** formatting.

## Section 2

More content here.
`

	err := os.WriteFile(taskFilePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the file
	taskFile, err := ParseTaskFile(taskFilePath)
	if err != nil {
		t.Fatalf("ParseTaskFile() error = %v", err)
	}

	// Verify metadata
	if taskFile.Metadata.TaskKey != "T-E04-F05-001" {
		t.Errorf("TaskKey = %q, want %q", taskFile.Metadata.TaskKey, "T-E04-F05-001")
	}
	if taskFile.Metadata.Status != "todo" {
		t.Errorf("Status = %q, want %q", taskFile.Metadata.Status, "todo")
	}
	if taskFile.Metadata.Title != "Test task" {
		t.Errorf("Title = %q, want %q", taskFile.Metadata.Title, "Test task")
	}
	if taskFile.Metadata.Description != "A test task for parsing" {
		t.Errorf("Description = %q, want %q", taskFile.Metadata.Description, "A test task for parsing")
	}
	if taskFile.Metadata.Priority != 5 {
		t.Errorf("Priority = %d, want %d", taskFile.Metadata.Priority, 5)
	}

	// Verify content is present
	if len(taskFile.Content) == 0 {
		t.Error("Content is empty")
	}
	if !contains(taskFile.Content, "Task Description") {
		t.Error("Content doesn't contain expected text")
	}
}

func TestParseTaskFile_NonExistent(t *testing.T) {
	_, err := ParseTaskFile("/nonexistent/file.md")
	if err == nil {
		t.Error("ParseTaskFile() expected error for nonexistent file, got nil")
	}
}

func TestParseTaskFile_InvalidFormat(t *testing.T) {
	tempDir := t.TempDir()
	taskFilePath := filepath.Join(tempDir, "invalid.md")

	// Create file without frontmatter
	content := "# Just a heading\n\nNo frontmatter here"
	err := os.WriteFile(taskFilePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = ParseTaskFile(taskFilePath)
	if err == nil {
		t.Error("ParseTaskFile() expected error for invalid format, got nil")
	}
}

func TestTaskMetadata_Validate(t *testing.T) {
	tests := []struct {
		name     string
		metadata TaskMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: TaskMetadata{
				TaskKey: "T-E04-F05-001",
				Status:  "todo",
				Title:   "Test task",
			},
			wantErr: false,
		},
		{
			name: "missing task_key",
			metadata: TaskMetadata{
				Status: "todo",
				Title:  "Test task",
			},
			wantErr: true,
		},
		{
			name: "missing status",
			metadata: TaskMetadata{
				TaskKey: "T-E04-F05-001",
				Title:   "Test task",
			},
			wantErr: true,
		},
		{
			name: "missing title",
			metadata: TaskMetadata{
				TaskKey: "T-E04-F05-001",
				Status:  "todo",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			metadata: TaskMetadata{
				TaskKey: "T-E04-F05-001",
				Status:  "invalid_status",
				Title:   "Test task",
			},
			wantErr: true,
		},
		{
			name: "all valid statuses",
			metadata: TaskMetadata{
				TaskKey: "T-E04-F05-001",
				Status:  "completed",
				Title:   "Test task",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseTaskFileContent_Dependencies(t *testing.T) {
	content := `---
task_key: T-E04-F05-003
status: blocked
title: Task with dependencies
dependencies:
  - T-E04-F05-001
  - T-E04-F05-002
---

# Content

Task content here.
`

	taskFile, err := ParseTaskFileContent(content)
	if err != nil {
		t.Fatalf("ParseTaskFileContent() error = %v", err)
	}

	if len(taskFile.Metadata.Dependencies) != 2 {
		t.Errorf("len(Dependencies) = %d, want 2", len(taskFile.Metadata.Dependencies))
	}

	expectedDeps := []string{"T-E04-F05-001", "T-E04-F05-002"}
	for i, dep := range taskFile.Metadata.Dependencies {
		if dep != expectedDeps[i] {
			t.Errorf("Dependencies[%d] = %q, want %q", i, dep, expectedDeps[i])
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
