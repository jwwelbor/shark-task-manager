package scope

// ScopeType represents the type of entity being referenced in a CLI command
type ScopeType string

const (
	// ScopeEpic indicates the command operates on an epic
	ScopeEpic ScopeType = "epic"

	// ScopeFeature indicates the command operates on a feature
	ScopeFeature ScopeType = "feature"

	// ScopeTask indicates the command operates on a task
	ScopeTask ScopeType = "task"
)

// Scope represents a parsed scope from CLI arguments
// It contains the type of entity (epic, feature, or task) and the normalized key
type Scope struct {
	// Type is the scope type (epic, feature, or task)
	Type ScopeType

	// Key is the normalized key for the entity
	// Examples:
	//   - Epic: "E01"
	//   - Feature: "E01-F01"
	//   - Task: "T-E01-F01-001"
	Key string
}
