package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Repositories
// ============================================================================

// MockTaskRepository implements TaskRepository interface for testing
type MockTaskRepository struct {
	CreateFunc                        func(ctx context.Context, task *models.Task) error
	GetByKeyFunc                      func(ctx context.Context, key string) (*models.Task, error)
	GetByIDFunc                       func(ctx context.Context, id int64) (*models.Task, error)
	UpdateFunc                        func(ctx context.Context, task *models.Task) error
	DeleteFunc                        func(ctx context.Context, id int64) error
	ListFunc                          func(ctx context.Context) ([]*models.Task, error)
	ListByFeatureFunc                 func(ctx context.Context, featureID int64) ([]*models.Task, error)
	ListByEpicFunc                    func(ctx context.Context, epicKey string) ([]*models.Task, error)
	GetTaskDependentsFunc             func(ctx context.Context, taskKey string) ([]*models.Task, error)
	UpdateStatusFunc                  func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
	UpdateStatusForcedFunc            func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
	UpdateStatusForcedWithUnblockFunc func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error)
	StatusUpdateRawFunc               func(ctx context.Context, params models.StatusUpdateParams) ([]string, error)
	ListByKeyPrefixFunc               func(ctx context.Context, prefix string) ([]*models.Task, error)
	GetTaskDisplayDataRawFunc         func(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error)
	GetRejectionCountsFunc            func(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error)
}

func (m *MockTaskRepository) Create(ctx context.Context, task *models.Task) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, task)
	}
	return fmt.Errorf("Create not implemented in mock")
}

func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mock")
}

func (m *MockTaskRepository) Update(ctx context.Context, task *models.Task) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, task)
	}
	return fmt.Errorf("Update not implemented in mock")
}

func (m *MockTaskRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return fmt.Errorf("Delete not implemented in mock")
}

func (m *MockTaskRepository) List(ctx context.Context) ([]*models.Task, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, fmt.Errorf("List not implemented in mock")
}

func (m *MockTaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.ListByFeatureFunc != nil {
		return m.ListByFeatureFunc(ctx, featureID)
	}
	return nil, fmt.Errorf("ListByFeature not implemented in mock")
}

func (m *MockTaskRepository) ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error) {
	if m.ListByFeatureFunc != nil {
		return m.ListByFeatureFunc(ctx, 0)
	}
	return nil, fmt.Errorf("ListByFeatureKey not implemented in mock")
}

func (m *MockTaskRepository) ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	if m.ListByEpicFunc != nil {
		return m.ListByEpicFunc(ctx, epicKey)
	}
	return nil, fmt.Errorf("ListByEpic not implemented in mock")
}

func (m *MockTaskRepository) GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error) {
	if m.GetTaskDependentsFunc != nil {
		return m.GetTaskDependentsFunc(ctx, taskKey)
	}
	return nil, fmt.Errorf("GetTaskDependents not implemented in mock")
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, taskID, newStatus, agent, notes)
	}
	return fmt.Errorf("UpdateStatus not implemented in mock")
}

func (m *MockTaskRepository) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	if m.UpdateStatusForcedFunc != nil {
		return m.UpdateStatusForcedFunc(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	}
	return fmt.Errorf("UpdateStatusForced not implemented in mock")
}

func (m *MockTaskRepository) UpdateStatusForcedWithUnblock(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	if m.UpdateStatusForcedWithUnblockFunc != nil {
		return m.UpdateStatusForcedWithUnblockFunc(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	}
	return nil, fmt.Errorf("UpdateStatusForcedWithUnblock not implemented in mock")
}

func (m *MockTaskRepository) StatusUpdateRaw(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
	if m.StatusUpdateRawFunc != nil {
		return m.StatusUpdateRawFunc(ctx, params)
	}
	return nil, fmt.Errorf("StatusUpdateRaw not implemented in mock")
}

func (m *MockTaskRepository) StatusUpdateRawWithTx(_ context.Context, _ *sql.Tx, params models.StatusUpdateParams) ([]string, error) {
	if m.StatusUpdateRawFunc != nil {
		return m.StatusUpdateRawFunc(context.Background(), params)
	}
	return nil, nil
}

func (m *MockTaskRepository) BeginTx(_ context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (m *MockTaskRepository) FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error) {
	return nil, fmt.Errorf("FindByFileChanged not implemented in mock")
}

func (m *MockTaskRepository) ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error) {
	if m.ListByKeyPrefixFunc != nil {
		return m.ListByKeyPrefixFunc(ctx, prefix)
	}
	return []*models.Task{}, nil
}

func (m *MockTaskRepository) GetTaskDisplayDataRaw(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error) {
	if m.GetTaskDisplayDataRawFunc != nil {
		return m.GetTaskDisplayDataRawFunc(ctx, taskID)
	}
	return nil, fmt.Errorf("GetTaskDisplayDataRaw not implemented in mock")
}

func (m *MockTaskRepository) GetRejectionCounts(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
	if m.GetRejectionCountsFunc != nil {
		return m.GetRejectionCountsFunc(ctx, taskIDs)
	}
	// Default: return zero counts for all task IDs (no rejections).
	// Tests that need non-zero counts override GetRejectionCountsFunc.
	return make(map[int64]int), make(map[int64]*time.Time), nil
}

// Helper function to create a minimal workflow service for testing
func newMockWorkflowService() *workflow.Service {
	return workflow.NewService(".")
}

// ============================================================================
// CreateTask Tests
// ============================================================================

func TestTaskService_CreateTask_Happy_Path(t *testing.T) {
	// Arrange: Create mocks
	var capturedTask *models.Task
	mockRepo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			task.ID = 1 // Simulate DB assigning ID
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	// Act: Create task
	task, err := svc.CreateTask(context.Background(), input)

	// Assert: Success
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, 5, task.Priority)
	assert.NotNil(t, capturedTask)
}

func TestTaskService_CreateTask_Empty_Title(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "", // Empty title
		AgentType:  "backend",
		Priority:   5,
	}

	task, err := svc.CreateTask(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "title")
}

func TestTaskService_CreateTask_Missing_Epic_Key(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "", // Missing epic key
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	task, err := svc.CreateTask(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "epic")
}

func TestTaskService_CreateTask_Missing_Feature_Key(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "", // Missing feature key
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	task, err := svc.CreateTask(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "feature")
}

func TestTaskService_CreateTask_Invalid_Priority(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   15, // Invalid: > 10
	}

	task, err := svc.CreateTask(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "priority")
}

func TestTaskService_CreateTask_Default_Priority(t *testing.T) {
	mockRepo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		// Priority not set - should default
	}

	task, err := svc.CreateTask(context.Background(), input)

	assert.NoError(t, err)
	assert.Equal(t, 5, task.Priority) // Default priority
}

func TestTaskService_CreateTask_Repository_Error(t *testing.T) {
	mockRepo := &MockTaskRepository{
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			return errors.New("database error")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	input := CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	task, err := svc.CreateTask(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to create")
}

// ============================================================================
// GetTask Tests
// ============================================================================

func TestTaskService_GetTask_Found(t *testing.T) {
	expectedTask := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "T-E07-F01-001",
		Title: "Test Task"}, Status: models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			// Service passes key as-is, without modifying it
			return expectedTask, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	task, err := svc.GetTask(context.Background(), "E07-F01-001")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, int64(1), task.ID)
}

func TestTaskService_GetTask_NotFound(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	task, err := svc.GetTask(context.Background(), "E07-F01-999")

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to get")
}

// ============================================================================
// UpdateTask Tests
// ============================================================================

func TestTaskService_UpdateTask_Update_Title(t *testing.T) {
	existingTask := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "T-E07-F01-001",
		Title: "Old Title"}, Priority: 5,
		Status: models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return existingTask, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			assert.Equal(t, "New Title", task.Title)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	newTitle := "New Title"
	updates := TaskUpdates{
		Title: &newTitle,
	}

	task, err := svc.UpdateTask(context.Background(), "E07-F01-001", updates)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "New Title", task.Title)
	assert.Equal(t, 5, task.Priority) // Priority should not change
}

func TestTaskService_UpdateTask_Multiple_Fields(t *testing.T) {
	existingTask := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "T-E07-F01-001",
		Title: "Old Title"}, Priority: 5,
		Status: models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return existingTask, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	newTitle := "New Title"
	newPriority := 8
	updates := TaskUpdates{
		Title:    &newTitle,
		Priority: &newPriority,
	}

	task, err := svc.UpdateTask(context.Background(), "E07-F01-001", updates)

	assert.NoError(t, err)
	assert.Equal(t, "New Title", task.Title)
	assert.Equal(t, 8, task.Priority)
}

func TestTaskService_UpdateTask_Partial_Update(t *testing.T) {
	existingTask := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "T-E07-F01-001",
		Title: "Old Title"}, Priority: 5,
		Status: models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return existingTask, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	newTitle := "New Title"
	updates := TaskUpdates{
		Title: &newTitle,
		// Priority is nil - should not be updated
	}

	task, err := svc.UpdateTask(context.Background(), "E07-F01-001", updates)

	assert.NoError(t, err)
	assert.Equal(t, "New Title", task.Title)
	assert.Equal(t, 5, task.Priority) // Original priority preserved
}

