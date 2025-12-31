package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskRepository_GetByKey_NumericFormat tests lookup with traditional numeric key
func TestTaskRepository_GetByKey_NumericFormat(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E99-F99-100'")

	// Seed epic and feature
	_, featureID := test.SeedTestData()

	// Create task with numeric key
	task := &models.Task{
		FeatureID: featureID,
		Key:       "T-E99-F99-100",
		Title:     "Test Numeric Key Lookup",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err := repo.Create(ctx, task)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

	// Test: Lookup by numeric key should succeed
	retrieved, err := repo.GetByKey(ctx, "T-E99-F99-100")
	require.NoError(t, err)
	assert.Equal(t, "T-E99-F99-100", retrieved.Key)
	assert.Equal(t, "Test Numeric Key Lookup", retrieved.Title)
}

// TestTaskRepository_GetByKey_SluggedFormat tests lookup with slugged key
func TestTaskRepository_GetByKey_SluggedFormat(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E98-F01-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create dedicated epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic for Dual Key Lookup",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)

	// Create dedicated feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E98-F01",
		Title:  "Test Feature for Dual Key",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)

	// Create task with numeric key (slug will be auto-generated)
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E98-F01-001",
		Title:     "Implement User Authentication",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

	// Verify slug was generated
	require.NotNil(t, task.Slug)
	expectedSlug := "implement-user-authentication"
	assert.Equal(t, expectedSlug, *task.Slug)

	// Test: Lookup by slugged key should succeed
	sluggedKey := "T-E98-F01-001-implement-user-authentication"
	retrieved, err := repo.GetByKey(ctx, sluggedKey)
	require.NoError(t, err, "Should find task by slugged key")
	assert.Equal(t, "T-E98-F01-001", retrieved.Key)
	assert.Equal(t, "Implement User Authentication", retrieved.Title)
}

// TestTaskRepository_GetByKey_SlugMismatch tests that mismatched slug doesn't match
func TestTaskRepository_GetByKey_SlugMismatch(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E98-F02-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F02'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create dedicated epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic for Slug Mismatch",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)

	// Create dedicated feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E98-F02",
		Title:  "Test Feature for Slug Mismatch",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)

	// Create task
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E98-F02-001",
		Title:     "Fix Database Bug",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

	// Verify slug was generated
	require.NotNil(t, task.Slug)
	expectedSlug := "fix-database-bug"
	assert.Equal(t, expectedSlug, *task.Slug)

	// Test: Lookup with wrong slug should fail
	wrongSluggedKey := "T-E98-F02-001-wrong-slug-name"
	retrieved, err := repo.GetByKey(ctx, wrongSluggedKey)
	assert.Error(t, err, "Should fail when slug doesn't match")
	assert.Nil(t, retrieved)
}

// TestTaskRepository_GetByKey_PartialSlugMatch tests partial slug matching
func TestTaskRepository_GetByKey_PartialSlugMatch(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E98-F03-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F03'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create dedicated epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic for Partial Slug",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)

	// Create dedicated feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E98-F03",
		Title:  "Test Feature for Partial Slug",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)

	// Create task with long title (slug gets truncated to 100 chars)
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E98-F03-001",
		Title:     "Implement Advanced User Authentication System With Multi-Factor Support And Session Management",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

	// Verify slug was generated
	require.NotNil(t, task.Slug)

	// Test: Lookup with full slug should succeed
	fullSluggedKey := "T-E98-F03-001-" + *task.Slug
	retrieved, err := repo.GetByKey(ctx, fullSluggedKey)
	require.NoError(t, err, "Should find task by full slugged key")
	assert.Equal(t, "T-E98-F03-001", retrieved.Key)

	// Test: Lookup with partial slug should fail (we match exact slug only)
	partialSluggedKey := "T-E98-F03-001-implement-advanced"
	retrieved, err = repo.GetByKey(ctx, partialSluggedKey)
	assert.Error(t, err, "Should fail with partial slug")
	assert.Nil(t, retrieved)
}

// TestTaskRepository_GetByKey_NoSlug tests lookup when task has no slug
func TestTaskRepository_GetByKey_NoSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E98-F04-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F04'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create dedicated epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E98",
		Title:         "Test Epic for No Slug",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)

	// Create dedicated feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E98-F04",
		Title:  "Test Feature for No Slug",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)

	// Create task and then manually clear the slug to simulate legacy data
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E98-F04-001",
		Title:     "Legacy Task Without Slug",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

	// Manually clear slug to simulate legacy task
	_, err = database.ExecContext(ctx, "UPDATE tasks SET slug = NULL WHERE id = ?", task.ID)
	require.NoError(t, err)

	// Test: Lookup by numeric key should still work
	retrieved, err := repo.GetByKey(ctx, "T-E98-F04-001")
	require.NoError(t, err)
	assert.Equal(t, "T-E98-F04-001", retrieved.Key)
	assert.Equal(t, "Legacy Task Without Slug", retrieved.Title)
	assert.Nil(t, retrieved.Slug)

	// Test: Lookup with any slugged format should fail gracefully
	retrieved, err = repo.GetByKey(ctx, "T-E98-F04-001-some-slug")
	assert.Error(t, err, "Should fail when task has no slug")
	assert.Nil(t, retrieved)
}

// TestTaskRepository_GetByKey_InvalidKey tests error handling for invalid keys
func TestTaskRepository_GetByKey_InvalidKey(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTaskRepository(db)

	// Test: Empty key
	retrieved, err := repo.GetByKey(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Test: Invalid format (not a task key)
	retrieved, err = repo.GetByKey(ctx, "invalid-key-format")
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Test: Non-existent numeric key
	retrieved, err = repo.GetByKey(ctx, "T-E99-F99-999")
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Test: Non-existent slugged key
	retrieved, err = repo.GetByKey(ctx, "T-E99-F99-999-non-existent-task")
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}
