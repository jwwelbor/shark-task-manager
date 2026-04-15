package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTaskLinkCommand tests the task link command functionality via entity_relationships
func TestTaskLinkCommand(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	entityRelRepo := repository.NewEntityRelationshipRepository(db)

	test.SeedTestData()

	// Clean up relationships before test
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE from_entity_type = 'task' OR to_entity_type = 'task'")

	// Create test epic
	testEpicKey := "E88"
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: testEpicKey,
		Title: "Task Relationship Test Epic"}, Status: models.EpicStatusActive,
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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: testFeatureKey,
		Title: "Task Relationship Test Feature"}, EpicID: epic.ID,
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
			task := &models.Task{BaseEntity: models.BaseEntity{Key: taskData.key,
				Title: taskData.title}, FeatureID: feature.ID,
				Status:    models.TaskStatus("todo"),
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
		_, _ = database.ExecContext(ctx,
			"DELETE FROM entity_relationships WHERE from_entity_type = 'task' AND (from_entity_id = ? OR to_entity_id = ?)",
			task1.ID, task1.ID)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     task1.ID,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       task2.ID,
			RelationshipType: models.EntityRelationshipType("depends_on"),
		}

		err := entityRelRepo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Failed to create depends_on relationship: %v", err)
		}

		// Verify relationship was created
		rels, err := entityRelRepo.GetOutgoing(ctx, models.EntityTypeTask, task1.ID, []models.EntityRelationshipType{"depends_on"})
		if err != nil {
			t.Fatalf("Failed to get outgoing relationships: %v", err)
		}

		if len(rels) != 1 {
			t.Errorf("Expected 1 depends_on relationship, got %d", len(rels))
		}

		if len(rels) > 0 && rels[0].ToEntityID != task2.ID {
			t.Errorf("Expected relationship to task %d, got %d", task2.ID, rels[0].ToEntityID)
		}
	})

	// Test: Create blocks relationship
	t.Run("CreateBlocksRelationship", func(t *testing.T) {
		// Clean up first
		_, _ = database.ExecContext(ctx,
			"DELETE FROM entity_relationships WHERE from_entity_type = 'task' AND from_entity_id = ? AND to_entity_id = ? AND relationship_type = 'blocks'",
			task2.ID, task3.ID)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     task2.ID,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       task3.ID,
			RelationshipType: models.EntityRelationshipType("blocks"),
		}

		err := entityRelRepo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Failed to create blocks relationship: %v", err)
		}

		// Verify relationship was created
		rels, err := entityRelRepo.GetOutgoing(ctx, models.EntityTypeTask, task2.ID, []models.EntityRelationshipType{"blocks"})
		if err != nil {
			t.Fatalf("Failed to get outgoing blocks relationships: %v", err)
		}

		foundBlocks := false
		for _, r := range rels {
			if r.ToEntityID == task3.ID {
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
		_, _ = database.ExecContext(ctx,
			"DELETE FROM entity_relationships WHERE from_entity_type = 'task' AND from_entity_id = ? AND to_entity_id = ? AND relationship_type = 'related_to'",
			task1.ID, task3.ID)

		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     task1.ID,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       task3.ID,
			RelationshipType: models.EntityRelationshipType("related_to"),
		}

		err := entityRelRepo.Create(ctx, rel)
		if err != nil {
			t.Fatalf("Failed to create related_to relationship: %v", err)
		}

		// Verify relationship was created
		rels, err := entityRelRepo.GetOutgoing(ctx, models.EntityTypeTask, task1.ID, []models.EntityRelationshipType{"related_to"})
		if err != nil {
			t.Fatalf("Failed to get outgoing related_to relationships: %v", err)
		}

		foundRelated := false
		for _, r := range rels {
			if r.ToEntityID == task3.ID {
				foundRelated = true
				break
			}
		}

		if !foundRelated {
			t.Error("Expected to find related_to relationship to task3")
		}
	})

	// Test: Get incoming relationships (what depends on this task)
	t.Run("GetIncomingRelationships", func(t *testing.T) {
		// task2 should have incoming depends_on from task1
		rels, err := entityRelRepo.GetIncoming(ctx, models.EntityTypeTask, task2.ID, []models.EntityRelationshipType{"depends_on"})
		if err != nil {
			t.Fatalf("Failed to get incoming relationships for task2: %v", err)
		}

		found := false
		for _, r := range rels {
			if r.FromEntityID == task1.ID && r.ToEntityID == task2.ID {
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
		rels, err := entityRelRepo.GetOutgoing(ctx, models.EntityTypeTask, task1.ID, []models.EntityRelationshipType{"depends_on"})
		if err != nil {
			t.Fatalf("Failed to get outgoing relationships for task1: %v", err)
		}

		found := false
		for _, r := range rels {
			if r.FromEntityID == task1.ID && r.ToEntityID == task2.ID {
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
		tempRel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     task1.ID,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       task3.ID,
			RelationshipType: models.EntityRelationshipType("follows"),
		}
		err := entityRelRepo.Create(ctx, tempRel)
		if err != nil {
			t.Fatalf("Failed to create temporary relationship: %v", err)
		}

		// Delete it directly using SQL (DeleteByEntityAndType)
		_, err = database.ExecContext(ctx,
			"DELETE FROM entity_relationships WHERE from_entity_type = 'task' AND from_entity_id = ? AND to_entity_type = 'task' AND to_entity_id = ? AND relationship_type = 'follows'",
			task1.ID, task3.ID)
		if err != nil {
			t.Fatalf("Failed to delete relationship: %v", err)
		}

		// Verify deletion
		rels, err := entityRelRepo.GetOutgoing(ctx, models.EntityTypeTask, task1.ID, []models.EntityRelationshipType{"follows"})
		if err != nil {
			t.Fatalf("Failed to get relationships after deletion: %v", err)
		}

		for _, r := range rels {
			if r.ToEntityID == task3.ID {
				t.Error("Expected relationship to be deleted, but it still exists")
			}
		}
	})
}

// TestTaskRelationshipTypes tests all relationship types via entity_relationships
func TestTaskRelationshipTypes(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	entityRelRepo := repository.NewEntityRelationshipRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Clean up before test - use unique task keys with E99
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE from_entity_type = 'task' OR to_entity_type = 'task'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E99-F99-040', 'T-E99-F99-041')")

	// Seed epic and feature (E99, E99-F99)
	_, featureID := test.SeedTestData()

	// Create two test tasks
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F99-040",
		Title: "Test Task 1"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err := taskRepo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Failed to create test task 1: %v", err)
	}

	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F99-041",
		Title: "Test Task 2"}, Status: "todo",
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
	relationshipTypes := []models.EntityRelationshipType{
		"depends_on",
		"blocks",
		"related_to",
		"follows",
		"spawned_from",
		"duplicates",
		"references",
	}

	for _, relType := range relationshipTypes {
		t.Run(fmt.Sprintf("Create_%s_Relationship", relType), func(t *testing.T) {
			rel := &models.EntityRelationship{
				FromEntityType:   models.EntityTypeTask,
				FromEntityID:     task1ID,
				ToEntityType:     models.EntityTypeTask,
				ToEntityID:       task2ID,
				RelationshipType: relType,
			}

			err := entityRelRepo.Create(ctx, rel)
			if err != nil {
				t.Errorf("Failed to create %s relationship: %v", relType, err)
			}

			// Verify it was created
			rels, err := entityRelRepo.GetOutgoing(ctx, models.EntityTypeTask, task1ID, []models.EntityRelationshipType{relType})
			if err != nil {
				t.Errorf("Failed to get %s relationships: %v", relType, err)
			}

			found := false
			for _, r := range rels {
				if r.RelationshipType == relType && r.ToEntityID == task2ID {
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

// TestTaskRelationshipValidation tests validation rules via entity_relationships
func TestTaskRelationshipValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)
	entityRelRepo := repository.NewEntityRelationshipRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Clean up before test - use unique task keys with E99
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_relationships WHERE from_entity_type = 'task' OR to_entity_type = 'task'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E99-F99-050', 'T-E99-F99-051')")

	// Seed epic and feature (E99, E99-F99)
	_, featureID := test.SeedTestData()

	// Create two test tasks
	task1 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F99-050",
		Title: "Test Task 1"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err := taskRepo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Failed to create test task 1: %v", err)
	}

	task2 := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E99-F99-051",
		Title: "Test Task 2"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}
	err = taskRepo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Failed to create test task 2: %v", err)
	}

	task1ID := task1.ID
	task2ID := task2.ID

	// Test: Invalid from_entity_id (zero)
	t.Run("InvalidFromEntityID", func(t *testing.T) {
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     0,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       task2ID,
			RelationshipType: models.EntityRelationshipType("depends_on"),
		}

		err := entityRelRepo.Create(ctx, rel)
		if err == nil {
			t.Error("Expected validation error for invalid from_entity_id, got nil")
		}
	})

	// Test: Invalid to_entity_id (zero)
	t.Run("InvalidToEntityID", func(t *testing.T) {
		rel := &models.EntityRelationship{
			FromEntityType:   models.EntityTypeTask,
			FromEntityID:     task1ID,
			ToEntityType:     models.EntityTypeTask,
			ToEntityID:       0,
			RelationshipType: models.EntityRelationshipType("depends_on"),
		}

		err := entityRelRepo.Create(ctx, rel)
		if err == nil {
			t.Error("Expected validation error for invalid to_entity_id, got nil")
		}
	})
}
