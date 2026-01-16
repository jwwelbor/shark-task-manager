package status

import (
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// GetActionItems categorizes tasks requiring immediate attention
//
// This function takes a list of tasks and categorizes them into:
// - AwaitingApproval: tasks in ready_for_approval status
// - Blocked: tasks in blocked status with their blocking reasons
// - InProgress: tasks in in_progress or in_development status
//
// For waiting tasks, it calculates how many days they've been waiting.
// For blocked tasks, it extracts the reason for blocking.
//
// Parameters:
// - tasks: slice of Task models
// - cfg: Config with status metadata for determining age calculations
//
// Returns ActionItems with categorized task items
func GetActionItems(tasks []*models.Task, cfg *config.WorkflowConfig) *ActionItems {
	items := &ActionItems{
		AwaitingApproval: []*TaskActionItem{},
		Blocked:          []*TaskActionItem{},
		InProgress:       []*TaskActionItem{},
	}

	now := time.Now()

	for _, task := range tasks {
		if task == nil {
			continue
		}

		statusStr := string(task.Status)
		switch statusStr {
		case "ready_for_approval", "ready_for_code_review":
			// Calculate age in days for waiting tasks
			ageDays := int(now.Sub(task.UpdatedAt).Hours() / 24)
			items.AwaitingApproval = append(items.AwaitingApproval, &TaskActionItem{
				TaskKey:       task.Key,
				Title:         task.Title,
				Status:        statusStr,
				AgeDays:       &ageDays,
				BlockedReason: nil,
			})

		case "blocked":
			// For blocked tasks, include the block reason if available
			// Note: block reason would typically come from task history or additional field
			// For now, we use a pointer to nil since the reason isn't in the current model
			items.Blocked = append(items.Blocked, &TaskActionItem{
				TaskKey:       task.Key,
				Title:         task.Title,
				Status:        statusStr,
				AgeDays:       nil,
				BlockedReason: nil,
			})

		case "in_progress", "in_development":
			// Track tasks currently in progress
			items.InProgress = append(items.InProgress, &TaskActionItem{
				TaskKey:       task.Key,
				Title:         task.Title,
				Status:        statusStr,
				AgeDays:       nil,
				BlockedReason: nil,
			})
		}
	}

	return items
}
