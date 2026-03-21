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
	UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
	GetContextData(ctx context.Context, taskID int64) (*string, error)
	UpdateContextData(ctx context.Context, taskID int64, contextData *string) error
}

// TaskRepositoryAdapter wraps a typed task repository to satisfy EntityRepository.
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
	return a.repo.UpdateStatus(ctx, id, models.TaskStatus(status), nil, nil)
}

func (a *TaskRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
	task, ok := entity.(*models.Task)
	if !ok {
		return fmt.Errorf("TaskRepositoryAdapter.Update: expected *models.Task, got %T", entity)
	}
	return a.repo.Update(ctx, task)
}

func (a *TaskRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	return a.repo.GetContextData(ctx, id)
}

func (a *TaskRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return a.repo.UpdateContextData(ctx, id, data)
}
