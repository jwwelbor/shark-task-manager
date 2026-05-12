package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ============================================================================
// B030 regression: sprint and idea adapter coverage.
//
// Before B030, `shark create note S###` failed at the key parser and
// `shark create note I-...` failed at the EntityRegistry. These tests pin
// the contract that:
//   - both adapters satisfy the EntityRepository interface (compile-time);
//   - GetByKey / GetByID flow lifts the typed models into models.Entity;
//   - Update delegates to the typed repo and rejects wrong types;
//   - GetContextData / UpdateContextData return a stable "not supported"
//     error for the two entity tables that lack a context_data column.
// ============================================================================

// ─── Sprint adapter ─────────────────────────────────────────────────────────

type fakeSprintAdapterRepo struct {
	getByKeyFn      func(ctx context.Context, key string) (*models.Sprint, error)
	getByIDFn       func(ctx context.Context, id int64) (*models.Sprint, error)
	updateFn        func(ctx context.Context, sprint *models.Sprint) error
	updateStatusFn  func(ctx context.Context, id int64, status models.SprintStatus) error
	updateStatusArg models.SprintStatus
}

func (f *fakeSprintAdapterRepo) GetByKey(ctx context.Context, key string) (*models.Sprint, error) {
	return f.getByKeyFn(ctx, key)
}
func (f *fakeSprintAdapterRepo) GetByID(ctx context.Context, id int64) (*models.Sprint, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeSprintAdapterRepo) Update(ctx context.Context, sprint *models.Sprint) error {
	return f.updateFn(ctx, sprint)
}
func (f *fakeSprintAdapterRepo) UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error {
	f.updateStatusArg = status
	if f.updateStatusFn != nil {
		return f.updateStatusFn(ctx, id, status)
	}
	return nil
}

func TestSprintRepositoryAdapter_GetByKey_LiftsToEntity(t *testing.T) {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	fake := &fakeSprintAdapterRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.Sprint, error) {
			return &models.Sprint{
				ID:        42,
				Key:       key,
				Name:      "Sprint Seven",
				Goal:      "Ship B030 fix",
				Status:    "active",
				Slug:      "sprint-seven",
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}
	adapter := NewSprintRepositoryAdapter(fake)

	entity, err := adapter.GetByKey(context.Background(), "S007")
	if err != nil {
		t.Fatalf("GetByKey error: %v", err)
	}
	if entity.GetID() != 42 {
		t.Errorf("GetID() = %d, want 42", entity.GetID())
	}
	if entity.GetKey() != "S007" {
		t.Errorf("GetKey() = %q, want %q", entity.GetKey(), "S007")
	}
	// Sprint stores its display label in Name, not Title -- the Entity
	// adapter must expose it via GetTitle (B030 regression).
	if entity.GetTitle() != "Sprint Seven" {
		t.Errorf("GetTitle() = %q, want %q", entity.GetTitle(), "Sprint Seven")
	}
	if entity.GetEntityType() != models.EntityTypeSprint {
		t.Errorf("GetEntityType() = %q, want %q", entity.GetEntityType(), models.EntityTypeSprint)
	}
}

func TestSprintRepositoryAdapter_UpdateStatus_CastsType(t *testing.T) {
	fake := &fakeSprintAdapterRepo{}
	adapter := NewSprintRepositoryAdapter(fake)

	if err := adapter.UpdateStatus(context.Background(), 1, "completed"); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if fake.updateStatusArg != models.SprintStatus("completed") {
		t.Errorf("UpdateStatus passed %q, want %q", fake.updateStatusArg, "completed")
	}
}

func TestSprintRepositoryAdapter_Update_RejectsWrongType(t *testing.T) {
	adapter := NewSprintRepositoryAdapter(&fakeSprintAdapterRepo{})
	err := adapter.Update(context.Background(), &models.Bug{})
	if err == nil {
		t.Fatal("expected type error for non-sprint entity, got nil")
	}
}

func TestSprintRepositoryAdapter_ContextData_NotSupported(t *testing.T) {
	adapter := NewSprintRepositoryAdapter(&fakeSprintAdapterRepo{})
	if _, err := adapter.GetContextData(context.Background(), 1); err == nil {
		t.Error("expected GetContextData to return error for sprint")
	}
	if err := adapter.UpdateContextData(context.Background(), 1, nil); err == nil {
		t.Error("expected UpdateContextData to return error for sprint")
	}
}

