package taskfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatTaskFile(t *testing.T) {
	tests := []struct {
		name         string
		taskFile     *TaskFile
		wantContains []string
		wantErr      bool
	}{
		{
			name: "minimal task",
			taskFile: &TaskFile{
				Metadata: TaskMetadata{
					TaskKey: "T-E04-F05-001",
					Status:  "todo",
					Title:   "Test task",
				},
				Content: "# Task Description\n\nContent here.",
			},
			wantContains: []string{
				"---",
				"task_key: T-E04-F05-001",
				"status: todo",
				"title: Test task",
				"# Task Description",
			},
			wantErr: false,
		},
		{
			name: "task with dependencies",
			taskFile: &TaskFile{
				Metadata: TaskMetadata{
					TaskKey: "T-E04-F05-002",
					Status:  "blocked",
					Title:   "Task with deps",
					Dependencies: []string{
						"T-E04-F05-001",
						"T-E04-F04-003",
					},
				},
				Content: "Implementation details.",
			},
			wantContains: []string{
				"dependencies:",
				"- T-E04-F05-001",
				"- T-E04-F04-003",
			},
			wantErr: false,
		},
		{
			name: "task with all fields",
			taskFile: &TaskFile{
				Metadata: TaskMetadata{
					TaskKey:       "T-E04-F05-003",
					Status:        "in_progress",
					Title:         "Full metadata task",
					Description:   "A task with all fields",
					Feature:       "E04-F05",
					CreatedAt:     "2025-12-16",
					AssignedAgent: "developer",
					Priority:      5,
					EstimatedTime: "4 hours",
				},
				Content: "# Implementation\n\nFull content here.",
			},
			wantContains: []string{
				"task_key: T-E04-F05-003",
				"status: in_progress",
				"title: Full metadata task",
				"description: A task with all fields",
				"feature: E04-F05",
				"priority: 5",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatTaskFile(tt.taskFile)

			if tt.wantErr {
				if err == nil {
					t.Error("FormatTaskFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("FormatTaskFile() unexpected error: %v", err)
				return
			}

			// Check for expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatTaskFile() result missing expected content: %q\nGot:\n%s", want, result)
				}
			}

			// Check frontmatter delimiters
			if !strings.HasPrefix(result, "---\n") {
				t.Error("FormatTaskFile() result doesn't start with '---\\n'")
			}

			// Count delimiter occurrences (should be 2: opening and closing)
			delimCount := strings.Count(result, "---")
			if delimCount < 2 {
				t.Errorf("FormatTaskFile() has %d '---' delimiters, want at least 2", delimCount)
			}

			// Check file ends with newline
			if !strings.HasSuffix(result, "\n") {
				t.Error("FormatTaskFile() result doesn't end with newline")
			}
		})
	}
}

func TestWriteTaskFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "tasks", "T-E04-F05-001.md")

	taskFile := &TaskFile{
		Metadata: TaskMetadata{
			TaskKey: "T-E04-F05-001",
			Status:  "todo",
			Title:   "Test task for writing",
		},
		Content: "# Task Description\n\nThis is a test task.",
	}

	// Write file
	err := WriteTaskFile(filePath, taskFile)
	if err != nil {
		t.Fatalf("WriteTaskFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("WriteTaskFile() didn't create file")
	}

	// Verify parent directory was created
	parentDir := filepath.Dir(filePath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("WriteTaskFile() didn't create parent directory")
	}

	// Read back and verify content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "task_key: T-E04-F05-001") {
		t.Error("Written file doesn't contain expected metadata")
	}
	if !strings.Contains(contentStr, "This is a test task") {
		t.Error("Written file doesn't contain expected content")
	}
}

func TestWriteTaskFile_InvalidMetadata(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.md")

	taskFile := &TaskFile{
		Metadata: TaskMetadata{
			// Missing required fields
			TaskKey: "T-E04-F05-001",
			// Status and Title missing
		},
		Content: "Content",
	}

	err := WriteTaskFile(filePath, taskFile)
	if err == nil {
		t.Error("WriteTaskFile() expected error for invalid metadata, got nil")
	}
}

func TestWriteTaskFile_Atomic(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "atomic-test.md")

	taskFile := &TaskFile{
		Metadata: TaskMetadata{
			TaskKey: "T-E04-F05-001",
			Status:  "todo",
			Title:   "Atomic write test",
		},
		Content: "Test content",
	}

	// Write file
	err := WriteTaskFile(filePath, taskFile)
	if err != nil {
		t.Fatalf("WriteTaskFile() error = %v", err)
	}

	// Verify temp file doesn't exist
	tempFile := filePath + ".tmp"
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("WriteTaskFile() left temp file behind")
	}

	// Verify actual file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("WriteTaskFile() didn't create final file")
	}
}

