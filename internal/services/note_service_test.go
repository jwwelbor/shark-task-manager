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

// mockNoteEntityRepo provides a configurable mock EntityRepository for NoteService tests.
type mockNoteEntityRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (models.Entity, error)
	getByIDFunc  func(ctx context.Context, id int64) (models.Entity, error)
}

func (m *mockNoteEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented")
}

func (m *mockNoteEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented")
}

func (m *mockNoteEntityRepo) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}

func (m *mockNoteEntityRepo) Update(_ context.Context, _ models.Entity) error {
	return nil
}

func (m *mockNoteEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}

func (m *mockNoteEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

// newNoteTestRegistry creates a registry with all 5 entity types using the given defaults.
func newNoteTestRegistry() *EntityRegistry {
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test Epic"}}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: id, Key: "E01", Title: "Test Epic"}}, nil
		},
	})
	reg.Register(models.EntityTypeFeature, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 2, Key: key, Title: "Test Feature"}}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: id, Key: "E01-F01", Title: "Test Feature"}}, nil
		},
	})
	reg.Register(models.EntityTypeTask, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: 3, Key: key, Title: "Test Task"}}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: id, Key: "E01-F01-001", Title: "Test Task"}}, nil
		},
	})
	reg.Register(models.EntityTypeBug, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 4, Key: key, Title: "Test Bug"}}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: id, Key: "B001", Title: "Test Bug"}}, nil
		},
	})
	reg.Register(models.EntityTypeChange, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 5, Key: key, Title: "Test Change"}}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: id, Key: "CC-001", Title: "Test Change"}}, nil
		},
	})
	// B030: sprints and ideas join the registry so polymorphic note
	// operations on S### / I-YYYY-MM-DD-## keys resolve correctly.
	reg.Register(models.EntityTypeSprint, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Sprint{ID: 6, Key: key, Name: "Test Sprint", Status: "planning"}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.Sprint{ID: id, Key: "S001", Name: "Test Sprint", Status: "planning"}, nil
		},
	})
	reg.Register(models.EntityTypeIdea, &mockNoteEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Idea{ID: 7, Key: key, Title: "Test Idea", Status: models.IdeaStatusNew}, nil
		},
		getByIDFunc: func(_ context.Context, id int64) (models.Entity, error) {
			return &models.Idea{ID: id, Key: "I-2026-05-11-01", Title: "Test Idea", Status: models.IdeaStatusNew}, nil
		},
	})
	return reg
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
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

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
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

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
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

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

func TestNoteService_AddNote_AllEntityTypes(t *testing.T) {
	entityTypes := []struct {
		entityType models.EntityType
		key        string
		expectedID int64
	}{
		{models.EntityTypeEpic, "E01", 1},
		{models.EntityTypeFeature, "E01-F01", 2},
		{models.EntityTypeTask, "E01-F01-001", 3},
		{models.EntityTypeBug, "B001", 4},
		{models.EntityTypeChange, "CC-001", 5},
		// B030: sprint and idea note operations must succeed end-to-end
		// (registry resolution -> repo.GetByKey -> note.Create).
		{models.EntityTypeSprint, "S001", 6},
		{models.EntityTypeIdea, "I-2026-05-11-01", 7},
	}

	for _, et := range entityTypes {
		t.Run(string(et.entityType), func(t *testing.T) {
			noteRepo := &mockNoteEntityNoteRepo{
				createFunc: func(ctx context.Context, note *models.EntityNote) error {
					note.ID = 100
					if note.EntityID != et.expectedID {
						t.Errorf("expected entity ID %d, got %d", et.expectedID, note.EntityID)
					}
					if note.EntityType != et.entityType {
						t.Errorf("expected entity type %s, got %s", et.entityType, note.EntityType)
					}
					return nil
				},
			}
			svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
			if err != nil {
				t.Fatalf("NewNoteService() unexpected error: %v", err)
			}

			note, err := svc.AddNote(context.Background(), et.entityType, et.key, "comment", "test content", "agent")
			if err != nil {
				t.Fatalf("AddNote() error = %v", err)
			}
			if note.EntityID != et.expectedID {
				t.Errorf("expected entity ID %d, got %d", et.expectedID, note.EntityID)
			}
		})
	}
}

func TestNoteService_GetEntityDetails_AllEntityTypes(t *testing.T) {
	registry := newNoteTestRegistry()
	svc, err := NewNoteService(&mockNoteEntityNoteRepo{}, registry)
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	tests := []struct {
		entityType models.EntityType
		entityID   int64
	}{
		{models.EntityTypeEpic, 1},
		{models.EntityTypeFeature, 2},
		{models.EntityTypeTask, 3},
		{models.EntityTypeBug, 4},
		{models.EntityTypeChange, 5},
		// B030: sprint and idea entities expose their key/title via the
		// new Entity interface accessors -- GetEntityDetails must succeed.
		{models.EntityTypeSprint, 6},
		{models.EntityTypeIdea, 7},
	}

	for _, tt := range tests {
		t.Run(string(tt.entityType), func(t *testing.T) {
			details := svc.GetEntityDetails(context.Background(), tt.entityType, tt.entityID)
			if details == nil {
				t.Fatal("expected non-nil details")
			}
			if details.Key == "" {
				t.Error("expected non-empty key")
			}
			if details.Name == "" {
				t.Error("expected non-empty name")
			}
		})
	}
}

