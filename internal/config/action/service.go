package action

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/entitytype"
)

// DefaultEntityType is the entity type that base ActionService methods
// (without an explicit ForEntity binding) resolve against. The action
// service historically only handled task statuses; this default preserves
// behavior for callers that still don't pass an entity type.
const DefaultEntityType = "task"

// NormalizeEntityType maps CLI/storage aliases to canonical workflow action
// slots. Workflow config stores change-card actions under "change", while
// key detection for CC-### returns "change_card".
func NormalizeEntityType(entityType string) string {
	return entitytype.WorkflowLevelOrSelf(entityType)
}

// ActionService provides access to orchestrator action configuration.
//
// In Shark 2.0 the per-entity workflow is the source of truth. Callers that
// know the entity type should narrow the service via ForEntity before
// calling the status methods, e.g.:
//
//	entitySvc := svc.ForEntity("feature")
//	pop, err := entitySvc.GetStatusActionPopulated(ctx, "active", vars)
//
// The bare-service methods (without ForEntity) resolve against
// DefaultEntityType ("task") for backward compatibility with task-only
// call sites that predate the multi-entity refactor.
type ActionService interface {
	// GetStatusAction returns the orchestrator action for a given status.
	// Returns nil if no action is defined for the status.
	GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error)

	// GetStatusActionPopulated returns action with template variables populated.
	// vars is a map of template variable names to values (e.g., from TaskPlaceholders).
	GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error)

	// GetAllActions returns all orchestrator actions indexed by status name.
	GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)

	// ValidateActions checks that all actionable statuses have valid actions.
	// Returns list of statuses missing actions or with invalid configuration.
	ValidateActions(ctx context.Context) (*ValidationResult, error)

	// Reload forces reload of configuration from disk (useful after config changes).
	Reload(ctx context.Context) error

	// ForEntity returns an ActionService view scoped to the given entity type
	// (e.g. "task", "feature", "epic", "bug", "change", "tech_debt"). All
	// subsequent status lookups on the returned view resolve against that
	// entity's workflow only. Status names from other entity workflows are
	// not visible to the view, which is what makes cross-entity status name
	// collisions (e.g. "completed") unambiguous.
	ForEntity(entityType string) ActionService
}

// Ensure DefaultActionService satisfies ActionService at compile time (AC-T2)
var _ ActionService = (*DefaultActionService)(nil)
var _ ActionService = (*entityActionView)(nil)

// PopulatedAction is an orchestrator action with template variables replaced
type PopulatedAction struct {
	Action      string   `json:"action"`
	AgentType   string   `json:"agent_type,omitempty"`
	Provider    string   `json:"provider,omitempty"` // AI provider (e.g., "anthropic", "openai")
	Model       string   `json:"model,omitempty"`    // Model override (e.g., "o3", "claude-opus-4-5")
	Effort      string   `json:"effort,omitempty"`   // Reasoning-effort override (low, medium, high, xhigh)
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
// It returns a nested map: entityType -> status -> StatusActionData. The outer key
// is one of "task", "feature", "epic", "bug", "change", "tech_debt", "sprint".
// Implementations that only have a single (task) workflow should return a map with
// just the "task" entry.
//
// The error return surfaces fatal configuration problems (e.g. .sharkconfig.json
// still pointing at a Shark 1.x legacy workflow file). Soft failures — missing
// files, optional entities not yet defined — return (data, nil); the caller will
// see a partial map and fill any remaining slots with hardcoded defaults.
//
// This breaks the circular dependency between action/ and root config/: root config
// provides the loader function, action/ uses it without importing root config.
type WorkflowDataLoader func(configPath string) (map[string]map[string]StatusActionData, error)

// DefaultActionService is the default implementation of ActionService.
//
// Internally it stores a per-entity map of status -> action. Lookups go
// through getStatusActionFor(entityType, status); bare-service methods use
// DefaultEntityType and the ForEntity-view methods use their bound type.
type DefaultActionService struct {
	mu         sync.RWMutex
	configPath string
	statusData map[string]map[string]StatusActionData
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

// ForEntity returns an ActionService view bound to entityType. All status
// lookups via the returned view resolve only against that entity's workflow.
func (s *DefaultActionService) ForEntity(entityType string) ActionService {
	return &entityActionView{parent: s, entityType: NormalizeEntityType(entityType)}
}

// getStatusActionFor is the single internal lookup path used by both the
// bare service and the entity view.
func (s *DefaultActionService) getStatusActionFor(entityType, status string) (*OrchestratorAction, error) {
	entityType = NormalizeEntityType(entityType)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.statusData == nil {
		return nil, errors.New("workflow config not loaded")
	}

	entityMap, ok := s.statusData[entityType]
	if !ok {
		// Entity not loaded — no workflow for this entity type, so the status
		// cannot resolve. Surface NotFound either way.
		return nil, &StatusNotFoundError{Status: status}
	}

	data, exists := entityMap[status]
	if !exists {
		return nil, &StatusNotFoundError{Status: status}
	}

	// Return nil if no action defined (not an error)
	return data.OrchestratorAction, nil
}

// GetStatusAction retrieves action for a status against the default entity type.
func (s *DefaultActionService) GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error) {
	return s.getStatusActionFor(DefaultEntityType, status)
}

// GetStatusActionPopulated retrieves action with template populated for the default entity type.
// vars is a map of template variable names to values (e.g., from TaskPlaceholders).
// If vars is nil, an empty map is used.
func (s *DefaultActionService) GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error) {
	return populateAction(s, DefaultEntityType, status, vars)
}

