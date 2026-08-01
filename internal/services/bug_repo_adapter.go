package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// BugAdapterRepository defines the minimal interface needed by
// NewBugRepositoryAdapter.
type BugAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
	GetByID(ctx context.Context, id int64) (*models.Bug, error)
	Update(ctx context.Context, bug *models.Bug) error
	UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expectedStatus models.BugStatus, newStatus models.BugStatus) (bool, error)
	GetContextData(ctx context.Context, id int64) (*string, error)
	UpdateContextData(ctx context.Context, id int64, contextData *string) error
}

// NewBugRepositoryAdapter creates an EntityRepository adapter for bugs.
func NewBugRepositoryAdapter(repo BugAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.Bug, models.BugStatus]("Bug", repo)
}
