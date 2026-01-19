package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCreateRejectionNoteBasic tests creating a rejection note with metadata
func TestCreateRejectionNoteBasic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data to get task and history IDs
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create a rejection note with metadata
	historyID := int64(123)
	fromStatus := "ready_for_code_review"
	toStatus := "in_development"
	rejectedBy := "reviewer-agent"
	reason := "Missing error handling on line 67"

	note, err := noteRepo.CreateRejectionNote(
		ctx,
		taskID,
		historyID,
		fromStatus,
		toStatus,
		reason,
		rejectedBy,
		nil,
	)

	if err != nil {
		t.Fatalf("Failed to create rejection note: %v", err)
	}

	// Verify note was created
	if note == nil {
		t.Fatal("Expected note to be returned, got nil")
	}

	if note.ID == 0 {
		t.Error("Expected note ID to be set after creation")
	}

	// Verify note type is rejection
	if note.NoteType != models.NoteTypeRejection {
		t.Errorf("Expected note type 'rejection', got %q", note.NoteType)
	}

	// Verify content
	if note.Content != reason {
		t.Errorf("Expected content %q, got %q", reason, note.Content)
	}

	// Verify created_by
	if note.CreatedBy == nil || *note.CreatedBy != rejectedBy {
		t.Errorf("Expected created_by %q, got %v", rejectedBy, note.CreatedBy)
	}

	// Verify metadata is valid JSON and contains expected fields
	if note.Metadata == nil {
		t.Fatal("Expected metadata to be set, got nil")
	}

	var metadata map[string]interface{}
	err = json.Unmarshal([]byte(*note.Metadata), &metadata)
	if err != nil {
		t.Fatalf("Failed to parse metadata JSON: %v", err)
	}

	// Verify metadata fields
	if histID, ok := metadata["history_id"].(float64); !ok || int64(histID) != historyID {
		t.Errorf("Expected history_id %d in metadata, got %v", historyID, metadata["history_id"])
	}

	if fromStat, ok := metadata["from_status"].(string); !ok || fromStat != fromStatus {
		t.Errorf("Expected from_status %q in metadata, got %v", fromStatus, metadata["from_status"])
	}

	if toStat, ok := metadata["to_status"].(string); !ok || toStat != toStatus {
		t.Errorf("Expected to_status %q in metadata, got %v", toStatus, metadata["to_status"])
	}

	// Verify note in database
	retrieved, err := noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve created note: %v", err)
	}

	if retrieved.NoteType != models.NoteTypeRejection {
		t.Errorf("Expected retrieved note type to be rejection, got %q", retrieved.NoteType)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
}

// TestCreateRejectionNoteWithDocumentPath tests creating a rejection note with document path
func TestCreateRejectionNoteWithDocumentPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create rejection note with document path
	historyID := int64(456)
	fromStatus := "ready_for_qa"
	toStatus := "in_development"
	rejectedBy := "qa-agent"
	reason := "Tests fail on edge case"
	documentPath := "docs/bugs/BUG-2026-046.md"

	note, err := noteRepo.CreateRejectionNote(
		ctx,
		taskID,
		historyID,
		fromStatus,
		toStatus,
		reason,
		rejectedBy,
		&documentPath,
	)

	if err != nil {
		t.Fatalf("Failed to create rejection note with document path: %v", err)
	}

	// Verify metadata contains document_path
	if note.Metadata == nil {
		t.Fatal("Expected metadata to be set, got nil")
	}

	var metadata map[string]interface{}
	err = json.Unmarshal([]byte(*note.Metadata), &metadata)
	if err != nil {
		t.Fatalf("Failed to parse metadata JSON: %v", err)
	}

	if docPath, ok := metadata["document_path"].(string); !ok || docPath != documentPath {
		t.Errorf("Expected document_path %q in metadata, got %v", documentPath, metadata["document_path"])
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
}

