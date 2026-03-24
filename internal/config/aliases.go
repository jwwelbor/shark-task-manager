package config

// aliases.go provides backward-compatible re-exports for symbols moved to sub-packages.
// All 97+ files importing internal/config continue to work without import changes.
//
// Pattern established by E07-F36 (internal/repository/aliases.go).

import (
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

// --- action/ types ---

// OrchestratorAction defines the action to take when a task enters a status.
// Canonical definition: internal/config/action/orchestrator.go
type OrchestratorAction = action.OrchestratorAction

// PopulatedAction is an orchestrator action with template variables replaced.
// Canonical definition: internal/config/action/service.go
type PopulatedAction = action.PopulatedAction

// ActionService provides access to orchestrator action configuration.
// Canonical definition: internal/config/action/service.go
type ActionService = action.ActionService

// DefaultActionService is the default implementation of ActionService.
// Canonical definition: internal/config/action/service.go
type DefaultActionService = action.DefaultActionService

// ValidationResult contains status action validation results.
// Canonical definition: internal/config/action/service.go
type ValidationResult = action.ValidationResult

// InvalidAction describes an action that failed validation.
// Canonical definition: internal/config/action/service.go
type InvalidAction = action.InvalidAction

// StatusNotFoundError indicates a status doesn't exist in config.
// Canonical definition: internal/config/action/service.go
type StatusNotFoundError = action.StatusNotFoundError

// MockActionService is a mock implementation of ActionService for testing.
// Canonical definition: internal/config/action/mock_service.go
type MockActionService = action.MockActionService

// OrchestratorValidationError provides detailed context for orchestrator action configuration errors.
// NOTE: This is temporarily defined in action/ until validation/ sub-package is extracted (Task T-E07-F37-001).
// Canonical definition: internal/config/action/validation_error.go (temporary)
type OrchestratorValidationError = action.OrchestratorValidationError

// ValidationError is a backward-compatible alias for OrchestratorValidationError.
type ValidationError = action.OrchestratorValidationError

// StatusActionData holds action data for a single status.
// Canonical definition: internal/config/action/service.go
type StatusActionData = action.StatusActionData

// WorkflowDataLoader is a function that loads workflow action data from a config path.
// Canonical definition: internal/config/action/service.go
type WorkflowDataLoader = action.WorkflowDataLoader

// --- action/ constants ---
// Go does not allow const X = pkg.X for string constants.
// Re-declared with identical values; canonical source: internal/config/action/orchestrator.go

const (
	ActionSpawnAgent    = action.ActionSpawnAgent
	ActionPause         = action.ActionPause
	ActionWaitForTriage = action.ActionWaitForTriage
	ActionArchive       = action.ActionArchive
	ActionAdvanceStatus = action.ActionAdvanceStatus
	ActionCheckOrResume = action.ActionCheckOrResume
	ActionCascade       = action.ActionCascade
)

// --- action/ variables ---

// ValidActionTypes defines the allowed action types.
// Canonical definition: internal/config/action/orchestrator.go
var ValidActionTypes = action.ValidActionTypes

// --- action/ functions ---

// ValidateAllOrchestratorActions validates all orchestrator actions in status metadata.
// Canonical definition: internal/config/action/orchestrator.go
//
// NOTE: The signature changed from map[string]StatusMetadata to map[string]*OrchestratorAction
// to break the circular dependency between action/ and workflow types.
// A backward-compatible wrapper ValidateAllOrchestratorActionsFromMetadata is provided below.
var ValidateAllOrchestratorActions = action.ValidateAllOrchestratorActions

// ValidateAllOrchestratorActionsFromMetadata is a backward-compatible wrapper that accepts
// map[string]StatusMetadata (the original signature) and extracts the OrchestratorAction pointers.
func ValidateAllOrchestratorActionsFromMetadata(statusMetadata map[string]StatusMetadata) []*OrchestratorValidationError {
	actionsByStatus := make(map[string]*OrchestratorAction)
	for name, meta := range statusMetadata {
		actionsByStatus[name] = meta.OrchestratorAction
	}
	return action.ValidateAllOrchestratorActions(actionsByStatus)
}

// NewActionService creates a new action service.
// This wrapper provides a simplified signature matching the original API by injecting
// the default workflow data loader that uses GetWorkflowOrDefault from root config.
func NewActionService(configPath string) (*DefaultActionService, error) {
	return action.NewActionService(configPath, defaultWorkflowDataLoader)
}

// defaultWorkflowDataLoader loads workflow data using GetWorkflowOrDefault.
// This bridges the gap between action/ (which cannot import root config)
// and the workflow loading functions that live in root config.
func defaultWorkflowDataLoader(configPath string) map[string]action.StatusActionData {
	workflow := GetWorkflowOrDefault(configPath)
	if workflow == nil {
		return nil
	}

	data := make(map[string]action.StatusActionData)
	for status, metadata := range workflow.StatusMetadata {
		data[status] = action.StatusActionData{
			OrchestratorAction: metadata.OrchestratorAction,
		}
	}
	return data
}
