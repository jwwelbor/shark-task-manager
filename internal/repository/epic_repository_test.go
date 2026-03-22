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

// TestEpicRepository_UpdateCustomFolderPath removed - custom_folder_path feature no longer supported

// TestEpicRepository_UpdatePreservesCustomFolderPath removed - custom_folder_path feature no longer supported

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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic With Spaces"}, Status: models.EpicStatusDraft,
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Fix Bug: API Endpoint (v2)"}, Status: models.EpicStatusDraft,
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Epic With Slug"}, Status: models.EpicStatusDraft,
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Epic With Slug"}, Status: models.EpicStatusDraft,
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "My Epic Title"}, Status: models.EpicStatusDraft,
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

// ============================================================================
// Status Rollup Tests (E07-F23)
// ============================================================================

// TestEpicRepository_GetFeatureStatusRollup_WithMultipleFeatures tests feature status aggregation
func TestEpicRepository_GetFeatureStatusRollup_WithMultipleFeatures(t *testing.T) {
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic - Feature Rollup"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create features with different statuses
	statusCounts := map[models.FeatureStatus]int{
		models.FeatureStatusActive:    2,
		models.FeatureStatusCompleted: 3,
		models.FeatureStatusDraft:     1,
	}

	i := 1
	for status, count := range statusCounts {
		for j := 0; j < count; j++ {
			feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F%02d", epicKey, i),
				Title: fmt.Sprintf("Feature %s %d", status, i)}, EpicID: epic.ID,

				Status:      status,
				ProgressPct: 50.0,
			}
			err = featureRepo.Create(ctx, feature)
			require.NoError(t, err)
			i++
		}
	}

	// Get feature status rollup
	rollup, err := epicRepo.GetFeatureStatusRollup(ctx, epic.ID)
	require.NoError(t, err)

	// Verify counts
	assert.NotNil(t, rollup, "Rollup should not be nil")
	assert.Equal(t, 2, rollup[string(models.FeatureStatusActive)],
		"Should have 2 active features")
	assert.Equal(t, 3, rollup[string(models.FeatureStatusCompleted)],
		"Should have 3 completed features")
	assert.Equal(t, 1, rollup[string(models.FeatureStatusDraft)],
		"Should have 1 draft feature")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_GetFeatureStatusRollup_EmptyEpic tests empty epic returns empty map
func TestEpicRepository_GetFeatureStatusRollup_EmptyEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic - Empty"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Get feature status rollup (should be empty)
	rollup, err := epicRepo.GetFeatureStatusRollup(ctx, epic.ID)
	require.NoError(t, err)

	// Verify empty map
	assert.NotNil(t, rollup, "Rollup should not be nil")
	assert.Equal(t, 0, len(rollup), "Rollup should be empty for epic with no features")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_GetTaskStatusRollup_WithMultipleTasks tests task status aggregation
func TestEpicRepository_GetTaskStatusRollup_WithMultipleTasks(t *testing.T) {
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic - Task Rollup"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create feature
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F01", epicKey),
		Title: "Test Feature"}, EpicID: epic.ID,

		Status:      models.FeatureStatusActive,
		ProgressPct: 50.0,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Create tasks with different statuses
	statusCounts := map[models.TaskStatus]int{
		models.TaskStatus("todo"):        3,
		models.TaskStatus("in_progress"): 2,
		models.TaskStatus("completed"):   4,
		models.TaskStatus("blocked"):     1,
	}

	i := 1
	for status, count := range statusCounts {
		for j := 0; j < count; j++ {
			task := &models.Task{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("T-%s-%03d", feature.Key, i),
				Title: fmt.Sprintf("Task %s %d", status, i)}, FeatureID: feature.ID,

				Status:   status,
				Priority: 5,
			}
			err = taskRepo.Create(ctx, task)
			require.NoError(t, err)
			i++
		}
	}

	// Get task status rollup
	rollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	require.NoError(t, err)

	// Verify counts
	assert.NotNil(t, rollup, "Rollup should not be nil")
	assert.Equal(t, 3, rollup[string(models.TaskStatus("todo"))],
		"Should have 3 todo tasks")
	assert.Equal(t, 2, rollup[string(models.TaskStatus("in_progress"))],
		"Should have 2 in_progress tasks")
	assert.Equal(t, 4, rollup[string(models.TaskStatus("completed"))],
		"Should have 4 completed tasks")
	assert.Equal(t, 1, rollup[string(models.TaskStatus("blocked"))],
		"Should have 1 blocked task")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_GetTaskStatusRollup_MultipleFeatures tests task rollup across multiple features
