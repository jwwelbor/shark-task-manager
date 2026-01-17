package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCreateTaskNote tests creating a task note
func TestCreateTaskNote(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task to add note to
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create a task note
	createdBy := "test-agent"
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeDecision,
		Content:   "Decided to use SQLite for persistence",
		CreatedBy: &createdBy,
	}

	err = noteRepo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Failed to create task note: %v", err)
	}

	if note.ID == 0 {
		t.Error("Expected note ID to be set after creation")
	}

	// Verify note was created in database
	var count int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_notes WHERE id = ?", note.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query task notes: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 note in database, got %d", count)
	}
}

// TestCreateTaskNoteValidation tests validation during note creation
func TestCreateTaskNoteValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	tests := []struct {
		name        string
		note        *models.TaskNote
		expectError bool
	}{
		{
			name: "invalid task ID",
			note: &models.TaskNote{
				TaskID:   0,
				NoteType: models.NoteTypeComment,
				Content:  "Test",
			},
			expectError: true,
		},
		{
			name: "invalid note type",
			note: &models.TaskNote{
				TaskID:   1,
				NoteType: "invalid_type",
				Content:  "Test",
			},
			expectError: true,
		},
		{
			name: "empty content",
			note: &models.TaskNote{
				TaskID:   1,
				NoteType: models.NoteTypeComment,
				Content:  "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := noteRepo.Create(ctx, tt.note)
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestGetTaskNoteByID tests retrieving a task note by ID
func TestGetTaskNoteByID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create a note
	createdBy := "test-agent"
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeSolution,
		Content:   "Fixed bug by adding null check",
		CreatedBy: &createdBy,
	}
	err = noteRepo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Retrieve the note
	retrieved, err := noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to get note by ID: %v", err)
	}

	// Verify
	if retrieved.ID != note.ID {
		t.Errorf("Expected ID %d, got %d", note.ID, retrieved.ID)
	}
	if retrieved.TaskID != taskID {
		t.Errorf("Expected TaskID %d, got %d", taskID, retrieved.TaskID)
	}
	if retrieved.NoteType != models.NoteTypeSolution {
		t.Errorf("Expected NoteType %s, got %s", models.NoteTypeSolution, retrieved.NoteType)
	}
	if retrieved.Content != note.Content {
		t.Errorf("Expected Content %s, got %s", note.Content, retrieved.Content)
	}
	if retrieved.CreatedBy == nil || *retrieved.CreatedBy != "test-agent" {
		t.Error("Expected CreatedBy to be 'test-agent'")
	}
}

// TestGetTaskNoteByTaskID tests retrieving all notes for a task
func TestGetTaskNoteByTaskID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-002'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create multiple notes
	createdBy := "test-agent"
	notes := []*models.TaskNote{
		{
			TaskID:    taskID,
			NoteType:  models.NoteTypeDecision,
			Content:   "Decision 1",
			CreatedBy: &createdBy,
		},
		{
			TaskID:    taskID,
			NoteType:  models.NoteTypeImplementation,
			Content:   "Implementation note",
			CreatedBy: &createdBy,
		},
		{
			TaskID:    taskID,
			NoteType:  models.NoteTypeTesting,
			Content:   "Testing note",
			CreatedBy: &createdBy,
		},
	}

	for _, note := range notes {
		err = noteRepo.Create(ctx, note)
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}
	}

	// Retrieve all notes for the task
	retrieved, err := noteRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		t.Fatalf("Failed to get notes by task ID: %v", err)
	}

	// Verify at least 3 notes returned (there might be more from other tests)
	if len(retrieved) < 3 {
		t.Errorf("Expected at least 3 notes, got %d", len(retrieved))
	}

	// Verify notes are ordered by created_at ascending
	for i := 1; i < len(retrieved); i++ {
		if retrieved[i].CreatedAt.Before(retrieved[i-1].CreatedAt) {
			t.Error("Notes should be ordered by created_at ascending")
		}
	}
}

