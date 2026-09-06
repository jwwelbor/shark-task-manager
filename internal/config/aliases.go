package config

// aliases.go provides backward-compatible type aliases and function re-exports
// for symbols moved to sub-packages. All existing callers continue to use the
// config.XxxType and config.XxxFunc names unchanged.
//
// Pattern follows internal/repository/aliases.go (established in E07-F36).

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	cfgtemplate "github.com/jwwelbor/shark-task-manager/internal/config/template"
	"github.com/jwwelbor/shark-task-manager/internal/config/validation"
	"github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/pathutil"
)

// ErrSharkDataPathEscapes is returned by ResolveSharkDataRoot when a relative
// shark_data_path resolves outside the project root. Callers branch on it with
// errors.Is rather than string-matching the message.
var ErrSharkDataPathEscapes = errors.New("shark_data_path escapes the project root")

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

// Step is an alias for workflow.Step (Shark 2.x route-based schema, E35-F01).
type Step = workflow.Step

// MultiLevelWorkflow is an alias for workflow.MultiLevelWorkflow.
type MultiLevelWorkflow = workflow.MultiLevelWorkflow

// WorkflowValidationError is an alias for workflow.WorkflowValidationError.
type WorkflowValidationError = workflow.WorkflowValidationError

// WorkflowValidationFinding is an alias for workflow.WorkflowValidationFinding.
type WorkflowValidationFinding = workflow.WorkflowValidationFinding

// NoCandidateError is an alias for workflow.NoCandidateError (named selectors,
// selectors.go).
type NoCandidateError = workflow.NoCandidateError

// AmbiguousSelectionError is an alias for workflow.AmbiguousSelectionError
// (named selectors, selectors.go).
type AmbiguousSelectionError = workflow.AmbiguousSelectionError

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

// ResultContractLegacy and ResultContractGateResultV1 are the REQ-F-006
// `result_contract` values a route step can select.
const (
	ResultContractLegacy       = workflow.ResultContractLegacy
	ResultContractGateResultV1 = workflow.ResultContractGateResultV1
)

// KnownWorkflowLevels lists every entity workflow level shark supports in
// canonical display order.
var KnownWorkflowLevels = workflow.KnownLevels

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

// ValidateWorkflowFiles validates supported workflow configuration sources.
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
// It delegates all directory, index-file, inline, and embedded-default
// resolution to LoadMultiLevelWorkflowFromBytes, then projects every entity
// workflow into the action-service shape. Calling GetWorkflowForLevel fills
// omitted slots from the same canonical defaults used by workflow.Service.
//
// Hard error: when `workflow_config` explicitly points at a deprecated JSON
// workflow file, this loader fails with a migration hint. The minimal fix is
// removing workflow_config; if a root .sharkworkflow.json is present, remove or
// rename it too before expecting embedded defaults. The editable-files fix is
// `shark admin install-shark-data`. The two-path fallback that used to silently
// load the JSON file's task slot was removed because it silently dropped every
// other entity's workflow on the floor and caused status lookups to misroute
// (see B020).
func defaultWorkflowDataLoader(configPath string) (map[string]map[string]action.StatusActionData, error) {
	// TD-023: read .sharkconfig.json once, then use the bytes-accepting parser
	// so ActionService and every other workflow consumer share one source
	// resolution path.
	configBytes, _ := readConfigFile(configPath)

	configuredWorkflow := readWorkflowConfigField(configBytes)
	if workflow.IsDeprecatedWorkflowConfigTarget(configuredWorkflow) {
		return nil, workflow.DeprecatedWorkflowConfigJSONError()
	}

	defaultWorkflowDir := ""
	if configuredWorkflow == "" {
		projectRoot := filepath.Dir(configPath)
		sharkDataRoot, rootErr := ResolveSharkDataRoot(projectRoot, configBytes)
		if rootErr != nil {
			slog.Warn("invalid shark_data_path; falling back to default bundle root",
				"error", rootErr, "default", DefaultSharkDataPath)
			sharkDataRoot = filepath.Join(projectRoot, DefaultSharkDataPath)
		}
		defaultWorkflowDir = filepath.Join(sharkDataRoot, workflow.YAMLWorkflowDir)
	}

	// Reload is documented to observe on-disk workflow changes. The parser's
	// cache is correct for normal consumers, but ActionService calls this loader
	// again on Reload, so clear it at this boundary before parsing.
	workflow.ClearWorkflowCache()
	mlw, err := workflow.LoadMultiLevelWorkflowFromBytesWithDefaultWorkflowDir(configPath, configBytes, defaultWorkflowDir)
	if err != nil {
		return nil, err
	}

	out := make(map[string]map[string]action.StatusActionData, len(entityTypesForLoader()))
	for _, entityType := range entityTypesForLoader() {
		out[entityType] = workflowToStatusActionData(mlw.GetWorkflowForLevel(entityType))
	}

	return out, nil
}

