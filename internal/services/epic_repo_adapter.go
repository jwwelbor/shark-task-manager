package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicAdapterRepository defines the minimal interface needed by
// NewEpicRepositoryAdapter. It combines methods from EpicRepository and
// ContextEpicRepository.
type EpicAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
	Update(ctx context.Context, epic *models.Epic) error
	UpdateStatus(ctx context.Context, epicID int64, status models.EpicStatus) error
	UpdateStatusIfCurrent(ctx context.Context, epicID int64, expectedStatus models.EpicStatus, newStatus models.EpicStatus) (bool, error)
	GetContextData(ctx context.Context, epicID int64) (*string, error)
	UpdateContextData(ctx context.Context, epicID int64, contextData *string) error
}

// NewEpicRepositoryAdapter creates an EntityRepository adapter for epics.
func NewEpicRepositoryAdapter(repo EpicAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.Epic, models.EpicStatus]("Epic", repo)
}
