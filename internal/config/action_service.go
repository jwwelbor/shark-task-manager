package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ActionService provides access to orchestrator action configuration
type ActionService interface {
	// GetStatusAction returns the orchestrator action for a given status
	// Returns nil if no action is defined for the status
	GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error)

	// GetStatusActionPopulated returns action with template variables populated
	GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error)

	// GetAllActions returns all orchestrator actions indexed by status name
	GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)

	// ValidateActions checks that all actionable statuses have valid actions
	// Returns list of statuses missing actions or with invalid configuration
	ValidateActions(ctx context.Context) (*ValidationResult, error)

	// Reload forces reload of configuration from disk (useful after config changes)
	Reload(ctx context.Context) error
}

// PopulatedAction is an orchestrator action with template variables replaced
type PopulatedAction struct {
	Action      string   `json:"action"`
	AgentType   string   `json:"agent_type,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Instruction string   `json:"instruction"` // Template populated
}

// ValidationResult contains status action validation results
type ValidationResult struct {
	Valid          bool            `json:"valid"`
	MissingActions []string        `json:"missing_actions,omitempty"` // Statuses without actions
	InvalidActions []InvalidAction `json:"invalid_actions,omitempty"` // Actions with validation errors
	Warnings       []string        `json:"warnings,omitempty"`        // Non-fatal issues
}

// InvalidAction describes an action that failed validation
type InvalidAction struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// StatusNotFoundError indicates a status doesn't exist in config
type StatusNotFoundError struct {
	Status string
}

func (e *StatusNotFoundError) Error() string {
	return fmt.Sprintf("status '%s' not found in workflow configuration", e.Status)
}

// DefaultActionService is the default implementation of ActionService
type DefaultActionService struct {
	mu         sync.RWMutex
	configPath string
	workflow   *WorkflowConfig
}

// NewActionService creates a new action service
func NewActionService(configPath string) (*DefaultActionService, error) {
	service := &DefaultActionService{
		configPath: configPath,
	}

	// Load initial config
	if err := service.Reload(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	return service, nil
}

// GetStatusAction retrieves action for a status
func (s *DefaultActionService) GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.workflow == nil {
		return nil, errors.New("workflow config not loaded")
	}

	metadata, exists := s.workflow.StatusMetadata[status]
	if !exists {
		return nil, &StatusNotFoundError{Status: status}
	}

	// Return nil if no action defined (not an error)
	return metadata.OrchestratorAction, nil
}

// GetStatusActionPopulated retrieves action with template populated
func (s *DefaultActionService) GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error) {
	action, err := s.GetStatusAction(ctx, status)
	if err != nil {
		return nil, err
	}

	if action == nil {
		return nil, nil // No action defined
	}

	// Populate template
	instruction := action.PopulateTemplate(taskID)

	return &PopulatedAction{
		Action:      action.Action,
		AgentType:   action.AgentType,
		Skills:      action.Skills,
		Instruction: instruction,
	}, nil
}

// GetAllActions returns all actions indexed by status
func (s *DefaultActionService) GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.workflow == nil {
		return nil, errors.New("workflow config not loaded")
	}

	actions := make(map[string]*OrchestratorAction)
	for status, metadata := range s.workflow.StatusMetadata {
		if metadata.OrchestratorAction != nil {
			actions[status] = metadata.OrchestratorAction
		}
	}

	return actions, nil
}

// ValidateActions validates all orchestrator actions in config
func (s *DefaultActionService) ValidateActions(ctx context.Context) (*ValidationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.workflow == nil {
		return nil, errors.New("workflow config not loaded")
	}

	result := &ValidationResult{
		Valid:          true,
		MissingActions: []string{},
		InvalidActions: []InvalidAction{},
		Warnings:       []string{},
	}

	for status, metadata := range s.workflow.StatusMetadata {
		// Check if actionable status (ready_for_*) lacks action
		if strings.HasPrefix(status, "ready_for_") && metadata.OrchestratorAction == nil {
			result.MissingActions = append(result.MissingActions, status)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Status '%s' has no orchestrator_action defined", status))
		}

		// Validate action if present
		if metadata.OrchestratorAction != nil {
			if err := metadata.OrchestratorAction.Validate(); err != nil {
				result.Valid = false
				result.InvalidActions = append(result.InvalidActions, InvalidAction{
					Status: status,
					Error:  err.Error(),
				})
			}
		}
	}

	// Set overall validity
	if len(result.InvalidActions) > 0 {
		result.Valid = false
	}

	return result, nil
}

// Reload reloads configuration from disk
func (s *DefaultActionService) Reload(ctx context.Context) error {
	workflow := GetWorkflowOrDefault(s.configPath)
	if workflow == nil {
		return fmt.Errorf("failed to load workflow config from %s", s.configPath)
	}

	s.mu.Lock()
	s.workflow = workflow
	s.mu.Unlock()

	return nil
}
