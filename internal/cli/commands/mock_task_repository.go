package commands

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskRepositoryInterface defines the methods needed for task workflow operations
type TaskRepositoryInterface interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *models.AgentType, maxPriority *int) ([]*models.Task, error)
}

// MockTaskRepository is a mock implementation of TaskRepository for testing
type MockTaskRepository struct {
	tasks      map[string]*models.Task
	CreateFunc func(ctx context.Context, task *models.Task) error
}

// NewMockTaskRepository creates a new mock repository with test data
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[string]*models.Task),
	}
}

// AddTask adds a task to the mock repository
func (m *MockTaskRepository) AddTask(task *models.Task) {
	m.tasks[task.Key] = task
}

// GetByKey retrieves a task by its key
func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	task, exists := m.tasks[key]
	if !exists {
		return nil, fmt.Errorf("task not found with key %s", key)
	}
	return task, nil
}

// FilterCombined filters tasks based on criteria
func (m *MockTaskRepository) FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *models.AgentType, maxPriority *int) ([]*models.Task, error) {
	var result []*models.Task
	for _, task := range m.tasks {
		// Apply filters
		if status != nil && task.Status != *status {
			continue
		}
		if agentType != nil && (task.AgentType == nil || *task.AgentType != *agentType) {
			continue
		}
		if maxPriority != nil && task.Priority > *maxPriority {
			continue
		}
		result = append(result, task)
	}
	return result, nil
}

// Create mocks the Create method
func (m *MockTaskRepository) Create(ctx context.Context, task *models.Task) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, task)
	}
	return nil
}
