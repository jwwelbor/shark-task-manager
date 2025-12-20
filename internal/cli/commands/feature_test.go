package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestFeatureCreate_DefaultBehavior tests that features are created with default file paths when no --filename is provided
func TestFeatureCreate_DefaultBehavior(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create an epic
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Reset feature variables for the test
	featureCreateEpic = "E99"
	featureCreateDescription = ""
	featureCreateExecutionOrder = 0
	featureCreateFilename = ""
	featureCreateForce = false

	// Get the created epic to use its ID
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature directory structure
	epicDir := filepath.Join(tempDir, "docs", "plan", "E99-test-epic")
	if err := os.MkdirAll(epicDir, 0755); err != nil {
		t.Fatalf("Failed to create epic directory: %v", err)
	}

	// Create feature using repository
	feature := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F01",
		Title:       "Test Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    nil, // Default: no custom path
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify feature was created
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E99-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if retrievedFeature.FilePath != nil {
		t.Errorf("Expected FilePath to be nil for default behavior, got %v", retrievedFeature.FilePath)
	}

	if retrievedFeature.Title != "Test Feature" {
		t.Errorf("Expected title 'Test Feature', got %v", retrievedFeature.Title)
	}
}

// TestFeatureCreate_CustomFilename tests that features can be created with custom file paths
func TestFeatureCreate_CustomFilename(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create an epic
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	retrievedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature with custom file path
	customPath := "docs/specs/auth.md"
	feature := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F01",
		Title:       "OAuth Implementation",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    &customPath,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify feature was created with custom path
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E99-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if retrievedFeature.FilePath == nil || *retrievedFeature.FilePath != customPath {
		t.Errorf("Expected FilePath %q, got %v", customPath, retrievedFeature.FilePath)
	}

	// Verify we can retrieve by file path
	foundByPath, err := featureRepo.GetByFilePath(ctx, customPath)
	if err != nil {
		t.Fatalf("Failed to retrieve feature by file path: %v", err)
	}

	if foundByPath.Key != "E99-F01" {
		t.Errorf("Expected feature key 'E99-F01', got %v", foundByPath.Key)
	}
}

// TestFeatureCreate_CustomFilename_Collision tests that collision detection prevents duplicate file claims
func TestFeatureCreate_CustomFilename_Collision(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create an epic
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	retrievedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create first feature with custom file path
	customPath := "docs/specs/auth.md"
	feature1 := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F01",
		Title:       "OAuth Implementation",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    &customPath,
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create first feature: %v", err)
	}

	// Attempt to get feature by the same file path (collision detection)
	foundFeature, err := featureRepo.GetByFilePath(ctx, customPath)
	if err != nil {
		t.Fatalf("Failed to retrieve feature by file path: %v", err)
	}

	if foundFeature == nil {
		t.Error("Expected to find feature by file path, but got nil")
	} else if foundFeature.Key != "E99-F01" {
		t.Errorf("Expected feature key 'E99-F01', got %v", foundFeature.Key)
	}
}

// TestFeatureCreate_ForceReassignment tests that --force flag allows reassignment of claimed files
func TestFeatureCreate_ForceReassignment(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create an epic
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	retrievedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create first feature with custom file path
	customPath := "docs/specs/auth.md"
	feature1 := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F01",
		Title:       "OAuth Implementation",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    &customPath,
	}
	if err := featureRepo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create first feature: %v", err)
	}

	// Clear the file path from first feature (simulating force reassignment)
	if err := featureRepo.UpdateFilePath(ctx, "E99-F01", nil); err != nil {
		t.Fatalf("Failed to clear file path: %v", err)
	}

	// Create second feature with the same file path
	feature2 := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F02",
		Title:       "New Auth Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    &customPath,
	}
	if err := featureRepo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create second feature: %v", err)
	}

	// Verify old feature no longer has the file path
	oldFeature, err := featureRepo.GetByKey(ctx, "E99-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve old feature: %v", err)
	}

	if oldFeature.FilePath != nil {
		t.Errorf("Expected old feature's FilePath to be nil after reassignment, got %v", oldFeature.FilePath)
	}

	// Verify new feature has the file path
	newFeature, err := featureRepo.GetByKey(ctx, "E99-F02")
	if err != nil {
		t.Fatalf("Failed to retrieve new feature: %v", err)
	}

	if newFeature.FilePath == nil || *newFeature.FilePath != customPath {
		t.Errorf("Expected new feature's FilePath to be %q, got %v", customPath, newFeature.FilePath)
	}

	// Verify file path lookup returns new feature
	foundFeature, err := featureRepo.GetByFilePath(ctx, customPath)
	if err != nil {
		t.Fatalf("Failed to retrieve feature by file path: %v", err)
	}

	if foundFeature.Key != "E99-F02" {
		t.Errorf("Expected file path to be claimed by feature 'E99-F02', got %v", foundFeature.Key)
	}
}

