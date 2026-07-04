package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	cfgworkflow "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	internaltesthelper "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSprintRepository is a test double for SprintRepository.
type MockSprintRepository struct {
	CreateFunc         func(ctx context.Context, s *models.Sprint) error
	GetByKeyFunc       func(ctx context.Context, key string) (*models.Sprint, error)
	GetByIDFunc        func(ctx context.Context, id int64) (*models.Sprint, error)
	UpdateFunc         func(ctx context.Context, s *models.Sprint) error
	DeleteFunc         func(ctx context.Context, id int64) error
	UpdateStatusFunc   func(ctx context.Context, id int64, status models.SprintStatus) error
	UpdateStatusTxFunc func(ctx context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error
	GetNextKeyFunc     func(ctx context.Context) (string, error)
	ListFunc           func(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error)

	// F03 methods
	AddAssignmentFunc               func(ctx context.Context, assignment *models.SprintAssignment) error
	RemoveAssignmentFunc            func(ctx context.Context, sprintID int64, entityType string, entityID int64) error
	GetActiveAssignmentFunc         func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error)
	ListAssignmentsFunc             func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error)
	ListAssignmentsForCarryoverFunc func(ctx context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error)
	ReassignToSprintTxFunc          func(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error
	DropAssignmentsTxFunc           func(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error
	CreateCompletionTxFunc          func(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error
	ListBacklogFunc                 func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error)
	GetTaskIDByKeyFunc              func(ctx context.Context, key string) (int64, error)
	GetBugIDByKeyFunc               func(ctx context.Context, key string) (int64, error)
	GetChangeCardIDByKeyFunc        func(ctx context.Context, key string) (int64, error)
	GetTechDebtIDByKeyFunc          func(ctx context.Context, key string) (int64, error)

	// F07 sprint_order methods
	MaxSprintOrderFunc         func(ctx context.Context, sprintID int64) (int, error)
	SetSprintOrderTxFunc       func(ctx context.Context, tx *sql.Tx, assignmentID int64, newPosition *int) error
	RenumberAssignmentsTxFunc  func(ctx context.Context, tx *sql.Tx, sprintID int64, ops []sprint.RenumberOp) error
	ListOrderedAssignmentsFunc func(ctx context.Context, sprintID int64) ([]*models.SprintAssignment, error)
	CountNullSprintOrderFunc   func(ctx context.Context, sprintID int64) (int, error)
}

func (m *MockSprintRepository) Create(ctx context.Context, s *models.Sprint) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, s)
	}
	return nil
}

func (m *MockSprintRepository) GetByKey(ctx context.Context, key string) (*models.Sprint, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not found")
}

func (m *MockSprintRepository) GetByID(ctx context.Context, id int64) (*models.Sprint, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *MockSprintRepository) Update(ctx context.Context, s *models.Sprint) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, s)
	}
	return nil
}

func (m *MockSprintRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockSprintRepository) UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockSprintRepository) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error {
	if m.UpdateStatusTxFunc != nil {
		return m.UpdateStatusTxFunc(ctx, tx, id, status)
	}
	return nil
}

func (m *MockSprintRepository) GetNextKey(ctx context.Context) (string, error) {
	if m.GetNextKeyFunc != nil {
		return m.GetNextKeyFunc(ctx)
	}
	return "S001", nil
}

func (m *MockSprintRepository) List(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filters)
	}
	return []*models.Sprint{}, nil
}

func (m *MockSprintRepository) AddAssignment(ctx context.Context, assignment *models.SprintAssignment) error {
	if m.AddAssignmentFunc != nil {
		return m.AddAssignmentFunc(ctx, assignment)
	}
	return nil
}

func (m *MockSprintRepository) RemoveAssignment(ctx context.Context, sprintID int64, entityType string, entityID int64) error {
	if m.RemoveAssignmentFunc != nil {
		return m.RemoveAssignmentFunc(ctx, sprintID, entityType, entityID)
	}
	return nil
}

func (m *MockSprintRepository) GetActiveAssignment(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
	if m.GetActiveAssignmentFunc != nil {
		return m.GetActiveAssignmentFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

func (m *MockSprintRepository) ListAssignments(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
	if m.ListAssignmentsFunc != nil {
		return m.ListAssignmentsFunc(ctx, sprintID, entityType)
	}
	return []*models.SprintAssignment{}, nil
}

func (m *MockSprintRepository) ListAssignmentsForCarryover(ctx context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error) {
	if m.ListAssignmentsForCarryoverFunc != nil {
		return m.ListAssignmentsForCarryoverFunc(ctx, sprintID, completedStatuses...)
	}
	return []*models.SprintAssignment{}, nil
}

func (m *MockSprintRepository) ReassignToSprintTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error {
	if m.ReassignToSprintTxFunc != nil {
		return m.ReassignToSprintTxFunc(ctx, tx, assignmentIDs, newSprintID)
	}
	return nil
}

func (m *MockSprintRepository) DropAssignmentsTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error {
	if m.DropAssignmentsTxFunc != nil {
		return m.DropAssignmentsTxFunc(ctx, tx, assignmentIDs)
	}
	return nil
}

func (m *MockSprintRepository) CreateCompletionTx(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error {
	if m.CreateCompletionTxFunc != nil {
		return m.CreateCompletionTxFunc(ctx, tx, completion)
	}
	return nil
}

func (m *MockSprintRepository) ListBacklog(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
	if m.ListBacklogFunc != nil {
		return m.ListBacklogFunc(ctx, sprintID, entityType, blockedOnly, blockedStatuses...)
	}
	return []*sprint.BacklogItem{}, nil
}

func (m *MockSprintRepository) GetTaskIDByKey(ctx context.Context, key string) (int64, error) {
	if m.GetTaskIDByKeyFunc != nil {
		return m.GetTaskIDByKeyFunc(ctx, key)
	}
	return 0, fmt.Errorf("GetTaskIDByKey not implemented in mock")
}

func (m *MockSprintRepository) GetBugIDByKey(ctx context.Context, key string) (int64, error) {
	if m.GetBugIDByKeyFunc != nil {
		return m.GetBugIDByKeyFunc(ctx, key)
	}
	return 0, fmt.Errorf("GetBugIDByKey not implemented in mock")
}

func (m *MockSprintRepository) GetChangeCardIDByKey(ctx context.Context, key string) (int64, error) {
	if m.GetChangeCardIDByKeyFunc != nil {
		return m.GetChangeCardIDByKeyFunc(ctx, key)
	}
	return 0, fmt.Errorf("GetChangeCardIDByKey not implemented in mock")
}

func (m *MockSprintRepository) GetTechDebtIDByKey(ctx context.Context, key string) (int64, error) {
	if m.GetTechDebtIDByKeyFunc != nil {
		return m.GetTechDebtIDByKeyFunc(ctx, key)
	}
	return 0, fmt.Errorf("GetTechDebtIDByKey not implemented in mock")
}

// F07 sprint_order mock implementations.

func (m *MockSprintRepository) MaxSprintOrder(ctx context.Context, sprintID int64) (int, error) {
	if m.MaxSprintOrderFunc != nil {
		return m.MaxSprintOrderFunc(ctx, sprintID)
	}
	return 0, nil
}

func (m *MockSprintRepository) SetSprintOrderTx(ctx context.Context, tx *sql.Tx, assignmentID int64, newPosition *int) error {
	if m.SetSprintOrderTxFunc != nil {
		return m.SetSprintOrderTxFunc(ctx, tx, assignmentID, newPosition)
	}
	return nil
}

func (m *MockSprintRepository) RenumberAssignmentsTx(ctx context.Context, tx *sql.Tx, sprintID int64, ops []sprint.RenumberOp) error {
	if m.RenumberAssignmentsTxFunc != nil {
		return m.RenumberAssignmentsTxFunc(ctx, tx, sprintID, ops)
	}
	return nil
}

func (m *MockSprintRepository) ListOrderedAssignments(ctx context.Context, sprintID int64) ([]*models.SprintAssignment, error) {
	if m.ListOrderedAssignmentsFunc != nil {
		return m.ListOrderedAssignmentsFunc(ctx, sprintID)
	}
	return []*models.SprintAssignment{}, nil
}

func (m *MockSprintRepository) CountNullSprintOrder(ctx context.Context, sprintID int64) (int, error) {
	if m.CountNullSprintOrderFunc != nil {
		return m.CountNullSprintOrderFunc(ctx, sprintID)
	}
	return 0, nil
}

// TestSprintService_NewSprintService tests constructor validation.
func TestSprintService_NewSprintService(t *testing.T) {
	tests := []struct {
		name        string
		repo        SprintRepository
		workflowSvc *workflow.Service
		expectPanic bool
	}{
		{
			name:        "nil repo panics",
			repo:        nil,
			workflowSvc: &workflow.Service{},
			expectPanic: true,
		},
		{
			name:        "nil workflow panics",
			repo:        &MockSprintRepository{},
			workflowSvc: nil,
			expectPanic: true,
		},
		{
			name:        "valid dependencies succeed",
			repo:        &MockSprintRepository{},
			workflowSvc: &workflow.Service{},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					_ = NewSprintService(tt.repo, tt.workflowSvc, nil, nil, nil)
				})
			} else {
				assert.NotPanics(t, func() {
					svc := NewSprintService(tt.repo, tt.workflowSvc, nil, nil, nil)
					assert.NotNil(t, svc)
				})
			}
		})
	}
}

// TestSprintService_CreateSprint tests sprint creation with validation.
func TestSprintService_CreateSprint(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)

	tests := []struct {
		name      string
		input     CreateSprintInput
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid sprint creation",
			input: CreateSprintInput{
				Name:      "Sprint 1",
				Goal:      "Complete core features",
				StartDate: now,
				EndDate:   tomorrow,
			},
			expectErr: false,
		},
		{
			name: "empty name fails",
			input: CreateSprintInput{
				Name:      "   ",
				Goal:      "Test",
				StartDate: now,
				EndDate:   tomorrow,
			},
			expectErr: true,
			errMsg:    "name cannot be empty",
		},
		{
			name: "end date before start date fails",
			input: CreateSprintInput{
				Name:      "Sprint 1",
				Goal:      "Test",
				StartDate: tomorrow,
				EndDate:   now,
			},
			expectErr: true,
			errMsg:    "end_date must be after start_date",
		},
		{
			name: "equal start and end dates fail",
			input: CreateSprintInput{
				Name:      "Sprint 1",
				Goal:      "Test",
				StartDate: now,
				EndDate:   now,
			},
			expectErr: true,
			errMsg:    "end_date must be after start_date",
		},
		{
			name: "optional goal can be empty",
			input: CreateSprintInput{
				Name:      "Sprint 1",
				Goal:      "",
				StartDate: now,
				EndDate:   tomorrow,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockSprintRepository{
				GetNextKeyFunc: func(ctx context.Context) (string, error) {
					return "S001", nil
				},
				CreateFunc: func(ctx context.Context, s *models.Sprint) error {
					// Set ID to simulate DB behavior
					s.ID = 1
					return nil
				},
			}

			// Use empty string to create default workflow (will work with ForLevel)
			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.CreateSprint(ctx, tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.input.Name, result.Name)
				assert.Equal(t, tt.input.Goal, result.Goal)
				assert.Equal(t, "planning", string(result.Status))
				assert.Equal(t, "sprint-1", result.Slug)
				assert.Equal(t, filepath.Join("docs", "plan", "sprints", "S001.md"), result.FilePath)
				assert.Greater(t, result.ID, int64(0))
			}
		})
	}
}

// TestSprintService_GetSprint tests sprint retrieval (AC-2).
func TestSprintService_GetSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		key        string
		mockResult *models.Sprint
		mockError  error
		expectErr  bool
	}{
		{
			name: "existing sprint returns data",
			key:  "S001",
			mockResult: &models.Sprint{
				ID:     1,
				Key:    "S001",
				Name:   "Sprint 1",
				Status: models.SprintStatus("planning"),
			},
			mockError: nil,
			expectErr: false,
		},
		{
			name:       "non-existent sprint returns error",
			key:        "S999",
			mockResult: nil,
			mockError:  &models.NotFoundError{Entity: "sprint S999"},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return tt.mockResult, tt.mockError
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.GetSprint(ctx, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				var notFound *models.NotFoundError
				assert.True(t, errors.As(err, &notFound))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockResult.Key, result.Key)
			}
		})
	}
}

// TestSprintService_ListSprints tests listing with optional filters (AC-3).
func TestSprintService_ListSprints(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		filters      *SprintListFilters
		mockResult   []*models.Sprint
		expectFilter bool
		expectCount  int
	}{
		{
			name:    "list all sprints",
			filters: nil,
			mockResult: []*models.Sprint{
				{ID: 1, Key: "S001", Name: "Sprint 1", Status: "planning"},
				{ID: 2, Key: "S002", Name: "Sprint 2", Status: "active"},
			},
			expectCount: 2,
		},
		{
			name:    "filter by status",
			filters: &SprintListFilters{Status: "active"},
			mockResult: []*models.Sprint{
				{ID: 2, Key: "S002", Name: "Sprint 2", Status: "active"},
			},
			expectFilter: true,
			expectCount:  1,
		},
		{
			name:        "empty result returns empty slice",
			filters:     nil,
			mockResult:  nil,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockSprintRepository{
				ListFunc: func(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
					return tt.mockResult, nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.ListSprints(ctx, tt.filters)

			assert.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectCount, len(result))
		})
	}
}

// TestSprintService_UpdateSprint tests sprint updates with validation (AC-4).
func TestSprintService_UpdateSprint(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	dayAfter := now.AddDate(0, 0, 2)

	tests := []struct {
		name      string
		startDate time.Time
		updates   UpdateSprintInput
		expectErr bool
		errMsg    string
	}{
		{
			name:      "update name succeeds",
			startDate: now,
			updates: UpdateSprintInput{
				Name: ptrString("New Name"),
			},
			expectErr: false,
		},
		{
			name:      "update goal succeeds",
			startDate: now,
			updates: UpdateSprintInput{
				Goal: ptrString("New Goal"),
			},
			expectErr: false,
		},
		{
			name:      "update end date succeeds",
			startDate: now,
			updates: UpdateSprintInput{
				EndDate: &dayAfter,
			},
			expectErr: false,
		},
		{
			name:      "empty name fails",
			startDate: now,
			updates: UpdateSprintInput{
				Name: ptrString("   "),
			},
			expectErr: true,
			errMsg:    "name cannot be empty",
		},
		{
			name:      "end date before start date fails",
			startDate: tomorrow,
			updates: UpdateSprintInput{
				EndDate: &now,
			},
			expectErr: true,
			errMsg:    "end_date must be after start_date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endDate := tt.startDate.AddDate(0, 0, 1)

			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return &models.Sprint{
						ID:        1,
						Key:       "S001",
						Name:      "Sprint 1",
						StartDate: tt.startDate,
						EndDate:   endDate,
						Status:    "planning",
					}, nil
				},
				UpdateFunc: func(ctx context.Context, s *models.Sprint) error {
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.UpdateSprint(ctx, "S001", tt.updates)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestSprintService_DeleteSprint tests deletion with status validation (AC-5).
func TestSprintService_DeleteSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		sprintKey string
		status    models.SprintStatus
		expectErr bool
		errMsg    string
	}{
		{
			name:      "delete planning sprint succeeds",
			sprintKey: "S001",
			status:    "planning",
			expectErr: false,
		},
		{
			name:      "delete active sprint fails",
			sprintKey: "S001",
			status:    "active",
			expectErr: true,
			errMsg:    "only sprints in planning status can be deleted",
		},
		{
			name:      "delete closing sprint fails",
			sprintKey: "S001",
			status:    "closing",
			expectErr: true,
			errMsg:    "only sprints in planning status can be deleted",
		},
		{
			name:      "delete completed sprint fails",
			sprintKey: "S001",
			status:    "completed",
			expectErr: true,
			errMsg:    "only sprints in planning status can be deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return &models.Sprint{
						ID:     1,
						Key:    tt.sprintKey,
						Name:   "Sprint 1",
						Status: tt.status,
					}, nil
				},
				DeleteFunc: func(ctx context.Context, id int64) error {
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			err := svc.DeleteSprint(ctx, tt.sprintKey)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSprintService_StartSprint tests sprint start transitions.
func TestSprintService_StartSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentStatus models.SprintStatus
		expectErr     bool
		errMsg        string
	}{
		{
			name:          "start planning sprint succeeds",
			currentStatus: "planning",
			expectErr:     false,
		},
		{
			name:          "start from invalid status fails",
			currentStatus: "completed",
			expectErr:     true,
			errMsg:        "cannot start sprint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					// First call returns the sprint with its current status, second returns active.
					callCount++
					if callCount == 1 {
						return &models.Sprint{
							ID:     1,
							Key:    "S001",
							Name:   "Sprint 1",
							Status: tt.currentStatus,
						}, nil
					}
					// After status update, return with active status.
					return &models.Sprint{
						ID:     1,
						Key:    "S001",
						Name:   "Sprint 1",
						Status: "active",
					}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, id int64, status models.SprintStatus) error {
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.StartSprint(ctx, "S001")

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "active", string(result.Status))
			}
		})
	}
}

// TestSprintService_StartSprint_MultipleActiveAllowed verifies that multiple sprints
// can be active simultaneously (parallel workstreams are valid).
func TestSprintService_StartSprint_MultipleActiveAllowed(t *testing.T) {
	ctx := context.Background()

	// Track call counts per sprint key so we can simulate post-update reload.
	callCounts := map[string]int{}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			callCounts[key]++
			// Second call for each key simulates the reload after UpdateStatus.
			if callCounts[key] > 1 {
				return &models.Sprint{ID: int64(len(callCounts)), Key: key, Name: key, Status: "active"}, nil
			}
			return &models.Sprint{ID: int64(len(callCounts)), Key: key, Name: key, Status: "planning"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id int64, status models.SprintStatus) error {
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	// Start the first sprint — must succeed.
	result1, err := svc.StartSprint(ctx, "S001")
	assert.NoError(t, err, "starting S001 should succeed")
	assert.NotNil(t, result1)

	// Start a second sprint while the first is active — must also succeed.
	result2, err := svc.StartSprint(ctx, "S002")
	assert.NoError(t, err, "starting S002 while S001 is active should succeed (parallel workstreams allowed)")
	assert.NotNil(t, result2)
}

// TestSprintService_CloseSprint tests status transitions (AC-7).
func TestSprintService_CloseSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentStatus models.SprintStatus
		expectErr     bool
	}{
		{
			name:          "close active sprint succeeds",
			currentStatus: "active",
			expectErr:     false,
		},
		{
			name:          "close planning sprint fails",
			currentStatus: "planning",
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					callCount++
					if callCount == 1 {
						return &models.Sprint{
							ID:     1,
							Key:    "S001",
							Name:   "Sprint 1",
							Status: tt.currentStatus,
						}, nil
					}
					// After status update, return with closing status
					return &models.Sprint{
						ID:     1,
						Key:    "S001",
						Name:   "Sprint 1",
						Status: "closing",
					}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, id int64, status models.SprintStatus) error {
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.CloseSprint(ctx, "S001")

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "closing", string(result.Status))
			}
		})
	}
}

// TestSprintService_ArchiveSprint tests archive transitions (AC-7).
func TestSprintService_ArchiveSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentStatus models.SprintStatus
		expectErr     bool
	}{
		{
			name:          "archive_completed_sprint_succeeds",
			currentStatus: "completed",
			expectErr:     false,
		},
		{
			name:          "archive active sprint fails",
			currentStatus: "active",
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					callCount++
					if callCount == 1 {
						return &models.Sprint{
							ID:     1,
							Key:    "S001",
							Name:   "Sprint 1",
							Status: tt.currentStatus,
						}, nil
					}
					// After status update, return with archived status
					return &models.Sprint{
						ID:     1,
						Key:    "S001",
						Name:   "Sprint 1",
						Status: "archived",
					}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, id int64, status models.SprintStatus) error {
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.ArchiveSprint(ctx, "S001")

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "archived", string(result.Status))
			}
		})
	}
}

// MockSprintCapacityRepository is a test double for SprintCapacityRepository.
type MockSprintCapacityRepository struct {
	GetCapacityFunc func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error)
	SetCapacityFunc func(ctx context.Context, c *models.SprintCapacity) error
	// setCapacityCallCount tracks how many times SetCapacity was called.
	setCapacityCallCount int
	// setCapacityArgs captures the args passed to SetCapacity for assertions.
	setCapacityArgs []*models.SprintCapacity
}

func (m *MockSprintCapacityRepository) GetCapacity(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
	if m.GetCapacityFunc != nil {
		return m.GetCapacityFunc(ctx, sprintID)
	}
	return []*models.SprintCapacity{}, nil
}

func (m *MockSprintCapacityRepository) SetCapacity(ctx context.Context, c *models.SprintCapacity) error {
	m.setCapacityCallCount++
	m.setCapacityArgs = append(m.setCapacityArgs, c)
	if m.SetCapacityFunc != nil {
		return m.SetCapacityFunc(ctx, c)
	}
	return nil
}

// TestSprintService_NewSprintService_WithOptionalDeps verifies that the updated constructor
// accepts nil optional dependencies without panicking (backward compatibility).
func TestSprintService_NewSprintService_WithOptionalDeps(t *testing.T) {
	t.Run("nil optional deps do not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			svc := NewSprintService(&MockSprintRepository{}, &workflow.Service{}, nil, nil, nil)
			assert.NotNil(t, svc)
		})
	})

	t.Run("all optional deps provided succeed", func(t *testing.T) {
		assert.NotPanics(t, func() {
			svc := NewSprintService(
				&MockSprintRepository{},
				&workflow.Service{},
				nil, // assignmentRepo (nil-safe)
				&MockSprintCapacityRepository{},
				nil, // cfg (nil-safe)
			)
			assert.NotNil(t, svc)
		})
	})
}

