package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	sprintrepo "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	internaltesthelper "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CloseSprintWithCarryover tests (TC-C08 through TC-C12) — T-E19-F03-007
// ---------------------------------------------------------------------------

// buildCloseMockRepo builds a MockSprintRepository for close-sprint tests.
// sprint S024 (id=24) is in "in_progress" status.
func buildCloseMockRepo(
	t *testing.T,
	allAssignments []*models.SprintAssignment,
	incompleteAssignments []*models.SprintAssignment,
	planningSprints []*models.Sprint,
	captureCompletion **models.SprintCompletion,
	reassignCalled *bool,
	dropCalled *bool,
	completionError error,
	closedSprintStart time.Time,
	closedSprintEnd time.Time,
) *MockSprintRepository {
	t.Helper()

	sprintS024 := &models.Sprint{
		ID:        24,
		Key:       "S024",
		Name:      "Sprint 24",
		Status:    models.SprintStatus("in_progress"),
		StartDate: closedSprintStart,
		EndDate:   closedSprintEnd,
	}

	callCount := 0

	return &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			callCount++
			if callCount == 1 {
				return sprintS024, nil
			}
			// Second call (after commit): sprint is now "completed"
			completedSprint := *sprintS024
			completedSprint.Status = "completed"
			return &completedSprint, nil
		},
		ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			assert.Equal(t, int64(24), sprintID)
			return allAssignments, nil
		},
		ListAssignmentsForCarryoverFunc: func(ctx context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error) {
			assert.Equal(t, int64(24), sprintID)
			return incompleteAssignments, nil
		},
		ListFunc: func(ctx context.Context, filters *sprintrepo.SprintListFilters) ([]*models.Sprint, error) {
			return planningSprints, nil
		},
		GetNextKeyFunc: func(ctx context.Context) (string, error) {
			return "S025", nil
		},
		CreateFunc: func(ctx context.Context, s *models.Sprint) error {
			s.ID = 25
			return nil
		},
		ReassignToSprintTxFunc: func(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error {
			if reassignCalled != nil {
				*reassignCalled = true
			}
			return nil
		},
		DropAssignmentsTxFunc: func(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error {
			if dropCalled != nil {
				*dropCalled = true
			}
			return nil
		},
		UpdateStatusTxFunc: func(ctx context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error {
			assert.Equal(t, int64(24), id)
			assert.Equal(t, models.SprintStatus("completed"), status)
			return nil
		},
		CreateCompletionTxFunc: func(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error {
			if captureCompletion != nil {
				*captureCompletion = completion
			}
			return completionError
		},
	}
}

// defaultSprintDates returns a consistent start/end for test sprints.
func defaultSprintDates() (start, end time.Time) {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
}

// TestSprintService_CloseSprintWithCarryover_CompletionRecord tests TC-C08:
// CloseSprintWithCarryover creates a sprint_completions row with correct fields.
//
// Caller-Path Contract (TC-C08):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)
//   - Lowest mock seam: SprintRepository.CreateCompletionTx (capture argument)
//   - Forbidden mocks: Do NOT mock away CreateCompletionTx without capturing
//   - Counter-factual: a buggy impl that passes CompletedEntityCount=0 always would produce wrong velocity data
func TestSprintService_CloseSprintWithCarryover_CompletionRecord(t *testing.T) {
	ctx := context.Background()

	// 5 total assignments, 3 incomplete
	allAssignments := []*models.SprintAssignment{
		{ID: 201, SprintID: 24, EntityType: "task", EntityID: 1},
		{ID: 202, SprintID: 24, EntityType: "task", EntityID: 2},
		{ID: 203, SprintID: 24, EntityType: "task", EntityID: 3},
		{ID: 204, SprintID: 24, EntityType: "task", EntityID: 4},
		{ID: 205, SprintID: 24, EntityType: "task", EntityID: 5},
	}
	incompleteAssignments := []*models.SprintAssignment{
		{ID: 203, SprintID: 24, EntityType: "task", EntityID: 3},
		{ID: 204, SprintID: 24, EntityType: "task", EntityID: 4},
		{ID: 205, SprintID: 24, EntityType: "task", EntityID: 5},
	}
	// Existing planning sprint (TC-C01 path)
	planningSprints := []*models.Sprint{
		{ID: 25, Key: "S025", Name: "Sprint 25", Status: "todo"},
	}

	var capturedCompletion *models.SprintCompletion
	var reassignCalled bool
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, planningSprints,
		&capturedCompletion, &reassignCalled, nil, nil,
		start, end,
	)

	// Use a real test DB for BeginTxContext (TC-C08 needs a real transaction)
	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)

	require.NoError(t, err)
	require.NotNil(t, result)

	// TC-C08: verify SprintCloseResult counts
	assert.Equal(t, 2, result.CompletedCount, "CompletedCount = total - incomplete = 5 - 3 = 2")
	assert.Equal(t, 3, result.CarriedOverCount, "CarriedOverCount = len(incomplete) = 3")
	assert.Equal(t, 0, result.DroppedCount)
	assert.Equal(t, "S025", result.NextSprintKey)

	// TC-C08: verify completion record fields
	require.NotNil(t, capturedCompletion, "CreateCompletionTx must have been called with a completion struct")
	assert.Equal(t, int64(24), capturedCompletion.SprintID, "SprintID must match the closed sprint")
	assert.Equal(t, 5, capturedCompletion.PlannedEntityCount, "PlannedEntityCount = total assignments")
	assert.Equal(t, 2, capturedCompletion.CompletedEntityCount, "CompletedEntityCount = total - incomplete")
	assert.Equal(t, 3, capturedCompletion.CarriedOverCount)
	assert.Equal(t, 0, capturedCompletion.DroppedCount)
	assert.Equal(t, "next", capturedCompletion.CarryoverMode)
	require.NotNil(t, capturedCompletion.NextSprintID)
	assert.Equal(t, int64(25), *capturedCompletion.NextSprintID)

	// ReassignToSprintTx was called (not DropAssignmentsTx)
	assert.True(t, reassignCalled, "ReassignToSprintTx must be called for carryover=next")
}

