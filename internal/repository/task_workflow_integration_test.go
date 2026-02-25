package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskRepository_WithWorkflowConfig tests that TaskRepository can be initialized with
// NewTaskRepositoryWithWorkflow (backward-compat constructor). The workflow param is now
// ignored since workflow validation moved to the service layer (E15-F13).
func TestTaskRepository_WithWorkflowConfig(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create custom workflow config (ignored by constructor, but kept for backward compat)
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

	// Create repository with custom workflow (deprecated constructor, workflow is ignored)
	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	// Verify repository is initialized correctly
	if repo == nil {
		t.Fatal("Expected repository to be initialized")
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

// TestTaskRepository_DefaultConstructor tests that NewTaskRepository creates a working repo
func TestTaskRepository_DefaultConstructor(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create repository with default constructor
	repo := NewTaskRepository(db)

	// Verify repository is initialized correctly
	if repo == nil {
		t.Fatal("Expected repository to be initialized")
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

// TestTaskRepository_ConstructorsEquivalent tests that both constructors produce
// equivalent repositories (since workflow is no longer stored in the repo).
func TestTaskRepository_ConstructorsEquivalent(t *testing.T) {
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
	}

	repo1 := NewTaskRepository(db)
	repo2 := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	// Both should work identically since workflow is no longer stored
	if repo1 == nil || repo2 == nil {
		t.Fatal("Expected both repositories to be initialized")
	}

	// Both can query the database
	ctx := context.Background()
	test.SeedTestData()

	task1, err1 := repo1.GetByKey(ctx, "T-E99-F99-001")
	task2, err2 := repo2.GetByKey(ctx, "T-E99-F99-001")

	if err1 != nil {
		t.Fatalf("repo1.GetByKey failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("repo2.GetByKey failed: %v", err2)
	}

	if task1.Key != task2.Key {
		t.Errorf("Expected same task key, got %s and %s", task1.Key, task2.Key)
	}
}
