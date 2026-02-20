package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Mock implementations for NoteService dependencies

type mockNoteEntityNoteRepo struct {
	createFunc               func(ctx context.Context, note *models.EntityNote) error
	getByEntityFunc          func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
	getByEntityTypeFunc      func(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []string) ([]*models.EntityNote, error)
	searchFunc               func(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error)
	searchWithTimePeriodFunc func(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error)
}

func (m *mockNoteEntityNoteRepo) Create(ctx context.Context, note *models.EntityNote) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, note)
	}
	note.ID = 1
	return nil
}

func (m *mockNoteEntityNoteRepo) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
	if m.getByEntityFunc != nil {
		return m.getByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

func (m *mockNoteEntityNoteRepo) GetByEntityAndType(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []string) ([]*models.EntityNote, error) {
	if m.getByEntityTypeFunc != nil {
		return m.getByEntityTypeFunc(ctx, entityType, entityID, noteTypes)
	}
	return nil, nil
}

func (m *mockNoteEntityNoteRepo) Search(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, noteTypes, entityType, epicKey, featureKey)
	}
	return nil, nil
}

func (m *mockNoteEntityNoteRepo) SearchWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error) {
	if m.searchWithTimePeriodFunc != nil {
		return m.searchWithTimePeriodFunc(ctx, query, noteTypes, epicKey, featureKey, since, until)
	}
	return nil, nil
}

type mockNoteEpicRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.Epic, error)
	getByIDFunc  func(ctx context.Context, id int64) (*models.Epic, error)
}

func (m *mockNoteEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.Epic{ID: 1, Key: key}, nil
}

func (m *mockNoteEpicRepo) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &models.Epic{ID: id, Key: "E01"}, nil
}

type mockNoteFeatureRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.Feature, error)
	getByIDFunc  func(ctx context.Context, id int64) (*models.Feature, error)
}

func (m *mockNoteFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.Feature{ID: 2, Key: key}, nil
}

func (m *mockNoteFeatureRepo) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &models.Feature{ID: id, Key: "E01-F01"}, nil
}

type mockNoteTaskRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
	getByIDFunc  func(ctx context.Context, id int64) (*models.Task, error)
}

func (m *mockNoteTaskRepo) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.Task{ID: 3, Key: key}, nil
}

func (m *mockNoteTaskRepo) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &models.Task{ID: id, Key: "E01-F01-001"}, nil
}

func TestNoteService_AddNote_Epic(t *testing.T) {
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(ctx context.Context, note *models.EntityNote) error {
			note.ID = 42
			if note.EntityType != models.EntityTypeEpic {
				t.Errorf("expected entity type epic, got %s", note.EntityType)
			}
			if note.EntityID != 1 {
				t.Errorf("expected entity ID 1, got %d", note.EntityID)
			}
			return nil
		},
	}
	epicRepo := &mockNoteEpicRepo{}
	svc := NewNoteService(noteRepo, epicRepo, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	note, err := svc.AddNote(context.Background(), models.EntityTypeEpic, "E16", "comment", "Test note", "test-agent")
	if err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if note.ID != 42 {
		t.Errorf("expected note ID 42, got %d", note.ID)
	}
	if note.Content != "Test note" {
		t.Errorf("expected content 'Test note', got %s", note.Content)
	}
}

func TestNoteService_AddNote_Feature(t *testing.T) {
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(ctx context.Context, note *models.EntityNote) error {
			note.ID = 43
			if note.EntityType != models.EntityTypeFeature {
				t.Errorf("expected entity type feature, got %s", note.EntityType)
			}
			if note.EntityID != 2 {
				t.Errorf("expected entity ID 2, got %d", note.EntityID)
			}
			return nil
		},
	}
	featureRepo := &mockNoteFeatureRepo{}
	svc := NewNoteService(noteRepo, &mockNoteEpicRepo{}, featureRepo, &mockNoteTaskRepo{})

	note, err := svc.AddNote(context.Background(), models.EntityTypeFeature, "E16-F01", "decision", "Use polymorphic table", "dev")
	if err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if note.ID != 43 {
		t.Errorf("expected note ID 43, got %d", note.ID)
	}
}

