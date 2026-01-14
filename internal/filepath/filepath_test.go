package filepath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidTaskKey(t *testing.T) {
	tests := []struct {
		name     string
		taskKey  string
		expected bool
	}{
		{"valid task key", "T-E04-F05-001", true},
		{"valid with different numbers", "T-E01-F02-123", true},
		{"invalid - no T prefix", "E04-F05-001", false},
		{"invalid - lowercase", "t-e04-f05-001", false},
		{"invalid - missing sequence", "T-E04-F05", false},
		{"invalid - wrong format", "invalid-key", false},
		{"invalid - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTaskKey(tt.taskKey)
			if result != tt.expected {
				t.Errorf("IsValidTaskKey(%q) = %v, want %v", tt.taskKey, result, tt.expected)
			}
		})
	}
}

func TestParseTaskKey(t *testing.T) {
	tests := []struct {
		name         string
		taskKey      string
		wantEpic     string
		wantFeature  string
		wantSequence string
		wantErr      bool
	}{
		{
			name:         "valid task key",
			taskKey:      "T-E04-F05-001",
			wantEpic:     "E04",
			wantFeature:  "F05",
			wantSequence: "001",
			wantErr:      false,
		},
		{
			name:         "different numbers",
			taskKey:      "T-E01-F02-123",
			wantEpic:     "E01",
			wantFeature:  "F02",
			wantSequence: "123",
			wantErr:      false,
		},
		{
			name:    "invalid format",
			taskKey: "invalid-key",
			wantErr: true,
		},
		{
			name:    "empty",
			taskKey: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epic, feature, sequence, err := ParseTaskKey(tt.taskKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTaskKey(%q) expected error, got nil", tt.taskKey)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseTaskKey(%q) unexpected error: %v", tt.taskKey, err)
				return
			}

			if epic != tt.wantEpic {
				t.Errorf("epic = %q, want %q", epic, tt.wantEpic)
			}
			if feature != tt.wantFeature {
				t.Errorf("feature = %q, want %q", feature, tt.wantFeature)
			}
			if sequence != tt.wantSequence {
				t.Errorf("sequence = %q, want %q", sequence, tt.wantSequence)
			}
		})
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Reset cache before test
	ResetProjectRootCache()
	defer ResetProjectRootCache()

	root, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot() error = %v", err)
	}

	// Should find a project root (this test is running in the project)
	if root == "" {
		t.Error("FindProjectRoot() returned empty string")
	}

	// Should be an absolute path
	if !filepath.IsAbs(root) {
		t.Errorf("FindProjectRoot() returned relative path: %s", root)
	}

	// Should contain .git or go.mod
	hasGit := fileExists(filepath.Join(root, ".git"))
	hasGoMod := fileExists(filepath.Join(root, "go.mod"))

	if !hasGit && !hasGoMod {
		t.Errorf("FindProjectRoot() returned %s which contains neither .git nor go.mod", root)
	}

	// Calling again should return cached value
	root2, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot() second call error = %v", err)
	}
	if root != root2 {
		t.Errorf("FindProjectRoot() not using cache: first=%s, second=%s", root, root2)
	}
}

