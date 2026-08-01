package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskAdapterRepository defines the minimal interface needed by
// NewTaskRepositoryAdapter.
type TaskAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
	UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
	UpdateStatusIfCurrent(ctx context.Context, taskID int64, expectedStatus models.TaskStatus, newStatus models.TaskStatus) (bool, error)
	GetContextData(ctx context.Context, taskID int64) (*string, error)
	UpdateContextData(ctx context.Context, taskID int64, contextData *string) error
}

// taskRepositoryAdapter wraps a typed task repository to satisfy
// EntityRepository. Unlike the other entity adapters, tasks can't
// instantiate the generic entityAdapter directly: TaskRepository.UpdateStatus
// additionally threads an agent/notes pair and records task_history side
// effects, so it embeds the generic adapter for the shared mechanics and
// overrides only UpdateStatus to call the typed method directly.
type taskRepositoryAdapter struct {
	*entityAdapter[*models.Task, models.TaskStatus]
	repo TaskAdapterRepository
}

// NewTaskRepositoryAdapter creates an EntityRepository adapter for tasks.
func NewTaskRepositoryAdapter(repo TaskAdapterRepository) EntityRepository {
	return &taskRepositoryAdapter{
		entityAdapter: newEntityAdapter[*models.Task, models.TaskStatus]("Task", repo),
		repo:          repo,
	}
}

func (a *taskRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.repo.UpdateStatus(ctx, id, models.TaskStatus(status), nil, nil)
}
