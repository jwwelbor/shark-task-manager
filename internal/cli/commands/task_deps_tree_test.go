package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

// MockTaskRepositoryForTree is a mock for testing tree visualization
type MockTaskRepositoryForTree struct {
	tasks map[string]*models.Task
}

func NewMockTaskRepositoryForTree() *MockTaskRepositoryForTree {
	return &MockTaskRepositoryForTree{
		tasks: make(map[string]*models.Task),
	}
}

func (m *MockTaskRepositoryForTree) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	task, exists := m.tasks[key]
	if !exists {
		return nil, fmt.Errorf("task not found with key %s", key)
	}
	return task, nil
}

func (m *MockTaskRepositoryForTree) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	for _, task := range m.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task not found with id %d", id)
}

func (m *MockTaskRepositoryForTree) AddTask(task *models.Task) {
	m.tasks[task.Key] = task
}

// MockRelationshipRepositoryForTree is a mock for testing tree visualization
type MockRelationshipRepositoryForTree struct {
	relationships []*models.TaskRelationship
}

func NewMockRelationshipRepositoryForTree() *MockRelationshipRepositoryForTree {
	return &MockRelationshipRepositoryForTree{
		relationships: []*models.TaskRelationship{},
	}
}

func (m *MockRelationshipRepositoryForTree) GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error) {
	var result []*models.TaskRelationship
	for _, rel := range m.relationships {
		if rel.FromTaskID == taskID {
			// Filter by relationship types if specified
			if len(relTypes) > 0 {
				for _, relType := range relTypes {
					if string(rel.RelationshipType) == relType {
						result = append(result, rel)
						break
					}
				}
			} else {
				result = append(result, rel)
			}
		}
	}
	return result, nil
}

func (m *MockRelationshipRepositoryForTree) GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error) {
	var result []*models.TaskRelationship
	for _, rel := range m.relationships {
		if rel.ToTaskID == taskID {
			// Filter by relationship types if specified
			if len(relTypes) > 0 {
				for _, relType := range relTypes {
					if string(rel.RelationshipType) == relType {
						result = append(result, rel)
						break
					}
				}
			} else {
				result = append(result, rel)
			}
		}
	}
	return result, nil
}

func (m *MockRelationshipRepositoryForTree) AddRelationship(rel *models.TaskRelationship) {
	m.relationships = append(m.relationships, rel)
}

// TestBuildDependencyTreeSimple tests building a simple dependency tree
func TestBuildDependencyTreeSimple(t *testing.T) {
	ctx := context.Background()
	taskRepo := NewMockTaskRepositoryForTree()
	relRepo := NewMockRelationshipRepositoryForTree()

	// Create tasks:
	// T3 depends on T2
	// T2 depends on T1
	// T1 has no dependencies
	task1 := &models.Task{
		ID:     1,
		Key:    "T-E01-F01-001",
		Title:  "Base task",
		Status: models.TaskStatusCompleted,
	}
	task2 := &models.Task{
		ID:     2,
		Key:    "T-E01-F01-002",
		Title:  "Middle task",
		Status: models.TaskStatusInProgress,
	}
	task3 := &models.Task{
		ID:     3,
		Key:    "T-E01-F01-003",
		Title:  "Top task",
		Status: models.TaskStatusTodo,
	}

	taskRepo.AddTask(task1)
	taskRepo.AddTask(task2)
	taskRepo.AddTask(task3)

	// T2 depends on T1
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               1,
		FromTaskID:       2,
		ToTaskID:         1,
		RelationshipType: models.RelationshipDependsOn,
	})

	// T3 depends on T2
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               2,
		FromTaskID:       3,
		ToTaskID:         2,
		RelationshipType: models.RelationshipDependsOn,
	})

	// Build tree for T3
	var taskRepoInterface TaskRepositoryInterfaceWithID = taskRepo
	var relRepoInterface RelationshipRepositoryInterface = relRepo
	tree, err := buildDependencyTree(ctx, taskRepoInterface, relRepoInterface, task3, make(map[int64]bool), 0, 5)

	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Equal(t, "T-E01-F01-003", tree.Task.Key)
	assert.Equal(t, 1, len(tree.Dependencies))
	assert.Equal(t, "T-E01-F01-002", tree.Dependencies[0].Task.Key)
	assert.Equal(t, 1, len(tree.Dependencies[0].Dependencies))
	assert.Equal(t, "T-E01-F01-001", tree.Dependencies[0].Dependencies[0].Task.Key)
}

// TestBuildDependencyTreeMultipleDeps tests a task with multiple dependencies
func TestBuildDependencyTreeMultipleDeps(t *testing.T) {
	ctx := context.Background()
	taskRepo := NewMockTaskRepositoryForTree()
	relRepo := NewMockRelationshipRepositoryForTree()

	// Create tasks:
	// T4 depends on T1, T2, T3
	task1 := &models.Task{
		ID:     1,
		Key:    "T-E01-F01-001",
		Title:  "Dep 1",
		Status: models.TaskStatusCompleted,
	}
	task2 := &models.Task{
		ID:     2,
		Key:    "T-E01-F01-002",
		Title:  "Dep 2",
		Status: models.TaskStatusCompleted,
	}
	task3 := &models.Task{
		ID:     3,
		Key:    "T-E01-F01-003",
		Title:  "Dep 3",
		Status: models.TaskStatusInProgress,
	}
	task4 := &models.Task{
		ID:     4,
		Key:    "T-E01-F01-004",
		Title:  "Main task",
		Status: models.TaskStatusTodo,
	}

	taskRepo.AddTask(task1)
	taskRepo.AddTask(task2)
	taskRepo.AddTask(task3)
	taskRepo.AddTask(task4)

	// T4 depends on T1, T2, T3
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               1,
		FromTaskID:       4,
		ToTaskID:         1,
		RelationshipType: models.RelationshipDependsOn,
	})
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               2,
		FromTaskID:       4,
		ToTaskID:         2,
		RelationshipType: models.RelationshipDependsOn,
	})
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               3,
		FromTaskID:       4,
		ToTaskID:         3,
		RelationshipType: models.RelationshipDependsOn,
	})

	// Build tree for T4
	var taskRepoInterface TaskRepositoryInterfaceWithID = taskRepo
	var relRepoInterface RelationshipRepositoryInterface = relRepo
	tree, err := buildDependencyTree(ctx, taskRepoInterface, relRepoInterface, task4, make(map[int64]bool), 0, 5)

	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Equal(t, "T-E01-F01-004", tree.Task.Key)
	assert.Equal(t, 3, len(tree.Dependencies))
}

