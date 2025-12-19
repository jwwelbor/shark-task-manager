package utils

import (
	"strings"
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Task Management CLI Core", "task-management-cli-core"},
		{"Task Creation", "task-creation"},
		{"UPPERCASE TEXT", "uppercase-text"},
		{"Special @#$ Characters!", "special-characters"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"hello_world", "hello-world"},
		{"CamelCaseTitle", "camelcasetitle"},
		{"Title with numbers 123", "title-with-numbers-123"},
		{"---triple-hyphens---", "triple-hyphens"},
		{"  leading and trailing spaces  ", "leading-and-trailing-spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GenerateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateSlug_LongTitle(t *testing.T) {
	// Create a title longer than 50 characters
	longTitle := "This is a very long title that exceeds fifty characters and should be truncated"
	result := GenerateSlug(longTitle)

	// Should be truncated to 50 characters (after trimming trailing hyphens)
	if len(result) > 50 {
		t.Errorf("GenerateSlug should truncate to 50 chars, got %d chars: %q", len(result), result)
	}

	// Should not start or end with hyphens
	if strings.HasPrefix(result, "-") || strings.HasSuffix(result, "-") {
		t.Errorf("GenerateSlug result should not start/end with hyphens: %q", result)
	}
}

func TestGenerateSlug_Empty(t *testing.T) {
	result := GenerateSlug("")
	if result != "" {
		t.Errorf("GenerateSlug(\"\") = %q, want \"\"", result)
	}
}

func TestGenerateSlug_OnlySpecialChars(t *testing.T) {
	result := GenerateSlug("@#$%^&*()")
	if result != "" {
		t.Errorf("GenerateSlug with only special chars should return empty, got %q", result)
	}
}