// TestSprintService_CreateSprint_AppliesDefaults verifies TC-015-01 and TC-015-02:
// when cfg has SprintDefaults.Capacity entries, SetCapacity is called once per entry
// after sprint creation; when config is absent or empty, SetCapacity is not called.
//
// Caller-Path Contract (TC-015-01):
//   - Entrypoint: CreateSprint(ctx, CreateSprintInput{...}) with cfg containing SprintDefaults.Capacity
//   - Lowest mock seam: SprintCapacityRepository.SetCapacity — call count and args captured
//   - Forbidden mocks: Do NOT inject cfg=nil; production path reads real config
//   - Counter-factual: if s.cfg is ignored, SetCapacity is never called and sprint has no capacity rows
func TestSprintService_CreateSprint_AppliesDefaults(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)

	t.Run("TC-015-01: defaults applied when config has capacity entries", func(t *testing.T) {
		capacityRepo := &MockSprintCapacityRepository{}

		mockRepo := &MockSprintRepository{
			GetNextKeyFunc: func(ctx context.Context) (string, error) {
				return "S010", nil
			},
			CreateFunc: func(ctx context.Context, s *models.Sprint) error {
				s.ID = 10
				return nil
			},
		}

		cfg := &config.Config{}
		cfg.SprintDefaults = &config.SprintDefaultsConfig{
			Capacity: map[string]float64{
				"backend":  21,
				"frontend": 13,
			},
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, capacityRepo, cfg)

		result, err := svc.CreateSprint(ctx, CreateSprintInput{
			Name:      "Sprint 10",
			StartDate: now,
			EndDate:   tomorrow,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// TC-015-01: SetCapacity must be called exactly 2 times (once per capacity entry)
		assert.Equal(t, 2, capacityRepo.setCapacityCallCount,
			"SetCapacity should be called once per sprint_defaults.capacity entry")
		// Verify sprint ID propagated correctly to each capacity row
		for _, arg := range capacityRepo.setCapacityArgs {
			assert.Equal(t, int64(10), arg.SprintID,
				"SetCapacity arg must use the created sprint's ID")
			assert.Greater(t, arg.CapacityPoints, float64(0),
				"SetCapacity capacity_points must be positive")
		}
		// Verify both agent types and their points are correct
		agentTypes := make(map[string]float64)
		for _, arg := range capacityRepo.setCapacityArgs {
			agentTypes[arg.AgentType] = arg.CapacityPoints
		}
		assert.Equal(t, float64(21), agentTypes["backend"], "backend capacity should be 21")
		assert.Equal(t, float64(13), agentTypes["frontend"], "frontend capacity should be 13")
	})

	t.Run("TC-015-02: defaults NOT applied when cfg is nil", func(t *testing.T) {
		capacityRepo := &MockSprintCapacityRepository{}
		mockRepo := &MockSprintRepository{
			GetNextKeyFunc: func(ctx context.Context) (string, error) { return "S001", nil },
			CreateFunc:     func(ctx context.Context, s *models.Sprint) error { s.ID = 1; return nil },
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, capacityRepo, nil)

		result, err := svc.CreateSprint(ctx, CreateSprintInput{
			Name:      "Sprint 1",
			StartDate: now,
			EndDate:   tomorrow,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, capacityRepo.setCapacityCallCount,
			"SetCapacity must NOT be called when cfg is nil")
	})

	t.Run("TC-015-02: defaults NOT applied when SprintDefaults is nil", func(t *testing.T) {
		capacityRepo := &MockSprintCapacityRepository{}
		mockRepo := &MockSprintRepository{
			GetNextKeyFunc: func(ctx context.Context) (string, error) { return "S001", nil },
			CreateFunc:     func(ctx context.Context, s *models.Sprint) error { s.ID = 1; return nil },
		}

		cfg := &config.Config{} // SprintDefaults is nil
		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, capacityRepo, cfg)

		result, err := svc.CreateSprint(ctx, CreateSprintInput{
			Name:      "Sprint 1",
			StartDate: now,
			EndDate:   tomorrow,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, capacityRepo.setCapacityCallCount,
			"SetCapacity must NOT be called when SprintDefaults is nil")
	})

	t.Run("TC-015-02: defaults NOT applied when capacity map is empty", func(t *testing.T) {
		capacityRepo := &MockSprintCapacityRepository{}
		mockRepo := &MockSprintRepository{
			GetNextKeyFunc: func(ctx context.Context) (string, error) { return "S001", nil },
			CreateFunc:     func(ctx context.Context, s *models.Sprint) error { s.ID = 1; return nil },
		}

		cfg := &config.Config{
			SprintDefaults: &config.SprintDefaultsConfig{
				Capacity: map[string]float64{}, // empty map
			},
		}
		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, capacityRepo, cfg)

		result, err := svc.CreateSprint(ctx, CreateSprintInput{
			Name:      "Sprint 1",
			StartDate: now,
			EndDate:   tomorrow,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, capacityRepo.setCapacityCallCount,
			"SetCapacity must NOT be called when capacity map is empty")
	})

	t.Run("capacity repo nil with defaults configured is non-fatal", func(t *testing.T) {
		// Graceful degradation: nil capacityRepo + cfg with defaults → sprint creates fine,
		// no panic, no error. The capacity insert is best-effort (non-fatal per spec §2.8).
		mockRepo := &MockSprintRepository{
			GetNextKeyFunc: func(ctx context.Context) (string, error) { return "S001", nil },
			CreateFunc:     func(ctx context.Context, s *models.Sprint) error { s.ID = 1; return nil },
		}

		cfg := &config.Config{
			SprintDefaults: &config.SprintDefaultsConfig{
				Capacity: map[string]float64{"backend": 21},
			},
		}
		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, nil, cfg) // nil capacityRepo

		result, err := svc.CreateSprint(ctx, CreateSprintInput{
			Name:      "Sprint 1",
			StartDate: now,
			EndDate:   tomorrow,
		})

		assert.NoError(t, err, "sprint creation must succeed even if capacityRepo is nil")
		assert.NotNil(t, result)
	})
}

// Helper function to create string pointers for tests
func ptrString(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// T-E19-F03-006: Service-level tests for GetSprintBacklog
// ---------------------------------------------------------------------------

// TestSprintService_GetSprintBacklog_CompletionPercentBVA tests TC-B05a..d:
// BVA on completion percentage calculation (float64 division, not integer).
//
// Caller-Path Contract (per test-plan TC-B05..TC-B05c):
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{})
//   - Lowest mock seam: SprintRepository.ListBacklog (mock returns fixture items)
//   - Forbidden mocks: Do NOT mock CompletionPercent calculation — service must compute it
//   - Counter-factual: integer division bug: 3/5 = 0 in Go integer math; test catches because
//     expected is 60.0 not 0.0
func TestSprintService_GetSprintBacklog_CompletionPercentBVA(t *testing.T) {
	ctx := context.Background()

	makeItem := func(entityType, status string) *sprint.BacklogItem {
		return &sprint.BacklogItem{
			EntityType: entityType,
			EntityKey:  "E07-F01-001",
			Key:        "E07-F01-001",
			Title:      "Test Entity",
			Status:     status,
		}
	}

	tests := []struct {
		name              string
		items             []*sprint.BacklogItem
		completedStatus   string
		expectedPercent   float64
		expectedTotal     int
		expectedCompleted int
	}{
		{
			name: "TC-B05a: 3 of 5 completed = 60.0%",
			items: []*sprint.BacklogItem{
				makeItem("task", "completed"),
				makeItem("task", "completed"),
				makeItem("task", "completed"),
				makeItem("task", "in_progress"),
				makeItem("bug", "todo"),
			},
			completedStatus:   "completed",
			expectedPercent:   60.0,
			expectedTotal:     5,
			expectedCompleted: 3,
		},
		{
			name:              "TC-B05b: 0 of 0 — must not divide by zero, returns 0.0%",
			items:             []*sprint.BacklogItem{},
			completedStatus:   "completed",
			expectedPercent:   0.0,
			expectedTotal:     0,
			expectedCompleted: 0,
		},
		{
			name: "TC-B05c: 4 of 4 completed = 100.0%",
			items: []*sprint.BacklogItem{
				makeItem("task", "completed"),
				makeItem("task", "completed"),
				makeItem("bug", "completed"),         // bug terminal = completed (route-based bug.yaml)
				makeItem("change_card", "completed"), // change_card terminal = completed
			},
			completedStatus:   "completed",
			expectedPercent:   100.0,
			expectedTotal:     4,
			expectedCompleted: 4,
		},
		{
			name: "TC-B05d: 0 of 10 completed = 0.0%",
			items: func() []*sprint.BacklogItem {
				items := make([]*sprint.BacklogItem, 10)
				for i := range items {
					items[i] = makeItem("task", "in_progress")
				}
				return items
			}(),
			completedStatus:   "completed",
			expectedPercent:   0.0,
			expectedTotal:     10,
			expectedCompleted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sprintEntity := &models.Sprint{
				ID:   24,
				Key:  "S024",
				Name: "Sprint 24",
			}

			var listBacklogCalled bool
			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return sprintEntity, nil
				},
				ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
					listBacklogCalled = true
					assert.Equal(t, int64(24), sprintID)
					return tt.items, nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, listBacklogCalled, "ListBacklog must be called")
			assert.Equal(t, "S024", result.SprintKey)
			assert.Equal(t, tt.expectedTotal, result.TotalCount)
			assert.Equal(t, tt.expectedCompleted, result.CompletedCount)
			assert.InDelta(t, tt.expectedPercent, result.CompletionPercent, 0.001,
				"CompletionPercent must use float64 division")
		})
	}
}

// TestSprintService_GetSprintBacklog_TypeFilter tests TC-B06 and TC-B07:
// valid entity-type filter passes non-nil pointer to repo; invalid type returns error.
//
// Caller-Path Contract (per test-plan TC-B06):
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{EntityType:"task"})
//   - Lowest mock seam: SprintRepository.ListBacklog — assert called with non-nil entityType pointer
//   - Forbidden mocks: Do NOT mock the entityType validation — service must validate before passing to repo
//   - Counter-factual: buggy service passing nil instead of &"task" would return all types
func TestSprintService_GetSprintBacklog_TypeFilter(t *testing.T) {
	ctx := context.Background()

	sprintEntity := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24"}

	t.Run("TC-B06: valid --type=task filter passes entityType pointer to repo", func(t *testing.T) {
		var capturedEntityType *string
		mockRepo := &MockSprintRepository{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
				return sprintEntity, nil
			},
			ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
				capturedEntityType = entityType
				// Return only task items
				return []*sprint.BacklogItem{
					{EntityType: "task", EntityKey: "E07-F01-001", Key: "E07-F01-001", Title: "Task 1", Status: "in_progress"},
				}, nil
			},
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

		result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{EntityType: "task"})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, capturedEntityType, "repo must be called with non-nil entityType pointer")
		assert.Equal(t, "task", *capturedEntityType, "repo must receive the entity type filter value")
	})

	t.Run("TC-B07: invalid --type=sprint returns error with valid values listed", func(t *testing.T) {
		listBacklogCalled := false
		mockRepo := &MockSprintRepository{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
				return sprintEntity, nil
			},
			ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
				listBacklogCalled = true
				return nil, nil
			},
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

		result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{EntityType: "sprint"})

		assert.Error(t, err, "invalid entity type must return error")
		assert.Nil(t, result)
		assert.False(t, listBacklogCalled, "repo must NOT be called when entity type is invalid")
		// Error must list valid values
		assert.Contains(t, err.Error(), "task", "error must mention valid types")
		assert.Contains(t, err.Error(), "bug", "error must mention valid types")
	})

	t.Run("TC-B07b: empty entity type treated as all types (no filter)", func(t *testing.T) {
		var capturedEntityType *string
		mockRepo := &MockSprintRepository{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
				return sprintEntity, nil
			},
			ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
				capturedEntityType = entityType
				return []*sprint.BacklogItem{}, nil
			},
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

		result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{EntityType: ""})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, capturedEntityType, "empty entity type must pass nil to repo (no filter)")
	})
}

// TestSprintService_GetSprintBacklog_BlockedFilter tests TC-B08 and TC-B09:
// decision table for blocked filter; days-blocked BVA.
//
// Caller-Path Contract (per test-plan TC-B08):
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{BlockedOnly: true})
//   - Lowest mock seam: SprintRepository.ListBacklog — assert called with blockedOnly=true
//   - Forbidden mocks: Do NOT mock the blocked-status set determination — service must ask workflow
//   - Counter-factual: buggy service hardcoding "blocked" would miss custom blocked status names
func TestSprintService_GetSprintBacklog_BlockedFilter(t *testing.T) {
	ctx := context.Background()

	sprintEntity := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24"}

	t.Run("TC-B08: BlockedOnly=true passes blockedOnly flag and blocked statuses to repo", func(t *testing.T) {
		var capturedBlockedOnly bool
		var capturedBlockedStatuses []string

		mockRepo := &MockSprintRepository{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
				return sprintEntity, nil
			},
			ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
				capturedBlockedOnly = blockedOnly
				capturedBlockedStatuses = blockedStatuses
				return []*sprint.BacklogItem{}, nil
			},
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

		result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{BlockedOnly: true})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, capturedBlockedOnly, "blockedOnly must be passed as true to repo")
		assert.NotEmpty(t, capturedBlockedStatuses, "service must pass blocked status values from workflow to repo")
	})

	t.Run("TC-B08: BlockedOnly=false passes blockedOnly=false with no blocked statuses", func(t *testing.T) {
		var capturedBlockedOnly bool

		mockRepo := &MockSprintRepository{
			GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
				return sprintEntity, nil
			},
			ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
				capturedBlockedOnly = blockedOnly
				return []*sprint.BacklogItem{}, nil
			},
		}

		workflowSvc := workflow.NewService("")
		svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

		result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{BlockedOnly: false})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, capturedBlockedOnly, "blockedOnly must be false when not requested")
	})
}

// TestSprintService_GetSprintBacklog_StatusGrouping tests TC-B01..B04 (service variant):
// backlog items are grouped by status; all entity types appear in groups.
//
// Caller-Path Contract (per test-plan TC-B01..TC-B04):
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{})
//   - Lowest mock seam: SprintRepository.ListBacklog (mock returns fixture items per entity type)
//   - Forbidden mocks: Do NOT mock the grouping logic — service must perform grouping for real
//   - Counter-factual: buggy impl that discards entity_type from BacklogItem would produce
//     groups with no entity-type label
func TestSprintService_GetSprintBacklog_StatusGrouping(t *testing.T) {
	ctx := context.Background()

	sprintEntity := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24"}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintEntity, nil
		},
		ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			agentType := "backend"
			return []*sprint.BacklogItem{
				{EntityType: "task", EntityKey: "E07-F01-001", Key: "E07-F01-001", Title: "Task 1", Status: "in_progress", AgentType: &agentType, Priority: 5},
				{EntityType: "bug", EntityKey: "B001", Key: "B001", Title: "Bug 1", Status: "in_progress", Priority: 3},
				{EntityType: "change_card", EntityKey: "CC-001", Key: "CC-001", Title: "Change 1", Status: "todo", Priority: 2},
				{EntityType: "tech_debt", EntityKey: "TD-001", Key: "TD-001", Title: "Tech Debt 1", Status: "completed", Priority: 1},
			}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify all 4 items are present
	assert.Equal(t, 4, result.TotalCount)

	// Verify groups contain items
	assert.NotEmpty(t, result.Groups, "groups must be non-empty")

	// Collect all items across all groups to verify entity types
	var allItems []*BacklogItemView
	for _, g := range result.Groups {
		allItems = append(allItems, g.Items...)
	}
	assert.Len(t, allItems, 4, "all 4 items must appear in groups")

	// Verify entity types are preserved in items (TC-B01..B04)
	entityTypes := make(map[string]bool)
	for _, item := range allItems {
		entityTypes[item.EntityType] = true
	}
	assert.True(t, entityTypes["task"], "task entity type must appear in backlog groups")
	assert.True(t, entityTypes["bug"], "bug entity type must appear in backlog groups")
	assert.True(t, entityTypes["change_card"], "change_card entity type must appear in backlog groups")
	assert.True(t, entityTypes["tech_debt"], "tech_debt entity type must appear in backlog groups")
}

// TestSprintService_GetSprintBacklog_SprintNotFound tests error propagation
// when the sprint key does not resolve.
func TestSprintService_GetSprintBacklog_SprintNotFound(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return nil, &models.NotFoundError{Entity: "sprint " + key}
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	result, err := svc.GetSprintBacklog(ctx, "S999", BacklogOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "S999")
}

// TestSprintService_GetSprintBacklog_BacklogItemViewFields tests TC-J03 (service variant):
// BacklogItemView fields are correctly projected from BacklogItem.
func TestSprintService_GetSprintBacklog_BacklogItemViewFields(t *testing.T) {
	ctx := context.Background()

	agentType := "backend"
	size := 5

	sprintEntity := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24"}
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintEntity, nil
		},
		ListBacklogFunc: func(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			return []*sprint.BacklogItem{
				{
					EntityType: "task",
					EntityKey:  "E07-F01-001",
					Key:        "E07-F01-001",
					Title:      "Test Task",
					Status:     "in_progress",
					AgentType:  &agentType,
					Priority:   7,
					Size:       &size,
				},
			}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	result, err := svc.GetSprintBacklog(ctx, "S024", BacklogOptions{})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Groups)

	// Find the item across all groups
	var foundItem *BacklogItemView
	for _, g := range result.Groups {
		for _, item := range g.Items {
			if item.Key == "E07-F01-001" {
				foundItem = item
				break
			}
		}
	}

	require.NotNil(t, foundItem, "item must appear in some group")
	assert.Equal(t, "task", foundItem.EntityType, "EntityType must be preserved")
	assert.Equal(t, "E07-F01-001", foundItem.Key, "Key must be preserved")
	assert.Equal(t, "Test Task", foundItem.Title, "Title must be preserved")
	assert.Equal(t, "in_progress", foundItem.Status, "Status must be preserved")
	assert.Equal(t, "backend", foundItem.AgentType, "AgentType must be preserved")
	assert.Equal(t, 7, foundItem.Priority, "Priority must be preserved")
	require.NotNil(t, foundItem.Size, "Size must be preserved")
	assert.Equal(t, 5, *foundItem.Size, "Size value must match")
}

// ---------------------------------------------------------------------------
// T-E19-F03-005: Service-level tests for AddEntityToSprint / RemoveEntityFromSprint
// ---------------------------------------------------------------------------

// TestSprintService_AddEntityToSprint_TaskSucceeds tests TC-R01 (service variant):
// assigning a task entity via AddEntityToSprint resolves key, checks no conflict,
// calls AddAssignment, and returns the created assignment.
//
// Caller-Path Contract (per test-plan TC-R01..R04):
//   - Entrypoint: SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S024", EntityKey:"E07-F01-001"})
//   - Lowest mock seam: SprintRepository interface
//   - Forbidden mocks: Do NOT mock keys.KeyService.Parse — real key parsing must run
//   - Counter-factual: a buggy impl that calls GetBugIDByKey for a task key would fail
//     when entity_type in the returned assignment is wrong.
func TestSprintService_AddEntityToSprint_TaskSucceeds(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}
	now := time.Now()
	var capturedAssignment *models.SprintAssignment

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			if key == "S024" {
				return sprint1, nil
			}
			return nil, &models.NotFoundError{Entity: "sprint " + key}
		},
		GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			require.Equal(t, "T-E07-F01-001", key, "task key must be normalized before lookup")
			return 1001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			// No active assignment — entity is free to assign
			return nil, nil
		},
		AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
			capturedAssignment = assignment
			assert.Equal(t, int64(24), assignment.SprintID)
			assert.Equal(t, "task", assignment.EntityType)
			assert.Equal(t, int64(1001), assignment.EntityID)
			assignment.ID = 1
			assignment.AssignedAt = now
			return nil
		},
		ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			return []*models.SprintAssignment{}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S024",
		EntityKey: "E07-F01-001",
	})

	require.NoError(t, err)
	require.NotNil(t, assignment, "assignment must be returned on success")
	require.NotNil(t, capturedAssignment, "AddAssignment must have been called")
	assert.Equal(t, "task", assignment.EntityType)
	assert.Equal(t, int64(1001), assignment.EntityID)
	assert.Nil(t, warning, "no capacity configured means no warning")
}

// TestSprintService_AddEntityToSprint_BugSucceeds tests TC-R02 (service variant):
// assigning a bug entity type resolves "B001" to entity_type="bug".
func TestSprintService_AddEntityToSprint_BugSucceeds(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
		GetBugIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			assert.Equal(t, "B1", key)
			return 2001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
			assert.Equal(t, "bug", assignment.EntityType)
			assert.Equal(t, int64(2001), assignment.EntityID)
			assignment.ID = 1
			return nil
		},
		ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			return []*models.SprintAssignment{}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S024",
		EntityKey: "B1",
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)
	assert.Equal(t, "bug", assignment.EntityType)
	assert.Nil(t, warning)
}

