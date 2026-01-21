// Package progress provides pure progress calculation functions with no dependencies.
//
// This package contains zero dependencies on repository or database layers,
// making it safe to import from both repository and status packages without
// creating circular dependencies.
package progress

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// ProgressInfo provides weighted and completion progress metrics
// Used by feature get and feature list to show realistic progress
type ProgressInfo struct {
	WeightedPct     float64 // Weighted progress (e.g., 68.0) - recognizes partial work
	CompletionPct   float64 // Traditional completion % (e.g., 40.0) - only completed tasks
	WeightedRatio   string  // "3.4/5" (weighted tasks complete)
	CompletionRatio string  // "2/5" (completed tasks / total)
	TotalTasks      int     // Total task count
}

// CalculateProgress calculates weighted and completion progress from status counts
// Uses progress_weight from config to recognize partial completion
//
// Parameters:
// - statusCounts: map of status -> count (e.g., {"completed": 2, "in_progress": 3})
// - cfg: WorkflowConfig containing status_metadata with progress_weight values
//
// Returns ProgressInfo with:
// - WeightedPct: percentage accounting for partial progress (e.g., ready_for_approval = 90%)
// - CompletionPct: traditional percentage (only 100% = fully completed)
// - WeightedRatio: "3.4/5" format showing weighted progress
// - CompletionRatio: "2/5" format showing only completed tasks
// - TotalTasks: total number of tasks
//
// Example:
// 2 completed, 1 ready_for_approval, 1 in_development, 1 draft
// Weighted: (2.0 + 0.9 + 0.5 + 0.0) / 5 = 68%
// Completion: 2 / 5 = 40%
func CalculateProgress(statusCounts map[string]int, cfg *config.WorkflowConfig) *ProgressInfo {
	totalTasks := 0
	weightedProgress := 0.0
	completedTasks := 0

	// Sum up tasks and calculate weighted progress
	for status, count := range statusCounts {
		totalTasks += count

		// Get status metadata from config
		weight := 0.0
		if cfg != nil {
			meta, found := cfg.GetStatusMetadata(status)
			if found {
				weight = meta.ProgressWeight
			}
		} else {
			// Fallback: When no config available, treat completed/archived as 100% (weight 1.0)
			// This maintains backward compatibility with tests and deployments without config
			if status == "completed" || status == "archived" {
				weight = 1.0
			}
		}

		// Add weighted contribution
		weightedProgress += float64(count) * weight

		// Count only fully completed tasks (weight >= 1.0)
		if weight >= 1.0 {
			completedTasks += count
		}
	}

	// Handle empty task list
	if totalTasks == 0 {
		return &ProgressInfo{
			WeightedPct:     0.0,
			CompletionPct:   0.0,
			WeightedRatio:   "0/0",
			CompletionRatio: "0/0",
			TotalTasks:      0,
		}
	}

	// Calculate percentages
	weightedPct := (weightedProgress / float64(totalTasks)) * 100.0
	completionPct := (float64(completedTasks) / float64(totalTasks)) * 100.0

	return &ProgressInfo{
		WeightedPct:     weightedPct,
		CompletionPct:   completionPct,
		WeightedRatio:   fmt.Sprintf("%.1f/%d", weightedProgress, totalTasks),
		CompletionRatio: fmt.Sprintf("%d/%d", completedTasks, totalTasks),
		TotalTasks:      totalTasks,
	}
}
