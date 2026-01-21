package commands

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskLinkCommand tests the task link command functionality
func TestTaskLinkCommand(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	relRepo := repository.NewTaskRelationshipRepository(db)

	test.SeedTestData()

	// Clean up relationships
	_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships")

	// Create test epic
	testEpicKey := "E88"
	epic := &models.Epic{
		Key:           testEpicKey,
		Title:         "Task Relationship Test Epic",
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: ptrPriority(models.PriorityHigh),
	}

	// Skip if epic already exists
	existingEpic, _ := epicRepo.GetByKey(ctx, testEpicKey)
	if existingEpic == nil {
		if err := epicRepo.Create(ctx, epic); err != nil {
			t.Fatalf("Failed to create test epic: %v", err)
		}
	} else {
		epic = existingEpic
	}

	// Create test feature
	testFeatureKey := fmt.Sprintf("%s-F01", testEpicKey)
	execOrder := 1
	feature := &models.Feature{
		Key:            testFeatureKey,
		EpicID:         epic.ID,
		Title:          "Task Relationship Test Feature",
		Status:         models.FeatureStatusDraft,
		ExecutionOrder: &execOrder,
	}

	existingFeature, _ := featureRepo.GetByKey(ctx, testFeatureKey)
	if existingFeature == nil {
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("Failed to create test feature: %v", err)
		}
	} else {
		feature = existingFeature
	}

	// Create test tasks
	task1Key := fmt.Sprintf("T-%s-001", testFeatureKey)
	task2Key := fmt.Sprintf("T-%s-002", testFeatureKey)
	task3Key := fmt.Sprintf("T-%s-003", testFeatureKey)

	tasks := []struct {
		key   string
		title string
	}{
		{task1Key, "First Test Task"},
		{task2Key, "Second Test Task"},
		{task3Key, "Third Test Task"},
	}

	var task1, task2, task3 *models.Task

	for i, taskData := range tasks {
		agentType := "general"
		existing, _ := taskRepo.GetByKey(ctx, taskData.key)

		if existing == nil {
			task := &models.Task{
				Key:       taskData.key,
				FeatureID: feature.ID,
				Title:     taskData.title,
				Status:    models.TaskStatusTodo,
				AgentType: &agentType,
				Priority:  i + 1,
			}
			if err := taskRepo.Create(ctx, task); err != nil {
				t.Fatalf("Failed to create task %s: %v", taskData.key, err)
			}
			existing, _ = taskRepo.GetByKey(ctx, taskData.key)
		}

		switch i {
		case 0:
			task1 = existing
		case 1:
			task2 = existing
		case 2:
			task3 = existing
		}
	}

	// Test: Create depends_on relationship
	t.Run("CreateDependsOnRelationship", func(t *testing.T) {
		// Clean up first
		_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships WHERE from_task_id = ? OR to_task_id = ?", task1.ID, task1.ID)

		rel := &models.TaskRelationship{
			FromTaskID:       task1.ID,
			ToTaskID:         task2.ID,
			RelationshipType: models.RelationshipDependsOn,
		}

		err := relRepo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Failed to create depends_on relationship: %v", err)
		}

		// Verify relationship was created
		rels, err := relRepo.GetOutgoing(ctx, task1.ID, []string{"depends_on"})
		if err != nil {
			t.Fatalf("Failed to get outgoing relationships: %v", err)
		}

		if len(rels) != 1 {
			t.Errorf("Expected 1 depends_on relationship, got %d", len(rels))
		}

		if len(rels) > 0 && rels[0].ToTaskID != task2.ID {
			t.Errorf("Expected relationship to task %d, got %d", task2.ID, rels[0].ToTaskID)
		}
	})

	// Test: Create blocks relationship
	t.Run("CreateBlocksRelationship", func(t *testing.T) {
		// Clean up first
		_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships WHERE from_task_id = ? AND to_task_id = ? AND relationship_type = 'blocks'", task2.ID, task3.ID)

		rel := &models.TaskRelationship{
			FromTaskID:       task2.ID,
			ToTaskID:         task3.ID,
			RelationshipType: models.RelationshipBlocks,
		}

		err := relRepo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Failed to create blocks relationship: %v", err)
		}

		// Verify relationship was created
		rels, err := relRepo.GetOutgoing(ctx, task2.ID, []string{"blocks"})
		if err != nil {
			t.Fatalf("Failed to get outgoing blocks relationships: %v", err)
		}

		foundBlocks := false
		for _, r := range rels {
			if r.ToTaskID == task3.ID {
				foundBlocks = true
				break
			}
		}

		if !foundBlocks {
			t.Error("Expected to find blocks relationship to task3")
		}
	})

	// Test: Create related_to relationship
	t.Run("CreateRelatedToRelationship", func(t *testing.T) {
		// Clean up first
		_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships WHERE from_task_id = ? AND to_task_id = ? AND relationship_type = 'related_to'", task1.ID, task3.ID)

		rel := &models.TaskRelationship{
			FromTaskID:       task1.ID,
			ToTaskID:         task3.ID,
			RelationshipType: models.RelationshipRelatedTo,
		}

		err := relRepo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Failed to create related_to relationship: %v", err)
		}

		// Verify relationship was created
		rels, err := relRepo.GetOutgoing(ctx, task1.ID, []string{"related_to"})
		if err != nil {
			t.Fatalf("Failed to get outgoing related_to relationships: %v", err)
		}

		foundRelated := false
		for _, r := range rels {
			if r.ToTaskID == task3.ID {
				foundRelated = true
				break
			}
		}

		if !foundRelated {
			t.Error("Expected to find related_to relationship to task3")
		}
	})

	// Test: Get all relationships for a task
	t.Run("GetAllRelationshipsForTask", func(t *testing.T) {
		// task1 should have relationships to task2 (depends_on) and task3 (related_to)
		rels, err := relRepo.GetByTaskID(ctx, task1.ID)
		if err != nil {
			t.Fatalf("Failed to get all relationships for task1: %v", err)
		}

		// Should have at least 2 outgoing relationships
		if len(rels) < 2 {
			t.Errorf("Expected at least 2 relationships for task1, got %d", len(rels))
		}

		// Verify types
		hasDepends := false
		hasRelated := false
		for _, r := range rels {
			if r.FromTaskID == task1.ID {
				if r.RelationshipType == models.RelationshipDependsOn {
					hasDepends = true
				}
				if r.RelationshipType == models.RelationshipRelatedTo {
					hasRelated = true
				}
			}
		}

		if !hasDepends {
			t.Error("Expected to find depends_on relationship from task1")
		}
		if !hasRelated {
			t.Error("Expected to find related_to relationship from task1")
		}
	})

	// Test: Get incoming relationships (what depends on this task)
	t.Run("GetIncomingRelationships", func(t *testing.T) {
		// task2 should have incoming depends_on from task1
		rels, err := relRepo.GetIncoming(ctx, task2.ID, []string{"depends_on"})
		if err != nil {
			t.Fatalf("Failed to get incoming relationships for task2: %v", err)
		}

		found := false
		for _, r := range rels {
			if r.FromTaskID == task1.ID && r.ToTaskID == task2.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find incoming depends_on relationship from task1 to task2")
		}
	})

	// Test: Get outgoing relationships (what this task depends on)
	t.Run("GetOutgoingRelationships", func(t *testing.T) {
		// task1 should have outgoing depends_on to task2
		rels, err := relRepo.GetOutgoing(ctx, task1.ID, []string{"depends_on"})
		if err != nil {
			t.Fatalf("Failed to get outgoing relationships for task1: %v", err)
		}

		found := false
		for _, r := range rels {
			if r.FromTaskID == task1.ID && r.ToTaskID == task2.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find outgoing depends_on relationship from task1 to task2")
		}
	})

	// Test: Delete relationship
	t.Run("DeleteRelationship", func(t *testing.T) {
		// Create a temporary relationship to delete
		tempRel := &models.TaskRelationship{
			FromTaskID:       task1.ID,
			ToTaskID:         task3.ID,
			RelationshipType: models.RelationshipFollows,
		}
		err := relRepo.Create(ctx, tempRel)
		if err != nil {
			t.Fatalf("Failed to create temporary relationship: %v", err)
		}

		// Delete by tasks and type
		err = relRepo.DeleteByTasksAndType(ctx, task1.ID, task3.ID, "follows")
		if err != nil {
			t.Fatalf("Failed to delete relationship: %v", err)
		}

		// Verify deletion
		rels, err := relRepo.GetOutgoing(ctx, task1.ID, []string{"follows"})
		if err != nil {
			t.Fatalf("Failed to get relationships after deletion: %v", err)
		}

		for _, r := range rels {
			if r.ToTaskID == task3.ID {
				t.Error("Expected relationship to be deleted, but it still exists")
			}
		}
	})

	// Test: Prevent duplicate relationships
	t.Run("PreventDuplicateRelationships", func(t *testing.T) {
		// Try to create duplicate depends_on relationship
		dupRel := &models.TaskRelationship{
			FromTaskID:       task1.ID,
			ToTaskID:         task2.ID,
			RelationshipType: models.RelationshipDependsOn,
		}

		err := relRepo.Create(ctx, dupRel)
		if err == nil {
			t.Error("Expected error when creating duplicate relationship, got nil")
		}

		if err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' in error message, got: %v", err)
		}
	})
}

