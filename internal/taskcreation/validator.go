package taskcreation

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Validator handles input validation for task creation
type Validator struct {
	epicRepo    *repository.EpicRepository
	featureRepo *repository.FeatureRepository
	taskRepo    *repository.TaskRepository
}

// NewValidator creates a new Validator
func NewValidator(
	epicRepo *repository.EpicRepository,
	featureRepo *repository.FeatureRepository,
	taskRepo *repository.TaskRepository,
) *Validator {
	return &Validator{
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
	}
}

// TaskInput represents the input data for task creation
type TaskInput struct {
	EpicKey     string
	FeatureKey  string
	Title       string
	Description string
	AgentType   string
	Priority    int
	DependsOn   string // Comma-separated list
}

// ValidatedTaskData represents validated task creation data
type ValidatedTaskData struct {
	EpicID                int64
	FeatureID             int64
	NormalizedFeatureKey  string
	ValidatedDependencies []string
	AgentType             models.AgentType
}

// ValidateTaskInput validates all input fields for task creation
func (v *Validator) ValidateTaskInput(ctx context.Context, input TaskInput) (*ValidatedTaskData, error) {
	// 1. Validate epic exists
	epic, err := v.epicRepo.GetByKey(ctx, input.EpicKey)
	if err != nil {
		return nil, fmt.Errorf("epic %s does not exist. Use 'shark epic list' to see available epics", input.EpicKey)
	}

	// 2. Normalize feature key
	normalizedFeatureKey := normalizeFeatureKey(input.EpicKey, input.FeatureKey)

	// 3. Validate feature exists and belongs to epic
	feature, err := v.featureRepo.GetByKey(ctx, normalizedFeatureKey)
	if err != nil {
		return nil, fmt.Errorf("feature %s does not exist", normalizedFeatureKey)
	}

	if feature.EpicID != epic.ID {
		return nil, fmt.Errorf("feature %s does not belong to epic %s", normalizedFeatureKey, input.EpicKey)
	}

	// 4. Validate and convert agent type
	var agentType models.AgentType
	if input.AgentType != "" {
		// Validate agent type against valid values
		if err := models.ValidateAgentType(input.AgentType); err != nil {
			return nil, err
		}
		agentType = models.AgentType(input.AgentType)
	} else {
		// Default to general if not provided
		agentType = models.AgentTypeGeneral
	}

	// 5. Validate priority
	if input.Priority < 1 || input.Priority > 10 {
		return nil, fmt.Errorf("priority must be between 1 and 10, got %d", input.Priority)
	}

	// 6. Validate dependencies
	validatedDeps, err := v.validateDependencies(ctx, input.DependsOn)
	if err != nil {
		return nil, err
	}

	// 7. Validate title is not empty
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	return &ValidatedTaskData{
		EpicID:                epic.ID,
		FeatureID:             feature.ID,
		NormalizedFeatureKey:  normalizedFeatureKey,
		ValidatedDependencies: validatedDeps,
		AgentType:             agentType,
	}, nil
}

// validateDependencies validates that all dependency task keys exist
func (v *Validator) validateDependencies(ctx context.Context, dependsOn string) ([]string, error) {
	if dependsOn == "" {
		return []string{}, nil
	}

	// Parse comma-separated list
	deps := strings.Split(dependsOn, ",")
	validatedDeps := []string{}

	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}

		// Validate task key format
		if err := models.ValidateTaskKey(dep); err != nil {
			return nil, fmt.Errorf("invalid task key format in dependencies: %s", dep)
		}

		// Check if task exists
		_, err := v.taskRepo.GetByKey(ctx, dep)
		if err != nil {
			return nil, fmt.Errorf("dependency task %s does not exist", dep)
		}

		validatedDeps = append(validatedDeps, dep)
	}

	return validatedDeps, nil
}

// ValidateEpic validates that an epic exists
func (v *Validator) ValidateEpic(ctx context.Context, epicKey string) error {
	_, err := v.epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return fmt.Errorf("epic %s does not exist. Use 'shark epic list' to see available epics", epicKey)
	}
	return nil
}

// ValidateFeature validates that a feature exists and belongs to an epic
func (v *Validator) ValidateFeature(ctx context.Context, epicKey, featureKey string) error {
	epic, err := v.epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return fmt.Errorf("epic %s does not exist", epicKey)
	}

	normalizedFeatureKey := normalizeFeatureKey(epicKey, featureKey)
	feature, err := v.featureRepo.GetByKey(ctx, normalizedFeatureKey)
	if err != nil {
		return fmt.Errorf("feature %s does not exist", normalizedFeatureKey)
	}

	if feature.EpicID != epic.ID {
		return fmt.Errorf("feature %s does not belong to epic %s", normalizedFeatureKey, epicKey)
	}

	return nil
}

// ValidateAgentType validates the agent type value
func ValidateAgentType(agentType string) error {
	validTypes := []string{"frontend", "backend", "api", "testing", "devops", "general"}
	for _, valid := range validTypes {
		if agentType == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid agent type '%s'. Must be one of: %s", agentType, strings.Join(validTypes, ", "))
}

// ValidatePriority validates the priority value
func ValidatePriority(priority int) error {
	if priority < 1 || priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10, got %d", priority)
	}
	return nil
}
