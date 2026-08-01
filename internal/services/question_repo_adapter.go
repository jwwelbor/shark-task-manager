package services

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// QuestionAdapterRepository is the narrow typed repository seam required by
// the generic platform services. It deliberately adds no Question-specific
// generic operations.
type QuestionAdapterRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Question, error)
	GetByID(ctx context.Context, id int64) (*models.Question, error)
	Update(ctx context.Context, question *models.Question) error
	UpdateStatus(ctx context.Context, id int64, status models.QuestionStatus) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expected, next models.QuestionStatus) (bool, error)
}

// NewQuestionRepositoryAdapter creates an EntityRepository adapter for
// Questions. Questions have no dedicated repository GetContextData/
// UpdateContextData methods -- ConfigureWorkflow/Resolve/Withdraw own that
// column directly -- so the adapter emulates the generic seam through
// GetByID + Update instead of reporting "not supported".
func NewQuestionRepositoryAdapter(repo QuestionAdapterRepository) EntityRepository {
	return newEntityAdapterWithEmulatedContextData[*models.Question, models.QuestionStatus]("Question", repo)
}
