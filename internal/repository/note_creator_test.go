package repository

// NoteCreator interface decoupling tests.
//
// These tests verify:
//   - NoteCreator interface is defined (compile-time check via var _ NoteCreator)
//   - Nil NoteCreator causes graceful degradation during forced status update
//   - Non-nil NoteCreator is called on forced status update with rejection reason
//   - note package does not import task package (no import cycle)
//   - note.EntityNoteRepository satisfies NoteCreator (compile-time check in aliases.go)

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/note"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// mockNoteCreator is a test double for the NoteCreator interface.
type mockNoteCreator struct {
	callCount    int
	capturedArgs noteCreatorArgs
	returnErr    error
}

type noteCreatorArgs struct {
	entityType   models.EntityType
	entityID     int64
	historyID    int64
	fromStatus   string
	toStatus     string
	reason       string
	rejectedBy   string
	documentPath *string
}

func (m *mockNoteCreator) CreateRejectionNoteWithTx(
	_ context.Context,
	_ *sql.Tx,
	entityType models.EntityType,
	entityID int64,
	historyID int64,
	fromStatus, toStatus, reason, rejectedBy string,
	documentPath *string,
) (*models.EntityNote, error) {
	m.callCount++
	m.capturedArgs = noteCreatorArgs{
		entityType:   entityType,
		entityID:     entityID,
		historyID:    historyID,
		fromStatus:   fromStatus,
		toStatus:     toStatus,
		reason:       reason,
		rejectedBy:   rejectedBy,
		documentPath: documentPath,
	}
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	// Return a minimal EntityNote so callers don't panic on nil.
	return &models.EntityNote{ID: 999, NoteType: models.NoteTypeRejection}, nil
}

// TestNoteCreator_NilGracefulDegradation_ForcedStatus verifies that when NoteCreator is nil,
// forced status update with rejection reason succeeds without panicking or returning an error.
func TestNoteCreator_NilGracefulDegradation_ForcedStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	// Explicitly pass nil NoteCreator to test graceful degradation.
	// (NewTaskRepository auto-wires note.EntityNoteRepository; use WithNoteCreator for nil.)
	repo := NewTaskRepositoryWithNoteCreator(db, nil)

	test.SeedTestData()

	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Reset to a known status.
	_, err = database.ExecContext(ctx, "UPDATE tasks SET status = 'in_development' WHERE id = ?", task.ID)
	if err != nil {
		t.Fatalf("Failed to reset task status: %v", err)
	}

	agent := "test-agent"
	reason := "missing tests"
	newStatus := models.TaskStatus("in_development")

	// UpdateStatusForced with a rejection reason but nil NoteCreator must not panic.
	err = repo.UpdateStatusForced(ctx, task.ID, newStatus, &agent, nil, &reason, nil, true)
	if err != nil {
		t.Errorf("expected no error with nil NoteCreator, got: %v", err)
	}
}

// TestNoteCreator_WithMock_ForcedStatus verifies that when NoteCreator is set,
// forced status update calls CreateRejectionNoteWithTx with the correct arguments.
func TestNoteCreator_WithMock_ForcedStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)

	mock := &mockNoteCreator{}
	repo := NewTaskRepositoryWithNoteCreator(db, mock)

	test.SeedTestData()

	task, err := repo.GetByKey(ctx, "T-E99-F99-001")
	if err != nil {
		t.Fatalf("Failed to get test task: %v", err)
	}

	// Set to in_development so the forced transition is recorded.
	_, err = database.ExecContext(ctx, "UPDATE tasks SET status = 'in_development' WHERE id = ?", task.ID)
	if err != nil {
		t.Fatalf("Failed to reset task status: %v", err)
	}

	agent := "reviewer"
	reason := "code needs error handling"
	newStatus := models.TaskStatus("changes_requested")

	err = repo.UpdateStatusForced(ctx, task.ID, newStatus, &agent, nil, &reason, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount == 0 {
		t.Error("expected NoteCreator.CreateRejectionNoteWithTx to be called, got 0 calls")
	}

	if mock.capturedArgs.reason != reason {
		t.Errorf("expected reason %q, got %q", reason, mock.capturedArgs.reason)
	}

	if mock.capturedArgs.rejectedBy != agent {
		t.Errorf("expected rejectedBy %q, got %q", agent, mock.capturedArgs.rejectedBy)
	}
}

// TestNoteCreator_EntityNoteRepositoryAccessibleViaBothPaths verifies that EntityNoteRepository
// is accessible via both the root package alias and the note sub-package directly.
func TestNoteCreator_EntityNoteRepositoryAccessibleViaBothPaths(t *testing.T) {
	database := test.GetTestDB()
	db := NewDB(database)

	// Path 1: via root package alias (repository.NewEntityNoteRepository)
	repoFromAlias := NewEntityNoteRepository(db)
	if repoFromAlias == nil {
		t.Fatal("repository.NewEntityNoteRepository returned nil")
	}

	// Path 2: via note sub-package directly (note.NewEntityNoteRepository)
	repoFromSubpkg := note.NewEntityNoteRepository(db)
	if repoFromSubpkg == nil {
		t.Fatal("note.NewEntityNoteRepository returned nil")
	}

	// Both paths produce the same concrete type (they are the same type via alias).
	var _ *note.EntityNoteRepository = repoFromAlias
	var _ *note.EntityNoteRepository = repoFromSubpkg
}

// TestNoteCreator_NoImportCycle is a documentation test that passes when the project compiles.
// If note imported repository, `go build ./...` would fail with a cycle error.
func TestNoteCreator_NoImportCycle(t *testing.T) {
	t.Log("no import cycle between repository and note packages")
}
