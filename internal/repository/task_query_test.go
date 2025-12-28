package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestListByEpic retrieves all tasks for an epic without ambiguous column errors
func TestListByEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up before seeding
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	test.SeedTestData()

	// ListByEpic uses a JOIN query that should not have ambiguous column errors
	tasks, err := taskRepo.ListByEpic(ctx, "E99")
	if err != nil {
		t.Fatalf("Failed to list tasks by epic: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks for E99 epic, got empty list")
	}

	// Verify tasks have expected fields populated
	for _, task := range tasks {
		if task.Key == "" {
			t.Error("Expected task key to be populated")
		}
		if task.FeatureID == 0 {
			t.Error("Expected task feature_id to be populated")
		}
	}
}

// TestFilterCombined with epic filter uses JOINs without ambiguous columns
func TestFilterCombined(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	taskRepo := NewTaskRepository(db)

	// Clean up before seeding
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key IN ('E99-F99')")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key IN ('E99')")

	test.SeedTestData()

	// FilterCombined uses a JOIN query with epicKey parameter
	epicKey := "E99"
	tasks, err := taskRepo.FilterCombined(ctx, nil, &epicKey, nil, nil)
	if err != nil {
		t.Fatalf("Failed to filter tasks by epic: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks for E99 epic, got empty list")
	}

	// Verify tasks have expected fields populated
	for _, task := range tasks {
		if task.Key == "" {
			t.Error("Expected task key to be populated")
		}
	}
}
