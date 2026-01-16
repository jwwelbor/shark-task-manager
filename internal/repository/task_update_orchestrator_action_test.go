package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskRepository_UpdateStatusWithOrchestratorAction tests that UpdateStatus returns orchestrator_action
func TestTaskRepository_UpdateStatusWithOrchestratorAction(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E99-F01-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic for Orchestrator Actions",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create test feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E99-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create task
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E99-F01-001",
		Title:     "Test Task for Orchestrator Action",
		Status:    "draft",
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create workflow with orchestrator action for ready_for_development status
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":          {"in_development"},
			"in_development": {}, // Terminal state for this test
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"draft": {
				Color: "gray",
				Phase: "planning",
			},
			"in_development": {
				Color: "yellow",
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"backend", "testing"},
					InstructionTemplate: "Implement {task_id} following the specification",
				},
			},
		},
	}

	// Create repo with custom workflow
	taskRepoWithWorkflow := NewTaskRepositoryWithWorkflow(db, workflow)

	// Call UpdateStatusWithAction - the new method we're testing
	// This should return both the updated task and the orchestrator action
	updatedTask, action, err := taskRepoWithWorkflow.UpdateStatusWithAction(ctx, task.Key, "in_development")

	// Assertions
	require.NoError(t, err, "UpdateStatusWithAction should not error")
	require.NotNil(t, updatedTask, "Updated task should not be nil")
	assert.Equal(t, models.TaskStatus("in_development"), updatedTask.Status, "Task status should be updated")

	// Verify orchestrator action is returned
	require.NotNil(t, action, "Orchestrator action should be returned for status with action")
	assert.Equal(t, config.ActionSpawnAgent, action.Action)
	assert.Equal(t, "developer", action.AgentType)
	assert.Equal(t, []string{"backend", "testing"}, action.Skills)
	// Template should be populated with task ID
	assert.Equal(t, "Implement T-E99-F01-001 following the specification", action.Instruction)
}

// TestTaskRepository_UpdateStatusWithoutOrchestratorAction tests backward compatibility
func TestTaskRepository_UpdateStatusWithoutOrchestratorAction(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E99-F02-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F02'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create test feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E99-F02",
		Title:  "Test Feature No Action",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create task
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E99-F02-001",
		Title:     "Task without action",
		Status:    "draft",
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create workflow WITHOUT orchestrator action
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":          {"in_development"},
			"in_development": {}, // Terminal state for this test
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"draft": {
				Color: "gray",
				Phase: "planning",
			},
			"in_development": {
				Color: "yellow",
				Phase: "development",
				// No OrchestratorAction defined - should be backward compatible
			},
		},
	}

	// Create repo with custom workflow
	taskRepoWithWorkflow := NewTaskRepositoryWithWorkflow(db, workflow)

	// Call UpdateStatusWithAction
	updatedTask, action, err := taskRepoWithWorkflow.UpdateStatusWithAction(ctx, task.Key, "in_development")

	// Assertions
	require.NoError(t, err, "UpdateStatusWithAction should not error even without action")
	require.NotNil(t, updatedTask, "Updated task should not be nil")
	assert.Equal(t, models.TaskStatus("in_development"), updatedTask.Status)

	// Action should be nil when not defined (backward compatible)
	assert.Nil(t, action, "Orchestrator action should be nil when not defined")
}

// TestTaskRepository_UpdateStatusPopulatesTemplateVariables tests template variable population
func TestTaskRepository_UpdateStatusPopulatesTemplateVariables(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E99-F03-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F03'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		Key:           "E99",
		Title:         "Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create test feature
	testFeature := &models.Feature{
		EpicID: testEpic.ID,
		Key:    "E99-F03",
		Title:  "Test Feature",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create task
	task := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E99-F03-001",
		Title:     "Template Test Task",
		Status:    "ready_for_development",
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create workflow with template variables
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"ready_for_development": {"in_development"},
			"in_development":        {}, // Terminal state for this test
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_development": {
				Color: "gray",
				Phase: "planning",
			},
			"in_development": {
				Color: "yellow",
				Phase: "development",
				OrchestratorAction: &config.OrchestratorAction{
					Action:              config.ActionSpawnAgent,
					AgentType:           "developer",
					Skills:              []string{"backend"},
					InstructionTemplate: "Begin implementation for task {task_id}",
				},
			},
		},
	}

	taskRepoWithWorkflow := NewTaskRepositoryWithWorkflow(db, workflow)

	// Call UpdateStatusWithAction
	_, action, err := taskRepoWithWorkflow.UpdateStatusWithAction(ctx, task.Key, "in_development")

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, action, "Action should be returned")

	// Verify template variable is populated
	assert.Equal(t, "Begin implementation for task T-E99-F03-001", action.Instruction)
	assert.NotContains(t, action.Instruction, "{task_id}", "Template variable should be replaced")
}