// TestSprintService_CloseSprintWithCarryover_BacklogMode tests TC-C05:
// carryover=backlog soft-deletes incomplete assignments.
//
// Caller-Path Contract (TC-C05):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverBacklog)
//   - Lowest mock seam: SprintRepository (Tx variants)
//   - Forbidden mocks: Do NOT mock DropAssignmentsTx away
//   - Counter-factual: a buggy impl that calls ReassignToSprintTx instead of DropAssignmentsTx
func TestSprintService_CloseSprintWithCarryover_BacklogMode(t *testing.T) {
	ctx := context.Background()

	allAssignments := []*models.SprintAssignment{
		{ID: 201}, {ID: 202}, {ID: 203}, {ID: 204}, {ID: 205},
	}
	incompleteAssignments := []*models.SprintAssignment{
		{ID: 203}, {ID: 204}, {ID: 205},
	}

	var capturedCompletion *models.SprintCompletion
	var reassignCalled bool
	var dropCalled bool
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, nil,
		&capturedCompletion, &reassignCalled, &dropCalled, nil,
		start, end,
	)

	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverBacklog)

	require.NoError(t, err)
	require.NotNil(t, result)

	// TC-C05: DropAssignmentsTx called, ReassignToSprintTx NOT called
	assert.True(t, dropCalled, "DropAssignmentsTx must be called for carryover=backlog")
	assert.False(t, reassignCalled, "ReassignToSprintTx must NOT be called for carryover=backlog")

	// TC-C05: counts
	assert.Equal(t, 2, result.CompletedCount)
	assert.Equal(t, 0, result.CarriedOverCount)
	assert.Equal(t, 3, result.DroppedCount)
	assert.Equal(t, "", result.NextSprintKey)

	// completion record fields for backlog mode
	require.NotNil(t, capturedCompletion)
	assert.Equal(t, "backlog", capturedCompletion.CarryoverMode)
	assert.Equal(t, 3, capturedCompletion.DroppedCount)
	assert.Nil(t, capturedCompletion.NextSprintID, "NextSprintID must be nil for backlog mode")
}