// TestNoteService_AddNote_B030_SprintAndIdea is a focused regression for
// B030: before the fix, sprint and idea entity types could not be the
// target of `shark create note` -- the key parser rejected S### outright,
// and the registry rejected I-* with "no repository registered for entity
// type 'idea'". This test pins both code paths to "succeed" via the
// registered mock adapters.
func TestNoteService_AddNote_B030_SprintAndIdea(t *testing.T) {
	cases := []struct {
		name       string
		entityType models.EntityType
		key        string
		wantID     int64
	}{
		{"sprint S001", models.EntityTypeSprint, "S001", 6},
		{"idea I-2026-05-11-01", models.EntityTypeIdea, "I-2026-05-11-01", 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			noteRepo := &mockNoteEntityNoteRepo{
				createFunc: func(_ context.Context, n *models.EntityNote) error {
					if n.EntityType != tc.entityType {
						t.Errorf("note.EntityType = %q, want %q", n.EntityType, tc.entityType)
					}
					if n.EntityID != tc.wantID {
						t.Errorf("note.EntityID = %d, want %d", n.EntityID, tc.wantID)
					}
					n.ID = 1234
					return nil
				},
			}
			svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
			if err != nil {
				t.Fatalf("NewNoteService error: %v", err)
			}
			note, err := svc.AddNote(context.Background(), tc.entityType, tc.key, "comment", "B030 fix verified", "tester")
			if err != nil {
				t.Fatalf("AddNote(%s, %s) error = %v (B030 regression)", tc.entityType, tc.key, err)
			}
			if note.ID != 1234 {
				t.Errorf("note.ID = %d, want 1234", note.ID)
			}
		})
	}
}

func TestNoteService_AddNote_InvalidKey(t *testing.T) {
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockNoteEntityRepo{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, fmt.Errorf("not found")
		},
	})
	svc, err := NewNoteService(&mockNoteEntityNoteRepo{}, reg)
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	_, err = svc.AddNote(context.Background(), models.EntityTypeEpic, "E999", "comment", "Test", "agent")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestNoteService_AddNote_InvalidNoteType(t *testing.T) {
	svc, err := NewNoteService(&mockNoteEntityNoteRepo{}, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	_, err = svc.AddNote(context.Background(), models.EntityTypeEpic, "E16", "invalid_type", "Test", "agent")
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
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

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
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

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
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	notes, err := svc.SearchNotes(context.Background(), "API", nil, nil, "", "")
	if err != nil {
		t.Fatalf("SearchNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
}

func TestNoteService_ResolveEntityID_UnsupportedType(t *testing.T) {
	svc, err := NewNoteService(&mockNoteEntityNoteRepo{}, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	_, err = svc.AddNote(context.Background(), models.EntityType("unknown"), "key", "comment", "Test", "agent")
	if err == nil {
		t.Fatal("expected error for unsupported entity type")
	}
}

func TestNoteService_NilRegistry_ReturnsError(t *testing.T) {
	svc, err := NewNoteService(&mockNoteEntityNoteRepo{}, nil)
	if err == nil {
		t.Fatal("expected error for nil registry, got nil")
	}
	if svc != nil {
		t.Error("expected nil service when registry is nil")
	}
	if err.Error() != "NoteService: EntityRegistry must not be nil" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Review-finding notes carry structured metadata so review effectiveness is
// queryable (gate/round/severity/defect_class/fingerprint) instead of buried
// in free text. Metadata must be a JSON object; anything else is rejected
// before the repo is touched.
func TestNoteService_AddNoteWithMetadata(t *testing.T) {
	var stored *models.EntityNote
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(ctx context.Context, note *models.EntityNote) error {
			note.ID = 99
			stored = note
			return nil
		},
	}
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	meta := `{"gate":"qa","round":2,"severity":"high","defect_class":"unguarded dereference","fingerprint":"svc.go:Render:deref"}`
	note, err := svc.AddNoteWithMetadata(context.Background(), models.EntityTypeEpic, "E16", "review-finding", "Unguarded deref in Render", "sonnet", meta)
	if err != nil {
		t.Fatalf("AddNoteWithMetadata() error = %v", err)
	}
	if note.Metadata == nil || *stored.Metadata != meta {
		t.Errorf("metadata not stored verbatim: %+v", note.Metadata)
	}

	// Invalid JSON is rejected before any write.
	stored = nil
	if _, err := svc.AddNoteWithMetadata(context.Background(), models.EntityTypeEpic, "E16", "review-finding", "x", "", "{not json"); err == nil {
		t.Fatal("expected error for invalid metadata JSON")
	}
	if stored != nil {
		t.Error("repo must not be called when metadata is invalid")
	}

	// A non-object JSON value (array) is also rejected — metadata is a field map.
	if _, err := svc.AddNoteWithMetadata(context.Background(), models.EntityTypeEpic, "E16", "review-finding", "x", "", `["a"]`); err == nil {
		t.Fatal("expected error for non-object metadata JSON")
	}

	// AddNote (no metadata) still works and stores NULL metadata.
	note, err = svc.AddNote(context.Background(), models.EntityTypeEpic, "E16", "review-finding", "plain", "")
	if err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if note.Metadata != nil {
		t.Errorf("expected nil metadata, got %v", *note.Metadata)
	}
}
