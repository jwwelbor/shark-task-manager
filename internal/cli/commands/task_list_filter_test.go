package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestTaskListFiltering_HideCompletedByDefault tests that completed tasks
// are hidden by default unless --show-all flag is used
func TestTaskListFiltering_HideCompletedByDefault(t *testing.T) {
	tests := []struct {
		name                      string
		inputTasks                []*models.Task
		showAllFlag               bool
		statusFlag                string // If set, explicit status filter overrides default behavior
		expectedTaskCount         int
		expectedIncludesCompleted bool
	}{
		{
			name: "Default behavior - hide completed tasks",
			inputTasks: []*models.Task{
				{Key: "T-E01-F01-001", Status: models.TaskStatusTodo, Title: "Task 1"},
				{Key: "T-E01-F01-002", Status: models.TaskStatusInProgress, Title: "Task 2"},
				{Key: "T-E01-F01-003", Status: models.TaskStatusCompleted, Title: "Task 3"},
				{Key: "T-E01-F01-004", Status: models.TaskStatusReadyForReview, Title: "Task 4"},
				{Key: "T-E01-F01-005", Status: models.TaskStatusCompleted, Title: "Task 5"},
			},
			showAllFlag:               false,
			statusFlag:                "",
			expectedTaskCount:         3, // Should exclude 2 completed tasks
			expectedIncludesCompleted: false,
		},
		{
			name: "With --show-all - include completed tasks",
			inputTasks: []*models.Task{
				{Key: "T-E01-F01-001", Status: models.TaskStatusTodo, Title: "Task 1"},
				{Key: "T-E01-F01-002", Status: models.TaskStatusInProgress, Title: "Task 2"},
				{Key: "T-E01-F01-003", Status: models.TaskStatusCompleted, Title: "Task 3"},
				{Key: "T-E01-F01-004", Status: models.TaskStatusReadyForReview, Title: "Task 4"},
				{Key: "T-E01-F01-005", Status: models.TaskStatusCompleted, Title: "Task 5"},
			},
			showAllFlag:               true,
			statusFlag:                "",
			expectedTaskCount:         5, // Should include all tasks
			expectedIncludesCompleted: true,
		},
		{
			name: "Explicit --status=completed overrides default hiding",
			// NOTE: In real usage, the repository query filters by status BEFORE
			// this function is called, so inputTasks would already be filtered.
			// This test simulates that the repository already filtered the tasks.
			inputTasks: []*models.Task{
				{Key: "T-E01-F01-002", Status: models.TaskStatusCompleted, Title: "Task 2"},
				{Key: "T-E01-F01-003", Status: models.TaskStatusCompleted, Title: "Task 3"},
			},
			showAllFlag:               false,
			statusFlag:                "completed",
			expectedTaskCount:         2, // Repository already filtered, we pass through
			expectedIncludesCompleted: true,
		},
		{
			name: "Only completed tasks - default hides all",
			inputTasks: []*models.Task{
				{Key: "T-E01-F01-001", Status: models.TaskStatusCompleted, Title: "Task 1"},
				{Key: "T-E01-F01-002", Status: models.TaskStatusCompleted, Title: "Task 2"},
			},
			showAllFlag:               false,
			statusFlag:                "",
			expectedTaskCount:         0, // All tasks hidden
			expectedIncludesCompleted: false,
		},
		{
			name: "Only completed tasks - show-all shows them",
			inputTasks: []*models.Task{
				{Key: "T-E01-F01-001", Status: models.TaskStatusCompleted, Title: "Task 1"},
				{Key: "T-E01-F01-002", Status: models.TaskStatusCompleted, Title: "Task 2"},
			},
			showAllFlag:               true,
			statusFlag:                "",
			expectedTaskCount:         2, // All tasks shown
			expectedIncludesCompleted: true,
		},
		{
			name:                      "No tasks",
			inputTasks:                []*models.Task{},
			showAllFlag:               false,
			statusFlag:                "",
			expectedTaskCount:         0,
			expectedIncludesCompleted: false,
		},
		{
			name: "All non-completed tasks - no filtering needed",
			inputTasks: []*models.Task{
				{Key: "T-E01-F01-001", Status: models.TaskStatusTodo, Title: "Task 1"},
				{Key: "T-E01-F01-002", Status: models.TaskStatusInProgress, Title: "Task 2"},
				{Key: "T-E01-F01-003", Status: models.TaskStatusBlocked, Title: "Task 3"},
			},
			showAllFlag:               false,
			statusFlag:                "",
			expectedTaskCount:         3, // All tasks shown (none are completed)
			expectedIncludesCompleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the filtering logic that should be in runTaskList
			filteredTasks := filterTasksByCompletedStatus(tt.inputTasks, tt.showAllFlag, tt.statusFlag)

			// Verify task count
			if len(filteredTasks) != tt.expectedTaskCount {
				t.Errorf("Expected %d tasks, got %d", tt.expectedTaskCount, len(filteredTasks))
			}

			// Verify whether completed tasks are included
			hasCompleted := false
			for _, task := range filteredTasks {
				if task.Status == models.TaskStatusCompleted {
					hasCompleted = true
					break
				}
			}

			if hasCompleted != tt.expectedIncludesCompleted {
				t.Errorf("Expected includes completed = %v, got %v", tt.expectedIncludesCompleted, hasCompleted)
			}

			// Additional validation: If not showing all and no explicit status filter,
			// no task should be completed
			if !tt.showAllFlag && tt.statusFlag == "" {
				for _, task := range filteredTasks {
					if task.Status == models.TaskStatusCompleted {
						t.Errorf("Found completed task %s when showAll=false and no status filter", task.Key)
					}
				}
			}
		})
	}
}

// TestTaskListFiltering_StatusFilterPrecedence tests that explicit status filters
// take precedence over the default hiding behavior
func TestTaskListFiltering_StatusFilterPrecedence(t *testing.T) {
	tests := []struct {
		name                  string
		showAllFlag           bool
		statusFilter          string
		shouldFilterCompleted bool // Should completed tasks be filtered out?
	}{
		{
			name:                  "Default: hide completed",
			showAllFlag:           false,
			statusFilter:          "",
			shouldFilterCompleted: true,
		},
		{
			name:                  "show-all: don't hide completed",
			showAllFlag:           true,
			statusFilter:          "",
			shouldFilterCompleted: false,
		},
		{
			name:                  "status=completed: show completed (overrides default)",
			showAllFlag:           false,
			statusFilter:          "completed",
			shouldFilterCompleted: false, // Explicit filter overrides default
		},
		{
			name:                  "status=todo: only show todo (other filters active)",
			showAllFlag:           false,
			statusFilter:          "todo",
			shouldFilterCompleted: false, // Status filter takes precedence
		},
		{
			name:                  "show-all + status=todo: show todo only",
			showAllFlag:           true,
			statusFilter:          "todo",
			shouldFilterCompleted: false, // Status filter takes precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents the expected precedence behavior
			// Implementation should follow this logic
			t.Logf("showAll=%v, statusFilter=%q, shouldFilterCompleted=%v",
				tt.showAllFlag, tt.statusFilter, tt.shouldFilterCompleted)
		})
	}
}