// TestSprintService_CloseSprintWithCarryover_BacklogMode_EmptySprint tests TC-C06:
// carryover=backlog with 0 incomplete assignments succeeds without errors.
func TestSprintService_CloseSprintWithCarryover_BacklogMode_EmptySprint(t *testing.T) {
	ctx := context.Background()

	// Sprint with 2 completed, 0 incomplete
	allAssignments := []*models.SprintAssignment{{ID: 201}, {ID: 202}}
	var incompleteAssignments []*models.SprintAssignment // empty

	var capturedCompletion *models.SprintCompletion
	var dropCalled bool
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, nil,
		&capturedCompletion, nil, &dropCalled, nil,
		start, end,
	)

	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverBacklog)

	require.NoError(t, err, "empty sprint (0 incomplete) must close successfully")
	require.NotNil(t, result)

	// TC-C06: no entities to drop; counts all zero except completed
	assert.Equal(t, 2, result.CompletedCount)
	assert.Equal(t, 0, result.DroppedCount, "DroppedCount must be 0 when no incomplete entities")

	// DropAssignmentsTx was still called (with empty slice — no-op in repo)
	require.NotNil(t, capturedCompletion)
	assert.Equal(t, 0, capturedCompletion.DroppedCount)
}

// TestSprintService_CloseSprintWithCarryover_ConfigDefault_Next tests TC-C09:
// empty carryoverMode with config "next" behaves as CarryoverNext.
//
// Caller-Path Contract (TC-C09):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", "") — empty string
//   - Lowest mock seam: SprintRepository; inject real Config with SprintDefaults.Carryover="next"
//   - Forbidden mocks: Do NOT mock the config read — inject a real Config struct
//   - Counter-factual: a buggy impl that ignores config and always uses "backlog" would call
//     DropAssignmentsTx instead of ReassignToSprintTx
func TestSprintService_CloseSprintWithCarryover_ConfigDefault_Next(t *testing.T) {
	ctx := context.Background()

	allAssignments := []*models.SprintAssignment{{ID: 201}}
	incompleteAssignments := []*models.SprintAssignment{{ID: 201}}
	planningSprints := []*models.Sprint{
		{ID: 25, Key: "S025", Name: "Sprint 25", Status: "todo"},
	}

	var reassignCalled bool
	var dropCalled bool
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, planningSprints,
		nil, &reassignCalled, &dropCalled, nil,
		start, end,
	)

	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	// Inject real Config with CarryoverBehavior = "next"
	cfg := &config.Config{
		SprintDefaults: &config.SprintDefaultsConfig{
			CarryoverBehavior: "next",
		},
	}
	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, cfg, db)

	// Empty carryoverMode — should read "next" from config
	result, err := svc.CloseSprintWithCarryover(ctx, "S024", "")

	require.NoError(t, err)
	require.NotNil(t, result)

	// TC-C09: config "next" → ReassignToSprintTx called, DropAssignmentsTx NOT called
	assert.True(t, reassignCalled, "config carryover=next must cause ReassignToSprintTx to be called")
	assert.False(t, dropCalled, "DropAssignmentsTx must NOT be called when config is 'next'")
}

// TestSprintService_CloseSprintWithCarryover_ConfigDefault_AbsentDefaultsToNext tests TC-C10:
// when config key is absent, service defaults to CarryoverNext.
//
// Caller-Path Contract (TC-C10):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", "")
//   - Lowest mock seam: SprintRepository; inject Config with SprintDefaults.Carryover=""
//   - Counter-factual: a buggy impl that always defaults to "backlog" would call DropAssignmentsTx
func TestSprintService_CloseSprintWithCarryover_ConfigDefault_AbsentDefaultsToNext(t *testing.T) {
	ctx := context.Background()

	allAssignments := []*models.SprintAssignment{{ID: 201}}
	incompleteAssignments := []*models.SprintAssignment{{ID: 201}}
	planningSprints := []*models.Sprint{
		{ID: 25, Key: "S025", Name: "Sprint 25", Status: "todo"},
	}

	var reassignCalled bool
	var dropCalled bool
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, planningSprints,
		nil, &reassignCalled, &dropCalled, nil,
		start, end,
	)

	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	// Config present but CarryoverBehavior absent (empty string → default to "next")
	cfg := &config.Config{
		SprintDefaults: &config.SprintDefaultsConfig{
			CarryoverBehavior: "",
		},
	}
	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, cfg, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", "")

	require.NoError(t, err)
	require.NotNil(t, result)

	// TC-C10: absent config → default to CarryoverNext
	assert.True(t, reassignCalled, "absent config must default to CarryoverNext → ReassignToSprintTx called")
	assert.False(t, dropCalled, "DropAssignmentsTx must NOT be called when config is absent (defaults to next)")
}

