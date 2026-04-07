package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TechDebtAdapterRepository defines the minimal interface needed by TechDebtRepositoryAdapter.
type TechDebtAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.TechDebt, error)
	GetByID(ctx context.Context, id int64) (*models.TechDebt, error)
	Update(ctx context.Context, td *models.TechDebt) error
	UpdateStatus(ctx context.Context, id int64, status models.TechDebtStatus) error
	GetContextData(ctx context.Context, id int64) (*string, error)
	UpdateContextData(ctx context.Context, id int64, contextData *string) error
}

// TechDebtRepositoryAdapter wraps a typed tech-debt repository to satisfy EntityRepository.
type TechDebtRepositoryAdapter struct {
	repo TechDebtAdapterRepository
}

// Compile-time check that TechDebtRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*TechDebtRepositoryAdapter)(nil)

// NewTechDebtRepositoryAdapter creates an adapter wrapping the given tech-debt repository.
func NewTechDebtRepositoryAdapter(repo TechDebtAdapterRepository) *TechDebtRepositoryAdapter {
	return &TechDebtRepositoryAdapter{repo: repo}
}

func (a *TechDebtRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *TechDebtRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *TechDebtRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.repo.UpdateStatus(ctx, id, models.TechDebtStatus(status))
}

func (a *TechDebtRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	td, ok := entity.(*models.TechDebt)
	if !ok {
		return fmt.Errorf("TechDebtRepositoryAdapter.Update: expected *models.TechDebt, got %T", entity)
	}
	return a.repo.Update(ctx, td)
}

func (a *TechDebtRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	return a.repo.GetContextData(ctx, id)
}

func (a *TechDebtRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return a.repo.UpdateContextData(ctx, id, data)
}
