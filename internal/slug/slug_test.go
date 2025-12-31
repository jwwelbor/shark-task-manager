package slug

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Some Task Description",
			expected: "some-task-description",
		},
		{
			name:     "title with special characters",
			input:    "Fix bug: API endpoint",
			expected: "fix-bug-api-endpoint",
		},
		{
			name:     "title with multiple spaces",
			input:    "Add    new     feature",
			expected: "add-new-feature",
		},
		{
			name:     "title with numbers",
			input:    "Update version 2.0",
			expected: "update-version-2-0",
		},
		{
			name:     "title with punctuation",
			input:    "Task: update docs (urgent)!",
			expected: "task-update-docs-urgent",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()",
			expected: "",
		},
		{
			name:     "mixed case with underscores",
			input:    "Update_Task_Manager",
			expected: "update-task-manager",
		},
		{
			name:     "unicode characters",
			input:    "Add émoji support",
			expected: "add-emoji-support",
		},
		{
			name:     "very long title should be truncated",
			input:    "This is a very long task title that should be truncated to avoid creating extremely long filenames that might cause issues",
			expected: "this-is-a-very-long-task-title-that-should-be-truncated-to-avoid-creating-extremely-long-filenames-t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Generate(tt.input)
			if result != tt.expected {
				t.Errorf("Generate(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name     string
		taskKey  string
		title    string
		expected string
	}{
		{
			name:     "task with simple title",
			taskKey:  "T-E04-F01-001",
			title:    "Some Task Description",
			expected: "T-E04-F01-001-some-task-description.md",
		},
		{
			name:     "task with special characters in title",
			taskKey:  "T-E07-F02-015",
			title:    "Fix bug: API endpoint",
			expected: "T-E07-F02-015-fix-bug-api-endpoint.md",
		},
		{
			name:     "task with empty title",
			taskKey:  "T-E04-F01-001",
			title:    "",
			expected: "T-E04-F01-001.md",
		},
		{
			name:     "task with only special characters",
			taskKey:  "T-E04-F01-001",
			title:    "!@#$",
			expected: "T-E04-F01-001.md",
		},
		{
			name:     "task with long title",
			taskKey:  "T-E04-F01-001",
			title:    "This is a very long task title that should be truncated",
			expected: "T-E04-F01-001-this-is-a-very-long-task-title-that-should-be-truncated.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFilename(tt.taskKey, tt.title)
			if result != tt.expected {
				t.Errorf("GenerateFilename(%q, %q) = %q, expected %q", tt.taskKey, tt.title, result, tt.expected)
			}
		})
	}
}