// TestTaskRelationshipTypes tests all relationship types
func TestTaskRelationshipTypes(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	relRepo := repository.NewTaskRelationshipRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Clean up before test - use unique task keys with E99
	_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E99-F99-040', 'T-E99-F99-041')")

	// Seed epic and feature (E99, E99-F99)
	_, featureID := test.SeedTestData()

	// Create two test tasks
	task1 := &models.Task{
		Key:       "T-E99-F99-040",
		Title:     "Test Task 1",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err := taskRepo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Failed to create test task 1: %v", err)
	}

	task2 := &models.Task{
		Key:       "T-E99-F99-041",
		Title:     "Test Task 2",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err = taskRepo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Failed to create test task 2: %v", err)
	}

	task1ID := task1.ID
	task2ID := task2.ID

	// Test all relationship types
	relationshipTypes := []models.RelationshipType{
		models.RelationshipDependsOn,
		models.RelationshipBlocks,
		models.RelationshipRelatedTo,
		models.RelationshipFollows,
		models.RelationshipSpawnedFrom,
		models.RelationshipDuplicates,
		models.RelationshipReferences,
	}

	for _, relType := range relationshipTypes {
		t.Run(fmt.Sprintf("Create_%s_Relationship", relType), func(t *testing.T) {
			rel := &models.TaskRelationship{
				FromTaskID:       task1ID,
				ToTaskID:         task2ID,
				RelationshipType: relType,
			}

			err := relRepo.Create(ctx, rel)
			if err != nil {
				t.Errorf("Failed to create %s relationship: %v", relType, err)
			}

			// Verify it was created
			rels, err := relRepo.GetOutgoing(ctx, task1ID, []string{string(relType)})
			if err != nil {
				t.Errorf("Failed to get %s relationships: %v", relType, err)
			}

			found := false
			for _, r := range rels {
				if r.RelationshipType == relType && r.ToTaskID == task2ID {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected to find %s relationship, but it wasn't found", relType)
			}
		})
	}
}

// TestTaskRelationshipValidation tests validation rules
func TestTaskRelationshipValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	relRepo := repository.NewTaskRelationshipRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Clean up before test - use unique task keys with E99
	_, _ = database.ExecContext(ctx, "DELETE FROM task_relationships")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E99-F99-050', 'T-E99-F99-051')")

	// Seed epic and feature (E99, E99-F99)
	_, featureID := test.SeedTestData()

	// Create two test tasks
	task1 := &models.Task{
		Key:       "T-E99-F99-050",
		Title:     "Test Task 1",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err := taskRepo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Failed to create test task 1: %v", err)
	}

	task2 := &models.Task{
		Key:       "T-E99-F99-051",
		Title:     "Test Task 2",
		Status:    "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err = taskRepo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Failed to create test task 2: %v", err)
	}

	task1ID := task1.ID
	task2ID := task2.ID

	// Test: Invalid from_task_id
	t.Run("InvalidFromTaskID", func(t *testing.T) {
		rel := &models.TaskRelationship{
			FromTaskID:       0,
			ToTaskID:         task2ID,
			RelationshipType: models.RelationshipDependsOn,
		}

		err := relRepo.Create(ctx, rel)
		if err == nil {
			t.Error("Expected validation error for invalid from_task_id, got nil")
		}
	})

	// Test: Invalid to_task_id
	t.Run("InvalidToTaskID", func(t *testing.T) {
		rel := &models.TaskRelationship{
			FromTaskID:       task1ID,
			ToTaskID:         0,
			RelationshipType: models.RelationshipDependsOn,
		}

		err := relRepo.Create(ctx, rel)
		if err == nil {
			t.Error("Expected validation error for invalid to_task_id, got nil")
		}
	})

	// Test: Self-relationship
	t.Run("SelfRelationship", func(t *testing.T) {
		rel := &models.TaskRelationship{
			FromTaskID:       task1ID,
			ToTaskID:         task1ID,
			RelationshipType: models.RelationshipDependsOn,
		}

		err := relRepo.Create(ctx, rel)
		if err == nil {
			t.Error("Expected validation error for self-relationship, got nil")
		}
	})

	// Test: Invalid relationship type
	t.Run("InvalidRelationshipType", func(t *testing.T) {
		rel := &models.TaskRelationship{
			FromTaskID:       task1ID,
			ToTaskID:         task2ID,
			RelationshipType: "invalid_type",
		}

		err := relRepo.Create(ctx, rel)
		if err == nil {
			t.Error("Expected validation error for invalid relationship type, got nil")
		}
	})
}
