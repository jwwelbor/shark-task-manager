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
	"github.com/jwwelbor/shark-task-manager/internal/config/workflow"
)

// --- workflow/ types ---

// WorkflowConfig is an alias for workflow.WorkflowConfig.
type WorkflowConfig = workflow.WorkflowConfig

// StatusMetadata is an alias for workflow.StatusMetadata.
type StatusMetadata = workflow.StatusMetadata

// MultiLevelWorkflow is an alias for workflow.MultiLevelWorkflow.
type MultiLevelWorkflow = workflow.MultiLevelWorkflow

// WorkflowValidationError is an alias for workflow.WorkflowValidationError.
type WorkflowValidationError = workflow.WorkflowValidationError

// WorkflowValidationFinding is an alias for workflow.WorkflowValidationFinding.
type WorkflowValidationFinding = workflow.WorkflowValidationFinding

// --- workflow/ constants ---
// Go does not allow const X = pkg.X for string constants.
// These are re-declared with identical values; the canonical definition
// lives in workflow/schema.go.

// StartStatusKey defines initial statuses where new tasks begin.
const StartStatusKey = workflow.StartStatusKey

// CompleteStatusKey defines terminal statuses where tasks end.
const CompleteStatusKey = workflow.CompleteStatusKey

// AggregationStatusKey identifies statuses where an entity switches from
// its own workflow tracking to aggregating progress from children.
const AggregationStatusKey = workflow.AggregationStatusKey

// DefaultWorkflowVersion is the default version for workflow configs.
const DefaultWorkflowVersion = workflow.DefaultWorkflowVersion

// --- workflow/ functions ---

// LoadWorkflowConfig loads workflow configuration from .sharkconfig.json.
var LoadWorkflowConfig = workflow.LoadWorkflowConfig

// ClearWorkflowCache clears all workflow caches (multi-level and legacy).
var ClearWorkflowCache = workflow.ClearWorkflowCache

// GetWorkflowOrDefault loads workflow config or returns default if not configured.
var GetWorkflowOrDefault = workflow.GetWorkflowOrDefault

// LoadMultiLevelWorkflow loads all entity-level workflow configs.
var LoadMultiLevelWorkflow = workflow.LoadMultiLevelWorkflow

// LoadMultiLevelWorkflowOrDefault loads configs or returns defaults on failure.
var LoadMultiLevelWorkflowOrDefault = workflow.LoadMultiLevelWorkflowOrDefault

// ValidateWorkflow validates workflow configuration for correctness.
var ValidateWorkflow = workflow.ValidateWorkflow

// ValidateWorkflowFiles validates both .sharkconfig.json and .sharkworkflow.json.
var ValidateWorkflowFiles = workflow.ValidateWorkflowFiles

// ValidateTransition checks if a status transition is valid according to workflow.
var ValidateTransition = workflow.ValidateTransition

// DefaultWorkflow returns the backward-compatible default task workflow.
var DefaultWorkflow = workflow.DefaultWorkflow

// DefaultEpicWorkflow returns the default epic workflow.
var DefaultEpicWorkflow = workflow.DefaultEpicWorkflow

// DefaultFeatureWorkflow returns the default feature workflow.
var DefaultFeatureWorkflow = workflow.DefaultFeatureWorkflow

// DefaultBugWorkflow returns the default bug workflow.
var DefaultBugWorkflow = workflow.DefaultBugWorkflow

// DefaultChangeCardWorkflow returns the default change-card workflow.
var DefaultChangeCardWorkflow = workflow.DefaultChangeCardWorkflow

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
	wf := GetWorkflowOrDefault(configPath)
	if wf == nil {
		return nil
	}

	data := make(map[string]action.StatusActionData)
	for status, metadata := range wf.StatusMetadata {
		data[status] = action.StatusActionData{
			OrchestratorAction: metadata.OrchestratorAction,
		}
	}
	return data
}