// ─── Idea adapter ───────────────────────────────────────────────────────────

type fakeIdeaAdapterRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Idea, error)
	getByIDFn  func(ctx context.Context, id int64) (*models.Idea, error)
	updateFn   func(ctx context.Context, idea *models.Idea) error
	updated    *models.Idea
}

func (f *fakeIdeaAdapterRepo) GetByKey(ctx context.Context, key string) (*models.Idea, error) {
	return f.getByKeyFn(ctx, key)
}
func (f *fakeIdeaAdapterRepo) GetByID(ctx context.Context, id int64) (*models.Idea, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeIdeaAdapterRepo) Update(ctx context.Context, idea *models.Idea) error {
	f.updated = idea
	if f.updateFn != nil {
		return f.updateFn(ctx, idea)
	}
	return nil
}

func TestIdeaRepositoryAdapter_GetByKey_LiftsToEntity(t *testing.T) {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	fake := &fakeIdeaAdapterRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:        13,
				Key:       key,
				Title:     "Adopt typed feature flags",
				Status:    models.IdeaStatusNew,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}
	adapter := NewIdeaRepositoryAdapter(fake)

	entity, err := adapter.GetByKey(context.Background(), "I-2026-05-11-01")
	if err != nil {
		t.Fatalf("GetByKey error: %v", err)
	}
	if entity.GetID() != 13 {
		t.Errorf("GetID() = %d, want 13", entity.GetID())
	}
	if entity.GetKey() != "I-2026-05-11-01" {
		t.Errorf("GetKey() = %q, want %q", entity.GetKey(), "I-2026-05-11-01")
	}
	if entity.GetTitle() != "Adopt typed feature flags" {
		t.Errorf("GetTitle() = %q, want %q", entity.GetTitle(), "Adopt typed feature flags")
	}
	if entity.GetEntityType() != models.EntityTypeIdea {
		t.Errorf("GetEntityType() = %q, want %q", entity.GetEntityType(), models.EntityTypeIdea)
	}
}

func TestIdeaRepositoryAdapter_UpdateStatus_LoadsThenUpdates(t *testing.T) {
	idea := &models.Idea{ID: 9, Key: "I-2026-05-11-02", Title: "x", Status: models.IdeaStatusNew}
	fake := &fakeIdeaAdapterRepo{
		getByIDFn: func(_ context.Context, _ int64) (*models.Idea, error) {
			return idea, nil
		},
	}
	adapter := NewIdeaRepositoryAdapter(fake)

	if err := adapter.UpdateStatus(context.Background(), 9, "archived"); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if fake.updated == nil {
		t.Fatal("expected Update to be invoked, got nil")
	}
	if fake.updated.Status != models.IdeaStatusArchived {
		t.Errorf("Status = %q, want %q", fake.updated.Status, models.IdeaStatusArchived)
	}
}

func TestIdeaRepositoryAdapter_UpdateStatus_PropagatesLoadError(t *testing.T) {
	loadErr := errors.New("idea not found")
	fake := &fakeIdeaAdapterRepo{
		getByIDFn: func(_ context.Context, _ int64) (*models.Idea, error) { return nil, loadErr },
	}
	adapter := NewIdeaRepositoryAdapter(fake)

	err := adapter.UpdateStatus(context.Background(), 1, "archived")
	if !errors.Is(err, loadErr) {
		t.Errorf("expected wrapped loadErr, got %v", err)
	}
}

func TestIdeaRepositoryAdapter_Update_RejectsWrongType(t *testing.T) {
	adapter := NewIdeaRepositoryAdapter(&fakeIdeaAdapterRepo{})
	err := adapter.Update(context.Background(), &models.Bug{})
	if err == nil {
		t.Fatal("expected type error for non-idea entity, got nil")
	}
}

func TestIdeaRepositoryAdapter_ContextData_NotSupported(t *testing.T) {
	adapter := NewIdeaRepositoryAdapter(&fakeIdeaAdapterRepo{})
	if _, err := adapter.GetContextData(context.Background(), 1); err == nil {
		t.Error("expected GetContextData to return error for idea")
	}
	if err := adapter.UpdateContextData(context.Background(), 1, nil); err == nil {
		t.Error("expected UpdateContextData to return error for idea")
	}
}
