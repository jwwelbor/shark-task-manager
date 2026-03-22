package status

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// testWorkflowConfig returns a workflow config with both basic and advanced statuses
// for use in action_items tests. It covers all statuses referenced in tests.
func testWorkflowConfig() *config.WorkflowConfig {
	return &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo":                  {"in_progress", "blocked"},
			"draft":                 {"ready_for_refinement_ba"},
			"in_progress":           {"ready_for_review", "blocked"},
			"in_development":        {"ready_for_code_review", "blocked"},
			"ready_for_review":      {"completed", "in_progress"},
			"ready_for_code_review": {"in_code_review"},
			"ready_for_approval":    {"in_approval"},
			"completed":             {},
			"blocked":               {"todo", "in_progress"},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"todo":                  {Color: "gray", Phase: "planning"},
			"draft":                 {Color: "gray", Phase: "planning"},
			"in_progress":           {Color: "blue", Phase: "development"},
			"in_development":        {Color: "blue", Phase: "development"},
			"ready_for_review":      {Color: "yellow", Phase: "review"},
			"ready_for_code_review": {Color: "yellow", Phase: "review"},
			"ready_for_approval":    {Color: "cyan", Phase: "approval"},
			"completed":             {Color: "green", Phase: "done"},
			"blocked":               {Color: "red", Phase: "any"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"todo", "draft"},
			config.CompleteStatusKey: {"completed"},
		},
	}
}

func TestGetActionItems(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name                    string
		tasks                   []*models.Task
		expectedAwaitingCount   int
		expectedBlockedCount    int
		expectedInProgressCount int
	}{
		{
			name:                    "no tasks",
			tasks:                   []*models.Task{},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    0,
			expectedInProgressCount: 0,
		},
		{
			name: "single ready_for_approval task",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01-001", Title: "Task 1", UpdatedAt: now.Add(-24 * time.Hour)},
					Status:     "ready_for_approval",
				},
			},
			expectedAwaitingCount:   1,
			expectedBlockedCount:    0,
			expectedInProgressCount: 0,
		},
		{
			name: "single ready_for_code_review task",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 2, Key: "E07-F01-002", Title: "Task 2", UpdatedAt: now.Add(-48 * time.Hour)},
					Status:     "ready_for_code_review",
				},
			},
			expectedAwaitingCount:   0, // ready_for_code_review is NOT awaiting approval
			expectedBlockedCount:    0,
			expectedInProgressCount: 0,
		},
		{
			name: "single blocked task",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 3, Key: "E07-F01-003", Title: "Task 3", UpdatedAt: now},
					Status:     "blocked",
				},
			},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    1,
			expectedInProgressCount: 0,
		},
		{
			name: "single in_progress task",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 4, Key: "E07-F01-004", Title: "Task 4", UpdatedAt: now},
					Status:     "in_progress",
				},
			},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    0,
			expectedInProgressCount: 1,
		},
		{
			name: "single in_development task",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 5, Key: "E07-F01-005", Title: "Task 5", UpdatedAt: now},
					Status:     "in_development",
				},
			},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    0,
			expectedInProgressCount: 1,
		},
		{
			name: "mixed task statuses",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01-001", Title: "Task 1", UpdatedAt: now.Add(-1 * time.Hour)},
					Status:     "ready_for_approval",
				},
				{
					BaseEntity: models.BaseEntity{ID: 2, Key: "E07-F01-002", Title: "Task 2", UpdatedAt: now},
					Status:     "blocked",
				},
				{
					BaseEntity: models.BaseEntity{ID: 3, Key: "E07-F01-003", Title: "Task 3", UpdatedAt: now},
					Status:     "in_progress",
				},
				{
					BaseEntity: models.BaseEntity{ID: 4, Key: "E07-F01-004", Title: "Task 4", UpdatedAt: now},
					Status:     "todo",
				},
				{
					BaseEntity: models.BaseEntity{ID: 5, Key: "E07-F01-005", Title: "Task 5", UpdatedAt: now.Add(-2 * time.Hour)},
					Status:     "ready_for_code_review",
				},
			},
			expectedAwaitingCount:   1, // Only ready_for_approval, not ready_for_code_review
			expectedBlockedCount:    1,
			expectedInProgressCount: 1,
		},
		{
			name: "multiple in_progress tasks",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01-001", Title: "Task 1", UpdatedAt: now},
					Status:     "in_progress",
				},
				{
					BaseEntity: models.BaseEntity{ID: 2, Key: "E07-F01-002", Title: "Task 2", UpdatedAt: now},
					Status:     "in_progress",
				},
				{
					BaseEntity: models.BaseEntity{ID: 3, Key: "E07-F01-003", Title: "Task 3", UpdatedAt: now},
					Status:     "in_development",
				},
			},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    0,
			expectedInProgressCount: 3,
		},
		{
			name: "todo and completed tasks ignored",
			tasks: []*models.Task{
				{
					BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01-001", Title: "Task 1", UpdatedAt: now},
					Status:     "todo",
				},
				{
					BaseEntity: models.BaseEntity{ID: 2, Key: "E07-F01-002", Title: "Task 2", UpdatedAt: now},
					Status:     "completed",
				},
				{
					BaseEntity: models.BaseEntity{ID: 3, Key: "E07-F01-003", Title: "Task 3", UpdatedAt: now},
					Status:     "draft",
				},
			},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    0,
			expectedInProgressCount: 0,
		},
		{
			name:                    "nil task in slice is skipped",
			tasks:                   []*models.Task{nil},
			expectedAwaitingCount:   0,
			expectedBlockedCount:    0,
			expectedInProgressCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := GetActionItems(tt.tasks, testWorkflowConfig())

			if len(items.AwaitingApproval) != tt.expectedAwaitingCount {
				t.Errorf("AwaitingApproval count: got %d, want %d", len(items.AwaitingApproval), tt.expectedAwaitingCount)
			}

			if len(items.Blocked) != tt.expectedBlockedCount {
				t.Errorf("Blocked count: got %d, want %d", len(items.Blocked), tt.expectedBlockedCount)
			}

			if len(items.InProgress) != tt.expectedInProgressCount {
				t.Errorf("InProgress count: got %d, want %d", len(items.InProgress), tt.expectedInProgressCount)
			}
		})
	}
}

