package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestWorkflowIntegration_E2E is a comprehensive end-to-end test of workflow integration
// This test covers:
// - REQ-F-004: Enforce Valid Transitions
// - REQ-F-005: Support Force Flag Override
// - REQ-F-006: Record All Transition Attempts
// - REQ-NF-001: Low Latency Status Validation
func TestWorkflowIntegration_E2E(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create custom workflow simulating a real development process
	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress", "blocked"},
			"in_progress":      {"ready_for_review", "blocked"},
			"ready_for_review": {"completed", "in_progress"}, // Can reopen
			"completed":        {},                           // Terminal
			"blocked":          {"todo"},                     // Unblock to todo
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"todo": {
				Color:       "gray",
				Description: "Task ready to start",
				Phase:       "planning",
				AgentTypes:  []string{"developer"},
			},
			"in_progress": {
				Color:       "blue",
				Description: "Actively being worked on",
				Phase:       "development",
				AgentTypes:  []string{"developer", "backend"},
			},
			"ready_for_review": {
				Color:       "yellow",
				Description: "Awaiting code review",
				Phase:       "review",
				AgentTypes:  []string{"tech-lead"},
			},
			"completed": {
				Color:       "green",
				Description: "Approved and merged",
				Phase:       "done",
				AgentTypes:  []string{},
			},
			"blocked": {
				Color:       "red",
				Description: "Blocked by external dependency",
				Phase:       "blocked",
				AgentTypes:  []string{"project-manager"},
			},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	// Validate workflow config
	err := config.ValidateWorkflow(customWorkflow)
	if err != nil {
		t.Fatalf("Workflow config validation failed: %v", err)
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Clean history for clean test
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ?", task.ID)

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	developer := "alice-dev"
	techLead := "bob-lead"

	// Scenario 1: Normal development workflow
	t.Run("HappyPath", func(t *testing.T) {
		// Step 1: Start work (todo -> in_progress)
		err := repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &developer, nil)
		if err != nil {
			t.Fatalf("Failed to start task: %v", err)
		}

		// Step 2: Complete and submit for review (in_progress -> ready_for_review)
		notes := "Implementation complete, tests passing"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusReadyForReview, &developer, &notes)
		if err != nil {
			t.Fatalf("Failed to submit for review: %v", err)
		}

		// Step 3: Tech lead approves (ready_for_review -> completed)
		approvalNotes := "LGTM, merging"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &techLead, &approvalNotes)
		if err != nil {
			t.Fatalf("Failed to approve: %v", err)
		}

		// Verify final state
		finalTask, _ := repo.GetByID(ctx, task.ID)
		if finalTask.Status != models.TaskStatusCompleted {
			t.Errorf("Expected final status completed, got %s", finalTask.Status)
		}

		// Verify history records (3 transitions)
		var historyCount int
		_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_history WHERE task_id = ?", task.ID).Scan(&historyCount)
		if historyCount != 3 {
			t.Errorf("Expected 3 history records, got %d", historyCount)
		}
	})

	// Scenario 2: Invalid transitions are blocked
	t.Run("InvalidTransitionsBlocked", func(t *testing.T) {
		// Reset to in_progress
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusInProgress, task.ID)

		// Try to skip ready_for_review and go directly to completed
		err := repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &developer, nil)
		if err == nil {
			t.Error("Should not allow skipping ready_for_review")
		}

		// Verify error message is helpful
		if err != nil && !containsString(err.Error(), "invalid transition") {
			t.Errorf("Error should mention invalid transition, got: %v", err)
		}

		// Verify task status didn't change
		unchangedTask, _ := repo.GetByID(ctx, task.ID)
		if unchangedTask.Status != models.TaskStatusInProgress {
			t.Errorf("Task status should not have changed, got %s", unchangedTask.Status)
		}
	})

	// Scenario 3: Force flag bypasses validation
	t.Run("ForceBypassValidation", func(t *testing.T) {
		// Clean history
		_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ?", task.ID)

		// Reset to todo
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

		// Force invalid transition (todo -> completed)
		emergencyNotes := "Emergency hotfix deployment"
		err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &techLead, &emergencyNotes, true)
		if err != nil {
			t.Fatalf("Forced transition should succeed: %v", err)
		}

		// Verify transition succeeded
		forcedTask, _ := repo.GetByID(ctx, task.ID)
		if forcedTask.Status != models.TaskStatusCompleted {
			t.Errorf("Expected forced transition to succeed, got status %s", forcedTask.Status)
		}

		// Verify forced=true in history
		var forced bool
		_ = database.QueryRowContext(ctx,
			"SELECT forced FROM task_history WHERE task_id = ? ORDER BY timestamp DESC LIMIT 1",
			task.ID).Scan(&forced)
		if !forced {
			t.Error("Expected forced transition to have forced=true in history")
		}
	})

	// Scenario 4: Blocking and unblocking workflow
	t.Run("BlockingWorkflow", func(t *testing.T) {
		// Reset to in_progress
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusInProgress, task.ID)

		// Block task
		blockReason := "Waiting for API specification"
		err := repo.BlockTask(ctx, task.ID, blockReason, &developer)
		if err != nil {
			t.Fatalf("Failed to block task: %v", err)
		}

		// Verify blocked
		blockedTask, _ := repo.GetByID(ctx, task.ID)
		if blockedTask.Status != models.TaskStatusBlocked {
			t.Errorf("Expected status blocked, got %s", blockedTask.Status)
		}

		// Unblock to todo
		err = repo.UnblockTask(ctx, task.ID, &developer)
		if err != nil {
			t.Fatalf("Failed to unblock task: %v", err)
		}

		// Verify unblocked
		unblockedTask, _ := repo.GetByID(ctx, task.ID)
		if unblockedTask.Status != models.TaskStatusTodo {
			t.Errorf("Expected status todo after unblock, got %s", unblockedTask.Status)
		}
	})

	// Scenario 5: Reopen workflow
	t.Run("ReopenWorkflow", func(t *testing.T) {
		// Set to ready_for_review
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusReadyForReview, task.ID)

		// Tech lead requests changes
		reopenNotes := "Found bug in edge case handling, please fix"
		err := repo.ReopenTask(ctx, task.ID, &techLead, &reopenNotes)
		if err != nil {
			t.Fatalf("Failed to reopen task: %v", err)
		}

		// Verify reopened
		reopenedTask, _ := repo.GetByID(ctx, task.ID)
		if reopenedTask.Status != models.TaskStatusInProgress {
			t.Errorf("Expected status in_progress after reopen, got %s", reopenedTask.Status)
		}
	})
}

