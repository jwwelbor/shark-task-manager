package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetTaskStatusBreakdown tests the task status breakdown query
func TestGetTaskStatusBreakdown(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)
	taskRepo := NewTaskRepository(db)

	// Clean up test data
	_, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Status Cascade Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E98-F01",
		Title:  "Status Cascade Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Create tasks with various statuses
	tasks := []*models.Task{
		{FeatureID: feature.ID, Key: "T-E98-F01-001", Title: "Todo Task 1", Status: models.TaskStatusTodo, Priority: 5},
		{FeatureID: feature.ID, Key: "T-E98-F01-002", Title: "Todo Task 2", Status: models.TaskStatusTodo, Priority: 5},
		{FeatureID: feature.ID, Key: "T-E98-F01-003", Title: "In Progress Task", Status: models.TaskStatusInProgress, Priority: 5},
		{FeatureID: feature.ID, Key: "T-E98-F01-004", Title: "Completed Task 1", Status: models.TaskStatusCompleted, Priority: 5},
		{FeatureID: feature.ID, Key: "T-E98-F01-005", Title: "Completed Task 2", Status: models.TaskStatusCompleted, Priority: 5},
		{FeatureID: feature.ID, Key: "T-E98-F01-006", Title: "Blocked Task", Status: models.TaskStatusBlocked, Priority: 5},
	}

	for _, task := range tasks {
		err := taskRepo.Create(ctx, task)
		require.NoError(t, err)
	}

	// Test GetTaskStatusBreakdown
	counts, err := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
	require.NoError(t, err)

	assert.Equal(t, 2, counts[models.TaskStatusTodo], "Should have 2 todo tasks")
	assert.Equal(t, 1, counts[models.TaskStatusInProgress], "Should have 1 in_progress task")
	assert.Equal(t, 2, counts[models.TaskStatusCompleted], "Should have 2 completed tasks")
	assert.Equal(t, 1, counts[models.TaskStatusBlocked], "Should have 1 blocked task")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
}

// TestGetFeatureStatusBreakdown tests the feature status breakdown query
func TestGetFeatureStatusBreakdown(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E97-%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create test epic
	epic := &models.Epic{
		Key:      "E97",
		Title:    "Feature Status Breakdown Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create features with various statuses
	features := []*models.Feature{
		{EpicID: epic.ID, Key: "E97-F01", Title: "Draft Feature 1", Status: models.FeatureStatusDraft},
		{EpicID: epic.ID, Key: "E97-F02", Title: "Draft Feature 2", Status: models.FeatureStatusDraft},
		{EpicID: epic.ID, Key: "E97-F03", Title: "Active Feature", Status: models.FeatureStatusActive},
		{EpicID: epic.ID, Key: "E97-F04", Title: "Completed Feature 1", Status: models.FeatureStatusCompleted},
		{EpicID: epic.ID, Key: "E97-F05", Title: "Completed Feature 2", Status: models.FeatureStatusCompleted},
	}

	for _, feature := range features {
		err := featureRepo.Create(ctx, feature)
		require.NoError(t, err)
	}

	// Test GetFeatureStatusBreakdown
	counts, err := epicRepo.GetFeatureStatusBreakdown(ctx, epic.ID)
	require.NoError(t, err)

	assert.Equal(t, 2, counts[models.FeatureStatusDraft], "Should have 2 draft features")
	assert.Equal(t, 1, counts[models.FeatureStatusActive], "Should have 1 active feature")
	assert.Equal(t, 2, counts[models.FeatureStatusCompleted], "Should have 2 completed features")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E97-%'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")
}

// TestSetStatusOverride tests the status override functionality
func TestSetStatusOverride(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key = 'E96-F01'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")

	// Create test epic
	epic := &models.Epic{
		Key:      "E96",
		Title:    "Status Override Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E96-F01",
		Title:  "Status Override Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Verify initial override state is false
	retrievedFeature, err := featureRepo.GetByKey(ctx, "E96-F01")
	require.NoError(t, err)
	assert.False(t, retrievedFeature.StatusOverride, "Initial status_override should be false")

	// Test enabling override
	err = featureRepo.SetStatusOverride(ctx, feature.ID, true)
	require.NoError(t, err)

	retrievedFeature, err = featureRepo.GetByKey(ctx, "E96-F01")
	require.NoError(t, err)
	assert.True(t, retrievedFeature.StatusOverride, "status_override should be true after enabling")

	// Test disabling override
	err = featureRepo.SetStatusOverride(ctx, feature.ID, false)
	require.NoError(t, err)

	retrievedFeature, err = featureRepo.GetByKey(ctx, "E96-F01")
	require.NoError(t, err)
	assert.False(t, retrievedFeature.StatusOverride, "status_override should be false after disabling")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key = 'E96-F01'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")
}

// TestUpdateStatusIfNotOverridden tests conditional status update
func TestUpdateStatusIfNotOverridden(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	featureRepo := NewFeatureRepository(db)
	epicRepo := NewEpicRepository(db)

	// Clean up test data
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F01'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")

	// Create test epic
	epic := &models.Epic{
		Key:      "E95",
		Title:    "Conditional Update Test Epic",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E95-F01",
		Title:  "Conditional Update Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Test 1: Update should succeed when override is false
	updated, err := featureRepo.UpdateStatusIfNotOverridden(ctx, feature.ID, models.FeatureStatusActive)
	require.NoError(t, err)
	assert.True(t, updated, "Should update when override is false")

	retrievedFeature, err := featureRepo.GetByKey(ctx, "E95-F01")
	require.NoError(t, err)
	assert.Equal(t, models.FeatureStatusActive, retrievedFeature.Status, "Status should be updated to active")

	// Test 2: Enable override and try to update
	err = featureRepo.SetStatusOverride(ctx, feature.ID, true)
	require.NoError(t, err)

	updated, err = featureRepo.UpdateStatusIfNotOverridden(ctx, feature.ID, models.FeatureStatusCompleted)
	require.NoError(t, err)
	assert.False(t, updated, "Should not update when override is true")

	retrievedFeature, err = featureRepo.GetByKey(ctx, "E95-F01")
	require.NoError(t, err)
	assert.Equal(t, models.FeatureStatusActive, retrievedFeature.Status, "Status should remain active (not updated)")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F01'")
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")
}

// TestEpicUpdateStatus tests the epic status update method
func TestEpicUpdateStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	epicRepo := NewEpicRepository(db)

	// Clean up test data
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E94'")

	// Create test epic
	epic := &models.Epic{
		Key:      "E94",
		Title:    "Epic Status Update Test",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Update status
	err = epicRepo.UpdateStatus(ctx, epic.ID, models.EpicStatusActive)
	require.NoError(t, err)

	// Verify update
	retrievedEpic, err := epicRepo.GetByKey(ctx, "E94")
	require.NoError(t, err)
	assert.Equal(t, models.EpicStatusActive, retrievedEpic.Status, "Epic status should be updated to active")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E94'")
}
