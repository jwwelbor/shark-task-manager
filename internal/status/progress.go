package status

import (
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/progress"
)

// CalculateProgress delegates to progress.CalculateProgress for backward compatibility
// This function is maintained in the status package to avoid breaking existing callers,
// but the actual implementation lives in the progress package to avoid circular dependencies.
func CalculateProgress(statusCounts map[string]int, cfg *config.WorkflowConfig) *ProgressInfo {
	return progress.CalculateProgress(statusCounts, cfg)
}