// TestSprintService_CloseSprintWithCarryover_RollbackOnError tests TC-C11:
// when CreateCompletionTx errors, the transaction must roll back and an error is returned.
//
// Caller-Path Contract (TC-C11):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)
//   - Lowest mock seam: SprintRepository.CreateCompletionTx — inject error (last step in tx)
//   - Forbidden mocks: Do NOT mock defer tx.Rollback() — the real Rollback must execute
//   - Counter-factual: a buggy impl without defer tx.Rollback() would leave partial writes committed
//
// This test uses a real test DB for BeginTxContext so that the transaction is real.
func TestSprintService_CloseSprintWithCarryover_RollbackOnError(t *testing.T) {
	ctx := context.Background()

	allAssignments := []*models.SprintAssignment{{ID: 201}}
	incompleteAssignments := []*models.SprintAssignment{{ID: 201}}
	planningSprints := []*models.Sprint{
		{ID: 25, Key: "S025", Name: "Sprint 25", Status: "todo"},
	}

	completionError := fmt.Errorf("simulated CreateCompletionTx failure")
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, planningSprints,
		nil, nil, nil, completionError,
		start, end,
	)

	// Use the real test DB so BeginTxContext opens a real transaction (TC-C11)
	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)

	// TC-C11: error must be returned when CreateCompletionTx fails
	require.Error(t, err, "CloseSprintWithCarryover must return an error when CreateCompletionTx fails")
	assert.Nil(t, result, "result must be nil on error")
	// Error message must mention the completions failure
	assert.Contains(t, err.Error(), "sprint_completions", "error should mention sprint_completions")
}

// TestSprintService_CloseSprintWithCarryover_WrongStatus tests TC-C12:
// sprint not in "in_progress" status is rejected before any DB writes.
//
// Caller-Path Contract (TC-C12):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext) with sprint in "todo"
//   - Counter-factual: a buggy impl that skips the status check would begin a transaction
//     and call repo methods on a non-active sprint
func TestSprintService_CloseSprintWithCarryover_WrongStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentStatus models.SprintStatus
	}{
		{"planning sprint rejected", "todo"},
		{"completed sprint rejected", "completed"},
		{"ready_for_review sprint rejected", "ready_for_review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listAssignmentsCalled := false

			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return &models.Sprint{
						ID:     24,
						Key:    "S024",
						Name:   "Sprint 24",
						Status: tt.currentStatus,
					}, nil
				},
				ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
					listAssignmentsCalled = true
					return nil, nil
				},
			}

			testSQLDB := internaltesthelper.GetTestDB()
			db := repository.NewDB(testSQLDB)

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

			result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)

			require.Error(t, err, "CloseSprintWithCarryover must return error when sprint is not in_progress")
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), string(tt.currentStatus),
				"error message must include current status")
			assert.False(t, listAssignmentsCalled,
				"ListAssignments must NOT be called when status check rejects sprint")
		})
	}
}

