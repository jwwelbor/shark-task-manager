package workflow

// Entity level constants identify which workflow level to use.
// Used throughout the codebase to select the correct WorkflowConfig
// from the MultiLevelWorkflow container.
const (
	LevelEpic     = "epic"
	LevelFeature  = "feature"
	LevelTask     = "task"
	LevelSprint   = "sprint"
	LevelBug      = "bug"
	LevelChange   = "change"
	LevelTechDebt = "tech_debt"
)
