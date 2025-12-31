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

// TestEpicRepository_UpdateCustomFolderPath tests updating the custom folder path of an epic
func TestEpicRepository_UpdateCustomFolderPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Use unique epic key to avoid parallel test conflicts
	// Epic keys must match ^E\d{2}$ format, so use E10-E99 range with timestamp
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create an epic with a custom folder path
	initialPath := "docs/initial"
	epic := &models.Epic{
		Key:              epicKey,
		Title:            "Test Epic for Path Update",
		Status:           models.EpicStatusDraft,
		Priority:         models.PriorityMedium,
		CustomFolderPath: &initialPath,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err)
	require.NotZero(t, epic.ID)

	// Verify initial path was saved
	retrieved, err := repo.GetByKey(ctx, epicKey)
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
	updated, err := repo.GetByKey(ctx, epicKey)
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

// TestEpicRepository_Create_GeneratesAndStoresSlug tests that epic creation generates and stores slug
func TestEpicRepository_Create_GeneratesAndStoresSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Use unique epic key to avoid parallel test conflicts
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create an epic with a title that should generate a slug
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Test Epic With Spaces",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err, "Epic creation should succeed")
	require.NotZero(t, epic.ID, "Epic ID should be set after creation")

	// Verify slug was generated and populated in the epic object
	assert.NotNil(t, epic.Slug, "Slug should be generated and set in epic object")
	assert.Equal(t, "test-epic-with-spaces", *epic.Slug, "Slug should be generated from title")

	// Verify slug was stored in database by retrieving the epic
	retrieved, err := repo.GetByKey(ctx, epicKey)
	require.NoError(t, err, "Should retrieve epic from database")
	require.NotNil(t, retrieved, "Retrieved epic should not be nil")
	assert.NotNil(t, retrieved.Slug, "Slug should be stored in database")
	assert.Equal(t, "test-epic-with-spaces", *retrieved.Slug, "Stored slug should match generated slug")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_Create_SlugHandlesSpecialCharacters tests slug generation with special characters
func TestEpicRepository_Create_SlugHandlesSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with title containing special characters
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Fix Bug: API Endpoint (v2)",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityHigh,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err, "Epic creation should succeed")

	// Verify slug handles special characters correctly
	assert.NotNil(t, epic.Slug, "Slug should be generated")
	assert.Equal(t, "fix-bug-api-endpoint-v2", *epic.Slug, "Slug should remove special characters")

	// Verify in database
	retrieved, err := repo.GetByKey(ctx, epicKey)
	require.NoError(t, err)
	assert.Equal(t, "fix-bug-api-endpoint-v2", *retrieved.Slug, "Slug in DB should match")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}
