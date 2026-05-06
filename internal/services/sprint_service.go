package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// SprintRepository defines the repository interface for sprint operations.
type SprintRepository interface {
	Create(ctx context.Context, s *models.Sprint) error
	GetByKey(ctx context.Context, key string) (*models.Sprint, error)
	GetByID(ctx context.Context, id int64) (*models.Sprint, error)
	Update(ctx context.Context, s *models.Sprint) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error
	GetNextKey(ctx context.Context) (string, error)
	List(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error)
}

// CreateSprintInput contains parameters for creating a new sprint.
type CreateSprintInput struct {
	Name      string    // Required, non-empty
	Goal      string    // Optional
	StartDate time.Time // Required
	EndDate   time.Time // Required
}

// UpdateSprintInput contains parameters for updating a sprint.
// Fields are pointers to indicate optional updates.
type UpdateSprintInput struct {
	Name    *string    // Optional update
	Goal    *string    // Optional update
	EndDate *time.Time // Optional update
}

// SprintListFilters contains filter options for listing sprints.
type SprintListFilters struct {
	Status string // Optional filter, e.g. "active"
}

// SprintService provides business logic for sprint operations.
type SprintService struct {
	repo        SprintRepository
	workflowSvc *workflow.Service
}

// NewSprintService creates a SprintService with dependency injection.
// Panics if repo or workflowSvc is nil.
func NewSprintService(repo SprintRepository, workflowSvc *workflow.Service) *SprintService {
	if repo == nil {
		panic("SprintRepository cannot be nil")
	}
	if workflowSvc == nil {
		panic("workflow.Service cannot be nil")
	}

	return &SprintService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel("sprint"),
	}
}

// CreateSprint creates a new sprint with validation.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: CreateSprintInput with name, goal, and dates
//
// Returns:
//   - *models.Sprint: the created sprint with ID populated
//   - error: validation errors or repository errors
//
// Validation:
//   - Name must be non-empty after trimming whitespace
//   - StartDate must be before EndDate
func (s *SprintService) CreateSprint(ctx context.Context, input CreateSprintInput) (*models.Sprint, error) {
	// Validate name is non-empty
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("sprint name cannot be empty")
	}

	// Validate date ordering
	if !input.EndDate.After(input.StartDate) {
		return nil, fmt.Errorf("sprint end_date must be after start_date")
	}

	// Generate key
	key, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sprint key: %w", err)
	}

	// Generate slug from name
	slug := utils.GenerateSlug(input.Name)

	// Get default status from workflow
	initialStatus := s.workflowSvc.GetInitialStatusString()

	// Create sprint model
	newSprint := &models.Sprint{
		Key:       key,
		Name:      input.Name,
		Goal:      input.Goal,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
		Status:    models.SprintStatus(initialStatus),
		Slug:      slug,
	}

	// Validate model
	if err := newSprint.Validate(); err != nil {
		return nil, fmt.Errorf("sprint validation failed: %w", err)
	}

	// Create in repository
	if err := s.repo.Create(ctx, newSprint); err != nil {
		return nil, fmt.Errorf("failed to create sprint: %w", err)
	}

	return newSprint, nil
}

// GetSprint retrieves a sprint by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key (e.g., "S001")
//
// Returns:
//   - *models.Sprint: the sprint, or error if not found
//   - error: NotFoundError or repository errors
func (s *SprintService) GetSprint(ctx context.Context, key string) (*models.Sprint, error) {
	sprint, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint %s: %w", key, err)
	}
	return sprint, nil
}

// ListSprints retrieves sprints, optionally filtered by status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: SprintListFilters with optional status filter
//
// Returns:
//   - []*models.Sprint: list of sprints (empty slice if none found)
//   - error: repository errors
func (s *SprintService) ListSprints(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
	// Convert service-level filter to repository filter
	var repoFilters *sprint.SprintListFilters
	if filters != nil && filters.Status != "" {
		status := models.SprintStatus(filters.Status)
		repoFilters = &sprint.SprintListFilters{
			Status: &status,
		}
	}

	sprints, err := s.repo.List(ctx, repoFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}

	// Return empty slice instead of nil for consistency
	if sprints == nil {
		return []*models.Sprint{}, nil
	}

	return sprints, nil
}

