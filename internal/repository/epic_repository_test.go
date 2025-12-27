package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEpicRepository_UpdateCustomFolderPath tests updating the custom folder path of an epic
func TestEpicRepository_UpdateCustomFolderPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create an epic with a custom folder path
	initialPath := "docs/initial"
	epic := &models.Epic{
		Key:              "E99",
		Title:            "Test Epic for Path Update",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &initialPath,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err)
	require.NotZero(t, epic.ID)

	// Verify initial path was saved
	retrieved, err := repo.GetByKey(ctx, "E99")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.NotNil(t, retrieved.CustomFolderPath, "CustomFolderPath should not be nil after GetByKey")
	assert.Equal(t, "docs/initial", *retrieved.CustomFolderPath, "CustomFolderPath should match initial value")

	// Update the custom folder path
	newPath := "docs/updated"
	retrieved.CustomFolderPath = &newPath
	err = repo.Update(ctx, retrieved)
	require.NoError(t, err)

	// Verify the path was updated
	updated, err := repo.GetByKey(ctx, "E99")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.NotNil(t, updated.CustomFolderPath, "CustomFolderPath should not be nil after update")
	assert.Equal(t, "docs/updated", *updated.CustomFolderPath, "CustomFolderPath should be updated to new value")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_UpdatePreservesCustomFolderPath tests that updating other fields doesn't clear custom_folder_path
func TestEpicRepository_UpdatePreservesCustomFolderPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create an epic with a custom folder path
	initialPath := "docs/preserve-test"
	epic := &models.Epic{
		Key:              "E98",
		Title:            "Test Epic for Path Preservation",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &initialPath,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err)
	require.NotZero(t, epic.ID)

	// Get the epic
	retrieved, err := repo.GetByKey(ctx, "E98")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Update just the title (not the path)
	retrieved.Title = "Updated Title"
	err = repo.Update(ctx, retrieved)
	require.NoError(t, err)

	// Verify the path was preserved
	updated, err := repo.GetByKey(ctx, "E98")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated Title", updated.Title, "Title should be updated")
	assert.NotNil(t, updated.CustomFolderPath, "CustomFolderPath should still be set after title update")
	assert.Equal(t, "docs/preserve-test", *updated.CustomFolderPath, "CustomFolderPath should be preserved")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}