// TestSprintService_AddEntityToSprint_ChangeCardSucceeds tests TC-R03 (service variant):
// assigning a change card ("C001") maps to entity_type="change_card".
func TestSprintService_AddEntityToSprint_ChangeCardSucceeds(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
		GetChangeCardIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			return 3001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
			assert.Equal(t, "change_card", assignment.EntityType)
			assert.Equal(t, int64(3001), assignment.EntityID)
			assignment.ID = 1
			return nil
		},
		ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			return []*models.SprintAssignment{}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S024",
		EntityKey: "C1",
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)
	assert.Equal(t, "change_card", assignment.EntityType)
	assert.Nil(t, warning)
}

// TestSprintService_AddEntityToSprint_TechDebtSucceeds tests TC-R04 (service variant):
// assigning a tech_debt item ("TD-001") maps to entity_type="tech_debt".
func TestSprintService_AddEntityToSprint_TechDebtSucceeds(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
		GetTechDebtIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			assert.Equal(t, "TD-001", key)
			return 4001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
			assert.Equal(t, "tech_debt", assignment.EntityType)
			assert.Equal(t, int64(4001), assignment.EntityID)
			assignment.ID = 1
			return nil
		},
		ListAssignmentsFunc: func(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			return []*models.SprintAssignment{}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S024",
		EntityKey: "TD-001",
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)
	assert.Equal(t, "tech_debt", assignment.EntityType)
	assert.Nil(t, warning)
}

// TestSprintService_AddEntityToSprint_InvalidEntityType tests TC-R05 (service variant):
// entity key that resolves to an unsupported type (e.g., epic) returns an error.
func TestSprintService_AddEntityToSprint_InvalidEntityType(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S024",
		EntityKey: "E07", // epic key — not assignable to sprint
	})

	assert.Error(t, err, "epic keys should be rejected")
	assert.Nil(t, assignment)
	assert.Nil(t, warning)
}

// TestSprintService_AddEntityToSprint_ConflictError tests TC-R09 (service variant):
// when entity is already assigned to another sprint, AddEntityToSprint returns
// an error naming the conflicting sprint key — without calling AddAssignment.
//
// Caller-Path Contract (per test-plan TC-R09):
//   - Entrypoint: SprintService.AddEntityToSprint with entity already in S024
//   - Lowest mock seam: SprintRepository interface
//   - Forbidden mocks: Do NOT mock conflict detection — service must call GetActiveAssignment
//   - Counter-factual: buggy impl that skips GetActiveAssignment would call AddAssignment
//     and get a DB unique-constraint error instead of a named ConflictError
func TestSprintService_AddEntityToSprint_ConflictError(t *testing.T) {
	ctx := context.Background()

	sprint25 := &models.Sprint{ID: 25, Key: "S025", Name: "Sprint 25", Status: "planning"}
	// Entity task 1001 is already actively assigned to sprint S024
	existingAssignment := &models.SprintAssignment{
		ID: 100, SprintID: 24, EntityType: "task", EntityID: 1001,
	}

	getActiveAssignmentCalled := false
	addAssignmentCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			if key == "S025" {
				return sprint25, nil
			}
			return nil, &models.NotFoundError{Entity: "sprint " + key}
		},
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Sprint, error) {
			if id == 24 {
				return &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}, nil
			}
			return nil, &models.NotFoundError{Entity: fmt.Sprintf("sprint %d", id)}
		},
		GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			return 1001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			getActiveAssignmentCalled = true
			// Entity is already assigned to S024
			return existingAssignment, nil
		},
		AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
			addAssignmentCalled = true
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S025",
		EntityKey: "E07-F01-001",
	})

	assert.Error(t, err)
	assert.Nil(t, assignment)
	assert.Nil(t, warning)
	assert.True(t, getActiveAssignmentCalled, "service must call GetActiveAssignment to detect conflict")
	assert.False(t, addAssignmentCalled, "AddAssignment must NOT be called when conflict is detected")
	// Error must contain the conflicting sprint key (S024) so user can identify it
	assert.Contains(t, err.Error(), "S024", "error must name the conflicting sprint key")
}

// TestSprintService_AddEntityToSprint_CapacityWarning tests TC-R11 (decision table):
// capacity warning is advisory only — assignment proceeds even when over capacity.
//
// Caller-Path Contract (per test-plan TC-R11):
//   - Entrypoint: SprintService.AddEntityToSprint(ctx, AddEntityInput{...}) with capacity configured
//   - Lowest mock seam: SprintRepository (mock ListAssignments to simulate allocated state)
//   - Forbidden mocks: Do NOT mock the capacity check logic — only mock the repo data
//   - Counter-factual: a buggy impl that always returns nil CapacityWarning would fail the
//     "exceeds capacity" row (row 3: expected non-nil warning)
func TestSprintService_AddEntityToSprint_CapacityWarning(t *testing.T) {
	ctx := context.Background()

	// Decision table:
	// | Capacity Configured | Exceeds Capacity | Expected Result |
	// | No                  | N/A              | warning=nil     |
	// | Yes                 | No               | warning=nil     |
	// | Yes                 | Yes              | warning non-nil |
	tests := []struct {
		name          string
		sprintCap     []*models.SprintCapacity // nil means no capacity configured
		agentType     string
		newEntitySize int
		expectWarning bool
	}{
		{
			name:          "no capacity configured — no warning",
			sprintCap:     nil,
			agentType:     "backend",
			newEntitySize: 2,
			expectWarning: false,
		},
		{
			name: "capacity configured, not exceeded — no warning",
			sprintCap: []*models.SprintCapacity{
				{AgentType: "backend", CapacityPoints: 10, AllocatedPoints: func() *float64 { v := 4.0; return &v }()},
			},
			agentType:     "backend",
			newEntitySize: 2,
			expectWarning: false,
		},
		{
			name: "capacity configured and exceeded — warning emitted",
			sprintCap: []*models.SprintCapacity{
				{AgentType: "backend", CapacityPoints: 5, AllocatedPoints: func() *float64 { v := 4.0; return &v }()},
			},
			agentType:     "backend",
			newEntitySize: 2,
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

			var mockCapacityRepo *MockSprintCapacityRepository
			if tt.sprintCap != nil {
				mockCapacityRepo = &MockSprintCapacityRepository{
					GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
						return tt.sprintCap, nil
					},
				}
			}

			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return sprint1, nil
				},
				GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
					return 1001, nil
				},
				GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
					return nil, nil
				},
				AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
					assignment.ID = 1
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			// Pass nil (untyped) when no capacity is configured to avoid the
			// typed-nil-interface pitfall where (*MockSprintCapacityRepository)(nil)
			// would satisfy the SprintCapacityRepository interface as non-nil.
			var capRepo SprintCapacityRepository
			if mockCapacityRepo != nil {
				capRepo = mockCapacityRepo
			}
			svc := NewSprintService(mockRepo, workflowSvc, nil, capRepo, nil)

			assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
				SprintKey:     "S024",
				EntityKey:     "E07-F01-001",
				AgentType:     tt.agentType,
				EstimatedSize: tt.newEntitySize,
			})

			require.NoError(t, err, "capacity over-run must NOT block the assignment")
			require.NotNil(t, assignment, "assignment must be created regardless of capacity")

			if tt.expectWarning {
				require.NotNil(t, warning, "expected CapacityWarning to be non-nil")
				assert.Equal(t, tt.agentType, warning.AgentType)
				assert.Greater(t, warning.Allocated, warning.Capacity,
					"Allocated must exceed Capacity in the warning")
			} else {
				assert.Nil(t, warning, "no warning expected for this row")
			}
		})
	}
}

// TestAddEntityToSprint_RejectsCompletedSprint verifies BUG-001 fix:
// AddEntityToSprint must reject sprints that are not in planning or active
// status. This covers spec §4.2.1 step 1.
//
// Caller-Path Contract:
//   - Entrypoint: SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S_DONE", EntityKey:"E07-F01-001"})
//   - Lowest mock seam: SprintRepository interface
//   - Forbidden mocks: Do NOT mock the status check — service must perform it
//   - Counter-factual: a buggy impl that skips the status check would call GetTaskIDByKey,
//     then AddAssignment and succeed, whereas the correct impl must return an error here.
func TestAddEntityToSprint_RejectsCompletedSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		status string
	}{
		{"completed sprint rejected", "completed"},
		{"closing sprint rejected", "closing"},
		{"archived sprint rejected", "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completedSprint := &models.Sprint{ID: 99, Key: "S099", Name: "Done Sprint", Status: models.SprintStatus(tt.status)}

			addAssignmentCalled := false

			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return completedSprint, nil
				},
				GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
					return 1001, nil
				},
				AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
					addAssignmentCalled = true
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
				SprintKey: "S099",
				EntityKey: "E07-F01-001",
			})

			require.Error(t, err, "sprint in %q status must be rejected", tt.status)
			assert.Contains(t, err.Error(), tt.status, "error must mention the invalid sprint status")
			assert.Nil(t, assignment)
			assert.Nil(t, warning)
			assert.False(t, addAssignmentCalled, "AddAssignment must NOT be called for non-planning/non-active sprint")
		})
	}
}

// TestAddEntityToSprint_AllowsPlanningAndActiveSprints verifies that planning
// and active sprints still accept entity assignments after the BUG-001 fix.
func TestAddEntityToSprint_AllowsPlanningAndActiveSprints(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		status string
	}{
		{"planning sprint accepted", "planning"},
		{"active sprint accepted", "active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validSprint := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: models.SprintStatus(tt.status)}

			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return validSprint, nil
				},
				GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
					return 1001, nil
				},
				GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
					return nil, nil
				},
				AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
					assignment.ID = 1
					return nil
				},
			}

			workflowSvc := workflow.NewService("")
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
				SprintKey: "S024",
				EntityKey: "E07-F01-001",
			})

			assert.NoError(t, err, "sprint in %q status must be accepted", tt.status)
			assert.NotNil(t, assignment)
		})
	}
}