// TestGetTaskNoteByTaskIDAndType tests filtering notes by type
func TestGetTaskNoteByTaskIDAndType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-003'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create notes with different types
	createdBy := "test-agent"
	notes := []*models.TaskNote{
		{TaskID: taskID, NoteType: models.NoteTypeDecision, Content: "Decision 1", CreatedBy: &createdBy},
		{TaskID: taskID, NoteType: models.NoteTypeDecision, Content: "Decision 2", CreatedBy: &createdBy},
		{TaskID: taskID, NoteType: models.NoteTypeBlocker, Content: "Blocker 1", CreatedBy: &createdBy},
		{TaskID: taskID, NoteType: models.NoteTypeSolution, Content: "Solution 1", CreatedBy: &createdBy},
	}

	for _, note := range notes {
		err = noteRepo.Create(ctx, note)
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}
	}

	// Test filtering by single type
	decisions, err := noteRepo.GetByTaskIDAndType(ctx, taskID, []string{"decision"})
	if err != nil {
		t.Fatalf("Failed to get decision notes: %v", err)
	}

	if len(decisions) < 2 {
		t.Errorf("Expected at least 2 decision notes, got %d", len(decisions))
	}

	for _, note := range decisions {
		if note.NoteType != models.NoteTypeDecision {
			t.Errorf("Expected all notes to be decision type, got %s", note.NoteType)
		}
	}

	// Test filtering by multiple types
	filtered, err := noteRepo.GetByTaskIDAndType(ctx, taskID, []string{"decision", "solution"})
	if err != nil {
		t.Fatalf("Failed to get filtered notes: %v", err)
	}

	if len(filtered) < 3 {
		t.Errorf("Expected at least 3 notes (2 decision + 1 solution), got %d", len(filtered))
	}

	for _, note := range filtered {
		if note.NoteType != models.NoteTypeDecision && note.NoteType != models.NoteTypeSolution {
			t.Errorf("Expected decision or solution type, got %s", note.NoteType)
		}
	}

	// Test empty type filter (should return all notes)
	all, err := noteRepo.GetByTaskIDAndType(ctx, taskID, []string{})
	if err != nil {
		t.Fatalf("Failed to get all notes: %v", err)
	}

	if len(all) < 4 {
		t.Errorf("Expected at least 4 notes, got %d", len(all))
	}
}

// TestSearchTaskNotes tests searching across all notes
func TestSearchTaskNotes(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get tasks
	var task1ID, task2ID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&task1ID)
	if err != nil {
		t.Fatalf("Failed to get test task 1: %v", err)
	}
	err = database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-002'").Scan(&task2ID)
	if err != nil {
		t.Fatalf("Failed to get test task 2: %v", err)
	}

	// Create notes with searchable content
	createdBy := "test-agent"
	notes := []*models.TaskNote{
		{TaskID: task1ID, NoteType: models.NoteTypeDecision, Content: "Used singleton pattern for state management", CreatedBy: &createdBy},
		{TaskID: task2ID, NoteType: models.NoteTypeDecision, Content: "Considered singleton pattern but chose factory pattern", CreatedBy: &createdBy},
		{TaskID: task1ID, NoteType: models.NoteTypeSolution, Content: "Fixed Safari flash issue by moving script", CreatedBy: &createdBy},
	}

	for _, note := range notes {
		err = noteRepo.Create(ctx, note)
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}
	}

	// Test search across all notes
	results, err := noteRepo.Search(ctx, "singleton pattern", []string{}, "", "")
	if err != nil {
		t.Fatalf("Failed to search notes: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'singleton pattern', got %d", len(results))
	}

	// Test search with type filter
	decisionResults, err := noteRepo.Search(ctx, "singleton", []string{"decision"}, "", "")
	if err != nil {
		t.Fatalf("Failed to search decision notes: %v", err)
	}

	if len(decisionResults) < 2 {
		t.Errorf("Expected at least 2 decision notes for 'singleton', got %d", len(decisionResults))
	}

	for _, note := range decisionResults {
		if note.NoteType != models.NoteTypeDecision {
			t.Errorf("Expected decision type, got %s", note.NoteType)
		}
	}

	// Test case-insensitive search
	caseResults, err := noteRepo.Search(ctx, "SAFARI", []string{}, "", "")
	if err != nil {
		t.Fatalf("Failed to search with uppercase: %v", err)
	}

	if len(caseResults) < 1 {
		t.Errorf("Expected at least 1 result for 'SAFARI' (case-insensitive), got %d", len(caseResults))
	}
}

// TestSearchTaskNotesWithFilters tests search with epic and feature filters
func TestSearchTaskNotesWithFilters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create note
	createdBy := "test-agent"
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeReference,
		Content:   "Referenced external API documentation",
		CreatedBy: &createdBy,
	}
	err = noteRepo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Search with epic filter
	results, err := noteRepo.Search(ctx, "API", []string{}, "E99", "")
	if err != nil {
		t.Fatalf("Failed to search with epic filter: %v", err)
	}

	if len(results) < 1 {
		t.Errorf("Expected at least 1 result in epic E99, got %d", len(results))
	}

	// Search with feature filter
	results, err = noteRepo.Search(ctx, "API", []string{}, "", "E99-F99")
	if err != nil {
		t.Fatalf("Failed to search with feature filter: %v", err)
	}

	if len(results) < 1 {
		t.Errorf("Expected at least 1 result in feature E99-F99, got %d", len(results))
	}
}

