// Package status provides status derivation and calculation logic for cascading
// status updates in the Epic -> Feature -> Task hierarchy.
package status

import (
	"log"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// DeriveFeatureStatus calculates feature status from task status counts using workflow config.
// This is the config-driven version that uses the phase field from status_metadata.
//
// Parameters:
//   - statusCounts: map of status -> count (e.g., {"completed": 2, "in_qa": 3})
//   - cfg: WorkflowConfig containing status_metadata with phase information
//
// Returns FeatureStatus based on phase categorization:
//   - Empty (no tasks): FeatureStatusDraft
//   - All tasks in phase="done": FeatureStatusCompleted
//   - Any tasks in phase="development|review|qa|approval|any": FeatureStatusActive
//   - Mixed completed + planning: FeatureStatusActive (work in progress)
//   - All tasks in phase="planning": FeatureStatusDraft
//
// Unknown statuses (not in config) are treated as planning phase with a warning log.
func DeriveFeatureStatus(statusCounts map[string]int, cfg *config.WorkflowConfig) models.FeatureStatus {
	// Handle nil config gracefully
	if cfg == nil {
		log.Println("WARN: No workflow config provided to DeriveFeatureStatus, using safe defaults")
		return models.FeatureStatusDraft
	}

	total := 0
	completedCount := 0
	activeCount := 0
	planningCount := 0

	for status, count := range statusCounts {
		total += count

		// Get metadata from config
		meta, found := cfg.GetStatusMetadata(status)
		if !found {
			// Unknown status - treat as planning and log warning
			log.Printf("WARN: Status %q not found in workflow config, treating as planning phase", status)
			planningCount += count
			continue
		}

		// Categorize by phase
		switch meta.Phase {
		case "done":
			completedCount += count
		case "development", "review", "qa", "approval":
			activeCount += count
		case "planning":
			planningCount += count
		case "any":
			// Blocked/on_hold count as active work (blocks feature progress)
			activeCount += count
		default:
			// Unrecognized phase - treat as planning
			log.Printf("WARN: Unrecognized phase %q for status %q, treating as planning", meta.Phase, status)
			planningCount += count
		}
	}

	// Derive feature status from counts
	if total == 0 {
		return models.FeatureStatusDraft
	}

	// All completed → completed
	if completedCount == total {
		return models.FeatureStatusCompleted
	}

	// Any active work → active
	if activeCount > 0 {
		return models.FeatureStatusActive
	}

	// Mixed completed + planning = work in progress → active
	if completedCount > 0 && planningCount > 0 {
		return models.FeatureStatusActive
	}

	// All planning = draft
	return models.FeatureStatusDraft
}

// DeriveEpicStatus calculates epic status from feature status counts.
// Feature statuses are already derived, so this function uses the typed feature status constants.
//
// Rules:
// - Empty (no features): returns EpicStatusDraft
// - All completed/archived: returns EpicStatusCompleted
// - Any active: returns EpicStatusActive
// - Some completed + some draft (no active): returns EpicStatusActive
// - All draft: returns EpicStatusDraft
func DeriveEpicStatus(counts map[models.FeatureStatus]int) models.EpicStatus {
	total := 0
	for _, c := range counts {
		total += c
	}

	// No features = draft
	if total == 0 {
		return models.EpicStatusDraft
	}

	// Count completed (completed + archived)
	completed := counts[models.FeatureStatusCompleted] + counts[models.FeatureStatusArchived]
	if completed == total {
		return models.EpicStatusCompleted
	}

	// Count active features
	active := counts[models.FeatureStatusActive]
	if active > 0 {
		return models.EpicStatusActive
	}

	// Check for partial completion (some completed + some draft)
	// This is a "work in progress" state even without active features
	draft := counts[models.FeatureStatusDraft]
	if completed > 0 && draft > 0 {
		return models.EpicStatusActive
	}

	// All draft = draft
	return models.EpicStatusDraft
}

// IsTaskActiveStatus returns true if the task status counts as "active work"
// DEPRECATED: Use workflow config phase field instead. This function uses hardcoded logic.
func IsTaskActiveStatus(status models.TaskStatus) bool {
	switch status {
	case models.TaskStatusInProgress,
		models.TaskStatusReadyForReview,
		models.TaskStatusBlocked:
		return true
	default:
		return false
	}
}

// IsTaskCompletedStatus returns true if the task status counts as "completed"
// DEPRECATED: Use workflow config phase field instead. This function uses hardcoded logic.
func IsTaskCompletedStatus(status models.TaskStatus) bool {
	switch status {
	case models.TaskStatusCompleted,
		models.TaskStatusArchived:
		return true
	default:
		return false
	}
}

// IsFeatureActiveStatus returns true if the feature status counts as "active work"
func IsFeatureActiveStatus(status models.FeatureStatus) bool {
	return status == models.FeatureStatusActive
}

// IsFeatureCompletedStatus returns true if the feature status counts as "completed"
func IsFeatureCompletedStatus(status models.FeatureStatus) bool {
	switch status {
	case models.FeatureStatusCompleted,
		models.FeatureStatusArchived:
		return true
	default:
		return false
	}
}
