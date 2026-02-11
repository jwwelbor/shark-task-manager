package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestNotesSearchIntegration_EntityTypeFilter tests the entity-type filter functionality
func TestNotesSearchIntegration_EntityTypeFilter(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE content LIKE 'TEST-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E99-F99-999'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E99-F99'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

	// Create test entities
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	noteRepo := repository.NewEntityNoteRepository(db)

	// Create epic
	epic := &models.Epic{
		Key:      "E99",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
		Title:    "TEST-Integration Epic",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	})

	// Create feature
	feature := &models.Feature{
		Key:    "E99-F99",
		Title:  "TEST-Integration Feature",
		EpicID: epic.ID,
		Status: models.FeatureStatusActive,
	}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	})

	// Create task
	task := &models.Task{
		Key:       "T-E99-F99-999",
		Title:     "TEST-Integration Task",
		Status:    "todo",
		Priority:  5,
		FeatureID: feature.ID,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	})

	// Add notes to each entity
	createdBy := "test-agent"

	epicNote := &models.EntityNote{
		EntityType: models.EntityTypeEpic,
		EntityID:   epic.ID,
		NoteType:   models.NoteTypeComment,
		Content:    "TEST-Epic note",
		CreatedBy:  &createdBy,
	}
	if err := noteRepo.Create(ctx, epicNote); err != nil {
		t.Fatalf("Failed to create epic note: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE id = ?", epicNote.ID)
	})

	featureNote := &models.EntityNote{
		EntityType: models.EntityTypeFeature,
		EntityID:   feature.ID,
		NoteType:   models.NoteTypeDecision,
		Content:    "TEST-Feature note",
		CreatedBy:  &createdBy,
	}
	if err := noteRepo.Create(ctx, featureNote); err != nil {
		t.Fatalf("Failed to create feature note: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE id = ?", featureNote.ID)
	})

	taskNote := &models.EntityNote{
		EntityType: models.EntityTypeTask,
		EntityID:   task.ID,
		NoteType:   models.NoteTypeSolution,
		Content:    "TEST-Task note",
		CreatedBy:  &createdBy,
	}
	if err := noteRepo.Create(ctx, taskNote); err != nil {
		t.Fatalf("Failed to create task note: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE id = ?", taskNote.ID)
	})

	// Test search without entity type filter
	t.Run("Search all entities", func(t *testing.T) {
		notes, err := noteRepo.Search(ctx, "TEST-", nil, nil, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 3 {
			t.Errorf("Expected 3 notes, got %d", len(notes))
		}
	})

	// Test search with entity type filter - epic
	t.Run("Filter by entity type - epic", func(t *testing.T) {
		entityType := models.EntityTypeEpic
		notes, err := noteRepo.Search(ctx, "TEST-", nil, &entityType, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("Expected 1 epic note, got %d", len(notes))
		}
		if len(notes) > 0 && notes[0].EntityType != models.EntityTypeEpic {
			t.Errorf("Expected epic note, got %s", notes[0].EntityType)
		}
	})

	// Test search with entity type filter - feature
	t.Run("Filter by entity type - feature", func(t *testing.T) {
		entityType := models.EntityTypeFeature
		notes, err := noteRepo.Search(ctx, "TEST-", nil, &entityType, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("Expected 1 feature note, got %d", len(notes))
		}
		if len(notes) > 0 && notes[0].EntityType != models.EntityTypeFeature {
			t.Errorf("Expected feature note, got %s", notes[0].EntityType)
		}
	})

	// Test search with entity type filter - task
	t.Run("Filter by entity type - task", func(t *testing.T) {
		entityType := models.EntityTypeTask
		notes, err := noteRepo.Search(ctx, "TEST-", nil, &entityType, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("Expected 1 task note, got %d", len(notes))
		}
		if len(notes) > 0 && notes[0].EntityType != models.EntityTypeTask {
			t.Errorf("Expected task note, got %s", notes[0].EntityType)
		}
	})

	// Test JSON serialization
	t.Run("JSON output validation", func(t *testing.T) {
		jsonData, err := json.Marshal(epicNote)
		if err != nil {
			t.Fatalf("Failed to marshal note: %v", err)
		}

		var parsed models.EntityNote
		if err := json.Unmarshal(jsonData, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal note: %v", err)
		}

		if parsed.ID != epicNote.ID {
			t.Errorf("ID mismatch after JSON roundtrip")
		}
		if parsed.EntityType != epicNote.EntityType {
			t.Errorf("EntityType mismatch after JSON roundtrip")
		}
	})
}

// TestNotesSearchIntegration_EdgeCases tests edge cases
func TestNotesSearchIntegration_EdgeCases(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := repository.NewDB(database)

	noteRepo := repository.NewEntityNoteRepository(db)
	epicRepo := repository.NewEpicRepository(db)

	// Clean up test data
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE content LIKE 'TEST-EDGE%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E88'")

	// Create test epic
	epic := &models.Epic{
		Key:      "E88",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
		Title:    "TEST-Edge Epic",
	}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	})

	t.Run("Empty content validation", func(t *testing.T) {
		createdBy := "test-agent"
		note := &models.EntityNote{
			EntityType: models.EntityTypeEpic,
			EntityID:   epic.ID,
			NoteType:   models.NoteTypeComment,
			Content:    "",
			CreatedBy:  &createdBy,
		}
		err := noteRepo.Create(ctx, note)
		if err == nil {
			t.Errorf("Expected validation error for empty content")
		}
	})

	t.Run("Whitespace-only content validation", func(t *testing.T) {
		createdBy := "test-agent"
		note := &models.EntityNote{
			EntityType: models.EntityTypeEpic,
			EntityID:   epic.ID,
			NoteType:   models.NoteTypeComment,
			Content:    "   \n\t   ",
			CreatedBy:  &createdBy,
		}
		err := noteRepo.Create(ctx, note)
		if err == nil {
			t.Errorf("Expected validation error for whitespace-only content")
		}
	})

	t.Run("Invalid entity type validation", func(t *testing.T) {
		createdBy := "test-agent"
		note := &models.EntityNote{
			EntityType: "invalid_type",
			EntityID:   epic.ID,
			NoteType:   models.NoteTypeComment,
			Content:    "TEST-EDGE content",
			CreatedBy:  &createdBy,
		}
		err := noteRepo.Create(ctx, note)
		if err == nil {
			t.Errorf("Expected validation error for invalid entity type")
		}
	})

	t.Run("Note with null created_by", func(t *testing.T) {
		note := &models.EntityNote{
			EntityType: models.EntityTypeEpic,
			EntityID:   epic.ID,
			NoteType:   models.NoteTypeComment,
			Content:    "TEST-EDGE-NULL",
			CreatedBy:  nil,
		}
		if err := noteRepo.Create(ctx, note); err != nil {
			t.Fatalf("Failed to create note with null creator: %v", err)
		}
		t.Cleanup(func() {
			_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE id = ?", note.ID)
		})

		if note.CreatedBy != nil {
			t.Errorf("Expected nil creator, got %v", *note.CreatedBy)
		}
	})
}