// TestCreateRejectionNoteWithoutDocumentPath tests that document_path is omitted when nil
func TestCreateRejectionNoteWithoutDocumentPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create rejection note WITHOUT document path
	historyID := int64(789)
	fromStatus := "ready_for_review"
	toStatus := "todo"
	rejectedBy := "developer-agent"
	reason := "Need more unit test coverage"

	note, err := noteRepo.CreateRejectionNote(
		ctx,
		taskID,
		historyID,
		fromStatus,
		toStatus,
		reason,
		rejectedBy,
		nil, // No document path
	)

	if err != nil {
		t.Fatalf("Failed to create rejection note: %v", err)
	}

	// Verify metadata exists and doesn't have document_path
	if note.Metadata == nil {
		t.Fatal("Expected metadata to be set, got nil")
	}

	var metadata map[string]interface{}
	err = json.Unmarshal([]byte(*note.Metadata), &metadata)
	if err != nil {
		t.Fatalf("Failed to parse metadata JSON: %v", err)
	}

	// document_path should not be in metadata when nil
	if _, hasDocPath := metadata["document_path"]; hasDocPath {
		t.Error("Expected document_path to not be in metadata when nil, but it was present")
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
}

// TestCreateRejectionNoteMetadataStructure tests the complete metadata JSON structure
func TestCreateRejectionNoteMetadataStructure(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create rejection note
	historyID := int64(999)
	fromStatus := "in_code_review"
	toStatus := "in_development"
	documentPath := "docs/review-feedback.md"

	note, err := noteRepo.CreateRejectionNote(
		ctx,
		taskID,
		historyID,
		fromStatus,
		toStatus,
		"Needs refactoring for clarity",
		"code-reviewer",
		&documentPath,
	)

	if err != nil {
		t.Fatalf("Failed to create rejection note: %v", err)
	}

	// Retrieve and verify structure
	retrieved, err := noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve note: %v", err)
	}

	if retrieved.Metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	// Parse and validate complete structure
	var metadata struct {
		HistoryID    int64  `json:"history_id"`
		FromStatus   string `json:"from_status"`
		ToStatus     string `json:"to_status"`
		DocumentPath string `json:"document_path,omitempty"`
	}

	err = json.Unmarshal([]byte(*retrieved.Metadata), &metadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	// Verify all fields
	if metadata.HistoryID != historyID {
		t.Errorf("Expected history_id %d, got %d", historyID, metadata.HistoryID)
	}
	if metadata.FromStatus != fromStatus {
		t.Errorf("Expected from_status %q, got %q", fromStatus, metadata.FromStatus)
	}
	if metadata.ToStatus != toStatus {
		t.Errorf("Expected to_status %q, got %q", toStatus, metadata.ToStatus)
	}
	if metadata.DocumentPath != documentPath {
		t.Errorf("Expected document_path %q, got %q", documentPath, metadata.DocumentPath)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
}

// TestCreateRejectionNoteInTransaction tests creating rejection note within a transaction
func TestCreateRejectionNoteInTransaction(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Begin transaction
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Create rejection note within transaction
	note, err := noteRepo.CreateRejectionNoteWithTx(
		ctx,
		tx,
		taskID,
		int64(111),
		"ready_for_approval",
		"in_development",
		"Missing security validation",
		"security-reviewer",
		nil,
	)

	if err != nil {
		t.Fatalf("Failed to create rejection note in transaction: %v", err)
	}

	if note.ID == 0 {
		t.Error("Expected note ID to be set")
	}

	// Verify note can be queried within transaction
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_notes WHERE id = ?", note.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query note in transaction: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 note in transaction, got %d", count)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify note persisted after commit
	retrieved, err := noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve committed note: %v", err)
	}

	if retrieved.ID != note.ID {
		t.Errorf("Expected note ID %d, got %d", note.ID, retrieved.ID)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
}

// TestCreateRejectionNoteTransactionRollback tests that rollback prevents persistence
func TestCreateRejectionNoteTransactionRollback(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Begin transaction
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Create rejection note within transaction
	note, err := noteRepo.CreateRejectionNoteWithTx(
		ctx,
		tx,
		taskID,
		int64(222),
		"ready_for_qa",
		"in_development",
		"Performance issues detected",
		"qa-agent",
		nil,
	)

	if err != nil {
		t.Fatalf("Failed to create rejection note: %v", err)
	}

	createdNoteID := note.ID

	// Rollback the transaction
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify note is not in database after rollback
	_, err = noteRepo.GetByID(ctx, createdNoteID)
	if err == nil {
		t.Error("Expected error when retrieving rolled-back note, got nil")
	}
}

