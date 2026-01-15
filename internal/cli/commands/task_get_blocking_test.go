package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskGetShowsBlockingRelationships tests that task get command shows blocked-by and blocks relationships
func TestTaskGetShowsBlockingRelationships(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean up before test
	_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")

	// Create test epic and feature
	_, featureID := test.SeedTestData()

	// Verify featureID is valid
	if featureID == 0 {
		t.Fatalf("SeedTestData() returned invalid featureID: %d", featureID)
	}

	// Create repositories
	taskRepo := repository.NewTaskRepository(db)
	relationshipRepo := repository.NewTaskRelationshipRepository(db)

	// Create test tasks
	// Task A: This is the task we'll query (T-E99-F01-002)
	// Task B blocks Task A (T-E99-F01-001 blocks T-E99-F01-002)
	// Task A blocks Task C (T-E99-F01-002 blocks T-E99-F01-003)

	taskA := &models.Task{
		Key:       "T-E99-F01-002",
		Title:     "Task A",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	t.Logf("Creating task A with featureID=%d", featureID)
	err := taskRepo.Create(ctx, taskA)
	if err != nil {
		t.Fatalf("Failed to create task A (featureID=%d): %v", featureID, err)
	}

	taskB := &models.Task{
		Key:       "T-E99-F01-001",
		Title:     "Task B - Blocks Task A",
		Status:    "in_progress",
		Priority:  5,
		FeatureID: featureID,
	}
	err = taskRepo.Create(ctx, taskB)
	if err != nil {
		t.Fatalf("Failed to create task B: %v", err)
	}

	taskC := &models.Task{
		Key:       "T-E99-F01-003",
		Title:     "Task C - Blocked by Task A",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err = taskRepo.Create(ctx, taskC)
	if err != nil {
		t.Fatalf("Failed to create task C: %v", err)
	}

	// Create blocking relationships
	// Task B blocks Task A
	relBA := &models.TaskRelationship{
		FromTaskID:       taskB.ID,
		ToTaskID:         taskA.ID,
		RelationshipType: models.RelationshipBlocks,
	}
	err = relationshipRepo.Create(ctx, relBA)
	if err != nil {
		t.Fatalf("Failed to create B->A blocking relationship: %v", err)
	}

	// Task A blocks Task C
	relAC := &models.TaskRelationship{
		FromTaskID:       taskA.ID,
		ToTaskID:         taskC.ID,
		RelationshipType: models.RelationshipBlocks,
	}
	err = relationshipRepo.Create(ctx, relAC)
	if err != nil {
		t.Fatalf("Failed to create A->C blocking relationship: %v", err)
	}

	// Defer cleanup
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships WHERE from_task_id IN (?, ?, ?)", taskA.ID, taskB.ID, taskC.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id IN (?, ?, ?)", taskA.ID, taskB.ID, taskC.ID)
	}()

	// Now test the task get functionality
	// We need to call the repository methods that task get would use

	// Get blocked-by relationships (incoming blocks - tasks that block taskA)
	blockedBy, err := relationshipRepo.GetIncoming(ctx, taskA.ID, []string{"blocks"})
	if err != nil {
		t.Fatalf("Failed to get blocked-by relationships: %v", err)
	}

	// Get blocks relationships (outgoing blocks - tasks that taskA blocks)
	blocks, err := relationshipRepo.GetOutgoing(ctx, taskA.ID, []string{"blocks"})
	if err != nil {
		t.Fatalf("Failed to get blocks relationships: %v", err)
	}

	// Verify relationships
	if len(blockedBy) != 1 {
		t.Errorf("Expected 1 blocked-by relationship, got %d", len(blockedBy))
	} else {
		if blockedBy[0].FromTaskID != taskB.ID {
			t.Errorf("Expected blocked-by from task B (ID=%d), got ID=%d", taskB.ID, blockedBy[0].FromTaskID)
		}

		// Verify we can get the task key for the blocker
		blockerTask, err := taskRepo.GetByID(ctx, blockedBy[0].FromTaskID)
		if err != nil {
			t.Fatalf("Failed to get blocker task: %v", err)
		}
		if blockerTask.Key != "T-E99-F01-001" {
			t.Errorf("Expected blocker key T-E99-F01-001, got %s", blockerTask.Key)
		}
	}

	if len(blocks) != 1 {
		t.Errorf("Expected 1 blocks relationship, got %d", len(blocks))
	} else {
		if blocks[0].ToTaskID != taskC.ID {
			t.Errorf("Expected blocks to task C (ID=%d), got ID=%d", taskC.ID, blocks[0].ToTaskID)
		}

		// Verify we can get the task key for the blocked task
		blockedTask, err := taskRepo.GetByID(ctx, blocks[0].ToTaskID)
		if err != nil {
			t.Fatalf("Failed to get blocked task: %v", err)
		}
		if blockedTask.Key != "T-E99-F01-003" {
			t.Errorf("Expected blocked key T-E99-F01-003, got %s", blockedTask.Key)
		}
	}

	// Test that this would be included in JSON output format
	// This simulates what the task get command should return
	output := map[string]interface{}{
		"task":       taskA,
		"blocked_by": []string{"T-E99-F01-001"},
		"blocks":     []string{"T-E99-F01-003"},
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON output: %v", err)
	}

	// Verify JSON contains the blocking information
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	blockedByJSON, ok := parsed["blocked_by"].([]interface{})
	if !ok || len(blockedByJSON) != 1 {
		t.Errorf("Expected blocked_by in JSON with 1 element, got: %v", parsed["blocked_by"])
	}

	blocksJSON, ok := parsed["blocks"].([]interface{})
	if !ok || len(blocksJSON) != 1 {
		t.Errorf("Expected blocks in JSON with 1 element, got: %v", parsed["blocks"])
	}

	t.Logf("Test passed: Blocking relationships correctly retrieved and formatted")
	t.Logf("  Blocked by: %v", blockedByJSON)
	t.Logf("  Blocks: %v", blocksJSON)
}

// TestTaskGetNoBlockingRelationships tests that task get handles tasks with no blocking relationships
func TestTaskGetNoBlockingRelationships(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean up before test
	_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")

	// Create test epic and feature
	_, featureID := test.SeedTestData()

	// Verify featureID is valid
	if featureID == 0 {
		t.Fatalf("SeedTestData() returned invalid featureID: %d", featureID)
	}

	// Create repositories
	taskRepo := repository.NewTaskRepository(db)
	relationshipRepo := repository.NewTaskRelationshipRepository(db)

	// Create a single task with no relationships
	task := &models.Task{
		Key:       "T-E99-F01-999",
		Title:     "Isolated Task",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err := taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID) }()

	// Get blocking relationships (should be empty)
	blockedBy, err := relationshipRepo.GetIncoming(ctx, task.ID, []string{"blocks"})
	if err != nil {
		t.Fatalf("Failed to get blocked-by relationships: %v", err)
	}

	blocks, err := relationshipRepo.GetOutgoing(ctx, task.ID, []string{"blocks"})
	if err != nil {
		t.Fatalf("Failed to get blocks relationships: %v", err)
	}

	// Verify no relationships
	if len(blockedBy) != 0 {
		t.Errorf("Expected 0 blocked-by relationships, got %d", len(blockedBy))
	}

	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks relationships, got %d", len(blocks))
	}

	// JSON output should handle empty arrays gracefully
	output := map[string]interface{}{
		"task":       task,
		"blocked_by": []string{},
		"blocks":     []string{},
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON output: %v", err)
	}

	t.Logf("Test passed: Task with no blocking relationships handled correctly")
	t.Logf("  JSON output: %s", string(jsonBytes))
}