// TestDeleteTaskNote tests deleting a task note
func TestDeleteTaskNote(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create a note
	createdBy := "test-agent"
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeComment,
		Content:   "Temporary note to be deleted",
		CreatedBy: &createdBy,
	}
	err = noteRepo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Delete the note
	err = noteRepo.Delete(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to delete note: %v", err)
	}

	// Verify note is deleted
	_, err = noteRepo.GetByID(ctx, note.ID)
	if err == nil {
		t.Error("Expected error when getting deleted note, got nil")
	}

	// Test deleting non-existent note
	err = noteRepo.Delete(ctx, 999999)
	if err == nil {
		t.Error("Expected error when deleting non-existent note, got nil")
	}
}

// TestTaskNoteCascadeDelete tests that notes are deleted when task is deleted
func TestTaskNoteCascadeDelete(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()

	// Get an existing task
	task, err := taskRepo.GetByKey(ctx, "T-E99-F99-004")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}
	taskIDToDelete := task.ID

	// Create notes for the task
	createdBy := "test-agent"
	for i := 0; i < 3; i++ {
		note := &models.TaskNote{
			TaskID:    taskIDToDelete,
			NoteType:  models.NoteTypeComment,
			Content:   "Test note",
			CreatedBy: &createdBy,
		}
		err = noteRepo.Create(ctx, note)
		if err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}
	}

	// Verify notes exist
	notes, err := noteRepo.GetByTaskID(ctx, taskIDToDelete)
	if err != nil {
		t.Fatalf("Failed to get notes: %v", err)
	}
	if len(notes) != 3 {
		t.Errorf("Expected 3 notes, got %d", len(notes))
	}

	// Delete the task
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskIDToDelete)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify notes are cascade deleted
	notes, err = noteRepo.GetByTaskID(ctx, taskIDToDelete)
	if err != nil {
		t.Fatalf("Failed to get notes after task deletion: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("Expected 0 notes after task deletion, got %d", len(notes))
	}
}

// TestCreateTaskNoteWithMetadata tests creating a task note with metadata
func TestCreateTaskNoteWithMetadata(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task to add note to
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-001'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create a task note with metadata
	createdBy := "test-agent"
	metadata := `{"history_id": 123, "from_status": "draft", "to_status": "in_progress"}`
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeDecision,
		Content:   "Rejection reason",
		CreatedBy: &createdBy,
		Metadata:  &metadata,
	}

	err = noteRepo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Failed to create task note with metadata: %v", err)
	}

	if note.ID == 0 {
		t.Error("Expected note ID to be set after creation")
	}

	// Retrieve the note and verify metadata
	retrieved, err := noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to get note: %v", err)
	}

	if retrieved.Metadata == nil {
		t.Error("Expected metadata to be set")
	} else if *retrieved.Metadata != metadata {
		t.Errorf("Expected metadata %q, got %q", metadata, *retrieved.Metadata)
	}
}

// TestCreateTaskNoteWithoutMetadata tests creating a task note without metadata
func TestCreateTaskNoteWithoutMetadata(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks WHERE key = 'T-E99-F99-002'").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create a note without metadata (should be NULL)
	createdBy := "test-agent"
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeComment,
		Content:   "Simple comment",
		CreatedBy: &createdBy,
		Metadata:  nil,
	}

	err = noteRepo.Create(ctx, note)
	if err != nil {
		t.Fatalf("Failed to create task note: %v", err)
	}

	// Verify note was created with NULL metadata
	retrieved, err := noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to get note: %v", err)
	}

	if retrieved.Metadata != nil {
		t.Errorf("Expected metadata to be NULL, got %v", retrieved.Metadata)
	}
}

// TestIndexOnNoteTypeAndTaskID tests that the index exists for performance
func TestIndexOnNoteTypeAndTaskID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Check that the index exists
	query := `
		SELECT name FROM sqlite_master
		WHERE type = 'index'
		AND name = 'idx_task_notes_type_task'
	`

	var indexName string
	err := database.QueryRowContext(ctx, query).Scan(&indexName)
	if err != nil {
		t.Fatalf("Index idx_task_notes_type_task not found: %v", err)
	}

	if indexName != "idx_task_notes_type_task" {
		t.Errorf("Expected index name 'idx_task_notes_type_task', got %q", indexName)
	}
}

// TestMetadataColumnExists tests that metadata column exists in task_notes table
func TestMetadataColumnExists(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Check that metadata column exists
	var columnCount int
	err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pragma_table_info('task_notes') WHERE name = 'metadata'
	`).Scan(&columnCount)
	if err != nil {
		t.Fatalf("Failed to check metadata column: %v", err)
	}

	if columnCount != 1 {
		t.Errorf("Expected metadata column to exist, got count %d", columnCount)
	}
}
