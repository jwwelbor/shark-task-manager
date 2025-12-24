package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Test that the root command exists
	if RootCmd == nil {
		t.Fatal("RootCmd should not be nil")
	}

	// Test command properties
	if RootCmd.Use != "shark" {
		t.Errorf("Expected command use to be 'shark', got '%s'", RootCmd.Use)
	}

	// Version should default to "dev" (will be overridden at build time via ldflags)
	if RootCmd.Version != "dev" {
		t.Errorf("Expected default version to be 'dev', got '%s'", RootCmd.Version)
	}
}

func TestGlobalConfig(t *testing.T) {
	// Test that GlobalConfig exists
	if GlobalConfig == nil {
		t.Fatal("GlobalConfig should not be nil")
	}

	// Test default values
	if GlobalConfig.JSON {
		t.Error("Expected JSON to be false by default")
	}

	if GlobalConfig.NoColor {
		t.Error("Expected NoColor to be false by default")
	}

	if GlobalConfig.Verbose {
		t.Error("Expected Verbose to be false by default")
	}
}

func TestGetDBPath(t *testing.T) {
	// Set a test DB path
	GlobalConfig.DBPath = "test.db"

	path, err := GetDBPath()
	if err != nil {
		t.Fatalf("GetDBPath() returned error: %v", err)
	}

	if path == "" {
		t.Error("GetDBPath() returned empty path")
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create project structure:
	// tmpDir/
	//   .sharkconfig.json
	//   subdir1/
	//     subdir2/
	//       subdir3/

	// Create .sharkconfig.json in root
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create nested subdirectories
	subdir3 := filepath.Join(tmpDir, "subdir1", "subdir2", "subdir3")
	if err := os.MkdirAll(subdir3, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	tests := []struct {
		name         string
		startDir     string
		expectedRoot string
	}{
		{
			name:         "from root directory",
			startDir:     tmpDir,
			expectedRoot: tmpDir,
		},
		{
			name:         "from first level subdirectory",
			startDir:     filepath.Join(tmpDir, "subdir1"),
			expectedRoot: tmpDir,
		},
		{
			name:         "from deeply nested subdirectory",
			startDir:     subdir3,
			expectedRoot: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to the start directory
			if err := os.Chdir(tt.startDir); err != nil {
				t.Fatalf("Failed to change directory to %s: %v", tt.startDir, err)
			}

			// Find project root
			root, err := FindProjectRoot()
			if err != nil {
				t.Errorf("FindProjectRoot() error = %v", err)
				return
			}

			// Verify root is correct
			if root != tt.expectedRoot {
				t.Errorf("FindProjectRoot() = %v, want %v", root, tt.expectedRoot)
			}
		})
	}
}

func TestFindProjectRoot_WithDatabase(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create shark-tasks.db in root
	dbPath := filepath.Join(tmpDir, "shark-tasks.db")
	if err := os.WriteFile(dbPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create db file: %v", err)
	}

	// Create nested subdirectory
	subdir := filepath.Join(tmpDir, "docs", "plan")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to subdirectory
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Find project root
	root, err := FindProjectRoot()
	if err != nil {
		t.Errorf("FindProjectRoot() error = %v", err)
		return
	}

	// Verify root is correct
	if root != tmpDir {
		t.Errorf("FindProjectRoot() = %v, want %v", root, tmpDir)
	}
}

func TestFindProjectRoot_WithGitDir(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create .git directory in root
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create nested subdirectory
	subdir := filepath.Join(tmpDir, "internal", "cli")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to subdirectory
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Find project root
	root, err := FindProjectRoot()
	if err != nil {
		t.Errorf("FindProjectRoot() error = %v", err)
		return
	}

	// Verify root is correct
	if root != tmpDir {
		t.Errorf("FindProjectRoot() = %v, want %v", root, tmpDir)
	}
}

func TestFindProjectRoot_NoMarkers(t *testing.T) {
	// Create a temporary directory structure with no markers
	tmpDir := t.TempDir()

	// Create nested subdirectory
	subdir := filepath.Join(tmpDir, "some", "random", "path")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to subdirectory
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Find project root - should return current directory when no markers found
	root, err := FindProjectRoot()
	if err != nil {
		t.Errorf("FindProjectRoot() error = %v", err)
		return
	}

	// Should return the current working directory (subdir) since no markers were found
	if root != subdir {
		t.Errorf("FindProjectRoot() = %v, want %v (current dir when no markers)", root, subdir)
	}
}
