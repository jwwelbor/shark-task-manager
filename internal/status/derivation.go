// Package status provides status derivation and calculation logic for cascading
// status updates in the Epic -> Feature -> Task hierarchy.
package status

import (
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// DeriveFeatureStatus calculates feature status from task status counts.
// Returns empty string if counts are empty (no tasks).
//
// Rules:
// - Empty (no tasks): returns FeatureStatusDraft
// - All completed/archived: returns FeatureStatusCompleted
// - Any in_progress/ready_for_review/blocked: returns FeatureStatusActive
// - Some completed + some todo (no active): returns FeatureStatusActive
// - All todo: returns FeatureStatusDraft
func DeriveFeatureStatus(counts map[models.TaskStatus]int) models.FeatureStatus {
	total := 0
	for _, c := range counts {
		total += c
	}

	// No tasks = draft
	if total == 0 {
		return models.FeatureStatusDraft
	}

	// Count completed (completed + archived)
	completed := counts[models.TaskStatusCompleted] + counts[models.TaskStatusArchived]
	if completed == total {
		return models.FeatureStatusCompleted
	}

	// Count active (in_progress, ready_for_review, blocked)
	active := counts[models.TaskStatusInProgress] +
		counts[models.TaskStatusReadyForReview] +
		counts[models.TaskStatusBlocked]
	if active > 0 {
		return models.FeatureStatusActive
	}

	// Check for partial completion (some completed + some todo)
	// This is a "work in progress" state even without active tasks
	todo := counts[models.TaskStatusTodo]
	if completed > 0 && todo > 0 {
		return models.FeatureStatusActive
	}

	// All todo = draft
	return models.FeatureStatusDraft
}

// DeriveEpicStatus calculates epic status from feature status counts.
// Returns empty string if counts are empty (no features).
//
// Rules:
// - Empty (no features): returns EpicStatusDraft
// - All completed/archived: returns EpicStatusCompleted
// - Any active/blocked: returns EpicStatusActive
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
