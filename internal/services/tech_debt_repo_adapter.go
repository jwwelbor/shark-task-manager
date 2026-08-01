package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TechDebtAdapterRepository defines the minimal interface needed by
// NewTechDebtRepositoryAdapter.
type TechDebtAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.TechDebt, error)
	GetByID(ctx context.Context, id int64) (*models.TechDebt, error)
	Update(ctx context.Context, td *models.TechDebt) error
	UpdateStatus(ctx context.Context, id int64, status models.TechDebtStatus) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.TechDebtStatus, newStatus models.TechDebtStatus) (bool, error)
	GetContextData(ctx context.Context, id int64) (*string, error)
	UpdateContextData(ctx context.Context, id int64, contextData *string) error
}

// NewTechDebtRepositoryAdapter creates an EntityRepository adapter for tech debts.
func NewTechDebtRepositoryAdapter(repo TechDebtAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.TechDebt, models.TechDebtStatus]("TechDebt", repo)
}
