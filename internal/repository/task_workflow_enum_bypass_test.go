package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestForceFlag_BypassesEnumValidation verifies that the repository layer
// no longer performs enum validation - all status values pass through.
// Workflow validation is now handled entirely by the service layer.
// This test confirms the repository accepts any status string.
func TestForceFlag_BypassesEnumValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Use a repo with NO workflow config
	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

	agent := "enum-bypass-test"

	t.Run("non-standard status accepted without force (validation removed from repo)", func(t *testing.T) {
		// Repo no longer validates status enums - service layer handles this
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)
		err := repo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, false)
		if err != nil {
			t.Errorf("Repo should accept any status after validation removal, got error: %v", err)
		}
	})

	t.Run("non-standard status accepted with force", func(t *testing.T) {
		// Reset to todo
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		// With force=true, any status string should be accepted
		err := repo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, true)
		if err != nil {
			t.Errorf("Force flag should work, got error: %v", err)
		}

		// Verify the status was actually written to the DB
		updatedTask, err := repo.GetByID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get updated task: %v", err)
		}
		if updatedTask.Status != "ready_for_refinement_tech" {
			t.Errorf("Expected status 'ready_for_refinement_tech', got '%s'", updatedTask.Status)
		}
	})

	t.Run("force accepts completely custom status", func(t *testing.T) {
		// Reset
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		// Even a totally custom status should work
		err := repo.UpdateStatusForced(ctx, task.ID, "my_custom_status", &agent, nil, nil, nil, true)
		if err != nil {
			t.Errorf("Force flag should allow any status string, got error: %v", err)
		}

		updatedTask, err := repo.GetByID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get updated task: %v", err)
		}
		if updatedTask.Status != "my_custom_status" {
			t.Errorf("Expected status 'my_custom_status', got '%s'", updatedTask.Status)
		}
	})

	// Cleanup
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)
}

// TestWorkflowAwareRepo_AcceptsAnyStatus verifies that both workflow-aware
// and default repos accept any status now that validation is in the service layer.
func TestWorkflowAwareRepo_AcceptsAnyStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create an advanced-style workflow
	advancedWorkflow := &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":                      {"in_development"},
			"in_development":            {"ready_for_code_review"},
			"ready_for_code_review":     {"in_code_review"},
			"in_code_review":            {"ready_for_refinement_tech", "changes_requested"},
			"ready_for_refinement_tech": {"in_refinement_tech"},
			"in_refinement_tech":        {"ready_for_development"},
			"ready_for_development":     {"in_development"},
			"changes_requested":         {"in_development"},
			"completed":                 {},
			"blocked":                   {"todo"},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"todo":                      {Phase: "planning"},
			"in_development":            {Phase: "development"},
			"ready_for_code_review":     {Phase: "review"},
			"in_code_review":            {Phase: "review"},
			"ready_for_refinement_tech": {Phase: "planning"},
			"in_refinement_tech":        {Phase: "planning"},
			"ready_for_development":     {Phase: "planning"},
			"changes_requested":         {Phase: "development"},
			"completed":                 {Phase: "done"},
			"blocked":                   {Phase: "any"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	// Create repos: one with workflow, one without
	workflowRepo := NewTaskRepositoryWithWorkflow(db, advancedWorkflow)
	defaultRepo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := workflowRepo.GetByKey(ctx, "T-E99-F99-002")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	agent := "workflow-test"

	t.Run("workflow repo accepts any transition (validation in service layer)", func(t *testing.T) {
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "in_code_review", task.ID)

		err := workflowRepo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, false)
		if err != nil {
			t.Errorf("Workflow-aware repo should accept any status (validation moved to service): %v", err)
		}

		updatedTask, _ := workflowRepo.GetByID(ctx, task.ID)
		if updatedTask.Status != "ready_for_refinement_tech" {
			t.Errorf("Expected status 'ready_for_refinement_tech', got '%s'", updatedTask.Status)
		}
	})

	t.Run("default repo also accepts any status (validation in service layer)", func(t *testing.T) {
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "in_code_review", task.ID)

		err := defaultRepo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, false)
		if err != nil {
			t.Errorf("Default repo should accept any status (validation moved to service): %v", err)
		}
	})

	// Cleanup
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)
}

// TestForceFlag_WithUnblock_AcceptsAnyStatus verifies that
// UpdateStatusForcedWithUnblock accepts any status now that validation is removed.
func TestForceFlag_WithUnblock_AcceptsAnyStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-003")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	t.Run("non-standard status via UpdateStatusForcedWithUnblock accepted without force", func(t *testing.T) {
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		_, err := repo.UpdateStatusForcedWithUnblock(ctx, task.ID, "ready_for_qa", nil, nil, nil, nil, false)
		if err != nil {
			t.Errorf("Repo should accept any status (validation moved to service): %v", err)
		}
	})

	t.Run("non-standard status via UpdateStatusForcedWithUnblock accepted with force", func(t *testing.T) {
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		agent := "unblock-enum-test"
		_, err := repo.UpdateStatusForcedWithUnblock(ctx, task.ID, "ready_for_qa", &agent, nil, nil, nil, true)
		if err != nil {
			t.Errorf("Force should work via WithUnblock path: %v", err)
		}

		updatedTask, _ := repo.GetByID(ctx, task.ID)
		if updatedTask.Status != "ready_for_qa" {
			t.Errorf("Expected status 'ready_for_qa', got '%s'", updatedTask.Status)
		}
	})

	// Cleanup
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)
}