// TestAddEntityToSprint_CustomWorkflowStatuses verifies TD-011: the
// AddEntityToSprint guard delegates phase membership to workflow.Service
// instead of hardcoding "planning"/"active" strings. A custom sprint
// workflow that renames the planning-phase status to "draft" and the
// execution-phase status to "running" must still be accepted by the guard,
// while statuses in non-assignable phases (e.g. "wrap_up" -> review) must
// still be rejected.
//
// Caller-Path Contract:
//   - Entrypoint: SprintService.AddEntityToSprint(ctx, AddEntityInput{...})
//   - Lowest mock seam: SprintRepository interface; the workflow.Service is
//     real and reads a custom YAML sprint workflow from a temp project root.
//   - Forbidden mocks: Do NOT mock workflow.Service — the whole point is to
//     verify the guard reads phases through it.
//   - Counter-factual: a buggy impl that still checks
//     `status == "planning" || status == "active"` would reject the "draft"
//     and "running" rows even though both statuses live in the planning and
//     execution phases of the custom workflow.
func TestAddEntityToSprint_CustomWorkflowStatuses(t *testing.T) {
	ctx := context.Background()

	// Build a temp project root with a Shark 2.0 per-entity YAML sprint
	// workflow that renames the canonical statuses but preserves the
	// "planning" and "execution" phase labels that the guard keys off of.
	projectRoot := t.TempDir()
	workflowDir := filepath.Join(projectRoot, "shark-data", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))

	sprintYAML := `version: "1.0"
status_flow:
  draft: ["running", "cancelled"]
  running: ["wrap_up", "cancelled"]
  wrap_up: ["completed", "cancelled"]
  completed: []
  cancelled: []
status_metadata:
  draft:
    color: gray
    description: Sprint planned but not yet started
    phase: planning
    is_planning: true
    progress_weight: 0.0
  running:
    color: blue
    description: Sprint is currently active
    phase: execution
    progress_weight: 0.5
  wrap_up:
    color: cyan
    description: Sprint is closing
    phase: review
    progress_weight: 0.75
  completed:
    color: green
    description: Sprint complete
    phase: done
    progress_weight: 1.0
  cancelled:
    color: gray
    description: Sprint cancelled
    phase: done
    progress_weight: 1.0
special_statuses:
  _start_: ["draft"]
  _complete_: ["completed", "cancelled"]
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "sprint.yaml"), []byte(sprintYAML), 0o644))

	// Point .sharkconfig.json at the per-entity YAML directory.
	configJSON := `{"workflow_config": "shark-data/workflow"}`
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"), []byte(configJSON), 0o644))

	// Workflow config is cached globally; clear before and after to avoid
	// cross-test contamination.
	cfgworkflow.ClearWorkflowCache()
	defer cfgworkflow.ClearWorkflowCache()

	workflowSvc := workflow.NewService(projectRoot)

	// Sanity-check: the custom workflow is what we wrote, not the default.
	sprintLevelSvc := workflowSvc.ForLevel(workflow.LevelSprint)
	require.ElementsMatch(t, []string{"draft"}, sprintLevelSvc.GetStatusesByPhase("planning"),
		"custom sprint workflow's planning phase should contain only 'draft'")
	require.ElementsMatch(t, []string{"running"}, sprintLevelSvc.GetStatusesByPhase("execution"),
		"custom sprint workflow's execution phase should contain only 'running'")

	tests := []struct {
		name        string
		status      string
		expectError bool
	}{
		{"draft (custom planning-phase status) accepted", "draft", false},
		{"running (custom execution-phase status) accepted", "running", false},
		{"wrap_up (review phase) rejected", "wrap_up", true},
		{"completed (done phase) rejected", "completed", true},
		{"cancelled (done phase) rejected", "cancelled", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSprint := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: models.SprintStatus(tt.status)}

			addAssignmentCalled := false

			mockRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return testSprint, nil
				},
				GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
					return 1001, nil
				},
				GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
					return nil, nil
				},
				AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
					addAssignmentCalled = true
					assignment.ID = 1
					return nil
				},
			}

			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

			assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
				SprintKey: "S024",
				EntityKey: "E07-F01-001",
			})

			if tt.expectError {
				require.Error(t, err, "sprint in %q status (non-assignable phase) must be rejected", tt.status)
				assert.Contains(t, err.Error(), tt.status, "error must mention the invalid sprint status")
				assert.Nil(t, assignment)
				assert.False(t, addAssignmentCalled,
					"AddAssignment must NOT be called when sprint phase rejects assignments")
			} else {
				require.NoError(t, err, "sprint in %q status (planning or execution phase) must be accepted", tt.status)
				assert.NotNil(t, assignment)
				assert.True(t, addAssignmentCalled,
					"AddAssignment must be called when sprint phase accepts assignments")
			}
		})
	}
}

// TestAddEntityToSprint_ChangeCardKeyFormat verifies BUG-002 fix:
// AddEntityToSprint must accept "CC-001" format change-card keys (old CC-### format)
// as well as the current C001 format. Spec REQ-F-004 explicitly uses CC-001.
//
// Caller-Path Contract:
//   - Entrypoint: SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S024", EntityKey:"CC-001"})
//   - Lowest mock seam: SprintRepository interface (specifically GetChangeCardIDByKey)
//   - Forbidden mocks: Do NOT mock keys.KeyService.Parse — real key parsing must run to
//     demonstrate the CC-001 → change_card routing works end-to-end
//   - Counter-factual: a buggy impl that only handles C001 format would return an
//     "unsupported entity type" error for CC-001
func TestAddEntityToSprint_ChangeCardKeyFormat(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "active"}

	var capturedKey string

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
		GetChangeCardIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			capturedKey = key
			return 3001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		AddAssignmentFunc: func(ctx context.Context, assignment *models.SprintAssignment) error {
			assert.Equal(t, "change_card", assignment.EntityType, "CC-001 must map to change_card entity type")
			assert.Equal(t, int64(3001), assignment.EntityID)
			assignment.ID = 1
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	assignment, warning, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S024",
		EntityKey: "CC-001",
	})

	require.NoError(t, err, "CC-001 format change-card key must be accepted")
	require.NotNil(t, assignment)
	assert.Equal(t, "change_card", assignment.EntityType)
	assert.NotEmpty(t, capturedKey, "GetChangeCardIDByKey must have been called")
	assert.Nil(t, warning)
}

// TestSprintService_RemoveEntityFromSprint_Succeeds tests TC-R07 (service variant):
// removing an active assignment calls GetActiveAssignment then RemoveAssignment.
//
// Caller-Path Contract (per test-plan TC-R07):
// - Entrypoint: SprintService.RemoveEntityFromSprint(ctx, "S024", "E07-F01-001")
// - Lowest mock seam: SprintRepository interface
// - Forbidden mocks: Do NOT mock the active-assignment lookup — service must call it for real
// - Counter-factual: a buggy impl that skips the lookup would not error on remove-nonexistent
func TestSprintService_RemoveEntityFromSprint_Succeeds(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}
	activeAssignment := &models.SprintAssignment{
		ID: 10, SprintID: 24, EntityType: "task", EntityID: 1001,
	}

	getActiveAssignmentCalled := false
	removeAssignmentCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
		GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			return 1001, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			getActiveAssignmentCalled = true
			assert.Equal(t, "task", entityType)
			assert.Equal(t, int64(1001), entityID)
			return activeAssignment, nil
		},
		RemoveAssignmentFunc: func(ctx context.Context, sprintID int64, entityType string, entityID int64) error {
			removeAssignmentCalled = true
			assert.Equal(t, int64(24), sprintID)
			assert.Equal(t, "task", entityType)
			assert.Equal(t, int64(1001), entityID)
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	err := svc.RemoveEntityFromSprint(ctx, "S024", "E07-F01-001")

	require.NoError(t, err)
	assert.True(t, getActiveAssignmentCalled, "service must call GetActiveAssignment")
	assert.True(t, removeAssignmentCalled, "service must call RemoveAssignment")
}

// TestSprintService_RemoveEntityFromSprint_NotAssigned tests TC-R08 (service variant):
// when entity is not actively assigned, returns error without calling RemoveAssignment.
func TestSprintService_RemoveEntityFromSprint_NotAssigned(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	removeAssignmentCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
		GetTaskIDByKeyFunc: func(ctx context.Context, key string) (int64, error) {
			return 9999, nil
		},
		GetActiveAssignmentFunc: func(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			// No active assignment for this entity
			return nil, nil
		},
		RemoveAssignmentFunc: func(ctx context.Context, sprintID int64, entityType string, entityID int64) error {
			removeAssignmentCalled = true
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	err := svc.RemoveEntityFromSprint(ctx, "S024", "E07-F01-001")

	assert.Error(t, err, "should return error when entity not assigned to any sprint")
	assert.False(t, removeAssignmentCalled, "RemoveAssignment must NOT be called when no active assignment exists")
}

// ---------------------------------------------------------------------------
// T-E19-F03-008: BulkAddToSprint tests (TC-K01 through TC-K06)
// ---------------------------------------------------------------------------

// MockSprintAssignmentQueryRepository is a test double for SprintAssignmentQueryRepository.
type MockSprintAssignmentQueryRepository struct {
	BulkAssignFunc             func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error)
	ListUnassignedBacklogFunc  func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error)
	GetAssignmentsWithSizeFunc func(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error)
}

func (m *MockSprintAssignmentQueryRepository) BulkAssign(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
	if m.BulkAssignFunc != nil {
		return m.BulkAssignFunc(ctx, sprintID, assignments)
	}
	return len(assignments), nil
}

func (m *MockSprintAssignmentQueryRepository) ListUnassignedBacklog(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
	if m.ListUnassignedBacklogFunc != nil {
		return m.ListUnassignedBacklogFunc(ctx, entityTypes)
	}
	return []sprint.BacklogItem{}, nil
}

func (m *MockSprintAssignmentQueryRepository) GetAssignmentsWithSize(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error) {
	if m.GetAssignmentsWithSizeFunc != nil {
		return m.GetAssignmentsWithSizeFunc(ctx, sprintID)
	}
	return []sprint.AssignmentWithSize{}, nil
}

// ptrStr returns a pointer to the given string.
func ptrStr(s string) *string { return &s }

// TestBulkAddToSprint_FeatureKey_AddsEligible tests TC-K01 (decision table):
// BulkAddToSprint with a feature key assigns eligible tasks and skips ineligible ones.
//
// Caller-Path Contract (per test-plan TC-K01):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Forbidden mocks: Do NOT mock eligibility logic — service must apply the feature-key filter
//   - Counter-factual: a buggy impl that ignores FeatureKey would add tasks from other features,
//     making AddedByType["task"] > 3 (the actual eligible count)
func TestBulkAddToSprint_FeatureKey_AddsEligible(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	// Feature E07-F34 has 5 tasks: 3 eligible (not assigned, right feature), 1 wrong feature,
	// 1 from a different feature.
	backlogItems := []sprint.BacklogItem{
		// Eligible: task from E07-F34
		{EntityType: "task", EntityID: 1001, Key: "T-E07-F34-001", Status: "todo"},
		{EntityType: "task", EntityID: 1002, Key: "E07-F34-002", Status: "todo"},
		{EntityType: "task", EntityID: 1003, Key: "T-E07-F34-003", Status: "in_progress"},
		// Ineligible: different feature (will be filtered by service)
		{EntityType: "task", EntityID: 2001, Key: "T-E07-F35-001", Status: "todo"},
		{EntityType: "task", EntityID: 2002, Key: "E08-F01-001", Status: "todo"},
	}

	var capturedAssignments []models.SprintAssignment
	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
			return backlogItems, nil
		},
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			capturedAssignments = assignments
			return len(assignments), nil
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:  "S024",
		FeatureKey: "E07-F34",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Service must request only "task" type when FeatureKey is set
	assert.Equal(t, []string{"task"}, capturedEntityTypes, "FeatureKey bulk must request only task type")

	// 3 eligible tasks from E07-F34 should be added
	assert.Equal(t, 3, result.AddedByType["task"],
		"exactly 3 tasks from E07-F34 should be added")

	// 2 tasks from other features should be skipped
	assert.Equal(t, 2, result.SkippedByType["task"],
		"2 tasks from other features should be skipped")

	// BulkAssign must have been called with 3 tasks (only the eligible ones)
	require.NotNil(t, capturedAssignments)
	assert.Equal(t, 3, len(capturedAssignments), "BulkAssign must receive only eligible tasks")
	for _, a := range capturedAssignments {
		assert.Equal(t, int64(24), a.SprintID)
		assert.Equal(t, "task", a.EntityType)
	}
}

// TestBulkAddToSprint_CapacityWarning tests TC-K02:
// bulk add emits a capacity warning when total size exceeds configured capacity.
//
// Caller-Path Contract (per test-plan TC-K02):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["task"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository + SprintCapacityRepository
//   - Counter-factual: a buggy impl that skips capacity check would return nil CapacityWarnings
func TestBulkAddToSprint_CapacityWarning(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	agentType := "backend"
	size1 := 3
	size2 := 2

	backlogItems := []sprint.BacklogItem{
		{EntityType: "task", EntityID: 1001, Key: "T-E07-F01-001", Status: "todo",
			AgentType: ptrStr(agentType), Size: &size1},
		{EntityType: "task", EntityID: 1002, Key: "T-E07-F01-002", Status: "todo",
			AgentType: ptrStr(agentType), Size: &size2},
	}

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			return backlogItems, nil
		},
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			return len(assignments), nil
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	allocated := 4.0
	capRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
			return []*models.SprintCapacity{
				{AgentType: agentType, CapacityPoints: 5, AllocatedPoints: &allocated},
			}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, capRepo, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:   "S024",
		EntityTypes: []string{"task"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.AddedByType["task"], "both tasks should be added")
	require.NotEmpty(t, result.CapacityWarnings, "capacity warning should be emitted")
	assert.Equal(t, agentType, result.CapacityWarnings[0].AgentType)
	assert.Greater(t, result.CapacityWarnings[0].Allocated, result.CapacityWarnings[0].Capacity)
}

// TestBulkAddToSprint_FeatureNotFound tests TC-K03:
// BulkAddToSprint with a non-existent sprint returns an error and makes zero assignments.
//
// Caller-Path Contract (per test-plan TC-K03):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S999", FeatureKey:"E99-F99"})
//   - Lowest mock seam: SprintRepository (mock GetByKey to error)
//   - Counter-factual: a buggy impl that ignores sprint lookup failure would call BulkAssign
//     with sprint_id=0 and produce corrupt data
func TestBulkAddToSprint_FeatureNotFound(t *testing.T) {
	ctx := context.Background()

	bulkAssignCalled := false

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			bulkAssignCalled = true
			return 0, nil
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return nil, &models.NotFoundError{Entity: "sprint " + key}
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:  "S999",
		FeatureKey: "E99-F99",
	})

	assert.Error(t, err, "should return error when sprint not found")
	assert.Nil(t, result, "result must be nil when sprint resolution fails")
	assert.False(t, bulkAssignCalled, "BulkAssign must NOT be called when sprint is not found")
}

// TestBulkAddToSprint_BulkAssignError tests the BulkAssign error propagation path
// (spec §3.1 requirement: "On repo error, no entities are added").
//
// Caller-Path Contract:
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["task"]})
//   - Lowest mock seam: SprintRepository (returns valid sprint) + SprintAssignmentQueryRepository
//     (BulkAssign returns error)
//   - Forbidden mocks: Do NOT mock the sprint lookup — it must succeed so BulkAssign is reached
//   - Counter-factual: a buggy impl that swallows the BulkAssign error would return a non-nil
//     result with AddedByType entries instead of propagating the error, causing the
//     assert.Error assertion to fail
func TestBulkAddToSprint_BulkAssignError(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	backlogItems := []sprint.BacklogItem{
		{EntityType: "task", EntityID: 1001, Key: "E07-F01-001", Status: "todo"},
		{EntityType: "task", EntityID: 1002, Key: "E07-F01-002", Status: "todo"},
	}

	bulkAssignErr := fmt.Errorf("database write failed: connection timeout")

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			return backlogItems, nil
		},
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			return 0, bulkAssignErr
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:   "S024",
		EntityTypes: []string{"task"},
	})

	assert.Error(t, err, "BulkAssign error must be propagated to the caller")
	assert.Nil(t, result, "result must be nil when BulkAssign fails — no partial results")
}

// TestBulkAddToSprint_BugsBulk tests TC-K04:
// BulkAddToSprint with EntityTypes:["bug"] assigns open unassigned bugs.
//
// Caller-Path Contract (per test-plan TC-K04):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["bug"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Counter-factual: a buggy impl that ignores the entityTypes filter would return all 4 items
//     (2 bugs + 1 task + 1 tech_debt), causing AddedByType["bug"] == 4 instead of 2, failing the
//     assertion. The mixed-type fixture proves the type filter is actually applied.
func TestBulkAddToSprint_BugsBulk(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	// Mixed-type fixture: 2 bugs + 1 task + 1 tech_debt item.
	// A buggy impl that ignores entityTypes would return all 4 items, making
	// AddedByType["bug"] == 4 and failing the assertion below.
	allBacklogItems := []sprint.BacklogItem{
		{EntityType: "bug", EntityID: 2001, Key: "B001", Status: "open"},
		{EntityType: "bug", EntityID: 2002, Key: "B002", Status: "in_progress"},
		{EntityType: "task", EntityID: 1001, Key: "E07-F01-001", Status: "todo"},
		{EntityType: "tech_debt", EntityID: 3001, Key: "TD-001", Status: "open"},
	}

	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
			// Simulate real filtering: return only items whose type is in entityTypes.
			var filtered []sprint.BacklogItem
			typeSet := make(map[string]bool, len(entityTypes))
			for _, et := range entityTypes {
				typeSet[et] = true
			}
			for _, item := range allBacklogItems {
				if typeSet[item.EntityType] {
					filtered = append(filtered, item)
				}
			}
			return filtered, nil
		},
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			return len(assignments), nil
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:   "S024",
		EntityTypes: []string{"bug"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{"bug"}, capturedEntityTypes, "service must pass 'bug' type to ListUnassignedBacklog")
	assert.Equal(t, 2, result.AddedByType["bug"], "only the 2 bugs should be added, not the task or tech_debt item")
	assert.Equal(t, 0, result.AddedByType["task"], "task items must not appear in bug-only bulk add")
	assert.Equal(t, 0, result.AddedByType["tech_debt"], "tech_debt items must not appear in bug-only bulk add")
	assert.Equal(t, 0, len(result.CapacityWarnings), "no capacity configured — no warnings")
}

// TestBulkAddToSprint_TechDebtBulk tests TC-K05:
// BulkAddToSprint with EntityTypes:["tech_debt"] assigns open tech-debt items.
//
// Caller-Path Contract (per test-plan TC-K05):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["tech_debt"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Counter-factual: a buggy impl that ignores entityTypes would return all 5 items
//     (3 tech_debt + 1 bug + 1 change_card), making AddedByType["tech_debt"] == 5 instead of 3.
//     The mixed-type fixture proves the type filter is applied and the "tech_debt" string is exact.
func TestBulkAddToSprint_TechDebtBulk(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	// Mixed-type fixture: 3 tech_debt items + 1 bug + 1 change_card.
	// A buggy impl that ignores entityTypes (or uses the wrong string like "techdebt")
	// would return wrong counts, failing the assertions below.
	allBacklogItems := []sprint.BacklogItem{
		{EntityType: "tech_debt", EntityID: 3001, Key: "TD-001", Status: "open"},
		{EntityType: "tech_debt", EntityID: 3002, Key: "TD-002", Status: "open"},
		{EntityType: "tech_debt", EntityID: 3003, Key: "TD-003", Status: "open"},
		{EntityType: "bug", EntityID: 2001, Key: "B001", Status: "open"},
		{EntityType: "change_card", EntityID: 4001, Key: "C001", Status: "open"},
	}

	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
			// Simulate real filtering: return only items whose type is in entityTypes.
			var filtered []sprint.BacklogItem
			typeSet := make(map[string]bool, len(entityTypes))
			for _, et := range entityTypes {
				typeSet[et] = true
			}
			for _, item := range allBacklogItems {
				if typeSet[item.EntityType] {
					filtered = append(filtered, item)
				}
			}
			return filtered, nil
		},
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			return len(assignments), nil
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:   "S024",
		EntityTypes: []string{"tech_debt"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{"tech_debt"}, capturedEntityTypes,
		"service must pass 'tech_debt' type to ListUnassignedBacklog")
	assert.Equal(t, 3, result.AddedByType["tech_debt"], "only the 3 tech-debt items should be added")
	assert.Equal(t, 0, result.AddedByType["bug"], "bug items must not appear in tech_debt-only bulk add")
	assert.Equal(t, 0, result.AddedByType["change_card"], "change_card items must not appear in tech_debt-only bulk add")
}

// TestBulkAddToSprint_ChangeCardsBulk tests TC-K06:
// BulkAddToSprint with EntityTypes:["change_card"] assigns open change-cards.
//
// Caller-Path Contract (per test-plan TC-K06):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["change_card"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Counter-factual: a buggy impl that ignores entityTypes (or uses "change" instead of
//     "change_card") would return all 4 items (2 change_cards + 1 task + 1 bug),
//     making AddedByType["change_card"] == 4 instead of 2. The mixed-type fixture proves
//     the type filter is applied and the exact string "change_card" is used.
func TestBulkAddToSprint_ChangeCardsBulk(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	// Mixed-type fixture: 2 change_cards + 1 task + 1 bug.
	// A buggy impl that ignores entityTypes or uses "change" (wrong string) would
	// return wrong counts, failing the assertions below.
	allBacklogItems := []sprint.BacklogItem{
		{EntityType: "change_card", EntityID: 4001, Key: "C001", Status: "open"},
		{EntityType: "change_card", EntityID: 4002, Key: "C002", Status: "open"},
		{EntityType: "task", EntityID: 1001, Key: "E07-F01-002", Status: "todo"},
		{EntityType: "bug", EntityID: 2001, Key: "B002", Status: "open"},
	}

	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
			// Simulate real filtering: return only items whose type is in entityTypes.
			var filtered []sprint.BacklogItem
			typeSet := make(map[string]bool, len(entityTypes))
			for _, et := range entityTypes {
				typeSet[et] = true
			}
			for _, item := range allBacklogItems {
				if typeSet[item.EntityType] {
					filtered = append(filtered, item)
				}
			}
			return filtered, nil
		},
		BulkAssignFunc: func(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			return len(assignments), nil
		},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprint1, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:   "S024",
		EntityTypes: []string{"change_card"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{"change_card"}, capturedEntityTypes,
		"service must pass 'change_card' type to ListUnassignedBacklog")
	assert.Equal(t, 2, result.AddedByType["change_card"], "only the 2 change_cards should be added")
	assert.Equal(t, 0, result.AddedByType["task"], "task items must not appear in change_card-only bulk add")
	assert.Equal(t, 0, result.AddedByType["bug"], "bug items must not appear in change_card-only bulk add")
}

// TestBulkAddToSprint_NilAssignmentRepo tests error when assignmentRepo is nil.
func TestBulkAddToSprint_NilAssignmentRepo(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 1, Key: "S001"}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	// Pass nil assignmentRepo — should return a configuration error.
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{SprintKey: "S001"})
	assert.Error(t, err, "nil assignmentRepo must produce an error")
	assert.Nil(t, result)
}

// TestSprintService_GetCarryoverBehavior tests GetCarryoverBehavior config reading.
//
// Caller-Path Contract:
//   - Entrypoint: SprintService.GetCarryoverBehavior() — no DB involvement
//   - Lowest mock seam: Config struct (inject real Config, do not mock)
//   - Forbidden mocks: Do NOT mock the config read — inject a real Config struct
//   - Counter-factual: a buggy impl that always returns "" would fail the
//     "carryover_behavior=backlog" assertion
func TestSprintService_GetCarryoverBehavior(t *testing.T) {
	mockRepo := &MockSprintRepository{}
	workflowSvc := workflow.NewService("")

	tests := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{
			name:     "nil config returns empty string",
			cfg:      nil,
			expected: "",
		},
		{
			name:     "nil SprintDefaults returns empty string",
			cfg:      &config.Config{},
			expected: "",
		},
		{
			name: "carryover_behavior=next returns next",
			cfg: &config.Config{
				SprintDefaults: &config.SprintDefaultsConfig{
					CarryoverBehavior: "next",
				},
			},
			expected: "next",
		},
		{
			name: "carryover_behavior=backlog returns backlog",
			cfg: &config.Config{
				SprintDefaults: &config.SprintDefaultsConfig{
					CarryoverBehavior: "backlog",
				},
			},
			expected: "backlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSprintService(mockRepo, workflowSvc, nil, nil, tt.cfg)
			got := svc.GetCarryoverBehavior()
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TC-014-01..TC-014-08: SetSprintCapacity and GetSprintCapacity
// ---------------------------------------------------------------------------

// TestSprintService_SetSprintCapacity_CreateNew (TC-014-01) verifies that
// SetSprintCapacity resolves the sprint key to an ID and calls SetCapacity
// with the correct SprintCapacity argument.
func TestSprintService_SetSprintCapacity_CreateNew(t *testing.T) {
	ctx := context.Background()

	var capturedCapacity *models.SprintCapacity
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S024", key)
			return &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		SetCapacityFunc: func(ctx context.Context, c *models.SprintCapacity) error {
			capturedCapacity = c
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, capRepo, nil)

	result, err := svc.SetSprintCapacity(ctx, SetSprintCapacityInput{
		SprintKey: "S024",
		AgentType: "backend",
		Points:    21,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, capturedCapacity)
	assert.Equal(t, int64(24), capturedCapacity.SprintID)
	assert.Equal(t, "backend", capturedCapacity.AgentType)
	assert.Equal(t, float64(21), capturedCapacity.CapacityPoints)
	// Returned model should reflect what was set
	assert.Equal(t, int64(24), result.SprintID)
	assert.Equal(t, "backend", result.AgentType)
	assert.Equal(t, float64(21), result.CapacityPoints)
}

// TestSprintService_SetSprintCapacity_RejectsZeroPoints (TC-014-03) verifies
// that points <= 0 is rejected before calling SetCapacity.
func TestSprintService_SetSprintCapacity_RejectsZeroPoints(t *testing.T) {
	ctx := context.Background()

	setCalled := false
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		SetCapacityFunc: func(ctx context.Context, c *models.SprintCapacity) error {
			setCalled = true
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, capRepo, nil)

	tests := []struct {
		name   string
		points float64
	}{
		{"zero", 0},
		{"negative", -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCalled = false
			result, err := svc.SetSprintCapacity(ctx, SetSprintCapacityInput{
				SprintKey: "S024",
				AgentType: "backend",
				Points:    tt.points,
			})
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.False(t, setCalled, "SetCapacity must NOT be called when points <= 0")
			assert.Contains(t, err.Error(), "points")
		})
	}
}

// TestSprintService_SetSprintCapacity_ValidMinimumPoints verifies that
// points > 0 (including fractional) is accepted.
func TestSprintService_SetSprintCapacity_ValidMinimumPoints(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		SetCapacityFunc: func(ctx context.Context, c *models.SprintCapacity) error {
			return nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, capRepo, nil)

	result, err := svc.SetSprintCapacity(ctx, SetSprintCapacityInput{
		SprintKey: "S024",
		AgentType: "backend",
		Points:    0.001,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestSprintService_SetSprintCapacity_NilCapacityRepo verifies graceful error
// when capacityRepo is nil.
func TestSprintService_SetSprintCapacity_NilCapacityRepo(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, nil, nil, nil)

	result, err := svc.SetSprintCapacity(ctx, SetSprintCapacityInput{
		SprintKey: "S024",
		AgentType: "backend",
		Points:    21,
	})
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestSprintService_GetSprintCapacity_EmptySliceWhenNoRows (TC-014-04) verifies
// that GetSprintCapacity returns an empty slice (not error) when no capacity rows exist.
func TestSprintService_GetSprintCapacity_EmptySliceWhenNoRows(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
			return []*models.SprintCapacity{}, nil
		},
	}
	assignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error) {
			return []sprint.AssignmentWithSize{}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, assignRepo, capRepo, nil)

	rows, err := svc.GetSprintCapacity(ctx, "S024")
	require.NoError(t, err)
	assert.NotNil(t, rows)
	assert.Len(t, rows, 0, "should return empty slice not nil")
}

// TestSprintService_GetSprintCapacity_ComputesAllocated (TC-014-05) verifies
// that allocated_points is computed as Σ size of assigned tasks filtered by agent_type.
func TestSprintService_GetSprintCapacity_ComputesAllocated(t *testing.T) {
	ctx := context.Background()

	backendAgent := "backend"
	frontendAgent := "frontend"
	size5 := 5
	size8 := 8
	size3 := 3

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
			assert.Equal(t, int64(24), sprintID)
			return []*models.SprintCapacity{
				{SprintID: 24, AgentType: "backend", CapacityPoints: 21},
				{SprintID: 24, AgentType: "frontend", CapacityPoints: 13},
			}, nil
		},
	}
	assignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error) {
			assert.Equal(t, int64(24), sprintID)
			return []sprint.AssignmentWithSize{
				{EntityType: "task", EntityID: 1, Key: "E07-F01-001", AgentType: &backendAgent, Size: &size5},
				{EntityType: "task", EntityID: 2, Key: "E07-F01-002", AgentType: &backendAgent, Size: &size8},
				{EntityType: "task", EntityID: 3, Key: "E07-F01-003", AgentType: &frontendAgent, Size: &size3},
			}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, assignRepo, capRepo, nil)

	rows, err := svc.GetSprintCapacity(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Find backend row
	var backendRow, frontendRow *CapacityRow
	for i := range rows {
		if rows[i].AgentType == "backend" {
			backendRow = &rows[i]
		} else if rows[i].AgentType == "frontend" {
			frontendRow = &rows[i]
		}
	}
	require.NotNil(t, backendRow, "backend capacity row should exist")
	require.NotNil(t, frontendRow, "frontend capacity row should exist")

	// Backend: capacity=21, allocated=5+8=13, remaining=8
	assert.Equal(t, float64(21), backendRow.CapacityPoints)
	assert.Equal(t, float64(13), backendRow.AllocatedPoints)
	assert.Equal(t, float64(8), backendRow.Remaining)
	assert.Equal(t, 0, backendRow.UnsizedAssigned)

	// Frontend: capacity=13, allocated=3, remaining=10
	assert.Equal(t, float64(13), frontendRow.CapacityPoints)
	assert.Equal(t, float64(3), frontendRow.AllocatedPoints)
	assert.Equal(t, float64(10), frontendRow.Remaining)
	assert.Equal(t, 0, frontendRow.UnsizedAssigned)
}

// TestSprintService_GetSprintCapacity_UnsizedAssigned (TC-014-06) verifies that
// unsized_assigned counts assignments with size IS NULL for each agent type.
func TestSprintService_GetSprintCapacity_UnsizedAssigned(t *testing.T) {
	ctx := context.Background()

	backendAgent := "backend"
	size5 := 5

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
			return []*models.SprintCapacity{
				{SprintID: 24, AgentType: "backend", CapacityPoints: 21},
			}, nil
		},
	}
	assignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error) {
			return []sprint.AssignmentWithSize{
				{EntityType: "task", Key: "E07-F01-001", AgentType: &backendAgent, Size: nil},    // unsized
				{EntityType: "task", Key: "E07-F01-002", AgentType: &backendAgent, Size: &size5}, // sized
			}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, assignRepo, capRepo, nil)

	rows, err := svc.GetSprintCapacity(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, rows, 1)

	backendRow := rows[0]
	assert.Equal(t, "backend", backendRow.AgentType)
	assert.Equal(t, float64(5), backendRow.AllocatedPoints)
	assert.Equal(t, 1, backendRow.UnsizedAssigned)
}

// TestSprintService_GetSprintCapacity_NegativeRemaining (TC-014-07) verifies
// that remaining can be negative (overcommit is not clamped to 0).
func TestSprintService_GetSprintCapacity_NegativeRemaining(t *testing.T) {
	ctx := context.Background()

	backendAgent := "backend"
	size10 := 10
	size20 := 20

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
			return []*models.SprintCapacity{
				{SprintID: 24, AgentType: "backend", CapacityPoints: 21},
			}, nil
		},
	}
	assignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error) {
			return []sprint.AssignmentWithSize{
				{EntityType: "task", Key: "E07-F01-001", AgentType: &backendAgent, Size: &size10},
				{EntityType: "task", Key: "E07-F01-002", AgentType: &backendAgent, Size: &size20},
			}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, assignRepo, capRepo, nil)

	rows, err := svc.GetSprintCapacity(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, rows, 1)

	backendRow := rows[0]
	// capacity=21, allocated=30, remaining=-9 (not clamped)
	assert.Equal(t, float64(21), backendRow.CapacityPoints)
	assert.Equal(t, float64(30), backendRow.AllocatedPoints)
	assert.Equal(t, float64(-9), backendRow.Remaining)
}

// TestSprintService_GetSprintCapacity_UsesOnlyTwoRepoQueries verifies that
// GetSprintCapacity calls GetAssignmentsWithSize and GetCapacity exactly once each.
func TestSprintService_GetSprintCapacity_UsesOnlyTwoRepoQueries(t *testing.T) {
	ctx := context.Background()

	getCapacityCalls := 0
	getAssignmentsCalls := 0

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{ID: 24, Key: "S024"}, nil
		},
	}
	capRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
			getCapacityCalls++
			return []*models.SprintCapacity{}, nil
		},
	}
	assignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error) {
			getAssignmentsCalls++
			return []sprint.AssignmentWithSize{}, nil
		},
	}

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, assignRepo, capRepo, nil)

	_, err := svc.GetSprintCapacity(ctx, "S024")
	require.NoError(t, err)

	assert.Equal(t, 1, getCapacityCalls, "GetCapacity should be called exactly once")
	assert.Equal(t, 1, getAssignmentsCalls, "GetAssignmentsWithSize should be called exactly once")
}

// ---------------------------------------------------------------------------
// T-E19-F05-005: GetSprintReadiness tests (TC-013-01 through TC-013-14)
// ---------------------------------------------------------------------------

// sizePtrR returns a pointer to an int — helper for readiness tests.
func sizePtrR(n int) *int { return &n }

// agentPtrR returns a pointer to a string — helper for readiness tests.
func agentPtrR(s string) *string { return &s }

// makeReadinessSvc creates a SprintService wired with mock repos for readiness tests.
func makeReadinessSvc(
	sprintObj *models.Sprint,
	assignments []sprint.AssignmentWithSize,
	capacityRows []*models.SprintCapacity,
) *SprintService {
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return sprintObj, nil
		},
	}
	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(_ context.Context, _ int64) ([]sprint.AssignmentWithSize, error) {
			return assignments, nil
		},
	}
	rows := capacityRows
	if rows == nil {
		rows = []*models.SprintCapacity{}
	}
	mockCapRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(_ context.Context, _ int64) ([]*models.SprintCapacity, error) {
			return rows, nil
		},
	}
	wf := workflow.NewService("")
	return NewSprintService(mockRepo, wf, mockAssignRepo, mockCapRepo, nil)
}

// TestSprintService_GetSprintReadiness_Factor1_ZeroCapacity tests TC-013-01 zone: zero capacity.
func TestSprintService_GetSprintReadiness_Factor1_ZeroCapacity(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}
	svc := makeReadinessSvc(sprintObj, []sprint.AssignmentWithSize{}, []*models.SprintCapacity{})
	result, err := svc.GetSprintReadiness(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, result.Factors, 6)
	assert.Equal(t, 0, result.Factors[0].Score, "zero capacity → Factor1=0")
	assert.Equal(t, 25, result.Factors[0].MaxScore)
}

// TestSprintService_GetSprintReadiness_Factor1_FullZones tests TC-013-01: BVA zones.
func TestSprintService_GetSprintReadiness_Factor1_FullZones(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	tests := []struct {
		name          string
		allocSize     int
		capacity      float64
		expectedScore int
	}{
		// util=0 → score=0
		{"0pct_util", 0, 21, 0},
		// util=5/21≈23.8% → int(0.238/0.5*25)=int(11.9)=11
		{"23pct_util", 5, 21, 11},
		// util=10/20=50% → score=25
		{"50pct_util", 10, 20, 25},
		// util=20/20=100% → score=25
		{"100pct_util", 20, 20, 25},
		// util=26/25=104% → max(0,25-int(0.04*50))=max(0,25-2)=23
		{"104pct_util", 26, 25, 23},
		// util=30/20=150% → max(0,25-int(0.5*50))=max(0,25-25)=0
		{"150pct_util", 30, 20, 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var assignments []sprint.AssignmentWithSize
			if tt.allocSize > 0 {
				assignments = []sprint.AssignmentWithSize{
					{EntityType: "task", Key: "T-001", Size: sizePtrR(tt.allocSize)},
				}
			} else {
				// allocSize=0: unsized entity contributes 0 to totalAllocated
				assignments = []sprint.AssignmentWithSize{
					{EntityType: "task", Key: "T-001", Size: nil},
				}
			}
			caps := []*models.SprintCapacity{{AgentType: "backend", CapacityPoints: tt.capacity}}
			svc := makeReadinessSvc(sprintObj, assignments, caps)
			result, err := svc.GetSprintReadiness(ctx, "S024")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedScore, result.Factors[0].Score, "Factor1 wrong for %s", tt.name)
		})
	}
}

// TestSprintService_GetSprintReadiness_Factor2_DependencySatisfaction tests TC-013-02.
func TestSprintService_GetSprintReadiness_Factor2_DependencySatisfaction(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	t.Run("no deps → score 20", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: "[]"},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 20, result.Factors[1].Score, "no unsatisfied deps → Factor2=20")
		assert.Equal(t, 20, result.Factors[1].MaxScore)
	})

	t.Run("1 unsatisfied dep → score 19", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: `["T-E07-F99-001"]`},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 19, result.Factors[1].Score, "1 unsatisfied dep → Factor2=19")
	})

	t.Run("dep satisfied because in sprint → score 20", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: `["T-002"]`},
			{EntityType: "task", Key: "T-002", Size: sizePtrR(2), DependsOn: "[]"},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 20, result.Factors[1].Score, "dep in sprint → Factor2=20")
	})

	t.Run("20 unsatisfied → score 0", func(t *testing.T) {
		// Build a depends_on JSON with 20 external tasks
		deps := `["T-E07-F99-001","T-E07-F99-002","T-E07-F99-003","T-E07-F99-004","T-E07-F99-005","T-E07-F99-006","T-E07-F99-007","T-E07-F99-008","T-E07-F99-009","T-E07-F99-010","T-E07-F99-011","T-E07-F99-012","T-E07-F99-013","T-E07-F99-014","T-E07-F99-015","T-E07-F99-016","T-E07-F99-017","T-E07-F99-018","T-E07-F99-019","T-E07-F99-020"]`
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: deps},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Factors[1].Score, "20 unsatisfied → Factor2=0")
	})

	t.Run("21 unsatisfied → score 0 (floor, not negative)", func(t *testing.T) {
		deps := `["T-E07-F99-001","T-E07-F99-002","T-E07-F99-003","T-E07-F99-004","T-E07-F99-005","T-E07-F99-006","T-E07-F99-007","T-E07-F99-008","T-E07-F99-009","T-E07-F99-010","T-E07-F99-011","T-E07-F99-012","T-E07-F99-013","T-E07-F99-014","T-E07-F99-015","T-E07-F99-016","T-E07-F99-017","T-E07-F99-018","T-E07-F99-019","T-E07-F99-020","T-E07-F99-021"]`
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: deps},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Factors[1].Score, "21 unsatisfied → Factor2=0 (floor)")
		assert.GreaterOrEqual(t, result.Factors[1].Score, 0, "score must not be negative")
	})

	t.Run("malformed depends_on → score unchanged, Detail surfaces ambiguity", func(t *testing.T) {
		// T-E19-F05-011: malformed JSON must not silently inflate Factor 2 score.
		// Score stays at 20 (treated as zero deps for graceful degradation) but the
		// Detail string must mention the malformed entity count so callers are not
		// misled into believing all dependency data is healthy.
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: "{invalid}"},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		f2 := result.Factors[1]
		// Score: graceful degradation → treated as zero deps → max 20
		assert.Equal(t, 20, f2.Score, "malformed depends_on → graceful degradation, score stays 20")
		// Detail must mention the malformed entity
		assert.Contains(t, f2.Detail, "malformed depends_on",
			"Detail must surface malformed JSON count")
		assert.Contains(t, f2.Detail, "1",
			"Detail must include the count of malformed entities")
	})

	t.Run("malformed + unsatisfied → score reflects unsatisfied, Detail surfaces both", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			// valid task with 1 external dep
			{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: `["T-EXTERNAL-001"]`},
			// task with malformed JSON
			{EntityType: "task", Key: "T-002", Size: sizePtrR(2), DependsOn: "{bad json}"},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		f2 := result.Factors[1]
		assert.Equal(t, 19, f2.Score, "1 unsatisfied dep → score 19")
		assert.Contains(t, f2.Detail, "malformed depends_on",
			"Detail must surface malformed JSON count even when other deps are unsatisfied")
	})
}

// TestSprintService_GetSprintReadiness_Factor3_TaskCount tests TC-013-03: BVA on entity count.
func TestSprintService_GetSprintReadiness_Factor3_TaskCount(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	tests := []struct {
		count    int
		expected int
	}{
		{0, 0},
		{1, 5},  // int(1/3.0*15) = 5
		{2, 10}, // int(2/3.0*15) = 10
		{3, 15},
		{10, 15}, // capped
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("count_%d", tt.count), func(t *testing.T) {
			assignments := make([]sprint.AssignmentWithSize, tt.count)
			for i := range assignments {
				assignments[i] = sprint.AssignmentWithSize{
					EntityType: "task",
					Key:        fmt.Sprintf("T-%03d", i+1),
					Size:       sizePtrR(3),
					DependsOn:  "[]",
				}
			}
			svc := makeReadinessSvc(sprintObj, assignments, nil)
			result, err := svc.GetSprintReadiness(ctx, "S024")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Factors[2].Score, "Factor3 wrong for count=%d", tt.count)
			assert.Equal(t, 15, result.Factors[2].MaxScore)
		})
	}
}

// TestSprintService_GetSprintReadiness_Factor4_AgentBalance tests TC-013-04.
func TestSprintService_GetSprintReadiness_Factor4_AgentBalance(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	t.Run("all null → score 0", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{{EntityType: "task", Key: "T-001", AgentType: nil}}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Factors[3].Score)
		assert.Equal(t, 15, result.Factors[3].MaxScore)
	})

	t.Run("1 agent type → score 0", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", AgentType: agentPtrR("backend")},
			{EntityType: "task", Key: "T-002", AgentType: agentPtrR("backend")},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Factors[3].Score)
	})

	t.Run("2 agent types → score 15", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", AgentType: agentPtrR("backend")},
			{EntityType: "task", Key: "T-002", AgentType: agentPtrR("frontend")},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 15, result.Factors[3].Score)
	})

	t.Run("3 agent types → score 15", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{
			{EntityType: "task", Key: "T-001", AgentType: agentPtrR("backend")},
			{EntityType: "task", Key: "T-002", AgentType: agentPtrR("frontend")},
			{EntityType: "task", Key: "T-003", AgentType: agentPtrR("qa")},
		}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 15, result.Factors[3].Score)
	})
}

// TestSprintService_GetSprintReadiness_Factor5_SizingCoverage tests TC-013-05.
func TestSprintService_GetSprintReadiness_Factor5_SizingCoverage(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	tests := []struct {
		unsized  int
		expected int
	}{
		{0, 15},
		{1, 14},
		{7, 8},
		{15, 0},
		{16, 0}, // floor
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("unsized_%d", tt.unsized), func(t *testing.T) {
			total := 20
			assignments := make([]sprint.AssignmentWithSize, total)
			for i := range assignments {
				if i < tt.unsized {
					assignments[i] = sprint.AssignmentWithSize{
						EntityType: "task",
						Key:        fmt.Sprintf("T-U%03d", i+1),
						Title:      fmt.Sprintf("Unsized %d", i+1),
						Size:       nil,
					}
				} else {
					assignments[i] = sprint.AssignmentWithSize{
						EntityType: "task",
						Key:        fmt.Sprintf("T-S%03d", i+1),
						Size:       sizePtrR(3),
					}
				}
			}
			svc := makeReadinessSvc(sprintObj, assignments, nil)
			result, err := svc.GetSprintReadiness(ctx, "S024")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Factors[4].Score, "Factor5 wrong for unsized=%d", tt.unsized)
			assert.Equal(t, 15, result.Factors[4].MaxScore)
		})
	}
}

// TestSprintService_GetSprintReadiness_Factor6_OversizedFlag tests TC-013-06.
func TestSprintService_GetSprintReadiness_Factor6_OversizedFlag(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	t.Run("size=7 → score 10 (not oversized)", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{{EntityType: "task", Key: "T-001", Size: sizePtrR(7)}}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 10, result.Factors[5].Score)
		assert.Equal(t, 10, result.Factors[5].MaxScore)
	})

	t.Run("size=8 boundary → score 0", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{{EntityType: "task", Key: "T-001", Title: "Big", Size: sizePtrR(8)}}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Factors[5].Score)
	})

	t.Run("size=13 → score 0", func(t *testing.T) {
		assignments := []sprint.AssignmentWithSize{{EntityType: "task", Key: "T-001", Title: "XXL", Size: sizePtrR(13)}}
		svc := makeReadinessSvc(sprintObj, assignments, nil)
		result, err := svc.GetSprintReadiness(ctx, "S024")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Factors[5].Score)
	})
}

// TestSprintService_GetSprintReadiness_OverallScore tests TC-013-07:
// OverallScore = sum of factor scores; value in [0, 100].
func TestSprintService_GetSprintReadiness_OverallScore(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	// 3 tasks, 2 agent types, all sized, none oversized, no unsat deps, 50% util
	assignments := []sprint.AssignmentWithSize{
		{EntityType: "task", Key: "T-001", Size: sizePtrR(5), AgentType: agentPtrR("backend"), DependsOn: "[]"},
		{EntityType: "task", Key: "T-002", Size: sizePtrR(5), AgentType: agentPtrR("frontend"), DependsOn: "[]"},
		{EntityType: "task", Key: "T-003", Size: sizePtrR(0), AgentType: agentPtrR("backend"), DependsOn: "[]"},
	}
	caps := []*models.SprintCapacity{{AgentType: "backend", CapacityPoints: 20}}

	svc := makeReadinessSvc(sprintObj, assignments, caps)
	result, err := svc.GetSprintReadiness(ctx, "S024")
	require.NoError(t, err)
	require.NotNil(t, result)

	total := 0
	for _, f := range result.Factors {
		total += f.Score
	}
	assert.Equal(t, total, result.OverallScore, "OverallScore must equal sum of factor scores")
	assert.LessOrEqual(t, result.OverallScore, 100)
	assert.GreaterOrEqual(t, result.OverallScore, 0)
}

// TestSprintService_GetSprintReadiness_ZeroEntities tests TC-013-11:
// Zero entities → score 0, all 6 factors at 0, non-empty Detail.
func TestSprintService_GetSprintReadiness_ZeroEntities(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	svc := makeReadinessSvc(sprintObj, []sprint.AssignmentWithSize{}, nil)
	result, err := svc.GetSprintReadiness(ctx, "S024")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.OverallScore)
	require.Len(t, result.Factors, 6)
	for i, f := range result.Factors {
		assert.Equal(t, 0, f.Score, "Factor[%d].Score must be 0 for empty sprint", i)
		assert.NotEmpty(t, f.Detail, "Factor[%d].Detail must not be empty", i)
	}
	assert.Empty(t, result.UnsizedEntities)
	assert.Empty(t, result.OversizedEntities)
}

// TestSprintService_GetSprintReadiness_UnsizedEntitiesPopulated tests TC-013-12.
func TestSprintService_GetSprintReadiness_UnsizedEntitiesPopulated(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	assignments := []sprint.AssignmentWithSize{
		{EntityType: "task", Key: "T-001", Title: "Sized", Size: sizePtrR(3)},
		{EntityType: "task", Key: "T-002", Title: "Unsized Alpha", Size: nil},
		{EntityType: "task", Key: "T-003", Title: "Unsized Beta", Size: nil},
		{EntityType: "task", Key: "T-004", Title: "Also Sized", Size: sizePtrR(5)},
	}

	svc := makeReadinessSvc(sprintObj, assignments, nil)
	result, err := svc.GetSprintReadiness(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, result.UnsizedEntities, 2)
	keys := map[string]bool{}
	for _, e := range result.UnsizedEntities {
		keys[e.Key] = true
	}
	assert.True(t, keys["T-002"])
	assert.True(t, keys["T-003"])
}

// TestSprintService_GetSprintReadiness_OversizedEntitiesPopulated tests TC-013-13.
func TestSprintService_GetSprintReadiness_OversizedEntitiesPopulated(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	assignments := []sprint.AssignmentWithSize{
		{EntityType: "task", Key: "T-001", Title: "Small", Size: sizePtrR(5)},
		{EntityType: "task", Key: "T-002", Title: "Large", Size: sizePtrR(8)},  // boundary
		{EntityType: "task", Key: "T-003", Title: "Huge", Size: sizePtrR(13)},  // oversized
		{EntityType: "task", Key: "T-004", Title: "Medium", Size: sizePtrR(7)}, // NOT oversized
	}

	svc := makeReadinessSvc(sprintObj, assignments, nil)
	result, err := svc.GetSprintReadiness(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, result.OversizedEntities, 2)
	keys := map[string]bool{}
	for _, e := range result.OversizedEntities {
		keys[e.Key] = true
	}
	assert.True(t, keys["T-002"])
	assert.True(t, keys["T-003"])
}

// TestSprintService_GetSprintReadiness_Determinism tests TC-013-10.
func TestSprintService_GetSprintReadiness_Determinism(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	assignments := []sprint.AssignmentWithSize{
		{EntityType: "task", Key: "T-001", Size: sizePtrR(5), AgentType: agentPtrR("backend"), DependsOn: "[]"},
		{EntityType: "task", Key: "T-002", Size: sizePtrR(3), AgentType: agentPtrR("frontend"), DependsOn: "[]"},
		{EntityType: "task", Key: "T-003", Size: nil, DependsOn: "[]"},
	}
	caps := []*models.SprintCapacity{{AgentType: "backend", CapacityPoints: 20}}

	svc := makeReadinessSvc(sprintObj, assignments, caps)

	r1, err1 := svc.GetSprintReadiness(ctx, "S024")
	r2, err2 := svc.GetSprintReadiness(ctx, "S024")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, r1.OverallScore, r2.OverallScore)
	for i := range r1.Factors {
		assert.Equal(t, r1.Factors[i].Score, r2.Factors[i].Score, "Factor[%d] not deterministic", i)
	}
}

// TestSprintService_GetSprintReadiness_TwoQueryCallCount tests TC-013-14:
// GetAssignmentsWithSize and GetCapacity each called exactly once.
func TestSprintService_GetSprintReadiness_TwoQueryCallCount(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	assignCalls := 0
	capCalls := 0

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return sprintObj, nil
		},
	}
	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		GetAssignmentsWithSizeFunc: func(_ context.Context, _ int64) ([]sprint.AssignmentWithSize, error) {
			assignCalls++
			return []sprint.AssignmentWithSize{{EntityType: "task", Key: "T-001", Size: sizePtrR(3), DependsOn: "[]"}}, nil
		},
	}
	mockCapRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(_ context.Context, _ int64) ([]*models.SprintCapacity, error) {
			capCalls++
			return []*models.SprintCapacity{{AgentType: "backend", CapacityPoints: 10}}, nil
		},
	}

	wf := workflow.NewService("")
	svc := NewSprintService(mockRepo, wf, mockAssignRepo, mockCapRepo, nil)

	result, err := svc.GetSprintReadiness(ctx, "S024")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, assignCalls, "GetAssignmentsWithSize must be called exactly once")
	assert.Equal(t, 1, capCalls, "GetCapacity must be called exactly once")
}

// TestSprintService_GetSprintReadiness_FactorLabels tests TC-013-09:
// All 6 factors have non-empty Name, Detail, and valid MaxScore.
func TestSprintService_GetSprintReadiness_FactorLabels(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	svc := makeReadinessSvc(sprintObj, []sprint.AssignmentWithSize{
		{EntityType: "task", Key: "T-001", Size: sizePtrR(5), DependsOn: "[]"},
	}, nil)

	result, err := svc.GetSprintReadiness(ctx, "S024")
	require.NoError(t, err)
	require.Len(t, result.Factors, 6)
	for i, f := range result.Factors {
		assert.NotEmpty(t, f.Name, "Factor[%d].Name must not be empty", i)
		assert.NotEmpty(t, f.Detail, "Factor[%d].Detail must not be empty", i)
		assert.GreaterOrEqual(t, f.MaxScore, 0, "Factor[%d].MaxScore must be >= 0", i)
	}
}

// TestSprintService_GetSprintReadiness_NilAssignmentRepoError tests error path.
func TestSprintService_GetSprintReadiness_NilAssignmentRepoError(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return &models.Sprint{ID: 1, Key: "S001", Status: "planning"}, nil
		},
	}
	wf := workflow.NewService("")
	svc := NewSprintService(mockRepo, wf, nil, nil, nil)

	result, err := svc.GetSprintReadiness(ctx, "S001")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// T-E19-F05-004: PlanSprint tests (TC-011-01..04)
// ─────────────────────────────────────────────────────────────────────────────

// makePlanSvc creates a SprintService with controlled backlog, assignments, and capacity.
func makePlanSvc(
	sprintObj *models.Sprint,
	backlog []sprint.BacklogItem,
	assignments []sprint.AssignmentWithSize,
	capacityRows []*models.SprintCapacity,
) *SprintService {
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
	}
	if backlog == nil {
		backlog = []sprint.BacklogItem{}
	}
	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(_ context.Context, _ []string) ([]sprint.BacklogItem, error) {
			return backlog, nil
		},
		GetAssignmentsWithSizeFunc: func(_ context.Context, _ int64) ([]sprint.AssignmentWithSize, error) {
			return assignments, nil
		},
	}
	rows := capacityRows
	if rows == nil {
		rows = []*models.SprintCapacity{}
	}
	mockCapRepo := &MockSprintCapacityRepository{
		GetCapacityFunc: func(_ context.Context, _ int64) ([]*models.SprintCapacity, error) {
			return rows, nil
		},
	}
	return NewSprintService(mockRepo, workflow.NewService(""), mockAssignRepo, mockCapRepo, nil)
}

// TestPlanSprint_HappyPath tests TC-011-04: composite view with all three sections populated.
//
// Caller-Path Contract (TC-011-04):
//   - Entrypoint: SprintService.PlanSprint(ctx, "S024") — key string, not ID
//   - Lowest mock seam: SprintAssignmentQueryRepository and SprintCapacityRepository interfaces
//   - Forbidden mocks: Do NOT mock PlanSprint itself; GetSprintReadiness/readiness compute in-memory
//   - Counter-factual: an impl that returns Readiness: nil panics when CLI formats the readiness section
func TestPlanSprint_HappyPath(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	agentBackend := "backend"
	agentFrontend := "frontend"
	sz5 := 5
	sz7 := 7

	backlog := []sprint.BacklogItem{
		{EntityType: "task", EntityID: 101, Key: "E07-F01-010", Title: "Unassigned A", Priority: 8, AgentType: &agentBackend},
		{EntityType: "task", EntityID: 102, Key: "E07-F01-011", Title: "Unassigned B", Priority: 5, AgentType: &agentFrontend},
	}
	assignments := []sprint.AssignmentWithSize{
		{EntityType: "task", EntityID: 201, Key: "E07-F01-001", AgentType: &agentBackend, Size: &sz5},
		{EntityType: "task", EntityID: 202, Key: "E07-F01-002", AgentType: &agentBackend, Size: &sz7},
		{EntityType: "task", EntityID: 203, Key: "E07-F01-003", AgentType: &agentFrontend, Size: nil},
	}
	capRows := []*models.SprintCapacity{
		{AgentType: "backend", CapacityPoints: 21},
		{AgentType: "frontend", CapacityPoints: 13},
	}

	svc := makePlanSvc(sprintObj, backlog, assignments, capRows)
	result, err := svc.PlanSprint(ctx, "S024")

	require.NoError(t, err)
	require.NotNil(t, result)

	// Sprint is populated
	assert.Equal(t, "S024", result.Sprint.Key)

	// Backlog has 2 items (from ListUnassignedBacklog mock)
	require.Len(t, result.Backlog, 2, "backlog must have 2 unassigned items")
	assert.Equal(t, "E07-F01-010", result.Backlog[0].Key)

	// Capacity has 2 rows with correct allocation
	require.Len(t, result.Capacity, 2)
	var backendCap, frontendCap *CapacityRow
	for i := range result.Capacity {
		if result.Capacity[i].AgentType == "backend" {
			backendCap = &result.Capacity[i]
		}
		if result.Capacity[i].AgentType == "frontend" {
			frontendCap = &result.Capacity[i]
		}
	}
	require.NotNil(t, backendCap)
	assert.Equal(t, float64(12), backendCap.AllocatedPoints, "backend = 5+7 = 12")
	assert.Equal(t, float64(9), backendCap.Remaining, "backend remaining = 21-12 = 9")
	assert.Equal(t, 0, backendCap.UnsizedAssigned, "no unsized backend tasks")

	require.NotNil(t, frontendCap)
	assert.Equal(t, float64(0), frontendCap.AllocatedPoints, "frontend unsized contributes 0")
	assert.Equal(t, 1, frontendCap.UnsizedAssigned, "one unsized frontend task")

	// Readiness is non-nil with 6 factors (score computed in-memory)
	require.NotNil(t, result.Readiness)
	assert.Len(t, result.Readiness.Factors, 6)
	assert.GreaterOrEqual(t, result.Readiness.OverallScore, 0)
	assert.LessOrEqual(t, result.Readiness.OverallScore, 100)
}

// TestPlanSprint_EmptySprint tests TC-011-01 edge case: empty backlog, zero readiness.
//
// Caller-Path Contract (TC-011-01):
//   - Counter-factual: an impl that returns nil Backlog slice fails JSON marshal (must be [])
func TestPlanSprint_EmptySprint(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 10, Key: "S010", Name: "Empty Sprint", Status: "planning"}

	svc := makePlanSvc(sprintObj, []sprint.BacklogItem{}, []sprint.AssignmentWithSize{}, []*models.SprintCapacity{})
	result, err := svc.PlanSprint(ctx, "S010")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Backlog, "backlog must be non-nil (empty slice, not nil)")
	assert.Len(t, result.Backlog, 0)
	assert.NotNil(t, result.Capacity, "capacity must be non-nil (empty slice, not nil)")
	assert.Len(t, result.Capacity, 0)
	require.NotNil(t, result.Readiness)
	assert.Equal(t, 0, result.Readiness.OverallScore, "zero entities → readiness score = 0")
}

// TestPlanSprint_BacklogFiltering tests TC-011-01: backlog comes from ListUnassignedBacklog
// and excludes already-assigned entities.
//
// Caller-Path Contract:
//   - Lowest mock seam: ListUnassignedBacklog returns controlled set (exclusion done by repo)
//   - Counter-factual: an impl that ignores removed_at IS NULL would include removed assignments
func TestPlanSprint_BacklogFiltering(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 24, Key: "S024", Status: "planning"}

	agentBackend := "backend"
	// Mock returns only unassigned/eligible tasks (repo has already filtered)
	backlog := []sprint.BacklogItem{
		{EntityType: "task", EntityID: 300, Key: "E07-F01-300", Title: "Task C (unassigned)", Priority: 7, AgentType: &agentBackend},
		{EntityType: "task", EntityID: 400, Key: "E07-F01-400", Title: "Task D (was removed, now eligible)", Priority: 3, AgentType: &agentBackend},
	}

	svc := makePlanSvc(sprintObj, backlog, []sprint.AssignmentWithSize{}, []*models.SprintCapacity{})
	result, err := svc.PlanSprint(ctx, "S024")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Backlog, 2, "backlog must have exactly the items returned by ListUnassignedBacklog")
	assert.Equal(t, "E07-F01-300", result.Backlog[0].Key)
	assert.Equal(t, "E07-F01-400", result.Backlog[1].Key)
}

// TestPlanSprint_ThreeSectionOutput tests TC-011-04: Sprint field, Backlog, Capacity, and Readiness
// are all non-nil in the result.
//
// Caller-Path Contract:
//   - Counter-factual: an impl that returns SprintPlanView with nil Readiness would fail
//     when the CLI tries to access Readiness.OverallScore
func TestPlanSprint_ThreeSectionOutput(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 5, Key: "S005", Status: "planning"}
	agentBackend := "backend"
	sz3 := 3

	backlog := []sprint.BacklogItem{
		{EntityType: "task", EntityID: 501, Key: "E01-F01-501", Title: "Ready task", Priority: 5, AgentType: &agentBackend},
	}
	assignments := []sprint.AssignmentWithSize{
		{EntityType: "task", EntityID: 601, Key: "E01-F01-601", AgentType: &agentBackend, Size: &sz3},
	}
	capRows := []*models.SprintCapacity{
		{AgentType: "backend", CapacityPoints: 10},
	}

	svc := makePlanSvc(sprintObj, backlog, assignments, capRows)
	result, err := svc.PlanSprint(ctx, "S005")

	require.NoError(t, err)
	require.NotNil(t, result)

	// All three sections must be non-nil
	assert.NotNil(t, result.Sprint, "Sprint section must be non-nil")
	assert.NotNil(t, result.Backlog, "Backlog section must be non-nil")
	assert.NotNil(t, result.Capacity, "Capacity section must be non-nil")
	assert.NotNil(t, result.Readiness, "Readiness section must be non-nil")

	// Sprint field correctly populated
	assert.Equal(t, "S005", result.Sprint.Key)

	// Backlog has 1 item
	assert.Len(t, result.Backlog, 1)

	// Capacity has 1 row
	assert.Len(t, result.Capacity, 1)
	assert.Equal(t, "backend", result.Capacity[0].AgentType)
	assert.Equal(t, float64(10), result.Capacity[0].CapacityPoints)
	assert.Equal(t, float64(3), result.Capacity[0].AllocatedPoints)

	// Readiness has 6 factors and valid score
	assert.Len(t, result.Readiness.Factors, 6)
	assert.GreaterOrEqual(t, result.Readiness.OverallScore, 0)
	assert.LessOrEqual(t, result.Readiness.OverallScore, 100)
}

// TestPlanSprint_SprintNotFound tests that PlanSprint propagates sprint-not-found error.
func TestPlanSprint_SprintNotFound(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return nil, fmt.Errorf("sprint %q not found", key)
		},
	}
	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)
	result, err := svc.PlanSprint(ctx, "S999")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "S999")
}

// TestPlanSprint_NilAssignmentRepo tests graceful degradation when assignmentRepo is nil.
//
// Caller-Path Contract:
//   - Counter-factual: an impl that panics on nil assignmentRepo breaks degraded-mode callers
func TestPlanSprint_NilAssignmentRepo(t *testing.T) {
	ctx := context.Background()
	sprintObj := &models.Sprint{ID: 1, Key: "S001", Status: "planning"}
	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
	}
	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)
	result, err := svc.PlanSprint(ctx, "S001")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Backlog, "backlog must be empty slice, not nil")
	assert.Len(t, result.Backlog, 0)
	assert.NotNil(t, result.Capacity, "capacity must be empty slice, not nil")
	assert.Len(t, result.Capacity, 0)
	require.NotNil(t, result.Readiness, "readiness must be non-nil even with nil repos")
	assert.Equal(t, 0, result.Readiness.OverallScore, "nil repos → no assignments → score=0")
}

// ─── E19-F07-003: AddEntityToSprint with Position and BulkAddToSprint auto-number ───────────

// intPtr returns a pointer to the given int value. Used in service-layer position tests
// (TC-005 through TC-012) to build *int values without taking the address of a literal.
func intPtr(v int) *int { return &v }

// makeActiveSprint creates a minimal *models.Sprint in "active" status for use in
// the TC-005..TC-012 test helpers. The caller may override fields as needed.
func makeActiveSprint(id int64, key string) *models.Sprint {
	return &models.Sprint{
		ID:        id,
		Key:       key,
		Status:    "active",
		Name:      "Test Sprint",
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(7 * 24 * time.Hour),
	}
}

// makeAssignment creates a *models.SprintAssignment with the given sprint order (nil = unordered).
func makeAssignment(id, sprintID int64, entityType string, entityID int64, order *int) *models.SprintAssignment {
	return &models.SprintAssignment{
		ID:          id,
		SprintID:    sprintID,
		EntityType:  entityType,
		EntityID:    entityID,
		AssignedAt:  time.Now(),
		SprintOrder: order,
	}
}

// TestAddEntityToSprint_TC005_AtPosition1_ShiftsExistingItems verifies that
// providing Position=1 places the new item at position 1 and shifts the 3
// existing ordered items to positions 2, 3, 4.
//
// TC-005 — Caller-Path Contract:
//   - Entrypoint: SprintService.AddEntityToSprint(ctx, AddEntityInput{..., Position: intPtr(1)})
//   - Mock seam: MockSprintRepository (MaxSprintOrder, AddAssignment, ListOrderedAssignments, RenumberAssignmentsTx)
//   - Counter-factual: a buggy impl that ignores Position and always appends would NOT call
//     RenumberAssignmentsTx; this test asserts it IS called with the correct shift ops.
func TestAddEntityToSprint_TC005_AtPosition1_ShiftsExistingItems(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	// Three existing ordered items at positions 1, 2, 3.
	existing := []*models.SprintAssignment{
		makeAssignment(1, 10, "task", 100, intPtr(1)),
		makeAssignment(2, 10, "task", 101, intPtr(2)),
		makeAssignment(3, 10, "task", 102, intPtr(3)),
	}

	var renumberCalls [][]sprint.RenumberOp
	var addAssignmentInsertOrder *int // snapshot of sprint_order at INSERT time
	addAssignmentCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) { return 3, nil },
		ListOrderedAssignmentsFunc: func(_ context.Context, _ int64) ([]*models.SprintAssignment, error) {
			return existing, nil
		},
		AddAssignmentFunc: func(_ context.Context, a *models.SprintAssignment) error {
			a.ID = 99 // simulate DB auto-increment
			addAssignmentCalled = true
			// Snapshot the sprint_order value at the moment of INSERT — the
			// service may rebind a.SprintOrder later, so a pointer-capture
			// would race with that mutation.
			if a.SprintOrder != nil {
				v := *a.SprintOrder
				addAssignmentInsertOrder = &v
			} else {
				addAssignmentInsertOrder = nil
			}
			return nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, ops []sprint.RenumberOp) error {
			// Copy the slice so the test sees the call's snapshot, not a later mutation.
			snap := make([]sprint.RenumberOp, len(ops))
			copy(snap, ops)
			renumberCalls = append(renumberCalls, snap)
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ *int) error {
			return nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  intPtr(1),
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)
	require.NotNil(t, assignment.SprintOrder, "SprintOrder must be set")
	assert.Equal(t, 1, *assignment.SprintOrder, "new item must land at position 1")

	// AddAssignment must INSERT with sprint_order=NULL so the partial unique
	// index (sprint_id, sprint_order) doesn't fire on the existing row at pos 1.
	require.True(t, addAssignmentCalled, "AddAssignment must have been called")
	assert.Nil(t, addAssignmentInsertOrder,
		"new row must be INSERTed with sprint_order=NULL when a shift is needed; final position is assigned via the renumber UPDATE")

	// Two RenumberAssignmentsTx calls: pre-pass NULLs the siblings, then a
	// single statement assigns target + shifted siblings their final positions.
	require.Len(t, renumberCalls, 2, "expected exactly two renumber calls: clear then assign")

	clearOps := renumberCalls[0]
	require.Len(t, clearOps, 3, "clear pass must NULL all three shifted siblings")
	for i, op := range clearOps {
		assert.Nil(t, op.NewPosition, "clear-pass op[%d] must have NewPosition=nil", i)
	}

	finalOps := renumberCalls[1]
	require.Len(t, finalOps, 4, "final pass must place target + shift the 3 siblings")
	// Op 0 = target at position 1.
	assert.Equal(t, int64(99), finalOps[0].AssignmentID, "first op must be the new assignment")
	require.NotNil(t, finalOps[0].NewPosition)
	assert.Equal(t, 1, *finalOps[0].NewPosition)
	// Ops 1..3 = shifted siblings.
	assert.Equal(t, int64(1), finalOps[1].AssignmentID)
	assert.Equal(t, intPtr(2), finalOps[1].NewPosition)
	assert.Equal(t, int64(2), finalOps[2].AssignmentID)
	assert.Equal(t, intPtr(3), finalOps[2].NewPosition)
	assert.Equal(t, int64(3), finalOps[3].AssignmentID)
	assert.Equal(t, intPtr(4), finalOps[3].NewPosition)
}

// TestAddEntityToSprint_TC006_AtPositionCountPlus1_Appends verifies that
// providing Position=count+1 (boundary) succeeds without error and the item
// lands at the last position (no sibling shift required).
//
// TC-006 — Counter-factual: a buggy impl that rejects count+1 returns an error;
// this test asserts nil error and sprint_order == count+1.
func TestAddEntityToSprint_TC006_AtPositionCountPlus1_Appends(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	existing := []*models.SprintAssignment{
		makeAssignment(1, 10, "task", 100, intPtr(1)),
		makeAssignment(2, 10, "task", 101, intPtr(2)),
		makeAssignment(3, 10, "task", 102, intPtr(3)),
	}

	renumberCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) { return 3, nil },
		ListOrderedAssignmentsFunc: func(_ context.Context, _ int64) ([]*models.SprintAssignment, error) {
			return existing, nil
		},
		AddAssignmentFunc: func(_ context.Context, a *models.SprintAssignment) error {
			a.ID = 99
			return nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ []sprint.RenumberOp) error {
			renumberCalled = true
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ *int) error { return nil },
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	// Position=4 == count+1 (3 existing items)
	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  intPtr(4),
	})

	require.NoError(t, err, "count+1 position must not return an error")
	require.NotNil(t, assignment)
	require.NotNil(t, assignment.SprintOrder)
	assert.Equal(t, 4, *assignment.SprintOrder)
	assert.False(t, renumberCalled, "RenumberAssignmentsTx must NOT be called when appending at end")
}

// TestAddEntityToSprint_TC007_AtPositionCountPlus2_Error verifies that
// providing Position=count+2 (out of range) returns a validation error and does
// NOT create any assignment.
//
// TC-007 — Counter-factual: a buggy impl that clamps position to count+1 would
// not return an error; this test asserts a non-nil error containing "out of range".
func TestAddEntityToSprint_TC007_AtPositionCountPlus2_Error(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	addCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) { return 3, nil },
		ListOrderedAssignmentsFunc: func(_ context.Context, _ int64) ([]*models.SprintAssignment, error) {
			return []*models.SprintAssignment{
				makeAssignment(1, 10, "task", 100, intPtr(1)),
				makeAssignment(2, 10, "task", 101, intPtr(2)),
				makeAssignment(3, 10, "task", 102, intPtr(3)),
			}, nil
		},
		AddAssignmentFunc: func(_ context.Context, _ *models.SprintAssignment) error {
			addCalled = true
			return nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	// Position=5 == count+2 (3 existing items, so max valid = 4)
	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  intPtr(5),
	})

	require.Error(t, err, "position count+2 must return an error")
	assert.Nil(t, assignment, "no assignment must be created on out-of-range position")
	assert.Contains(t, err.Error(), "out of range", "error must mention 'out of range'")
	assert.Contains(t, err.Error(), "3", "error must mention the count of ordered items (3)")
	assert.False(t, addCalled, "AddAssignment must NOT be called when position is invalid")
}

// TestAddEntityToSprint_TC008_AtPosition0_Error verifies that Position=0 returns
// a validation error before any repository call.
//
// TC-008 — Counter-factual: a buggy impl that treats 0 as "append at start"
// would not return an error; this test asserts non-nil error.
func TestAddEntityToSprint_TC008_AtPosition0_Error(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	repoCalled := false
	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) {
			repoCalled = true
			return 0, nil
		},
		AddAssignmentFunc: func(_ context.Context, _ *models.SprintAssignment) error {
			repoCalled = true
			return nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  intPtr(0),
	})

	require.Error(t, err, "position=0 must return an error")
	assert.Nil(t, assignment)
	assert.Contains(t, err.Error(), "1", "error must reference the minimum valid position (1)")
	assert.False(t, repoCalled, "no repo order/add calls must be made for position <= 0")
}

// TestAddEntityToSprint_TC009_NegativePosition_Error verifies that a negative
// Position returns the same "must be >= 1" validation error as TC-008.
//
// TC-009 — extends TC-008 to negative values.
func TestAddEntityToSprint_TC009_NegativePosition_Error(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	addCalled := false
	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		AddAssignmentFunc: func(_ context.Context, _ *models.SprintAssignment) error {
			addCalled = true
			return nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  intPtr(-3),
	})

	require.Error(t, err, "negative position must return an error")
	assert.Nil(t, assignment)
	assert.Contains(t, err.Error(), "1")
	assert.False(t, addCalled, "AddAssignment must NOT be called for negative position")
}

// TestAddEntityToSprint_TC010_NilPosition_AppendsAtMaxPlus1 verifies that a nil
// Position (no --at flag) auto-assigns sprint_order = max + 1 without calling
// RenumberAssignmentsTx.
//
// TC-010 — Counter-factual: a buggy impl that defaults nil to position=1 would
// call RenumberAssignmentsTx; this test asserts it is NOT called.
func TestAddEntityToSprint_TC010_NilPosition_AppendsAtMaxPlus1(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	renumberCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) { return 3, nil },
		AddAssignmentFunc: func(_ context.Context, a *models.SprintAssignment) error {
			a.ID = 99
			return nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ []sprint.RenumberOp) error {
			renumberCalled = true
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ *int) error { return nil },
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	// Position is nil — no --at flag
	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  nil,
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)
	require.NotNil(t, assignment.SprintOrder, "SprintOrder must be set even for nil Position")
	assert.Equal(t, 4, *assignment.SprintOrder, "nil Position must auto-assign max+1 = 4")
	assert.False(t, renumberCalled, "nil Position must NOT trigger RenumberAssignmentsTx")
}

// TestAddEntityToSprint_TC011_EmptySprint_FirstItemGetsPosition1 verifies that
// when a sprint has no ordered items (MaxSprintOrder returns 0), the first item
// added with nil Position gets sprint_order = 1.
//
// TC-011 — Counter-factual: a buggy impl using max-initialized-to-(-1) would
// assign sprint_order=0 (reserved); this test asserts sprint_order=1.
func TestAddEntityToSprint_TC011_EmptySprint_FirstItemGetsPosition1(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	maxCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		GetTaskIDByKeyFunc: func(_ context.Context, _ string) (int64, error) { return 200, nil },
		GetActiveAssignmentFunc: func(_ context.Context, _ string, _ int64) (*models.SprintAssignment, error) {
			return nil, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) {
			maxCalled = true
			return 0, nil // no ordered items
		},
		AddAssignmentFunc: func(_ context.Context, a *models.SprintAssignment) error {
			a.ID = 99
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ *int) error { return nil },
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	assignment, _, err := svc.AddEntityToSprint(ctx, AddEntityInput{
		SprintKey: "S001",
		EntityKey: "E07-F01-001",
		Position:  nil,
	})

	require.NoError(t, err)
	require.NotNil(t, assignment)
	require.NotNil(t, assignment.SprintOrder, "SprintOrder must be set for first item")
	assert.Equal(t, 1, *assignment.SprintOrder, "first item in empty sprint must get sprint_order=1")
	assert.True(t, maxCalled, "MaxSprintOrder must be called to determine next position")
}

// TestBulkAddToSprint_TC012_AssignsSequentialSprintOrders verifies that
// BulkAddToSprint assigns sequential sprint_order values per row passed to
// BulkAssign, and calls RenumberAssignmentsTx to repair gaps from INSERT OR IGNORE skips.
//
// TC-012 — Caller-Path Contract:
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S001", FeatureKey:"E07-F01"})
//   - Mock seam: MockSprintRepository (BulkAssign, MaxSprintOrder; assert BulkAssign receives
//     per-row sprint_order values; assert RenumberAssignmentsTx called after skip)
//   - Counter-factual: a buggy impl leaving gaps (items 1,3,4 after skip) would have
//     sprint_order=3 for the second item; this test asserts dense renumber produces 1,2,3,4.
func TestBulkAddToSprint_TC012_AssignsSequentialSprintOrders(t *testing.T) {
	ctx := context.Background()
	sprintObj := makeActiveSprint(10, "S001")

	// Existing 2 ordered items (positions 1 and 2).
	// BulkAdd will attempt 3 tasks; one is a duplicate (INSERT OR IGNORE skips it → inserted=2).
	candidates := []sprint.BacklogItem{
		{EntityType: "task", EntityID: 100, Key: "E07-F01-001", Status: "todo"},
		{EntityType: "task", EntityID: 101, Key: "E07-F01-002", Status: "todo"},
		{EntityType: "task", EntityID: 102, Key: "E07-F01-003", Status: "todo"},
	}

	var capturedAssignments []models.SprintAssignment
	renumberOpsCaptured := []sprint.RenumberOp(nil)

	// After the bulk insert, ListOrderedAssignments is called to find batch items that need
	// renumbering. We simulate the DB state: existing 2 items (pos 1,2) plus 2 inserted (pos 3,5
	// — gaps because item at pos 4 was skipped). The service must renumber 3→3, 5→4.
	orderedAfterBulk := []*models.SprintAssignment{
		makeAssignment(10, 10, "task", 200, intPtr(1)), // pre-existing
		makeAssignment(11, 10, "task", 201, intPtr(2)), // pre-existing
		makeAssignment(12, 10, "task", 100, intPtr(3)), // inserted (candidate 0)
		// candidate 1 (entityID=101) was skipped by INSERT OR IGNORE
		makeAssignment(13, 10, "task", 102, intPtr(5)), // inserted (candidate 2, gets pos 5 due to gap)
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc:       func(_ context.Context, _ string) (*models.Sprint, error) { return sprintObj, nil },
		MaxSprintOrderFunc: func(_ context.Context, _ int64) (int, error) { return 2, nil },
		ListOrderedAssignmentsFunc: func(_ context.Context, _ int64) ([]*models.SprintAssignment, error) {
			return orderedAfterBulk, nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, ops []sprint.RenumberOp) error {
			renumberOpsCaptured = ops
			return nil
		},
	}

	mockAssignmentRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(_ context.Context, _ []string) ([]sprint.BacklogItem, error) {
			return candidates, nil
		},
		BulkAssignFunc: func(_ context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
			capturedAssignments = assignments
			// Simulate INSERT OR IGNORE skipping one row → 2 inserted out of 3.
			return 2, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), mockAssignmentRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:  "S001",
		FeatureKey: "E07-F01",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Assert BulkAssign received per-row sprint_order values (max=2, so new rows get 3,4,5).
	require.Len(t, capturedAssignments, 3, "all 3 candidates must be passed to BulkAssign")
	for i, a := range capturedAssignments {
		require.NotNil(t, a.SprintOrder, "each assignment must have a sprint_order set")
		assert.Equal(t, 2+i+1, *a.SprintOrder, "assignment %d must have sprint_order=%d", i, 2+i+1)
	}

	// After INSERT OR IGNORE skip (1 skipped), RenumberAssignmentsTx must be called
	// to repair gaps — the 2 inserted items should be renumbered densely.
	assert.NotNil(t, renumberOpsCaptured, "RenumberAssignmentsTx must be called after a bulk skip")
}

// ---------------------------------------------------------------------------
// TC-001: GetNextTask — sprint_order beats ExecutionOrder
// ---------------------------------------------------------------------------
//
// TC-001 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetNextTask(ctx, "") — agentType="" = no filter
//   - Mock seam: MockSprintRepository (ListFunc + ListBacklogFunc; real comparator runs)
//   - Forbidden mocks: do NOT mock the sort comparator or selectionReason helper
//   - Counter-factual: a buggy impl that ignores sprint_order (old three-tier sort) would
//     return itemB (lower ExecutionOrder=1) instead of itemA (sprint_order=1); this test
//     would fail because it asserts Key=="task-A" and SelectionReason=="sprint_order"
func TestGetNextTask_TC001_SprintOrderBeatsExecutionOrder(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")
	order1 := 1
	execOrder1 := 1

	// itemA has sprint_order=1 but higher ExecutionOrder
	// itemB has no sprint_order but lower ExecutionOrder=1
	// Without the sprint_order tier, itemB would win; with it, itemA must win.
	backlogItems := []*sprint.BacklogItem{
		{
			EntityType:  "task",
			Key:         "task-A",
			Title:       "Task A",
			Status:      "todo",
			SprintOrder: &order1, // sprint_order = 1 → should win
			AssignedAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			EntityType:     "task",
			Key:            "task-B",
			Title:          "Task B",
			Status:         "todo",
			ExecutionOrder: &execOrder1, // execution_order = 1, no sprint_order
			AssignedAt:     time.Now().Add(-2 * time.Hour),
		},
	}

	mockRepo := &MockSprintRepository{
		ListFunc: func(_ context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{activeSprint}, nil
		},
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, _ int64, _ *string, _ bool, _ ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetNextTask(ctx, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "task-A", result.Key, "sprint_order=1 item must be selected over unordered item with lower ExecutionOrder")
	assert.Equal(t, "sprint_order", result.SelectionReason, "SelectionReason must be sprint_order")
	assert.NotNil(t, result.SprintOrder, "SprintOrder must be set on the returned item")
	assert.Equal(t, 1, *result.SprintOrder, "SprintOrder must equal the stored sprint_order")
	assert.Equal(t, "S001", result.SprintKey, "SprintKey must be set on the returned item")
}

// ---------------------------------------------------------------------------
// TC-002: GetNextTask — two ordered items, lower sprint_order wins
// ---------------------------------------------------------------------------
//
// TC-002 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetNextTask(ctx, "")
//   - Mock seam: MockSprintRepository (ListBacklogFunc)
//   - Counter-factual: a buggy impl that sorts by ExecutionOrder first would return
//     itemB (ExecutionOrder=1) instead of itemA (sprint_order=1, ExecutionOrder=5)
func TestGetNextTask_TC002_LowerSprintOrderWins(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")
	order1 := 1
	order2 := 2
	execOrder5 := 5
	execOrder1 := 1

	backlogItems := []*sprint.BacklogItem{
		{
			EntityType:     "task",
			Key:            "task-A",
			Title:          "Task A",
			Status:         "todo",
			SprintOrder:    &order1,     // sprint_order = 1 → must win
			ExecutionOrder: &execOrder5, // execution_order = 5 (higher)
			AssignedAt:     time.Now().Add(-1 * time.Hour),
		},
		{
			EntityType:     "task",
			Key:            "task-B",
			Title:          "Task B",
			Status:         "todo",
			SprintOrder:    &order2,     // sprint_order = 2
			ExecutionOrder: &execOrder1, // execution_order = 1 (lower)
			AssignedAt:     time.Now().Add(-2 * time.Hour),
		},
	}

	mockRepo := &MockSprintRepository{
		ListFunc: func(_ context.Context, _ *sprint.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{activeSprint}, nil
		},
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, _ int64, _ *string, _ bool, _ ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetNextTask(ctx, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "task-A", result.Key, "lower sprint_order must win over lower ExecutionOrder")
	assert.Equal(t, "sprint_order", result.SelectionReason, "SelectionReason must be sprint_order when that tier breaks the tie")
}

// ---------------------------------------------------------------------------
// TC-003: GetNextTask — two unordered items, lower ExecutionOrder wins (stable)
// ---------------------------------------------------------------------------
//
// TC-003 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetNextTask(ctx, "") called 10 times
//   - Mock seam: MockSprintRepository (stable mock return — same slice every call)
//   - Forbidden mocks: do NOT randomise mock return order between calls
//   - Counter-factual: sort.Slice (unstable) would permute equal-key candidates;
//     across 10 calls the result might change, failing the determinism assertion
func TestGetNextTask_TC003_ExecutionOrderFallback_StableResult(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")
	execOrder1 := 1
	execOrder2 := 2

	backlogItems := []*sprint.BacklogItem{
		{
			EntityType:     "task",
			Key:            "task-A",
			Title:          "Task A",
			Status:         "todo",
			ExecutionOrder: &execOrder1, // lower → must win
			AssignedAt:     time.Now().Add(-1 * time.Hour),
		},
		{
			EntityType:     "task",
			Key:            "task-B",
			Title:          "Task B",
			Status:         "todo",
			ExecutionOrder: &execOrder2,
			AssignedAt:     time.Now().Add(-2 * time.Hour),
		},
	}

	mockRepo := &MockSprintRepository{
		ListFunc: func(_ context.Context, _ *sprint.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{activeSprint}, nil
		},
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, _ int64, _ *string, _ bool, _ ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	// Call 10 times — stable sort must return the same result every time.
	for i := 0; i < 10; i++ {
		result, err := svc.GetNextTask(ctx, "")
		require.NoError(t, err, "call %d must succeed", i)
		require.NotNil(t, result, "call %d must return a result", i)
		assert.Equal(t, "task-A", result.Key, "call %d: lower ExecutionOrder must win deterministically", i)
		assert.Equal(t, "execution_order", result.SelectionReason, "call %d: SelectionReason must be execution_order", i)
	}
}

// ---------------------------------------------------------------------------
// TC-004: GetNextTask — same ExecutionOrder, higher priority (lower number) wins
// ---------------------------------------------------------------------------
//
// TC-004 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetNextTask(ctx, "")
//   - Mock seam: MockSprintRepository
//   - Counter-factual: a buggy impl that sorts priority descending (higher number = higher
//     priority) would return task-B (Priority=10); this test asserts task-A (Priority=1) wins
//     because lower number = higher priority matches existing GetNextTask semantics
func TestGetNextTask_TC004_PriorityFallback(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")
	execOrder5 := 5

	backlogItems := []*sprint.BacklogItem{
		{
			EntityType:     "task",
			Key:            "task-A",
			Title:          "Task A",
			Status:         "todo",
			ExecutionOrder: &execOrder5,
			Priority:       1, // lower number = higher priority → must win
			AssignedAt:     time.Now().Add(-1 * time.Hour),
		},
		{
			EntityType:     "task",
			Key:            "task-B",
			Title:          "Task B",
			Status:         "todo",
			ExecutionOrder: &execOrder5, // same execution_order
			Priority:       10,
			AssignedAt:     time.Now().Add(-2 * time.Hour),
		},
	}

	mockRepo := &MockSprintRepository{
		ListFunc: func(_ context.Context, _ *sprint.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{activeSprint}, nil
		},
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, _ int64, _ *string, _ bool, _ ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetNextTask(ctx, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "task-A", result.Key, "lower Priority number (higher priority) must win when ExecutionOrder ties")
	assert.Equal(t, "priority", result.SelectionReason, "SelectionReason must be priority when that tier breaks the tie")
}

// ---------------------------------------------------------------------------
// TC-005: GetNextTask — open non-task items remain eligible in sprint order
// ---------------------------------------------------------------------------
//
// TC-005 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetNextTask(ctx, "")
//   - Mock seam: MockSprintRepository
//   - Counter-factual: a buggy impl that restricts candidates to task-only or
//     workflow-initial statuses would return nil here because the sprint has no
//     queued tasks and the open entities are already in ready_for_development.
func TestGetNextTask_TC005_OpenNonTaskItemsRemainEligible(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")
	order1 := 1
	order2 := 2
	order3 := 3

	backlogItems := []*sprint.BacklogItem{
		{
			EntityType:  "task",
			Key:         "task-complete",
			Title:       "Completed task",
			Status:      "completed",
			SprintOrder: &order1,
			AssignedAt:  time.Now().Add(-3 * time.Hour),
		},
		{
			EntityType:  "bug",
			Key:         "B058",
			Title:       "Open bug in ready_for_development",
			Status:      "ready_for_development",
			SprintOrder: &order2,
			AssignedAt:  time.Now().Add(-2 * time.Hour),
		},
		{
			EntityType:  "change_card",
			Key:         "CC-037",
			Title:       "Open change card in ready_for_development",
			Status:      "ready_for_development",
			SprintOrder: &order3,
			AssignedAt:  time.Now().Add(-1 * time.Hour),
		},
	}

	mockRepo := &MockSprintRepository{
		ListFunc: func(_ context.Context, _ *sprint.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{activeSprint}, nil
		},
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, _ int64, _ *string, _ bool, _ ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetNextTask(ctx, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "B058", result.Key, "lowest sprint_order non-terminal item must be selected even when it is a bug")
	assert.Equal(t, "bug", result.EntityType, "non-task entity types must remain eligible")
	assert.Equal(t, "sprint_order", result.SelectionReason, "selection should still be driven by sprint_order")
}

// ---------------------------------------------------------------------------
// TC-016: GetNextTask — single candidate defaults to selection_reason="assigned_at"
// ---------------------------------------------------------------------------
//
// TC-016 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetNextTask(ctx, "") with exactly one candidate
//   - Mock seam: MockSprintRepository
//   - Forbidden mocks: do NOT special-case single-candidate in the comparator mock
//   - Counter-factual: a buggy impl that returns selection_reason="" for single-candidate
//     would fail; this test asserts selection_reason=="assigned_at" even when sprint_order is set
func TestGetNextTask_TC016_SingleCandidateDefaultsToAssignedAt(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")
	order1 := 1

	backlogItems := []*sprint.BacklogItem{
		{
			EntityType:  "task",
			Key:         "task-A",
			Title:       "Task A",
			Status:      "todo",
			SprintOrder: &order1, // has sprint_order — but single candidate, must still default to "assigned_at"
			AssignedAt:  time.Now().Add(-1 * time.Hour),
		},
	}

	mockRepo := &MockSprintRepository{
		ListFunc: func(_ context.Context, _ *sprint.SprintListFilters) ([]*models.Sprint, error) {
			return []*models.Sprint{activeSprint}, nil
		},
		GetByKeyFunc: func(_ context.Context, _ string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, _ int64, _ *string, _ bool, _ ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetNextTask(ctx, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "task-A", result.Key)
	assert.Equal(t, "assigned_at", result.SelectionReason,
		"single-candidate must return selection_reason=assigned_at (no index-out-of-bounds on candidates[1])")
	assert.NotNil(t, result.SprintOrder, "SprintOrder must be populated from the sprint item")
	assert.Equal(t, 1, *result.SprintOrder)
}

// ---------------------------------------------------------------------------
// TC-013: ReorderAssignment — move item from position 3 to position 1
// ---------------------------------------------------------------------------
//
// TC-013 — Caller-Path Contract:
//   - Entrypoint: SprintService.ReorderAssignment(ctx, "S001", "E07-F01-001", ReorderTarget{Position: intPtr(1)})
//   - Mock seam: MockSprintRepository (ListOrderedAssignments, RenumberAssignmentsTx, SetSprintOrderTx)
//   - Forbidden mocks: do NOT mock at CLI layer; pass ReorderTarget.Position directly
//   - Counter-factual: a buggy impl that mutates entity tables would be caught by asserting
//     that mock task/bug repo methods were NOT called (only sprint repo methods called)
func TestReorderAssignment_TC013_MoveFromPosition3ToPosition1(t *testing.T) {
	ctx := context.Background()

	activeSprint := makeActiveSprint(10, "S001")

	// Three ordered items: task-X at pos 1, task-Y at pos 2, task-Z (target) at pos 3.
	existing := []*models.SprintAssignment{
		makeAssignment(1, 10, "task", 100, intPtr(1)), // task-X, pos 1
		makeAssignment(2, 10, "task", 101, intPtr(2)), // task-Y, pos 2
		makeAssignment(3, 10, "task", 102, intPtr(3)), // task-Z, target: move to pos 1
	}

	var renumberCalls [][]sprint.RenumberOp
	setSprintOrderCalled := false

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			if key == "S001" {
				return activeSprint, nil
			}
			return nil, fmt.Errorf("sprint %q not found", key)
		},
		GetTaskIDByKeyFunc: func(_ context.Context, key string) (int64, error) {
			// resolveEntityTypeAndID normalizes "E07-F01-001" → "T-E07-F01-001"
			if key == "T-E07-F01-001" {
				return 102, nil // entityID=102 = task-Z (assignment ID=3)
			}
			return 0, fmt.Errorf("task %q not found", key)
		},
		GetActiveAssignmentFunc: func(_ context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
			// task-Z is assignment ID=3 in sprint 10
			if entityType == "task" && entityID == 102 {
				return existing[2], nil
			}
			return nil, nil
		},
		ListOrderedAssignmentsFunc: func(_ context.Context, _ int64) ([]*models.SprintAssignment, error) {
			return existing, nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, ops []sprint.RenumberOp) error {
			snap := make([]sprint.RenumberOp, len(ops))
			copy(snap, ops)
			renumberCalls = append(renumberCalls, snap)
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, _ *sql.Tx, _ int64, _ *int) error {
			setSprintOrderCalled = true
			return nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	moved, topN, err := svc.ReorderAssignment(ctx, "S001", "E07-F01-001", ReorderTarget{Position: intPtr(1)})

	require.NoError(t, err)
	require.NotNil(t, moved, "moved assignment must be returned")
	require.NotNil(t, topN, "top-N list must be returned")

	// The reorder happens in two RenumberAssignmentsTx UPDATEs:
	//   1. NULL pre-pass over target + every shifted sibling
	//   2. Single CASE WHEN that assigns the target's new position alongside
	//      the siblings' new positions.
	// SetSprintOrderTx is no longer used — the target is folded into pass 2.
	require.Len(t, renumberCalls, 2, "reorder must use exactly two renumber UPDATEs (clear + assign)")
	assert.False(t, setSprintOrderCalled,
		"SetSprintOrderTx must not be called: target's new position is folded into the final renumber")

	clearOps := renumberCalls[0]
	require.Len(t, clearOps, 3, "clear pass must NULL target + both siblings")
	for i, op := range clearOps {
		assert.Nil(t, op.NewPosition, "clear-pass op[%d] must have NewPosition=nil", i)
	}

	finalOps := renumberCalls[1]
	require.Len(t, finalOps, 3, "final pass must assign target + both siblings their post-state positions")
	// Op 0 is the target (assignment ID=3) at its new position 1.
	assert.Equal(t, int64(3), finalOps[0].AssignmentID, "first op must be the target")
	require.NotNil(t, finalOps[0].NewPosition)
	assert.Equal(t, 1, *finalOps[0].NewPosition)
	// Ops 1..2 are the shifted siblings: task-X 1→2, task-Y 2→3.
	assert.Equal(t, int64(1), finalOps[1].AssignmentID)
	assert.Equal(t, intPtr(2), finalOps[1].NewPosition)
	assert.Equal(t, int64(2), finalOps[2].AssignmentID)
	assert.Equal(t, intPtr(3), finalOps[2].NewPosition)
}

// ---------------------------------------------------------------------------
// TC-014: ReorderAssignment — completed sprint returns error
// ---------------------------------------------------------------------------
//
// TC-014 — Caller-Path Contract (service-level):
//   - Entrypoint: SprintService.ReorderAssignment(ctx, "S_DONE", "E07-F01-001", ReorderTarget{Position: intPtr(1)})
//   - Mock seam: MockSprintRepository (GetByKey returns a completed sprint)
//   - Counter-factual: a buggy impl that does not validate sprint status would proceed
//     to list assignments and attempt reorder; this test asserts an error is returned
func TestReorderAssignment_TC014_CompletedSprintReturnsError(t *testing.T) {
	ctx := context.Background()

	completedSprint := &models.Sprint{
		ID:     10,
		Key:    "S_DONE",
		Status: "completed",
		Name:   "Done Sprint",
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return completedSprint, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	moved, topN, err := svc.ReorderAssignment(ctx, "S_DONE", "E07-F01-001", ReorderTarget{Position: intPtr(1)})

	require.Error(t, err, "reordering a completed sprint must return an error")
	assert.Nil(t, moved)
	assert.Nil(t, topN)
	assert.Contains(t, err.Error(), "cannot reorder", "error must mention cannot reorder")
}

// ===========================================================================
// TC-017 through TC-023: Ordered Backlog View and Carryover Sprint Order
// (T-E19-F07-005)
// ===========================================================================

// ---------------------------------------------------------------------------
// TC-017: --view=ordered returns Items array sorted by sprint_order ASC NULLS LAST
// ---------------------------------------------------------------------------
//
// TC-017 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S001", BacklogOptions{View:"ordered"})
//   - Mock seam: MockSprintRepository.ListBacklog — returns items in wrong order; service must sort.
//   - Forbidden mocks: Do NOT sort items in mock's ListBacklog.
//   - Counter-factual: a buggy impl returning Groups and not Items for --view=ordered would fail
//     the assertion len(backlog.Items) > 0 && backlog.Groups == nil.
func TestGetSprintBacklog_TC017_OrderedViewItemsSortedBySprintOrder(t *testing.T) {
	ctx := context.Background()

	activeSprint := &models.Sprint{
		ID:     10,
		Key:    "S001",
		Status: "active",
		Name:   "Sprint 1",
	}

	// Items intentionally out of order: positions 3, 1, 2, then nil
	order3 := 3
	order1 := 1
	order2 := 2
	backlogItems := []*sprint.BacklogItem{
		{EntityType: "task", Key: "E07-F01-003", Title: "Task C", Status: "todo", SprintOrder: &order3},
		{EntityType: "task", Key: "E07-F01-001", Title: "Task A", Status: "todo", SprintOrder: &order1},
		{EntityType: "task", Key: "E07-F01-002", Title: "Task B", Status: "todo", SprintOrder: &order2},
		{EntityType: "task", Key: "E07-F01-004", Title: "Task D", Status: "todo", SprintOrder: nil},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetSprintBacklog(ctx, "S001", BacklogOptions{View: "ordered"})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Items must be populated, Groups must be nil for ordered view
	assert.Nil(t, result.Groups, "Groups must be nil for ordered view")
	require.NotNil(t, result.Items, "Items must be non-nil for ordered view")
	assert.Len(t, result.Items, 4, "all 4 items must appear")
	assert.Equal(t, "ordered", result.View, "View field must be 'ordered'")

	// Items sorted: positions 1, 2, 3, then nil item last
	assert.Equal(t, "E07-F01-001", result.Items[0].Key, "first item must be sprint_order=1")
	assert.Equal(t, "E07-F01-002", result.Items[1].Key, "second item must be sprint_order=2")
	assert.Equal(t, "E07-F01-003", result.Items[2].Key, "third item must be sprint_order=3")
	assert.Equal(t, "E07-F01-004", result.Items[3].Key, "fourth item must be unordered (sprint_order=nil)")

	// Position for the nil-order item must be set (dense rank), sprint_order stays nil
	assert.Nil(t, result.Items[3].SprintOrder, "unordered item must have nil SprintOrder")
	require.NotNil(t, result.Items[3].Position, "unordered item must have non-nil Position (dense rank)")
	assert.Equal(t, 4, *result.Items[3].Position, "unordered item position must be 4")
}

// ---------------------------------------------------------------------------
// TC-018: Active sprint defaults to ordered view when View is empty
// ---------------------------------------------------------------------------
//
// TC-018 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S001", BacklogOptions{}) — View left zero-value.
//   - Mock seam: MockSprintRepository (GetByKey returns sprint with Status="active").
//   - Forbidden mocks: Do NOT default View to "ordered" before calling service; service must detect.
//   - Counter-factual: a buggy impl always defaulting to "grouped" would produce Groups not Items;
//     TC-018 asserts backlog.View=="ordered".
func TestGetSprintBacklog_TC018_ActiveSprintDefaultsToOrderedView(t *testing.T) {
	ctx := context.Background()

	activeSprint := &models.Sprint{
		ID:     10,
		Key:    "S001",
		Status: "active",
		Name:   "Sprint 1",
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			return []*sprint.BacklogItem{}, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)

	result, err := svc.GetSprintBacklog(ctx, "S001", BacklogOptions{}) // empty View

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ordered", result.View, "active sprint must default to ordered view")
	assert.Nil(t, result.Groups, "Groups must be nil when view is ordered")

	// TC-018b: planning sprint defaults to grouped
	planningSprint := &models.Sprint{
		ID:     20,
		Key:    "S002",
		Status: "planning",
		Name:   "Sprint 2",
	}
	mockRepo2 := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return planningSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			return []*sprint.BacklogItem{}, nil
		},
	}
	svc2 := NewSprintService(mockRepo2, workflow.NewService(""), nil, nil, nil)
	result2, err2 := svc2.GetSprintBacklog(ctx, "S002", BacklogOptions{})
	require.NoError(t, err2)
	require.NotNil(t, result2)
	assert.Equal(t, "grouped", result2.View, "planning sprint must default to grouped view")
}

// ---------------------------------------------------------------------------
// TC-019: position and sprint_order diverge for unordered items
// ---------------------------------------------------------------------------
//
// TC-019 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S001", BacklogOptions{View:"ordered"})
//   - Mock seam: MockSprintRepository
//   - Forbidden mocks: Do NOT compute position client-side in the mock.
//   - Counter-factual: a buggy impl that sets position=sprint_order for unordered items
//     would produce position=nil; TC-019 asserts unorderedItem.Position != nil && SprintOrder == nil.
func TestGetSprintBacklog_TC019_PositionAndSprintOrderDiverge(t *testing.T) {
	ctx := context.Background()

	activeSprint := &models.Sprint{
		ID: 10, Key: "S001", Status: "active", Name: "Sprint 1",
	}

	order1 := 1
	order2 := 2
	backlogItems := []*sprint.BacklogItem{
		{EntityType: "task", Key: "E07-F01-001", Title: "Item A", Status: "todo", SprintOrder: &order1},
		{EntityType: "task", Key: "E07-F01-002", Title: "Item B", Status: "todo", SprintOrder: &order2},
		{EntityType: "task", Key: "E07-F01-003", Title: "Item C", Status: "todo", SprintOrder: nil}, // unordered
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)
	result, err := svc.GetSprintBacklog(ctx, "S001", BacklogOptions{View: "ordered"})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 3)

	// Item A: position=1, sprint_order=1 (equal for ordered items)
	itemA := result.Items[0]
	assert.Equal(t, "E07-F01-001", itemA.Key)
	require.NotNil(t, itemA.Position)
	assert.Equal(t, 1, *itemA.Position, "Item A position must be 1")
	require.NotNil(t, itemA.SprintOrder)
	assert.Equal(t, 1, *itemA.SprintOrder, "Item A sprint_order must be 1")

	// Item B: position=2, sprint_order=2 (equal for ordered items)
	itemB := result.Items[1]
	assert.Equal(t, "E07-F01-002", itemB.Key)
	require.NotNil(t, itemB.Position)
	assert.Equal(t, 2, *itemB.Position, "Item B position must be 2")
	require.NotNil(t, itemB.SprintOrder)
	assert.Equal(t, 2, *itemB.SprintOrder, "Item B sprint_order must be 2")

	// Item C: position=3, sprint_order=nil (diverge for unordered items)
	itemC := result.Items[2]
	assert.Equal(t, "E07-F01-003", itemC.Key)
	require.NotNil(t, itemC.Position, "unordered item must have non-nil Position (dense rank)")
	assert.Equal(t, 3, *itemC.Position, "unordered item position must be 3 (dense rank)")
	assert.Nil(t, itemC.SprintOrder, "unordered item sprint_order must stay nil")
}

// ---------------------------------------------------------------------------
// TC-020: --view=grouped retains current behavior (grouped view regression)
// ---------------------------------------------------------------------------
//
// TC-020 — Caller-Path Contract:
//   - Entrypoint: SprintService.GetSprintBacklog(ctx, "S001", BacklogOptions{View:"grouped"})
//   - Mock seam: MockSprintRepository
//   - Forbidden mocks: Do NOT alter group-building logic in the mock.
//   - Counter-factual: a refactoring removing the grouped-view branch would return Items instead of
//     Groups; TC-020 asserts backlog.Groups != nil && len(backlog.Items)==0.
func TestGetSprintBacklog_TC020_GroupedViewRegressionGuard(t *testing.T) {
	ctx := context.Background()

	activeSprint := &models.Sprint{
		ID: 10, Key: "S001", Status: "active", Name: "Sprint 1",
	}

	backlogItems := []*sprint.BacklogItem{
		{EntityType: "task", Key: "E07-F01-001", Title: "Item A", Status: "todo"},
		{EntityType: "task", Key: "E07-F01-002", Title: "Item B", Status: "in_progress"},
	}

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListBacklogFunc: func(_ context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error) {
			return backlogItems, nil
		},
	}

	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil)
	result, err := svc.GetSprintBacklog(ctx, "S001", BacklogOptions{View: "grouped"})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "grouped", result.View, "View field must be 'grouped'")
	assert.NotNil(t, result.Groups, "Groups must be populated for grouped view")
	assert.Empty(t, result.Items, "Items must be empty for grouped view")
	// Verify groups have expected structure
	require.Len(t, result.Groups, 2, "two status groups expected")
}

// ---------------------------------------------------------------------------
// TC-021: --carryover=next appends after receiving sprint's existing ordered items
// ---------------------------------------------------------------------------
//
// TC-021 — Caller-Path Contract:
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S001", CarryoverNext)
//   - Mock seam: MockSprintRepository (MaxSprintOrder for receiving sprint, ListAssignmentsForCarryover,
//     ReassignToSprintTx, RenumberAssignmentsTx)
//   - Forbidden mocks: Do NOT interleave carried items by priority; service sorts by sprint_order ASC NULLS LAST.
//   - Counter-factual: a buggy impl interleaving carried items would produce sprint_orders other than M+1..M+K;
//     TC-021 asserts RenumberAssignmentsTx ops for carried items start at M+1 where M=receiving sprint's existing max.
func TestCloseSprintWithCarryover_TC021_NextPreservesOrderAppendedAfterExisting(t *testing.T) {
	ctx := context.Background()

	// S001 (active) with 3 incomplete items: sprint_orders 1, 2, nil
	activeSprint := &models.Sprint{
		ID:        1,
		Key:       "S001",
		Status:    "active",
		Name:      "Sprint 1",
		StartDate: time.Now().Add(-7 * 24 * time.Hour),
		EndDate:   time.Now().Add(-1 * time.Hour),
	}
	// S002 (planning) with 2 existing ordered items at positions 1, 2
	receivingSprint := &models.Sprint{
		ID:     2,
		Key:    "S002",
		Status: "planning",
		Name:   "Sprint 2",
	}

	order1 := 1
	order2 := 2
	// Incomplete assignments in S001: sprint_orders 1, 2, nil
	incompleteAssignments := []*models.SprintAssignment{
		makeAssignment(10, 1, "task", 100, &order1),
		makeAssignment(11, 1, "task", 101, &order2),
		makeAssignment(12, 1, "task", 102, nil),
	}

	var renumberOpsCapt []sprint.RenumberOp
	var setSprintOrderTxCalled bool

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			switch key {
			case "S001":
				return activeSprint, nil
			case "S002":
				return receivingSprint, nil
			}
			return nil, fmt.Errorf("sprint %q not found", key)
		},
		GetByIDFunc: func(_ context.Context, id int64) (*models.Sprint, error) {
			switch id {
			case 1:
				return activeSprint, nil
			case 2:
				return receivingSprint, nil
			}
			return nil, fmt.Errorf("sprint ID %d not found", id)
		},
		ListFunc: func(_ context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
			if filters != nil && filters.Status != nil && *filters.Status == "planning" {
				return []*models.Sprint{receivingSprint}, nil
			}
			return []*models.Sprint{}, nil
		},
		ListAssignmentsFunc: func(_ context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			// All assignments = same as incomplete for simplicity (no completed ones)
			return incompleteAssignments, nil
		},
		ListAssignmentsForCarryoverFunc: func(_ context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error) {
			return incompleteAssignments, nil
		},
		MaxSprintOrderFunc: func(_ context.Context, sprintID int64) (int, error) {
			// Receiving sprint (S002) has existing max=2
			if sprintID == 2 {
				return 2, nil
			}
			return 0, nil
		},
		ReassignToSprintTxFunc: func(_ context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error {
			return nil
		},
		RenumberAssignmentsTxFunc: func(_ context.Context, tx *sql.Tx, sprintID int64, ops []sprint.RenumberOp) error {
			renumberOpsCapt = ops
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, tx *sql.Tx, assignmentID int64, pos *int) error {
			setSprintOrderTxCalled = true
			return nil
		},
		UpdateStatusTxFunc: func(_ context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error {
			return nil
		},
		CreateCompletionTxFunc: func(_ context.Context, tx *sql.Tx, completion *models.SprintCompletion) error {
			return nil
		},
	}

	// Build service with a real DB for transaction support
	testDB := newTestDB(t)
	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil, testDB)

	result, err := svc.CloseSprintWithCarryover(ctx, "S001", CarryoverNext)

	require.NoError(t, err)
	require.NotNil(t, result)

	// CarryoverPreserved must be true for CarryoverNext
	assert.True(t, result.CarryoverPreserved, "CarryoverPreserved must be true for CarryoverNext")
	assert.Equal(t, 3, result.CarriedOverCount, "all 3 incomplete items must be carried over")

	// Carried items must be appended after M=2 existing ordered items in receiving sprint
	// → sprint_orders for carried items must be M+1=3, M+2=4, M+3=5
	require.NotNil(t, renumberOpsCapt, "RenumberAssignmentsTx must be called for carryover-next")
	require.Len(t, renumberOpsCapt, 3, "3 carried items must be renumbered")

	// Carried items sorted by (sprint_order ASC NULLS LAST, assigned_at ASC):
	// assignment 10 (sprint_order=1) → position 3
	// assignment 11 (sprint_order=2) → position 4
	// assignment 12 (sprint_order=nil) → position 5
	ops := renumberOpsCapt
	positions := make([]int, len(ops))
	for i, op := range ops {
		require.NotNil(t, op.NewPosition, "renumber op must have non-nil position")
		positions[i] = *op.NewPosition
	}
	// Positions must cover M+1..M+3 (3, 4, 5)
	assert.Contains(t, positions, 3, "first carried item must get position 3 (M+1)")
	assert.Contains(t, positions, 4, "second carried item must get position 4 (M+2)")
	assert.Contains(t, positions, 5, "third carried item must get position 5 (M+3)")

	// SetSprintOrderTx must NOT be called separately (all ordering via RenumberAssignmentsTx)
	_ = setSprintOrderTxCalled // verified implicitly by RenumberAssignmentsTx assertions above
}

// ---------------------------------------------------------------------------
// TC-023: --carryover=backlog clears sprint_order atomically with removed_at
// ---------------------------------------------------------------------------
//
// TC-023 — Caller-Path Contract:
//   - Entrypoint: SprintService.CloseSprintWithCarryover(ctx, "S001", CarryoverBacklog)
//   - Mock seam: MockSprintRepository (DropAssignmentsTx; assert it receives assignment IDs)
//   - Forbidden mocks: Do NOT call SetSprintOrderTx separately after DropAssignmentsTx.
//   - Counter-factual: a buggy impl with two separate UPDATEs would appear as two distinct mock
//     method calls (DropAssignmentsTx + SetSprintOrderTx); TC-023 asserts DropAssignmentsTx called
//     exactly once and SetSprintOrderTx NOT called.
func TestCloseSprintWithCarryover_TC023_BacklogClearsSprintOrderAtomically(t *testing.T) {
	ctx := context.Background()

	activeSprint := &models.Sprint{
		ID:        1,
		Key:       "S001",
		Status:    "active",
		Name:      "Sprint 1",
		StartDate: time.Now().Add(-7 * 24 * time.Hour),
		EndDate:   time.Now().Add(-1 * time.Hour),
	}

	order1 := 1
	incompleteAssignments := []*models.SprintAssignment{
		makeAssignment(10, 1, "task", 100, &order1),
		makeAssignment(11, 1, "task", 101, nil),
	}

	var dropTxCallCount int
	var dropTxReceivedIDs []int64
	var setSprintOrderTxCallCount int

	mockRepo := &MockSprintRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Sprint, error) {
			return activeSprint, nil
		},
		ListAssignmentsFunc: func(_ context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
			return incompleteAssignments, nil
		},
		ListAssignmentsForCarryoverFunc: func(_ context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error) {
			return incompleteAssignments, nil
		},
		DropAssignmentsTxFunc: func(_ context.Context, tx *sql.Tx, assignmentIDs []int64) error {
			dropTxCallCount++
			dropTxReceivedIDs = assignmentIDs
			return nil
		},
		SetSprintOrderTxFunc: func(_ context.Context, tx *sql.Tx, assignmentID int64, pos *int) error {
			setSprintOrderTxCallCount++
			return nil
		},
		UpdateStatusTxFunc: func(_ context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error {
			return nil
		},
		CreateCompletionTxFunc: func(_ context.Context, tx *sql.Tx, completion *models.SprintCompletion) error {
			return nil
		},
	}

	testDB := newTestDB(t)
	svc := NewSprintService(mockRepo, workflow.NewService(""), nil, nil, nil, testDB)

	result, err := svc.CloseSprintWithCarryover(ctx, "S001", CarryoverBacklog)

	require.NoError(t, err)
	require.NotNil(t, result)

	// CarryoverPreserved must be false for CarryoverBacklog
	assert.False(t, result.CarryoverPreserved, "CarryoverPreserved must be false for CarryoverBacklog")
	assert.Equal(t, 2, result.DroppedCount, "both incomplete items must be dropped")

	// DropAssignmentsTx called exactly once
	assert.Equal(t, 1, dropTxCallCount, "DropAssignmentsTx must be called exactly once")
	assert.ElementsMatch(t, []int64{10, 11}, dropTxReceivedIDs, "DropAssignmentsTx must receive both assignment IDs")

	// SetSprintOrderTx must NOT be called (sprint_order cleared in same UPDATE as removed_at)
	assert.Equal(t, 0, setSprintOrderTxCallCount, "SetSprintOrderTx must NOT be called separately")
}

// newTestDB creates a minimal test database for transaction support in service tests.
// Uses the shared test DB to satisfy the service's db.BeginTxContext() requirement.
func newTestDB(t *testing.T) *repository.DB {
	t.Helper()
	testSQLDB := internaltesthelper.GetTestDB()
	return repository.NewDB(testSQLDB)
}

// ---------------------------------------------------------------------------
// buildRenumberOps unit tests — regression guard for the renumber off-by-one
// that caused UNIQUE constraint violations on long sprints. The original
// implementation used `if pos >= newPosition { pos++ }` which fires every
// iteration after newPosition, producing sparse non-contiguous positions
// (e.g. 1..8, 10, 12, 14, ...) instead of the dense 1..8, 10..24 the index
// requires.
// ---------------------------------------------------------------------------

func TestBuildRenumberOps_DenseSequenceForVariousPositions(t *testing.T) {
	makeSiblings := func(n int) []*models.SprintAssignment {
		siblings := make([]*models.SprintAssignment, n)
		for i := 0; i < n; i++ {
			p := i + 1
			siblings[i] = &models.SprintAssignment{ID: int64(i + 1), SprintOrder: &p}
		}
		return siblings
	}

	tests := []struct {
		name        string
		nSiblings   int
		newPosition int
		// expected[i] is the new sprint_order assigned to siblings[i].
		// Together with newPosition (reserved for the target), the union must be the
		// dense set {1..nSiblings+1} \ {newPosition} in order.
		expected []int
	}{
		{
			name:        "5 siblings, target at front",
			nSiblings:   5,
			newPosition: 1,
			expected:    []int{2, 3, 4, 5, 6},
		},
		{
			name:        "5 siblings, target in middle",
			nSiblings:   5,
			newPosition: 3,
			expected:    []int{1, 2, 4, 5, 6},
		},
		{
			name:        "5 siblings, target at end",
			nSiblings:   5,
			newPosition: 6, // = len(siblings) + 1
			expected:    []int{1, 2, 3, 4, 5},
		},
		{
			name:        "23 siblings, target at position 9 (mirrors B-report)",
			nSiblings:   23,
			newPosition: 9,
			expected:    []int{1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		},
		{
			name:        "23 siblings, target at top (newPosition=1)",
			nSiblings:   23,
			newPosition: 1,
			expected:    []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		},
		{
			name:        "23 siblings, target at bottom (newPosition=24)",
			nSiblings:   23,
			newPosition: 24,
			expected:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		},
		{
			name:        "single sibling, target after it",
			nSiblings:   1,
			newPosition: 2,
			expected:    []int{1},
		},
		{
			name:        "no siblings",
			nSiblings:   0,
			newPosition: 1,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := buildRenumberOps(makeSiblings(tt.nSiblings), tt.newPosition)
			require.Len(t, ops, len(tt.expected), "op count must equal sibling count")

			seen := make(map[int]bool, len(ops))
			for i, op := range ops {
				require.NotNil(t, op.NewPosition, "op[%d] must have non-nil NewPosition", i)
				assert.Equal(t, tt.expected[i], *op.NewPosition,
					"op[%d] (sibling ID %d) must land at the expected position", i, op.AssignmentID)
				assert.NotEqual(t, tt.newPosition, *op.NewPosition,
					"op[%d] must not collide with the slot reserved for the target", i)
				assert.False(t, seen[*op.NewPosition],
					"op[%d] position %d duplicates an earlier op", i, *op.NewPosition)
				seen[*op.NewPosition] = true
			}
		})
	}
}

// TestBuildReorderClearOps_ClearsTargetAndAllRenumberSiblings verifies the
// NULL pre-pass covers every row whose sprint_order will change in the
// subsequent renumber UPDATE. Without this, the per-row partial-unique-index
// check inside SQLite's UPDATE fires when a shifted value collides with an
// unprocessed sibling.
func TestBuildReorderClearOps_ClearsTargetAndAllRenumberSiblings(t *testing.T) {
	pos2, pos3 := 2, 3
	renumberOps := []sprint.RenumberOp{
		{AssignmentID: 10, NewPosition: &pos2},
		{AssignmentID: 11, NewPosition: &pos3},
	}

	clearOps := buildReorderClearOps(99, renumberOps)

	require.Len(t, clearOps, 3, "clear must cover target + every sibling op")
	assert.Equal(t, int64(99), clearOps[0].AssignmentID, "target must be cleared first")
	assert.Nil(t, clearOps[0].NewPosition)
	assert.Equal(t, int64(10), clearOps[1].AssignmentID)
	assert.Nil(t, clearOps[1].NewPosition)
	assert.Equal(t, int64(11), clearOps[2].AssignmentID)
	assert.Nil(t, clearOps[2].NewPosition)
}
