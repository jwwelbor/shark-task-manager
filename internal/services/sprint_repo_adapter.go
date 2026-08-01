package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// SprintAdapterRepository defines the minimal interface needed by
// NewSprintRepositoryAdapter: only the methods actually used by
// polymorphic cross-cutting services (NoteService.AddNote,
// NoteService.GetEntityDetails, etc.) are declared here.
//
// The sprints table does not carry a context_data column, so
// GetContextData/UpdateContextData are NOT part of this interface; the
// generic adapter reports "not supported" for those callers instead of
// delegating to the underlying repository.
//
// Added for B030: `shark create note S###` failed because the registry had
// no repository registered for entity type "sprint".
type SprintAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Sprint, error)
	GetByID(ctx context.Context, id int64) (*models.Sprint, error)
	Update(ctx context.Context, sprint *models.Sprint) error
	UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.SprintStatus, newStatus models.SprintStatus) (bool, error)
}

// NewSprintRepositoryAdapter creates an EntityRepository adapter for sprints.
func NewSprintRepositoryAdapter(repo SprintAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.Sprint, models.SprintStatus]("Sprint", repo)
}
