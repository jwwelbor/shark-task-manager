package taskcreation

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCustomFilename_ValidPaths tests successful validation of valid paths
func TestValidateCustomFilename_ValidPaths(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		projectRoot  string
		expectAbsEnd string // Expected end of absolute path (platform-agnostic)
		expectRelPath string
	}{
		{
			name:          "simple_markdown_file",
			filename:      "test.md",
			projectRoot:   "/project",
			expectRelPath: "test.md",
			expectAbsEnd:  filepath.Join("project", "test.md"),
		},
		{
			name:          "markdown_in_subdirectory",
			filename:      "docs/plan/task.md",
			projectRoot:   "/project",
			expectRelPath: filepath.Join("docs", "plan", "task.md"),
			expectAbsEnd:  filepath.Join("project", "docs", "plan", "task.md"),
		},
		{
			name:          "relative_path_with_dot",
			filename:      "./docs/task.md",
			projectRoot:   "/project",
			expectRelPath: filepath.Join("docs", "task.md"),
			expectAbsEnd:  filepath.Join("project", "docs", "task.md"),
		},
		{
			name:          "nested_directories",
			filename:      "docs/plan/E01/E01-F01/task.md",
			projectRoot:   "/project",
			expectRelPath: filepath.Join("docs", "plan", "E01", "E01-F01", "task.md"),
			expectAbsEnd:  filepath.Join("project", "docs", "plan", "E01", "E01-F01", "task.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &Creator{
				projectRoot: tt.projectRoot,
			}

			absPath, relPath, err := creator.ValidateCustomFilename(tt.filename, tt.projectRoot)

			require.NoError(t, err)
			assert.NotEmpty(t, absPath)
			assert.Equal(t, tt.expectRelPath, relPath)
			// Check that absolute path ends with expected path (platform-agnostic)
			assert.True(t, strings.HasSuffix(absPath, filepath.FromSlash(tt.expectAbsEnd)),
				"Expected absolute path to end with %s, got %s", tt.expectAbsEnd, absPath)
		})
	}
}

// TestValidateCustomFilename_InvalidPaths tests rejection of invalid paths
func TestValidateCustomFilename_InvalidPaths(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		projectRoot string
		expectError string
	}{
		{
			name:        "absolute_path_rejected",
			filename:    "/absolute/path.md",
			projectRoot: "/project",
			expectError: "absolute path",
		},
		{
			name:        "path_traversal_double_dot",
			filename:    "../outside.md",
			projectRoot: "/project",
			expectError: "..",
		},
		{
			name:        "path_traversal_in_middle",
			filename:    "docs/../../../outside.md",
			projectRoot: "/project",
			expectError: "..",
		},
		{
			name:        "wrong_extension_txt",
			filename:    "file.txt",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "wrong_extension_none",
			filename:    "file",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "empty_filename",
			filename:    "",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "dot_only",
			filename:    ".",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "double_dot_only",
			filename:    "..",
			projectRoot: "/project",
			expectError: "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &Creator{
				projectRoot: tt.projectRoot,
			}

			absPath, relPath, err := creator.ValidateCustomFilename(tt.filename, tt.projectRoot)

			assert.Error(t, err)
			assert.Empty(t, absPath)
			assert.Empty(t, relPath)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

// TestValidateCustomFilename_PathNormalization tests path normalization
func TestValidateCustomFilename_PathNormalization(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		expectRelPath string
	}{
		{
			name:          "forward_slashes_normalized",
			filename:      "docs/plan/task.md",
			expectRelPath: filepath.Join("docs", "plan", "task.md"),
		},
		{
			name:          "mixed_slashes_normalized",
			filename:      "./docs/plan/task.md",
			expectRelPath: filepath.Join("docs", "plan", "task.md"),
		},
		{
			name:          "leading_dot_slash_removed",
			filename:      "./task.md",
			expectRelPath: "task.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &Creator{
				projectRoot: "/project",
			}

			_, relPath, err := creator.ValidateCustomFilename(tt.filename, "/project")

			require.NoError(t, err)
			assert.Equal(t, tt.expectRelPath, relPath)
		})
	}
}

// TestValidateCustomFilename_CasePreservation tests that case is preserved
func TestValidateCustomFilename_CasePreservation(t *testing.T) {
	creator := &Creator{
		projectRoot: "/project",
	}

	_, relPath, err := creator.ValidateCustomFilename("Docs/Plan/MyTask.md", "/project")

	require.NoError(t, err)
	// Case should be preserved in the relative path
	assert.Contains(t, relPath, "Docs")
	assert.Contains(t, relPath, "Plan")
	assert.Contains(t, relPath, "MyTask")
}

// TestValidateCustomFilename_SpecialCharacters tests handling of special characters
func TestValidateCustomFilename_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		valid    bool
	}{
		{
			name:     "hyphenated_filename",
			filename: "my-task-name.md",
			valid:    true,
		},
		{
			name:     "underscored_filename",
			filename: "my_task_name.md",
			valid:    true,
		},
		{
			name:     "numbered_filename",
			filename: "task-001.md",
			valid:    true,
		},
		{
			name:     "numbers_in_path",
			filename: "E01/E01-F01/001.md",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &Creator{
				projectRoot: "/project",
			}

			_, _, err := creator.ValidateCustomFilename(tt.filename, "/project")

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestValidateCustomFilename_DeepNesting tests deeply nested paths
func TestValidateCustomFilename_DeepNesting(t *testing.T) {
	creator := &Creator{
		projectRoot: "/project",
	}

	// Deep nesting should be valid
	_, relPath, err := creator.ValidateCustomFilename(
		"docs/plan/E01/E01-F01/E01-F01-sub/task.md",
		"/project",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, relPath)
}

// TestValidateCustomFilename_AbsPathResolution tests absolute path resolution
func TestValidateCustomFilename_AbsPathResolution(t *testing.T) {
	creator := &Creator{
		projectRoot: "/project",
	}

	absPath, _, err := creator.ValidateCustomFilename("docs/task.md", "/project")

	require.NoError(t, err)
	// Absolute path should be absolute
	assert.True(t, filepath.IsAbs(absPath))
	// Should contain the project root
	assert.True(t, strings.HasPrefix(absPath, "/project") || absPath[0] == filepath.Separator)
}

// TestValidateCustomFilename_ConsistentResults tests that same input gives consistent output
func TestValidateCustomFilename_ConsistentResults(t *testing.T) {
	creator := &Creator{
		projectRoot: "/project",
	}

	filename := "docs/plan/task.md"
	projectRoot := "/project"

	// Call multiple times
	absPath1, relPath1, err1 := creator.ValidateCustomFilename(filename, projectRoot)
	absPath2, relPath2, err2 := creator.ValidateCustomFilename(filename, projectRoot)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, relPath1, relPath2)
	assert.Equal(t, absPath1, absPath2)
}
