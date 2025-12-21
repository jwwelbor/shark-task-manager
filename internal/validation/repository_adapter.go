package validation

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// RepositoryAdapter adapts the repository layer to the validation.Repository interface
type RepositoryAdapter struct {
	epicRepo    *repository.EpicRepository
	featureRepo *repository.FeatureRepository
	taskRepo    *repository.TaskRepository
}

// NewRepositoryAdapter creates a new RepositoryAdapter
func NewRepositoryAdapter(
	epicRepo *repository.EpicRepository,
	featureRepo *repository.FeatureRepository,
	taskRepo *repository.TaskRepository,
) *RepositoryAdapter {
	return &RepositoryAdapter{
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
	}
}

// GetAllEpics retrieves all epics from the database
func (a *RepositoryAdapter) GetAllEpics(ctx context.Context) ([]*models.Epic, error) {
	return a.epicRepo.List(ctx, nil) // nil means no status filter
}

// GetAllFeatures retrieves all features from the database
func (a *RepositoryAdapter) GetAllFeatures(ctx context.Context) ([]*models.Feature, error) {
	return a.featureRepo.List(ctx)
}

// GetAllTasks retrieves all tasks from the database
func (a *RepositoryAdapter) GetAllTasks(ctx context.Context) ([]*models.Task, error) {
	return a.taskRepo.List(ctx)
}

// GetEpicByID retrieves an epic by ID
func (a *RepositoryAdapter) GetEpicByID(ctx context.Context, id int64) (*models.Epic, error) {
	return a.epicRepo.GetByID(ctx, id)
}

// GetFeatureByID retrieves a feature by ID
func (a *RepositoryAdapter) GetFeatureByID(ctx context.Context, id int64) (*models.Feature, error) {
	return a.featureRepo.GetByID(ctx, id)
}
