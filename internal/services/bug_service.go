package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// BugRepository defines the repository interface for bug operations.
type BugRepository interface {
	Create(ctx context.Context, bug *models.Bug) error
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
	GetByID(ctx context.Context, id int64) (*models.Bug, error)
	Update(ctx context.Context, bug *models.Bug) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error
	GetNextKey(ctx context.Context) (string, error)
	List(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	CountBySeverity(ctx context.Context) (map[string]int, error)
}

// LinkValidatorEpicRepo defines the interface for validating epic links.
type LinkValidatorEpicRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// LinkValidatorFeatureRepo defines the interface for validating feature links.
type LinkValidatorFeatureRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// LinkValidatorTaskRepo defines the interface for validating task links.
type LinkValidatorTaskRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
}

// BugService provides business logic for bug operations.
type BugService struct {
	repo        BugRepository
	workflowSvc *workflow.Service
	epicRepo    LinkValidatorEpicRepo
	featureRepo LinkValidatorFeatureRepo
	taskRepo    LinkValidatorTaskRepo
}

// NewBugService creates a new BugService with injected dependencies.
func NewBugService(
	repo BugRepository,
	workflowSvc *workflow.Service,
	epicRepo LinkValidatorEpicRepo,
	featureRepo LinkValidatorFeatureRepo,
	taskRepo LinkValidatorTaskRepo,
) *BugService {
	return &BugService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelBug),
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
	}
}

// CreateBug creates a new bug with auto-generated key and slug.
func (s *BugService) CreateBug(ctx context.Context, input CreateBugInput) (*models.Bug, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("bug title cannot be empty")
	}

	if !models.ValidBugSeverities[input.Severity] {
		return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", input.Severity)
	}

	// Validate linked entity if provided
	if input.LinkedEntityType != "" || input.LinkedEntityKey != "" {
		if input.LinkedEntityType == "" || input.LinkedEntityKey == "" {
			return nil, fmt.Errorf("both linked_entity_type and linked_entity_key must be provided together")
		}
		if err := s.validateLinkedEntity(ctx, input.LinkedEntityType, input.LinkedEntityKey); err != nil {
			return nil, fmt.Errorf("linked entity validation failed: %w", err)
		}
	}

	// Generate key
	key, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate bug key: %w", err)
	}

	// Get default status from workflow
	defaultStatus := s.workflowSvc.GetDefaultStatus()

	// Generate slug
	slug := utils.GenerateSlug(input.Title)

	bug := &models.Bug{
		Key:      key,
		Title:    input.Title,
		Slug:     &slug,
		Status:   models.BugStatus(defaultStatus),
		Severity: input.Severity,
	}

	if input.Description != "" {
		bug.Description = &input.Description
	}

	if input.LinkedEntityType != "" {
		bug.LinkedEntityType = &input.LinkedEntityType
		bug.LinkedEntityKey = &input.LinkedEntityKey
	}

	if err := s.repo.Create(ctx, bug); err != nil {
		return nil, fmt.Errorf("failed to create bug: %w", err)
	}

	return bug, nil
}

// GetBug retrieves a bug by its key.
func (s *BugService) GetBug(ctx context.Context, key string) (*models.Bug, error) {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}
	return bug, nil
}