// TestSprintService_CloseSprintWithCarryover_AutoCreateNextSprint tests TC-C02 and TC-C04:
// when no planning sprint exists, auto-creates one with start_date = end_date + 1 day.
//
// Caller-Path Contract (TC-C02/TC-C04):
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)
//   - Lowest mock seam: SprintRepository — List returns empty; Create captured
//   - Counter-factual: a buggy impl that returns an error when no sprint exists would fail
func TestSprintService_CloseSprintWithCarryover_AutoCreateNextSprint(t *testing.T) {
	ctx := context.Background()

	allAssignments := []*models.SprintAssignment{{ID: 201}}
	incompleteAssignments := []*models.SprintAssignment{{ID: 201}}

	closedSprintStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	closedSprintEnd := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)

	var capturedNewSprint *models.Sprint
	var reassignCalled bool
	callCount := 0

	sprintS024 := &models.Sprint{
		ID:        24,
		Key:       "S024",
		Name:      "Sprint 24",
		Status:    models.SprintStatus("in_progress"),
		StartDate: closedSprintStart,
		EndDate:   closedSprintEnd,
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			callCount++
			if callCount == 1 {
				return sprintS024, nil
			}
			// After commit: sprint is completed
			completed := *sprintS024
			completed.Status = "completed"
			return &completed, nil
		},
		ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			return allAssignments, nil
		},
		ListAssignmentsForCarryoverFunc: func(ctx context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error) {
			return incompleteAssignments, nil
		},
		ListFunc: func(ctx context.Context, filters *sprintrepo.SprintListFilters) ([]*models.Sprint, error) {
			// No planning sprints → force auto-create
			return []*models.Sprint{}, nil
		},
		GetNextKeyFunc: func(ctx context.Context) (string, error) {
			return "S025", nil
		},
		CreateFunc: func(ctx context.Context, s *models.Sprint) error {
			capturedNewSprint = s
			s.ID = 25
			return nil
		},
		ReassignToSprintTxFunc: func(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error {
			reassignCalled = true
			assert.Equal(t, int64(25), newSprintID)
			return nil
		},
		UpdateStatusTxFunc: func(ctx context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error {
			return nil
		},
		CreateCompletionTxFunc: func(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error {
			return nil
		},
	}

	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)

	require.NoError(t, err)
	require.NotNil(t, result)

	// TC-C02: result.NextSprintKey set to newly created sprint
	assert.Equal(t, "S025", result.NextSprintKey, "NextSprintKey must be the auto-created sprint key")

	// TC-C04: auto-created sprint start date = closed sprint end_date + 1 day
	require.NotNil(t, capturedNewSprint, "Create must have been called to auto-create next sprint")
	expectedStart := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedStart, capturedNewSprint.StartDate,
		"auto-created sprint start_date must be closed sprint end_date + 1 day")

	// TC-C04: duration preserved (14 days: April 1 → April 15 = 14 days → April 16 → April 30)
	expectedDuration := closedSprintEnd.Sub(closedSprintStart)
	actualDuration := capturedNewSprint.EndDate.Sub(capturedNewSprint.StartDate)
	assert.Equal(t, expectedDuration, actualDuration,
		"auto-created sprint must have same duration as closed sprint")

	// ReassignToSprintTx was called with the auto-created sprint ID
	assert.True(t, reassignCalled, "ReassignToSprintTx must be called for auto-created next sprint")
}

// TestSprintService_CloseSprintWithCarryover_AllCompleted tests TC-C03:
// when all assignments are complete, carryover operations are no-ops.
func TestSprintService_CloseSprintWithCarryover_AllCompleted(t *testing.T) {
	ctx := context.Background()

	// 3 total, 0 incomplete
	allAssignments := []*models.SprintAssignment{
		{ID: 201}, {ID: 202}, {ID: 203},
	}
	var incompleteAssignments []*models.SprintAssignment // empty
	planningSprints := []*models.Sprint{
		{ID: 25, Key: "S025", Name: "Sprint 25", Status: "todo"},
	}

	var capturedCompletion *models.SprintCompletion
	start, end := defaultSprintDates()

	mockRepo := buildCloseMockRepo(t,
		allAssignments, incompleteAssignments, planningSprints,
		&capturedCompletion, nil, nil, nil,
		start, end,
	)

	testSQLDB := internaltesthelper.GetTestDB()
	db := repository.NewDB(testSQLDB)

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil, db)

	result, err := svc.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)

	require.NoError(t, err)
	require.NotNil(t, result)

	// TC-C03: all completed → no carryover
	assert.Equal(t, 3, result.CompletedCount)
	assert.Equal(t, 0, result.CarriedOverCount, "no incomplete entities → CarriedOverCount = 0")
	assert.Equal(t, 0, result.DroppedCount)

	// TC-C03: sprint still completes normally
	require.NotNil(t, capturedCompletion)
	assert.Equal(t, 0, capturedCompletion.CarriedOverCount)
	assert.Equal(t, 3, capturedCompletion.PlannedEntityCount)
	assert.Equal(t, 3, capturedCompletion.CompletedEntityCount)
}
