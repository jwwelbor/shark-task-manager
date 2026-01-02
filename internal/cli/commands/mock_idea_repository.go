package commands

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// MockIdeaRepository is a mock implementation of IdeaRepository for testing
type MockIdeaRepository struct {
	CreateFunc                 func(ctx context.Context, idea *models.Idea) error
	GetByIDFunc                func(ctx context.Context, id int64) (*models.Idea, error)
	GetByKeyFunc               func(ctx context.Context, key string) (*models.Idea, error)
	ListFunc                   func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error)
	UpdateFunc                 func(ctx context.Context, idea *models.Idea) error
	DeleteFunc                 func(ctx context.Context, id int64) error
	MarkAsConvertedFunc        func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
	GetNextSequenceForDateFunc func(ctx context.Context, dateStr string) (int, error)
}

// Create mocks the Create method
func (m *MockIdeaRepository) Create(ctx context.Context, idea *models.Idea) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, idea)
	}
	return nil
}

// GetByID mocks the GetByID method
func (m *MockIdeaRepository) GetByID(ctx context.Context, id int64) (*models.Idea, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

// GetByKey mocks the GetByKey method
func (m *MockIdeaRepository) GetByKey(ctx context.Context, key string) (*models.Idea, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, nil
}

// List mocks the List method
func (m *MockIdeaRepository) List(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	return []*models.Idea{}, nil
}

// Update mocks the Update method
func (m *MockIdeaRepository) Update(ctx context.Context, idea *models.Idea) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, idea)
	}
	return nil
}

// Delete mocks the Delete method
func (m *MockIdeaRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MarkAsConverted mocks the MarkAsConverted method
func (m *MockIdeaRepository) MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
	if m.MarkAsConvertedFunc != nil {
		return m.MarkAsConvertedFunc(ctx, ideaID, convertedToType, convertedToKey)
	}
	return nil
}

// GetNextSequenceForDate mocks the GetNextSequenceForDate method
func (m *MockIdeaRepository) GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error) {
	if m.GetNextSequenceForDateFunc != nil {
		return m.GetNextSequenceForDateFunc(ctx, dateStr)
	}
	return 1, nil
}