// populateAction is a small helper shared by the bare service and the
// entity view so both paths produce identical PopulatedAction shapes.
func populateAction(s *DefaultActionService, entityType, status string, vars map[string]string) (*PopulatedAction, error) {
	action, err := s.getStatusActionFor(entityType, status)
	if err != nil {
		return nil, err
	}
	if action == nil {
		return nil, nil
	}
	if vars == nil {
		vars = make(map[string]string)
	}
	return action.ToPopulatedAction(vars), nil
}

// GetAllActions returns all actions indexed by status for the default entity type.
func (s *DefaultActionService) GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error) {
	return s.getAllActionsFor(DefaultEntityType)
}

func (s *DefaultActionService) getAllActionsFor(entityType string) (map[string]*OrchestratorAction, error) {
	entityType = NormalizeEntityType(entityType)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.statusData == nil {
		return nil, errors.New("workflow config not loaded")
	}

	actions := make(map[string]*OrchestratorAction)
	entityMap, ok := s.statusData[entityType]
	if !ok {
		return actions, nil
	}
	for status, data := range entityMap {
		if data.OrchestratorAction != nil {
			actions[status] = data.OrchestratorAction
		}
	}
	return actions, nil
}

// ValidateActions validates all orchestrator actions across every loaded entity workflow.
// Status names are reported as <entityType>:<status> when an entity dimension is meaningful.
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

	for entityType, entityMap := range s.statusData {
		validateEntityMap(entityType, entityMap, true, result)
	}

	// Set overall validity
	if len(result.InvalidActions) > 0 {
		result.Valid = false
	}

	return result, nil
}

// validateEntityMap walks a single entity's status -> action map and appends any
// missing/invalid actions and warnings to result. When qualifyKeys is true,
// status keys reported in MissingActions, InvalidActions, and Warnings are
// prefixed with "<entityType>:" (except for the default entity type, which is
// always reported unqualified for back-compat). When qualifyKeys is false,
// raw status names are reported — used by entityActionView, which is already
// scoped to a single entity and would otherwise emit redundant prefixes.
func validateEntityMap(entityType string, entityMap map[string]StatusActionData, qualifyKeys bool, result *ValidationResult) {
	for status, data := range entityMap {
		key := status
		if qualifyKeys && entityType != DefaultEntityType {
			key = entityType + ":" + status
		}

		// Check if actionable status (ready_for_*) lacks action
		if strings.HasPrefix(status, "ready_for_") && data.OrchestratorAction == nil {
			result.MissingActions = append(result.MissingActions, key)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Status '%s' has no orchestrator_action defined", key))
		}

		// Validate action if present
		if data.OrchestratorAction != nil {
			if err := data.OrchestratorAction.Validate(); err != nil {
				result.Valid = false
				result.InvalidActions = append(result.InvalidActions, InvalidAction{
					Status: key,
					Error:  err.Error(),
				})
			}
		}
	}
}

// Reload reloads configuration from disk
func (s *DefaultActionService) Reload(ctx context.Context) error {
	data, err := s.loader(s.configPath)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("failed to load workflow config from %s", s.configPath)
	}

	s.mu.Lock()
	s.statusData = data
	s.mu.Unlock()

	return nil
}

// entityActionView is the ActionService facade returned by ForEntity.
// It does not duplicate state — every lookup goes back to the parent
// DefaultActionService with the bound entity type.
type entityActionView struct {
	parent     *DefaultActionService
	entityType string
}

func (v *entityActionView) GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error) {
	return v.parent.getStatusActionFor(v.entityType, status)
}

func (v *entityActionView) GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error) {
	return populateAction(v.parent, v.entityType, status, vars)
}

func (v *entityActionView) GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error) {
	return v.parent.getAllActionsFor(v.entityType)
}

func (v *entityActionView) ValidateActions(ctx context.Context) (*ValidationResult, error) {
	// Validate only this entity's actions.
	v.parent.mu.RLock()
	defer v.parent.mu.RUnlock()

	if v.parent.statusData == nil {
		return nil, errors.New("workflow config not loaded")
	}

	result := &ValidationResult{
		Valid:          true,
		MissingActions: []string{},
		InvalidActions: []InvalidAction{},
		Warnings:       []string{},
	}

	validateEntityMap(v.entityType, v.parent.statusData[v.entityType], false, result)

	if len(result.InvalidActions) > 0 {
		result.Valid = false
	}
	return result, nil
}

func (v *entityActionView) Reload(ctx context.Context) error {
	return v.parent.Reload(ctx)
}

func (v *entityActionView) ForEntity(entityType string) ActionService {
	return v.parent.ForEntity(entityType)
}