func TestGetTaskFilePath(t *testing.T) {
	// Reset cache
	ResetProjectRootCache()
	defer ResetProjectRootCache()

	tests := []struct {
		name        string
		epicKey     string
		featureKey  string
		taskKey     string
		wantErr     bool
		checkSuffix string
	}{
		{
			name:        "valid task",
			epicKey:     "E04-task-mgmt-cli-core",
			featureKey:  "F05-file-path-management",
			taskKey:     "T-E04-F05-001",
			wantErr:     false,
			checkSuffix: "docs/plan/E04-task-mgmt-cli-core/F05-file-path-management/tasks/T-E04-F05-001.md",
		},
		{
			name:       "invalid task key",
			epicKey:    "E04-task-mgmt-cli-core",
			featureKey: "F05-file-path-management",
			taskKey:    "invalid-key",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetTaskFilePath(tt.epicKey, tt.featureKey, tt.taskKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTaskFilePath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetTaskFilePath() unexpected error: %v", err)
				return
			}

			// Check that path is absolute
			if !filepath.IsAbs(path) {
				t.Errorf("GetTaskFilePath() returned relative path: %s", path)
			}

			// Check that path contains expected suffix
			if tt.checkSuffix != "" {
				// Normalize path separators for comparison
				normalizedPath := filepath.ToSlash(path)
				normalizedSuffix := filepath.ToSlash(tt.checkSuffix)

				if !strings.HasSuffix(normalizedPath, normalizedSuffix) {
					t.Errorf("GetTaskFilePath() = %s, want suffix %s", path, tt.checkSuffix)
				}
			}

			// Check that path contains all key components
			if !strings.Contains(path, tt.epicKey) {
				t.Errorf("path %s doesn't contain epic key %s", path, tt.epicKey)
			}
			if !strings.Contains(path, tt.featureKey) {
				t.Errorf("path %s doesn't contain feature key %s", path, tt.featureKey)
			}
			if !strings.Contains(path, tt.taskKey) {
				t.Errorf("path %s doesn't contain task key %s", path, tt.taskKey)
			}
		})
	}
}

func TestGetTasksDirectory(t *testing.T) {
	ResetProjectRootCache()
	defer ResetProjectRootCache()

	epicKey := "E04-task-mgmt-cli-core"
	featureKey := "F05-file-path-management"

	dir, err := GetTasksDirectory(epicKey, featureKey)
	if err != nil {
		t.Fatalf("GetTasksDirectory() error = %v", err)
	}

	// Should be absolute
	if !filepath.IsAbs(dir) {
		t.Errorf("GetTasksDirectory() returned relative path: %s", dir)
	}

	// Should contain expected components
	if !strings.Contains(dir, epicKey) {
		t.Errorf("directory %s doesn't contain epic key %s", dir, epicKey)
	}
	if !strings.Contains(dir, featureKey) {
		t.Errorf("directory %s doesn't contain feature key %s", dir, featureKey)
	}
	if !strings.HasSuffix(dir, "tasks") {
		t.Errorf("directory %s doesn't end with 'tasks'", dir)
	}
}

func TestCreateTasksDirectory(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Create temp directory with project marker
	tempDir := t.TempDir()

	// Create .sharkconfig.json marker so FindProjectRoot finds this directory
	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create config marker: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	epicKey := "E04-test-epic"
	featureKey := "F01-test-feature"

	// Create directory
	dir, err := CreateTasksDirectory(epicKey, featureKey)
	if err != nil {
		t.Fatalf("CreateTasksDirectory() error = %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("created directory doesn't exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("created path is not a directory")
	}

	// Call again - should be idempotent
	dir2, err := CreateTasksDirectory(epicKey, featureKey)
	if err != nil {
		t.Fatalf("CreateTasksDirectory() second call error = %v", err)
	}
	if dir != dir2 {
		t.Errorf("CreateTasksDirectory() not idempotent: first=%s, second=%s", dir, dir2)
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		taskKey  string
		wantErr  bool
	}{
		{
			name:     "valid path",
			filePath: "/home/user/project/docs/plan/E04-task-mgmt-cli-core/F05-file-path-management/tasks/T-E04-F05-001.md",
			taskKey:  "T-E04-F05-001",
			wantErr:  false,
		},
		{
			name:     "missing docs folder",
			filePath: "/home/user/project/plan/E04-task-mgmt-cli-core/F05-file-path-management/tasks/T-E04-F05-001.md",
			taskKey:  "T-E04-F05-001",
			wantErr:  true,
		},
		{
			name:     "wrong task key",
			filePath: "/home/user/project/docs/plan/E04-task-mgmt-cli-core/F05-file-path-management/tasks/T-E04-F05-002.md",
			taskKey:  "T-E04-F05-001",
			wantErr:  true,
		},
		{
			name:     "invalid task key",
			filePath: "/some/path",
			taskKey:  "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.filePath, tt.taskKey)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateFilePath() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFilePath() unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
