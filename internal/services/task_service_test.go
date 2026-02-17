package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Mock Repositories
// ============================================================================

// MockTaskRepository implements TaskRepository interface for testing
type MockTaskRepository struct {
	CreateFunc             func(ctx context.Context, task *models.Task) error
	GetByKeyFunc           func(ctx context.Context, key string) (*models.Task, error)
	GetByIDFunc            func(ctx context.Context, id int64) (*models.Task, error)
	UpdateFunc             func(ctx context.Context, task *models.Task) error
	DeleteFunc             func(ctx context.Context, id int64) error
	ListFunc               func(ctx context.Context) ([]*models.Task, error)
	ListByFeatureFunc      func(ctx context.Context, featureID int64) ([]*models.Task, error)
	ListByEpicFunc         func(ctx context.Context, epicKey string) ([]*models.Task, error)
	GetTaskDependentsFunc  func(ctx context.Context, taskKey string) ([]*models.Task, error)
	UpdateStatusFunc       func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
	UpdateStatusForcedFunc func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	expectedTask := &models.Task{
		ID:     1,
		Key:    "T-E07-F01-001",
		Title:  "Test Task",
		Status: models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			// Service passes key as-is, without modifying it
			return expectedTask, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	task, err := svc.GetTask(context.Background(), "E07-F01-999")

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to get")
}

// ============================================================================
// UpdateTask Tests
// ============================================================================

func TestTaskService_UpdateTask_Update_Title(t *testing.T) {
	existingTask := &models.Task{
		ID:       1,
		Key:      "T-E07-F01-001",
		Title:    "Old Title",
		Priority: 5,
		Status:   models.TaskStatus("todo"),
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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	existingTask := &models.Task{
		ID:       1,
		Key:      "T-E07-F01-001",
		Title:    "Old Title",
		Priority: 5,
		Status:   models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return existingTask, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	existingTask := &models.Task{
		ID:       1,
		Key:      "T-E07-F01-001",
		Title:    "Old Title",
		Priority: 5,
		Status:   models.TaskStatus("todo"),
	}

	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return existingTask, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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

func TestTaskService_UpdateTask_Not_Found(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
	taskToDelete := &models.Task{
		ID:  1,
		Key: "E07-F01-001",
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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	err := svc.DeleteTask(context.Background(), "E07-F01-001")

	assert.NoError(t, err)
}

func TestTaskService_DeleteTask_Not_Found(t *testing.T) {
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	err := svc.DeleteTask(context.Background(), "E07-F01-999")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete")
}

// ============================================================================
// ListTasks Tests
// ============================================================================

func TestTaskService_ListTasks_No_Filters(t *testing.T) {
	allTasks := []*models.Task{
		{Key: "E07-F01-001", Title: "Task 1", Status: models.TaskStatus("todo"), Priority: 5},
		{Key: "E07-F01-002", Title: "Task 2", Status: models.TaskStatus("in_progress"), Priority: 8},
		{Key: "E07-F01-003", Title: "Task 3", Status: models.TaskStatus("completed"), Priority: 3},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{})

	assert.NoError(t, err)
	assert.NotNil(t, tasks)
	// Completed tasks excluded by default
	assert.Equal(t, 2, len(tasks))
}

func TestTaskService_ListTasks_Show_All(t *testing.T) {
	allTasks := []*models.Task{
		{Key: "E07-F01-001", Title: "Task 1", Status: models.TaskStatus("todo"), Priority: 5},
		{Key: "E07-F01-002", Title: "Task 2", Status: models.TaskStatus("completed"), Priority: 8},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{ShowAll: true})

	assert.NoError(t, err)
	assert.Equal(t, 2, len(tasks))
}

func TestTaskService_ListTasks_Filter_By_Status(t *testing.T) {
	allTasks := []*models.Task{
		{Key: "E07-F01-001", Title: "Task 1", Status: models.TaskStatus("todo"), Priority: 5},
		{Key: "E07-F01-002", Title: "Task 2", Status: models.TaskStatus("in_progress"), Priority: 8},
		{Key: "E07-F01-003", Title: "Task 3", Status: models.TaskStatus("todo"), Priority: 3},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
		{Key: "E07-F01-001", Title: "Task 1", Status: models.TaskStatus("todo"), Priority: 3, ExecutionOrder: &ord2},
		{Key: "E07-F01-002", Title: "Task 2", Status: models.TaskStatus("todo"), Priority: 8, ExecutionOrder: &ord1},
		{Key: "E07-F01-003", Title: "Task 3", Status: models.TaskStatus("todo"), Priority: 5, ExecutionOrder: &ord1},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	tasks, err := svc.ListTasks(context.Background(), TaskFilters{ShowAll: true})

	assert.NoError(t, err)
	// Should be sorted by execution order (1, 1, 2), then priority (8, 5, 3)
	assert.Equal(t, "E07-F01-002", tasks[0].Key) // Order 1, Priority 8
	assert.Equal(t, "E07-F01-003", tasks[1].Key) // Order 1, Priority 5
	assert.Equal(t, "E07-F01-001", tasks[2].Key) // Order 2, Priority 3
}

// ============================================================================
// StartTask Tests (Lifecycle)
// ============================================================================

func TestTaskService_StartTask_Happy_Path_From_Todo(t *testing.T) {
	// Arrange: Task in "todo" status
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			assert.Equal(t, "E07-F01-001", key)
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Title:  "Test Task",
				Status: models.TaskStatus("todo"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, int64(1), taskID)
			assert.Equal(t, models.TaskStatus("in_progress"), newStatus)
			assert.NotNil(t, agent)
			assert.Equal(t, "agent123", *agent)
			assert.Nil(t, notes)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Start task
	task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

	// Assert: Task started successfully
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}

func TestTaskService_StartTask_Happy_Path_From_Blocked(t *testing.T) {
	// Arrange: Task in "blocked" status can be started
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("blocked"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, models.TaskStatus("in_progress"), newStatus)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

	assert.NoError(t, err)
	assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}

func TestTaskService_StartTask_Not_Found(t *testing.T) {
	// Arrange: Task doesn't exist
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("task not found")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to start non-existent task
	task, err := svc.StartTask(context.Background(), "E07-F01-999", "agent123")

	// Assert: Error returned (task not found)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to start task")
}

func TestTaskService_StartTask_Already_In_Progress(t *testing.T) {
	// Arrange: Task already in_progress
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to start task that's already in progress
	task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

	// Assert: Error returned (workflow validation should reject)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "cannot start task")
}

func TestTaskService_StartTask_From_Completed(t *testing.T) {
	// Arrange: Task already completed
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("completed"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to start completed task
	task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

	// Assert: Error returned (can't restart completed task)
	assert.Error(t, err)
	assert.Nil(t, task)
}

// ============================================================================
// CompleteTask Tests (Lifecycle)
// ============================================================================

func TestTaskService_CompleteTask_Happy_Path(t *testing.T) {
	// Arrange: Task in "in_progress" status
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, int64(1), taskID)
			assert.Equal(t, models.TaskStatus("ready_for_review"), newStatus)
			assert.NotNil(t, notes)
			assert.Equal(t, "Implementation completed", *notes)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Complete task
	task, err := svc.CompleteTask(context.Background(), "E07-F01-001", "Implementation completed")

	// Assert: Task marked ready for review
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatus("ready_for_review"), task.Status)
}

func TestTaskService_CompleteTask_Not_In_Progress(t *testing.T) {
	// Arrange: Task in "todo" status (can't complete without starting)
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("todo"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to complete task that's not in progress
	task, err := svc.CompleteTask(context.Background(), "E07-F01-001", "Done")

	// Assert: Error returned
	assert.Error(t, err)
	assert.Nil(t, task)
}

func TestTaskService_CompleteTask_Empty_Notes(t *testing.T) {
	// Arrange: Task in progress but notes are empty
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			// Notes can be empty or nil
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Complete task with empty notes
	task, err := svc.CompleteTask(context.Background(), "E07-F01-001", "")

	// Assert: Success (empty notes allowed)
	assert.NoError(t, err)
	assert.NotNil(t, task)
}

// ============================================================================
// ApproveTask Tests (Lifecycle)
// ============================================================================

func TestTaskService_ApproveTask_Happy_Path(t *testing.T) {
	// Arrange: Task in "ready_for_review" status
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("ready_for_review"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, models.TaskStatus("completed"), newStatus)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Approve task
	task, err := svc.ApproveTask(context.Background(), "E07-F01-001", "Looks good")

	// Assert: Task marked completed
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatus("completed"), task.Status)
}

func TestTaskService_ApproveTask_Not_Ready(t *testing.T) {
	// Arrange: Task still in progress
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to approve task that's not ready
	task, err := svc.ApproveTask(context.Background(), "E07-F01-001", "")

	// Assert: Error returned
	assert.Error(t, err)
	assert.Nil(t, task)
}

// ============================================================================
// ReopenTask Tests (Lifecycle)
// ============================================================================

func TestTaskService_ReopenTask_Happy_Path(t *testing.T) {
	// Arrange: Task in "ready_for_review" status
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("ready_for_review"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, models.TaskStatus("in_progress"), newStatus)
			assert.NotNil(t, notes)
			assert.Contains(t, *notes, "fixes needed")
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Reopen task
	task, err := svc.ReopenTask(context.Background(), "E07-F01-001", "Some fixes needed")

	// Assert: Task back to in_progress
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatus("in_progress"), task.Status)
}

func TestTaskService_ReopenTask_Already_In_Progress(t *testing.T) {
	// Arrange: Task already in progress
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to reopen task that's already in progress
	task, err := svc.ReopenTask(context.Background(), "E07-F01-001", "")

	// Assert: Error returned (already open)
	assert.Error(t, err)
	assert.Nil(t, task)
}

// ============================================================================
// BlockTask Tests (Lifecycle)
// ============================================================================

func TestTaskService_BlockTask_Happy_Path(t *testing.T) {
	// Arrange: Task in any status can be blocked
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, models.TaskStatus("blocked"), newStatus)
			assert.NotNil(t, notes)
			assert.Contains(t, *notes, "API")
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Block task
	task, err := svc.BlockTask(context.Background(), "E07-F01-001", "Waiting on external API")

	// Assert: Task marked blocked
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatus("blocked"), task.Status)
}

func TestTaskService_BlockTask_Empty_Reason(t *testing.T) {
	// Arrange: Attempt to block without reason
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Block task without reason
	task, err := svc.BlockTask(context.Background(), "E07-F01-001", "")

	// Assert: Error returned (reason required)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "empty")
}

// ============================================================================
// UnblockTask Tests (Lifecycle)
// ============================================================================

func TestTaskService_UnblockTask_Happy_Path(t *testing.T) {
	// Arrange: Task in "blocked" status
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("blocked"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			assert.Equal(t, models.TaskStatus("todo"), newStatus)
			return nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Unblock task
	task, suggestions, err := svc.UnblockTask(context.Background(), "E07-F01-001")

	// Assert: Task unblocked (back to todo)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatus("todo"), task.Status)
	// Suggestions can be empty or contain next actions
	_ = suggestions
}

func TestTaskService_UnblockTask_Not_Blocked(t *testing.T) {
	// Arrange: Task not in blocked status
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to unblock task that's not blocked
	task, suggestions, err := svc.UnblockTask(context.Background(), "E07-F01-001")

	// Assert: Error returned (not blocked)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Nil(t, suggestions)
}

// ============================================================================
// ValidateStatus Tests
// ============================================================================

func TestTaskService_ValidateStatus_Valid_Status(t *testing.T) {
	svc := NewTaskService(&MockTaskRepository{}, newMockWorkflowService(), nil, nil)

	// Act: Validate valid status
	err := svc.ValidateStatus("todo")

	// Assert: No error for valid status
	assert.NoError(t, err)
}

func TestTaskService_ValidateStatus_Invalid_Status(t *testing.T) {
	svc := NewTaskService(&MockTaskRepository{}, newMockWorkflowService(), nil, nil)

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
		NewTaskService(nil, newMockWorkflowService(), nil, nil)
	})
}

func TestNewTaskService_Requires_WorkflowService(t *testing.T) {
	// Arrange & Act & Assert: Should panic without workflow service
	mockRepo := &MockTaskRepository{}
	assert.Panics(t, func() {
		NewTaskService(mockRepo, nil, nil, nil)
	})
}

func TestNewTaskService_Optional_Dependencies(t *testing.T) {
	// Arrange & Act: Create service with only required dependencies
	mockRepo := &MockTaskRepository{}
	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Assert: Service created successfully
	assert.NotNil(t, svc)
}

// ============================================================================
// Additional Lifecycle Edge Cases
// ============================================================================

func TestTaskService_StartTask_UpdateStatus_Error(t *testing.T) {
	// Arrange: UpdateStatus fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("todo"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to start task
	task, err := svc.StartTask(context.Background(), "E07-F01-001", "agent123")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_CompleteTask_UpdateStatus_Error(t *testing.T) {
	// Arrange: UpdateStatus fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to complete task
	task, err := svc.CompleteTask(context.Background(), "E07-F01-001", "Done")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_ApproveTask_UpdateStatus_Error(t *testing.T) {
	// Arrange: UpdateStatus fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("ready_for_review"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to approve task
	task, err := svc.ApproveTask(context.Background(), "E07-F01-001", "")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_ReopenTask_UpdateStatus_Error(t *testing.T) {
	// Arrange: UpdateStatus fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("ready_for_review"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to reopen task
	task, err := svc.ReopenTask(context.Background(), "E07-F01-001", "Needs fixes")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_BlockTask_UpdateStatus_Error(t *testing.T) {
	// Arrange: UpdateStatus fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("in_progress"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to block task
	task, err := svc.BlockTask(context.Background(), "E07-F01-001", "Waiting on API")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_UnblockTask_UpdateStatus_Error(t *testing.T) {
	// Arrange: UpdateStatus fails
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Status: models.TaskStatus("blocked"),
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to unblock task
	task, suggestions, err := svc.UnblockTask(context.Background(), "E07-F01-001")

	// Assert: Database error returned
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Nil(t, suggestions)
	assert.Contains(t, err.Error(), "database error")
}

// ============================================================================
// Update and Delete Edge Cases
// ============================================================================

func TestTaskService_UpdateTask_Invalid_Priority(t *testing.T) {
	// Arrange: Update with invalid priority
	mockRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				ID:       1,
				Key:      "T-E07-F01-001",
				Title:    "Test Task",
				Priority: 5,
				Status:   models.TaskStatus("todo"),
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
			return &models.Task{
				ID:       1,
				Key:      "T-E07-F01-001",
				Title:    "Old Title",
				Priority: 5,
				Status:   models.TaskStatus("todo"),
			}, nil
		},
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
			return &models.Task{
				ID:  1,
				Key: "T-E07-F01-001",
			}, nil
		},
		GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			// Return dependent tasks
			return []*models.Task{
				{Key: "T-E07-F01-002", Status: models.TaskStatus("todo")},
			}, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
			return &models.Task{
				ID:  1,
				Key: "E07-F01-001",
			}, nil
		},
		GetTaskDependentsFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			return []*models.Task{}, nil // No dependents
		},
		DeleteFunc: func(ctx context.Context, id int64) error {
			return fmt.Errorf("database error")
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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
		{Key: "E07-F01-001", Status: models.TaskStatus("todo"), AgentType: &backend},
		{Key: "E07-F01-002", Status: models.TaskStatus("todo"), AgentType: &frontend},
		{Key: "E07-F01-003", Status: models.TaskStatus("todo"), AgentType: &backend},
	}

	mockRepo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) {
			return allTasks, nil
		},
	}

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

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

	svc := NewTaskService(mockRepo, newMockWorkflowService(), nil, nil)

	// Act: Attempt to list tasks
	tasks, err := svc.ListTasks(context.Background(), TaskFilters{})

	// Assert: Error propagated
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "failed to list")
}
