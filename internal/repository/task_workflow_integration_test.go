package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskRepository_WithWorkflowConfig tests that TaskRepository can be initialized with a workflow config
func TestTaskRepository_WithWorkflowConfig(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create custom workflow config
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":        {"in_progress"},
			"in_progress": {"done"},
			"done":        {},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"done"},
		},
	}

	// Create repository with custom workflow
	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	// Verify repository is initialized correctly
	if repo == nil {
		t.Fatal("Expected repository to be initialized")
	}

	if repo.workflow == nil {
		t.Fatal("Expected workflow config to be set")
	}

	if repo.workflow.Version != "1.0" {
		t.Errorf("Expected workflow version 1.0, got %s", repo.workflow.Version)
	}

	// Verify we can use the repository normally
	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if task == nil {
		t.Fatal("Expected task to be retrieved")
	}
}

// TestTaskRepository_DefaultWorkflow tests that TaskRepository uses default workflow when no config provided
func TestTaskRepository_DefaultWorkflow(t *testing.T) {
	database := test.GetTestDB()
	db := NewDB(database)

	// Create repository without workflow (should use default)
	repo := NewTaskRepository(db)

	// Verify repository uses default workflow
	if repo.workflow == nil {
		t.Fatal("Expected repository to have default workflow")
	}

	// Default workflow should have standard statuses
	if repo.workflow.StatusFlow == nil {
		t.Fatal("Expected default workflow to have status flow")
	}

	// Verify default workflow has expected statuses
	expectedStatuses := []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
	for _, status := range expectedStatuses {
		if _, exists := repo.workflow.StatusFlow[status]; !exists {
			t.Errorf("Expected default workflow to include status '%s'", status)
		}
	}
}

// TestTaskRepository_WorkflowGetter tests GetWorkflow method
func TestTaskRepository_WorkflowGetter(t *testing.T) {
	database := test.GetTestDB()
	db := NewDB(database)

	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"backlog": {"todo"},
			"todo":    {"doing"},
			"doing":   {"done"},
			"done":    {},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"backlog"},
			config.CompleteStatusKey: {"done"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	// Get workflow from repository
	retrievedWorkflow := repo.GetWorkflow()

	if retrievedWorkflow == nil {
		t.Fatal("Expected GetWorkflow to return workflow config")
	}

	if retrievedWorkflow.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", retrievedWorkflow.Version)
	}

	// Verify status flow matches
	if len(retrievedWorkflow.StatusFlow) != 4 {
		t.Errorf("Expected 4 statuses, got %d", len(retrievedWorkflow.StatusFlow))
	}

	if _, exists := retrievedWorkflow.StatusFlow["backlog"]; !exists {
		t.Error("Expected workflow to include 'backlog' status")
	}
}
