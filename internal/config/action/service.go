package action

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

	// GetStatusActionPopulated returns action with template variables populated.
	// vars is a map of template variable names to values (e.g., from TaskPlaceholders).
	GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error)

	// GetAllActions returns all orchestrator actions indexed by status name
	GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)

	// ValidateActions checks that all actionable statuses have valid actions
	// Returns list of statuses missing actions or with invalid configuration
	ValidateActions(ctx context.Context) (*ValidationResult, error)

	// Reload forces reload of configuration from disk (useful after config changes)
	Reload(ctx context.Context) error
}

// Ensure DefaultActionService satisfies ActionService at compile time (AC-T2)
var _ ActionService = (*DefaultActionService)(nil)

// PopulatedAction is an orchestrator action with template variables replaced
type PopulatedAction struct {
	Action      string   `json:"action"`
	AgentType   string   `json:"agent_type,omitempty"`
	Provider    string   `json:"provider,omitempty"` // AI provider (e.g., "anthropic", "openai")
	Model       string   `json:"model,omitempty"`    // Model override (e.g., "o3", "claude-opus-4-5")
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

// StatusActionData holds the action data for a single status, as needed by DefaultActionService.
// This abstracts the StatusMetadata from the workflow package so action/ does not import it.
type StatusActionData struct {
	OrchestratorAction *OrchestratorAction
}

// WorkflowDataLoader is a function that loads workflow action data from a config path.
// It returns a map of status name -> StatusActionData.
// This breaks the circular dependency between action/ and root config/:
// root config provides the loader function, action/ uses it without importing root config.
type WorkflowDataLoader func(configPath string) map[string]StatusActionData

// DefaultActionService is the default implementation of ActionService
type DefaultActionService struct {
	mu         sync.RWMutex
	configPath string
	statusData map[string]StatusActionData
	loader     WorkflowDataLoader
}

// NewActionService creates a new action service using the given config path and workflow data loader.
// The loader function abstracts workflow config loading so action/ does not import root config.
func NewActionService(configPath string, loader WorkflowDataLoader) (*DefaultActionService, error) {
	service := &DefaultActionService{
		configPath: configPath,
		loader:     loader,
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

	if s.statusData == nil {
		return nil, errors.New("workflow config not loaded")
	}

	data, exists := s.statusData[status]
	if !exists {
		return nil, &StatusNotFoundError{Status: status}
	}

	// Return nil if no action defined (not an error)
	return data.OrchestratorAction, nil
}

// GetStatusActionPopulated retrieves action with template populated.
// vars is a map of template variable names to values (e.g., from TaskPlaceholders).
// If vars is nil, an empty map is used.
func (s *DefaultActionService) GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error) {
	action, err := s.GetStatusAction(ctx, status)
	if err != nil {
		return nil, err
	}

	if action == nil {
		return nil, nil // No action defined
	}

	// Use empty map if nil to avoid nil pointer issues
	if vars == nil {
		vars = make(map[string]string)
	}

	return action.ToPopulatedAction(vars), nil
}

// GetAllActions returns all actions indexed by status
func (s *DefaultActionService) GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.statusData == nil {
		return nil, errors.New("workflow config not loaded")
	}

	actions := make(map[string]*OrchestratorAction)
	for status, data := range s.statusData {
		if data.OrchestratorAction != nil {
			actions[status] = data.OrchestratorAction
		}
	}

	return actions, nil
}

// ValidateActions validates all orchestrator actions in config
func (s *DefaultActionService) ValidateActions(ctx context.Context) (*ValidationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.statusData == nil {
		return nil, errors.New("workflow config not loaded")
	}

	result := &ValidationResult{
		Valid:          true,
		MissingActions: []string{},
		InvalidActions: []InvalidAction{},
		Warnings:       []string{},
	}

	for status, data := range s.statusData {
		// Check if actionable status (ready_for_*) lacks action
		if strings.HasPrefix(status, "ready_for_") && data.OrchestratorAction == nil {
			result.MissingActions = append(result.MissingActions, status)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Status '%s' has no orchestrator_action defined", status))
		}

		// Validate action if present
		if data.OrchestratorAction != nil {
			if err := data.OrchestratorAction.Validate(); err != nil {
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
	data := s.loader(s.configPath)
	if data == nil {
		return fmt.Errorf("failed to load workflow config from %s", s.configPath)
	}

	s.mu.Lock()
	s.statusData = data
	s.mu.Unlock()

	return nil
}
