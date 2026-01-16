package status

import "github.com/jwwelbor/shark-task-manager/internal/config"

// CalculateWorkRemaining breaks down work by responsibility
// Uses responsibility and blocks_feature from config to categorize work:
// - agent: work assigned to AI agents
// - human/qa_team: work assigned to humans/QA
// - none + blocks_feature=true: blocked work
// - none + progress_weight=0.0: not started work
// - progress_weight>=1.0: completed work
//
// Parameters:
//   - statusCounts: map of status -> count of tasks in that status
//   - cfg: Config with status metadata containing responsibility and blocks_feature
//
// Returns WorkSummary with breakdown:
//   - TotalTasks: total count across all statuses
//   - AgentWork: tasks with responsibility="agent"
//   - HumanWork: tasks with responsibility="human" or "qa_team"
//   - BlockedWork: tasks with blocks_feature=true
//   - NotStarted: tasks with progress_weight=0.0 (excluding completed)
//   - CompletedTasks: tasks with progress_weight>=1.0
func CalculateWorkRemaining(statusCounts map[string]int, cfg *config.Config) *WorkSummary {
	summary := &WorkSummary{}

	for status, count := range statusCounts {
		meta := cfg.GetStatusMetadata(status)
		if meta == nil {
			// Skip statuses without metadata
			continue
		}

		summary.TotalTasks += count

		// Categorize by responsibility
		switch meta.Responsibility {
		case "agent":
			summary.AgentWork += count
		case "human", "qa_team":
			summary.HumanWork += count
		case "none":
			// For "none" responsibility, categorize based on blocks_feature and progress_weight
			if meta.BlocksFeature {
				summary.BlockedWork += count
			} else if meta.ProgressWeight == 0.0 {
				summary.NotStarted += count
			}
		}

		// Count completed tasks (progress_weight >= 1.0)
		// These are counted separately from responsibility categorization
		if meta.ProgressWeight >= 1.0 {
			summary.CompletedTasks += count
		}
	}

	return summary
}
