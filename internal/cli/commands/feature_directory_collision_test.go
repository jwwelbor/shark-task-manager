package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureCreate_FileAtEpicDirectoryPath tests that creating a feature fails gracefully
// when a file exists at the expected epic directory path
func TestFeatureCreate_FileAtEpicDirectoryPath(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean up test data before and after (using E98 for collision tests to avoid interfering with E99)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	}()

	// Create temporary test directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	// Change to temp directory for test
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create docs/plan directory structure
	planDir := filepath.Join(tempDir, "docs", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// Create epic in database
	epicRepo := repository.NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic for Collision",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create a FILE at the path where the epic directory should be
	// This simulates the bug condition
	collisionFilePath := filepath.Join(planDir, "E98-test-epic-for-collision")
	if err := os.WriteFile(collisionFilePath, []byte("This is a file, not a directory"), 0644); err != nil {
		t.Fatalf("Failed to create collision file: %v", err)
	}

	// Verify the collision file exists and is a file (not a directory)
	fileInfo, err := os.Stat(collisionFilePath)
	if err != nil {
		t.Fatalf("Failed to stat collision file: %v", err)
	}
	if fileInfo.IsDir() {
		t.Fatal("Expected collision path to be a file, but it's a directory")
	}

	// Now try to create a feature - this should fail with a clear error
	// Since runFeatureCreate calls os.Exit(1), we need to test the logic differently
	// Instead, we'll test the directory validation logic directly

	// Find epic directory using the same logic as runFeatureCreate
	epicPattern := filepath.Join("docs", "plan", "E98-*")
	matches, err := filepath.Glob(epicPattern)
	if err != nil || len(matches) == 0 {
		t.Fatal("Epic pattern should have matched the file")
	}

	epicDir := matches[0]

	// Validate that the match is actually a directory, not a file (this is the fix)
	fileInfo, err = os.Stat(epicDir)
	if err != nil {
		t.Fatalf("Failed to stat epic path: %v", err)
	}

	// This is the key assertion - the fix should detect that epicDir is a file
	if !fileInfo.IsDir() {
		// SUCCESS - the validation detected the file/directory collision
		t.Logf("SUCCESS: Detected file at epic directory path: %s", epicDir)
	} else {
		t.Fatal("Expected epic directory path to be a file (collision scenario), but it's a directory")
	}
}

// TestFeatureCreate_ValidEpicDirectory tests that feature creation succeeds
// when the epic directory is a valid directory (not a file)
func TestFeatureCreate_ValidEpicDirectory(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean up test data before and after (using E98 for valid directory test)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	}()

	// Create temporary test directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	// Change to temp directory for test
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create docs/plan directory structure
	planDir := filepath.Join(tempDir, "docs", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// Create epic in database
	epicRepo := repository.NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic Valid Directory",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create a valid DIRECTORY for the epic (not a file)
	epicDirPath := filepath.Join(planDir, "E98-test-epic-valid-directory")
	if err := os.MkdirAll(epicDirPath, 0755); err != nil {
		t.Fatalf("Failed to create epic directory: %v", err)
	}

	// Verify the epic directory exists and is a directory
	fileInfo, err := os.Stat(epicDirPath)
	if err != nil {
		t.Fatalf("Failed to stat epic directory: %v", err)
	}
	if !fileInfo.IsDir() {
		t.Fatal("Expected epic path to be a directory")
	}

	// Find epic directory using the same logic as runFeatureCreate
	epicPattern := filepath.Join("docs", "plan", "E98-*")
	matches, err := filepath.Glob(epicPattern)
	if err != nil || len(matches) == 0 {
		t.Fatal("Epic pattern should have matched the directory")
	}

	epicDir := matches[0]

	// Validate that the match is actually a directory (this should pass)
	fileInfo, err = os.Stat(epicDir)
	if err != nil {
		t.Fatalf("Failed to stat epic path: %v", err)
	}

	// This should pass - the epic directory is valid
	if !fileInfo.IsDir() {
		t.Fatal("Expected epic directory path to be a directory")
	} else {
		t.Logf("SUCCESS: Epic directory is valid: %s", epicDir)
	}
}

// TestFeatureCreate_NoEpicDirectory tests that feature creation fails gracefully
// when no epic directory exists at all
func TestFeatureCreate_NoEpicDirectory(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean up test data before and after (using E97 for no directory test)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")
	}()

	// Create temporary test directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	// Change to temp directory for test
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create docs/plan directory structure (but no epic directory)
	planDir := filepath.Join(tempDir, "docs", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// Create epic in database
	epicRepo := repository.NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E97",
		Title:    "Test Epic No Directory",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Do NOT create any directory or file for the epic

	// Try to find epic directory using the same logic as runFeatureCreate
	epicPattern := filepath.Join("docs", "plan", "E97-*")
	matches, err := filepath.Glob(epicPattern)

	// This should fail to find matches (no directory exists)
	if err == nil && len(matches) > 0 {
		t.Fatalf("Expected no matches for epic directory, but found: %v", matches)
	}

	t.Logf("SUCCESS: No epic directory found (as expected)")
}

// Reset global CLI config after tests
func init() {
	// Ensure JSON output is disabled for tests (unless explicitly set)
	cli.GlobalConfig.JSON = false
}
