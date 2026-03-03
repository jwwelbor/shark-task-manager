package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ContextEpicRepository defines the epic repository interface needed by ContextService.
type ContextEpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetContextData(ctx context.Context, epicID int64) (*string, error)
	UpdateContextData(ctx context.Context, epicID int64, contextData *string) error
}

// ContextFeatureRepository defines the feature repository interface needed by ContextService.
type ContextFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetContextData(ctx context.Context, featureID int64) (*string, error)
	UpdateContextData(ctx context.Context, featureID int64, contextData *string) error
}

// ContextTaskRepository defines the task repository interface needed by ContextService.
type ContextTaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
}

// ContextBugRepository defines the bug repository interface needed by ContextService.
type ContextBugRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
	Update(ctx context.Context, bug *models.Bug) error
}

// Bug is an alias to avoid import issues - the actual type comes from models.
type Bug = models.Bug

// ContextService provides business logic for context data operations across all entity types.
type ContextService struct {
	epicRepo    ContextEpicRepository
	featureRepo ContextFeatureRepository
	taskRepo    ContextTaskRepository
	bugRepo     ContextBugRepository
}

// NewContextService creates a new ContextService with injected dependencies.
func NewContextService(epicRepo ContextEpicRepository, featureRepo ContextFeatureRepository, taskRepo ContextTaskRepository) *ContextService {
	return &ContextService{
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
	}
}

// SetBugRepo sets the optional bug repository for context operations on bugs.
func (s *ContextService) SetBugRepo(repo ContextBugRepository) {
	s.bugRepo = repo
}

// GetContext returns the parsed context data for an entity.
// Returns nil if the entity has no context data.
func (s *ContextService) GetContext(ctx context.Context, entityType models.EntityType, entityKey string) (*models.ContextData, error) {
	contextJSON, err := s.getContextJSON(ctx, entityType, entityKey)
	if err != nil {
		return nil, err
	}

	if contextJSON == nil || *contextJSON == "" || *contextJSON == "{}" {
		return nil, nil
	}

	contextData, err := models.FromJSON(*contextJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse context data: %w", err)
	}

	return contextData, nil
}

// SetContextField sets a single field in the entity's context data using merge semantics.
func (s *ContextService) SetContextField(ctx context.Context, entityType models.EntityType, entityKey string, field string, value string) error {
	// Validate field name
	if !isValidContextField(field) {
		return fmt.Errorf("invalid context field: %s", field)
	}

	// Get current context data
	contextJSON, err := s.getContextJSON(ctx, entityType, entityKey)
	if err != nil {
		return err
	}

	var contextData *models.ContextData
	if contextJSON != nil && *contextJSON != "" {
		contextData, err = models.FromJSON(*contextJSON)
		if err != nil {
			return fmt.Errorf("failed to parse existing context data: %w", err)
		}
	} else {
		contextData = &models.ContextData{}
	}

	// Update the specified field
	if err := updateContextField(contextData, field, value); err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	// Convert back to JSON and save
	jsonStr, err := contextData.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize context data: %w", err)
	}

	return s.setContextJSON(ctx, entityType, entityKey, &jsonStr)
}

// ClearContext removes all context data from an entity.
func (s *ContextService) ClearContext(ctx context.Context, entityType models.EntityType, entityKey string) error {
	return s.setContextJSON(ctx, entityType, entityKey, nil)
}

// getContextJSON retrieves the raw context JSON string for an entity.
func (s *ContextService) getContextJSON(ctx context.Context, entityType models.EntityType, entityKey string) (*string, error) {
	switch entityType {
	case models.EntityTypeEpic:
		epic, err := s.epicRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return nil, fmt.Errorf("epic not found: %s: %w", entityKey, err)
		}
		return s.epicRepo.GetContextData(ctx, epic.ID)
	case models.EntityTypeFeature:
		feature, err := s.featureRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return nil, fmt.Errorf("feature not found: %s: %w", entityKey, err)
		}
		return s.featureRepo.GetContextData(ctx, feature.ID)
	case models.EntityTypeTask:
		task, err := s.taskRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s: %w", entityKey, err)
		}
		return task.ContextData, nil
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

// setContextJSON writes the raw context JSON string for an entity.
func (s *ContextService) setContextJSON(ctx context.Context, entityType models.EntityType, entityKey string, contextJSON *string) error {
	switch entityType {
	case models.EntityTypeEpic:
		epic, err := s.epicRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("epic not found: %s: %w", entityKey, err)
		}
		return s.epicRepo.UpdateContextData(ctx, epic.ID, contextJSON)
	case models.EntityTypeFeature:
		feature, err := s.featureRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("feature not found: %s: %w", entityKey, err)
		}
		return s.featureRepo.UpdateContextData(ctx, feature.ID, contextJSON)
	case models.EntityTypeTask:
		task, err := s.taskRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("task not found: %s: %w", entityKey, err)
		}
		task.ContextData = contextJSON
		return s.taskRepo.Update(ctx, task)
	default:
		return fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

// isValidContextField checks if a field name is a valid context field.
func isValidContextField(field string) bool {
	validFields := map[string]bool{
		"current_step":               true,
		"completed_steps":            true,
		"remaining_steps":            true,
		"implementation_decisions":   true,
		"open_questions":             true,
		"blockers":                   true,
		"acceptance_criteria_status": true,
	}
	return validFields[field]
}

// updateContextField updates a specific field in the context data.
// This is extracted from the CLI layer for reuse across entity types.
func updateContextField(cd *models.ContextData, field, value string) error {
	switch field {
	case "current_step":
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.CurrentStep = &value

	case "completed_steps":
		var steps []string
		if err := json.Unmarshal([]byte(value), &steps); err != nil {
			return fmt.Errorf("invalid JSON for completed_steps: %w", err)
		}
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.CompletedSteps = steps

	case "remaining_steps":
		var steps []string
		if err := json.Unmarshal([]byte(value), &steps); err != nil {
			return fmt.Errorf("invalid JSON for remaining_steps: %w", err)
		}
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.RemainingSteps = steps

	case "implementation_decisions":
		var decisions map[string]string
		if err := json.Unmarshal([]byte(value), &decisions); err != nil {
			return fmt.Errorf("invalid JSON for implementation_decisions: %w", err)
		}
		if cd.ImplementationDecisions == nil {
			cd.ImplementationDecisions = make(map[string]string)
		}
		for k, v := range decisions {
			cd.ImplementationDecisions[k] = v
		}

	case "open_questions":
		var questions []string
		if err := json.Unmarshal([]byte(value), &questions); err != nil {
			return fmt.Errorf("invalid JSON for open_questions: %w", err)
		}
		cd.OpenQuestions = questions

	case "blockers":
		var blockers []models.BlockerContext
		if err := json.Unmarshal([]byte(value), &blockers); err != nil {
			return fmt.Errorf("invalid JSON for blockers: %w", err)
		}
		cd.Blockers = blockers

	case "acceptance_criteria_status":
		var criteria []models.AcceptanceCriterionContext
		if err := json.Unmarshal([]byte(value), &criteria); err != nil {
			return fmt.Errorf("invalid JSON for acceptance_criteria_status: %w", err)
		}
		cd.AcceptanceCriteriaStatus = criteria

	default:
		return fmt.Errorf("unsupported context field: %s", field)
	}

	return nil
}
