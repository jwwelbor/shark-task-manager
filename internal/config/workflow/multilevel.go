package workflow

// KnownLevels lists every entity workflow level shark supports, in the canonical
// display order used by `shark admin workflow list/validate` and any other
// consumer that needs to iterate all levels. Adding a new entity workflow
// requires appending its level name here so every consumer picks it up.
var KnownLevels = []string{
	"epic",
	"feature",
	"task",
	"sprint",
	"bug",
	"change",
	"tech_debt",
}

// MultiLevelWorkflow holds workflow configurations for all entity levels.
// Any level may be nil, meaning "use default workflow for that level."
type MultiLevelWorkflow struct {
	Epic     *WorkflowConfig
	Feature  *WorkflowConfig
	Task     *WorkflowConfig
	Sprint   *WorkflowConfig
	Bug      *WorkflowConfig
	Change   *WorkflowConfig
	TechDebt *WorkflowConfig

	// TemplateDirectory from the workflow file, if present.
	// When non-nil, takes precedence over Config.TemplateDirectory.
	TemplateDirectory *string

	// Sources tracks where each entity workflow was loaded from.
	// Keys: "epic", "feature", "task", "sprint", "bug", "change", "tech_debt"
	// Values: file path (e.g., ".sharkworkflow.json", ".sharkconfig.json") or "default"
	Sources map[string]string

	// HasLegacyTaskKeys is true when legacy top-level status_flow/status_metadata
	// keys coexist with a task_workflow block (either inline or in workflow file).
	HasLegacyTaskKeys bool
}

// GetByType returns the workflow slot for the given entity type, or nil if
// the slot is unset or the entity type is unknown.
//
// Unlike GetWorkflowForLevel, this does NOT fall back to a default — callers
// that need a guaranteed non-nil config should use GetWorkflowForLevel
// instead. This method is the single source of truth for the entity-type →
// slot mapping; other dispatchers (e.g. in internal/config/aliases.go) should
// delegate to it rather than re-enumerating the slot fields.
//
// Parameters:
//   - entityType: one of "epic", "feature", "task", "sprint", "bug",
//     "change", "tech_debt"
//
// Returns:
//   - *WorkflowConfig: the slot value (may be nil)
func (m *MultiLevelWorkflow) GetByType(entityType string) *WorkflowConfig {
	switch entityType {
	case "epic":
		return m.Epic
	case "feature":
		return m.Feature
	case "task":
		return m.Task
	case "sprint":
		return m.Sprint
	case "bug":
		return m.Bug
	case "change":
		return m.Change
	case "tech_debt":
		return m.TechDebt
	}
	return nil
}

// RawForLevel returns the raw (possibly nil) workflow config for the given
// level. Unlike GetWorkflowForLevel, it does NOT fall back to defaults — a nil
// return means "no custom workflow configured for this level."
//
// Callers can use this to distinguish custom-vs-default sources when rendering
// or validating the multi-level workflow. It is a thin alias for GetByType.
func (m *MultiLevelWorkflow) RawForLevel(level string) *WorkflowConfig {
	return m.GetByType(level)
}

// EntityTypes returns the list of entity-type keys recognized by GetByType.
// Order is stable: epic, feature, task, sprint, bug, change, tech_debt.
//
// Callers that want to iterate every slot (e.g. to build a per-entity map
// and stay in sync as new entity types are added) should range over this
// slice rather than maintaining a parallel list.
func EntityTypes() []string {
	return []string{"epic", "feature", "task", "sprint", "bug", "change", "tech_debt"}
}

// GetWorkflowForLevel returns the workflow config for the given level.
// Falls back to the appropriate default if nil.
//
// Parameters:
//   - level: one of "epic", "feature", "task", "sprint", "bug", "change",
//     "tech_debt"
//
// Returns:
//   - *WorkflowConfig: never nil (falls back to defaults)
func (m *MultiLevelWorkflow) GetWorkflowForLevel(level string) *WorkflowConfig {
	if wf := m.GetByType(level); wf != nil {
		return wf
	}
	return defaultForType(level)
}

// defaultForType returns the hardcoded default workflow for an entity type.
// Unknown types fall back to DefaultWorkflow (the task workflow) to preserve
// historical GetWorkflowForLevel behavior.
func defaultForType(entityType string) *WorkflowConfig {
	switch entityType {
	case "epic":
		return DefaultEpicWorkflow()
	case "feature":
		return DefaultFeatureWorkflow()
	case "task":
		return DefaultWorkflow()
	case "sprint":
		return DefaultSprintWorkflow()
	case "bug":
		return DefaultBugWorkflow()
	case "change":
		return DefaultChangeCardWorkflow()
	case "tech_debt":
		return DefaultTechDebtWorkflow()
	default:
		return DefaultWorkflow()
	}
}
