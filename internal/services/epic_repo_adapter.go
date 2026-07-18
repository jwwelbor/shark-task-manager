package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicAdapterRepository defines the minimal interface needed by EpicRepositoryAdapter.
// It combines methods from EpicRepository and ContextEpicRepository.
type EpicAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
	Update(ctx context.Context, epic *models.Epic) error
	UpdateStatus(ctx context.Context, epicID int64, status models.EpicStatus) error
	UpdateStatusIfCurrent(ctx context.Context, epicID int64, expectedStatus models.EpicStatus, newStatus models.EpicStatus) (bool, error)
	GetContextData(ctx context.Context, epicID int64) (*string, error)
	UpdateContextData(ctx context.Context, epicID int64, contextData *string) error
}

// EpicRepositoryAdapter wraps a typed epic repository to satisfy EntityRepository.
type EpicRepositoryAdapter struct {
	repo EpicAdapterRepository
}

// Compile-time check that EpicRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*EpicRepositoryAdapter)(nil)

// NewEpicRepositoryAdapter creates an adapter wrapping the given epic repository.
func NewEpicRepositoryAdapter(repo EpicAdapterRepository) *EpicRepositoryAdapter {
	return &EpicRepositoryAdapter{repo: repo}
}

func (a *EpicRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *EpicRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *EpicRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.repo.UpdateStatus(ctx, id, models.EpicStatus(status))
}

func (a *EpicRepositoryAdapter) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	return a.repo.UpdateStatusIfCurrent(ctx, id, models.EpicStatus(expectedCurrentStatus), models.EpicStatus(newStatus))
}

func (a *EpicRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	epic, ok := entity.(*models.Epic)
	if !ok {
		return fmt.Errorf("EpicRepositoryAdapter.Update: expected *models.Epic, got %T", entity)
	}
	return a.repo.Update(ctx, epic)
}

func (a *EpicRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	return a.repo.GetContextData(ctx, id)
}

func (a *EpicRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return a.repo.UpdateContextData(ctx, id, data)
}
