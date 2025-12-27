package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCriteriaTestDB(t *testing.T) *DB {
	testDB, err := db.InitDB(":memory:")
	require.NoError(t, err)
	return &DB{DB: testDB}
}

func createTestTask(t *testing.T, db *DB) int64 {
	// Create epic
	epicRepo := NewEpicRepository(db)
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   "active",
		Priority: "high",
	}
	err := epicRepo.Create(context.Background(), epic)
	require.NoError(t, err)

	// Create feature
	featureRepo := NewFeatureRepository(db)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Test Feature",
		Status: "active",
	}
	err = featureRepo.Create(context.Background(), feature)
	require.NoError(t, err)

	// Create task
	taskRepo := NewTaskRepository(db)
	agentType := models.AgentTypeBackend
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E01-F01-001",
		Title:     "Test Task",
		Status:    "todo",
		Priority:  5,
		AgentType: &agentType,
	}
	err = taskRepo.Create(context.Background(), task)
	require.NoError(t, err)

	return task.ID
}

func TestTaskCriteriaRepository_Create(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	criteria := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "All tests passing",
		Status:    models.CriteriaStatusPending,
	}

	err := repo.Create(ctx, criteria)
	assert.NoError(t, err)
	assert.NotZero(t, criteria.ID)
}

func TestTaskCriteriaRepository_GetByID(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	// Create criterion
	criteria := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "Code reviewed",
		Status:    models.CriteriaStatusPending,
	}
	err := repo.Create(ctx, criteria)
	require.NoError(t, err)

	// Retrieve it
	retrieved, err := repo.GetByID(ctx, criteria.ID)
	assert.NoError(t, err)
	assert.Equal(t, criteria.Criterion, retrieved.Criterion)
	assert.Equal(t, criteria.Status, retrieved.Status)
}

func TestTaskCriteriaRepository_GetByTaskID(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	// Create multiple criteria
	criteria1 := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "Tests written",
		Status:    models.CriteriaStatusComplete,
	}
	criteria2 := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "Documentation updated",
		Status:    models.CriteriaStatusPending,
	}

	require.NoError(t, repo.Create(ctx, criteria1))
	require.NoError(t, repo.Create(ctx, criteria2))

	// Retrieve all
	all, err := repo.GetByTaskID(ctx, taskID)
	assert.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestTaskCriteriaRepository_Update(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	// Create criterion
	criteria := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "Original text",
		Status:    models.CriteriaStatusPending,
	}
	require.NoError(t, repo.Create(ctx, criteria))

	// Update it
	criteria.Criterion = "Updated text"
	criteria.Status = models.CriteriaStatusComplete
	now := time.Now()
	criteria.VerifiedAt = &now
	notes := "Verified manually"
	criteria.VerificationNotes = &notes

	err := repo.Update(ctx, criteria)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, criteria.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated text", retrieved.Criterion)
	assert.Equal(t, models.CriteriaStatusComplete, retrieved.Status)
	assert.NotNil(t, retrieved.VerifiedAt)
	assert.NotNil(t, retrieved.VerificationNotes)
}

func TestTaskCriteriaRepository_UpdateStatus(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	// Create criterion
	criteria := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "Deploy to production",
		Status:    models.CriteriaStatusPending,
	}
	require.NoError(t, repo.Create(ctx, criteria))

	// Mark as complete
	notes := "Deployed successfully"
	err := repo.UpdateStatus(ctx, criteria.ID, models.CriteriaStatusComplete, &notes)
	assert.NoError(t, err)

	// Verify
	retrieved, err := repo.GetByID(ctx, criteria.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.CriteriaStatusComplete, retrieved.Status)
	assert.NotNil(t, retrieved.VerifiedAt)
	assert.NotNil(t, retrieved.VerificationNotes)
	assert.Equal(t, notes, *retrieved.VerificationNotes)
}

func TestTaskCriteriaRepository_Delete(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	// Create criterion
	criteria := &models.TaskCriteria{
		TaskID:    taskID,
		Criterion: "To be deleted",
		Status:    models.CriteriaStatusPending,
	}
	require.NoError(t, repo.Create(ctx, criteria))

	// Delete it
	err := repo.Delete(ctx, criteria.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, criteria.ID)
	assert.Error(t, err)
}

func TestTaskCriteriaRepository_GetSummary(t *testing.T) {
	db := setupCriteriaTestDB(t)
	defer db.Close()

	taskID := createTestTask(t, db)
	repo := NewTaskCriteriaRepository(db)
	ctx := context.Background()

	// Create criteria with different statuses
	statuses := []models.CriteriaStatus{
		models.CriteriaStatusPending,
		models.CriteriaStatusPending,
		models.CriteriaStatusInProgress,
		models.CriteriaStatusComplete,
		models.CriteriaStatusComplete,
		models.CriteriaStatusComplete,
		models.CriteriaStatusFailed,
		models.CriteriaStatusNA,
	}

	for i, status := range statuses {
		criteria := &models.TaskCriteria{
			TaskID:    taskID,
			Criterion: fmt.Sprintf("Criterion %d", i+1),
			Status:    status,
		}
		require.NoError(t, repo.Create(ctx, criteria))
	}

	// Get summary
	summary, err := repo.GetSummaryByTaskID(ctx, taskID)
	assert.NoError(t, err)
	assert.Equal(t, 8, summary.TotalCount)
	assert.Equal(t, 2, summary.PendingCount)
	assert.Equal(t, 1, summary.InProgressCount)
	assert.Equal(t, 3, summary.CompleteCount)
	assert.Equal(t, 1, summary.FailedCount)
	assert.Equal(t, 1, summary.NACount)

	// Completion % = (complete + na) / total = (3 + 1) / 8 = 50%
	assert.InDelta(t, 50.0, summary.CompletionPct, 0.01)
}
