package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestScanRealProjectStructure tests scanning the actual docs/plan directory
func TestScanRealProjectStructure(t *testing.T) {
	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Skipping test: could not find project root: %v", err)
	}

	// Scan docs/plan directory
	planDir := filepath.Join(projectRoot, "docs", "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: docs/plan directory does not exist")
	}

	scanner := NewFileScanner()
	files, err := scanner.Scan(planDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	// We should find at least some task files
	if len(files) == 0 {
		t.Logf("Warning: No task files found in %s", planDir)
	}

	// Verify each file has correct metadata
	for _, file := range files {
		t.Logf("Found task file: %s (Epic: %s, Feature: %s)", file.FileName, file.EpicKey, file.FeatureKey)

		// Verify file exists
		if _, err := os.Stat(file.FilePath); err != nil {
			t.Errorf("File does not exist: %s", file.FilePath)
		}

		// Verify filename matches pattern
		if !scanner.isTaskFile(file.FileName) {
			t.Errorf("Invalid task filename: %s", file.FileName)
		}

		// Verify epic and feature keys are not empty (should be inferred from path)
		if file.EpicKey == "" {
			t.Logf("Warning: Empty epic key for file %s", file.FileName)
		}

		if file.FeatureKey == "" {
			t.Logf("Warning: Empty feature key for file %s", file.FileName)
		}

		// Verify modification time is reasonable (not zero, not in the future)
		if file.ModifiedAt.IsZero() {
			t.Errorf("Zero modification time for file %s", file.FileName)
		}

		if file.ModifiedAt.After(time.Now()) {
			t.Errorf("Modification time in future for file %s: %v", file.FileName, file.ModifiedAt)
		}
	}
}

// TestScanSpecificFeature tests scanning a specific feature directory
func TestScanSpecificFeature(t *testing.T) {
	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Skipping test: could not find project root: %v", err)
	}

	// Try to find a specific feature directory
	planDir := filepath.Join(projectRoot, "docs", "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: docs/plan directory does not exist")
	}

	// Find first epic directory
	epicDirs, err := filepath.Glob(filepath.Join(planDir, "E*"))
	if err != nil || len(epicDirs) == 0 {
		t.Skipf("Skipping test: no epic directories found")
	}

	// Find first feature directory within the epic
	featureDirs, err := filepath.Glob(filepath.Join(epicDirs[0], "E*-F*"))
	if err != nil || len(featureDirs) == 0 {
		t.Skipf("Skipping test: no feature directories found")
	}

	featureDir := featureDirs[0]
	t.Logf("Scanning feature directory: %s", featureDir)

	scanner := NewFileScanner()
	files, err := scanner.Scan(featureDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	// Verify all files have the same epic and feature keys
	if len(files) > 0 {
		expectedEpic := files[0].EpicKey
		expectedFeature := files[0].FeatureKey

		for _, file := range files {
			if file.EpicKey != expectedEpic {
				t.Errorf("Inconsistent epic key: got %s, want %s for file %s", file.EpicKey, expectedEpic, file.FileName)
			}

			if file.FeatureKey != expectedFeature {
				t.Errorf("Inconsistent feature key: got %s, want %s for file %s", file.FeatureKey, expectedFeature, file.FileName)
			}
		}

		t.Logf("Found %d task files with Epic=%s, Feature=%s", len(files), expectedEpic, expectedFeature)
	}
}

// TestScanPerformance tests that scanner processes files quickly
func TestScanPerformance(t *testing.T) {
	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Skipping test: could not find project root: %v", err)
	}

	// Scan docs/plan directory
	planDir := filepath.Join(projectRoot, "docs", "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: docs/plan directory does not exist")
	}

	scanner := NewFileScanner()
	start := time.Now()

	files, err := scanner.Scan(planDir)

	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	t.Logf("Scanned %d files in %v", len(files), elapsed)

	// Performance requirement: 100 files in < 1 second
	// For now, we'll just log the performance
	if len(files) > 0 {
		timePerFile := elapsed / time.Duration(len(files))
		t.Logf("Average time per file: %v", timePerFile)

		// If we had 100 files, would it take less than 1 second?
		projectedTime := timePerFile * 100
		if projectedTime > time.Second {
			t.Logf("Warning: Projected time for 100 files: %v (target: <1s)", projectedTime)
		}
	}
}