func TestEpicRepository_GetTaskStatusRollup_MultipleFeatures(t *testing.T) {
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic - Multi-Feature Task Rollup"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create 2 features
	var features []*models.Feature
	for f := 1; f <= 2; f++ {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F%02d", epicKey, f),
			Title: fmt.Sprintf("Feature %d", f)}, EpicID: epic.ID,

			Status:      models.FeatureStatusActive,
			ProgressPct: 50.0,
		}
		err = featureRepo.Create(ctx, feature)
		require.NoError(t, err)
		features = append(features, feature)
	}

	// Create tasks in both features
	totalCreated := 0
	for _, feature := range features {
		// Create 3 completed tasks in each feature
		for taskNum := 1; taskNum <= 3; taskNum++ {
			task := &models.Task{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("T-%s-%03d", feature.Key, taskNum),
				Title: fmt.Sprintf("Task %d", taskNum)}, FeatureID: feature.ID,

				Status:   models.TaskStatus("completed"),
				Priority: 5,
			}
			err = taskRepo.Create(ctx, task)
			require.NoError(t, err)
			totalCreated++
		}
	}

	// Get task status rollup
	rollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	require.NoError(t, err)

	// Verify counts
	assert.NotNil(t, rollup, "Rollup should not be nil")
	assert.Equal(t, 6, rollup[string(models.TaskStatus("completed"))],
		"Should have 6 completed tasks (3 in each feature)")
	assert.Equal(t, 1, len(rollup), "Map should contain only 1 status")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_GetTaskStatusRollup_EmptyEpic tests empty epic returns empty map
func TestEpicRepository_GetTaskStatusRollup_EmptyEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)

	// Use unique epic key
	epicNum := 10 + (time.Now().UnixNano() % 90)
	epicKey := fmt.Sprintf("E%02d", epicNum)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic with no features
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic - Empty Tasks"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Get task status rollup (should be empty)
	rollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	require.NoError(t, err)

	// Verify empty map
	assert.NotNil(t, rollup, "Rollup should not be nil")
	assert.Equal(t, 0, len(rollup), "Rollup should be empty for epic with no tasks")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestEpicRepository_StatusRollups_Performance tests that queries are efficient with GROUP BY
func TestEpicRepository_StatusRollups_Performance(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Use deterministic key outside the random range (E10-E99) used by other repo tests
	// and outside E01-E05 used by status/benchmark tests
	epicKey := "E08"

	// Pre-cleanup: CASCADE delete handles features and tasks
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create epic
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: epicKey,
		Title: "Test Epic - Performance"}, Status: models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	}()

	// Create multiple features with tasks
	for f := 1; f <= 5; f++ {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("%s-F%02d", epicKey, f),
			Title: fmt.Sprintf("Feature %d", f)}, EpicID: epic.ID,

			Status:      models.FeatureStatusActive,
			ProgressPct: 50.0,
		}
		err = featureRepo.Create(ctx, feature)
		require.NoError(t, err)

		// Create tasks in each feature
		for taskNum := 1; taskNum <= 10; taskNum++ {
			task := &models.Task{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("T-%s-%03d", feature.Key, taskNum),
				Title: fmt.Sprintf("Task %d", taskNum)}, FeatureID: feature.ID,

				Status:   models.TaskStatus("completed"),
				Priority: 5,
			}
			err = taskRepo.Create(ctx, task)
			require.NoError(t, err)
		}
	}

	// Test feature rollup query
	featureRollup, err := epicRepo.GetFeatureStatusRollup(ctx, epic.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, featureRollup[string(models.FeatureStatusActive)])

	// Test task rollup query
	taskRollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, taskRollup[string(models.TaskStatus("completed"))])
}
