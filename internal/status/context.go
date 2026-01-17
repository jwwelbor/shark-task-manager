package status

import "github.com/jwwelbor/shark-task-manager/internal/config"

// GetStatusContext returns contextual status suffix for display
// Examples: "active (waiting)", "active (blocked)", "active (development)"
//
// This function takes status counts and returns a contextual string that represents
// what work is currently happening in a feature or epic. It uses status metadata
// from the configuration to determine the context.
//
// Parameters:
// - statusCounts: map of status -> count of tasks in that status
// - cfg: Config with status metadata containing responsibility and blocks_feature
//
// Returns a contextual status string that provides insight into current work state:
// - "active (waiting)" if tasks are awaiting approval or review
// - "active (blocked)" if tasks are blocked
// - "active (development)" if tasks are actively being developed
// - "active" for other active statuses
// - The original status if the feature is not in "active" status
func GetStatusContext(statusCounts map[string]int, cfg *config.WorkflowConfig) string {
	// Check for waiting (tasks awaiting approval or code review)
	if statusCounts["ready_for_approval"] > 0 || statusCounts["ready_for_code_review"] > 0 {
		return "active (waiting)"
	}

	// Check for blocked
	if statusCounts["blocked"] > 0 {
		return "active (blocked)"
	}

	// Check for development
	if statusCounts["in_development"] > 0 || statusCounts["in_progress"] > 0 {
		return "active (development)"
	}

	return "active"
}
