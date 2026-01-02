package status

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDeriveFeatureStatus(t *testing.T) {
	tests := []struct {
		name     string
		counts   map[models.TaskStatus]int
		expected models.FeatureStatus
	}{
		{
			name:     "empty_returns_draft",
			counts:   map[models.TaskStatus]int{},
			expected: models.FeatureStatusDraft,
		},
		{
			name: "all_todo_returns_draft",
			counts: map[models.TaskStatus]int{
				models.TaskStatusTodo: 5,
			},
			expected: models.FeatureStatusDraft,
		},
		{
			name: "all_completed_returns_completed",
			counts: map[models.TaskStatus]int{
				models.TaskStatusCompleted: 5,
			},
			expected: models.FeatureStatusCompleted,
		},
		{
			name: "all_archived_returns_completed",
			counts: map[models.TaskStatus]int{
				models.TaskStatusArchived: 3,
			},
			expected: models.FeatureStatusCompleted,
		},
		{
			name: "mixed_completed_archived_returns_completed",
			counts: map[models.TaskStatus]int{
				models.TaskStatusCompleted: 3,
				models.TaskStatusArchived:  2,
			},
			expected: models.FeatureStatusCompleted,
		},
		{
			name: "any_in_progress_returns_active",
			counts: map[models.TaskStatus]int{
				models.TaskStatusTodo:       2,
				models.TaskStatusInProgress: 1,
				models.TaskStatusCompleted:  3,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "any_ready_for_review_returns_active",
			counts: map[models.TaskStatus]int{
				models.TaskStatusTodo:           2,
				models.TaskStatusReadyForReview: 1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "only_blocked_returns_active",
			counts: map[models.TaskStatus]int{
				models.TaskStatusBlocked: 2,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "blocked_with_todo_returns_active",
			counts: map[models.TaskStatus]int{
				models.TaskStatusTodo:    3,
				models.TaskStatusBlocked: 1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "some_completed_some_todo_returns_active",
			counts: map[models.TaskStatus]int{
				models.TaskStatusTodo:      2,
				models.TaskStatusCompleted: 3,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "complex_mix_with_active_returns_active",
			counts: map[models.TaskStatus]int{
				models.TaskStatusTodo:           1,
				models.TaskStatusInProgress:     1,
				models.TaskStatusBlocked:        1,
				models.TaskStatusReadyForReview: 1,
				models.TaskStatusCompleted:      1,
				models.TaskStatusArchived:       1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "in_progress_priority_over_blocked",
			counts: map[models.TaskStatus]int{
				models.TaskStatusInProgress: 1,
				models.TaskStatusBlocked:    5,
			},
			expected: models.FeatureStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveFeatureStatus(tt.counts)
			assert.Equal(t, tt.expected, result, "DeriveFeatureStatus(%v)", tt.counts)
		})
	}
}

func TestDeriveEpicStatus(t *testing.T) {
	tests := []struct {
		name     string
		counts   map[models.FeatureStatus]int
		expected models.EpicStatus
	}{
		{
			name:     "empty_returns_draft",
			counts:   map[models.FeatureStatus]int{},
			expected: models.EpicStatusDraft,
		},
		{
			name: "all_draft_returns_draft",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusDraft: 4,
			},
			expected: models.EpicStatusDraft,
		},
		{
			name: "all_completed_returns_completed",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusCompleted: 3,
			},
			expected: models.EpicStatusCompleted,
		},
		{
			name: "all_archived_returns_completed",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusArchived: 2,
			},
			expected: models.EpicStatusCompleted,
		},
		{
			name: "mixed_completed_archived_returns_completed",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusCompleted: 2,
				models.FeatureStatusArchived:  1,
			},
			expected: models.EpicStatusCompleted,
		},
		{
			name: "any_active_returns_active",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusDraft:     1,
				models.FeatureStatusActive:    1,
				models.FeatureStatusCompleted: 1,
			},
			expected: models.EpicStatusActive,
		},
		{
			name: "mixed_draft_completed_returns_active",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusDraft:     2,
				models.FeatureStatusCompleted: 2,
			},
			expected: models.EpicStatusActive,
		},
		{
			name: "only_active_returns_active",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusActive: 3,
			},
			expected: models.EpicStatusActive,
		},
		{
			name: "complex_mix_returns_active",
			counts: map[models.FeatureStatus]int{
				models.FeatureStatusDraft:     1,
				models.FeatureStatusActive:    2,
				models.FeatureStatusCompleted: 3,
				models.FeatureStatusArchived:  1,
			},
			expected: models.EpicStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveEpicStatus(tt.counts)
			assert.Equal(t, tt.expected, result, "DeriveEpicStatus(%v)", tt.counts)
		})
	}
}

func TestIsTaskActiveStatus(t *testing.T) {
	tests := []struct {
		status   models.TaskStatus
		expected bool
	}{
		{models.TaskStatusInProgress, true},
		{models.TaskStatusReadyForReview, true},
		{models.TaskStatusBlocked, true},
		{models.TaskStatusTodo, false},
		{models.TaskStatusCompleted, false},
		{models.TaskStatusArchived, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := IsTaskActiveStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTaskCompletedStatus(t *testing.T) {
	tests := []struct {
		status   models.TaskStatus
		expected bool
	}{
		{models.TaskStatusCompleted, true},
		{models.TaskStatusArchived, true},
		{models.TaskStatusInProgress, false},
		{models.TaskStatusReadyForReview, false},
		{models.TaskStatusBlocked, false},
		{models.TaskStatusTodo, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := IsTaskCompletedStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFeatureActiveStatus(t *testing.T) {
	tests := []struct {
		status   models.FeatureStatus
		expected bool
	}{
		{models.FeatureStatusActive, true},
		{models.FeatureStatusDraft, false},
		{models.FeatureStatusCompleted, false},
		{models.FeatureStatusArchived, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := IsFeatureActiveStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFeatureCompletedStatus(t *testing.T) {
	tests := []struct {
		status   models.FeatureStatus
		expected bool
	}{
		{models.FeatureStatusCompleted, true},
		{models.FeatureStatusArchived, true},
		{models.FeatureStatusActive, false},
		{models.FeatureStatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := IsFeatureCompletedStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}
