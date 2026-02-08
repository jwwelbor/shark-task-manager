package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestForceFlag_BypassesEnumValidation verifies that --force skips isValidStatusEnum,
// allowing advanced workflow statuses even when repo has no workflow config.
// This is a regression test for the bug where force=true still failed on
// statuses not in the hardcoded enum list (e.g., "ready_for_refinement_tech").
func TestForceFlag_BypassesEnumValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Use a repo with NO workflow config — simulates the old bug scenario
	// where commands didn't load workflow config from .sharkconfig.json.
	// Without workflow, isValidStatusEnum falls back to the hardcoded 6-status list.
	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to todo
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

	agent := "enum-bypass-test"

	t.Run("non-standard status rejected without force", func(t *testing.T) {
		// "ready_for_refinement_tech" is NOT in the default hardcoded enum
		err := repo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, false)
		if err == nil {
			t.Error("Expected error for non-standard status without force, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "invalid status") {
			t.Errorf("Expected 'invalid status' error, got: %v", err)
		}
	})

	t.Run("non-standard status accepted with force", func(t *testing.T) {
		// Reset to todo
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		// With force=true, any status string should be accepted
		err := repo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, true)
		if err != nil {
			t.Errorf("Force flag should bypass enum validation, got error: %v", err)
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

	t.Run("force bypasses enum for completely custom status", func(t *testing.T) {
		// Reset
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		// Even a totally custom status should work with force
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

// TestWorkflowAwareRepo_ValidatesAgainstConfiguredWorkflow verifies that a repo
// created with NewTaskRepositoryWithWorkflow validates statuses against the
// provided workflow config rather than the hardcoded fallback.
// This is a regression test for commands that used NewTaskRepository instead of
// NewTaskRepositoryWithWorkflow, causing advanced statuses to be rejected.
func TestWorkflowAwareRepo_ValidatesAgainstConfiguredWorkflow(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Create an advanced-style workflow with custom statuses
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

	t.Run("workflow repo accepts advanced status on valid transition", func(t *testing.T) {
		// Set task to in_code_review (a status that allows transition to ready_for_refinement_tech)
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "in_code_review", task.ID)

		// This is a backward transition (review -> planning), so rejection reason is required
		rejectionReason := "Needs further technical refinement"
		err := workflowRepo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, &rejectionReason, nil, false)
		if err != nil {
			t.Errorf("Workflow-aware repo should accept 'ready_for_refinement_tech' as valid: %v", err)
		}

		updatedTask, _ := workflowRepo.GetByID(ctx, task.ID)
		if updatedTask.Status != "ready_for_refinement_tech" {
			t.Errorf("Expected status 'ready_for_refinement_tech', got '%s'", updatedTask.Status)
		}
	})

	t.Run("default repo rejects advanced status without force", func(t *testing.T) {
		// Reset
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "in_code_review", task.ID)

		// Default repo doesn't know about advanced statuses
		err := defaultRepo.UpdateStatusForced(ctx, task.ID, "ready_for_refinement_tech", &agent, nil, nil, nil, false)
		if err == nil {
			t.Error("Default repo should reject 'ready_for_refinement_tech' as unknown status")
		}
	})

	t.Run("workflow repo rejects invalid transition", func(t *testing.T) {
		// Set to todo — cannot go directly to completed in this workflow
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		err := workflowRepo.UpdateStatusForced(ctx, task.ID, "completed", &agent, nil, nil, nil, false)
		if err == nil {
			t.Error("Workflow repo should reject invalid transition todo->completed")
		}
	})

	// Cleanup
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)
}

// TestForceFlag_WithUnblock_BypassesEnumValidation verifies the force flag
// bypass works through the UpdateStatusForcedWithUnblock path (used by set-status command).
func TestForceFlag_WithUnblock_BypassesEnumValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Repo without workflow (simulates missing workflow loading)
	repo := NewTaskRepository(db)

	test.SeedTestData()
	task, err := repo.GetByKey(ctx, "T-E99-F99-003")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	agent := "unblock-enum-test"

	t.Run("non-standard status via UpdateStatusForcedWithUnblock rejected without force", func(t *testing.T) {
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		_, err := repo.UpdateStatusForcedWithUnblock(ctx, task.ID, "ready_for_qa", nil, nil, nil, nil, false)
		if err == nil {
			t.Error("Expected error for non-standard status without force")
		}
	})

	t.Run("non-standard status via UpdateStatusForcedWithUnblock accepted with force", func(t *testing.T) {
		_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)

		_, err := repo.UpdateStatusForcedWithUnblock(ctx, task.ID, "ready_for_qa", &agent, nil, nil, nil, true)
		if err != nil {
			t.Errorf("Force should bypass enum validation via WithUnblock path: %v", err)
		}

		updatedTask, _ := repo.GetByID(ctx, task.ID)
		if updatedTask.Status != "ready_for_qa" {
			t.Errorf("Expected status 'ready_for_qa', got '%s'", updatedTask.Status)
		}
	})

	// Cleanup
	_, _ = database.ExecContext(ctx, "UPDATE tasks SET status = ? WHERE id = ?", "todo", task.ID)
}

// TestWorkflowPhaseBasedReopenTargets verifies that reopening uses phase-based
// targets from workflow config rather than a hardcoded list.
// This is a regression test for the runTaskReopen command that used to hardcode
// a list of reopen target statuses.
func TestWorkflowPhaseBasedReopenTargets(t *testing.T) {
	// Test that GetStatusesByPhase returns correct statuses for reopen logic
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

		// Verify reopen targets do NOT include review, qa, approval, done, or any phases
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

		// Verify it includes at least the key development statuses
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

// TestIsValidStatusEnum_WithWorkflow verifies that isValidStatusEnum checks
// against workflow StatusFlow keys when a workflow is configured.
func TestIsValidStatusEnum_WithWorkflow(t *testing.T) {
	database := test.GetTestDB()
	db := NewDB(database)

	advancedWorkflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"todo":                      {"in_development"},
			"in_development":            {"ready_for_code_review"},
			"ready_for_code_review":     {"in_code_review"},
			"in_code_review":            {"completed"},
			"ready_for_refinement_tech": {"in_refinement_tech"},
			"in_refinement_tech":        {"ready_for_development"},
			"ready_for_development":     {"in_development"},
			"completed":                 {},
			"blocked":                   {"todo"},
		},
	}

	repo := NewTaskRepositoryWithWorkflow(db, advancedWorkflow)

	t.Run("advanced status is valid with matching workflow", func(t *testing.T) {
		if !repo.isValidStatusEnum("ready_for_refinement_tech") {
			t.Error("'ready_for_refinement_tech' should be valid when present in workflow StatusFlow")
		}
	})

	t.Run("unknown status is invalid", func(t *testing.T) {
		if repo.isValidStatusEnum("nonexistent_status") {
			t.Error("'nonexistent_status' should be invalid when not in workflow StatusFlow")
		}
	})

	t.Run("default repo only accepts basic statuses", func(t *testing.T) {
		defaultRepo := NewTaskRepository(db)

		if !defaultRepo.isValidStatusEnum(models.TaskStatus("todo")) {
			t.Error("'todo' should be valid in default repo")
		}
		if !defaultRepo.isValidStatusEnum(models.TaskStatus("in_progress")) {
			t.Error("'in_progress' should be valid in default repo")
		}
		if defaultRepo.isValidStatusEnum("ready_for_refinement_tech") {
			t.Error("'ready_for_refinement_tech' should be invalid in default repo (no workflow config)")
		}
	})
}
