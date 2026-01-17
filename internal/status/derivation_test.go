package status

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

// createMockWorkflowConfig creates a mock workflow config for testing
func createMockWorkflowConfig() *config.WorkflowConfig {
	return &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"draft": {
				Phase:          "planning",
				ProgressWeight: 0.0,
			},
			"todo": {
				Phase:          "planning",
				ProgressWeight: 0.0,
			},
			"in_progress": {
				Phase:          "development",
				ProgressWeight: 0.5,
			},
			"ready_for_review": {
				Phase:          "review",
				ProgressWeight: 0.75,
			},
			"ready_for_code_review": {
				Phase:          "review",
				ProgressWeight: 0.75,
			},
			"in_qa": {
				Phase:          "qa",
				ProgressWeight: 0.85,
			},
			"ready_for_qa": {
				Phase:          "qa",
				ProgressWeight: 0.8,
			},
			"ready_for_approval": {
				Phase:          "approval",
				ProgressWeight: 0.9,
			},
			"in_approval": {
				Phase:          "approval",
				ProgressWeight: 0.95,
			},
			"completed": {
				Phase:          "done",
				ProgressWeight: 1.0,
			},
			"archived": {
				Phase:          "done",
				ProgressWeight: 1.0,
			},
			"blocked": {
				Phase:          "any",
				ProgressWeight: 0.0,
				BlocksFeature:  true,
			},
			"on_hold": {
				Phase:          "any",
				ProgressWeight: 0.0,
			},
		},
	}
}

func TestDeriveFeatureStatus(t *testing.T) {
	cfg := createMockWorkflowConfig()

	tests := []struct {
		name     string
		counts   map[string]int
		expected models.FeatureStatus
	}{
		{
			name:     "empty_returns_draft",
			counts:   map[string]int{},
			expected: models.FeatureStatusDraft,
		},
		{
			name: "all_todo_returns_draft",
			counts: map[string]int{
				"todo": 5,
			},
			expected: models.FeatureStatusDraft,
		},
		{
			name: "all_completed_returns_completed",
			counts: map[string]int{
				"completed": 5,
			},
			expected: models.FeatureStatusCompleted,
		},
		{
			name: "all_archived_returns_completed",
			counts: map[string]int{
				"archived": 3,
			},
			expected: models.FeatureStatusCompleted,
		},
		{
			name: "mixed_completed_archived_returns_completed",
			counts: map[string]int{
				"completed": 3,
				"archived":  2,
			},
			expected: models.FeatureStatusCompleted,
		},
		{
			name: "any_in_progress_returns_active",
			counts: map[string]int{
				"todo":        2,
				"in_progress": 1,
				"completed":   3,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "any_ready_for_review_returns_active",
			counts: map[string]int{
				"todo":             2,
				"ready_for_review": 1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "only_blocked_returns_active",
			counts: map[string]int{
				"blocked": 2,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "blocked_with_todo_returns_active",
			counts: map[string]int{
				"todo":    3,
				"blocked": 1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "some_completed_some_todo_returns_active",
			counts: map[string]int{
				"todo":      2,
				"completed": 3,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "complex_mix_with_active_returns_active",
			counts: map[string]int{
				"todo":             1,
				"in_progress":      1,
				"blocked":          1,
				"ready_for_review": 1,
				"completed":        1,
				"archived":         1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "in_progress_priority_over_blocked",
			counts: map[string]int{
				"in_progress": 1,
				"blocked":     5,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "tasks_in_qa_returns_active",
			counts: map[string]int{
				"in_qa": 3,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "ready_for_approval_returns_active",
			counts: map[string]int{
				"ready_for_approval": 8,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "ready_for_code_review_returns_active",
			counts: map[string]int{
				"ready_for_code_review": 1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "mixed_review_qa_approval_returns_active",
			counts: map[string]int{
				"ready_for_code_review": 1,
				"in_qa":                 1,
				"ready_for_approval":    8,
				"completed":             1,
			},
			expected: models.FeatureStatusActive,
		},
		{
			name: "unknown_status_treated_as_planning",
			counts: map[string]int{
				"unknown_status": 3,
			},
			expected: models.FeatureStatusDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveFeatureStatus(tt.counts, cfg)
			assert.Equal(t, tt.expected, result, "DeriveFeatureStatus(%v)", tt.counts)
		})
	}
}

func TestDeriveFeatureStatus_NilConfig(t *testing.T) {
	counts := map[string]int{
		"in_progress": 1,
	}

	// Should return draft when config is nil (safe default)
	result := DeriveFeatureStatus(counts, nil)
	assert.Equal(t, models.FeatureStatusDraft, result)
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