func TestGetActionItems_AgeDays(t *testing.T) {
	tests := []struct {
		name            string
		task            *models.Task
		hoursAgo        int
		expectedAgeDays int
	}{
		{
			name: "task updated today",
			task: &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "E07-F01-001",
				Title: "Task 1"}, Status: "ready_for_approval",
			},
			hoursAgo:        1,
			expectedAgeDays: 0,
		},
		{
			name: "task updated 1 day ago",
			task: &models.Task{BaseEntity: models.BaseEntity{ID: 2,
				Key:   "E07-F01-002",
				Title: "Task 2"}, Status: "ready_for_approval",
			},
			hoursAgo:        24,
			expectedAgeDays: 1,
		},
		{
			name: "task updated 5 days ago",
			task: &models.Task{BaseEntity: models.BaseEntity{ID: 3,
				Key:   "E07-F01-003",
				Title: "Task 3"}, Status: "ready_for_approval",
			},
			hoursAgo:        120,
			expectedAgeDays: 5,
		},
		{
			name: "task updated 10 days ago",
			task: &models.Task{BaseEntity: models.BaseEntity{ID: 4,
				Key:   "E07-F01-004",
				Title: "Task 4"}, Status: "ready_for_approval", // Changed to ready_for_approval since ready_for_code_review doesn't count
			},
			hoursAgo:        240,
			expectedAgeDays: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			tt.task.UpdatedAt = now.Add(-time.Duration(tt.hoursAgo) * time.Hour)

			items := GetActionItems([]*models.Task{tt.task}, testWorkflowConfig())

			if len(items.AwaitingApproval) != 1 {
				t.Fatalf("Expected 1 awaiting approval task, got %d", len(items.AwaitingApproval))
			}

			item := items.AwaitingApproval[0]
			if item.AgeDays == nil {
				t.Fatalf("Expected AgeDays to be set, got nil")
			}

			if *item.AgeDays != tt.expectedAgeDays {
				t.Errorf("AgeDays: got %d, want %d", *item.AgeDays, tt.expectedAgeDays)
			}
		})
	}
}

