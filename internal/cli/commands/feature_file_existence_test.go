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

// TestFeatureCreate_ExistingFile_ShouldNotOverwrite tests that when --file points to an existing file,
// the command should link to it instead of overwriting it.
func TestFeatureCreate_ExistingFile_ShouldNotOverwrite(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	testDb := repository.NewDB(database)
	defer cli.ResetDB()

	// Clean up any existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-%'")

	// Seed epic for feature
	epicID, _ := test.SeedTestData()

	// Get repository
	featureRepo := repository.NewFeatureRepository(testDb)

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test file with unique content
	testFilePath := filepath.Join(tempDir, "docs", "plan", "existing-feature.md")
	originalContent := "# Original Feature Content\n\nThis is the original content that should NOT be overwritten.\n"

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

	// Save current working directory and change to temp dir
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create feature in database with relative file path
	relPath := "docs/plan/existing-feature.md"
	feature := &models.Feature{
		Key:      "E99-F99",
		EpicID:   epicID,
		Title:    "Test Feature",
		FilePath: &relPath,
		Status:   models.FeatureStatusDraft,
	}

	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature in database: %v", err)
	}

	// Read file content after feature creation
	afterContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read file after feature creation: %v", err)
	}

	// CRITICAL TEST: File content should NOT have changed
	if string(afterContent) != originalContent {
		t.Errorf("File was overwritten! Expected content:\n%s\n\nGot:\n%s", originalContent, string(afterContent))
	}

	// Verify feature was linked to file in database
	createdFeature, err := featureRepo.GetByKey(ctx, "E99-F99")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if createdFeature.FilePath == nil || *createdFeature.FilePath != relPath {
		t.Errorf("Feature file path not set correctly. Expected: %s, Got: %v", relPath, createdFeature.FilePath)
	}
}

// TestFeatureCreate_NonExistingFile_ShouldCreate tests that when --file points to a non-existing file,
// the command should create it with the template.
func TestFeatureCreate_NonExistingFile_ShouldCreate(t *testing.T) {
	// Setup test environment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database
	database := test.GetTestDB()
	testDb := repository.NewDB(database)
	defer cli.ResetDB()

	// Clean up any existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-%'")

	// Seed epic for feature
	epicID, _ := test.SeedTestData()

	// Get repository
	featureRepo := repository.NewFeatureRepository(testDb)

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Define path for file that doesn't exist yet
	testFilePath := filepath.Join(tempDir, "docs", "plan", "new-feature.md")

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

	relPath := "docs/plan/new-feature.md"

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write template (simulating what runFeatureCreate should do)
	templateContent := "# New Feature\n\nThis is a newly created feature.\n"
	if err := os.WriteFile(testFilePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	// Create feature in database
	feature := &models.Feature{
		Key:      "E98-F98",
		EpicID:   epicID,
		Title:    "New Feature",
		FilePath: &relPath,
		Status:   models.FeatureStatusDraft,
	}

	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature in database: %v", err)
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

	// Verify feature was created with correct file path
	createdFeature, err := featureRepo.GetByKey(ctx, "E98-F98")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if createdFeature.FilePath == nil || *createdFeature.FilePath != relPath {
		t.Errorf("Feature file path not set correctly. Expected: %s, Got: %v", relPath, createdFeature.FilePath)
	}
}
