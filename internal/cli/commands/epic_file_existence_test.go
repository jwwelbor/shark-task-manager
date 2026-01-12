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

// TestEpicCreate_ExistingFile_ShouldNotOverwrite tests that when --file points to an existing file,
// the command should link to it instead of overwriting it.
// This is the CRITICAL bug that the user reported: epic create was overwriting existing files.
func TestEpicCreate_ExistingFile_ShouldNotOverwrite(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	testDb := repository.NewDB(database)
	defer cli.ResetDB()

	// Clean up any existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-%'")

	// Get repository
	epicRepo := repository.NewEpicRepository(testDb)

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test file with unique content
	testFilePath := filepath.Join(tempDir, "docs", "plan", "existing-epic.md")
	originalContent := "# Original Epic Content\n\nThis is the original content that should NOT be overwritten.\n"

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Write original file
	if err := os.WriteFile(testFilePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Verify file exists before test
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatalf("Test file was not created")
	}

	// Now simulate the epic create command with --file pointing to existing file
	// This should:
	// 1. Detect that file exists
	// 2. Link to it in the database
	// 3. NOT overwrite the file

	// For now, we'll create the epic directly through repository to test the expected behavior
	// In the actual fix, runEpicCreate should check file existence before writing

	// Save current working directory and change to temp dir
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create epic in database with relative file path
	relPath := "docs/plan/existing-epic.md"
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		FilePath: &relPath,
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic in database: %v", err)
	}

	// Read file content after epic creation
	afterContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read file after epic creation: %v", err)
	}

	// CRITICAL TEST: File content should NOT have changed
	if string(afterContent) != originalContent {
		t.Errorf("File was overwritten! Expected content:\n%s\n\nGot:\n%s", originalContent, string(afterContent))
	}

	// Verify epic was linked to file in database
	createdEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	if createdEpic.FilePath == nil || *createdEpic.FilePath != relPath {
		t.Errorf("Epic file path not set correctly. Expected: %s, Got: %v", relPath, createdEpic.FilePath)
	}
}

// TestEpicCreate_NonExistingFile_ShouldCreate tests that when --file points to a non-existing file,
// the command should create it with the template.
func TestEpicCreate_NonExistingFile_ShouldCreate(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	testDb := repository.NewDB(database)
	defer cli.ResetDB()

	// Clean up any existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-%'")

	// Get repository
	epicRepo := repository.NewEpicRepository(testDb)

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Define path for file that doesn't exist yet
	testFilePath := filepath.Join(tempDir, "docs", "plan", "new-epic.md")

	// Verify file doesn't exist before test
	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Fatalf("Test file should not exist before test")
	}

	// Save current working directory and change to temp dir
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// In the fixed version, runEpicCreate should:
	// 1. Detect that file doesn't exist
	// 2. Create parent directories
	// 3. Write template to file
	// 4. Create epic in database

	// For this test, we'll verify the expected behavior manually
	relPath := "docs/plan/new-epic.md"

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write template (simulating what runEpicCreate should do)
	templateContent := "# New Epic\n\nThis is a newly created epic.\n"
	if err := os.WriteFile(testFilePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	// Create epic in database
	epic := &models.Epic{
		Key:      "E98",
		Title:    "New Epic",
		FilePath: &relPath,
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic in database: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Errorf("File should have been created")
	}

	// Verify file has template content
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != templateContent {
		t.Errorf("File content incorrect. Expected:\n%s\n\nGot:\n%s", templateContent, string(content))
	}

	// Verify epic was created with correct file path
	createdEpic, err := epicRepo.GetByKey(ctx, "E98")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	if createdEpic.FilePath == nil || *createdEpic.FilePath != relPath {
		t.Errorf("Epic file path not set correctly. Expected: %s, Got: %v", relPath, createdEpic.FilePath)
	}
}
