package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestSearchWithTimePeriodSince tests filtering notes by since date
func TestSearchWithTimePeriodSince(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

	// Seed test data
	_, featureID := test.SeedTestData()

	// Create a test task
	taskQuery := `INSERT INTO tasks (key, title, status, feature_id) VALUES (?, ?, ?, ?)`
	result, err := database.ExecContext(ctx, taskQuery, "TEST-E07-F01-999", "Test Task", "todo", featureID)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	taskID, _ := result.LastInsertId()
	defer func() {
		database.ExecContext(ctx, "DELETE FROM task_notes WHERE task_id = ?", taskID)
		database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)
	}()

	// Create notes with different timestamps
	oldNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Old rejection from 2025-12-01",
		CreatedBy: createStr("reviewer"),
	}
	err = noteRepo.Create(ctx, oldNote)
	if err != nil {
		t.Fatalf("Failed to create old note: %v", err)
	}

	// Update the created_at timestamp manually
	_, _ = database.ExecContext(ctx, "UPDATE task_notes SET created_at = '2025-12-01 10:00:00' WHERE id = ?", oldNote.ID)

	recentNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Recent rejection from 2026-01-15",
		CreatedBy: createStr("reviewer"),
	}
	err = noteRepo.Create(ctx, recentNote)
	if err != nil {
		t.Fatalf("Failed to create recent note: %v", err)
	}

	// Search notes created since 2026-01-01
	notes, err := noteRepo.SearchWithTimePeriod(ctx, "", []string{string(models.NoteTypeRejection)}, "", "", "2026-01-01", "")

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return only the recent note
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}

	// Verify the returned note is the recent one
	if len(notes) > 0 && notes[0].ID != recentNote.ID {
		t.Errorf("expected recent note (id=%d), got id=%d", recentNote.ID, notes[0].ID)
	}
}

// TestSearchWithTimePeriodUntil tests filtering notes by until date
func TestSearchWithTimePeriodUntil(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

	// Seed test data
	_, featureID := test.SeedTestData()

	// Create a test task
	taskQuery := `INSERT INTO tasks (key, title, status, feature_id) VALUES (?, ?, ?, ?)`
	result, err := database.ExecContext(ctx, taskQuery, "TEST-E07-F01-998", "Test Task", "todo", featureID)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	taskID, _ := result.LastInsertId()
	defer func() {
		database.ExecContext(ctx, "DELETE FROM task_notes WHERE task_id = ?", taskID)
		database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)
	}()

	// Create notes with different timestamps
	oldNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Old rejection",
		CreatedBy: createStr("reviewer"),
	}
	err = noteRepo.Create(ctx, oldNote)
	if err != nil {
		t.Fatalf("Failed to create old note: %v", err)
	}
	_, _ = database.ExecContext(ctx, "UPDATE task_notes SET created_at = '2025-12-01 10:00:00' WHERE id = ?", oldNote.ID)

	recentNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Recent rejection",
		CreatedBy: createStr("reviewer"),
	}
	err = noteRepo.Create(ctx, recentNote)
	if err != nil {
		t.Fatalf("Failed to create recent note: %v", err)
	}

	// Search notes created until 2025-12-15
	notes, err := noteRepo.SearchWithTimePeriod(ctx, "", []string{string(models.NoteTypeRejection)}, "", "", "", "2025-12-15")

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return only the old note
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}

	// Verify the returned note is the old one
	if len(notes) > 0 && notes[0].ID != oldNote.ID {
		t.Errorf("expected old note (id=%d), got id=%d", oldNote.ID, notes[0].ID)
	}
}

// TestSearchWithTimePeriodBothDates tests filtering with both since and until
func TestSearchWithTimePeriodBothDates(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id > 0")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")

	// Seed test data
	_, featureID := test.SeedTestData()

	// Create a test task
	taskQuery := `INSERT INTO tasks (key, title, status, feature_id) VALUES (?, ?, ?, ?)`
	result, err := database.ExecContext(ctx, taskQuery, "TEST-E07-F01-997", "Test Task", "todo", featureID)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	taskID, _ := result.LastInsertId()
	defer func() {
		database.ExecContext(ctx, "DELETE FROM task_notes WHERE task_id = ?", taskID)
		database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)
	}()

	// Create notes with different timestamps
	veryOldNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Very old",
		CreatedBy: createStr("reviewer"),
	}
	noteRepo.Create(ctx, veryOldNote)
	_, _ = database.ExecContext(ctx, "UPDATE task_notes SET created_at = '2025-11-01 10:00:00' WHERE id = ?", veryOldNote.ID)

	middleNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Middle",
		CreatedBy: createStr("reviewer"),
	}
	noteRepo.Create(ctx, middleNote)
	_, _ = database.ExecContext(ctx, "UPDATE task_notes SET created_at = '2025-12-15 10:00:00' WHERE id = ?", middleNote.ID)

	veryRecentNote := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   "Very recent",
		CreatedBy: createStr("reviewer"),
	}
	noteRepo.Create(ctx, veryRecentNote)
	_, _ = database.ExecContext(ctx, "UPDATE task_notes SET created_at = '2026-01-15 10:00:00' WHERE id = ?", veryRecentNote.ID)

	// Search notes between 2025-12-01 and 2025-12-31
	notes, err := noteRepo.SearchWithTimePeriod(ctx, "", []string{string(models.NoteTypeRejection)}, "", "", "2025-12-01", "2025-12-31")

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return only the middle note
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}

	if len(notes) > 0 && notes[0].ID != middleNote.ID {
		t.Errorf("expected middle note (id=%d), got id=%d", middleNote.ID, notes[0].ID)
	}
}

// Helper function
func createStr(s string) *string {
	return &s
}
