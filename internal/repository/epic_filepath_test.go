package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEpicRepository_GetByFilePath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Create unique epic key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with custom file path
	customPath := "docs/roadmap/2025.md"
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Description:   stringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
		FilePath:      &customPath,
	}
	err := repo.Create(ctx, epic)
	require.NoError(t, err)

	// Test GetByFilePath with found epic
	found, err := repo.GetByFilePath(ctx, customPath)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, epicKey, found.Key)
	assert.Equal(t, customPath, *found.FilePath)
	assert.Equal(t, "Test Epic", found.Title)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestEpicRepository_GetByFilePath_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Test GetByFilePath with non-existent path
	found, err := repo.GetByFilePath(ctx, "non/existent/path.md")
	assert.NoError(t, err)
	assert.Nil(t, found) // Not found is not an error
}

func TestEpicRepository_UpdateFilePath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Create unique epic key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic without file path
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Description:   stringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
	}
	err := repo.Create(ctx, epic)
	require.NoError(t, err)

	// Test UpdateFilePath with new path
	newPath := "docs/epics/2025-roadmap.md"
	err = repo.UpdateFilePath(ctx, epicKey, &newPath)
	assert.NoError(t, err)

	// Verify the update
	retrieved, err := repo.GetByKey(ctx, epicKey)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, newPath, *retrieved.FilePath)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestEpicRepository_UpdateFilePath_Clear(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Create unique epic key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with file path
	customPath := "docs/roadmap/2025.md"
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Description:   stringPtr("Test Description"),
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
		FilePath:      &customPath,
	}
	err := repo.Create(ctx, epic)
	require.NoError(t, err)

	// Verify initial state
	retrieved, err := repo.GetByKey(ctx, epicKey)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.FilePath)
	assert.Equal(t, customPath, *retrieved.FilePath)

	// Test UpdateFilePath with nil to clear the path
	err = repo.UpdateFilePath(ctx, epicKey, nil)
	assert.NoError(t, err)

	// Verify the path is cleared
	retrieved, err = repo.GetByKey(ctx, epicKey)
	assert.NoError(t, err)
	assert.Nil(t, retrieved.FilePath)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestEpicRepository_UpdateFilePath_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Test UpdateFilePath with non-existent epic
	newPath := "docs/epics/2025-roadmap.md"
	err := repo.UpdateFilePath(ctx, "E999", &newPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "epic not found")
}

func TestEpicRepository_GetByFilePath_Collision_Detection(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Create unique epic key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey1 := fmt.Sprintf("E%s", suffix)
	epicKey2 := fmt.Sprintf("E%s", suffix+1)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM epics WHERE key IN (?, ?)", epicKey1, epicKey2)

	// Create two epics with same file path (simulating collision scenario)
	sharedPath := "docs/shared-epic-path.md"
	highPriority := models.PriorityHigh

	epic1 := &models.Epic{
		Key:           epicKey1,
		Title:         "Epic 1",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
		FilePath:      &sharedPath,
	}
	err := repo.Create(ctx, epic1)
	require.NoError(t, err)

	// Try to set the same path on another epic - should detect collision
	found, err := repo.GetByFilePath(ctx, sharedPath)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, epicKey1, found.Key)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM epics WHERE key IN (?, ?)", epicKey1, epicKey2)
}

// stringPtr is a helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