// TestCreateRejectionNoteValidation tests validation of parameters
func TestCreateRejectionNoteValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a valid task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	tests := []struct {
		name        string
		taskID      int64
		historyID   int64
		fromStatus  string
		toStatus    string
		reason      string
		rejectedBy  string
		expectError bool
	}{
		{
			name:        "invalid task ID (zero)",
			taskID:      0,
			historyID:   1,
			fromStatus:  "draft",
			toStatus:    "todo",
			reason:      "test",
			rejectedBy:  "agent",
			expectError: true,
		},
		{
			name:        "empty reason",
			taskID:      taskID,
			historyID:   1,
			fromStatus:  "draft",
			toStatus:    "todo",
			reason:      "",
			rejectedBy:  "agent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := noteRepo.CreateRejectionNote(
				ctx,
				tt.taskID,
				tt.historyID,
				tt.fromStatus,
				tt.toStatus,
				tt.reason,
				tt.rejectedBy,
				nil,
			)

			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestGetRejectionNotesForTask tests retrieving rejection notes for a task
func TestGetRejectionNotesForTask(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up rejection notes
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create multiple rejection notes
	for i := 1; i <= 3; i++ {
		_, err := noteRepo.CreateRejectionNote(
			ctx,
			taskID,
			int64(100+i),
			"draft",
			"todo",
			"Rejection reason "+string(rune('0'+i)),
			"reviewer",
			nil,
		)
		if err != nil {
			t.Fatalf("Failed to create rejection note %d: %v", i, err)
		}
	}

	// Retrieve rejection notes for the task
	notes, err := noteRepo.GetByTaskIDAndType(ctx, taskID, []string{"rejection"})
	if err != nil {
		t.Fatalf("Failed to retrieve rejection notes: %v", err)
	}

	if len(notes) < 3 {
		t.Errorf("Expected at least 3 rejection notes, got %d", len(notes))
	}

	for _, note := range notes {
		if note.NoteType != models.NoteTypeRejection {
			t.Errorf("Expected rejection note type, got %q", note.NoteType)
		}

		if note.Metadata == nil {
			t.Error("Expected metadata to be set in rejection note")
		}
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection' AND task_id = ?", taskID)
}

// TestCreateRejectionNoteReasonEdgeCases tests rejection reason validation edge cases
func TestCreateRejectionNoteReasonEdgeCases(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	tests := []struct {
		name        string
		reason      string
		expectError bool
		description string
	}{
		{
			name:        "reason with special characters",
			reason:      "Missing validation for @#$%^&*() special chars",
			expectError: false,
			description: "Should accept reasons with special characters",
		},
		{
			name:        "reason with newlines",
			reason:      "Line 1\nLine 2\nLine 3",
			expectError: false,
			description: "Should preserve newlines in reason",
		},
		{
			name:        "reason with quotes",
			reason:      `Error: "undefined method" on line 42`,
			expectError: false,
			description: "Should handle quotes in reason",
		},
		{
			name:        "reason with unicode",
			reason:      "Unicode test: ñ é ü 中文 🔧",
			expectError: false,
			description: "Should support unicode in reason",
		},
		{
			name:        "whitespace-only reason",
			reason:      "   \t\n  ",
			expectError: true,
			description: "Whitespace-only reasons should fail validation",
		},
		{
			name:        "very long reason",
			reason:      "This is a very long reason. " + string(make([]byte, 5000)),
			expectError: false,
			description: "Should accept long reasons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := noteRepo.CreateRejectionNote(
				ctx,
				taskID,
				int64(9000),
				"draft",
				"todo",
				tt.reason,
				"reviewer",
				nil,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got nil", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
				}
				if note != nil {
					// Cleanup
					_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
				}
			}
		})
	}
}

