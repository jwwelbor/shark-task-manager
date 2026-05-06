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
	CreateFunc       func(ctx context.Context, s *models.Sprint) error
	GetByKeyFunc     func(ctx context.Context, key string) (*models.Sprint, error)
	GetByIDFunc      func(ctx context.Context, id int64) (*models.Sprint, error)
	UpdateFunc       func(ctx context.Context, s *models.Sprint) error
	DeleteFunc       func(ctx context.Context, id int64) error
	UpdateStatusFunc func(ctx context.Context, id int64, status models.SprintStatus) error
	GetNextKeyFunc   func(ctx context.Context) (string, error)
	ListFunc         func(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error)

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