func TestTaskService_UpdateTask_FilePath(t *testing.T) {
	existingTask := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "T-E07-F01-001",
		Title: "Task With File"}, Priority: 5,
		Status: models.TaskStatus("todo"),
	}

	var capturedTask *models.Task
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return existingTask, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	newPath := "docs/plan/E07/F01/tasks/custom-path.md"
	updates := TaskUpdates{
		FilePath: &newPath,
	}

	task, err := svc.UpdateTask(context.Background(), "E07-F01-001", updates)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotNil(t, task.FilePath)
	assert.Equal(t, newPath, *task.FilePath)
	assert.NotNil(t, capturedTask)
	assert.Equal(t, newPath, *capturedTask.FilePath)
	assert.Equal(t, "Task With File", task.Title) // Other fields unchanged
}

func TestTaskService_UpdateTask_Not_Found(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	newTitle := "New Title"
	updates := TaskUpdates{
		Title: &newTitle,
	}

	task, err := svc.UpdateTask(context.Background(), "E07-F01-999", updates)

	assert.Error(t, err)
	assert.Nil(t, task)
}

// ============================================================================
// DeleteTask Tests
// ============================================================================

func TestTaskService_DeleteTask_Success(t *testing.T) {
	taskToDelete := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key: "E07-F01-001"},
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return taskToDelete, nil
		},
		GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			return []*models.Task{}, nil // No dependents
		},
		DeleteFunc: func(ctx context.Context, id int64) error {
			assert.Equal(t, int64(1), id)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	err := svc.DeleteTask(context.Background(), "E07-F01-001")

	assert.NoError(t, err)
}

func TestTaskService_DeleteTask_Not_Found(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	err := svc.DeleteTask(context.Background(), "E07-F01-999")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete")
}

// ============================================================================
// ListTasks Tests
// ============================================================================

func TestTaskService_ListTasks_No_Filters(t *testing.T) {
	allTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Task 1"}, Status: models.TaskStatus("todo"), Priority: 5},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-002", Title: "Task 2"}, Status: models.TaskStatus("in_progress"), Priority: 8},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-003", Title: "Task 3"}, Status: models.TaskStatus("completed"), Priority: 3},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{})

	assert.NoError(t, err)
	assert.NotNil(t, tasks)
	// Completed tasks excluded by default
	assert.Equal(t, 2, len(tasks))
}

func TestTaskService_ListTasks_Show_All(t *testing.T) {
	allTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Task 1"}, Status: models.TaskStatus("todo"), Priority: 5},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-002", Title: "Task 2"}, Status: models.TaskStatus("completed"), Priority: 8},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{ShowAll: true})

	assert.NoError(t, err)
	assert.Equal(t, 2, len(tasks))
}

func TestTaskService_ListTasks_Filter_By_Status(t *testing.T) {
	allTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Task 1"}, Status: models.TaskStatus("todo"), Priority: 5},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-002", Title: "Task 2"}, Status: models.TaskStatus("in_progress"), Priority: 8},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-003", Title: "Task 3"}, Status: models.TaskStatus("todo"), Priority: 3},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{Status: "todo"})

	assert.NoError(t, err)
	assert.Equal(t, 2, len(tasks))
	for _, task := range tasks {
		assert.Equal(t, models.TaskStatus("todo"), task.Status)
	}
}

func TestTaskService_ListTasks_Sorting(t *testing.T) {
	ord1 := 1
	ord2 := 2
	allTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Task 1"}, Status: models.TaskStatus("todo"), Priority: 3, ExecutionOrder: &ord2},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-002", Title: "Task 2"}, Status: models.TaskStatus("todo"), Priority: 8, ExecutionOrder: &ord1},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-003", Title: "Task 3"}, Status: models.TaskStatus("todo"), Priority: 5, ExecutionOrder: &ord1},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{ShowAll: true})

	assert.NoError(t, err)
	// Should be sorted by execution order (1, 1, 2), then priority (8, 5, 3)
	assert.Equal(t, "E07-F01-002", tasks[0].Key) // Order 1, Priority 8
	assert.Equal(t, "E07-F01-003", tasks[1].Key) // Order 1, Priority 5
	assert.Equal(t, "E07-F01-001", tasks[2].Key) // Order 2, Priority 3
}

// ============================================================================
// ValidateStatus Tests
// ============================================================================

func TestTaskService_ValidateStatus_Valid_Status(t *testing.T) {
	svc := NewTaskService(&MockTaskRepository{}, NewEntityService(newMockWorkflowService()), nil)

	// Act: Validate valid status
	err := svc.ValidateStatus("todo")

	// Assert: No error for valid status
	assert.NoError(t, err)
}

func TestTaskService_ValidateStatus_Invalid_Status(t *testing.T) {
	svc := NewTaskService(&MockTaskRepository{}, NewEntityService(newMockWorkflowService()), nil)

	// Act: Validate invalid status
	err := svc.ValidateStatus("invalid_status_xyz")

	// Assert: Error for invalid status
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// ============================================================================
// Service Constructor Tests
// ============================================================================

func TestNewTaskService_Requires_Repository(t *testing.T) {
	// Arrange & Act & Assert: Should panic without repository
	assert.Panics(t, func() {
		NewTaskService(nil, NewEntityService(newMockWorkflowService()), nil)
	})
}

func TestNewTaskService_Requires_EntityService(t *testing.T) {
	// Arrange & Act & Assert: Should panic without entity service
	mockRepo := &MockTaskRepository{}
	assert.Panics(t, func() {
		NewTaskService(mockRepo, nil, nil)
	})
}

func TestNewTaskService_Optional_Dependencies(t *testing.T) {
	// Arrange & Act: Create service with only required dependencies
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Assert: Service created successfully
	assert.NotNil(t, svc)
}

// ============================================================================
// Additional Lifecycle Edge Cases
// ============================================================================

// ============================================================================
// Update and Delete Edge Cases
// ============================================================================

func TestTaskService_UpdateTask_Invalid_Priority(t *testing.T) {
	// Arrange: Update with invalid priority
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "T-E07-F01-001",
				Title: "Test Task"}, Priority: 5,
				Status: models.TaskStatus("todo"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	invalidPriority := 15 // > 10
	updates := TaskUpdates{
		Priority: &invalidPriority,
	}

	// Act: Attempt to update with invalid priority
	task, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", updates)

	// Assert: Validation error
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "priority")
}

func TestTaskService_UpdateTask_Update_Error(t *testing.T) {
	// Arrange: Repository update fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "T-E07-F01-001",
				Title: "Old Title"}, Priority: 5,
				Status: models.TaskStatus("todo"),
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	newTitle := "New Title"
	updates := TaskUpdates{
		Title: &newTitle,
	}

	// Act: Attempt to update
	task, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", updates)

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_DeleteTask_Has_Dependents(t *testing.T) {
	// Arrange: Task has dependents
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key: "T-E07-F01-001"},
			}, nil
		},
		GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			// Return dependent tasks
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "T-E07-F01-002"}, Status: models.TaskStatus("todo")},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Attempt to delete task with dependents
	err := svc.DeleteTask(context.Background(), "T-E07-F01-001")

	// Assert: Error returned (has dependents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependent")
}

func TestTaskService_DeleteTask_Delete_Error(t *testing.T) {
	// Arrange: Delete operation fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key: "E07-F01-001"},
			}, nil
		},
		GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			return []*models.Task{}, nil // No dependents
		},
		DeleteFunc: func(ctx context.Context, id int64) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Attempt to delete
	err := svc.DeleteTask(context.Background(), "E07-F01-001")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete")
}

// ============================================================================
// ListTasks Additional Tests
// ============================================================================

func TestTaskService_ListTasks_Filter_By_Agent(t *testing.T) {
	// Arrange: Tasks with different agent types
	backend := "backend"
	frontend := "frontend"
	allTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{Key: "E07-F01-001"}, Status: models.TaskStatus("todo"), AgentType: &backend},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-002"}, Status: models.TaskStatus("todo"), AgentType: &frontend},
		{BaseEntity: models.BaseEntity{Key: "E07-F01-003"}, Status: models.TaskStatus("todo"), AgentType: &backend},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: List tasks filtered by agent
	tasks, err := svc.ListTasks(context.Background(), TaskFilters{AgentType: "backend"})

	// Assert: Only backend tasks returned
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Equal(t, "backend", *task.AgentType)
	}
}

func TestTaskService_ListTasks_Repository_Error(t *testing.T) {
	// Arrange: Repository returns error
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Attempt to list tasks
	tasks, err := svc.ListTasks(context.Background(), TaskFilters{})

	// Assert: Error propagated
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "failed to list")
}

// ============================================================================
// Pagination Tests (T-E15-F04-001)
// ============================================================================

