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

func TestFeatureRepository_GetByFilePath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)
	featureKey := fmt.Sprintf("E%s-F01", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create parent epic
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create feature with custom file path
	customPath := "docs/plan/E01/F01/feature.md"
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       "Test Feature",
		Description: stringPtr("Test Description"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
		FilePath:    &customPath,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Test GetByFilePath with found feature
	found, err := featureRepo.GetByFilePath(ctx, customPath)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, featureKey, found.Key)
	assert.Equal(t, customPath, *found.FilePath)
	assert.Equal(t, "Test Feature", found.Title)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestFeatureRepository_GetByFilePath_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Test GetByFilePath with non-existent path
	found, err := featureRepo.GetByFilePath(ctx, "non/existent/feature/path.md")
	assert.NoError(t, err)
	assert.Nil(t, found) // Not found is not an error
}

func TestFeatureRepository_UpdateFilePath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)
	featureKey := fmt.Sprintf("E%s-F01", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create parent epic
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create feature without file path
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       "Test Feature",
		Description: stringPtr("Test Description"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Test UpdateFilePath with new path
	newPath := "docs/plan/E01/F01/custom-feature.md"
	err = featureRepo.UpdateFilePath(ctx, featureKey, &newPath)
	assert.NoError(t, err)

	// Verify the update
	retrieved, err := featureRepo.GetByKey(ctx, featureKey)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, newPath, *retrieved.FilePath)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestFeatureRepository_UpdateFilePath_Clear(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)
	featureKey := fmt.Sprintf("E%s-F01", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create parent epic
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create feature with file path
	customPath := "docs/plan/E01/F01/feature.md"
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       "Test Feature",
		Description: stringPtr("Test Description"),
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
		FilePath:    &customPath,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Verify initial state
	retrieved, err := featureRepo.GetByKey(ctx, featureKey)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.FilePath)
	assert.Equal(t, customPath, *retrieved.FilePath)

	// Test UpdateFilePath with nil to clear the path
	err = featureRepo.UpdateFilePath(ctx, featureKey, nil)
	assert.NoError(t, err)

	// Verify the path is cleared
	retrieved, err = featureRepo.GetByKey(ctx, featureKey)
	assert.NoError(t, err)
	assert.Nil(t, retrieved.FilePath)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}

func TestFeatureRepository_UpdateFilePath_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)

	// Test UpdateFilePath with non-existent feature
	newPath := "docs/plan/E01/F01/feature.md"
	err := featureRepo.UpdateFilePath(ctx, "E999-F999", &newPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feature not found")
}

func TestFeatureRepository_GetByFilePath_Collision_Detection(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create unique key using timestamp
	suffix := fmt.Sprintf("%02d", (time.Now().UnixNano()) % 1000 / 10)
	epicKey := fmt.Sprintf("E%s", suffix)
	featureKey1 := fmt.Sprintf("E%s-F01", suffix)
	featureKey2 := fmt.Sprintf("E%s-F02", suffix)

	// Clean up any existing data
	database.ExecContext(ctx, "DELETE FROM features WHERE key IN (?, ?)", featureKey1, featureKey2)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create parent epic
	highPriority := models.PriorityHigh
	epic := &models.Epic{
		Key:           epicKey,
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityMedium,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create first feature with shared path
	sharedPath := "docs/plan/E01/shared-feature.md"
	feature1 := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey1,
		Title:       "Feature 1",
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
		FilePath:    &sharedPath,
	}
	err = featureRepo.Create(ctx, feature1)
	require.NoError(t, err)

	// Check collision detection - should find the existing feature
	found, err := featureRepo.GetByFilePath(ctx, sharedPath)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, featureKey1, found.Key)

	// Cleanup
	database.ExecContext(ctx, "DELETE FROM features WHERE key IN (?, ?)", featureKey1, featureKey2)
	database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)
}
