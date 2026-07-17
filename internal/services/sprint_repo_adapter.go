package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// SprintAdapterRepository defines the minimal interface needed by
// SprintRepositoryAdapter. Mirrors the BugAdapterRepository / TechDebtAdapterRepository
// pattern: only the methods actually used by polymorphic cross-cutting
// services (NoteService.AddNote, NoteService.GetEntityDetails, etc.) are
// declared here.
//
// The sprints table does not carry a context_data column, so
// GetContextData/UpdateContextData are NOT part of this interface; the
// adapter returns a "not supported" error for those callers instead of
// delegating to the underlying repository.
type SprintAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Sprint, error)
	GetByID(ctx context.Context, id int64) (*models.Sprint, error)
	Update(ctx context.Context, sprint *models.Sprint) error
	UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error
}

// SprintRepositoryAdapter wraps a typed sprint repository to satisfy
// EntityRepository so sprints can be registered in EntityRegistry.
//
// Added for B030: `shark create note S###` failed because the registry had
// no repository registered for entity type "sprint". This adapter is the
// minimal wrapper required to plumb sprint key resolution through
// NoteService (and other future cross-cutting services).
type SprintRepositoryAdapter struct {
	repo SprintAdapterRepository
}

// Compile-time check that SprintRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*SprintRepositoryAdapter)(nil)

// NewSprintRepositoryAdapter creates an adapter wrapping the given sprint
// repository.
func NewSprintRepositoryAdapter(repo SprintAdapterRepository) *SprintRepositoryAdapter {
	return &SprintRepositoryAdapter{repo: repo}
}

// GetByKey delegates to the typed sprint repository and lifts the result
// into the models.Entity interface.
func (a *SprintRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

// GetByID delegates to the typed sprint repository and lifts the result
// into the models.Entity interface.
func (a *SprintRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

// UpdateStatus updates the sprint status; the string is cast to the typed
// SprintStatus before being passed to the underlying repository.
func (a *SprintRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.repo.UpdateStatus(ctx, id, models.SprintStatus(status))
}

func (a *SprintRepositoryAdapter) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	sprint, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("SprintRepositoryAdapter.UpdateStatusIfCurrent: load: %w", err)
	}
	if sprint == nil || !strings.EqualFold(string(sprint.Status), expectedCurrentStatus) {
		return false, nil
	}
	if err := a.repo.UpdateStatus(ctx, id, models.SprintStatus(newStatus)); err != nil {
		return false, err
	}
	return true, nil
}

// Update persists all fields of the sprint. The entity parameter must be
// the concrete *models.Sprint type for this adapter.
func (a *SprintRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	sprint, ok := entity.(*models.Sprint)
	if !ok {
		return fmt.Errorf("SprintRepositoryAdapter.Update: expected *models.Sprint, got %T", entity)
	}
	return a.repo.Update(ctx, sprint)
}

// GetContextData returns an error: the sprints table has no context_data
// column. Callers that need polymorphic context access should branch on
// entity type, or we can add the column later (E19-followup) without
// changing this contract.
func (a *SprintRepositoryAdapter) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, fmt.Errorf("SprintRepositoryAdapter: context_data is not supported for sprints")
}

// UpdateContextData returns an error: the sprints table has no
// context_data column. See GetContextData for rationale.
func (a *SprintRepositoryAdapter) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return fmt.Errorf("SprintRepositoryAdapter: context_data is not supported for sprints")
}