// TestCreateRejectionNoteDocumentPathValidation tests document path handling
func TestCreateRejectionNoteDocumentPathValidation(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	tests := []struct {
		name             string
		documentPath     *string
		expectInMetadata bool
		expectedValue    string
		description      string
	}{
		{
			name:             "nil document path",
			documentPath:     nil,
			expectInMetadata: false,
			expectedValue:    "",
			description:      "Nil document path should be omitted from metadata",
		},
		{
			name:             "empty string document path",
			documentPath:     toStringPtr(""),
			expectInMetadata: false,
			expectedValue:    "",
			description:      "Empty string document path should be omitted from metadata",
		},
		{
			name:             "relative document path",
			documentPath:     toStringPtr("docs/bugs/BUG-001.md"),
			expectInMetadata: true,
			expectedValue:    "docs/bugs/BUG-001.md",
			description:      "Relative document path should be preserved",
		},
		{
			name:             "document path with special characters",
			documentPath:     toStringPtr("docs/2025-01-16_review-feedback.md"),
			expectInMetadata: true,
			expectedValue:    "docs/2025-01-16_review-feedback.md",
			description:      "Document path with special characters should be preserved",
		},
		{
			name:             "document path with backslashes",
			documentPath:     toStringPtr("docs\\windows\\path.md"),
			expectInMetadata: true,
			expectedValue:    "docs\\windows\\path.md",
			description:      "Document path with backslashes should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := noteRepo.CreateRejectionNote(
				ctx,
				taskID,
				int64(9100+len(tt.name)),
				"draft",
				"todo",
				"Test reason",
				"reviewer",
				tt.documentPath,
			)

			if err != nil {
				t.Fatalf("Failed to create rejection note: %v", err)
			}

			// Parse metadata
			if note.Metadata == nil {
				t.Fatal("Expected metadata to be set")
			}

			var metadata map[string]interface{}
			err = json.Unmarshal([]byte(*note.Metadata), &metadata)
			if err != nil {
				t.Fatalf("Failed to parse metadata: %v", err)
			}

			// Check document_path presence
			docPath, hasDocPath := metadata["document_path"]
			if tt.expectInMetadata {
				if !hasDocPath {
					t.Errorf("%s: Expected document_path in metadata, but it was missing", tt.description)
				}
				if docPathStr, ok := docPath.(string); ok && docPathStr != tt.expectedValue {
					t.Errorf("%s: Expected document_path %q, got %q", tt.description, tt.expectedValue, docPathStr)
				}
			} else {
				if hasDocPath {
					t.Errorf("%s: Expected document_path to be omitted, but got %v", tt.description, docPath)
				}
			}

			// Cleanup
			_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
		})
	}
}

// TestRejectionNotesOrderedByTimestamp tests that rejection notes are ordered correctly
func TestRejectionNotesOrderedByTimestamp(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create multiple rejection notes
	var createdNotes []*models.TaskNote
	for i := 0; i < 5; i++ {
		note, err := noteRepo.CreateRejectionNote(
			ctx,
			taskID,
			int64(9200+i),
			"draft",
			"todo",
			fmt.Sprintf("Rejection reason %d", i),
			"reviewer",
			nil,
		)
		if err != nil {
			t.Fatalf("Failed to create rejection note %d: %v", i, err)
		}
		createdNotes = append(createdNotes, note)
	}

	// Retrieve rejection notes
	notes, err := noteRepo.GetByTaskIDAndType(ctx, taskID, []string{"rejection"})
	if err != nil {
		t.Fatalf("Failed to retrieve rejection notes: %v", err)
	}

	if len(notes) < 5 {
		t.Errorf("Expected at least 5 rejection notes, got %d", len(notes))
	}

	// Verify notes are ordered by created_at ascending
	for i := 0; i < len(notes)-1; i++ {
		if notes[i].CreatedAt.After(notes[i+1].CreatedAt) {
			t.Errorf("Notes not ordered by timestamp: note %d created at %v, note %d created at %v",
				i, notes[i].CreatedAt, i+1, notes[i+1].CreatedAt)
		}
	}

	// Cleanup
	for _, note := range createdNotes {
		_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
	}
}

