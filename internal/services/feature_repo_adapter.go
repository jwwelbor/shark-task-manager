package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// FeatureAdapterRepository defines the minimal interface needed by
// NewFeatureRepositoryAdapter.
type FeatureAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
	UpdateStatus(ctx context.Context, featureID int64, status models.FeatureStatus) error
	UpdateStatusIfCurrent(ctx context.Context, featureID int64, expectedStatus models.FeatureStatus, newStatus models.FeatureStatus) (bool, error)
	GetContextData(ctx context.Context, featureID int64) (*string, error)
	UpdateContextData(ctx context.Context, featureID int64, contextData *string) error
}

// NewFeatureRepositoryAdapter creates an EntityRepository adapter for features.
func NewFeatureRepositoryAdapter(repo FeatureAdapterRepository) EntityRepository {
	return newEntityAdapter[*models.Feature, models.FeatureStatus]("Feature", repo)
}
