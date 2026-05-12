package config

// aliases.go provides backward-compatible type aliases and function re-exports
// for symbols moved to sub-packages. All existing callers continue to use the
// config.XxxType and config.XxxFunc names unchanged.
//
// Pattern follows internal/repository/aliases.go (established in E07-F36).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	cfgtemplate "github.com/jwwelbor/shark-task-manager/internal/config/template"
	"github.com/jwwelbor/shark-task-manager/internal/config/validation"
	"github.com/jwwelbor/shark-task-manager/internal/config/workflow"
)

// jsonUnmarshal is a tiny indirection so the workflow_config raw decode in
// resolveWorkflowDir can be replaced in tests without dragging encoding/json
// imports into hot paths elsewhere. It mirrors json.Unmarshal exactly.
var jsonUnmarshal = json.Unmarshal

// readConfigFile is the os.ReadFile call defaultWorkflowDataLoader uses to
// pull .sharkconfig.json once per invocation (TD-023). Exposed as a package
// variable so tests can wrap it with a counter and assert that downstream
// helpers never reach back to the filesystem on their own.
var readConfigFile = os.ReadFile

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

// TechDebtPlaceholders creates a map of template placeholders from a TechDebt.
var TechDebtPlaceholders = cfgtemplate.TechDebtPlaceholders

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
// This is a backward-compatible wrapper that preserves the original function signature
// (map[string]StatusMetadata), while delegating to the action sub-package implementation
// which accepts map[string]*OrchestratorAction.
var ValidateAllOrchestratorActions = func(statusMetadata map[string]StatusMetadata) []*OrchestratorValidationError {
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

// entityTypesForLoader is the canonical list of entity slots this loader
// covers. Delegates to workflow.EntityTypes so adding a new entity type only
// requires extending MultiLevelWorkflow + EntityTypes — no edit here.
func entityTypesForLoader() []string {
	return workflow.EntityTypes()
}

// defaultWorkflowDataLoader loads per-entity workflow action data.
//
// Resolution order:
//  1. The `workflow_config` field in `.sharkconfig.json` (default
//     `shark-data/workflow/`) is read as the workflow *directory* and
//     per-entity YAMLs are loaded from it. Override files at
//     `<workflowDir>/../overrides/workflow/<entity>.yaml` (when the workflow
//     dir lives under a `shark-data/`-shaped tree) fully replace the default
//     YAML for that entity.
//  2. Inline workflows in `.sharkconfig.json` (multi-level
//     `bug_workflow`/`feature_workflow`/etc. blocks, plus the legacy
//     top-level `status_flow` for the task slot) backstop any entity the
//     YAML directory did not cover. This is what fresh checkouts and the
//     legacy test fixtures rely on.
//  3. Hardcoded defaults fill any entity slot left empty (e.g. tech_debt,
//     sprint — Shark 2.0 may not have shipped YAMLs for every entity).
//
// projectRoot is derived from configPath (the directory containing
// .sharkconfig.json).
//
// Hard error: when `workflow_config` explicitly points at a Shark 1.x JSON
// workflow file (e.g. `shark-templates/.sharkworkflow.json`), this loader
// fails with a directive to run `shark init`. The two-path fallback that
// used to silently load the JSON file's task slot was removed because it
// silently dropped every other entity's workflow on the floor and caused
// status lookups to misroute (see B020).
func defaultWorkflowDataLoader(configPath string) (map[string]map[string]action.StatusActionData, error) {
	projectRoot := filepath.Dir(configPath)

	// TD-023: read .sharkconfig.json bytes once and thread them through every
	// helper that would otherwise re-read the same file. The previous code
	// hit os.ReadFile 2-3× per call (resolveWorkflowDir, the legacy-error
	// helper, and LoadMultiLevelWorkflowOrDefault). On the cold-startup path
	// behind ActionService's sync.Once, that adds gratuitous syscalls.
	//
	// configBytes may be nil/empty when .sharkconfig.json is missing — every
	// downstream helper tolerates that (resolveWorkflowDir defaults the dir,
	// the parser returns an empty MultiLevelWorkflow). An I/O error other
	// than os.IsNotExist also collapses to nil bytes; the helpers behave the
	// same as if the file were absent, which matches the prior best-effort
	// semantics of the individual readers.
	configBytes, _ := readConfigFile(configPath)

	workflowDir, overridesDir, isLegacyFile := resolveWorkflowDir(projectRoot, configBytes)

	// Reject explicitly-configured legacy JSON workflow files. resolveWorkflowDir
	// signals this case with isLegacyFile=true AND workflowDir=="". The
	// alternate isLegacyFile=true case (no workflow_config set, default dir
	// missing) leaves workflowDir non-empty and falls through to Pass 1/2/3
	// so fresh projects and inline-config tests keep working.
	if isLegacyFile && workflowDir == "" {
		configured := readWorkflowConfigField(configBytes)
		return nil, fmt.Errorf(
			".sharkconfig.json field \"workflow_config\" = %q is a Shark 1.x "+
				"JSON workflow file; Shark 2.0 uses per-entity YAML in "+
				"shark-data/workflow/. Run `shark init` to materialize the "+
				"shark-data/ tree and migrate the field.",
			configured,
		)
	}

	out := map[string]map[string]action.StatusActionData{}

	entityTypes := entityTypesForLoader()

	// Pass 1: per-entity YAML at the configured workflow_config directory.
	if workflowDir != "" {
		if mlw, err := workflow.LoadMultiLevelWorkflowFromYAMLDir(workflowDir, overridesDir); err == nil && mlw != nil {
			for _, entityType := range entityTypes {
				if wf := mlw.GetByType(entityType); wf != nil {
					out[entityType] = workflowToStatusActionData(wf)
				}
			}
		}
	}

	// Pass 2: inline multi-level workflows from .sharkconfig.json itself.
	// Covers fresh checkouts that haven't run `shark init` yet and the
	// legacy top-level `status_flow` shape still used by many tests.
	// Use the bytes-accepting variant so we don't re-read the file.
	if mlw := workflow.LoadMultiLevelWorkflowOrDefaultFromBytes(configPath, configBytes); mlw != nil {
		for _, entityType := range entityTypes {
			if _, already := out[entityType]; already {
				continue
			}
			if wf := mlw.GetByType(entityType); wf != nil {
				out[entityType] = workflowToStatusActionData(wf)
			}
		}
	}

	// Pass 3: hardcoded defaults for any entity still missing. Uses
	// GetWorkflowForLevel as the single source of truth for entity-type →
	// default mapping (it falls back through defaultForType internally).
	emptyMLW := &workflow.MultiLevelWorkflow{}
	for _, entityType := range entityTypes {
		if _, ok := out[entityType]; ok {
			continue
		}
		if wf := emptyMLW.GetWorkflowForLevel(entityType); wf != nil {
			out[entityType] = workflowToStatusActionData(wf)
		}
	}

	return out, nil
}

// resolveWorkflowDir picks the directory of per-entity workflow YAMLs to load.
//
// Resolution:
//  1. Read `workflow_config` from .sharkconfig.json bytes (raw decode to
//     avoid pulling in the entire Manager). Empty/missing → use the default
//     `<projectRoot>/shark-data/workflow/`.
//  2. Stat the resolved path:
//     - Directory → return (workflowDir, derived overridesDir, false).
//     - File → treat as legacy JSON config; return ("", "", true) so the
//     caller skips Pass 1 and falls back to JSON loading.
//     - Missing → return the default path anyway so the loader can stat it
//     and silently produce zero YAMLs, leaving Pass 3 (hardcoded defaults)
//     to provide working workflows. This keeps fresh projects working
//     without an init.
//
// overridesDir defaults to `<dataDir>/overrides/workflow/` where dataDir is
// the parent of workflowDir. When workflow_config is set to a custom path
// outside a shark-data/-shaped tree, the overrides dir is computed the same
// way and may simply not exist — in which case override lookup silently
// no-ops.
//
// configBytes is the already-read contents of .sharkconfig.json. Pass nil/
// empty when the file is missing — both produce the same fallback behavior
// (default workflow dir). This signature change (was: configPath) is part of
// TD-023's "read .sharkconfig.json once" pass.
func resolveWorkflowDir(projectRoot string, configBytes []byte) (workflowDir, overridesDir string, isLegacyFile bool) {
	const defaultRel = "shark-data/workflow"

	configured := readWorkflowConfigField(configBytes)

	if configured == "" {
		workflowDir = filepath.Join(projectRoot, defaultRel)
	} else {
		// Honor absolute paths verbatim; resolve relative paths against the
		// project root (the directory holding .sharkconfig.json).
		if filepath.IsAbs(configured) {
			workflowDir = configured
		} else {
			workflowDir = filepath.Join(projectRoot, configured)
		}
	}

	info, err := os.Stat(workflowDir)
	switch {
	case err == nil && info.IsDir():
		// Directory: derive an overrides path next to it.
		overridesDir = filepath.Join(filepath.Dir(workflowDir), "overrides", "workflow")
		return workflowDir, overridesDir, false
	case err == nil && !info.IsDir():
		// File: legacy .sharkworkflow.json case. Signal caller to fall back.
		return "", "", true
	default:
		// Doesn't exist (or stat error). If the user explicitly pointed
		// workflow_config at this path, we still try Pass 1 (the YAML
		// loader is silent on missing dirs) — but if they didn't, the
		// default shark-data/workflow/ may simply not exist yet because
		// the project never ran `shark init`. In that case fall back to
		// the legacy JSON loader so inline `.sharkconfig.json` workflows
		// keep working (tests, fresh checkouts, etc.).
		overridesDir = filepath.Join(filepath.Dir(workflowDir), "overrides", "workflow")
		if configured == "" {
			return workflowDir, overridesDir, true
		}
		return workflowDir, overridesDir, false
	}
}

// readWorkflowConfigField extracts the `workflow_config` field from raw
// .sharkconfig.json bytes without going through the full Manager. Returns
// "" when the file is missing, malformed, or the field is absent.
func readWorkflowConfigField(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var probe struct {
		WorkflowConfig string `json:"workflow_config"`
	}
	if err := jsonUnmarshal(raw, &probe); err != nil {
		return ""
	}
	return probe.WorkflowConfig
}

// workflowToStatusActionData flattens a WorkflowConfig's StatusMetadata into the
// status -> StatusActionData shape the action service consumes.
func workflowToStatusActionData(wf *workflow.WorkflowConfig) map[string]action.StatusActionData {
	data := make(map[string]action.StatusActionData, len(wf.StatusMetadata))
	for status, metadata := range wf.StatusMetadata {
		data[status] = action.StatusActionData{
			OrchestratorAction: metadata.OrchestratorAction,
		}
	}
	return data
}
