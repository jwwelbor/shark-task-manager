//go:build integration
// +build integration

package taskcreation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskCreation_EndToEnd tests the complete workflow
func TestTaskCreation_EndToEnd(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup test data
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F01")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create task
	input := CreateTaskInput{
		EpicKey:     "E01",
		FeatureKey:  "F01",
		Title:       "Test Task",
		Description: "Test Description",
		AgentType:   "backend",
		Priority:    5,
		DependsOn:   "",
	}

	result, err := creator.CreateTask(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "T-E01-F01-001", result.Task.Key)
	assert.Equal(t, "Test Task", result.Task.Title)
	assert.Equal(t, models.TaskStatusTodo, result.Task.Status)
	assert.NotEmpty(t, result.FilePath)

	// Verify database record
	task, err := taskRepo.GetByKey(context.Background(), result.Task.Key)
	require.NoError(t, err)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, feature.ID, task.FeatureID)
}

func TestTaskCreation_WithDependencies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup test data
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F02")
	depTask := createTestTask(t, db, feature.ID, "T-E01-F02-001", "Dependency Task")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	// Create task with dependency
	input := CreateTaskInput{
		EpicKey:     "E01",
		FeatureKey:  "F02",
		Title:       "Dependent Task",
		Description: "Depends on first task",
		AgentType:   "frontend",
		Priority:    7,
		DependsOn:   depTask.Key,
	}

	result, err := creator.CreateTask(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "T-E01-F02-002", result.Task.Key)
	assert.NotNil(t, result.Task.DependsOn)
}

func TestTaskCreation_SequentialKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E02")
	feature := createTestFeature(t, db, epic.ID, "E02-F01")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	// Create multiple tasks
	for i := 1; i <= 5; i++ {
		input := CreateTaskInput{
			EpicKey:    "E02",
			FeatureKey: "F01",
			Title:      fmt.Sprintf("Task %d", i),
			AgentType:  "general",
			Priority:   5,
		}

		result, err := creator.CreateTask(context.Background(), input)
		require.NoError(t, err)
		expectedKey := fmt.Sprintf("T-E02-F01-%03d", i)
		assert.Equal(t, expectedKey, result.Task.Key)
	}
}

func TestTaskCreation_AllAgentTypes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E03")
	feature := createTestFeature(t, db, epic.ID, "E03-F01")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	agentTypes := []string{"frontend", "backend", "api", "testing", "devops", "general"}

	for idx, agentType := range agentTypes {
		input := CreateTaskInput{
			EpicKey:    "E03",
			FeatureKey: "F01",
			Title:      fmt.Sprintf("%s Task", agentType),
			AgentType:  agentType,
			Priority:   5,
		}

		result, err := creator.CreateTask(context.Background(), input)
		require.NoError(t, err, "Failed for agent type: %s", agentType)
		assert.Equal(t, fmt.Sprintf("T-E03-F01-%03d", idx+1), result.Task.Key)
		assert.Equal(t, agentType, string(*result.Task.AgentType))
	}
}

func TestTaskCreation_ValidationErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E04")
	createTestFeature(t, db, epic.ID, "E04-F01")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	tests := []struct {
		name        string
		input       CreateTaskInput
		expectedErr string
	}{
		{
			name: "Invalid epic",
			input: CreateTaskInput{
				EpicKey:    "E99",
				FeatureKey: "F01",
				Title:      "Test",
				AgentType:  "backend",
				Priority:   5,
			},
			expectedErr: "epic E99 does not exist",
		},
		{
			name: "Invalid feature",
			input: CreateTaskInput{
				EpicKey:    "E04",
				FeatureKey: "F99",
				Title:      "Test",
				AgentType:  "backend",
				Priority:   5,
			},
			expectedErr: "feature E04-F99 does not exist",
		},
		{
			name: "Invalid agent type",
			input: CreateTaskInput{
				EpicKey:    "E04",
				FeatureKey: "F01",
				Title:      "Test",
				AgentType:  "invalid",
				Priority:   5,
			},
			expectedErr: "invalid agent type",
		},
		{
			name: "Invalid priority low",
			input: CreateTaskInput{
				EpicKey:    "E04",
				FeatureKey: "F01",
				Title:      "Test",
				AgentType:  "backend",
				Priority:   0,
			},
			expectedErr: "priority must be between 1 and 10",
		},
		{
			name: "Invalid priority high",
			input: CreateTaskInput{
				EpicKey:    "E04",
				FeatureKey: "F01",
				Title:      "Test",
				AgentType:  "backend",
				Priority:   11,
			},
			expectedErr: "priority must be between 1 and 10",
		},
		{
			name: "Non-existent dependency",
			input: CreateTaskInput{
				EpicKey:    "E04",
				FeatureKey: "F01",
				Title:      "Test",
				AgentType:  "backend",
				Priority:   5,
				DependsOn:  "T-E04-F01-999",
			},
			expectedErr: "dependency task T-E04-F01-999 does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := creator.CreateTask(context.Background(), tt.input)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestTaskCreation_FileGeneration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E05")
	feature := createTestFeature(t, db, epic.ID, "E05-F01")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	// Create task
	input := CreateTaskInput{
		EpicKey:     "E05",
		FeatureKey:  "F01",
		Title:       "File Generation Test",
		Description: "Testing file creation",
		AgentType:   "frontend",
		Priority:    8,
	}

	result, err := creator.CreateTask(context.Background(), input)
	require.NoError(t, err)

	// Check file path uses hierarchical pattern
	assert.Contains(t, result.FilePath, "docs/plan")
	assert.Contains(t, result.FilePath, "/tasks/")
	assert.Contains(t, result.FilePath, result.Task.Key)
	assert.Contains(t, result.FilePath, ".md")
}

func TestTaskCreation_HistoryRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E06")
	feature := createTestFeature(t, db, epic.ID, "E06-F01")

	// Create components
	taskRepo := repository.NewTaskRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	epicRepo := repository.NewEpicRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, t.TempDir(), nil)

	// Create task
	input := CreateTaskInput{
		EpicKey:    "E06",
		FeatureKey: "F01",
		Title:      "History Test",
		AgentType:  "backend",
		Priority:   5,
	}

	result, err := creator.CreateTask(context.Background(), input)
	require.NoError(t, err)

	// Verify history record was created
	history, err := historyRepo.ListByTask(context.Background(), result.Task.ID)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "", history[0].OldStatus)
	assert.Equal(t, "todo", history[0].NewStatus)
}