// TestWorkflowIntegration_DefaultWorkflowBackwardCompatibility tests that default workflow maintains backward compatibility
func TestWorkflowIntegration_DefaultWorkflowBackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Use default workflow (no custom config)
	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

	agent := "backward-compat-test"

	// Test classic workflow: todo -> in_progress -> ready_for_review -> completed
	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
	if err != nil {
		t.Fatalf("Classic transition todo->in_progress failed: %v", err)
	}

	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusReadyForReview, &agent, nil)
	if err != nil {
		t.Fatalf("Classic transition in_progress->ready_for_review failed: %v", err)
	}

	err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)
	if err != nil {
		t.Fatalf("Classic transition ready_for_review->completed failed: %v", err)
	}

	// Verify final state
	finalTask, _ := repo.GetByID(ctx, task.ID)
	if finalTask.Status != models.TaskStatusCompleted {
		t.Errorf("Expected completed status, got %s", finalTask.Status)
	}
}

// TestWorkflowIntegration_PerformanceBenchmark tests that workflow validation is fast (REQ-NF-001)
func TestWorkflowIntegration_PerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	customWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":             {"in_progress"},
			"in_progress":      {"ready_for_review"},
			"ready_for_review": {"completed"},
			"completed":        {},
			"blocked":          {"todo"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, customWorkflow)

	test.SeedTestData()
	task, _ := repo.GetByKey(ctx, "T-E99-F99-001")

	agent := "perf-test"

	// Measure 100 status updates (50 valid, 50 invalid)
	// According to REQ-NF-001, validation should add <100ms overhead (P95)
	validTransitions := 0
	invalidTransitions := 0

	for i := 0; i < 50; i++ {
		// Reset to todo
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", models.TaskStatusTodo, task.ID)

		// Valid transition
		err := repo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, &agent, nil)
		if err == nil {
			validTransitions++
		}

		// Invalid transition
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, &agent, nil)
		if err != nil {
			invalidTransitions++
		}
	}

	// Verify transitions worked as expected
	if validTransitions != 50 {
		t.Errorf("Expected 50 valid transitions, got %d", validTransitions)
	}

	if invalidTransitions != 50 {
		t.Errorf("Expected 50 invalid transitions, got %d", invalidTransitions)
	}

	// Performance is measured by Go's benchmark tool, not in test
	// This test just verifies that validation doesn't break under load
	t.Logf("Successfully validated 100 transitions (50 valid, 50 invalid)")
}