// resolveWorkflowDir picks the directory of per-entity workflow YAMLs to load.
//
// Resolution:
//  1. Read `workflow_config` from .sharkconfig.json bytes (raw decode to
//     avoid pulling in the entire Manager). Empty/missing → use the default
//     `<projectRoot>/shark-data/workflow/`.
//  2. Return the resolved canonical YAML directory and its derived overrides
//     directory. The YAML loader handles missing directories while embedded
//     defaults backstop missing workflow slots.
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
func resolveWorkflowDir(projectRoot string, configBytes []byte) (workflowDir, overridesDir string) {
	configured := readWorkflowConfigField(configBytes)

	if configured == "" {
		// No explicit workflow_config: derive the default workflow directory
		// from the configured shark_data_path bundle root
		// (<shark_data_path>/workflow). Routed through the single
		// ResolveSharkDataRoot resolver so "~/" expansion, absolute-path
		// honoring, and the in-project escape check are applied identically
		// to `shark validate` and prompt resolution. shark_data_path defaults
		// to "shark-data", preserving the historical
		// <projectRoot>/shark-data/workflow default. An explicit
		// workflow_config always wins (handled in the else branch below).
		workflowDir = filepath.Join(resolveSharkDataRootOrDefault(projectRoot, configBytes), "workflow")
	} else {
		// Honor absolute paths verbatim; resolve relative paths against the
		// project root (the directory holding .sharkconfig.json).
		if filepath.IsAbs(configured) {
			workflowDir = configured
		} else {
			workflowDir = filepath.Join(projectRoot, configured)
		}
	}

	overridesDir = filepath.Join(filepath.Dir(workflowDir), "overrides", "workflow")
	return workflowDir, overridesDir
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

// readSharkDataPathField extracts the `shark_data_path` field from raw
// .sharkconfig.json bytes without going through the full Manager. Mirrors
// readWorkflowConfigField. Returns the configured value, or the default
// "shark-data" when the file is missing, malformed, or the field is
// absent/empty. The default is baked in here (rather than returning "") so
// callers that combine this with path joins always get a usable bundle root.
func readSharkDataPathField(raw []byte) string {
	if len(raw) == 0 {
		return DefaultSharkDataPath
	}
	var probe struct {
		SharkDataPath string `json:"shark_data_path"`
	}
	if err := jsonUnmarshal(raw, &probe); err != nil {
		return DefaultSharkDataPath
	}
	if probe.SharkDataPath == "" {
		return DefaultSharkDataPath
	}
	return probe.SharkDataPath
}

// ResolveSharkDataRoot returns the absolute path to the content-bundle root
// (skills/, prompts/, agents/, overrides/) selected by `shark_data_path` in
// .sharkconfig.json, defaulting to <projectRoot>/shark-data.
//
// configBytes is the already-read contents of .sharkconfig.json (pass nil/
// empty when absent — the default is used). projectRoot is the directory
// holding .sharkconfig.json.
//
// Resolution rules:
//   - A leading "~/" is expanded to the user's home directory (shared-bundle
//     convention, matching template_directory resolution).
//   - Absolute shark_data_path: honored verbatim (shared-bundle parity with
//     workflow_config trust).
//   - Relative shark_data_path: resolved against projectRoot, then cleaned and
//     verified to stay WITHIN projectRoot. A relative path that escapes the
//     project root via `..` is rejected with ErrSharkDataPathEscapes.
//
// This is the single source of truth for interpreting shark_data_path; the
// workflow-dir and prompts-dir resolvers derive from it so all consumers agree
// on one bundle root for any given config value.
func ResolveSharkDataRoot(projectRoot string, configBytes []byte) (string, error) {
	dataPath := pathutil.ExpandHome(readSharkDataPathField(configBytes))

	if filepath.IsAbs(dataPath) {
		return filepath.Clean(dataPath), nil
	}

	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root %q: %w", projectRoot, err)
	}
	absRoot = filepath.Clean(absRoot)

	resolved := filepath.Clean(filepath.Join(absRoot, dataPath))

	// Reject paths that escape the project root via `..`. filepath.Rel gives a
	// robust, cross-platform containment check: the resolved path is in-bounds
	// only when its path relative to the root neither is ".." nor starts with
	// "../". This avoids the string-prefix edge cases (root == "/" or "C:\",
	// and sibling-prefix collisions like /foo/bar-evil vs /foo/bar).
	rel, err := filepath.Rel(absRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf(
			"%w: %q (root %q); use an absolute path for shared bundles",
			ErrSharkDataPathEscapes, dataPath, absRoot,
		)
	}

	return resolved, nil
}

// resolveSharkDataRootOrDefault resolves the bundle root for internal callers
// that have no error channel (e.g. workflow-dir resolution). On any resolution
// failure — most notably an escaping relative shark_data_path — it logs a
// warning and falls back to <projectRoot>/shark-data so resolution fails safe.
// `shark validate` calls ResolveSharkDataRoot directly and surfaces the same
// misconfiguration as a hard error, so the problem is never silently swallowed.
func resolveSharkDataRootOrDefault(projectRoot string, configBytes []byte) string {
	root, err := ResolveSharkDataRoot(projectRoot, configBytes)
	if err != nil {
		slog.Warn("invalid shark_data_path; falling back to default bundle root",
			"error", err, "default", DefaultSharkDataPath)
		return filepath.Join(projectRoot, DefaultSharkDataPath)
	}
	return root
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
