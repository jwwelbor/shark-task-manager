package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ChangeCardAdapterRepository defines the minimal interface needed by
// NewChangeCardRepositoryAdapter.
type ChangeCardAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
	GetByID(ctx context.Context, id int64) (*models.ChangeCard, error)
	Update(ctx context.Context, card *models.ChangeCard) error
	UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.ChangeCardStatus, newStatus models.ChangeCardStatus) (bool, error)
	GetContextData(ctx context.Context, id int64) (*string, error)
	UpdateContextData(ctx context.Context, id int64, contextData *string) error
}

// NewChangeCardRepositoryAdapter creates an EntityRepository adapter for change cards.
func NewChangeCardRepositoryAdapter(repo ChangeCardAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.ChangeCard, models.ChangeCardStatus]("ChangeCard", repo)
}