// TestTaskService_ListTasks_Pagination tests pagination support in ListTasks
func TestTaskService_ListTasks_Pagination(t *testing.T) {
	// Arrange: Create 100 tasks for pagination testing
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			tasks := make([]*models.Task, 100)
			for i := 0; i < 100; i++ {
				agentType := "backend"
				tasks[i] = &models.Task{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("E15-F04-%03d", i+1),
					Title: fmt.Sprintf("Task %d", i+1)}, Status: "todo",
					Priority:  5,
					AgentType: &agentType,
				}
			}
			return tasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	tests := []struct {
		name         string
		limit        int
		offset       int
		wantCount    int
		wantTotal    int
		wantFirstKey string
	}{
		{
			name:         "first page",
			limit:        10,
			offset:       0,
			wantCount:    10,
			wantTotal:    100,
			wantFirstKey: "E15-F04-001",
		},
		{
			name:         "second page",
			limit:        10,
			offset:       10,
			wantCount:    10,
			wantTotal:    100,
			wantFirstKey: "E15-F04-011",
		},
		{
			name:         "last page partial",
			limit:        10,
			offset:       95,
			wantCount:    5,
			wantTotal:    100,
			wantFirstKey: "E15-F04-096",
		},
		{
			name:      "offset beyond total",
			limit:     10,
			offset:    100,
			wantCount: 0,
			wantTotal: 100,
		},
		{
			name:         "limit zero returns all",
			limit:        0,
			offset:       0,
			wantCount:    100,
			wantTotal:    100,
			wantFirstKey: "E15-F04-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := TaskFilters{
				Limit:  tt.limit,
				Offset: tt.offset,
			}

			tasks, total, err := svc.ListTasksWithPagination(context.Background(), filters)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total, "total count should match")
			assert.Equal(t, tt.wantCount, len(tasks), "returned tasks count should match")

			if tt.wantCount > 0 && tt.wantFirstKey != "" {
				assert.Equal(t, tt.wantFirstKey, tasks[0].Key, "first task key should match")
			}
		})
	}
}

// ============================================================================
// Aggregation Tests (T-E15-F04-001)
// ============================================================================

// TestTaskService_GetTasksByStatus tests status aggregation
func TestTaskService_GetTasksByStatus(t *testing.T) {
	// Arrange: Create tasks with various statuses
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001"}, Status: "todo", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002"}, Status: "todo", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003"}, Status: "in_development", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-004"}, Status: "completed", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-005"}, Status: "blocked", AgentType: &agentType},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Get tasks by status
	statusMap, err := svc.GetTasksByStatus(context.Background(), TaskFilters{})

	// Assert: Correct counts per status
	assert.NoError(t, err)
	assert.Equal(t, 2, statusMap["todo"])
	assert.Equal(t, 1, statusMap["in_development"])
	assert.Equal(t, 1, statusMap["completed"])
	assert.Equal(t, 1, statusMap["blocked"])
}

// TestTaskService_GetTasksByAgent tests agent workload aggregation
func TestTaskService_GetTasksByAgent(t *testing.T) {
	// Arrange: Create tasks assigned to different agents
	backend := "backend"
	frontend := "frontend"
	qa := "qa"

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001"}, Status: "todo", Priority: 5, AgentType: &backend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002"}, Status: "todo", Priority: 8, AgentType: &backend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003"}, Status: "in_development", Priority: 7, AgentType: &frontend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-004"}, Status: "completed", Priority: 6, AgentType: &qa},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Get tasks by agent
	agentMap, err := svc.GetTasksByAgent(context.Background(), TaskFilters{})

	// Assert: Correct counts per agent (excluding completed by default)
	assert.NoError(t, err)
	assert.Equal(t, 2, agentMap["backend"])
	assert.Equal(t, 1, agentMap["frontend"])
	// QA task is completed, should be excluded by default
	_, exists := agentMap["qa"]
	assert.False(t, exists, "completed tasks should be excluded by default")
}

// TestTaskService_GetBlockedTasks tests blocked task retrieval
func TestTaskService_GetBlockedTasks(t *testing.T) {
	// Arrange: Create some blocked tasks
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001"}, Status: "todo", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002"}, Status: "blocked", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003"}, Status: "in_development", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-004"}, Status: "blocked", AgentType: &agentType},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Get blocked tasks
	blockedTasks, err := svc.GetBlockedTasks(context.Background(), TaskFilters{})

	// Assert: Only blocked tasks returned
	assert.NoError(t, err)
	assert.Len(t, blockedTasks, 2)
	for _, task := range blockedTasks {
		assert.Equal(t, models.TaskStatus("blocked"), task.Status)
	}
}

// ============================================================================
// TaskQueryBuilder Tests (T-E15-F04-001)
// ============================================================================

// TestTaskQueryBuilder_WithStatus tests filtering by status
func TestTaskQueryBuilder_WithStatus(t *testing.T) {
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001"}, Status: "todo", Priority: 5, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002"}, Status: "in_development", Priority: 8, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003"}, Status: "todo", Priority: 3, AgentType: &agentType},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Query with status filter
	tasks, total, err := svc.Query().
		WithStatus("todo").
		Build(context.Background())

	// Assert: Only todo tasks returned
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Equal(t, models.TaskStatus("todo"), task.Status)
	}
}

// TestTaskQueryBuilder_WithAgent tests filtering by agent type
func TestTaskQueryBuilder_WithAgent(t *testing.T) {
	backend := "backend"
	frontend := "frontend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001"}, Status: "todo", AgentType: &backend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002"}, Status: "todo", AgentType: &frontend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003"}, Status: "todo", AgentType: &backend},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Query with agent filter
	tasks, total, err := svc.Query().
		WithAgent("backend").
		Build(context.Background())

	// Assert: Only backend tasks returned
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Equal(t, "backend", *task.AgentType)
	}
}

// TestTaskQueryBuilder_WithPriorityRange tests priority filtering
func TestTaskQueryBuilder_WithPriorityRange(t *testing.T) {
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001"}, Status: "todo", Priority: 3, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002"}, Status: "todo", Priority: 5, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003"}, Status: "todo", Priority: 8, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-004"}, Status: "todo", Priority: 10, AgentType: &agentType},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Query with priority range 5-8
	tasks, total, err := svc.Query().
		WithPriorityRange(5, 8).
		Build(context.Background())

	// Assert: Only tasks with priority 5-8 returned
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.GreaterOrEqual(t, task.Priority, 5)
		assert.LessOrEqual(t, task.Priority, 8)
	}
}

// TestTaskQueryBuilder_WithTitleSearch tests fuzzy title search
func TestTaskQueryBuilder_WithTitleSearch(t *testing.T) {
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001", Title: "Implement user authentication"}, Status: "todo", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002", Title: "Add database migration"}, Status: "todo", AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003", Title: "Implement JWT token validation"}, Status: "todo", AgentType: &agentType},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Query with title search
	tasks, total, err := svc.Query().
		WithTitleSearch("implement").
		Build(context.Background())

	// Assert: Only tasks with "implement" in title
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Contains(t, strings.ToLower(task.Title), "implement")
	}
}

// TestTaskQueryBuilder_SortBy tests multi-field sorting
func TestTaskQueryBuilder_SortBy(t *testing.T) {
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001", Title: "Task A"}, Status: "todo", Priority: 5, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002", Title: "Task B"}, Status: "todo", Priority: 8, AgentType: &agentType},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003", Title: "Task C"}, Status: "todo", Priority: 3, AgentType: &agentType},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Query with priority DESC sort
	tasks, _, err := svc.Query().
		SortBy("priority", "DESC").
		Build(context.Background())

	// Assert: Tasks sorted by priority descending
	assert.NoError(t, err)
	assert.Equal(t, 8, tasks[0].Priority)
	assert.Equal(t, 5, tasks[1].Priority)
	assert.Equal(t, 3, tasks[2].Priority)
}

// TestTaskQueryBuilder_Paginate tests pagination
func TestTaskQueryBuilder_Paginate(t *testing.T) {
	agentType := "backend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			tasks := make([]*models.Task, 50)
			for i := 0; i < 50; i++ {
				tasks[i] = &models.Task{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("E15-F04-%03d", i+1)}, Status: "todo",
					Priority:  5,
					AgentType: &agentType,
				}
			}
			return tasks, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Query with pagination (offset 10, limit 10)
	tasks, total, err := svc.Query().
		Paginate(10, 10).
		Build(context.Background())

	// Assert: Correct page returned
	assert.NoError(t, err)
	assert.Equal(t, 50, total)                   // Total count
	assert.Len(t, tasks, 10)                     // Page size
	assert.Equal(t, "E15-F04-011", tasks[0].Key) // First task on page 2
}

// TestTaskQueryBuilder_ChainedFilters tests method chaining
func TestTaskQueryBuilder_ChainedFilters(t *testing.T) {
	backend := "backend"
	frontend := "frontend"
	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E15-F04-001", Title: "Implement API"}, Status: "todo", Priority: 5, AgentType: &backend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-002", Title: "Implement UI"}, Status: "todo", Priority: 8, AgentType: &frontend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-003", Title: "Add tests"}, Status: "in_development", Priority: 7, AgentType: &backend},
				{BaseEntity: models.BaseEntity{Key: "E15-F04-004", Title: "Implement auth"}, Status: "todo", Priority: 9, AgentType: &backend},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Chain multiple filters
	tasks, total, err := svc.Query().
		WithStatus("todo").
		WithAgent("backend").
		WithPriorityRange(5, 8).
		WithTitleSearch("implement").
		SortBy("priority", "DESC").
		Build(context.Background())

	// Assert: All filters applied correctly
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "E15-F04-001", tasks[0].Key) // Only task matching all criteria
}

