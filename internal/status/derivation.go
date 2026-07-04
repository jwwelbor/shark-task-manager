// Package status provides status derivation and calculation logic for cascading
// status updates in the Epic -> Feature -> Task hierarchy.
package status

import (
	"log/slog"

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
		slog.Warn("No workflow config provided to DeriveFeatureStatus, using safe defaults")
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
			slog.Warn("Status not found in workflow config, treating as planning phase", "status", status)
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
			slog.Warn("Unrecognized phase for status, treating as planning", "phase", meta.Phase, "status", status)
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
//
// counts is keyed by each feature's raw stored status, which under a
// route-based workflow (feature.yaml) may be any of its ~15 real statuses
// (assessment, research, code_review, qa, ...), not just the 4 legacy
// draft/active/completed/archived values — GetFeatureStatusBreakdown reads
// the DB column directly. Rather than enumerate every possible intermediate
// status, "active" is computed as everything left over once completed-ish
// and draft features are subtracted out, so any in-progress route-based
// status is correctly counted as active work without this function needing
// workflow awareness.
//
// Rules:
// - Empty (no features): returns EpicStatusDraft
// - All completed/archived/cancelled: returns EpicStatusCompleted
// - Any remaining (non-completed-ish, non-draft) status: returns EpicStatusActive
// - Some completed + some draft (no other work): returns EpicStatusActive
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

	// Count completed-ish (completed, archived, and cancelled all mean "no
	// longer pending work" for rollup purposes; archived is the pre-migration
	// terminal name, cancelled is route-based feature.yaml's).
	completed := counts[models.FeatureStatusCompleted] + counts[models.FeatureStatusArchived] + counts[models.FeatureStatusCancelled]
	if completed == total {
		return models.EpicStatusCompleted
	}

	draft := counts[models.FeatureStatusDraft]

	// Everything else — literal "active" plus any route-based intermediate
	// status (assessment, research, code_review, qa, ...) — counts as active work.
	active := total - completed - draft
	if active > 0 {
		return models.EpicStatusActive
	}

	// Check for partial completion (some completed + some draft)
	// This is a "work in progress" state even without active features
	if completed > 0 && draft > 0 {
		return models.EpicStatusActive
	}

	// All draft = draft
	return models.EpicStatusDraft
}

// IsFeatureCompletedStatus returns true if the feature status counts as "completed"
func IsFeatureCompletedStatus(status models.FeatureStatus) bool {
	switch status {
	case models.FeatureStatusCompleted,
		models.FeatureStatusArchived,
		models.FeatureStatusCancelled:
		return true
	default:
		return false
	}
}
