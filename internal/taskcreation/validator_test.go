package taskcreation

import (
	"context"
	"testing"

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
	assert.Equal(t, "backend", result.AgentType)
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

	// Test - whitespace-only agent type should fail
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "   ",
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot be empty")
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

	// Test all valid agent types - both standard and custom
	agentTypes := []string{
		// Standard types
		"frontend", "backend", "api", "testing", "devops", "general",
		// Custom types (now supported)
		"frontend-react", "ml-engineer", "custom-123",
	}

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

// TestValidator_CustomAgentType_Success tests task creation with custom agent types
func TestValidator_CustomAgentType_Success(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
	}{
		{"custom architect", "architect"},
		{"custom business-analyst", "business-analyst"},
		{"custom qa", "qa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()

			// Setup
			epic := createTestEpic(t, db, "E01")
			f := createTestFeature(t, db, epic.ID, "E01-F01")

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
				AgentType:  tt.agentType,
				Priority:   5,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, result)
            assert.Equal(t, tt.agentType, result.AgentType)
			_ = f                                           // Use feature to avoid unused variable
		})
	}
}

// TestValidator_CustomAgentType_BackwardCompatibility tests that standard types still work
func TestValidator_CustomAgentType_BackwardCompatibility(t *testing.T) {
	standardTypes := []string{"frontend", "backend", "api", "testing", "devops", "general"}

	for _, agentType := range standardTypes {
		t.Run(agentType, func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()

			// Setup
			epic := createTestEpic(t, db, "E01")
			f := createTestFeature(t, db, epic.ID, "E01-F01")

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
				AgentType:  agentType,
				Priority:   5,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			// Assert
			require.NoError(t, err, "Standard agent type %q should work", agentType)
			assert.NotNil(t, result)
			assert.Equal(t, agentType, result.AgentType) //nolint:staticcheck // AgentType is deprecated but still used in Task model
			_ = f                                        // Use feature to avoid unused variable
		})
	}
}

// TestValidator_MultiAgentWorkflow_AllCustomTypes tests all multi-agent workflow custom types
func TestValidator_MultiAgentWorkflow_AllCustomTypes(t *testing.T) {
	customTypes := []string{
		"architect",
		"business-analyst",
		"product-manager",
		"qa",
		"tech-lead",
		"ux-designer",
		"data-engineer",
		"ml-engineer",
		"security-analyst",
	}

	for _, agentType := range customTypes {
		t.Run(agentType, func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()

			// Setup
			epic := createTestEpic(t, db, "E01")
			f := createTestFeature(t, db, epic.ID, "E01-F01")

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
				AgentType:  agentType,
				Priority:   5,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			// Assert
			require.NoError(t, err, "Custom agent type %q should work", agentType)
			assert.NotNil(t, result)
			assert.Equal(t, agentType, result.AgentType) //nolint:staticcheck // AgentType is deprecated but still used in Task model
			_ = f                                        // Use feature to avoid unused variable
		})
	}
}

// TestValidator_SpecializedTeamRoles tests specialized team role custom types
func TestValidator_SpecializedTeamRoles(t *testing.T) {
	specializedRoles := []string{
		"database-admin",
		"cloud-architect",
		"mobile-developer",
		"accessibility-expert",
		"performance-engineer",
		"documentation-writer",
	}

	for _, agentType := range specializedRoles {
		t.Run(agentType, func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()

			// Setup
			epic := createTestEpic(t, db, "E01")
			f := createTestFeature(t, db, epic.ID, "E01-F01")

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
				AgentType:  agentType,
				Priority:   5,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			// Assert
			require.NoError(t, err, "Specialized role %q should work", agentType)
			assert.NotNil(t, result)
			assert.Equal(t, agentType, result.AgentType) //nolint:staticcheck // AgentType is deprecated but still used in Task model
			_ = f                                        // Use feature to avoid unused variable
		})
	}
}

// TestValidator_CustomAgentType_EdgeCases tests edge cases for custom agent types
func TestValidator_CustomAgentType_EdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		agentType string
		wantErr   bool
		errMsg    string
	}{
		{"single character", "a", false, ""},
		{"with numbers", "agent123", false, ""},
		{"with special characters", "agent@type", false, ""},
		{"very long name", "very-long-custom-agent-type-name-that-is-still-valid", false, ""},
		{"mixed case", "BackEnd-Engineer", false, ""},
		{"whitespace only", "   ", true, "cannot be empty"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := setupTestDB(t)
			defer cleanup()

			// Setup
			epic := createTestEpic(t, db, "E01")
			f := createTestFeature(t, db, epic.ID, "E01-F01")

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
				AgentType:  tt.agentType,
				Priority:   5,
			}

			result, err := validator.ValidateTaskInput(context.Background(), input)

			if tt.wantErr {
				require.Error(t, err, "Expected error for agent type %q", tt.agentType)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err, "Agent type %q should be valid", tt.agentType)
				assert.NotNil(t, result)
				assert.Equal(t, tt.agentType, result.AgentType) //nolint:staticcheck // AgentType is deprecated but still used in Task model
			}
			_ = f // Use feature to avoid unused variable
		})
	}
}

// TestValidator_DefaultAgentType_WhenEmpty tests default to 'general' when agent type is empty
func TestValidator_DefaultAgentType_WhenEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	epic := createTestEpic(t, db, "E01")
	f := createTestFeature(t, db, epic.ID, "E01-F01")

	// Create validator
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)

	// Test - no agent type provided
	input := TaskInput{
		EpicKey:    "E01",
		FeatureKey: "F01",
		Title:      "Test Task",
		AgentType:  "", // Empty - should default to general
		Priority:   5,
	}

	result, err := validator.ValidateTaskInput(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "general", result.AgentType)
	_ = f // Use feature to avoid unused variable
}