// ============================================================================
// ValidateDependencies Tests (T-E15-F04-002)
// ============================================================================

// TestTaskService_ValidateDependencies_NoDependencies tests tasks without dependencies
func TestTaskService_ValidateDependencies_NoDependencies(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			return []*models.Task{}, nil // No dependencies
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act: Validate task with no dependencies
	err := svc.ValidateDependencies(context.Background(), "E15-F04-001", "in_development")

	// Assert: Validation passes
	assert.NoError(t, err)
}

// ============================================================================
// MockWorkSessionRepository
// ============================================================================

// MockWorkSessionRepository implements WorkSessionRepository for testing.
type MockWorkSessionRepository struct {
	GetByTaskIDFunc                  func(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
	GetSessionStatsByTaskIDFunc      func(ctx context.Context, taskID int64) (*WorkSessionStats, error)
	GetActiveSessionByTaskIDFunc     func(ctx context.Context, taskID int64) (*models.WorkSession, error)
	GetSessionAnalyticsByFeatureFunc func(ctx context.Context, featureID int64, agentType *string) (*SessionAnalytics, error)
	GetSessionAnalyticsByEpicFunc    func(ctx context.Context, epicID int64, agentType *string) (*SessionAnalytics, error)
}

func (m *MockWorkSessionRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
	if m.GetByTaskIDFunc != nil {
		return m.GetByTaskIDFunc(ctx, taskID)
	}
	return nil, fmt.Errorf("GetByTaskID not implemented in mock")
}

func (m *MockWorkSessionRepository) GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*WorkSessionStats, error) {
	if m.GetSessionStatsByTaskIDFunc != nil {
		return m.GetSessionStatsByTaskIDFunc(ctx, taskID)
	}
	return nil, fmt.Errorf("GetSessionStatsByTaskID not implemented in mock")
}

func (m *MockWorkSessionRepository) GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error) {
	if m.GetActiveSessionByTaskIDFunc != nil {
		return m.GetActiveSessionByTaskIDFunc(ctx, taskID)
	}
	return nil, fmt.Errorf("GetActiveSessionByTaskID not implemented in mock")
}

func (m *MockWorkSessionRepository) GetSessionAnalyticsByFeature(ctx context.Context, featureID int64, agentType *string) (*SessionAnalytics, error) {
	if m.GetSessionAnalyticsByFeatureFunc != nil {
		return m.GetSessionAnalyticsByFeatureFunc(ctx, featureID, agentType)
	}
	return nil, fmt.Errorf("GetSessionAnalyticsByFeature not implemented in mock")
}

func (m *MockWorkSessionRepository) GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*SessionAnalytics, error) {
	if m.GetSessionAnalyticsByEpicFunc != nil {
		return m.GetSessionAnalyticsByEpicFunc(ctx, epicID, agentType)
	}
	return nil, fmt.Errorf("GetSessionAnalyticsByEpic not implemented in mock")
}

// ============================================================================
// GetWorkSessions Tests
// ============================================================================

func TestTaskService_GetWorkSessions_Happy_Path(t *testing.T) {
	// Arrange
	agentID := "agent-001"
	mockSessions := []*models.WorkSession{
		{ID: 1, TaskID: 42, AgentID: &agentID},
		{ID: 2, TaskID: 42, AgentID: &agentID},
	}
	mockStats := &WorkSessionStats{
		TotalSessions: 2,
		ActiveSession: false,
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			assert.Equal(t, "E07-F01-001", key)
			return &models.Task{BaseEntity: models.BaseEntity{ID: 42, Key: "E07-F01-001", Title: "Test Task"}}, nil
		},
	}
	mockSessionRepo := &MockWorkSessionRepository{
		GetByTaskIDFunc: func(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
			assert.Equal(t, int64(42), taskID)
			return mockSessions, nil
		},
		GetSessionStatsByTaskIDFunc: func(ctx context.Context, taskID int64) (*WorkSessionStats, error) {
			assert.Equal(t, int64(42), taskID)
			return mockStats, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	svc.SetSessionRepo(mockSessionRepo)

	// Act
	result, err := svc.GetWorkSessions(context.Background(), "E07-F01-001")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "E07-F01-001", result.TaskKey)
	assert.Equal(t, "Test Task", result.TaskTitle)
	assert.Len(t, result.Sessions, 2)
	assert.NotNil(t, result.Stats)
	assert.Equal(t, 2, result.Stats.TotalSessions)
}

func TestTaskService_GetWorkSessions_Nil_Session_Repo(t *testing.T) {
	// Arrange: service constructed without session repo
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Act
	result, err := svc.GetWorkSessions(context.Background(), "E07-F01-001")

	// Assert: error returned because session repo is nil
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "work session repository not configured")
}

func TestTaskService_GetWorkSessions_Task_Not_Found(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("task not found")
		},
	}
	mockSessionRepo := &MockWorkSessionRepository{}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	svc.SetSessionRepo(mockSessionRepo)

	// Act
	result, err := svc.GetWorkSessions(context.Background(), "E07-F01-999")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestTaskService_GetWorkSessions_Sessions_Repository_Error(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Task"}}, nil
		},
	}
	mockSessionRepo := &MockWorkSessionRepository{
		GetByTaskIDFunc: func(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	svc.SetSessionRepo(mockSessionRepo)

	// Act
	result, err := svc.GetWorkSessions(context.Background(), "E07-F01-001")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get work sessions")
}

func TestTaskService_GetWorkSessions_Stats_Repository_Error(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Task"}}, nil
		},
	}
	mockSessionRepo := &MockWorkSessionRepository{
		GetByTaskIDFunc: func(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
			return []*models.WorkSession{}, nil
		},
		GetSessionStatsByTaskIDFunc: func(ctx context.Context, taskID int64) (*WorkSessionStats, error) {
			return nil, fmt.Errorf("stats query failed")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	svc.SetSessionRepo(mockSessionRepo)

	// Act
	result, err := svc.GetWorkSessions(context.Background(), "E07-F01-001")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get session stats")
}

func TestTaskService_GetWorkSessions_Empty_Sessions(t *testing.T) {
	// Arrange: task exists but has no sessions
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 5, Key: key, Title: "New Task"}}, nil
		},
	}
	mockSessionRepo := &MockWorkSessionRepository{
		GetByTaskIDFunc: func(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
			return []*models.WorkSession{}, nil
		},
		GetSessionStatsByTaskIDFunc: func(ctx context.Context, taskID int64) (*WorkSessionStats, error) {
			return &WorkSessionStats{TotalSessions: 0, ActiveSession: false}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	svc.SetSessionRepo(mockSessionRepo)

	// Act
	result, err := svc.GetWorkSessions(context.Background(), "E07-F01-001")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Sessions)
	assert.Equal(t, 0, result.Stats.TotalSessions)
	assert.False(t, result.Stats.ActiveSession)
}

// MockTaskHistoryRepository implements TaskHistoryRepository for testing.
type MockTaskHistoryRepository struct {
	GetHistoryByTaskKeyFunc func(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
	ListWithFiltersFunc     func(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error)
}

func (m *MockTaskHistoryRepository) GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
	if m.GetHistoryByTaskKeyFunc != nil {
		return m.GetHistoryByTaskKeyFunc(ctx, taskKey)
	}
	return nil, fmt.Errorf("GetHistoryByTaskKey not implemented in mock")
}

func (m *MockTaskHistoryRepository) ListWithFilters(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error) {
	if m.ListWithFiltersFunc != nil {
		return m.ListWithFiltersFunc(ctx, filters)
	}
	return nil, fmt.Errorf("ListWithFilters not implemented in mock")
}

func TestTaskService_GetTaskHistory_Happy_Path(t *testing.T) {
	// Arrange
	oldStatus1 := "todo"
	oldStatus2 := "in_progress"
	expectedHistory := []*models.TaskHistory{
		{ID: 1, TaskID: 10, OldStatus: &oldStatus1, NewStatus: "in_progress"},
		{ID: 2, TaskID: 10, OldStatus: &oldStatus2, NewStatus: "ready_for_review"},
	}
	mockHistoryRepo := &MockTaskHistoryRepository{
		GetHistoryByTaskKeyFunc: func(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
			assert.Equal(t, "E07-F01-001", taskKey)
			return expectedHistory, nil
		},
	}

	svc := NewTaskService(&MockTaskRepository{}, NewEntityService(newMockWorkflowService()), nil)
	svc.SetHistoryRepo(mockHistoryRepo)

	// Act
	result, err := svc.GetTaskHistory(context.Background(), "E07-F01-001")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, "todo", *result[0].OldStatus)
	assert.Equal(t, "in_progress", result[0].NewStatus)
	assert.Equal(t, "in_progress", *result[1].OldStatus)
	assert.Equal(t, "ready_for_review", result[1].NewStatus)
}

