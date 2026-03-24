package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// IdeaRepository defines the repository interface needed by IdeaService.
// This interface is satisfied by *repository.IdeaRepository.
type IdeaRepository interface {
	// CRUD operations
	Create(ctx context.Context, idea *models.Idea) error
	GetByID(ctx context.Context, id int64) (*models.Idea, error)
	GetByKey(ctx context.Context, key string) (*models.Idea, error)
	Update(ctx context.Context, idea *models.Idea) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error)

	// Business operations
	MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
	GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error)
}

// IdeaService provides business logic for idea operations.
// It orchestrates idea lifecycle, key generation, status transitions,
// and coordinates with the idea repository.
type IdeaService struct {
	repo IdeaRepository
}

// NewIdeaService creates a new IdeaService with the required dependencies.
//
// Parameters:
//   - repo: idea repository for data access (required)
//
// Returns:
//   - *IdeaService: configured idea service instance
//   - error: if repo is nil
func NewIdeaService(repo IdeaRepository) (*IdeaService, error) {
	if repo == nil {
		return nil, fmt.Errorf("IdeaService requires a non-nil IdeaRepository")
	}
	return &IdeaService{repo: repo}, nil
}

// CreateIdea creates a new idea with a date-based key generated automatically.
//
// The key format is I-YYYY-MM-DD-xx where:
//   - YYYY-MM-DD is today's date (or input.CreatedDate if provided)
//   - xx is a zero-padded sequence number (01, 02, ...) unique per date
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: creation parameters (Title is required)
//
// Returns:
//   - *models.Idea: the created idea with generated key and ID
//   - error: validation errors or repository errors
func (s *IdeaService) CreateIdea(ctx context.Context, input CreateIdeaInput) (*models.Idea, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("idea title is required")
	}

	// Determine creation date
	createdDate := time.Now()
	if input.CreatedDate != nil {
		createdDate = *input.CreatedDate
	}
	dateStr := createdDate.Format("2006-01-02")

	// Generate key using date-based sequence
	seq, err := s.repo.GetNextSequenceForDate(ctx, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to generate idea key: %w", err)
	}
	key := fmt.Sprintf("I-%s-%02d", dateStr, seq)

	status := models.IdeaStatusNew
	if input.Status != "" {
		status = models.IdeaStatus(input.Status)
	}

	idea := &models.Idea{
		Key:          key,
		Title:        input.Title,
		Description:  input.Description,
		CreatedDate:  createdDate,
		Priority:     input.Priority,
		Order:        input.Order,
		Notes:        input.Notes,
		RelatedDocs:  input.RelatedDocs,
		Dependencies: input.Dependencies,
		Status:       status,
	}

	if err := s.repo.Create(ctx, idea); err != nil {
		return nil, fmt.Errorf("failed to create idea: %w", err)
	}

	return idea, nil
}

// GetIdea retrieves an idea by its key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: idea key (e.g., I-2026-01-15-01)
//
// Returns:
//   - *models.Idea: the found idea
//   - error: not-found error or repository error
func (s *IdeaService) GetIdea(ctx context.Context, key string) (*models.Idea, error) {
	idea, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get idea %s: %w", key, err)
	}
	return idea, nil
}

// ListIdeas retrieves ideas matching the given filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: filtering options (Status filter supported; empty = no filter)
//
// Returns:
//   - []*models.Idea: list of matching ideas (may be empty, never nil)
//   - error: repository error
func (s *IdeaService) ListIdeas(ctx context.Context, filters IdeaFilters) ([]*models.Idea, error) {
	var repoFilter *repository.IdeaFilter

	if filters.Status != "" {
		status := models.IdeaStatus(filters.Status)
		repoFilter = &repository.IdeaFilter{Status: &status}
	}

	ideas, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list ideas: %w", err)
	}

	if ideas == nil {
		ideas = []*models.Idea{}
	}

	return ideas, nil
}

// UpdateIdea updates an existing idea with the provided changes.
//
// Only non-nil fields in input are applied to the existing idea.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: idea key to update
//   - input: fields to update (only non-nil fields are applied)
//
// Returns:
//   - *models.Idea: the updated idea
//   - error: not-found error, validation error, or repository error
func (s *IdeaService) UpdateIdea(ctx context.Context, key string, input UpdateIdeaInput) (*models.Idea, error) {
	idea, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get idea %s for update: %w", key, err)
	}

	// Apply non-nil updates
	if input.Title != nil {
		idea.Title = *input.Title
	}
	if input.Description != nil {
		idea.Description = input.Description
	}
	if input.Priority != nil {
		idea.Priority = input.Priority
	}
	if input.Order != nil {
		idea.Order = input.Order
	}
	if input.Notes != nil {
		idea.Notes = input.Notes
	}
	if input.RelatedDocs != nil {
		idea.RelatedDocs = input.RelatedDocs
	}
	if input.Dependencies != nil {
		idea.Dependencies = input.Dependencies
	}
	if input.Status != nil {
		idea.Status = models.IdeaStatus(*input.Status)
	}

	if err := s.repo.Update(ctx, idea); err != nil {
		return nil, fmt.Errorf("failed to update idea %s: %w", key, err)
	}

	return idea, nil
}

// DeleteIdea deletes an idea by its key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: idea key to delete
//
// Returns:
//   - error: not-found error or repository error
func (s *IdeaService) DeleteIdea(ctx context.Context, key string) error {
	idea, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get idea %s for deletion: %w", key, err)
	}

	if err := s.repo.Delete(ctx, idea.ID); err != nil {
		return fmt.Errorf("failed to delete idea %s: %w", key, err)
	}

	return nil
}

// ConvertIdea marks an idea as converted to another entity type.
//
// This sets the idea status to "converted" and records the target entity
// type and key. It returns an error if the idea is already converted.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: idea key to convert
//   - convertedToType: entity type ("epic", "feature", "task")
//   - convertedToKey: key of the created entity
//
// Returns:
//   - error: not-found error, already-converted error, or repository error
func (s *IdeaService) ConvertIdea(ctx context.Context, key, convertedToType, convertedToKey string) error {
	idea, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get idea %s for conversion: %w", key, err)
	}

	if idea.Status == models.IdeaStatusConverted {
		return fmt.Errorf("idea %s is already converted", key)
	}

	if err := s.repo.MarkAsConverted(ctx, idea.ID, convertedToType, convertedToKey); err != nil {
		return fmt.Errorf("failed to convert idea %s: %w", key, err)
	}

	return nil
}
