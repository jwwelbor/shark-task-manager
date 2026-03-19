package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskAdapterRepository defines the minimal interface needed by TaskRepositoryAdapter.
type TaskAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
}

// TaskRepositoryAdapter wraps a typed task repository to satisfy EntityRepository.
// UpdateStatus, GetContextData, and UpdateContextData all use the get-set-update
// pattern because the TaskRepository interface does not expose direct methods
// with compatible signatures.
type TaskRepositoryAdapter struct {
	repo TaskAdapterRepository
}

// Compile-time check that TaskRepositoryAdapter implements EntityRepository.
var _ EntityRepository = (*TaskRepositoryAdapter)(nil)

// NewTaskRepositoryAdapter creates an adapter wrapping the given task repository.
func NewTaskRepositoryAdapter(repo TaskAdapterRepository) *TaskRepositoryAdapter {
	return &TaskRepositoryAdapter{repo: repo}
}

func (a *TaskRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *TaskRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *TaskRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get task for status update: %w", err)
	}
	task.Status = models.TaskStatus(status)
	return a.repo.Update(ctx, task)
}

func (a *TaskRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	task, ok := entity.(*models.Task)
	if !ok {
		return fmt.Errorf("TaskRepositoryAdapter.Update: expected *models.Task, got %T", entity)
	}
	return a.repo.Update(ctx, task)
}

func (a *TaskRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return task.ContextData, nil
}

func (a *TaskRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	task.ContextData = data
	return a.repo.Update(ctx, task)
}
