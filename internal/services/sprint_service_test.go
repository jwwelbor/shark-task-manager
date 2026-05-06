package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
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
				Status: models.SprintStatus("todo"),
			},
			mockError: nil,
			expectErr: false,
		},
		{
			name:       "non-existent sprint returns error",
			key:        "S999",
			mockResult: nil,
			mockError:  errors.New("sprint not found"),
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
				{ID: 1, Key: "S001", Name: "Sprint 1", Status: "todo"},
				{ID: 2, Key: "S002", Name: "Sprint 2", Status: "in_progress"},
			},
			expectCount: 2,
		},
		{
			name:    "filter by status",
			filters: &SprintListFilters{Status: "in_progress"},
			mockResult: []*models.Sprint{
				{ID: 2, Key: "S002", Name: "Sprint 2", Status: "in_progress"},
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
						Status:    "todo",
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
			status:    "todo",
			expectErr: false,
		},
		{
			name:      "delete active sprint fails",
			sprintKey: "S001",
			status:    "in_progress",
			expectErr: true,
			errMsg:    "only sprints in todo status can be deleted",
		},
		{
			name:      "delete closing sprint fails",
			sprintKey: "S001",
			status:    "ready_for_review",
			expectErr: true,
			errMsg:    "only sprints in todo status can be deleted",
		},
		{
			name:      "delete completed sprint fails",
			sprintKey: "S001",
			status:    "completed",
			expectErr: true,
			errMsg:    "only sprints in todo status can be deleted",
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

// TestSprintService_StartSprint tests single-active constraint (AC-6).
func TestSprintService_StartSprint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                  string
		currentStatus         models.SprintStatus
		existingActiveSprints []*models.Sprint
		expectErr             bool
		errMsg                string
	}{
		{
			name:          "start planning sprint succeeds",
			currentStatus: "todo",
			expectErr:     false,
		},
		{
			name:          "start when another is active fails",
			currentStatus: "todo",
			existingActiveSprints: []*models.Sprint{
				{ID: 2, Key: "S002", Name: "Active Sprint", Status: "in_progress"},
			},
			expectErr: true,
			errMsg:    "cannot activate",
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
					// First call returns planning sprint, second returns active sprint
					callCount++
					if callCount == 1 {
						return &models.Sprint{
							ID:     1,
							Key:    "S001",
							Name:   "Sprint 1",
							Status: tt.currentStatus,
						}, nil
					}
					// After status update, return with active status
					return &models.Sprint{
						ID:     1,
						Key:    "S001",
						Name:   "Sprint 1",
						Status: "in_progress",
					}, nil
				},
				ListFunc: func(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
					return tt.existingActiveSprints, nil
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
				assert.Equal(t, "in_progress", string(result.Status))
			}
		})
	}
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
			currentStatus: "in_progress",
			expectErr:     false,
		},
		{
			name:          "close planning sprint fails",
			currentStatus: "todo",
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
						Status: "ready_for_review",
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
				assert.Equal(t, "ready_for_review", string(result.Status))
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
			name:          "archive_ready_for_review_sprint_succeeds",
			currentStatus: "ready_for_review",
			expectErr:     false,
		},
		{
			name:          "archive active sprint fails",
			currentStatus: "in_progress",
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
						Status: "completed",
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
				assert.Equal(t, "completed", string(result.Status))
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
				makeItem("bug", "completed"),
				makeItem("change_card", "completed"),
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
			return nil, fmt.Errorf("sprint not found: %q", key)
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
			return nil, fmt.Errorf("sprint not found: %q", key)
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
			return nil, fmt.Errorf("sprint not found: %q", key)
		},
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Sprint, error) {
			if id == 24 {
				return &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}, nil
			}
			return nil, fmt.Errorf("sprint not found with id %d", id)
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
			return nil, fmt.Errorf("sprint not found: %q", key)
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

// TestBulkAddToSprint_BugsBulk tests TC-K04:
// BulkAddToSprint with EntityTypes:["bug"] assigns open unassigned bugs.
//
// Caller-Path Contract (per test-plan TC-K04):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["bug"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Counter-factual: a buggy impl that passes the wrong entity type to ListUnassignedBacklog
//     would return tasks instead of bugs, making AddedByType["bug"] == 0
func TestBulkAddToSprint_BugsBulk(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	backlogItems := []sprint.BacklogItem{
		{EntityType: "bug", EntityID: 2001, Key: "B001", Status: "open"},
		{EntityType: "bug", EntityID: 2002, Key: "B002", Status: "in_progress"},
	}

	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
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

	workflowSvc := workflow.NewService("")
	svc := NewSprintService(mockRepo, workflowSvc, mockAssignRepo, nil, nil)

	result, err := svc.BulkAddToSprint(ctx, BulkAddInput{
		SprintKey:   "S024",
		EntityTypes: []string{"bug"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{"bug"}, capturedEntityTypes, "service must pass 'bug' type to ListUnassignedBacklog")
	assert.Equal(t, 2, result.AddedByType["bug"], "both unassigned bugs should be added")
	assert.Equal(t, 0, len(result.CapacityWarnings), "no capacity configured — no warnings")
}

// TestBulkAddToSprint_TechDebtBulk tests TC-K05:
// BulkAddToSprint with EntityTypes:["tech_debt"] assigns open tech-debt items.
//
// Caller-Path Contract (per test-plan TC-K05):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["tech_debt"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Counter-factual: a buggy impl that uses entity_type="techdebt" (typo) would fail
//     ListUnassignedBacklog to return the correct items, making AddedByType["tech_debt"] == 0
func TestBulkAddToSprint_TechDebtBulk(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	backlogItems := []sprint.BacklogItem{
		{EntityType: "tech_debt", EntityID: 3001, Key: "TD-001", Status: "open"},
		{EntityType: "tech_debt", EntityID: 3002, Key: "TD-002", Status: "open"},
		{EntityType: "tech_debt", EntityID: 3003, Key: "TD-003", Status: "open"},
	}

	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
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
	assert.Equal(t, 3, result.AddedByType["tech_debt"], "all 3 tech-debt items should be added")
}

// TestBulkAddToSprint_ChangeCardsBulk tests TC-K06:
// BulkAddToSprint with EntityTypes:["change_card"] assigns open change-cards.
//
// Caller-Path Contract (per test-plan TC-K06):
//   - Entrypoint: SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", EntityTypes:["change_card"]})
//   - Lowest mock seam: SprintRepository + SprintAssignmentQueryRepository interfaces
//   - Counter-factual: a buggy impl that uses entity_type="change" (wrong string) would return
//     wrong items, making AddedByType["change_card"] != 2
func TestBulkAddToSprint_ChangeCardsBulk(t *testing.T) {
	ctx := context.Background()

	sprint1 := &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "planning"}

	backlogItems := []sprint.BacklogItem{
		{EntityType: "change_card", EntityID: 4001, Key: "C001", Status: "open"},
		{EntityType: "change_card", EntityID: 4002, Key: "C002", Status: "open"},
	}

	var capturedEntityTypes []string

	mockAssignRepo := &MockSprintAssignmentQueryRepository{
		ListUnassignedBacklogFunc: func(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error) {
			capturedEntityTypes = entityTypes
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
	assert.Equal(t, 2, result.AddedByType["change_card"], "both change cards should be added")
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
			return &models.Sprint{ID: 24, Key: "S024", Name: "Sprint 24", Status: "todo"}, nil
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
