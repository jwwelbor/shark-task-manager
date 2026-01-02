package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestValidateCustomPath_NotProvided tests that ValidateCustomPath returns nil when flag not provided
func TestValidateCustomPath_NotProvided(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("path", "", "test path flag")

	result, err := ValidateCustomPath(cmd, "path")

	if err != nil {
		t.Errorf("Expected no error when flag not provided, got: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result when flag not provided, got: %+v", result)
	}
}

// TestValidateCustomPath_Valid tests valid path cases
func TestValidateCustomPath_Valid(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantRelative string
	}{
		{
			name:         "simple relative path",
			path:         "docs/plan",
			wantRelative: "docs/plan",
		},
		{
			name:         "nested relative path",
			path:         "docs/roadmap/2025",
			wantRelative: "docs/roadmap/2025",
		},
		{
			name:         "path with trailing slash",
			path:         "docs/plan/",
			wantRelative: "docs/plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("path", "", "test path flag")
			_ = cmd.Flags().Set("path", tt.path)

			result, err := ValidateCustomPath(cmd, "path")

			if err != nil {
				t.Errorf("Expected no error for valid path %q, got: %v", tt.path, err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result for valid path")
			}
			if result.RelativePath != tt.wantRelative {
				t.Errorf("Expected relative path %q, got %q", tt.wantRelative, result.RelativePath)
			}
			if result.AbsolutePath == "" {
				t.Error("Expected non-empty absolute path")
			}
		})
	}
}

// TestValidateCustomPath_Invalid tests invalid path cases
func TestValidateCustomPath_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantErrType string
	}{
		{
			name:        "absolute path",
			path:        "/absolute/path",
			wantErrType: "absolute",
		},
		{
			name:        "path traversal with ..",
			path:        "../outside",
			wantErrType: "traversal",
		},
		{
			name:        "empty path",
			path:        "   ",
			wantErrType: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("path", "", "test path flag")
			_ = cmd.Flags().Set("path", tt.path)

			result, err := ValidateCustomPath(cmd, "path")

			if err == nil {
				t.Errorf("Expected error for invalid path %q, got nil", tt.path)
			}
			if result != nil {
				t.Errorf("Expected nil result for invalid path, got: %+v", result)
			}
		})
	}
}

// TestValidateCustomFilename_NotProvided tests that ValidateCustomFilename returns nil when flag not provided
func TestValidateCustomFilename_NotProvided(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("filename", "", "test filename flag")

	result, err := ValidateCustomFilename(cmd, "filename", "/project/root")

	if err != nil {
		t.Errorf("Expected no error when flag not provided, got: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result when flag not provided, got: %+v", result)
	}
}

// TestValidateCustomFilename_Valid tests valid filename cases
func TestValidateCustomFilename_Valid(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		wantRelative string
	}{
		{
			name:         "markdown file in docs",
			filename:     "docs/epic.md",
			wantRelative: "docs/epic.md",
		},
		{
			name:         "markdown file in subdirectory",
			filename:     "docs/plan/feature.md",
			wantRelative: "docs/plan/feature.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("filename", "", "test filename flag")
			_ = cmd.Flags().Set("filename", tt.filename)

			result, err := ValidateCustomFilename(cmd, "filename", "/project/root")

			if err != nil {
				t.Errorf("Expected no error for valid filename %q, got: %v", tt.filename, err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result for valid filename")
			}
			if result.RelativePath != tt.wantRelative {
				t.Errorf("Expected relative path %q, got %q", tt.wantRelative, result.RelativePath)
			}
		})
	}
}

// TestValidateCustomFilename_Invalid tests invalid filename cases
func TestValidateCustomFilename_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "missing .md extension",
			filename: "docs/file.txt",
		},
		{
			name:     "absolute path",
			filename: "/absolute/path.md",
		},
		{
			name:     "path traversal",
			filename: "../outside.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("filename", "", "test filename flag")
			_ = cmd.Flags().Set("filename", tt.filename)

			result, err := ValidateCustomFilename(cmd, "filename", "/project/root")

			if err == nil {
				t.Errorf("Expected error for invalid filename %q, got nil", tt.filename)
			}
			if result != nil {
				t.Errorf("Expected nil result for invalid filename, got: %+v", result)
			}
		})
	}
}

