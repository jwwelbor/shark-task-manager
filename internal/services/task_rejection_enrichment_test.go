package services

// Tests for rejection-count enrichment on TaskService.GetTask and
// TaskService.ListTasks, plus the --has-rejections filter behavior.
// Uses MockTaskRepository from task_service_test.go.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestTaskService_GetTask_PopulatesRejectionCount verifies GetTask enriches
// the returned task with RejectionCount and LastRejectionAt drawn from the
// repository's batched query.
func TestTaskService_GetTask_PopulatesRejectionCount(t *testing.T) {
	ctx := context.Background()
	last := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 42, Key: key, Title: "T"},
				Status:     "in_progress",
				Priority:   5,
			}, nil
		},
		GetRejectionCountsFunc: func(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
			if len(taskIDs) != 1 || taskIDs[0] != 42 {
				t.Errorf("expected taskIDs=[42], got %v", taskIDs)
			}
			return map[int64]int{42: 3}, map[int64]*time.Time{42: &last}, nil
		},
	}

	svc := NewTaskService(repo, NewEntityService(newMockWorkflowService()), nil)
	task, err := svc.GetTask(ctx, "T-E07-F01-001")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.RejectionCount != 3 {
		t.Errorf("RejectionCount = %d, want 3", task.RejectionCount)
	}
	if task.LastRejectionAt == nil || !task.LastRejectionAt.Equal(last) {
		t.Errorf("LastRejectionAt = %v, want %v", task.LastRejectionAt, last)
	}
}

// TestTaskService_GetTask_ZeroRejectionCount verifies that when the repo
// returns no rejection rows for the task, RejectionCount stays 0 and
// LastRejectionAt stays nil (no false positives).
func TestTaskService_GetTask_ZeroRejectionCount(t *testing.T) {
	ctx := context.Background()
	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 7, Key: key, Title: "T"},
				Status:     "todo",
				Priority:   5,
			}, nil
		},
		// Default mock returns empty maps; explicit override here for clarity.
		GetRejectionCountsFunc: func(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
			return map[int64]int{}, map[int64]*time.Time{}, nil
		},
	}

	svc := NewTaskService(repo, NewEntityService(newMockWorkflowService()), nil)
	task, err := svc.GetTask(ctx, "T-E07-F01-001")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.RejectionCount != 0 {
		t.Errorf("RejectionCount = %d, want 0", task.RejectionCount)
	}
	if task.LastRejectionAt != nil {
		t.Errorf("LastRejectionAt = %v, want nil", task.LastRejectionAt)
	}
}

// TestTaskService_GetTask_RejectionCountError propagates errors from the
// underlying repository query; the caller should see a wrapped error and
// no task.
func TestTaskService_GetTask_RejectionCountError(t *testing.T) {
	ctx := context.Background()
	repo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "T"},
				Status:     "todo",
				Priority:   5,
			}, nil
		},
		GetRejectionCountsFunc: func(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
			return nil, nil, errors.New("db boom")
		},
	}

	svc := NewTaskService(repo, NewEntityService(newMockWorkflowService()), nil)
	_, err := svc.GetTask(ctx, "T-E07-F01-001")
	if err == nil {
		t.Fatal("GetTask() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rejection counts") {
		t.Errorf("error = %q, want it to mention 'rejection counts'", err.Error())
	}
}

// TestTaskService_ListTasks_PopulatesRejectionCounts verifies the batched
// enrichment populates each returned task with its count.
func TestTaskService_ListTasks_PopulatesRejectionCounts(t *testing.T) {
	ctx := context.Background()
	tasks := []*models.Task{
		{BaseEntity: models.BaseEntity{ID: 10, Key: "T-E07-F01-001", Title: "A"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 11, Key: "T-E07-F01-002", Title: "B"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 12, Key: "T-E07-F01-003", Title: "C"}, Status: "todo", Priority: 5},
	}
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) { return tasks, nil },
		GetRejectionCountsFunc: func(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
			if len(taskIDs) != 3 {
				t.Errorf("expected 3 task IDs, got %d", len(taskIDs))
			}
			return map[int64]int{10: 2, 12: 1}, map[int64]*time.Time{}, nil
		},
	}

	svc := NewTaskService(repo, NewEntityService(newMockWorkflowService()), nil)
	got, err := svc.ListTasks(ctx, TaskFilters{})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d tasks, want 3", len(got))
	}
	wantCounts := map[int64]int{10: 2, 11: 0, 12: 1}
	for _, tk := range got {
		if tk.RejectionCount != wantCounts[tk.ID] {
			t.Errorf("task %d RejectionCount = %d, want %d", tk.ID, tk.RejectionCount, wantCounts[tk.ID])
		}
	}
}

// TestTaskService_ListTasks_HasRejectionsFilter verifies that
// HasRejections=true drops tasks whose RejectionCount is 0.
func TestTaskService_ListTasks_HasRejectionsFilter(t *testing.T) {
	ctx := context.Background()
	tasks := []*models.Task{
		{BaseEntity: models.BaseEntity{ID: 10, Key: "T-E07-F01-001", Title: "A"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 11, Key: "T-E07-F01-002", Title: "B"}, Status: "todo", Priority: 5},
		{BaseEntity: models.BaseEntity{ID: 12, Key: "T-E07-F01-003", Title: "C"}, Status: "todo", Priority: 5},
	}
	repo := &MockTaskRepository{
		ListFunc: func(ctx context.Context) ([]*models.Task, error) { return tasks, nil },
		GetRejectionCountsFunc: func(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
			return map[int64]int{10: 2, 12: 1}, map[int64]*time.Time{}, nil
		},
	}

	svc := NewTaskService(repo, NewEntityService(newMockWorkflowService()), nil)
	got, err := svc.ListTasks(ctx, TaskFilters{HasRejections: true})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d tasks, want 2 (only those with rejections)", len(got))
	}
	for _, tk := range got {
		if tk.RejectionCount == 0 {
			t.Errorf("task %d included despite RejectionCount=0", tk.ID)
		}
	}
}