// UpdateSprint updates an existing sprint.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//   - updates: UpdateSprintInput with optional fields
//
// Returns:
//   - *models.Sprint: the updated sprint
//   - error: validation errors or repository errors
//
// Validation:
//   - Name (if provided) must be non-empty
//   - EndDate (if provided) must be after StartDate
func (s *SprintService) UpdateSprint(ctx context.Context, key string, updates UpdateSprintInput) (*models.Sprint, error) {
	// Get current sprint
	current, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to update sprint %s: %w", key, err)
	}

	// Apply updates
	if updates.Name != nil {
		name := strings.TrimSpace(*updates.Name)
		if name == "" {
			return nil, fmt.Errorf("sprint name cannot be empty")
		}
		current.Name = name
		current.Slug = utils.GenerateSlug(name)
	}

	if updates.Goal != nil {
		current.Goal = *updates.Goal
	}

	if updates.EndDate != nil {
		// Validate date ordering
		if !updates.EndDate.After(current.StartDate) {
			return nil, fmt.Errorf("sprint end_date must be after start_date")
		}
		current.EndDate = *updates.EndDate
	}

	// Validate updated model
	if err := current.Validate(); err != nil {
		return nil, fmt.Errorf("sprint validation failed: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, fmt.Errorf("failed to update sprint %s: %w", key, err)
	}

	return current, nil
}

// DeleteSprint deletes a sprint by key.
// Sprints can only be deleted when in "todo" status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - error: validation errors or repository errors
//
// Validation:
//   - Sprint status must be "todo"
func (s *SprintService) DeleteSprint(ctx context.Context, key string) error {
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete sprint %s: %w", key, err)
	}

	// Only allow deletion of sprints in todo status
	if string(sprint.Status) != "todo" {
		return fmt.Errorf("cannot delete sprint %s in status %s: only sprints in todo status can be deleted", key, sprint.Status)
	}

	if err := s.repo.Delete(ctx, sprint.ID); err != nil {
		return fmt.Errorf("failed to delete sprint %s: %w", key, err)
	}

	return nil
}

// StartSprint transitions a sprint to in_progress status.
// Enforces single-active-sprint constraint: only one sprint can be in_progress at a time.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - *models.Sprint: the updated sprint with new status
//   - error: validation errors or repository errors
//
// Validation:
//   - Current sprint status must allow transition to "in_progress"
//   - No other sprint can be in "in_progress" status
func (s *SprintService) StartSprint(ctx context.Context, key string) (*models.Sprint, error) {
	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to start sprint %s: %w", key, err)
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "in_progress"); err != nil {
		return nil, fmt.Errorf("cannot start sprint %s in status %s: %w", key, sprint.Status, err)
	}

	// Check single-active constraint: no other sprint should be in_progress
	activeSprints, err := s.ListSprints(ctx, &SprintListFilters{Status: "in_progress"})
	if err != nil {
		return nil, fmt.Errorf("failed to check active sprints for %s: %w", key, err)
	}

	if len(activeSprints) > 0 {
		// There's already an active sprint
		return nil, fmt.Errorf("cannot activate sprint %s: sprint %s is already active", key, activeSprints[0].Key)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("in_progress")); err != nil {
		return nil, fmt.Errorf("failed to start sprint %s: %w", key, err)
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after starting: %w", key, err)
	}

	return updated, nil
}

// CloseSprint transitions a sprint to ready_for_review status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - *models.Sprint: the updated sprint with new status
//   - error: validation errors or repository errors
//
// Validation:
//   - Current sprint status must allow transition to "ready_for_review"
//
// Note: Carryover logic is deferred to a future feature.
func (s *SprintService) CloseSprint(ctx context.Context, key string) (*models.Sprint, error) {
	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to close sprint %s: %w", key, err)
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "ready_for_review"); err != nil {
		return nil, fmt.Errorf("cannot close sprint %s in status %s: %w", key, sprint.Status, err)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("ready_for_review")); err != nil {
		return nil, fmt.Errorf("failed to close sprint %s: %w", key, err)
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after closing: %w", key, err)
	}

	return updated, nil
}

// ArchiveSprint transitions a sprint to completed status.
// This marks the sprint as done and archives it from active consideration.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - *models.Sprint: the updated sprint with new status
//   - error: validation errors or repository errors
//
// Validation:
//   - Current sprint status must allow transition to "completed"
func (s *SprintService) ArchiveSprint(ctx context.Context, key string) (*models.Sprint, error) {
	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to archive sprint %s: %w", key, err)
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "completed"); err != nil {
		return nil, fmt.Errorf("cannot archive sprint %s in status %s: %w", key, sprint.Status, err)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("completed")); err != nil {
		return nil, fmt.Errorf("failed to archive sprint %s: %w", key, err)
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after archiving: %w", key, err)
	}

	return updated, nil
}
