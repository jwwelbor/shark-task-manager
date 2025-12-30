package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
)

// TestFilterByMetadataAgentType tests filtering tasks by agent type from workflow metadata
func TestFilterByMetadataAgentType(t *testing.T) {
	ctx := context.Background()

	// Setup test database
	database := test.GetTestDB()
	dbWrapper := NewDB(database)

	// Clean up existing test data BEFORE test (critical for test isolation)
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Also clean up any leftover tasks that might interfere with status-based queries
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE status = 'in_progress' AND key NOT LIKE 'T-E99-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE status = 'ready_for_review' AND key NOT LIKE 'T-E99-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE status = 'todo' AND key NOT LIKE 'T-E99-F01-%'")

	// Create test epic and feature
	epicRepo := NewEpicRepository(dbWrapper)
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic",
		Status:   "active",
		Priority: "medium",
	}
	err := epicRepo.Create(ctx, epic)
	assert.NoError(t, err)

	featureRepo := NewFeatureRepository(dbWrapper)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E99-F01",
		Title:  "Test Feature",
		Status: "active",
	}
	err = featureRepo.Create(ctx, feature)
	assert.NoError(t, err)

	// Create test workflow config with metadata
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["ready_for_review"],
    "ready_for_review": ["completed"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "phase": "planning",
      "agent_types": ["business-analyst", "project-manager"]
    },
    "in_progress": {
      "phase": "development",
      "agent_types": ["developer", "backend", "frontend"]
    },
    "ready_for_review": {
      "phase": "review",
      "agent_types": ["qa", "tech-lead"]
    },
    "completed": {
      "phase": "done",
      "agent_types": []
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`

	err = os.WriteFile(configPath, []byte(configJSON), 0644)
	assert.NoError(t, err)

	// Clear workflow cache and load config
	config.ClearWorkflowCache()
	workflow, err := config.LoadWorkflowConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)

	// Create test tasks with different statuses
	taskRepo := NewTaskRepository(dbWrapper)

	task1 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-001",
		Title:     "Planning Task",
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task1)
	assert.NoError(t, err)

	task2 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-002",
		Title:     "Development Task 1",
		Status:    models.TaskStatus("in_progress"),
		Priority:  3,
	}
	err = taskRepo.Create(ctx, task2)
	assert.NoError(t, err)

	task3 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-003",
		Title:     "Development Task 2",
		Status:    models.TaskStatus("in_progress"),
		Priority:  4,
	}
	err = taskRepo.Create(ctx, task3)
	assert.NoError(t, err)

	task4 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-004",
		Title:     "Review Task",
		Status:    models.TaskStatus("ready_for_review"),
		Priority:  2,
	}
	err = taskRepo.Create(ctx, task4)
	assert.NoError(t, err)

	task5 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F01-005",
		Title:     "Completed Task",
		Status:    models.TaskStatus("completed"),
		Priority:  1,
	}
	err = taskRepo.Create(ctx, task5)
	assert.NoError(t, err)

	// Test filtering by developer agent type
	// Should return tasks in "in_progress" status
	devTasks, err := taskRepo.FilterByMetadataAgentType(ctx, "developer", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(devTasks), "Should find 2 tasks for developer")
	assert.Contains(t, []string{devTasks[0].Key, devTasks[1].Key}, "T-E99-F01-002")
	assert.Contains(t, []string{devTasks[0].Key, devTasks[1].Key}, "T-E99-F01-003")

	// Test filtering by QA agent type
	// Should return tasks in "ready_for_review" status
	qaTasks, err := taskRepo.FilterByMetadataAgentType(ctx, "qa", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(qaTasks), "Should find 1 task for QA")
	assert.Equal(t, "T-E99-F01-004", qaTasks[0].Key)

	// Test filtering by business-analyst agent type
	// Should return tasks in "todo" status
	baTasks, err := taskRepo.FilterByMetadataAgentType(ctx, "business-analyst", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(baTasks), "Should find 1 task for business analyst")
	assert.Equal(t, "T-E99-F01-001", baTasks[0].Key)

	// Test filtering by unknown agent type
	// Should return empty list
	unknownTasks, err := taskRepo.FilterByMetadataAgentType(ctx, "unknown-agent", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(unknownTasks), "Should find 0 tasks for unknown agent")

	// Cleanup
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-F01-%'")
	defer database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F01'")
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")
}

// TestFilterByMetadataPhase tests filtering tasks by workflow phase from metadata
func TestFilterByMetadataPhase(t *testing.T) {
	ctx := context.Background()

	// Setup test database
	database := test.GetTestDB()
	dbWrapper := NewDB(database)

	// Clean up existing test data BEFORE test (critical for test isolation)
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Also clean up any leftover tasks that might interfere with status-based queries
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE status = 'in_progress' AND key NOT LIKE 'T-E98-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE status = 'ready_for_review' AND key NOT LIKE 'T-E98-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE status = 'todo' AND key NOT LIKE 'T-E98-F01-%'")

	// Create test epic and feature
	epicRepo := NewEpicRepository(dbWrapper)
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic",
		Status:   "active",
		Priority: "medium",
	}
	err := epicRepo.Create(ctx, epic)
	assert.NoError(t, err)

	featureRepo := NewFeatureRepository(dbWrapper)
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E98-F01",
		Title:  "Test Feature",
		Status: "active",
	}
	err = featureRepo.Create(ctx, feature)
	assert.NoError(t, err)

	// Create test workflow config with metadata
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["ready_for_review"],
    "ready_for_review": ["completed"],
    "completed": []
  },
  "status_metadata": {
    "todo": {
      "phase": "planning"
    },
    "in_progress": {
      "phase": "development"
    },
    "ready_for_review": {
      "phase": "review"
    },
    "completed": {
      "phase": "done"
    }
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}`

	err = os.WriteFile(configPath, []byte(configJSON), 0644)
	assert.NoError(t, err)

	// Clear workflow cache and load config
	config.ClearWorkflowCache()
	workflow, err := config.LoadWorkflowConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)

	// Create test tasks with different statuses
	taskRepo := NewTaskRepository(dbWrapper)

	task1 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E98-F01-010",
		Title:     "Planning Task",
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task1)
	assert.NoError(t, err)

	task2 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E98-F01-011",
		Title:     "Development Task 1",
		Status:    models.TaskStatus("in_progress"),
		Priority:  3,
	}
	err = taskRepo.Create(ctx, task2)
	assert.NoError(t, err)

	task3 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E98-F01-012",
		Title:     "Development Task 2",
		Status:    models.TaskStatus("in_progress"),
		Priority:  4,
	}
	err = taskRepo.Create(ctx, task3)
	assert.NoError(t, err)

	task4 := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E98-F01-013",
		Title:     "Review Task",
		Status:    models.TaskStatus("ready_for_review"),
		Priority:  2,
	}
	err = taskRepo.Create(ctx, task4)
	assert.NoError(t, err)

	// Test filtering by development phase
	// Should return tasks in "in_progress" status
	devTasks, err := taskRepo.FilterByMetadataPhase(ctx, "development", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(devTasks), "Should find 2 tasks in development phase")
	assert.Contains(t, []string{devTasks[0].Key, devTasks[1].Key}, "T-E98-F01-011")
	assert.Contains(t, []string{devTasks[0].Key, devTasks[1].Key}, "T-E98-F01-012")

	// Test filtering by review phase
	// Should return tasks in "ready_for_review" status
	reviewTasks, err := taskRepo.FilterByMetadataPhase(ctx, "review", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(reviewTasks), "Should find 1 task in review phase")
	assert.Equal(t, "T-E98-F01-013", reviewTasks[0].Key)

	// Test filtering by planning phase
	planningTasks, err := taskRepo.FilterByMetadataPhase(ctx, "planning", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(planningTasks), "Should find 1 task in planning phase")
	assert.Equal(t, "T-E98-F01-010", planningTasks[0].Key)

	// Test filtering by unknown phase
	unknownTasks, err := taskRepo.FilterByMetadataPhase(ctx, "unknown-phase", workflow)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(unknownTasks), "Should find 0 tasks for unknown phase")

	// Cleanup
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F01-%'")
	defer database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F01'")
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
}
