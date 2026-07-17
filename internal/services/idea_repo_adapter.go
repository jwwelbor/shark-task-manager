package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// IdeaAdapterRepository defines the minimal interface needed by
// IdeaRepositoryAdapter. Mirrors the BugAdapterRepository /
// TechDebtAdapterRepository pattern: only the methods actually used by
// polymorphic cross-cutting services (NoteService.AddNote,
// NoteService.GetEntityDetails, etc.) are declared here.
//
// The ideas table does not carry a context_data column, so
// GetContextData/UpdateContextData are NOT part of this interface; the
// adapter returns a "not supported" error for those callers instead of
// delegating to the underlying repository.
type IdeaAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Idea, error)
	GetByID(ctx context.Context, id int64) (*models.Idea, error)
	Update(ctx context.Context, idea *models.Idea) error
}

// IdeaRepositoryAdapter wraps a typed idea repository to satisfy
// EntityRepository so ideas can be registered in EntityRegistry.
//
// Added for B030: `shark create note I-YYYY-MM-DD-##` failed with
// "EntityRegistry: no repository registered for entity type 'idea'".
// This adapter is the minimal wrapper required to plumb idea key
// resolution through NoteService (and other future cross-cutting
// services).
type IdeaRepositoryAdapter struct {
	repo IdeaAdapterRepository
}

// Compile-time check that IdeaRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*IdeaRepositoryAdapter)(nil)

// NewIdeaRepositoryAdapter creates an adapter wrapping the given idea
// repository.
func NewIdeaRepositoryAdapter(repo IdeaAdapterRepository) *IdeaRepositoryAdapter {
	return &IdeaRepositoryAdapter{repo: repo}
}

// GetByKey delegates to the typed idea repository and lifts the result
// into the models.Entity interface.
func (a *IdeaRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

// GetByID delegates to the typed idea repository and lifts the result
// into the models.Entity interface.
func (a *IdeaRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

// UpdateStatus updates the idea status by loading, mutating, and saving.
// The IdeaRepository does not expose a direct UpdateStatus, so we emulate
// it via Update -- callers that care about atomicity should use the
// IdeaService directly rather than going through this polymorphic shim.
func (a *IdeaRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	idea, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("IdeaRepositoryAdapter.UpdateStatus: load: %w", err)
	}
	idea.Status = models.IdeaStatus(status)
	return a.repo.Update(ctx, idea)
}

func (a *IdeaRepositoryAdapter) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	idea, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("IdeaRepositoryAdapter.UpdateStatusIfCurrent: load: %w", err)
	}
	if idea == nil || !strings.EqualFold(string(idea.Status), expectedCurrentStatus) {
		return false, nil
	}
	idea.Status = models.IdeaStatus(newStatus)
	if err := a.repo.Update(ctx, idea); err != nil {
		return false, err
	}
	return true, nil
}

// Update persists all fields of the idea. The entity parameter must be
// the concrete *models.Idea type for this adapter.
func (a *IdeaRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	idea, ok := entity.(*models.Idea)
	if !ok {
		return fmt.Errorf("IdeaRepositoryAdapter.Update: expected *models.Idea, got %T", entity)
	}
	return a.repo.Update(ctx, idea)
}

// GetContextData returns an error: the ideas table has no context_data
// column. Callers that need polymorphic context access should branch on
// entity type or add the column in a follow-up migration.
func (a *IdeaRepositoryAdapter) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, fmt.Errorf("IdeaRepositoryAdapter: context_data is not supported for ideas")
}

// UpdateContextData returns an error: the ideas table has no
// context_data column. See GetContextData for rationale.
func (a *IdeaRepositoryAdapter) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return fmt.Errorf("IdeaRepositoryAdapter: context_data is not supported for ideas")
}
