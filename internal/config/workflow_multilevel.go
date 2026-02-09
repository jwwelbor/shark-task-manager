package config

// MultiLevelWorkflow holds workflow configurations for all entity levels.
// Any level may be nil, meaning "use default workflow for that level."
type MultiLevelWorkflow struct {
	Epic    *WorkflowConfig
	Feature *WorkflowConfig
	Task    *WorkflowConfig
}

// GetWorkflowForLevel returns the workflow config for the given level.
// Falls back to the appropriate default if nil.
//
// Parameters:
//   - level: one of "epic", "feature", "task"
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
	default:
		return DefaultWorkflow()
	}
}