// TestWorkflowPhaseBasedReopenTargets verifies that GetStatusesByPhase returns
// correct statuses for reopen logic. This tests config behavior, not repo validation.
func TestWorkflowPhaseBasedReopenTargets(t *testing.T) {
	advancedWorkflow := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"draft":                     {Phase: "planning"},
			"ready_for_refinement_ba":   {Phase: "planning"},
			"in_refinement_ba":          {Phase: "planning"},
			"ready_for_refinement_tech": {Phase: "planning"},
			"in_refinement_tech":        {Phase: "planning"},
			"ready_for_development":     {Phase: "planning"},
			"in_development":            {Phase: "development"},
			"changes_requested":         {Phase: "development"},
			"ready_for_code_review":     {Phase: "review"},
			"in_code_review":            {Phase: "review"},
			"ready_for_qa":              {Phase: "qa"},
			"in_qa":                     {Phase: "qa"},
			"ready_for_approval":        {Phase: "approval"},
			"completed":                 {Phase: "done"},
			"blocked":                   {Phase: "any"},
		},
	}

	t.Run("planning phase includes all planning statuses", func(t *testing.T) {
		planningStatuses := advancedWorkflow.GetStatusesByPhase("planning")
		expectedStatuses := map[string]bool{
			"draft":                     true,
			"ready_for_refinement_ba":   true,
			"in_refinement_ba":          true,
			"ready_for_refinement_tech": true,
			"in_refinement_tech":        true,
			"ready_for_development":     true,
		}

		for _, s := range planningStatuses {
			if !expectedStatuses[s] {
				t.Errorf("Unexpected planning status: %s", s)
			}
			delete(expectedStatuses, s)
		}

		for missing := range expectedStatuses {
			t.Errorf("Missing planning status: %s", missing)
		}
	})

	t.Run("development phase includes development statuses", func(t *testing.T) {
		devStatuses := advancedWorkflow.GetStatusesByPhase("development")
		expectedStatuses := map[string]bool{
			"in_development":    true,
			"changes_requested": true,
		}

		for _, s := range devStatuses {
			if !expectedStatuses[s] {
				t.Errorf("Unexpected development status: %s", s)
			}
			delete(expectedStatuses, s)
		}

		for missing := range expectedStatuses {
			t.Errorf("Missing development status: %s", missing)
		}
	})

	t.Run("combined reopen targets cover planning and development", func(t *testing.T) {
		reopenTargets := append(
			advancedWorkflow.GetStatusesByPhase("planning"),
			advancedWorkflow.GetStatusesByPhase("development")...,
		)

		forbiddenPhases := map[string]bool{
			"ready_for_code_review": true,
			"in_code_review":        true,
			"ready_for_qa":          true,
			"in_qa":                 true,
			"ready_for_approval":    true,
			"completed":             true,
			"blocked":               true,
		}

		for _, target := range reopenTargets {
			if forbiddenPhases[target] {
				t.Errorf("Reopen targets should not include %s (non-planning/development phase)", target)
			}
		}

		targetSet := make(map[string]bool)
		for _, t := range reopenTargets {
			targetSet[t] = true
		}

		if !targetSet["in_development"] {
			t.Error("Reopen targets should include 'in_development'")
		}
		if !targetSet["ready_for_development"] {
			t.Error("Reopen targets should include 'ready_for_development'")
		}
	})

	t.Run("basic workflow reopen targets include in_progress", func(t *testing.T) {
		basicWorkflow := &config.WorkflowConfig{
			StatusMetadata: map[string]config.StatusMetadata{
				"todo":             {Phase: "planning"},
				"in_progress":      {Phase: "development"},
				"ready_for_review": {Phase: "review"},
				"completed":        {Phase: "done"},
				"blocked":          {Phase: "any"},
			},
		}

		reopenTargets := append(
			basicWorkflow.GetStatusesByPhase("planning"),
			basicWorkflow.GetStatusesByPhase("development")...,
		)

		targetSet := make(map[string]bool)
		for _, t := range reopenTargets {
			targetSet[t] = true
		}

		if !targetSet["in_progress"] {
			t.Error("Basic workflow reopen targets should include 'in_progress'")
		}
		if !targetSet["todo"] {
			t.Error("Basic workflow reopen targets should include 'todo'")
		}
	})
}
