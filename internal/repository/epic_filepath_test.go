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
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano())%1000/10)
	epicKey := fmt.Sprintf("E%s", suffix)

	// Clean up any existing data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

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

	// Set file path using UpdateFilePath
	customPath := "docs/roadmap/2025.md"
	err = repo.UpdateFilePath(ctx, epicKey, &customPath)
	require.NoError(t, err)

	// Test GetByFilePath with found epic
	found, err := repo.GetByFilePath(ctx, customPath)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, epicKey, found.Key)
	assert.Equal(t, customPath, *found.FilePath)
	assert.Equal(t, "Test Epic", found.Title)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
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

	// Create unique epic key using current nanoseconds to ensure uniqueness
	baseTime := time.Now().UnixNano()
	suffix := fmt.Sprintf("%010d", baseTime%10000000000)
	epicKey := fmt.Sprintf("E%s", suffix[:2])

	// Clean up any existing data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

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
	newPath := fmt.Sprintf("docs/epics/%d-roadmap.md", baseTime)
	err = repo.UpdateFilePath(ctx, epicKey, &newPath)
	assert.NoError(t, err)

	// Verify the update using GetByFilePath
	retrieved, err := repo.GetByFilePath(ctx, newPath)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, newPath, *retrieved.FilePath)
	assert.Equal(t, epicKey, retrieved.Key)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestEpicRepository_UpdateFilePath_Clear(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Create unique epic key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano())%1000/10)
	epicKey := fmt.Sprintf("E%s", suffix)

	// Clean up any existing data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

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

	// Set initial file path
	customPath := "docs/roadmap/2025.md"
	err = repo.UpdateFilePath(ctx, epicKey, &customPath)
	require.NoError(t, err)

	// Verify initial state
	retrieved, err := repo.GetByFilePath(ctx, customPath)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, customPath, *retrieved.FilePath)

	// Test UpdateFilePath with nil to clear the path
	err = repo.UpdateFilePath(ctx, epicKey, nil)
	assert.NoError(t, err)

	// Verify the path is cleared by trying to find it (should return nil)
	retrieved, err = repo.GetByFilePath(ctx, customPath)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
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
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano())%1000/10)
	epicKey1 := fmt.Sprintf("E%s", suffix)
	suffix2 := fmt.Sprintf("%02d", ((time.Now().UnixNano())%1000/10)+1)
	epicKey2 := fmt.Sprintf("E%s", suffix2)

	// Clean up any existing data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN (?, ?)", epicKey1, epicKey2)

	// Create two epics with same file path (simulating collision scenario)
	sharedPath := "docs/shared-epic-path.md"
	highPriority := models.PriorityHigh

	epic1 := &models.Epic{
		Key:           epicKey1,
		Title:         "Epic 1",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
	}
	err := repo.Create(ctx, epic1)
	require.NoError(t, err)

	// Set shared path on first epic
	err = repo.UpdateFilePath(ctx, epicKey1, &sharedPath)
	require.NoError(t, err)

	// Try to set the same path on another epic - should detect collision
	found, err := repo.GetByFilePath(ctx, sharedPath)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, epicKey1, found.Key)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN (?, ?)", epicKey1, epicKey2)
}
