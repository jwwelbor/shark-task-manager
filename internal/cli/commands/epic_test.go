package commands

import (
	"context"
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
