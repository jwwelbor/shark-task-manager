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

// TestEpicRepository_GetByKey_NumericFormat tests retrieval using numeric format (E04)
func TestEpicRepository_GetByKey_NumericFormat(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with slug
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Epic With Slug",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err, "Epic creation should succeed")
	require.NotNil(t, epic.Slug, "Slug should be generated")

	// Test: Retrieve using numeric key format (E04)
	retrieved, err := repo.GetByKey(ctx, epicKey)
	require.NoError(t, err, "Should retrieve epic using numeric key")
	require.NotNil(t, retrieved, "Retrieved epic should not be nil")
	assert.Equal(t, epicKey, retrieved.Key, "Key should match")
	assert.Equal(t, "Epic With Slug", retrieved.Title, "Title should match")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_GetByKey_SluggedFormat tests retrieval using slugged format (e04-epic-name)
func TestEpicRepository_GetByKey_SluggedFormat(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with slug
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Epic With Slug",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err, "Epic creation should succeed")
	require.NotNil(t, epic.Slug, "Slug should be generated")

	// Build slugged key format: e04-epic-with-slug (lowercase key + slug)
	sluggedKey := fmt.Sprintf("%s-%s", epicKey, *epic.Slug)

	// Test: Retrieve using slugged key format
	retrieved, err := repo.GetByKey(ctx, sluggedKey)
	require.NoError(t, err, "Should retrieve epic using slugged key")
	require.NotNil(t, retrieved, "Retrieved epic should not be nil")
	assert.Equal(t, epicKey, retrieved.Key, "Key should match original numeric key")
	assert.Equal(t, "Epic With Slug", retrieved.Title, "Title should match")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_GetByKey_InvalidSluggedFormat tests that invalid slugged keys fail gracefully
func TestEpicRepository_GetByKey_InvalidSluggedFormat(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Test: Try to retrieve with invalid slugged key (wrong slug)
	invalidKey := "E04-nonexistent-slug"
	retrieved, err := repo.GetByKey(ctx, invalidKey)
	assert.Error(t, err, "Should return error for invalid slugged key")
	assert.Nil(t, retrieved, "Retrieved epic should be nil")
}

// TestEpicRepository_GetByKey_PreferNumericLookup tests that numeric lookup is tried first
func TestEpicRepository_GetByKey_PreferNumericLookup(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewEpicRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with specific title/slug
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "My Epic Title",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	err := repo.Create(ctx, epic)
	require.NoError(t, err, "Epic creation should succeed")

	// Test with numeric key - should find it immediately
	retrieved, err := repo.GetByKey(ctx, epicKey)
	require.NoError(t, err, "Should retrieve epic with numeric key")
	assert.Equal(t, epicKey, retrieved.Key, "Key should match")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestContainsHyphen tests the containsHyphen helper function
func TestContainsHyphen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"No hyphen", "E04", false},
		{"With hyphen", "E04-epic-name", true},
		{"Multiple hyphens", "E04-epic-name-test", true},
		{"Empty string", "", false},
		{"Only hyphen", "-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHyphen(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSplitSluggedKey tests the splitSluggedKey helper function
func TestSplitSluggedKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Normal slugged key",
			input:    "E04-epic-name",
			expected: []string{"E04", "epic-name"},
		},
		{
			name:     "Multiple hyphens in slug",
			input:    "E04-epic-name-test",
			expected: []string{"E04", "epic-name-test"},
		},
		{
			name:     "No hyphen",
			input:    "E04",
			expected: []string{"E04"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSluggedKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEpicRepository_CalculateProgress_CompletedFeaturesCountAs100Percent tests that
// features with status="completed" contribute 100% progress regardless of task count/status
func TestEpicRepository_CalculateProgress_CompletedFeaturesCountAs100Percent(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Use unique epic key to avoid conflicts
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Test Epic for Progress Calculation",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)
	require.NotZero(t, epic.ID)

	// Create 7 features to simulate the E13 scenario
	// F01-F04: status="completed" (should each count as 100%)
	// F05-F06: status="draft", 0 tasks (should count as 0%)
	// F07: status="draft", would have tasks but we won't create them for this test

	completedFeatures := []string{"F01", "F02", "F03", "F04"}
	for _, fKey := range completedFeatures {
		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-%s", epicKey, fKey),
			Title:       fmt.Sprintf("Completed Feature %s", fKey),
			Status:      models.FeatureStatusCompleted,
			ProgressPct: 100.0, // Manually set to 100%
		}
		err := featureRepo.Create(ctx, feature)
		require.NoError(t, err, "Failed to create completed feature %s", fKey)
	}

	// Create 3 draft features with 0% progress
	draftFeatures := []string{"F05", "F06", "F07"}
	for _, fKey := range draftFeatures {
		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-%s", epicKey, fKey),
			Title:       fmt.Sprintf("Draft Feature %s", fKey),
			Status:      models.FeatureStatusDraft,
			ProgressPct: 0.0,
		}
		err := featureRepo.Create(ctx, feature)
		require.NoError(t, err, "Failed to create draft feature %s", fKey)
	}

	// Calculate epic progress
	progress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	require.NoError(t, err, "Failed to calculate epic progress")

	// Expected: 4 completed features / 7 total features = 57.14%
	// (4 * 100% + 3 * 0%) / 7 = 400 / 7 = 57.14%
	expectedProgress := (4.0 / 7.0) * 100.0
	assert.InDelta(t, expectedProgress, progress, 0.1,
		"Epic progress should be 57.14%% (4 completed features / 7 total features), got %.2f%%", progress)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_CalculateProgress_MixedFeatureStatuses tests progress calculation
// with a mix of completed and in-progress features
func TestEpicRepository_CalculateProgress_MixedFeatureStatuses(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Test Epic for Mixed Feature Statuses",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Feature 1: Completed (should count as 100% even with 0 tasks)
	f1 := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F01", epicKey),
		Title:       "Completed Feature with No Tasks",
		Status:      models.FeatureStatusCompleted,
		ProgressPct: 100.0,
	}
	err = featureRepo.Create(ctx, f1)
	require.NoError(t, err)

	// Feature 2: Active with 50% task completion (2 out of 4 tasks completed)
	f2 := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F02", epicKey),
		Title:       "Active Feature with 50% Progress",
		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	err = featureRepo.Create(ctx, f2)
	require.NoError(t, err)

	// Create 4 tasks for Feature 2 (2 completed, 2 todo)
	for i := 1; i <= 4; i++ {
		status := models.TaskStatusTodo
		if i <= 2 {
			status = models.TaskStatusCompleted
		}
		task := &models.Task{
			FeatureID: f2.ID,
			Key:       fmt.Sprintf("T-%s-F02-%03d", epicKey, i),
			Title:     fmt.Sprintf("Task %d", i),
			Status:    status,
			Priority:  5,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)
	}

	// Feature 3: Draft with 0% (no tasks)
	f3 := &models.Feature{
		EpicID:      epic.ID,
		Key:         fmt.Sprintf("%s-F03", epicKey),
		Title:       "Draft Feature",
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
	}
	err = featureRepo.Create(ctx, f3)
	require.NoError(t, err)

	// Calculate epic progress
	progress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	require.NoError(t, err)

	// Expected: (100% + 50% + 0%) / 3 = 50%
	expectedProgress := 50.0
	assert.InDelta(t, expectedProgress, progress, 0.1,
		"Epic progress should be 50%% (average of 100%%, 50%%, 0%%), got %.2f%%", progress)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_CalculateProgress_AllFeaturesCompleted tests that
// when all features are completed, epic shows 100% progress
func TestEpicRepository_CalculateProgress_AllFeaturesCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	epic := &models.Epic{
		Key:      epicKey,
		Title:    "Test Epic - All Features Completed",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create 3 completed features
	for i := 1; i <= 3; i++ {
		feature := &models.Feature{
			EpicID:      epic.ID,
			Key:         fmt.Sprintf("%s-F%02d", epicKey, i),
			Title:       fmt.Sprintf("Completed Feature %d", i),
			Status:      models.FeatureStatusCompleted,
			ProgressPct: 100.0,
		}
		err = featureRepo.Create(ctx, feature)
		require.NoError(t, err)
	}

	// Calculate epic progress
	progress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	require.NoError(t, err)

	// Expected: 100% (all features completed)
	assert.InDelta(t, 100.0, progress, 0.1,
		"Epic progress should be 100%% when all features are completed, got %.2f%%", progress)

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}