func TestUpdateTaskMetadata(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "T-E04-F05-001.md")

	// Create initial task file
	initialTask := &TaskFile{
		Metadata: TaskMetadata{
			TaskKey: "T-E04-F05-001",
			Status:  "todo",
			Title:   "Initial task",
		},
		Content: "# Original Content\n\nThis should be preserved.",
	}

	err := WriteTaskFile(filePath, initialTask)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Update metadata
	err = UpdateTaskMetadata(filePath, func(m *TaskMetadata) error {
		m.Status = "in_progress"
		m.AssignedAgent = "developer"
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateTaskMetadata() error = %v", err)
	}

	// Read back and verify
	updated, err := ParseTaskFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse updated file: %v", err)
	}

	// Check metadata was updated
	if updated.Metadata.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", updated.Metadata.Status, "in_progress")
	}
	if updated.Metadata.AssignedAgent != "developer" {
		t.Errorf("AssignedAgent = %q, want %q", updated.Metadata.AssignedAgent, "developer")
	}

	// Check other metadata preserved
	if updated.Metadata.TaskKey != "T-E04-F05-001" {
		t.Error("TaskKey was changed")
	}
	if updated.Metadata.Title != "Initial task" {
		t.Error("Title was changed")
	}

	// Check content preserved
	if !strings.Contains(updated.Content, "Original Content") {
		t.Error("Content was not preserved")
	}
}

func TestUpdateTaskMetadata_ValidationError(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "T-E04-F05-001.md")

	// Create initial task file
	initialTask := &TaskFile{
		Metadata: TaskMetadata{
			TaskKey: "T-E04-F05-001",
			Status:  "todo",
			Title:   "Initial task",
		},
		Content: "Content",
	}

	err := WriteTaskFile(filePath, initialTask)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Try to update with invalid status
	err = UpdateTaskMetadata(filePath, func(m *TaskMetadata) error {
		m.Status = "invalid_status"
		return nil
	})

	if err == nil {
		t.Error("UpdateTaskMetadata() expected error for invalid status, got nil")
	}

	// Verify file wasn't changed
	current, err := ParseTaskFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse file after failed update: %v", err)
	}

	if current.Metadata.Status != "todo" {
		t.Error("File was modified despite validation error")
	}
}

func TestCreateTaskFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "new-task.md")

	metadata := TaskMetadata{
		TaskKey: "T-E04-F05-001",
		Status:  "todo",
		Title:   "New task via CreateTaskFile",
	}

	content := "# Task Description\n\nThis is a new task."

	err := CreateTaskFile(filePath, metadata, content)
	if err != nil {
		t.Fatalf("CreateTaskFile() error = %v", err)
	}

	// Verify file was created correctly
	taskFile, err := ParseTaskFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse created file: %v", err)
	}

	if taskFile.Metadata.TaskKey != metadata.TaskKey {
		t.Error("TaskKey mismatch")
	}
	if taskFile.Metadata.Status != metadata.Status {
		t.Error("Status mismatch")
	}
	if taskFile.Metadata.Title != metadata.Title {
		t.Error("Title mismatch")
	}
	if !strings.Contains(taskFile.Content, content) {
		t.Error("Content mismatch")
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that we can write and read back a task file without data loss
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "roundtrip.md")

	original := &TaskFile{
		Metadata: TaskMetadata{
			TaskKey:       "T-E04-F05-001",
			Status:        "in_progress",
			Title:         "Round trip test",
			Description:   "Testing round trip write/read",
			Feature:       "E04-F05",
			Priority:      7,
			Dependencies:  []string{"T-E04-F05-000"},
			AssignedAgent: "developer",
		},
		Content: "# Implementation\n\n## Step 1\n\nDo something.\n\n## Step 2\n\nDo more.",
	}

	// Write
	err := WriteTaskFile(filePath, original)
	if err != nil {
		t.Fatalf("WriteTaskFile() error = %v", err)
	}

	// Read back
	readBack, err := ParseTaskFile(filePath)
	if err != nil {
		t.Fatalf("ParseTaskFile() error = %v", err)
	}

	// Verify metadata
	if readBack.Metadata.TaskKey != original.Metadata.TaskKey {
		t.Error("TaskKey mismatch")
	}
	if readBack.Metadata.Status != original.Metadata.Status {
		t.Error("Status mismatch")
	}
	if readBack.Metadata.Title != original.Metadata.Title {
		t.Error("Title mismatch")
	}
	if readBack.Metadata.Priority != original.Metadata.Priority {
		t.Error("Priority mismatch")
	}
	if len(readBack.Metadata.Dependencies) != len(original.Metadata.Dependencies) {
		t.Error("Dependencies count mismatch")
	}

	// Verify content (may have whitespace differences, so check key parts)
	if !strings.Contains(readBack.Content, "Implementation") {
		t.Error("Content missing 'Implementation'")
	}
	if !strings.Contains(readBack.Content, "Step 1") {
		t.Error("Content missing 'Step 1'")
	}
}