func TestGetActionItems_TaskMetadata(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E07-F01-001",
		Title: "Implement feature",

		UpdatedAt: time.Now().Add(-24 * time.Hour)}, Status: "ready_for_approval",
	}

	items := GetActionItems([]*models.Task{task}, testWorkflowConfig())

	if len(items.AwaitingApproval) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(items.AwaitingApproval))
	}

	item := items.AwaitingApproval[0]

	if item.TaskKey != "E07-F01-001" {
		t.Errorf("TaskKey: got %q, want %q", item.TaskKey, "E07-F01-001")
	}

	if item.Title != "Implement feature" {
		t.Errorf("Title: got %q, want %q", item.Title, "Implement feature")
	}

	if item.Status != "ready_for_approval" {
		t.Errorf("Status: got %q, want %q", item.Status, "ready_for_approval")
	}

	if item.BlockedReason != nil {
		t.Errorf("BlockedReason: got %v, want nil", item.BlockedReason)
	}
}

func TestGetActionItems_BlockedTaskMetadata(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E07-F01-001",
		Title: "Blocked task",

		UpdatedAt: time.Now()}, Status: "blocked",
	}

	items := GetActionItems([]*models.Task{task}, testWorkflowConfig())

	if len(items.Blocked) != 1 {
		t.Fatalf("Expected 1 blocked task, got %d", len(items.Blocked))
	}

	item := items.Blocked[0]

	if item.TaskKey != "E07-F01-001" {
		t.Errorf("TaskKey: got %q, want %q", item.TaskKey, "E07-F01-001")
	}

	if item.Title != "Blocked task" {
		t.Errorf("Title: got %q, want %q", item.Title, "Blocked task")
	}

	if item.Status != "blocked" {
		t.Errorf("Status: got %q, want %q", item.Status, "blocked")
	}

	if item.AgeDays != nil {
		t.Errorf("AgeDays: got %v, want nil", item.AgeDays)
	}
}

func TestGetActionItems_InProgressTaskMetadata(t *testing.T) {
	tasks := []*models.Task{
		{
			BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01-001", Title: "In progress task", UpdatedAt: time.Now()},
			Status:     "in_progress",
		},
		{
			BaseEntity: models.BaseEntity{ID: 2, Key: "E07-F01-002", Title: "In development task", UpdatedAt: time.Now()},
			Status:     "in_development",
		},
	}

	items := GetActionItems(tasks, testWorkflowConfig())

	if len(items.InProgress) != 2 {
		t.Fatalf("Expected 2 in progress tasks, got %d", len(items.InProgress))
	}

	// Verify both tasks are included
	keys := map[string]bool{}
	for _, item := range items.InProgress {
		keys[item.TaskKey] = true
	}

	if !keys["E07-F01-001"] {
		t.Error("Expected task E07-F01-001 in InProgress")
	}

	if !keys["E07-F01-002"] {
		t.Error("Expected task E07-F01-002 in InProgress")
	}
}

func TestGetActionItems_EmptyInput(t *testing.T) {
	items := GetActionItems(nil, testWorkflowConfig())

	if items == nil {
		t.Fatal("Expected non-nil ActionItems, got nil")
	}

	if items.AwaitingApproval == nil {
		t.Error("Expected non-nil AwaitingApproval slice, got nil")
	}

	if items.Blocked == nil {
		t.Error("Expected non-nil Blocked slice, got nil")
	}

	if items.InProgress == nil {
		t.Error("Expected non-nil InProgress slice, got nil")
	}

	if len(items.AwaitingApproval) != 0 {
		t.Errorf("Expected 0 awaiting approval tasks, got %d", len(items.AwaitingApproval))
	}

	if len(items.Blocked) != 0 {
		t.Errorf("Expected 0 blocked tasks, got %d", len(items.Blocked))
	}

	if len(items.InProgress) != 0 {
		t.Errorf("Expected 0 in progress tasks, got %d", len(items.InProgress))
	}
}