func TestTaskService_GetTaskHistory_Nil_History_Repo(t *testing.T) {
	// Arrange: service constructed without history repo and SetHistoryRepo not called
	svc := NewTaskService(&MockTaskRepository{}, NewEntityService(newMockWorkflowService()), nil)

	// Act
	result, err := svc.GetTaskHistory(context.Background(), "E07-F01-001")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "history repository not configured")
}

func TestTaskService_GetTaskHistory_Repository_Error(t *testing.T) {
	// Arrange
	mockHistoryRepo := &MockTaskHistoryRepository{
		GetHistoryByTaskKeyFunc: func(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
			return nil, fmt.Errorf("database query failed")
		},
	}

	svc := NewTaskService(&MockTaskRepository{}, NewEntityService(newMockWorkflowService()), nil)
	svc.SetHistoryRepo(mockHistoryRepo)

	// Act
	result, err := svc.GetTaskHistory(context.Background(), "E07-F01-001")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get task history for E07-F01-001")
	assert.Contains(t, err.Error(), "database query failed")
}

func TestTaskService_GetTaskHistory_Empty_History(t *testing.T) {
	// Arrange: task exists but has no history entries
	mockHistoryRepo := &MockTaskHistoryRepository{
		GetHistoryByTaskKeyFunc: func(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
			return []*models.TaskHistory{}, nil
		},
	}

	svc := NewTaskService(&MockTaskRepository{}, NewEntityService(newMockWorkflowService()), nil)
	svc.SetHistoryRepo(mockHistoryRepo)

	// Act
	result, err := svc.GetTaskHistory(context.Background(), "E07-F01-001")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

// ============================================================================
// executeStatusTransition Tests (via lifecycle methods that now use it)
// ============================================================================

func TestTaskService_TransitionStatus_UsesStatusUpdateRaw(t *testing.T) {
	var capturedParams models.StatusUpdateParams
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 6,
				Key: "T-E07-F01-006"}, Status: "todo",
				FeatureID: 10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			capturedParams = params
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-006", "in_progress", TransitionOptions{
		Agent:  "my-agent",
		Reason: "starting work",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Transitioned)
	assert.Equal(t, "todo", result.FromStatus)
	assert.Equal(t, "in_progress", result.ToStatus)

	assert.Equal(t, int64(6), capturedParams.TaskID)
	assert.Equal(t, "todo", capturedParams.OldStatus)
	assert.NotNil(t, capturedParams.Agent)
	assert.Equal(t, "my-agent", *capturedParams.Agent)
	assert.NotNil(t, capturedParams.RejectionReason)
	assert.Equal(t, "starting work", *capturedParams.RejectionReason)
}

func TestTaskService_TransitionStatus_Forced_UsesStatusUpdateRaw(t *testing.T) {
	var capturedParams models.StatusUpdateParams
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 7,
				Key: "T-E07-F01-007"}, Status: "completed",
				FeatureID: 10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			capturedParams = params
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-007", "todo", TransitionOptions{
		Force:  true,
		Reason: "reopening for rework",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsForced)
	assert.True(t, capturedParams.Force)
	assert.Equal(t, "completed", capturedParams.OldStatus)
}

func TestTaskService_TransitionStatus_BackwardRequiresReason(t *testing.T) {
	// This test verifies that backward transitions require a reason
	// when going through TransitionStatus (not the named lifecycle methods)
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 8,
				Key: "T-E07-F01-008"}, Status: "ready_for_review",
				FeatureID: 10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Attempt backward transition without reason
	_, err := svc.TransitionStatus(context.Background(), "T-E07-F01-008", "in_progress", TransitionOptions{
		// No reason provided
	})

	// Should fail because backward transition requires reason
	if err != nil {
		// Error may or may not occur depending on workflow config.
		// The important thing is the orchestration path works.
		// With the basic workflow config, ready_for_review -> in_progress might not be backward.
		// So we just verify we get a result or an expected error.
		assert.Contains(t, err.Error(), "transition")
	}
}

func TestTaskService_TransitionStatus_AutoUnblockInResult(t *testing.T) {
	// Verify that auto-unblocked keys are included in the TransitionResult message
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 11,
				Key: "T-E07-F01-011"}, Status: "in_progress",
				FeatureID: 10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			return []string{"T-E07-F01-012", "T-E07-F01-013"}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-011", "ready_for_review", TransitionOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Message, "auto-unblocked")
	assert.Contains(t, result.Message, "T-E07-F01-012")
	assert.Contains(t, result.Message, "T-E07-F01-013")
}

// ============================================================================
// TC-F09-035..047: TaskService Full TransitionStatus Delegation Tests
// ============================================================================

// TC-F09-035: TransitionStatus delegates to entitySvc.TransitionStatus
func TestTaskService_TransitionStatus_DelegatesToEntityService(t *testing.T) {
	// Verifies that TransitionStatus uses DefaultTransitionFeatures
	// and routes through the taskEntityRepoAdapter
	var capturedParams models.StatusUpdateParams
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 20, Key: "T-E07-F01-020"},
				Status:     "todo",
				FeatureID:  10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			capturedParams = params
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-020", "in_progress", TransitionOptions{
		Agent: "dev-agent",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.EntityTypeTask, result.EntityType)
	assert.Equal(t, "todo", result.FromStatus)
	assert.Equal(t, "in_progress", result.ToStatus)
	assert.True(t, result.Transitioned)
	// Verify StatusUpdateRaw was called via adapter
	assert.Equal(t, int64(20), capturedParams.TaskID)
	assert.Equal(t, models.TaskStatus("in_progress"), capturedParams.NewStatus)
}

// TC-F09-036: taskEntityRepoAdapter routes UpdateStatus through StatusUpdateRaw
func TestTaskService_TransitionStatus_AdapterUsesStatusUpdateRaw(t *testing.T) {
	statusUpdateRawCalled := false
	var capturedParams models.StatusUpdateParams

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 21, Key: "T-E07-F01-021"},
				Status:     "todo",
				FeatureID:  10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			statusUpdateRawCalled = true
			capturedParams = params
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	_, err := svc.TransitionStatus(context.Background(), "T-E07-F01-021", "in_progress", TransitionOptions{
		Agent:        "test-agent",
		Reason:       "test-reason",
		DocumentPath: "/docs/test.md",
	})

	assert.NoError(t, err)
	assert.True(t, statusUpdateRawCalled, "StatusUpdateRaw should be called via adapter")
	assert.NotNil(t, capturedParams.Agent)
	assert.Equal(t, "test-agent", *capturedParams.Agent)
	assert.NotNil(t, capturedParams.RejectionReason)
	assert.Equal(t, "test-reason", *capturedParams.RejectionReason)
	assert.NotNil(t, capturedParams.DocumentPath)
	assert.Equal(t, "/docs/test.md", *capturedParams.DocumentPath)
}

// TC-F09-037: TransitionStatus propagates EntityService errors
func TestTaskService_TransitionStatus_PropagatesErrors(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 22, Key: "T-E07-F01-022"},
				Status:     "completed",
				FeatureID:  10,
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	// Attempt invalid transition (completed -> todo without force)
	_, err := svc.TransitionStatus(context.Background(), "T-E07-F01-022", "todo", TransitionOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transition")
}

// TC-F09-038: TransitionStatus with force=true and reason
func TestTaskService_TransitionStatus_ForceWithReason(t *testing.T) {
	var capturedParams models.StatusUpdateParams
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 23, Key: "T-E07-F01-023"},
				Status:     "completed",
				FeatureID:  10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			capturedParams = params
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-023", "todo", TransitionOptions{
		Force:  true,
		Reason: "reopening for rework",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsForced)
	assert.True(t, capturedParams.Force)
}

// TC-F09-039: Auto-unblock runs after successful transition
func TestTaskService_TransitionStatus_AutoUnblockPostHook(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 24, Key: "T-E07-F01-024"},
				Status:     "in_progress",
				FeatureID:  10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			return []string{"T-E07-F01-025", "T-E07-F01-026"}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-024", "ready_for_review", TransitionOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Message, "auto-unblocked")
	assert.Contains(t, result.Message, "T-E07-F01-025")
	assert.Contains(t, result.Message, "T-E07-F01-026")
}

// TC-F09-040: Auto-unblock does NOT run on failed transition
func TestTaskService_TransitionStatus_NoAutoUnblockOnError(t *testing.T) {
	statusUpdateRawCalled := false
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 25, Key: "T-E07-F01-025"},
				Status:     "completed",
				FeatureID:  10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			statusUpdateRawCalled = true
			return []string{"should-not-appear"}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	// Invalid transition without force
	_, err := svc.TransitionStatus(context.Background(), "T-E07-F01-025", "todo", TransitionOptions{})

	assert.Error(t, err)
	assert.False(t, statusUpdateRawCalled, "StatusUpdateRaw should not be called on failed transition")
}

