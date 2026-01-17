package status

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

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
					ID:        1,
					Key:       "E07-F01-001",
					Title:     "Task 1",
					Status:    "ready_for_approval",
					UpdatedAt: now.Add(-24 * time.Hour),
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
					ID:        2,
					Key:       "E07-F01-002",
					Title:     "Task 2",
					Status:    "ready_for_code_review",
					UpdatedAt: now.Add(-48 * time.Hour),
				},
			},
			expectedAwaitingCount:   1,
			expectedBlockedCount:    0,
			expectedInProgressCount: 0,
		},
		{
			name: "single blocked task",
			tasks: []*models.Task{
				{
					ID:        3,
					Key:       "E07-F01-003",
					Title:     "Task 3",
					Status:    "blocked",
					UpdatedAt: now,
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
					ID:        4,
					Key:       "E07-F01-004",
					Title:     "Task 4",
					Status:    "in_progress",
					UpdatedAt: now,
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
					ID:        5,
					Key:       "E07-F01-005",
					Title:     "Task 5",
					Status:    "in_development",
					UpdatedAt: now,
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
					ID:        1,
					Key:       "E07-F01-001",
					Title:     "Task 1",
					Status:    "ready_for_approval",
					UpdatedAt: now.Add(-1 * time.Hour),
				},
				{
					ID:        2,
					Key:       "E07-F01-002",
					Title:     "Task 2",
					Status:    "blocked",
					UpdatedAt: now,
				},
				{
					ID:        3,
					Key:       "E07-F01-003",
					Title:     "Task 3",
					Status:    "in_progress",
					UpdatedAt: now,
				},
				{
					ID:        4,
					Key:       "E07-F01-004",
					Title:     "Task 4",
					Status:    "todo",
					UpdatedAt: now,
				},
				{
					ID:        5,
					Key:       "E07-F01-005",
					Title:     "Task 5",
					Status:    "ready_for_code_review",
					UpdatedAt: now.Add(-2 * time.Hour),
				},
			},
			expectedAwaitingCount:   2,
			expectedBlockedCount:    1,
			expectedInProgressCount: 1,
		},
		{
			name: "multiple in_progress tasks",
			tasks: []*models.Task{
				{
					ID:        1,
					Key:       "E07-F01-001",
					Title:     "Task 1",
					Status:    "in_progress",
					UpdatedAt: now,
				},
				{
					ID:        2,
					Key:       "E07-F01-002",
					Title:     "Task 2",
					Status:    "in_progress",
					UpdatedAt: now,
				},
				{
					ID:        3,
					Key:       "E07-F01-003",
					Title:     "Task 3",
					Status:    "in_development",
					UpdatedAt: now,
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
					ID:        1,
					Key:       "E07-F01-001",
					Title:     "Task 1",
					Status:    "todo",
					UpdatedAt: now,
				},
				{
					ID:        2,
					Key:       "E07-F01-002",
					Title:     "Task 2",
					Status:    "completed",
					UpdatedAt: now,
				},
				{
					ID:        3,
					Key:       "E07-F01-003",
					Title:     "Task 3",
					Status:    "draft",
					UpdatedAt: now,
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
			items := GetActionItems(tt.tasks, nil)

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
			task: &models.Task{
				ID:     1,
				Key:    "E07-F01-001",
				Title:  "Task 1",
				Status: "ready_for_approval",
			},
			hoursAgo:        1,
			expectedAgeDays: 0,
		},
		{
			name: "task updated 1 day ago",
			task: &models.Task{
				ID:     2,
				Key:    "E07-F01-002",
				Title:  "Task 2",
				Status: "ready_for_approval",
			},
			hoursAgo:        24,
			expectedAgeDays: 1,
		},
		{
			name: "task updated 5 days ago",
			task: &models.Task{
				ID:     3,
				Key:    "E07-F01-003",
				Title:  "Task 3",
				Status: "ready_for_approval",
			},
			hoursAgo:        120,
			expectedAgeDays: 5,
		},
		{
			name: "task updated 10 days ago",
			task: &models.Task{
				ID:     4,
				Key:    "E07-F01-004",
				Title:  "Task 4",
				Status: "ready_for_code_review",
			},
			hoursAgo:        240,
			expectedAgeDays: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			tt.task.UpdatedAt = now.Add(-time.Duration(tt.hoursAgo) * time.Hour)

			items := GetActionItems([]*models.Task{tt.task}, nil)

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
	task := &models.Task{
		ID:        1,
		Key:       "E07-F01-001",
		Title:     "Implement feature",
		Status:    "ready_for_approval",
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	items := GetActionItems([]*models.Task{task}, nil)

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
	task := &models.Task{
		ID:        1,
		Key:       "E07-F01-001",
		Title:     "Blocked task",
		Status:    "blocked",
		UpdatedAt: time.Now(),
	}

	items := GetActionItems([]*models.Task{task}, nil)

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
			ID:        1,
			Key:       "E07-F01-001",
			Title:     "In progress task",
			Status:    "in_progress",
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			Key:       "E07-F01-002",
			Title:     "In development task",
			Status:    "in_development",
			UpdatedAt: time.Now(),
		},
	}

	items := GetActionItems(tasks, nil)

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
	items := GetActionItems(nil, nil)

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
