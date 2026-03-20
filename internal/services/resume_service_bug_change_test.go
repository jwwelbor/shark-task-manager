package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// mockResumeEntityRepo provides a configurable mock EntityRepository for ResumeService tests.
type mockResumeEntityRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (models.Entity, error)
}

func (m *mockResumeEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented")
}

func (m *mockResumeEntityRepo) GetByID(_ context.Context, _ int64) (models.Entity, error) {
	return nil, nil
}

func (m *mockResumeEntityRepo) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}

func (m *mockResumeEntityRepo) Update(_ context.Context, _ models.Entity) error {
	return nil
}

func (m *mockResumeEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}

func (m *mockResumeEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

// newResumeTestRegistry creates a registry with bug and change entity types.
func newResumeTestRegistry(bugRepo, changeRepo EntityRepository) *EntityRegistry {
	reg := NewEntityRegistry()
	reg.Register(models.EntityTypeEpic, &mockResumeEntityRepo{})
	reg.Register(models.EntityTypeFeature, &mockResumeEntityRepo{})
	reg.Register(models.EntityTypeTask, &mockResumeEntityRepo{})
	if bugRepo != nil {
		reg.Register(models.EntityTypeBug, bugRepo)
	}
	if changeRepo != nil {
		reg.Register(models.EntityTypeChange, changeRepo)
	}
	return reg
}

// TestResumeService_GetBugResume_ReturnsContext tests bug resume context retrieval.
func TestResumeService_GetBugResume_ReturnsContext(t *testing.T) {
	ctx := context.Background()
	bugKey := "B001"

	bugRepo := &mockResumeEntityRepo{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			if key != bugKey {
				t.Errorf("GetByKey called with %q, want %q", key, bugKey)
			}
			return &models.Bug{
				ID:     10,
				Key:    key,
				Title:  "Login page crash",
				Status: "open",
			}, nil
		},
	}

	noteRepo := &mockResumeNoteRepo{}
	reg := newResumeTestRegistry(bugRepo, &mockResumeEntityRepo{})

	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	result, err := svc.GetBugResume(ctx, bugKey)
	if err != nil {
		t.Fatalf("GetBugResume() error = %v", err)
	}

	if result == nil {
		t.Fatal("GetBugResume() returned nil result")
	}

	if result.Bug == nil {
		t.Fatal("GetBugResume() result.Bug is nil")
	}

	if result.Bug.Key != bugKey {
		t.Errorf("GetBugResume() result.Bug.Key = %q, want %q", result.Bug.Key, bugKey)
	}

	if result.Bug.Title != "Login page crash" {
		t.Errorf("GetBugResume() result.Bug.Title = %q, want %q", result.Bug.Title, "Login page crash")
	}
}

// TestResumeService_GetBugResume_NotFound tests bug resume when bug does not exist.
func TestResumeService_GetBugResume_NotFound(t *testing.T) {
	ctx := context.Background()

	bugRepo := &mockResumeEntityRepo{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, fmt.Errorf("bug not found: %s", key)
		},
	}

	noteRepo := &mockResumeNoteRepo{}
	reg := newResumeTestRegistry(bugRepo, &mockResumeEntityRepo{})

	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	_, err := svc.GetBugResume(ctx, "B999")
	if err == nil {
		t.Error("GetBugResume() with non-existent bug should return error")
	}
}

// TestResumeService_GetBugResume_UnregisteredType tests error when bug type not registered.
func TestResumeService_GetBugResume_UnregisteredType(t *testing.T) {
	ctx := context.Background()

	noteRepo := &mockResumeNoteRepo{}
	// Create registry without bug type
	reg := newResumeTestRegistry(nil, &mockResumeEntityRepo{})

	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	_, err := svc.GetBugResume(ctx, "B001")
	if err == nil {
		t.Error("GetBugResume() with unregistered bug type should return error")
	}
	if err != nil && !containsString(err.Error(), "bug support not configured") {
		t.Errorf("GetBugResume() error should mention 'bug support not configured', got: %v", err)
	}
}

// TestResumeService_GetBugResume_IncludesNotes tests that notes are loaded.
func TestResumeService_GetBugResume_IncludesNotes(t *testing.T) {
	ctx := context.Background()

	bugRepo := &mockResumeEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.Bug{ID: 10, Key: key, Title: "Test Bug", Status: "open"}, nil
		},
	}

	notesCalled := false
	noteRepo := &mockResumeNoteRepo{
		getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
			if entityType == models.EntityTypeBug {
				notesCalled = true
				return []*models.EntityNote{
					{ID: 1, Content: "Reproduced in staging"},
				}, nil
			}
			return nil, nil
		},
	}

	reg := newResumeTestRegistry(bugRepo, &mockResumeEntityRepo{})
	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	result, err := svc.GetBugResume(ctx, "B001")
	if err != nil {
		t.Fatalf("GetBugResume() error = %v", err)
	}

	if !notesCalled {
		t.Error("GetBugResume() should call note repo with EntityTypeBug")
	}

	if len(result.Notes) == 0 {
		t.Error("GetBugResume() should include notes in result")
	}
}

