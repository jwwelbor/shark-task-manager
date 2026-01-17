package repository

import "github.com/jwwelbor/shark-task-manager/internal/models"

// CalculateStatusFromProgress determines the appropriate status based on progress percentage
// Used by both Epic and Feature repositories for consistent status calculation
//
// Logic:
//   - progress >= 100.0 → completed (all work done)
//   - progress > 0.0    → active (work in progress)
//   - progress == 0.0   → draft (not started)
func CalculateStatusFromProgress(progress float64) string {
	if progress >= 100.0 {
		return string(models.FeatureStatusCompleted) // Same as EpicStatusCompleted
	} else if progress > 0.0 {
		return string(models.FeatureStatusActive) // Same as EpicStatusActive
	}
	return string(models.FeatureStatusDraft) // Same as EpicStatusDraft
}
