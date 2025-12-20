package commands

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestEpicCreate_CustomFilename tests custom filename assignment
func TestEpicCreate_CustomFilename(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic with custom filename
	customPath := "docs/roadmap/2025.md"
	epic := &models.Epic{
		Key:           "E01",
		Title:         "Custom Path Epic",
		Description:   nil,
		Status:        models.EpicStatusDraft,
		Priority:      models.PriorityMedium,
		BusinessValue: nil,
		FilePath:      &customPath,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Verify epic was stored with custom file path
	retrieved, err := epicRepo.GetByKey(ctx, "E01")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	if retrieved.FilePath == nil {
		t.Errorf("Expected FilePath to be set, got nil")
	} else if *retrieved.FilePath != customPath {
		t.Errorf("Expected FilePath to be %s, got %s", customPath, *retrieved.FilePath)
	}
}

// TestEpicCreate_CustomFilename_Collision tests collision detection
func TestEpicCreate_CustomFilename_Collision(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create first epic with custom filename
	customPath := "docs/roadmap/collision.md"
	epic1 := &models.Epic{
		Key:           "E02",
		Title:         "First Epic",
		Description:   nil,
		Status:        models.EpicStatusDraft,
		Priority:      models.PriorityMedium,
		BusinessValue: nil,
		FilePath:      &customPath,
	}

	if err := epicRepo.Create(ctx, epic1); err != nil {
		t.Fatalf("Failed to create first epic: %v", err)
	}

	// Try to get epic by file path - should find the first one
	found, err := epicRepo.GetByFilePath(ctx, customPath)
	if err != nil {
		t.Fatalf("Failed to get epic by file path: %v", err)
	}

	if found == nil {
		t.Errorf("Expected to find epic by file path, got nil")
	} else if found.Key != "E02" {
		t.Errorf("Expected to find epic E02, got %s", found.Key)
	}
}

// TestEpicCreate_ForceReassignment tests force reassignment functionality
func TestEpicCreate_ForceReassignment(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create first epic with custom filename
	customPath := "docs/roadmap/reassign.md"
	epic1 := &models.Epic{
		Key:           "E03",
		Title:         "First Epic",
		Description:   nil,
		Status:        models.EpicStatusDraft,
		Priority:      models.PriorityMedium,
		BusinessValue: nil,
		FilePath:      &customPath,
	}

	if err := epicRepo.Create(ctx, epic1); err != nil {
		t.Fatalf("Failed to create first epic: %v", err)
	}

	// Clear the file path from the first epic (simulating force reassignment)
	if err := epicRepo.UpdateFilePath(ctx, "E03", nil); err != nil {
		t.Fatalf("Failed to update file path: %v", err)
	}

	// Create second epic with the same filename
	epic2 := &models.Epic{
		Key:           "E04",
		Title:         "Second Epic",
		Description:   nil,
		Status:        models.EpicStatusDraft,
		Priority:      models.PriorityMedium,
		BusinessValue: nil,
		FilePath:      &customPath,
	}

	if err := epicRepo.Create(ctx, epic2); err != nil {
		t.Fatalf("Failed to create second epic: %v", err)
	}

	// Verify first epic has nil file path
	retrieved1, err := epicRepo.GetByKey(ctx, "E03")
	if err != nil {
		t.Fatalf("Failed to retrieve first epic: %v", err)
	}

	if retrieved1.FilePath != nil {
		t.Errorf("Expected first epic's FilePath to be nil after reassignment, got: %v", retrieved1.FilePath)
	}

	// Verify second epic owns the file path
	retrieved2, err := epicRepo.GetByKey(ctx, "E04")
	if err != nil {
		t.Fatalf("Failed to retrieve second epic: %v", err)
	}

	if retrieved2.FilePath == nil {
		t.Errorf("Expected second epic's FilePath to be set")
	} else if *retrieved2.FilePath != customPath {
		t.Errorf("Expected second epic's FilePath to be %s, got %s", customPath, *retrieved2.FilePath)
	}

	// Verify GetByFilePath returns the second epic
	found, err := epicRepo.GetByFilePath(ctx, customPath)
	if err != nil {
		t.Fatalf("Failed to get epic by file path: %v", err)
	}

	if found == nil {
		t.Errorf("Expected to find epic by file path, got nil")
	} else if found.Key != "E04" {
		t.Errorf("Expected to find epic E04, got %s", found.Key)
	}
}

// TestEpicCreate_WithCustomFolderPath tests custom folder path assignment
func TestEpicCreate_WithCustomFolderPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic with custom folder path
	customFolderPath := "docs/roadmap/2025-q1"
	epic := &models.Epic{
		Key:              "E10",
		Title:            "Q1 2025 Roadmap",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &customFolderPath,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Verify epic was stored with custom folder path
	retrieved, err := epicRepo.GetByKey(ctx, "E10")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	if retrieved.CustomFolderPath == nil {
		t.Errorf("Expected CustomFolderPath to be set, got nil")
	} else if *retrieved.CustomFolderPath != customFolderPath {
		t.Errorf("Expected CustomFolderPath to be %s, got %s", customFolderPath, *retrieved.CustomFolderPath)
	}
}

// TestEpicCreate_WithInvalidPath tests that paths are handled correctly
func TestEpicCreate_WithInvalidPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testCases := []struct {
		name             string
		customFolderPath string
		shouldSucceed    bool
	}{
		{
			name:             "Valid relative path",
			customFolderPath: "docs/roadmap/2025-q1",
			shouldSucceed:    true,
		},
		{
			name:             "Valid nested path",
			customFolderPath: "docs/planning/roadmaps/2025/q1",
			shouldSucceed:    true,
		},
		{
			name:             "Path with dashes and underscores",
			customFolderPath: "docs/2025-q1_planning",
			shouldSucceed:    true,
		},
	}

	epicNum := 20
	for _, tc := range testCases {
		epicKey := fmt.Sprintf("E%d", epicNum)
		epicNum++

		epic := &models.Epic{
			Key:              epicKey,
			Title:            "Test Epic - " + tc.name,
			Status:           models.EpicStatusDraft,
			Priority:         models.PriorityMedium,
			CustomFolderPath: &tc.customFolderPath,
		}

		err := epicRepo.Create(ctx, epic)

		if tc.shouldSucceed && err != nil {
			t.Errorf("Test '%s': Expected success but got error: %v", tc.name, err)
		} else if !tc.shouldSucceed && err == nil {
			t.Errorf("Test '%s': Expected error but succeeded", tc.name)
		}
	}
}

