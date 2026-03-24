package workflow

// MultiLevelWorkflow holds workflow configurations for all entity levels.
// Any level may be nil, meaning "use default workflow for that level."
type MultiLevelWorkflow struct {
	Epic    *WorkflowConfig
	Feature *WorkflowConfig
	Task    *WorkflowConfig
	Bug     *WorkflowConfig
	Change  *WorkflowConfig

	// TemplateDirectory from the workflow file, if present.
	// When non-nil, takes precedence over Config.TemplateDirectory.
	TemplateDirectory *string

	// Sources tracks where each entity workflow was loaded from.
	// Keys: "epic", "feature", "task", "bug", "change"
	// Values: file path (e.g., ".sharkworkflow.json", ".sharkconfig.json") or "default"
	Sources map[string]string

	// HasLegacyTaskKeys is true when legacy top-level status_flow/status_metadata
	// keys coexist with a task_workflow block (either inline or in workflow file).
	HasLegacyTaskKeys bool
}

// GetWorkflowForLevel returns the workflow config for the given level.
// Falls back to the appropriate default if nil.
//
// Parameters:
//   - level: one of "epic", "feature", "task", "bug", "change"
//
// Returns:
//   - *WorkflowConfig: never nil (falls back to defaults)
func (m *MultiLevelWorkflow) GetWorkflowForLevel(level string) *WorkflowConfig {
	switch level {
	case "epic":
		if m.Epic != nil {
			return m.Epic
		}
		return DefaultEpicWorkflow()
	case "feature":
		if m.Feature != nil {
			return m.Feature
		}
		return DefaultFeatureWorkflow()
	case "task":
		if m.Task != nil {
			return m.Task
		}
		return DefaultWorkflow()
	case "bug":
		if m.Bug != nil {
			return m.Bug
		}
		return DefaultBugWorkflow()
	case "change":
		if m.Change != nil {
			return m.Change
		}
		return DefaultChangeCardWorkflow()
	default:
		return DefaultWorkflow()
	}
}
