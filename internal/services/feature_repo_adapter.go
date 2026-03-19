package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// FeatureAdapterRepository defines the minimal interface needed by FeatureRepositoryAdapter.
type FeatureAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
	GetContextData(ctx context.Context, featureID int64) (*string, error)
	UpdateContextData(ctx context.Context, featureID int64, contextData *string) error
}

// FeatureRepositoryAdapter wraps a typed feature repository to satisfy EntityRepository.
// UpdateStatus uses the get-set-update pattern because FeatureRepository does not
// expose a direct UpdateStatus method.
type FeatureRepositoryAdapter struct {
	repo FeatureAdapterRepository
}

// Compile-time check that FeatureRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*FeatureRepositoryAdapter)(nil)

// NewFeatureRepositoryAdapter creates an adapter wrapping the given feature repository.
func NewFeatureRepositoryAdapter(repo FeatureAdapterRepository) *FeatureRepositoryAdapter {
	return &FeatureRepositoryAdapter{repo: repo}
}

func (a *FeatureRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *FeatureRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *FeatureRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	feature, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get feature for status update: %w", err)
	}
	feature.Status = models.FeatureStatus(status)
	return a.repo.Update(ctx, feature)
}

func (a *FeatureRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	feature, ok := entity.(*models.Feature)
	if !ok {
		return fmt.Errorf("FeatureRepositoryAdapter.Update: expected *models.Feature, got %T", entity)
	}
	return a.repo.Update(ctx, feature)
}

func (a *FeatureRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	return a.repo.GetContextData(ctx, id)
}

func (a *FeatureRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return a.repo.UpdateContextData(ctx, id, data)
}