// TestValidateNoSpaces_Valid tests keys without spaces
func TestValidateNoSpaces_Valid(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		entityType string
	}{
		{
			name:       "simple key",
			key:        "my-epic",
			entityType: "epic",
		},
		{
			name:       "key with numbers",
			key:        "E01-enhancements",
			entityType: "epic",
		},
		{
			name:       "key with underscores",
			key:        "feature_one",
			entityType: "feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoSpaces(tt.key, tt.entityType)

			if err != nil {
				t.Errorf("Expected no error for key %q, got: %v", tt.key, err)
			}
		})
	}
}

// TestValidateNoSpaces_Invalid tests keys with spaces
func TestValidateNoSpaces_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		entityType string
	}{
		{
			name:       "key with spaces",
			key:        "my epic",
			entityType: "epic",
		},
		{
			name:       "key with multiple spaces",
			key:        "feature  one",
			entityType: "feature",
		},
		{
			name:       "key with leading space",
			key:        " leading",
			entityType: "epic",
		},
		{
			name:       "key with trailing space",
			key:        "trailing ",
			entityType: "epic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoSpaces(tt.key, tt.entityType)

			if err == nil {
				t.Errorf("Expected error for key with spaces %q, got nil", tt.key)
			}
		})
	}
}

// TestValidateStatus_AllValidValues tests all valid status values
func TestValidateStatus_AllValidValues(t *testing.T) {
	validStatuses := []string{"draft", "active", "completed", "archived"}

	// Test for epic entity type
	for _, status := range validStatuses {
		t.Run("epic_"+status, func(t *testing.T) {
			err := ValidateStatus(status, "epic")

			if err != nil {
				t.Errorf("Expected no error for valid status %q, got: %v", status, err)
			}
		})
	}

	// Test for feature entity type
	for _, status := range validStatuses {
		t.Run("feature_"+status, func(t *testing.T) {
			err := ValidateStatus(status, "feature")

			if err != nil {
				t.Errorf("Expected no error for valid status %q, got: %v", status, err)
			}
		})
	}

	// Test for generic entity type
	for _, status := range validStatuses {
		t.Run("generic_"+status, func(t *testing.T) {
			err := ValidateStatus(status, "task")

			if err != nil {
				t.Errorf("Expected no error for valid status %q, got: %v", status, err)
			}
		})
	}
}

// TestValidateStatus_InvalidValue tests invalid status value
func TestValidateStatus_InvalidValue(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{
			name:   "invalid status",
			status: "invalid",
		},
		{
			name:   "empty status",
			status: "",
		},
		{
			name:   "status with wrong case",
			status: "Draft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStatus(tt.status, "epic")

			if err == nil {
				t.Errorf("Expected error for invalid status %q, got nil", tt.status)
			}
		})
	}
}

// TestValidatePriority_AllValidValues tests all valid priority values
func TestValidatePriority_AllValidValues(t *testing.T) {
	validPriorities := []string{"low", "medium", "high"}

	for _, priority := range validPriorities {
		t.Run(priority, func(t *testing.T) {
			err := ValidatePriority(priority, "epic")

			if err != nil {
				t.Errorf("Expected no error for valid priority %q, got: %v", priority, err)
			}
		})
	}
}

// TestValidatePriority_InvalidValue tests invalid priority value
func TestValidatePriority_InvalidValue(t *testing.T) {
	tests := []struct {
		name     string
		priority string
	}{
		{
			name:     "invalid priority",
			priority: "invalid",
		},
		{
			name:     "empty priority",
			priority: "",
		},
		{
			name:     "priority with wrong case",
			priority: "High",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriority(tt.priority, "epic")

			if err == nil {
				t.Errorf("Expected error for invalid priority %q, got nil", tt.priority)
			}
		})
	}
}
