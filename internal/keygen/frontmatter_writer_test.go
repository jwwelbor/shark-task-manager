package keygen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontmatterWriter_WriteTaskKey(t *testing.T) {
	writer := NewFrontmatterWriter()

	tests := []struct {
		name          string
		initialContent string
		taskKey       string
		wantContent   string
		wantErr       bool
	}{
		{
			name: "add task_key to existing frontmatter",
			initialContent: `---
description: Add caching layer for API responses
status: todo
---

# Implement Caching

Some content here.
`,
			taskKey: "T-E04-F02-003",
			wantContent: `---
description: Add caching layer for API responses
status: todo
task_key: T-E04-F02-003
---

# Implement Caching

Some content here.
`,
		},
		{
			name: "create frontmatter with task_key when none exists",
			initialContent: `# Task Title

Some content without frontmatter.
`,
			taskKey: "T-E04-F02-001",
			wantContent: `---
task_key: T-E04-F02-001
---

# Task Title

Some content without frontmatter.
`,
		},
		{
			name: "update existing task_key",
			initialContent: `---
task_key: T-E04-F02-001
description: Old description
---

Content here.
`,
			taskKey: "T-E04-F02-002",
			wantContent: `---
description: Old description
task_key: T-E04-F02-002
---

Content here.
`,
		},
		{
			name: "preserve all other frontmatter fields",
			initialContent: `---
title: Implement Feature
description: Feature description
status: in_progress
priority: 1
assigned_agent: backend
---

# Implementation

Content.
`,
			taskKey: "T-E04-F02-005",
			wantContent: `---
assigned_agent: backend
description: Feature description
priority: 1
status: in_progress
task_key: T-E04-F02-005
title: Implement Feature
---

# Implementation

Content.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.md")

			// Write initial content
			if err := os.WriteFile(tmpFile, []byte(tt.initialContent), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Write task key
			err := writer.WriteTaskKey(tmpFile, tt.taskKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WriteTaskKey() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("WriteTaskKey() unexpected error = %v", err)
				return
			}

			// Read result
			result, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			// Normalize whitespace for comparison
			gotContent := normalizeContent(string(result))
			wantContent := normalizeContent(tt.wantContent)

			if gotContent != wantContent {
				t.Errorf("WriteTaskKey() content mismatch:\n=== GOT ===\n%s\n=== WANT ===\n%s\n", gotContent, wantContent)
			}

			// Verify file permissions were preserved
			info, err := os.Stat(tmpFile)
			if err != nil {
				t.Fatalf("Failed to stat result file: %v", err)
			}
			if info.Mode().Perm() != 0644 {
				t.Errorf("File permissions not preserved: got %o, want %o", info.Mode().Perm(), 0644)
			}
		})
	}
}

func TestFrontmatterWriter_ReadFrontmatter(t *testing.T) {
	writer := NewFrontmatterWriter()

	tests := []struct {
		name         string
		fileContent  string
		wantTaskKey  string
		wantHasKey   bool
		wantErr      bool
	}{
		{
			name: "read existing task_key",
			fileContent: `---
task_key: T-E04-F02-003
description: Test task
---

Content here.
`,
			wantTaskKey: "T-E04-F02-003",
			wantHasKey:  true,
			wantErr:     false,
		},
		{
			name: "no task_key in frontmatter",
			fileContent: `---
description: Test task
status: todo
---

Content here.
`,
			wantTaskKey: "",
			wantHasKey:  false,
			wantErr:     false,
		},
		{
			name: "no frontmatter",
			fileContent: `# Just Content

No frontmatter here.
`,
			wantTaskKey: "",
			wantHasKey:  false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.md")

			if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Read frontmatter
			fm, err := writer.ReadFrontmatter(tmpFile)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadFrontmatter() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ReadFrontmatter() unexpected error = %v", err)
				return
			}

			// Check task_key
			hasKey, taskKey, err := writer.HasTaskKey(tmpFile)
			if err != nil {
				t.Errorf("HasTaskKey() unexpected error = %v", err)
				return
			}

			if hasKey != tt.wantHasKey {
				t.Errorf("HasTaskKey() hasKey = %v, want %v", hasKey, tt.wantHasKey)
			}

			if tt.wantHasKey && taskKey != tt.wantTaskKey {
				t.Errorf("HasTaskKey() taskKey = %v, want %v", taskKey, tt.wantTaskKey)
			}

			// Verify frontmatter structure
			if tt.wantHasKey {
				if key, ok := fm["task_key"]; !ok || key != tt.wantTaskKey {
					t.Errorf("Frontmatter task_key = %v, want %v", key, tt.wantTaskKey)
				}
			}
		})
	}
}

func TestFrontmatterWriter_ValidateFileWritable(t *testing.T) {
	writer := NewFrontmatterWriter()

	t.Run("writable file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.md")

		if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		err := writer.ValidateFileWritable(tmpFile)
		if err != nil {
			t.Errorf("ValidateFileWritable() unexpected error = %v", err)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		err := writer.ValidateFileWritable("/nonexistent/file.md")
		if err == nil {
			t.Errorf("ValidateFileWritable() expected error for non-existent file")
		}
	})

	t.Run("read-only file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "readonly.md")

		if err := os.WriteFile(tmpFile, []byte("content"), 0444); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		err := writer.ValidateFileWritable(tmpFile)
		if err == nil {
			t.Logf("ValidateFileWritable() expected error for read-only file (may pass on some systems)")
		}
	})
}

func TestFrontmatterWriter_AtomicWrite(t *testing.T) {
	writer := NewFrontmatterWriter()

	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "atomic_test.md")

	initialContent := `---
description: Initial content
---

# Test
`

	if err := os.WriteFile(tmpFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Verify no temp files left after successful write
	err := writer.WriteTaskKey(tmpFile, "T-E04-F02-001")
	if err != nil {
		t.Fatalf("WriteTaskKey() unexpected error = %v", err)
	}

	// Check no temp files remain
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".shark-tmp-") {
			t.Errorf("Temporary file not cleaned up: %s", file.Name())
		}
	}

	// Verify only one file exists (the updated one)
	if len(files) != 1 {
		t.Errorf("Expected 1 file in temp dir, got %d", len(files))
	}
}

// Helper function to normalize content for comparison
func normalizeContent(s string) string {
	// Remove trailing whitespace from lines and normalize line endings
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		normalized = append(normalized, strings.TrimRight(line, " \t"))
	}
	result := strings.Join(normalized, "\n")
	// Ensure consistent ending
	return strings.TrimRight(result, "\n") + "\n"
}