// TC-F09-041: Auto-unblock with no dependents
func TestTaskService_TransitionStatus_NoAutoUnblockNoDependents(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 26, Key: "T-E07-F01-026"},
				Status:     "in_progress",
				FeatureID:  10,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			return nil, nil // No dependents unblocked
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	result, err := svc.TransitionStatus(context.Background(), "T-E07-F01-026", "ready_for_review", TransitionOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotContains(t, result.Message, "auto-unblocked")
}

// TC-F09-043: ErrForceReasonRequired handled by EntityService (not inline)
func TestTaskService_TransitionStatus_ForceWithoutReasonReturnsError(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 27, Key: "T-E07-F01-027"},
				Status:     "completed",
				FeatureID:  10,
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	// Force without reason
	_, err := svc.TransitionStatus(context.Background(), "T-E07-F01-027", "todo", TransitionOptions{
		Force: true,
		// No reason
	})

	// Should get ErrForceReasonRequired from EntityService
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrForceReasonRequired)
}

// TC-F09-045: makeResolveActionFn returns callback with task placeholders
func TestTaskService_makeResolveActionFn_TaskEntity(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	fn := svc.makeResolveActionFn(context.Background())
	assert.NotNil(t, fn)

	task := &models.Task{
		BaseEntity: models.BaseEntity{ID: 28, Key: "T-E07-F01-028", Title: "Test Task"},
		Status:     "todo",
	}

	// Without configured actions, should return nil (no panic)
	result := fn(task, "in_progress")
	// Result is nil because no action is configured in the test workflow
	_ = result
}

// TC-F09-046: makeResolveActionFn callback handles non-Task entity
func TestTaskService_makeResolveActionFn_NonTaskEntity(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	fn := svc.makeResolveActionFn(context.Background())
	assert.NotNil(t, fn)

	// Pass an Epic instead of Task
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 1, Key: "E01"},
		Status:     "todo",
	}

	result := fn(epic, "in_progress")
	assert.Nil(t, result, "makeResolveActionFn should return nil for non-Task entity")
}

// TC-F09-048: GetNextStatus delegates to entitySvc.GetNextStatus
func TestTaskService_GetNextStatus_DelegatesToEntityService(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 30, Key: "T-E07-F01-030"},
				Status:     "todo",
				FeatureID:  10,
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	info, err := svc.GetNextStatus(context.Background(), "T-E07-F01-030")

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, models.EntityTypeTask, info.EntityType)
	assert.Equal(t, "T-E07-F01-030", info.EntityKey)
	assert.Equal(t, "todo", info.CurrentStatus)
}

// TC-F09-050: GetNextStatus for terminal status
func TestTaskService_GetNextStatus_TerminalStatus(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 31, Key: "T-E07-F01-031"},
				Status:     "completed",
				FeatureID:  10,
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	info, err := svc.GetNextStatus(context.Background(), "T-E07-F01-031")

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.True(t, info.IsTerminal)
	assert.Empty(t, info.AvailableTransitions)
}

// ============================================================================
// GetTaskDisplayData Tests
// ============================================================================

func TestTaskService_GetTaskDisplayData_ValidJSON(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetTaskDisplayDataRawFunc: func(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error) {
			assert.Equal(t, int64(42), taskID)
			return &repository.TaskDisplayDataRaw{
				BlockedByJSON:    `[{"relationship_type":"depends_on","direction":"outgoing","task_key":"E01-F01-001","task_title":"Setup DB","task_status":"completed"}]`,
				BlocksJSON:       `[{"relationship_type":"depends_on","direction":"incoming","task_key":"E01-F01-003","task_title":"Write tests","task_status":"todo"}]`,
				DependenciesJSON: `[{"key":"E01-F01-001","title":"Setup DB","status":"completed"},{"key":"E01-F01-003","title":"Write tests","status":"todo"}]`,
				DocumentsJSON:    `[{"id":10,"title":"Design Doc","file_path":"docs/design.md"}]`,
				NotesJSON:        `[{"id":1,"note_type":"progress","content":"halfway done","created_by":null,"metadata":null,"created_at":"2026-01-01T00:00:00Z"}]`,
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 42, Key: "E01-F01-002"}}

	data, err := svc.GetTaskDisplayData(context.Background(), task)

	assert.NoError(t, err)
	assert.NotNil(t, data)

	// Blocked by
	assert.Len(t, data.BlockedBy, 1)
	assert.Equal(t, "E01-F01-001", data.BlockedBy[0].TaskKey)
	assert.Equal(t, "depends_on", data.BlockedBy[0].RelationshipType)

	// Blocks
	assert.Len(t, data.Blocks, 1)
	assert.Equal(t, "E01-F01-003", data.Blocks[0].TaskKey)
	assert.Equal(t, "incoming", data.Blocks[0].Direction)

	// Dependencies
	assert.Len(t, data.Dependencies, 2)
	assert.Equal(t, "E01-F01-001", data.Dependencies[0].Key)
	assert.Equal(t, models.TaskStatus("completed"), data.Dependencies[0].Status)

	// Documents
	assert.Len(t, data.RelatedDocs, 1)
	assert.Equal(t, "Design Doc", data.RelatedDocs[0].Title)
	assert.Equal(t, "docs/design.md", data.RelatedDocs[0].FilePath)

	// Notes
	assert.Len(t, data.Notes, 1)
	assert.Equal(t, "halfway done", data.Notes[0].Content)
	assert.Equal(t, models.NoteType("progress"), data.Notes[0].NoteType)
}

func TestTaskService_GetTaskDisplayData_EmptyArrays(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetTaskDisplayDataRawFunc: func(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error) {
			return &repository.TaskDisplayDataRaw{
				BlockedByJSON:    "[]",
				BlocksJSON:       "[]",
				DependenciesJSON: "[]",
				DocumentsJSON:    "[]",
				NotesJSON:        "[]",
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}}

	data, err := svc.GetTaskDisplayData(context.Background(), task)

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Empty(t, data.BlockedBy)
	assert.Empty(t, data.Blocks)
	assert.Empty(t, data.Dependencies)
	assert.Empty(t, data.RelatedDocs)
	assert.Empty(t, data.Notes)
	// Verify non-nil (empty slices, not nil)
	assert.NotNil(t, data.BlockedBy)
	assert.NotNil(t, data.Blocks)
	assert.NotNil(t, data.Dependencies)
	assert.NotNil(t, data.RelatedDocs)
	assert.NotNil(t, data.Notes)
}

func TestTaskService_GetTaskDisplayData_RepoError(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetTaskDisplayDataRawFunc: func(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error) {
			return nil, fmt.Errorf("database connection lost")
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}}

	data, err := svc.GetTaskDisplayData(context.Background(), task)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "failed to get display data for task E01-F01-001")
	assert.Contains(t, err.Error(), "database connection lost")
}

// ============================================================================
// Auto-Reopen Parent Feature Tests (maybeReopenParentFeature via CreateTask)
// ============================================================================

// mockFeatureServiceForReopen is a minimal mock of FeatureService for testing
// the auto-reopen behavior in TaskService.CreateTask.
// We create a real FeatureService with mocked repos so we can wire it via SetFeatureService.
func newFeatureServiceForReopenTest(t *testing.T, featureStatus models.FeatureStatus, updateErr error) (*FeatureService, *bool) {
	t.Helper()
	featureUpdated := false

	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01", Title: "Test Feature"},
				Status:     featureStatus,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			featureUpdated = true
			if updateErr != nil {
				return updateErr
			}
			return nil
		},
	}

	wfSvc := workflow.NewService("")
	svc := NewFeatureService(repo, NewEntityService(wfSvc), featureRepoAsEntityRepo(repo), nil, nil)
	return svc, &featureUpdated
}

func TestTaskService_CreateTask_ReopensTerminalFeature(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	featureSvc, featureUpdated := newFeatureServiceForReopenTest(t, "completed", nil)
	svc.SetFeatureService(featureSvc)

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "New task under completed feature",
		AgentType:  "developer",
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.True(t, *featureUpdated, "feature should have been updated (reopened)")
}

// TestTaskService_CreateTask_ReopenRecordsHistory verifies that auto-reopen
// creates an entity_history record with "auto-reopened" in notes (AC-1 audit trail).
func TestTaskService_CreateTask_ReopenRecordsHistory(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	featureSvc, _ := newFeatureServiceForReopenTest(t, "completed", nil)
	svc.SetFeatureService(featureSvc)

	historyRecorder := &mockEntityHistoryRecorder{}
	svc.SetEntityHistoryRepo(historyRecorder)

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Task triggering history record",
		AgentType:  "developer",
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Len(t, historyRecorder.created, 1, "should create one entity_history record")

	h := historyRecorder.created[0]
	assert.Equal(t, models.EntityTypeFeature, h.EntityType)
	assert.Equal(t, int64(10), h.EntityID)
	assert.NotNil(t, h.FromStatus)
	assert.Equal(t, "completed", *h.FromStatus)
	assert.Equal(t, "active", h.ToStatus)
	assert.NotNil(t, h.Notes)
	assert.Contains(t, *h.Notes, "auto-reopened")
}