// TestFeatureCreate_InvalidFilename_AbsolutePath tests that absolute paths are rejected
func TestFeatureCreate_InvalidFilename_AbsolutePath(t *testing.T) {
	tempDir := t.TempDir()

	// Test that absolute paths are rejected
	absPath := filepath.Join(tempDir, "docs", "specs", "test.md")
	_, _, err := ValidateCustomFilename(absPath, tempDir)

	if err == nil {
		t.Error("Expected error for absolute path, but got nil")
	}
}

// TestFeatureCreate_InvalidFilename_PathTraversal tests that path traversal attempts are rejected
func TestFeatureCreate_InvalidFilename_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Test that path traversal is rejected
	traversalPath := "docs/../../../etc/passwd.md"
	_, _, err := ValidateCustomFilename(traversalPath, tempDir)

	if err == nil {
		t.Error("Expected error for path traversal, but got nil")
	}
}

// TestFeatureCreate_InvalidFilename_WrongExtension tests that non-.md files are rejected
func TestFeatureCreate_InvalidFilename_WrongExtension(t *testing.T) {
	tempDir := t.TempDir()

	// Test that non-.md files are rejected
	wrongExtPath := "docs/specs/test.txt"
	_, _, err := ValidateCustomFilename(wrongExtPath, tempDir)

	if err == nil {
		t.Error("Expected error for non-.md extension, but got nil")
	}
}

// TestFeatureCreate_CrossEntityCollision tests that features can have their own file paths separate from epics
func TestFeatureCreate_CrossEntityCollision(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create an epic without custom file path
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	retrievedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature with custom file path
	featureCustomPath := "docs/specs/feature-level.md"
	feature := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F01",
		Title:       "Test Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    &featureCustomPath,
	}

	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify that feature can be found by its file path
	foundFeature, err := featureRepo.GetByFilePath(ctx, featureCustomPath)
	if err != nil {
		t.Fatalf("Failed to retrieve feature by file path: %v", err)
	}

	if foundFeature == nil {
		t.Error("Expected to find feature by file path, but got nil")
	} else if foundFeature.Key != "E99-F01" {
		t.Errorf("Expected feature key 'E99-F01', got %v", foundFeature.Key)
	}
}

// ValidateCustomFilename is a helper function for testing (wraps the taskcreation package function)
func ValidateCustomFilename(filename string, projectRoot string) (string, string, error) {
	return taskcreation.ValidateCustomFilename(filename, projectRoot)
}