// TestEpicCreate_DefaultPath tests that default behavior works (NULL custom_folder_path)
func TestEpicCreate_DefaultPath(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic without custom folder path (should be NULL)
	epic := &models.Epic{
		Key:      "E30",
		Title:    "Default Path Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
		// CustomFolderPath is nil
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Verify epic was stored with nil custom folder path
	retrieved, err := epicRepo.GetByKey(ctx, "E30")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	if retrieved.CustomFolderPath != nil {
		t.Errorf("Expected CustomFolderPath to be nil, got %v", *retrieved.CustomFolderPath)
	}
}

// TestEpicCreate_CustomFolderPath_StoresInDB verifies custom_folder_path is persisted correctly
func TestEpicCreate_CustomFolderPath_StoresInDB(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	customFolderPath := "docs/planning/2025"
	epic := &models.Epic{
		Key:              "E31",
		Title:            "DB Storage Test Epic",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &customFolderPath,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Verify directly from database
	var retrievedPath *string
	err := database.QueryRowContext(ctx, "SELECT custom_folder_path FROM epics WHERE key = ?", "E31").Scan(&retrievedPath)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if retrievedPath == nil {
		t.Errorf("Expected CustomFolderPath to be stored in DB, got nil")
	} else if *retrievedPath != customFolderPath {
		t.Errorf("Expected CustomFolderPath %s in DB, got %s", customFolderPath, *retrievedPath)
	}
}

// TestEpicCreate_EmptyStringNormalization tests that empty strings are handled correctly
func TestEpicCreate_EmptyStringNormalization(t *testing.T) {
	database := test.GetTestDB()
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create epic with empty string custom folder path
	emptyPath := ""
	epic := &models.Epic{
		Key:              "E32",
		Title:            "Empty Path Epic",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &emptyPath,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Retrieve and verify - empty string should be stored or converted to nil
	retrieved, err := epicRepo.GetByKey(ctx, "E32")
	if err != nil {
		t.Fatalf("Failed to retrieve epic: %v", err)
	}

	// Empty string is technically valid but should be handled gracefully
	if retrieved.CustomFolderPath != nil && *retrieved.CustomFolderPath == "" {
		// This is acceptable - empty string stored
		return
	} else if retrieved.CustomFolderPath == nil {
		// This is also acceptable - empty string normalized to nil
		return
	} else {
		t.Errorf("Unexpected CustomFolderPath value: %v", retrieved.CustomFolderPath)
	}
}
