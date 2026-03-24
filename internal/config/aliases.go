package config

// aliases.go provides backward-compatible type aliases and function re-exports
// for symbols moved to sub-packages. All existing callers continue to use the
// config.XxxType and config.XxxFunc names unchanged.
//
// Pattern follows internal/repository/aliases.go (established in E07-F36).

import (
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	cfgtemplate "github.com/jwwelbor/shark-task-manager/internal/config/template"
	"github.com/jwwelbor/shark-task-manager/internal/config/validation"
)

// --- validation/ types ---

// OrchestratorValidationError is an alias for validation.OrchestratorValidationError.
type OrchestratorValidationError = validation.OrchestratorValidationError

// ValidationError is an alias for validation.ValidationError (which is itself
// an alias for OrchestratorValidationError).
type ValidationError = validation.ValidationError

// --- template/ types ---

// TemplateEnrichmentData is an alias for template.TemplateEnrichmentData.
type TemplateEnrichmentData = cfgtemplate.TemplateEnrichmentData

// TemplateEnrichmentRepository is an alias for template.TemplateEnrichmentRepository.
type TemplateEnrichmentRepository = cfgtemplate.TemplateEnrichmentRepository

// DocumentRepository is an alias for template.DocumentRepository.
type DocumentRepository = cfgtemplate.DocumentRepository

// FeatureRelationshipRepository is an alias for template.FeatureRelationshipRepository.
type FeatureRelationshipRepository = cfgtemplate.FeatureRelationshipRepository

// EpicRelationshipRepository is an alias for template.EpicRelationshipRepository.
type EpicRelationshipRepository = cfgtemplate.EpicRelationshipRepository

// TaskRelationshipRepository is an alias for template.TaskRelationshipRepository.
type TaskRelationshipRepository = cfgtemplate.TaskRelationshipRepository

// --- template/ functions ---

// EntityPlaceholders creates a map of template placeholders from any Entity.
var EntityPlaceholders = cfgtemplate.EntityPlaceholders

// TaskPlaceholders creates a map of template placeholders from a Task.
var TaskPlaceholders = cfgtemplate.TaskPlaceholders

// FeaturePlaceholders creates a map of template placeholders from a Feature.
var FeaturePlaceholders = cfgtemplate.FeaturePlaceholders

// EpicPlaceholders creates a map of template placeholders from an Epic.
var EpicPlaceholders = cfgtemplate.EpicPlaceholders

// BugPlaceholders creates a map of template placeholders from a Bug.
var BugPlaceholders = cfgtemplate.BugPlaceholders

// ChangeCardPlaceholders creates a map of template placeholders from a ChangeCard.
var ChangeCardPlaceholders = cfgtemplate.ChangeCardPlaceholders

// TaskPlaceholdersWithRelated extends TaskPlaceholders with relationship data.
var TaskPlaceholdersWithRelated = cfgtemplate.TaskPlaceholdersWithRelated

// FeaturePlaceholdersWithRelated extends FeaturePlaceholders with relationship data.
var FeaturePlaceholdersWithRelated = cfgtemplate.FeaturePlaceholdersWithRelated

// EpicPlaceholdersWithRelated extends EpicPlaceholders with relationship data.
var EpicPlaceholdersWithRelated = cfgtemplate.EpicPlaceholdersWithRelated

// ApplyEnrichmentData merges enrichment data into the placeholder map.
var ApplyEnrichmentData = cfgtemplate.ApplyEnrichmentData

// ParseEpicKeyFromEntityKey extracts the epic key (E##) from a task or feature key.
var ParseEpicKeyFromEntityKey = cfgtemplate.ParseEpicKeyFromEntityKey

// ParseFeatureKeyFromTaskKey extracts the feature key (E##-F##) from a task key.
var ParseFeatureKeyFromTaskKey = cfgtemplate.ParseFeatureKeyFromTaskKey

// --- action/ types ---

// OrchestratorAction defines the action to take when a task enters a status.
type OrchestratorAction = action.OrchestratorAction

// PopulatedAction is an orchestrator action with template variables replaced.
type PopulatedAction = action.PopulatedAction

// ActionService provides access to orchestrator action configuration.
type ActionService = action.ActionService

// DefaultActionService is the default implementation of ActionService.
type DefaultActionService = action.DefaultActionService

// ValidationResult contains status action validation results.
type ValidationResult = action.ValidationResult

// InvalidAction describes an action that failed validation.
type InvalidAction = action.InvalidAction

// StatusNotFoundError indicates a status doesn't exist in config.
type StatusNotFoundError = action.StatusNotFoundError

// MockActionService is a mock implementation of ActionService for testing.
type MockActionService = action.MockActionService

// StatusActionData holds action data for a single status.
type StatusActionData = action.StatusActionData

// WorkflowDataLoader is a function that loads workflow action data from a config path.
type WorkflowDataLoader = action.WorkflowDataLoader

// --- action/ constants ---

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
var ValidActionTypes = action.ValidActionTypes

// --- action/ functions ---

// ValidateAllOrchestratorActions validates all orchestrator actions in status metadata.
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
func NewActionService(configPath string) (*DefaultActionService, error) {
	return action.NewActionService(configPath, defaultWorkflowDataLoader)
}

// defaultWorkflowDataLoader loads workflow data using GetWorkflowOrDefault.
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
