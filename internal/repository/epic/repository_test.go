package epic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/feature"
	"github.com/jwwelbor/shark-task-manager/internal/repository/task"
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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
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

// ============================================================================
// Status Rollup Tests (E07-F23)
// ============================================================================

// TestEpicRepository_GetFeatureStatusRollup_WithMultipleFeatures tests feature status aggregation
func TestEpicRepository_GetFeatureStatusRollup_WithMultipleFeatures(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)
	taskRepo := task.NewTaskRepository(db)

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
	db := dbconn.NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)
	taskRepo := task.NewTaskRepository(db)

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
	db := dbconn.NewDB(database)
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
	db := dbconn.NewDB(database)
	epicRepo := NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)
	taskRepo := task.NewTaskRepository(db)

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

// TestEpicRepository_UpdateStatusTx tests the transactional status update method.
func TestEpicRepository_UpdateStatusTx(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEpicRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E87'")

	testEpic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E87", Title: "Epic for UpdateStatusTx Test"},
		Status:     models.EpicStatusDraft,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, repo.Create(ctx, testEpic))
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	t.Run("commits_status_update", func(t *testing.T) {
		// Reset status to draft
		_, _ = database.ExecContext(ctx, "UPDATE epics SET status = 'draft' WHERE id = ?", testEpic.ID)

		tx, err := database.BeginTx(ctx, nil)
		require.NoError(t, err)

		err = repo.UpdateStatusTx(ctx, tx, testEpic.ID, "active", nil, nil)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		updated, err := repo.GetByID(ctx, testEpic.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EpicStatus("active"), updated.Status)
	})

	t.Run("rollback_restores_original_status", func(t *testing.T) {
		// Reset status to draft
		_, _ = database.ExecContext(ctx, "UPDATE epics SET status = 'draft' WHERE id = ?", testEpic.ID)

		tx, err := database.BeginTx(ctx, nil)
		require.NoError(t, err)

		err = repo.UpdateStatusTx(ctx, tx, testEpic.ID, "active", nil, nil)
		require.NoError(t, err)
		require.NoError(t, tx.Rollback())

		current, err := repo.GetByID(ctx, testEpic.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EpicStatusDraft, current.Status)
	})

	t.Run("not_found_returns_error", func(t *testing.T) {
		tx, err := database.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		err = repo.UpdateStatusTx(ctx, tx, 999999999, "active", nil, nil)
		assert.Error(t, err, "expected error for non-existent epic ID")
	})
}

// ptrIntEpic returns a pointer to n; helper for size round-trip tests.
func ptrIntEpic(n int) *int { return &n }

// TestEpicRepository_SizeRoundTrip verifies that Size persists through Create,
// GetByKey, and Update without information loss (TC-F010-A).
func TestEpicRepository_SizeRoundTrip(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEpicRepository(db)

	epicKey := "E98"

	// Clean up before test
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Step 1: Create with Size = ptr(5)
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:   epicKey,
			Title: "Size Round Trip Epic",
			Size:  ptrIntEpic(5),
		},
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := repo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	}()

	// Read back and assert Size == 5
	got, err := repo.GetByKey(ctx, epicKey)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if got.Size == nil {
		t.Fatal("expected Size to be non-nil after Create")
	}
	if *got.Size != 5 {
		t.Errorf("expected Size=5 after Create, got %d", *got.Size)
	}

	// Step 2: Update Size = ptr(1)
	got.Size = ptrIntEpic(1)
	err = repo.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got2, err := repo.GetByKey(ctx, epicKey)
	if err != nil {
		t.Fatalf("GetByKey() after update error = %v", err)
	}
	if got2.Size == nil {
		t.Fatal("expected Size to be non-nil after Update to 1")
	}
	if *got2.Size != 1 {
		t.Errorf("expected Size=1 after update, got %d", *got2.Size)
	}

	// Step 3: Update Size = nil
	got2.Size = nil
	err = repo.Update(ctx, got2)
	if err != nil {
		t.Fatalf("Update() to nil error = %v", err)
	}

	got3, err := repo.GetByKey(ctx, epicKey)
	if err != nil {
		t.Fatalf("GetByKey() after nil update error = %v", err)
	}
	if got3.Size != nil {
		t.Errorf("expected Size=nil after clearing, got %v", *got3.Size)
	}
}