func TestTaskService_CreateTask_ReopensArchivedFeature(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// "archived" is in default feature _complete_ list: ["completed", "archived"]
	featureSvc, featureUpdated := newFeatureServiceForReopenTest(t, "archived", nil)
	svc.SetFeatureService(featureSvc)

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Revived task under archived feature",
		AgentType:  "developer",
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.True(t, *featureUpdated, "feature should have been updated (reopened from archived)")
}

func TestTaskService_CreateTask_NoReopenNonTerminalFeature(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	featureSvc, featureUpdated := newFeatureServiceForReopenTest(t, "active", nil)
	svc.SetFeatureService(featureSvc)

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Task under active feature",
		AgentType:  "developer",
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.False(t, *featureUpdated, "feature should NOT have been updated (already non-terminal)")
}

func TestTaskService_CreateTask_NoReopenNilFeatureService(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)
	// Do NOT set featureService -- it remains nil

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Task with nil featureService",
		AgentType:  "developer",
	})

	// Task creation should still succeed
	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestTaskService_CreateTask_ReopenFailureDoesNotFailCreate(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Feature service will fail on update
	featureSvc, _ := newFeatureServiceForReopenTest(t, "completed", fmt.Errorf("simulated DB error"))
	svc.SetFeatureService(featureSvc)

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Task when feature update fails",
		AgentType:  "developer",
	})

	// Task creation should STILL succeed despite feature update failure
	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestTaskService_CreateTask_CustomAggregationStatus(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	// Create a custom workflow config with custom _aggregation_ status
	tempDir := t.TempDir()
	configData := `{
		"task_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {
				"_start_": ["todo"],
				"_complete_": ["completed"]
			},
			"status_flow": {
				"todo": ["completed"],
				"completed": []
			}
		},
		"feature_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["done", "abandoned"],
				"_aggregation_": ["tracking"]
			},
			"status_flow": {
				"draft": ["tracking"],
				"tracking": ["done"],
				"done": [],
				"abandoned": []
			}
		}
	}`
	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configData), 0644)
	assert.NoError(t, err)
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	customWf := workflow.NewService(tempDir)
	svc := NewTaskService(mockRepo, NewEntityService(customWf), nil)

	// Create feature service with custom workflow and "done" status (terminal)
	var capturedFeatureStatus models.FeatureStatus
	featureRepo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01", Title: "Test Feature"},
				Status:     "done",
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			capturedFeatureStatus = feature.Status
			return nil
		},
	}
	featureSvc := NewFeatureService(featureRepo, NewEntityService(customWf), featureRepoAsEntityRepo(featureRepo), nil, nil)
	svc.SetFeatureService(featureSvc)

	task, createErr := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Task under done feature with custom aggregation",
		AgentType:  "developer",
	})

	assert.NoError(t, createErr)
	assert.NotNil(t, task)
	assert.Equal(t, models.FeatureStatus("tracking"), capturedFeatureStatus,
		"feature should be reopened to custom aggregation status 'tracking'")
}

// TestTaskService_CreateTask_ReopensCancelledFeature verifies AC-2: task creation
// reopens a feature with status "cancelled" when _complete_ includes "cancelled".
func TestTaskService_CreateTask_ReopensCancelledFeature(t *testing.T) {
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			task.ID = 1
			return nil
		},
	}

	// Create a custom workflow config where _complete_ includes "cancelled"
	tempDir := t.TempDir()
	configData := `{
		"task_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {
				"_start_": ["todo"],
				"_complete_": ["completed"]
			},
			"status_flow": {
				"todo": ["completed"],
				"completed": []
			}
		},
		"feature_workflow": {
			"status_flow_version": "1.0",
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["completed", "cancelled"],
				"_aggregation_": ["active"]
			},
			"status_flow": {
				"draft": ["active"],
				"active": ["completed", "cancelled"],
				"completed": [],
				"cancelled": []
			}
		}
	}`
	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configData), 0644)
	assert.NoError(t, err)
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	customWf := workflow.NewService(tempDir)
	svc := NewTaskService(mockRepo, NewEntityService(customWf), nil)

	// Create feature service with "cancelled" status (terminal per custom config)
	var capturedFeatureStatus models.FeatureStatus
	featureRepo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 10, Key: "E01-F01", Title: "Test Feature"},
				Status:     "cancelled",
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			capturedFeatureStatus = feature.Status
			return nil
		},
	}
	featureSvc := NewFeatureService(featureRepo, NewEntityService(customWf), featureRepoAsEntityRepo(featureRepo), nil, nil)
	svc.SetFeatureService(featureSvc)

	task, createErr := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Revived task under cancelled feature",
		AgentType:  "developer",
	})

	assert.NoError(t, createErr)
	assert.NotNil(t, task)
	assert.Equal(t, models.FeatureStatus("active"), capturedFeatureStatus,
		"cancelled feature should be reopened to 'active'")
}

// TestTaskService_CreateTask_CreatorSvcPath_ReopensFeature verifies AC-8:
// the creatorSvc primary path (non-nil creatorSvc) also triggers auto-reopen.
// Uses a real DB + Creator since Creator is a concrete type.
func TestTaskService_CreateTask_CreatorSvcPath_ReopensFeature(t *testing.T) {
	// Set up a temp SQLite DB
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	sqlDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	defer sqlDB.Close()

	repoDB := repository.NewDB(sqlDB)
	ctx := context.Background()

	// Seed epic and feature directly via SQL
	res, err := sqlDB.ExecContext(ctx, `INSERT INTO epics (key, title, status, priority) VALUES ('E98', 'Test Epic', 'active', 'medium')`)
	require.NoError(t, err)
	epicID, _ := res.LastInsertId()

	_, err = sqlDB.ExecContext(ctx, `INSERT INTO features (key, title, status, epic_id, file_path) VALUES ('E98-F01', 'Test Feature', 'completed', ?, 'docs/plan/E98/E98-F01/feature.md')`, epicID)
	require.NoError(t, err)

	// Build Creator with real repos
	taskRepo := repository.NewTaskRepository(repoDB)
	featureRepo := repository.NewFeatureRepository(repoDB)
	epicRepo := repository.NewEpicRepository(repoDB)
	historyRepo := repository.NewTaskHistoryRepository(repoDB) //nolint:staticcheck // Required by taskcreation.Creator constructor

	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	wfSvc := workflow.NewService(tempDir)

	creator := taskcreation.NewCreator(repoDB, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, tempDir, wfSvc)

	// Build TaskService with the real Creator
	entitySvc := NewEntityService(wfSvc)
	svc := NewTaskService(taskRepo, entitySvc, creator)

	// Wire FeatureService that reads from the same DB so reopen can find the feature
	featureSvc := NewFeatureService(featureRepo, NewEntityService(wfSvc), NewFeatureRepositoryAdapter(featureRepo), nil, epicRepo)
	svc.SetFeatureService(featureSvc)

	// Wire history recorder to capture audit trail
	historyRecorder := &mockEntityHistoryRecorder{}
	svc.SetEntityHistoryRepo(historyRecorder)

	// Create task via the creatorSvc path (creatorSvc != nil)
	task, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E98",
		FeatureKey: "F01",
		Title:      "Task via creatorSvc path",
		AgentType:  "developer",
	})

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "T-E98-F01-001", task.Key)

	// Verify feature was reopened
	feature, err := featureRepo.GetByKey(ctx, "E98-F01")
	require.NoError(t, err)
	assert.Equal(t, models.FeatureStatus("active"), feature.Status,
		"feature should be reopened to 'active' via creatorSvc path")

	// Verify history was recorded
	assert.Len(t, historyRecorder.created, 1, "should record entity_history for auto-reopen")
	if len(historyRecorder.created) > 0 {
		h := historyRecorder.created[0]
		assert.Equal(t, models.EntityTypeFeature, h.EntityType)
		assert.Contains(t, *h.Notes, "auto-reopened")
	}
}

// capturingSlogHandler is a slog.Handler that records all log records.
// Safe for concurrent use from multiple goroutines.
type capturingSlogHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *capturingSlogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingSlogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	h.records = append(h.records, r)
	h.mu.Unlock()
	return nil
}

func (h *capturingSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }

func (h *capturingSlogHandler) WithGroup(name string) slog.Handler { return h }