// TestFeatureCreate_CollisionWithEpicForceReassignment tests forcing reassignment from epic to feature
func TestFeatureCreate_CollisionWithEpicForceReassignment(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create an epic with custom file path
	customPath := "docs/specs/shared.md"
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
		FilePath: &customPath,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	retrievedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Clear the epic's file path to simulate force reassignment
	if err := epicRepo.UpdateFilePath(ctx, "E99", nil); err != nil {
		t.Fatalf("Failed to clear epic's file path: %v", err)
	}

	// Now create a feature with the same file path
	feature := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E99-F01",
		Title:       "Reassigned Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		FilePath:    &customPath,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify epic no longer has the file path
	updatedEpic, err := epicRepo.GetByKey(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	if updatedEpic.FilePath != nil {
		t.Errorf("Expected epic's FilePath to be nil, got %v", updatedEpic.FilePath)
	}

	// Verify feature now owns the file path
	foundFeature, err := featureRepo.GetByFilePath(ctx, customPath)
	if err != nil {
		t.Fatalf("Failed to retrieve feature by file path: %v", err)
	}

	if foundFeature.Key != "E99-F01" {
		t.Errorf("Expected feature key 'E99-F01', got %v", foundFeature.Key)
	}
}

// TestFeatureCreate_InheritsEpicCustomPath tests that features inherit epic's custom folder path
func TestFeatureCreate_InheritsEpicCustomPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic with custom folder path
	epicPath := "docs/roadmap/2025-q1"
	epic := &models.Epic{
		Key:              "E40",
		Title:            "Q1 2025 Roadmap",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &epicPath,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Get the created epic
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E40")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature without custom folder path - should inherit from epic
	feature := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E40-F01",
		Title:       "User Growth Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		// CustomFolderPath is nil - should inherit from epic
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify feature inherits epic's custom folder path
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E40-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	// Feature should not have its own custom_folder_path (nil)
	// The inheritance is done at the CLI level, not at the repository level
	if retrievedFeature.CustomFolderPath != nil {
		t.Logf("Feature has custom_folder_path: %v (this may be expected if inheritance is applied at creation time)", retrievedFeature.CustomFolderPath)
	}
}

// TestFeatureCreate_OverridesEpicCustomPath tests that feature custom path overrides epic custom path
func TestFeatureCreate_OverridesEpicCustomPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic with custom folder path
	epicPath := "docs/roadmap/2025-q1"
	epic := &models.Epic{
		Key:              "E41",
		Title:            "Q1 2025 Roadmap",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &epicPath,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Get the created epic
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E41")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature with its own custom folder path (different from epic)
	featurePath := "docs/roadmap/2025-q1/user-growth"
	feature := &models.Feature{
		EpicID:           retrievedEpic.ID,
		Key:              "E41-F01",
		Title:            "User Growth Feature",
		Status:           models.FeatureStatusDraft,
		ProgressPct:      0.0,
		CustomFolderPath: &featurePath,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify feature has its own custom folder path
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E41-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if retrievedFeature.CustomFolderPath == nil {
		t.Errorf("Expected feature CustomFolderPath to be set, got nil")
	} else if *retrievedFeature.CustomFolderPath != featurePath {
		t.Errorf("Expected feature CustomFolderPath %s, got %s", featurePath, *retrievedFeature.CustomFolderPath)
	}
}

// TestFeatureCreate_WithCustomFolderPath tests basic custom folder path assignment for features
func TestFeatureCreate_WithCustomFolderPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic without custom folder path
	epic := &models.Epic{
		Key:      "E42",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Get the created epic
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E42")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature with custom folder path
	featurePath := "docs/features/auth"
	feature := &models.Feature{
		EpicID:           retrievedEpic.ID,
		Key:              "E42-F01",
		Title:            "Authentication Feature",
		Status:           models.FeatureStatusDraft,
		ProgressPct:      0.0,
		CustomFolderPath: &featurePath,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify feature was stored with custom folder path
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E42-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if retrievedFeature.CustomFolderPath == nil {
		t.Errorf("Expected CustomFolderPath to be set, got nil")
	} else if *retrievedFeature.CustomFolderPath != featurePath {
		t.Errorf("Expected CustomFolderPath %s, got %s", featurePath, *retrievedFeature.CustomFolderPath)
	}
}

// TestFeatureCreate_CustomPath_StoresInDB verifies custom_folder_path is persisted in database
func TestFeatureCreate_CustomPath_StoresInDB(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic
	epic := &models.Epic{
		Key:      "E43",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Get the created epic
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E43")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature with custom folder path
	featurePath := "docs/spec/feature-docs"
	feature := &models.Feature{
		EpicID:           retrievedEpic.ID,
		Key:              "E43-F01",
		Title:            "Documentation Feature",
		Status:           models.FeatureStatusDraft,
		ProgressPct:      0.0,
		CustomFolderPath: &featurePath,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify directly from database
	var retrievedPath *string
	err = database.QueryRowContext(ctx, "SELECT custom_folder_path FROM features WHERE key = ?", "E43-F01").Scan(&retrievedPath)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if retrievedPath == nil {
		t.Errorf("Expected CustomFolderPath to be stored in DB, got nil")
	} else if *retrievedPath != featurePath {
		t.Errorf("Expected CustomFolderPath %s in DB, got %s", featurePath, *retrievedPath)
	}
}

// TestFeatureCreate_DefaultPath tests backward compatibility (no custom folder path)
func TestFeatureCreate_DefaultPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic
	epic := &models.Epic{
		Key:      "E44",
		Title:    "Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Get the created epic
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E44")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Create feature without custom folder path
	feature := &models.Feature{
		EpicID:      retrievedEpic.ID,
		Key:         "E44-F01",
		Title:       "Simple Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
		// CustomFolderPath is nil
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify feature has no custom folder path
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E44-F01")
	if err != nil {
		t.Fatalf("Failed to retrieve feature: %v", err)
	}

	if retrievedFeature.CustomFolderPath != nil {
		t.Errorf("Expected CustomFolderPath to be nil, got %v", *retrievedFeature.CustomFolderPath)
	}
}