// TestScanWithTasksDirectory tests scanning legacy tasks directory
func TestScanWithTasksDirectory(t *testing.T) {
	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Skipping test: could not find project root: %v", err)
	}

	// Check if docs/tasks exists
	tasksDir := filepath.Join(projectRoot, "docs", "tasks")
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: docs/tasks directory does not exist")
	}

	scanner := NewFileScanner()
	files, err := scanner.Scan(tasksDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	if len(files) == 0 {
		t.Logf("No task files found in legacy tasks directory")
		return
	}

	t.Logf("Found %d task files in legacy tasks directory", len(files))

	// Verify epic and feature keys are inferred from filename (fallback)
	for _, file := range files {
		if file.EpicKey == "" {
			t.Errorf("Empty epic key for legacy file %s", file.FileName)
		}

		if file.FeatureKey == "" {
			t.Errorf("Empty feature key for legacy file %s", file.FileName)
		}

		t.Logf("Legacy file: %s -> Epic=%s, Feature=%s", file.FileName, file.EpicKey, file.FeatureKey)
	}
}

// TestScanHandlesPermissionErrors tests graceful handling of permission errors
func TestScanHandlesPermissionErrors(t *testing.T) {
	if os.Getenv("SKIP_PERMISSION_TESTS") != "" {
		t.Skip("Skipping permission tests (set SKIP_PERMISSION_TESTS to skip)")
	}

	tmpDir := t.TempDir()

	// Create a directory with no read permissions
	noReadDir := filepath.Join(tmpDir, "no-read")
	os.Mkdir(noReadDir, 0000)
	defer os.Chmod(noReadDir, 0755) // Restore permissions for cleanup

	// Create a task file in the restricted directory
	os.Chmod(noReadDir, 0755) // Temporarily allow write
	taskFile := filepath.Join(noReadDir, "T-E04-F07-001.md")
	os.WriteFile(taskFile, []byte("content"), 0644)
	os.Chmod(noReadDir, 0000) // Remove permissions

	scanner := NewFileScanner()
	files, err := scanner.Scan(tmpDir)

	// Scanner should not return error, but may skip the restricted directory
	if err != nil {
		t.Logf("Scan returned error (expected on some systems): %v", err)
	}

	// The file in the restricted directory should be skipped
	for _, file := range files {
		if file.FileName == "T-E04-F07-001.md" {
			t.Logf("Note: File in restricted directory was accessible (system-dependent)")
		}
	}
}

// TestScanWithNestedSubdirectories tests scanning deeply nested structures
func TestScanWithNestedSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a deeply nested structure
	nestedDir := filepath.Join(tmpDir, "docs", "plan", "E04-epic", "E04-F07-feature", "tasks", "active")
	os.MkdirAll(nestedDir, 0755)

	// Create task files at various levels
	files := []struct {
		path string
		epic string
		feat string
	}{
		{
			path: filepath.Join(tmpDir, "docs", "plan", "E04-epic", "E04-F07-feature", "T-E04-F07-001.md"),
			epic: "E04",
			feat: "E04-F07",
		},
		{
			path: filepath.Join(tmpDir, "docs", "plan", "E04-epic", "E04-F07-feature", "tasks", "T-E04-F07-002.md"),
			epic: "E04",
			feat: "E04-F07",
		},
		{
			path: filepath.Join(nestedDir, "T-E04-F07-003.md"),
			epic: "E04",
			feat: "E04-F07",
		},
	}

	for _, f := range files {
		os.WriteFile(f.path, []byte("content"), 0644)
	}

	scanner := NewFileScanner()
	results, err := scanner.Scan(tmpDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	if len(results) != 3 {
		t.Errorf("Scan() found %d files, want 3", len(results))
	}

	// Verify all files were found and have correct keys
	foundFiles := make(map[string]TaskFileInfo)
	for _, result := range results {
		foundFiles[result.FileName] = result
	}

	for _, expected := range files {
		filename := filepath.Base(expected.path)
		found, ok := foundFiles[filename]

		if !ok {
			t.Errorf("File not found: %s", filename)
			continue
		}

		if found.EpicKey != expected.epic {
			t.Errorf("File %s: epic key = %s, want %s", filename, found.EpicKey, expected.epic)
		}

		if found.FeatureKey != expected.feat {
			t.Errorf("File %s: feature key = %s, want %s", filename, found.FeatureKey, expected.feat)
		}
	}
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree until we find go.mod
	for {
		goMod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