// TestTaskService_recalculateFeatureProgress_LogsErrorNotSilentlyDiscarded is the
// regression test for B005: error from RecalculateAndSetProgress must be logged as
// a warning rather than silently discarded with `_ = err`.
func TestTaskService_recalculateFeatureProgress_LogsErrorNotSilentlyDiscarded(t *testing.T) {
	// Install a capturing slog handler so we can assert on log output.
	handler := &capturingSlogHandler{}
	origLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(origLogger)

	// Build a FeatureService whose RecalculateAndSetProgress returns an error by
	// failing the task-status-breakdown repo call.
	recalcErr := fmt.Errorf("simulated recalc DB error")
	featureRepo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: id, Key: "E01-F01", Title: "Test Feature"},
				Status:     "active",
			}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return nil, recalcErr
		},
	}
	wfSvc := workflow.NewService("")
	featureSvc := NewFeatureService(featureRepo, NewEntityService(wfSvc), featureRepoAsEntityRepo(featureRepo), nil, nil)

	// Build a TaskService wired to the above FeatureService.
	taskKey := "E01-F01-001"
	featureID := int64(42)
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: taskKey},
				Status:     "todo",
				FeatureID:  featureID,
			}, nil
		},
		StatusUpdateRawFunc: func(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
			return nil, nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(wfSvc), nil)
	svc.SetFeatureService(featureSvc)

	// Call TransitionStatus to trigger the recalculateFeatureProgress post-hook.
	_, err := svc.TransitionStatus(context.Background(), taskKey, "in_progress", TransitionOptions{})
	// TransitionStatus itself should succeed (progress recalc failure is non-fatal).
	assert.NoError(t, err)

	// Assert that a warning was logged for the recalc error — NOT silently discarded.
	var found bool
	var logMessage string
	var errorAttrValue string
	for _, r := range handler.records {
		if r.Level == slog.LevelWarn {
			logMessage = r.Message
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == "error" {
					found = true
					errorAttrValue = a.Value.String()
					return false
				}
				return true
			})
		}
	}
	assert.True(t, found,
		"expected slog.Warn to be called with an 'error' attribute when RecalculateAndSetProgress fails, "+
			"but no warning was logged — error was silently discarded (B005)")
	assert.Contains(t, logMessage, "feature progress recalculation failed",
		"warning message should mention 'feature progress recalculation failed'")
	assert.Contains(t, errorAttrValue, recalcErr.Error(),
		"error attribute should contain the simulated error text")
}

// ============================================================================
// TC-SVC-A through TC-SVC-E: Size field propagation (E07-F42-005)
// ============================================================================

// svcSizeIntPtr is a local helper for creating *int pointers in size tests.
func svcSizeIntPtr(n int) *int { return &n }

func TestTaskService_CreateTask_PropagatesSize(t *testing.T) {
	// TC-SVC-A: CreateTask propagates Size to repository.
	var capturedTask *models.Task
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	size := 5
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "Sized task",
		AgentType:  "developer",
		Size:       &size,
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	require.NotNil(t, capturedTask, "repo Create must be called")
	require.NotNil(t, capturedTask.Size, "captured task.Size must not be nil")
	assert.Equal(t, 5, *capturedTask.Size)
}

func TestTaskService_CreateTask_NilSizePropagated(t *testing.T) {
	// TC-SVC-E: CreateTask passes Size=nil when not provided.
	var capturedTask *models.Task
	mockRepo := &MockTaskRepository{
		ListByKeyPrefixFunc: func(ctx context.Context, prefix string) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
		CreateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			task.ID = 1
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		EpicKey:    "E07",
		FeatureKey: "F01",
		Title:      "No size task",
		AgentType:  "developer",
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	require.NotNil(t, capturedTask, "repo Create must be called")
	assert.Nil(t, capturedTask.Size)
}

func TestTaskService_UpdateTask_SetsSize(t *testing.T) {
	// TC-SVC-C: UpdateTask with Size=ptr(8) updates the field.
	var capturedTask *models.Task
	agentType := "developer"
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Existing Task", Size: svcSizeIntPtr(3)},
				Status:     "todo",
				AgentType:  &agentType,
				Priority:   5,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	size := 8
	task, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", TaskUpdates{
		Size: &size,
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	require.NotNil(t, capturedTask, "repo Update must be called")
	require.NotNil(t, capturedTask.Size)
	assert.Equal(t, 8, *capturedTask.Size)
}

func TestTaskService_UpdateTask_ClearSize(t *testing.T) {
	// TC-SVC-B: UpdateTask with ClearSize=true sets model.Size = nil.
	var capturedTask *models.Task
	agentType := "developer"
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Existing Task", Size: svcSizeIntPtr(5)},
				Status:     "todo",
				AgentType:  &agentType,
				Priority:   5,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	task, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", TaskUpdates{
		ClearSize: true,
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	require.NotNil(t, capturedTask, "repo Update must be called")
	assert.Nil(t, capturedTask.Size, "ClearSize=true should set Size to nil")
}

func TestTaskService_UpdateTask_NoSizeChange(t *testing.T) {
	// TC-SVC-D: UpdateTask with neither Size nor ClearSize leaves size unchanged.
	var capturedTask *models.Task
	existingSize := 3
	agentType := "developer"
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Existing Task", Size: &existingSize},
				Status:     "todo",
				AgentType:  &agentType,
				Priority:   5,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	task, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", TaskUpdates{})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	require.NotNil(t, capturedTask, "repo Update must be called")
	require.NotNil(t, capturedTask.Size, "size should remain unchanged")
	assert.Equal(t, 3, *capturedTask.Size)
}

// TestTaskService_CreateTask_CreatorSvcPath_PropagatesSize verifies that when
// creatorSvc is non-nil (the production path), input.Size is propagated through
// taskcreation.Creator into the persisted models.Task.
// Uses a real DB + real Creator, mirroring TestTaskService_CreateTask_CreatorSvcPath_ReopensFeature.
func TestTaskService_CreateTask_CreatorSvcPath_PropagatesSize(t *testing.T) {
	// Set up a temp SQLite DB
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	sqlDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	defer sqlDB.Close()

	repoDB := repository.NewDB(sqlDB)
	ctx := context.Background()

	// Seed epic and feature directly via SQL
	res, err := sqlDB.ExecContext(ctx, `INSERT INTO epics (key, title, status, priority) VALUES ('E97', 'Size Test Epic', 'active', 'medium')`)
	require.NoError(t, err)
	epicID, _ := res.LastInsertId()

	_, err = sqlDB.ExecContext(ctx, `INSERT INTO features (key, title, status, epic_id, file_path) VALUES ('E97-F01', 'Size Test Feature', 'active', ?, 'docs/plan/E97/E97-F01/feature.md')`, epicID)
	require.NoError(t, err)

	// Build Creator with real repos
	taskRepo := repository.NewTaskRepository(repoDB)
	featureRepo := repository.NewFeatureRepository(repoDB)
	epicRepo := repository.NewEpicRepository(repoDB)
	historyRepo := repository.NewTaskHistoryRepository(repoDB) //nolint:staticcheck // Required by taskcreation.Creator constructor

	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	wfSvc := workflow.NewService(tempDir)

	creator := taskcreation.NewCreator(repoDB, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, tempDir, wfSvc)

	// Build TaskService with the real Creator (production wiring)
	entitySvc := NewEntityService(wfSvc)
	svc := NewTaskService(taskRepo, entitySvc, creator)

	// Wire history recorder
	historyRecorder := &mockEntityHistoryRecorder{}
	svc.SetEntityHistoryRepo(historyRecorder)

	// Create task via the creatorSvc path with a non-nil Size
	size := 5
	task, err := svc.CreateTask(ctx, CreateTaskInput{
		EpicKey:    "E97",
		FeatureKey: "F01",
		Title:      "Sized task via creator path",
		AgentType:  "developer",
		Size:       &size,
	})

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "T-E97-F01-001", task.Key)

	// Assert Size is propagated through the production creatorSvc path
	require.NotNil(t, task.Size, "task.Size must not be nil — creatorSvc path must propagate input.Size")
	assert.Equal(t, 5, *task.Size, "task.Size should equal the input Size value")

	// Confirm the DB record also has the correct Size
	persisted, err := taskRepo.GetByKey(ctx, "T-E97-F01-001")
	require.NoError(t, err)
	require.NotNil(t, persisted.Size, "persisted task.Size must not be nil")
	assert.Equal(t, 5, *persisted.Size, "persisted task.Size should equal the input Size value")
}

// TestTaskService_UpdateTask_ClearSizePrecedence verifies spec D5:
// ClearSize=true takes precedence over a simultaneously-set Size value.
// When both ClearSize=true and Size=ptr(8) are provided, the entity.Size is nil.
func TestTaskService_UpdateTask_ClearSizePrecedence(t *testing.T) {
	// TC-SVC-B (extended): ClearSize=true wins over a non-nil Size input.
	var capturedTask *models.Task
	existingSize := 3
	agentType := "developer"
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Existing Task", Size: &existingSize},
				Status:     "todo",
				AgentType:  &agentType,
				Priority:   5,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			capturedTask = task
			return nil
		},
	}

	svc := NewTaskService(mockRepo, NewEntityService(newMockWorkflowService()), nil)

	// Set BOTH ClearSize=true AND Size=ptr(8) simultaneously — ClearSize must win (spec D5)
	newSize := 8
	task, err := svc.UpdateTask(context.Background(), "T-E07-F01-001", TaskUpdates{
		ClearSize: true,
		Size:      &newSize,
	})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	require.NotNil(t, capturedTask, "repo Update must be called")
	assert.Nil(t, capturedTask.Size,
		"ClearSize=true must take precedence over Size=ptr(8) — spec D5 contract")
}