// TestRejectionNoteMetadataIntegrity tests metadata consistency and integrity
func TestRejectionNoteMetadataIntegrity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Test cases for metadata integrity
	tests := []struct {
		name       string
		historyID  int64
		fromStatus string
		toStatus   string
		reason     string
		docPath    *string
	}{
		{
			name:       "complete metadata with all fields",
			historyID:  int64(10001),
			fromStatus: "ready_for_code_review",
			toStatus:   "in_development",
			reason:     "Needs refactoring",
			docPath:    toStringPtr("docs/feedback.md"),
		},
		{
			name:       "metadata without document path",
			historyID:  int64(10002),
			fromStatus: "ready_for_qa",
			toStatus:   "in_development",
			reason:     "Failing tests",
			docPath:    nil,
		},
		{
			name:       "metadata with special status values",
			historyID:  int64(10003),
			fromStatus: "ready_for_approval",
			toStatus:   "ready_for_refinement",
			reason:     "Specification unclear",
			docPath:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := noteRepo.CreateRejectionNote(
				ctx,
				taskID,
				tt.historyID,
				tt.fromStatus,
				tt.toStatus,
				tt.reason,
				"reviewer",
				tt.docPath,
			)

			if err != nil {
				t.Fatalf("Failed to create rejection note: %v", err)
			}

			// Retrieve and verify metadata integrity
			retrieved, err := noteRepo.GetByID(ctx, note.ID)
			if err != nil {
				t.Fatalf("Failed to retrieve note: %v", err)
			}

			if retrieved.Metadata == nil {
				t.Fatal("Expected metadata, got nil")
			}

			// Unmarshal and verify structure
			var metadata RejectionNoteMetadata
			err = json.Unmarshal([]byte(*retrieved.Metadata), &metadata)
			if err != nil {
				t.Fatalf("Failed to unmarshal metadata: %v", err)
			}

			// Verify all required fields
			if metadata.HistoryID != tt.historyID {
				t.Errorf("Expected history_id %d, got %d", tt.historyID, metadata.HistoryID)
			}
			if metadata.FromStatus != tt.fromStatus {
				t.Errorf("Expected from_status %q, got %q", tt.fromStatus, metadata.FromStatus)
			}
			if metadata.ToStatus != tt.toStatus {
				t.Errorf("Expected to_status %q, got %q", tt.toStatus, metadata.ToStatus)
			}

			// Verify content matches reason
			if retrieved.Content != tt.reason {
				t.Errorf("Expected content %q, got %q", tt.reason, retrieved.Content)
			}

			// Cleanup
			_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE id = ?", note.ID)
		})
	}
}

// TestRejectionNoteCountPerTask tests counting rejection notes for a task
func TestRejectionNoteCountPerTask(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	noteRepo := NewTaskNoteRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection'")

	// Seed test data
	_, _ = test.SeedTestData()

	// Get a task
	var taskID int64
	err := database.QueryRowContext(ctx, "SELECT id FROM tasks LIMIT 1").Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Create 10 rejection notes
	expectedCount := 10
	for i := 0; i < expectedCount; i++ {
		_, err := noteRepo.CreateRejectionNote(
			ctx,
			taskID,
			int64(10100+i),
			"draft",
			"todo",
			fmt.Sprintf("Rejection %d", i),
			"reviewer",
			nil,
		)
		if err != nil {
			t.Fatalf("Failed to create rejection note %d: %v", i, err)
		}
	}

	// Count rejection notes
	notes, err := noteRepo.GetByTaskIDAndType(ctx, taskID, []string{"rejection"})
	if err != nil {
		t.Fatalf("Failed to retrieve rejection notes: %v", err)
	}

	// Filter to only rejection notes for this task created in this test
	rejectionCount := 0
	for _, note := range notes {
		if note.NoteType == models.NoteTypeRejection && note.TaskID == taskID {
			rejectionCount++
		}
	}

	if rejectionCount != expectedCount {
		t.Errorf("Expected %d rejection notes, got %d", expectedCount, rejectionCount)
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM task_notes WHERE note_type = 'rejection' AND task_id = ?", taskID)
}

// Helper function to create a string pointer
func toStringPtr(s string) *string {
	return &s
}