// UpdateBug applies partial updates to a bug.
func (s *BugService) UpdateBug(ctx context.Context, key string, updates BugUpdates) (*models.Bug, error) {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	if updates.Title != nil {
		if strings.TrimSpace(*updates.Title) == "" {
			return nil, fmt.Errorf("bug title cannot be empty")
		}
		bug.Title = *updates.Title
		slug := utils.GenerateSlug(bug.Title)
		bug.Slug = &slug
	}

	if updates.Description != nil {
		bug.Description = updates.Description
	}

	if updates.Severity != nil {
		if !models.ValidBugSeverities[*updates.Severity] {
			return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", *updates.Severity)
		}
		bug.Severity = *updates.Severity
	}

	if updates.LinkedEntityType != nil || updates.LinkedEntityKey != nil {
		entityType := ""
		entityKey := ""
		if updates.LinkedEntityType != nil {
			entityType = *updates.LinkedEntityType
		}
		if updates.LinkedEntityKey != nil {
			entityKey = *updates.LinkedEntityKey
		}

		if entityType != "" || entityKey != "" {
			if entityType == "" || entityKey == "" {
				return nil, fmt.Errorf("both linked_entity_type and linked_entity_key must be provided together")
			}
			if err := s.validateLinkedEntity(ctx, entityType, entityKey); err != nil {
				return nil, fmt.Errorf("linked entity validation failed: %w", err)
			}
			bug.LinkedEntityType = &entityType
			bug.LinkedEntityKey = &entityKey
		} else {
			// Both empty means clear the link
			bug.LinkedEntityType = nil
			bug.LinkedEntityKey = nil
		}
	}

	if err := s.repo.Update(ctx, bug); err != nil {
		return nil, fmt.Errorf("failed to update bug %s: %w", key, err)
	}

	return bug, nil
}

// DeleteBug deletes a bug by its key.
func (s *BugService) DeleteBug(ctx context.Context, key string) error {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	if err := s.repo.Delete(ctx, bug.ID); err != nil {
		return fmt.Errorf("failed to delete bug %s: %w", key, err)
	}

	return nil
}

// ListBugs retrieves bugs with optional filters.
func (s *BugService) ListBugs(ctx context.Context, filters BugFilters) ([]*models.Bug, error) {
	repoFilters := &repository.BugListFilters{
		Status:   filters.Status,
		Severity: filters.Severity,
	}

	bugs, err := s.repo.List(ctx, repoFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list bugs: %w", err)
	}

	return bugs, nil
}

// AdvanceBugStatus advances a bug to the next workflow status.
func (s *BugService) AdvanceBugStatus(ctx context.Context, key string) (*models.Bug, error) {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	validTransitions := s.workflowSvc.GetValidTransitions(string(bug.Status))
	if len(validTransitions) == 0 {
		return nil, fmt.Errorf("cannot advance bug %s: no valid transitions from status %q", key, bug.Status)
	}

	nextStatus := validTransitions[0]

	if err := s.repo.UpdateStatus(ctx, bug.ID, models.BugStatus(nextStatus)); err != nil {
		return nil, fmt.Errorf("failed to advance bug %s status: %w", key, err)
	}

	bug.Status = models.BugStatus(nextStatus)
	return bug, nil
}

// SetBugStatus sets a bug to a specific status with workflow validation.
func (s *BugService) SetBugStatus(ctx context.Context, key string, status string, force bool) (*models.Bug, error) {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	// Validate the target status is a valid status in the workflow
	if err := s.workflowSvc.ValidateStatus(status); err != nil {
		return nil, fmt.Errorf("invalid bug status %q: %w", status, err)
	}

	// Validate transition unless forced
	if !force {
		if err := s.workflowSvc.ValidateTransition(string(bug.Status), status); err != nil {
			return nil, fmt.Errorf("cannot transition bug %s from %q to %q: %w", key, bug.Status, status, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, bug.ID, models.BugStatus(status)); err != nil {
		return nil, fmt.Errorf("failed to set bug %s status: %w", key, err)
	}

	bug.Status = models.BugStatus(status)
	return bug, nil
}

// validateLinkedEntity validates that a linked entity exists.
func (s *BugService) validateLinkedEntity(ctx context.Context, entityType, entityKey string) error {
	switch entityType {
	case "epic":
		if s.epicRepo == nil {
			return fmt.Errorf("epic repository not available for link validation")
		}
		_, err := s.epicRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("epic %q not found: %w", entityKey, err)
		}
	case "feature":
		if s.featureRepo == nil {
			return fmt.Errorf("feature repository not available for link validation")
		}
		_, err := s.featureRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("feature %q not found: %w", entityKey, err)
		}
	case "task":
		if s.taskRepo == nil {
			return fmt.Errorf("task repository not available for link validation")
		}
		_, err := s.taskRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("task %q not found: %w", entityKey, err)
		}
	default:
		return fmt.Errorf("invalid linked entity type %q: must be epic, feature, or task", entityType)
	}
	return nil
}