// TestResumeService_GetChangeResume_ReturnsContext tests change-card resume context retrieval.
func TestResumeService_GetChangeResume_ReturnsContext(t *testing.T) {
	ctx := context.Background()
	changeKey := "C001"

	changeRepo := &mockResumeEntityRepo{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			if key != changeKey {
				t.Errorf("GetByKey called with %q, want %q", key, changeKey)
			}
			return &models.ChangeCard{
				ID:    20,
				Key:   "CC-001",
				Title: "Update authentication flow",
			}, nil
		},
	}

	noteRepo := &mockResumeNoteRepo{}
	reg := newResumeTestRegistry(&mockResumeEntityRepo{}, changeRepo)

	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	result, err := svc.GetChangeResume(ctx, changeKey)
	if err != nil {
		t.Fatalf("GetChangeResume() error = %v", err)
	}

	if result == nil {
		t.Fatal("GetChangeResume() returned nil result")
	}

	if result.ChangeCard == nil {
		t.Fatal("GetChangeResume() result.ChangeCard is nil")
	}

	if result.ChangeCard.Title != "Update authentication flow" {
		t.Errorf("GetChangeResume() result.ChangeCard.Title = %q, want %q",
			result.ChangeCard.Title, "Update authentication flow")
	}
}

// TestResumeService_GetChangeResume_NotFound tests change resume when change does not exist.
func TestResumeService_GetChangeResume_NotFound(t *testing.T) {
	ctx := context.Background()

	changeRepo := &mockResumeEntityRepo{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, fmt.Errorf("change not found: %s", key)
		},
	}

	noteRepo := &mockResumeNoteRepo{}
	reg := newResumeTestRegistry(&mockResumeEntityRepo{}, changeRepo)

	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	_, err := svc.GetChangeResume(ctx, "C999")
	if err == nil {
		t.Error("GetChangeResume() with non-existent change should return error")
	}
}

// TestResumeService_GetChangeResume_UnregisteredType tests error when change type not registered.
func TestResumeService_GetChangeResume_UnregisteredType(t *testing.T) {
	ctx := context.Background()

	noteRepo := &mockResumeNoteRepo{}
	// Create registry without change type
	reg := newResumeTestRegistry(&mockResumeEntityRepo{}, nil)

	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	_, err := svc.GetChangeResume(ctx, "C001")
	if err == nil {
		t.Error("GetChangeResume() with unregistered change type should return error")
	}
	if err != nil && !containsString(err.Error(), "change support not configured") {
		t.Errorf("GetChangeResume() error should mention 'change support not configured', got: %v", err)
	}
}

// TestResumeService_GetChangeResume_IncludesNotes tests that notes are loaded for changes.
func TestResumeService_GetChangeResume_IncludesNotes(t *testing.T) {
	ctx := context.Background()

	changeRepo := &mockResumeEntityRepo{
		getByKeyFunc: func(_ context.Context, key string) (models.Entity, error) {
			return &models.ChangeCard{ID: 20, Key: "CC-001", Title: "Test Change"}, nil
		},
	}

	notesCalled := false
	noteRepo := &mockResumeNoteRepo{
		getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
			if entityType == models.EntityTypeChange {
				notesCalled = true
				return []*models.EntityNote{
					{ID: 2, Content: "Approved by tech lead"},
				}, nil
			}
			return nil, nil
		},
	}

	reg := newResumeTestRegistry(&mockResumeEntityRepo{}, changeRepo)
	svc := NewResumeService(nil, nil, nil, noteRepo, reg)

	result, err := svc.GetChangeResume(ctx, "C001")
	if err != nil {
		t.Fatalf("GetChangeResume() error = %v", err)
	}

	if !notesCalled {
		t.Error("GetChangeResume() should call note repo with EntityTypeChange")
	}

	if len(result.Notes) == 0 {
		t.Error("GetChangeResume() should include notes in result")
	}
}

// TestResumeService_NilRegistry_Panics tests that nil registry panics at construction.
func TestResumeService_NilRegistry_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil registry")
		}
		msg, ok := r.(string)
		if !ok || msg != "ResumeService: EntityRegistry must not be nil" {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()

	NewResumeService(nil, nil, nil, &mockResumeNoteRepo{}, nil)
}

// mockResumeNoteRepo is a mock for ResumeEntityNoteRepository.
type mockResumeNoteRepo struct {
	getByEntityFunc func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

func (m *mockResumeNoteRepo) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
	if m.getByEntityFunc != nil {
		return m.getByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}