func TestNoteService_AddNote_Task(t *testing.T) {
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(ctx context.Context, note *models.EntityNote) error {
			note.ID = 44
			if note.EntityType != models.EntityTypeTask {
				t.Errorf("expected entity type task, got %s", note.EntityType)
			}
			if note.EntityID != 3 {
				t.Errorf("expected entity ID 3, got %d", note.EntityID)
			}
			return nil
		},
	}
	taskRepo := &mockNoteTaskRepo{}
	svc := NewNoteService(noteRepo, &mockNoteEpicRepo{}, &mockNoteFeatureRepo{}, taskRepo)

	note, err := svc.AddNote(context.Background(), models.EntityTypeTask, "E16-F01-001", "blocker", "Blocked on API", "")
	if err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if note.ID != 44 {
		t.Errorf("expected note ID 44, got %d", note.ID)
	}
	if note.CreatedBy != nil {
		t.Error("expected nil CreatedBy for empty string")
	}
}

func TestNoteService_AddNote_InvalidKey(t *testing.T) {
	epicRepo := &mockNoteEpicRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	svc := NewNoteService(&mockNoteEntityNoteRepo{}, epicRepo, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	_, err := svc.AddNote(context.Background(), models.EntityTypeEpic, "E999", "comment", "Test", "agent")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestNoteService_AddNote_InvalidNoteType(t *testing.T) {
	svc := NewNoteService(&mockNoteEntityNoteRepo{}, &mockNoteEpicRepo{}, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	_, err := svc.AddNote(context.Background(), models.EntityTypeEpic, "E16", "invalid_type", "Test", "agent")
	if err == nil {
		t.Fatal("expected error for invalid note type")
	}
}

func TestNoteService_ListNotes(t *testing.T) {
	expectedNotes := []*models.EntityNote{
		{ID: 1, EntityType: models.EntityTypeEpic, EntityID: 1, NoteType: models.NoteTypeComment, Content: "Note 1"},
		{ID: 2, EntityType: models.EntityTypeEpic, EntityID: 1, NoteType: models.NoteTypeDecision, Content: "Note 2"},
	}

	noteRepo := &mockNoteEntityNoteRepo{
		getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
			return expectedNotes, nil
		},
	}
	svc := NewNoteService(noteRepo, &mockNoteEpicRepo{}, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	notes, err := svc.ListNotes(context.Background(), models.EntityTypeEpic, "E16", nil)
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestNoteService_ListNotes_WithTypeFilter(t *testing.T) {
	noteRepo := &mockNoteEntityNoteRepo{
		getByEntityTypeFunc: func(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []string) ([]*models.EntityNote, error) {
			if len(noteTypes) != 1 || noteTypes[0] != "decision" {
				t.Errorf("expected filter for 'decision', got %v", noteTypes)
			}
			return []*models.EntityNote{
				{ID: 2, NoteType: models.NoteTypeDecision, Content: "Decision note"},
			}, nil
		},
	}
	svc := NewNoteService(noteRepo, &mockNoteEpicRepo{}, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	notes, err := svc.ListNotes(context.Background(), models.EntityTypeEpic, "E16", []string{"decision"})
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
}

func TestNoteService_SearchNotes(t *testing.T) {
	noteRepo := &mockNoteEntityNoteRepo{
		searchFunc: func(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error) {
			if query != "API" {
				t.Errorf("expected query 'API', got %s", query)
			}
			return []*models.EntityNote{
				{ID: 1, Content: "API design decision"},
			}, nil
		},
	}
	svc := NewNoteService(noteRepo, &mockNoteEpicRepo{}, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	notes, err := svc.SearchNotes(context.Background(), "API", nil, nil, "", "")
	if err != nil {
		t.Fatalf("SearchNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
}

func TestNoteService_ResolveEntityID_UnsupportedType(t *testing.T) {
	svc := NewNoteService(&mockNoteEntityNoteRepo{}, &mockNoteEpicRepo{}, &mockNoteFeatureRepo{}, &mockNoteTaskRepo{})

	_, err := svc.AddNote(context.Background(), models.EntityType("unknown"), "key", "comment", "Test", "agent")
	if err == nil {
		t.Fatal("expected error for unsupported entity type")
	}
}