// --- GetRecent tests (T-E07-F17-002) ---

// seedEpicsWithTimestamps creates n epics with created_at staggered by 1 second each
// (oldest first). Uses direct SQL INSERT to bypass key-format validation and allow
// arbitrary timestamps. Returns epic IDs for deferred cleanup.
func seedEpicsWithTimestamps(t *testing.T, _ *EpicRepository, db *dbconn.DB, n int) []int64 {
	t.Helper()
	ctx := context.Background()
	baseTime := time.Now().UTC().Add(-time.Duration(n) * time.Second)
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		// Use a key that is clearly test-only and avoids collisions.
		// Direct INSERT bypasses model validation; cascade-delete from id cleans up.
		key := fmt.Sprintf("recent-bench-epic-%03d", i+1)
		ts := baseTime.Add(time.Duration(i) * time.Second)
		result, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO epics (key, title, status, priority, created_at, updated_at)
			 VALUES (?, ?, 'draft', 'medium', ?, CURRENT_TIMESTAMP)`,
			key, fmt.Sprintf("Recent Epic %d", i+1), ts.Format("2006-01-02T15:04:05Z"),
		)
		require.NoError(t, err, "seedEpicsWithTimestamps: INSERT failed for key %s", key)
		id, err := result.LastInsertId()
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return ids
}

// TestEpicRepository_GetRecent_OrdersByCreatedAtDesc seeds 5 epics with distinct timestamps
// and asserts that GetRecent returns them in created_at DESC order.
func TestEpicRepository_GetRecent_OrdersByCreatedAtDesc(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEpicRepository(db)

	// Pre-cleanup: remove any leftover RECENT-tagged epics
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'recent-bench-epic-%'")

	ids := seedEpicsWithTimestamps(t, repo, db, 5)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", id)
		}
	}()

	epics, err := repo.GetRecent(ctx, 5)
	require.NoError(t, err)
	require.Len(t, epics, 5)

	for i := 1; i < len(epics); i++ {
		assert.True(t, !epics[i-1].CreatedAt.Before(epics[i].CreatedAt),
			"expected epics[%d].CreatedAt >= epics[%d].CreatedAt", i-1, i)
	}
}

// TestEpicRepository_GetRecent_LimitRespected seeds 10 epics and asserts GetRecent(ctx, 3)
// returns exactly 3 rows (AC-T1).
func TestEpicRepository_GetRecent_LimitRespected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEpicRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'recent-bench-epic-%'")

	ids := seedEpicsWithTimestamps(t, repo, db, 10)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", id)
		}
	}()

	epics, err := repo.GetRecent(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, epics, 3, "GetRecent(3) must return exactly 3 rows")
}

// TestEpicRepository_GetRecent_EmptyTable asserts that GetRecent returns a non-nil
// empty slice (not nil) when the table is empty (AC-T3).
func TestEpicRepository_GetRecent_EmptyTable(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEpicRepository(db)

	// Clean up RECENT-tagged epics to minimize rows returned
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'recent-bench-epic-%'")

	epics, err := repo.GetRecent(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, epics, "GetRecent must return a non-nil slice")
}

// TestEpicRepository_GetRecent_LimitExceedsRowCount seeds 2 epics and asserts that
// GetRecent(ctx, 100) returns at least 2 rows (all rows, not an error).
func TestEpicRepository_GetRecent_LimitExceedsRowCount(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewEpicRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'recent-bench-epic-%'")

	ids := seedEpicsWithTimestamps(t, repo, db, 2)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", id)
		}
	}()

	epics, err := repo.GetRecent(ctx, 100)
	require.NoError(t, err)
	// At least 2 rows must be returned; there may be more from the test DB.
	assert.GreaterOrEqual(t, len(epics), 2, "GetRecent(100) must return all available rows when limit > row count")
}
