package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// IdeaAdapterRepository defines the minimal interface needed by
// NewIdeaRepositoryAdapter: only the methods actually used by polymorphic
// cross-cutting services (NoteService.AddNote, NoteService.GetEntityDetails,
// etc.) are declared here.
//
// IdeaRepository exposes no direct UpdateStatus method, so the generic
// adapter emulates it via GetByID + models.Entity.SetStatus + Update.
//
// The ideas table does not carry a context_data column, so
// GetContextData/UpdateContextData are NOT part of this interface; the
// generic adapter reports "not supported" for those callers instead of
// delegating to the underlying repository.
//
// Added for B030: `shark create note I-YYYY-MM-DD-##` failed with
// "EntityRegistry: no repository registered for entity type 'idea'".
type IdeaAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Idea, error)
	GetByID(ctx context.Context, id int64) (*models.Idea, error)
	Update(ctx context.Context, idea *models.Idea) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.IdeaStatus, newStatus models.IdeaStatus) (bool, error)
}

// NewIdeaRepositoryAdapter creates an EntityRepository adapter for ideas.
func NewIdeaRepositoryAdapter(repo IdeaAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.Idea, models.IdeaStatus]("Idea", repo)
}
