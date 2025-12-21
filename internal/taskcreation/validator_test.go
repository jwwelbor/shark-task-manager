package taskcreation

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateTaskInput_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F01")
	depTask := createTestTask(t, db, feature.ID, "T-E01-F01-001", "Dependency Task")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:     "E01",
		FeatureKey:  "F01",
		Title:       "Test Task",
		Description: "Test Description",
		AgentType:   "backend",
		Priority:    5,
		DependsOn:   depTask.Key,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, epic.ID, result.EpicID)
	assert.Equal(t, feature.ID, result.FeatureID)
	assert.Equal(t, "E01-F01", result.NormalizedFeatureKey)
	assert.Equal(t, models.AgentTypeBackend, result.AgentType)
	assert.Len(t, result.ValidatedDependencies, 1)
	assert.Equal(t, depTask.Key, result.ValidatedDependencies[0])
}

func TestValidator_ValidateTaskInput_InvalidEpic(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E99",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "epic E99 does not exist")
	assert.Contains(t, err.Error(), "shark epic list")
}

func TestValidator_ValidateTaskInput_InvalidFeature(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup - create epic but no feature
	createTestEpic(t, db, "E01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F99",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "feature E01-F99 does not exist")
}

func TestValidator_ValidateTaskInput_FeatureWrongEpic(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup - create two epics and a feature belonging to one
	epic1 := createTestEpic(t, db, "E01")
	_ = createTestEpic(t, db, "E02")
	createTestFeature(t, db, epic1.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test - try to create task with E02 epic but E01-F01 feature
	input := TaskInput{
		EpicKey:    "E02",
		FeatureKey: "E01-F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "does not belong to epic")
}

func TestValidator_ValidateTaskInput_InvalidAgentType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "invalid-agent",
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid agent type 'invalid-agent'")
	assert.Contains(t, err.Error(), "frontend, backend, api, testing, devops, general")
}

func TestValidator_ValidateTaskInput_AllAgentTypes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test all valid agent types
	agentTypes := []string{"frontend", "backend", "api", "testing", "devops", "general"}

	for _, agentType := range agentTypes {
		t.Run(agentType, func(t *testing.T) {
			input := TaskInput{
				EpicKey:    "E01",
				FeatureKey: "F01",
				Title:      "Test Task",
				AgentType:  agentType,
				Priority:   5,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			require.NoError(t, err)
			assert.Equal(t, epic.ID, result.EpicID)
			assert.Equal(t, feature.ID, result.FeatureID)
		})
	}
}

func TestValidator_ValidateTaskInput_InvalidPriority(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"Priority 0", 0, true},
		{"Priority 1", 1, false},
		{"Priority 5", 5, false},
		{"Priority 10", 10, false},
		{"Priority 11", 11, true},
		{"Priority -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TaskInput{
				EpicKey:    "E01",
				FeatureKey: "F01",
				Title:      "Test Task",
				AgentType:  "backend",
				Priority:   tt.priority,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "priority must be between 1 and 10")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestValidator_ValidateTaskInput_EmptyTitle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "",
		AgentType:  "backend",
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "title cannot be empty")
}

func TestValidator_ValidateDependencies_SingleDependency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F01")
	depTask := createTestTask(t, db, feature.ID, "T-E01-F01-001", "Dep Task")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
		DependsOn:  depTask.Key,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result.ValidatedDependencies, 1)
	assert.Equal(t, depTask.Key, result.ValidatedDependencies[0])
}

func TestValidator_ValidateDependencies_MultipleDependencies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F01")
	dep1 := createTestTask(t, db, feature.ID, "T-E01-F01-001", "Dep 1")
	dep2 := createTestTask(t, db, feature.ID, "T-E01-F01-002", "Dep 2")
	dep3 := createTestTask(t, db, feature.ID, "T-E01-F01-003", "Dep 3")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test with comma-separated dependencies
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
		DependsOn:  dep1.Key + ", " + dep2.Key + ", " + dep3.Key,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result.ValidatedDependencies, 3)
	assert.Contains(t, result.ValidatedDependencies, dep1.Key)
	assert.Contains(t, result.ValidatedDependencies, dep2.Key)
	assert.Contains(t, result.ValidatedDependencies, dep3.Key)
}

func TestValidator_ValidateDependencies_EmptyDependsOn(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	_ = createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
		DependsOn:  "",
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result.ValidatedDependencies)
}

func TestValidator_ValidateDependencies_NonExistentDependency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
		DependsOn:  "T-E01-F01-999",
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "dependency task T-E01-F01-999 does not exist")
}

func TestValidator_ValidateDependencies_InvalidKeyFormat(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
		DependsOn:  "INVALID-KEY",
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid task key format")
}

func TestValidator_ValidateDependencies_WithWhitespace(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	feature := createTestFeature(t, db, epic.ID, "E01-F01")
	dep1 := createTestTask(t, db, feature.ID, "T-E01-F01-001", "Dep 1")
	dep2 := createTestTask(t, db, feature.ID, "T-E01-F01-002", "Dep 2")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test with extra whitespace
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "backend",
		Priority:   5,
		DependsOn:  "  " + dep1.Key + "  ,  " + dep2.Key + "  ",
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result.ValidatedDependencies, 2)
	assert.Contains(t, result.ValidatedDependencies, dep1.Key)
	assert.Contains(t, result.ValidatedDependencies, dep2.Key)
}

func TestValidateAgentType(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"Frontend", "frontend", false},
		{"Backend", "backend", false},
		{"API", "api", false},
		{"Testing", "testing", false},
		{"DevOps", "devops", false},
		{"General", "general", false},
		{"Invalid", "invalid", true},
		{"Empty", "", true},
		{"Uppercase", "FRONTEND", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentType(tt.agentType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"Minimum valid", 1, false},
		{"Middle valid", 5, false},
		{"Maximum valid", 10, false},
		{"Below minimum", 0, true},
		{"Above maximum", 11, true},
		{"Negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriority(tt.priority)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
