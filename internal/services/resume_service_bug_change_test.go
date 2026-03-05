package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Mock implementations for bug/change resume dependencies

type mockResumeBugRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.Bug, error)
}

func (m *mockResumeBugRepo) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.Bug{ID: 10, Key: key, Title: "Test Bug", Status: "open"}, nil
}

type mockResumeChangeCardRepo struct {
	getByKeyFunc func(ctx context.Context, key string) (*models.ChangeCard, error)
}

func (m *mockResumeChangeCardRepo) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return &models.ChangeCard{ID: 20, Key: "CC-001", Title: "Test Change"}, nil
}

// TestResumeService_GetBugResume_ReturnsContext tests bug resume context retrieval.
func TestResumeService_GetBugResume_ReturnsContext(t *testing.T) {
	ctx := context.Background()
	bugKey := "B001"

	mockBugRepo := &mockResumeBugRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Bug, error) {
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

	svc := NewResumeService(nil, nil, nil, noteRepo)
	svc.SetBugRepo(mockBugRepo)

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

	mockBugRepo := &mockResumeBugRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found: %s", key)
		},
	}

	noteRepo := &mockResumeNoteRepo{}

	svc := NewResumeService(nil, nil, nil, noteRepo)
	svc.SetBugRepo(mockBugRepo)

	_, err := svc.GetBugResume(ctx, "B999")
	if err == nil {
		t.Error("GetBugResume() with non-existent bug should return error")
	}
}

// TestResumeService_GetBugResume_NilBugRepo tests graceful degradation when bugRepo is nil.
func TestResumeService_GetBugResume_NilBugRepo(t *testing.T) {
	ctx := context.Background()

	noteRepo := &mockResumeNoteRepo{}

	svc := NewResumeService(nil, nil, nil, noteRepo)
	// Do NOT set bug repo - it should remain nil

	_, err := svc.GetBugResume(ctx, "B001")
	if err == nil {
		t.Error("GetBugResume() with nil bugRepo should return error")
	}
	if err != nil && !containsString(err.Error(), "bug repository") {
		t.Errorf("GetBugResume() error should mention 'bug repository', got: %v", err)
	}
}

// TestResumeService_GetBugResume_IncludesNotes tests that notes are loaded.
func TestResumeService_GetBugResume_IncludesNotes(t *testing.T) {
	ctx := context.Background()

	mockBugRepo := &mockResumeBugRepo{}

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

	svc := NewResumeService(nil, nil, nil, noteRepo)
	svc.SetBugRepo(mockBugRepo)

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

	mockChangeRepo := &mockResumeChangeCardRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
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

	svc := NewResumeService(nil, nil, nil, noteRepo)
	svc.SetChangeCardRepo(mockChangeRepo)

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

	mockChangeRepo := &mockResumeChangeCardRepo{
		getByKeyFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("change not found: %s", key)
		},
	}

	noteRepo := &mockResumeNoteRepo{}

	svc := NewResumeService(nil, nil, nil, noteRepo)
	svc.SetChangeCardRepo(mockChangeRepo)

	_, err := svc.GetChangeResume(ctx, "C999")
	if err == nil {
		t.Error("GetChangeResume() with non-existent change should return error")
	}
}

// TestResumeService_GetChangeResume_NilChangeRepo tests graceful degradation when changeCardRepo is nil.
func TestResumeService_GetChangeResume_NilChangeRepo(t *testing.T) {
	ctx := context.Background()

	noteRepo := &mockResumeNoteRepo{}

	svc := NewResumeService(nil, nil, nil, noteRepo)
	// Do NOT set change card repo

	_, err := svc.GetChangeResume(ctx, "C001")
	if err == nil {
		t.Error("GetChangeResume() with nil changeCardRepo should return error")
	}
	if err != nil && !containsString(err.Error(), "change") {
		t.Errorf("GetChangeResume() error should mention 'change', got: %v", err)
	}
}

// TestResumeService_GetChangeResume_IncludesNotes tests that notes are loaded for changes.
func TestResumeService_GetChangeResume_IncludesNotes(t *testing.T) {
	ctx := context.Background()

	mockChangeRepo := &mockResumeChangeCardRepo{}

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

	svc := NewResumeService(nil, nil, nil, noteRepo)
	svc.SetChangeCardRepo(mockChangeRepo)

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

// mockResumeNoteRepo is a mock for ResumeEntityNoteRepository.
// Defined here to allow tests in this file; uses lowercase to stay package-private.
type mockResumeNoteRepo struct {
	getByEntityFunc func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

func (m *mockResumeNoteRepo) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
	if m.getByEntityFunc != nil {
		return m.getByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}