// TestBuildDependencyTreeNoDeps tests a task with no dependencies
func TestBuildDependencyTreeNoDeps(t *testing.T) {
	ctx := context.Background()
	taskRepo := NewMockTaskRepositoryForTree()
	relRepo := NewMockRelationshipRepositoryForTree()

	task1 := &models.Task{
		ID:     1,
		Key:    "T-E01-F01-001",
		Title:  "Standalone task",
		Status: models.TaskStatusTodo,
	}

	taskRepo.AddTask(task1)

	// Build tree for T1 (no dependencies)
	var taskRepoInterface TaskRepositoryInterfaceWithID = taskRepo
	var relRepoInterface RelationshipRepositoryInterface = relRepo
	tree, err := buildDependencyTree(ctx, taskRepoInterface, relRepoInterface, task1, make(map[int64]bool), 0, 5)

	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Equal(t, "T-E01-F01-001", tree.Task.Key)
	assert.Equal(t, 0, len(tree.Dependencies))
}

// TestBuildDependencyTreeCircular tests circular dependency detection
func TestBuildDependencyTreeCircular(t *testing.T) {
	ctx := context.Background()
	taskRepo := NewMockTaskRepositoryForTree()
	relRepo := NewMockRelationshipRepositoryForTree()

	// Create tasks with circular dependency:
	// T1 -> T2 -> T3 -> T1
	task1 := &models.Task{
		ID:     1,
		Key:    "T-E01-F01-001",
		Title:  "Task 1",
		Status: models.TaskStatusTodo,
	}
	task2 := &models.Task{
		ID:     2,
		Key:    "T-E01-F01-002",
		Title:  "Task 2",
		Status: models.TaskStatusTodo,
	}
	task3 := &models.Task{
		ID:     3,
		Key:    "T-E01-F01-003",
		Title:  "Task 3",
		Status: models.TaskStatusTodo,
	}

	taskRepo.AddTask(task1)
	taskRepo.AddTask(task2)
	taskRepo.AddTask(task3)

	// T1 depends on T2
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               1,
		FromTaskID:       1,
		ToTaskID:         2,
		RelationshipType: models.RelationshipDependsOn,
	})

	// T2 depends on T3
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               2,
		FromTaskID:       2,
		ToTaskID:         3,
		RelationshipType: models.RelationshipDependsOn,
	})

	// T3 depends on T1 (creates cycle)
	relRepo.AddRelationship(&models.TaskRelationship{
		ID:               3,
		FromTaskID:       3,
		ToTaskID:         1,
		RelationshipType: models.RelationshipDependsOn,
	})

	// Build tree for T1 - should stop at circular reference
	var taskRepoInterface TaskRepositoryInterfaceWithID = taskRepo
	var relRepoInterface RelationshipRepositoryInterface = relRepo
	tree, err := buildDependencyTree(ctx, taskRepoInterface, relRepoInterface, task1, make(map[int64]bool), 0, 5)

	assert.NoError(t, err)
	assert.NotNil(t, tree)
	// Should detect the cycle and not recurse infinitely
	assert.True(t, tree.HasCycle || len(tree.Dependencies) > 0)
}

// TestRenderTreeSimple tests ASCII tree rendering
func TestRenderTreeSimple(t *testing.T) {
	// Create a simple tree structure
	tree := &DependencyTree{
		Task: &models.Task{
			Key:    "T-E01-F01-003",
			Title:  "Top task",
			Status: models.TaskStatusTodo,
		},
		Dependencies: []*DependencyTree{
			{
				Task: &models.Task{
					Key:    "T-E01-F01-002",
					Title:  "Middle task",
					Status: models.TaskStatusInProgress,
				},
				Dependencies: []*DependencyTree{
					{
						Task: &models.Task{
							Key:    "T-E01-F01-001",
							Title:  "Base task",
							Status: models.TaskStatusCompleted,
						},
						Dependencies: []*DependencyTree{},
					},
				},
			},
		},
	}

	output := renderTree(tree, "", true)

	// Verify output contains task keys and tree structure
	assert.Contains(t, output, "T-E01-F01-003")
	assert.Contains(t, output, "T-E01-F01-002")
	assert.Contains(t, output, "T-E01-F01-001")
	assert.Contains(t, output, "└──") // Tree branch character
}

// TestGetStatusIconForTree tests status icon rendering
func TestGetStatusIconForTree(t *testing.T) {
	tests := []struct {
		status       models.TaskStatus
		expectedIcon string
	}{
		{models.TaskStatusCompleted, "✓"},
		{models.TaskStatusInProgress, "•"},
		{models.TaskStatusBlocked, "✗"},
		{models.TaskStatusTodo, "○"},
		{models.TaskStatusReadyForReview, "⊙"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			icon := getStatusIcon(string(tt.status))
			assert.Equal(t, tt.expectedIcon, icon)
		})
	}
}
