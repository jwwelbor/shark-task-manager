package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// BugAdapterRepository defines the minimal interface needed by BugRepositoryAdapter.
type BugAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
	GetByID(ctx context.Context, id int64) (*models.Bug, error)
	Update(ctx context.Context, bug *models.Bug) error
	UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error
}

// BugRepositoryAdapter wraps a typed bug repository to satisfy EntityRepository.
// GetContextData and UpdateContextData use the get-field-update pattern because
// BugRepository does not expose dedicated context data methods.
type BugRepositoryAdapter struct {
	repo BugAdapterRepository
}

// Compile-time check that BugRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*BugRepositoryAdapter)(nil)

// NewBugRepositoryAdapter creates an adapter wrapping the given bug repository.
func NewBugRepositoryAdapter(repo BugAdapterRepository) *BugRepositoryAdapter {
	return &BugRepositoryAdapter{repo: repo}
}

func (a *BugRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *BugRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *BugRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.repo.UpdateStatus(ctx, id, models.BugStatus(status))
}

func (a *BugRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	bug, ok := entity.(*models.Bug)
	if !ok {
		return fmt.Errorf("BugRepositoryAdapter.Update: expected *models.Bug, got %T", entity)
	}
	return a.repo.Update(ctx, bug)
}

func (a *BugRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	bug, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return bug.ContextData, nil
}

func (a *BugRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	bug, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	bug.ContextData = data
	return a.repo.Update(ctx, bug)
}
