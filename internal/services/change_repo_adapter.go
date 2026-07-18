package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ChangeCardAdapterRepository defines the minimal interface needed by ChangeCardRepositoryAdapter.
type ChangeCardAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
	GetByID(ctx context.Context, id int64) (*models.ChangeCard, error)
	Update(ctx context.Context, card *models.ChangeCard) error
	UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.ChangeCardStatus, newStatus models.ChangeCardStatus) (bool, error)
	GetContextData(ctx context.Context, id int64) (*string, error)
	UpdateContextData(ctx context.Context, id int64, contextData *string) error
}

// ChangeCardRepositoryAdapter wraps a typed change-card repository to satisfy EntityRepository.
type ChangeCardRepositoryAdapter struct {
	repo ChangeCardAdapterRepository
}

// Compile-time check that ChangeCardRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*ChangeCardRepositoryAdapter)(nil)

// NewChangeCardRepositoryAdapter creates an adapter wrapping the given change-card repository.
func NewChangeCardRepositoryAdapter(repo ChangeCardAdapterRepository) *ChangeCardRepositoryAdapter {
	return &ChangeCardRepositoryAdapter{repo: repo}
}

func (a *ChangeCardRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *ChangeCardRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *ChangeCardRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.repo.UpdateStatus(ctx, id, models.ChangeCardStatus(status))
}

func (a *ChangeCardRepositoryAdapter) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	return a.repo.UpdateStatusIfCurrent(ctx, id, models.ChangeCardStatus(expectedCurrentStatus), models.ChangeCardStatus(newStatus))
}

func (a *ChangeCardRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	card, ok := entity.(*models.ChangeCard)
	if !ok {
		return fmt.Errorf("ChangeCardRepositoryAdapter.Update: expected *models.ChangeCard, got %T", entity)
	}
	return a.repo.Update(ctx, card)
}

func (a *ChangeCardRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	return a.repo.GetContextData(ctx, id)
}

func (a *ChangeCardRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return a.repo.UpdateContextData(ctx, id, data)
}
