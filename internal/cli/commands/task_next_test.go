package commands

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestSelectNextTasks_OrderPrioritySorting tests that selectNextTasks considers execution_order first, then priority
func TestSelectNextTasks_OrderPrioritySorting(t *testing.T) {
	// Create mock tasks with different orders and priorities
	order1 := 1
	order2 := 2
	order3 := 3

	now := time.Now()

	tasks := []*models.Task{
		{
			ID:             1,
			Key:            "T-E01-F01-001",
			Title:          "Task with order 2, priority 5",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			ExecutionOrder: &order2,
			CreatedAt:      now,
		},
		{
			ID:             2,
			Key:            "T-E01-F01-002",
			Title:          "Task with order 1, priority 3",
			Status:         models.TaskStatusTodo,
			Priority:       3,
			ExecutionOrder: &order1,
			CreatedAt:      now.Add(1 * time.Second),
		},
		{
			ID:             3,
			Key:            "T-E01-F01-003",
			Title:          "Task with order 3, priority 1",
			Status:         models.TaskStatusTodo,
			Priority:       1,
			ExecutionOrder: &order3,
			CreatedAt:      now.Add(2 * time.Second),
		},
		{
			ID:             4,
			Key:            "T-E01-F01-004",
			Title:          "Task with no order, priority 1",
			Status:         models.TaskStatusTodo,
			Priority:       1,
			ExecutionOrder: nil, // No order set
			CreatedAt:      now.Add(3 * time.Second),
		},
	}

	// Select next tasks
	nextTasks := selectNextTasks(tasks)

	// Should return task with order=1 (T-E01-F01-002)
	if len(nextTasks) != 1 {
		t.Errorf("Expected 1 next task, got %d", len(nextTasks))
	}

	if len(nextTasks) > 0 && nextTasks[0].Key != "T-E01-F01-002" {
		t.Errorf("Expected T-E01-F01-002 (order=1), got %s (order=%v)",
			nextTasks[0].Key, nextTasks[0].ExecutionOrder)
	}
}

// TestSelectNextTasks_SameOrderReturnsAll tests that tasks with same execution_order are all returned
func TestSelectNextTasks_SameOrderReturnsAll(t *testing.T) {
	// Create mock tasks with same order (parallel work)
	order1 := 1
	order2 := 2

	now := time.Now()

	tasks := []*models.Task{
		{
			ID:             1,
			Key:            "T-E01-F01-001",
			Title:          "Parallel task 1",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			ExecutionOrder: &order1,
			CreatedAt:      now,
		},
		{
			ID:             2,
			Key:            "T-E01-F01-002",
			Title:          "Parallel task 2",
			Status:         models.TaskStatusTodo,
			Priority:       3,
			ExecutionOrder: &order1, // Same order as task 1
			CreatedAt:      now.Add(1 * time.Second),
		},
		{
			ID:             3,
			Key:            "T-E01-F01-003",
			Title:          "Parallel task 3",
			Status:         models.TaskStatusTodo,
			Priority:       1,
			ExecutionOrder: &order1, // Same order as tasks 1 and 2
			CreatedAt:      now.Add(2 * time.Second),
		},
		{
			ID:             4,
			Key:            "T-E01-F01-004",
			Title:          "Sequential task (order 2)",
			Status:         models.TaskStatusTodo,
			Priority:       1,
			ExecutionOrder: &order2,
			CreatedAt:      now.Add(3 * time.Second),
		},
	}

	// Select next tasks
	nextTasks := selectNextTasks(tasks)

	// Should return all 3 tasks with order=1
	if len(nextTasks) != 3 {
		t.Errorf("Expected 3 parallel tasks with order=1, got %d", len(nextTasks))
	}

	// Verify all returned tasks have order=1
	for _, task := range nextTasks {
		if task.ExecutionOrder == nil || *task.ExecutionOrder != 1 {
			t.Errorf("Expected task %s to have order=1, got %v", task.Key, task.ExecutionOrder)
		}
	}

	// Verify they are sorted by priority (1 is highest, so it comes first)
	if len(nextTasks) == 3 {
		if nextTasks[0].Priority != 1 || nextTasks[1].Priority != 3 || nextTasks[2].Priority != 5 {
			t.Errorf("Expected tasks sorted by priority (1,3,5), got (%d,%d,%d)",
				nextTasks[0].Priority, nextTasks[1].Priority, nextTasks[2].Priority)
		}
	}
}

// TestSelectNextTasks_NullOrderComesLast tests that tasks without order come after ordered tasks
func TestSelectNextTasks_NullOrderComesLast(t *testing.T) {
	// Create mock tasks - some with order, some without
	order5 := 5

	now := time.Now()

	tasks := []*models.Task{
		{
			ID:             1,
			Key:            "T-E01-F01-001",
			Title:          "No order, priority 1",
			Status:         models.TaskStatusTodo,
			Priority:       1,
			ExecutionOrder: nil,
			CreatedAt:      now,
		},
		{
			ID:             2,
			Key:            "T-E01-F01-002",
			Title:          "Order 5, priority 10",
			Status:         models.TaskStatusTodo,
			Priority:       10,
			ExecutionOrder: &order5,
			CreatedAt:      now.Add(1 * time.Second),
		},
	}

	// Select next tasks
	nextTasks := selectNextTasks(tasks)

	// Should return task with order=5, not the one without order (even though it has higher priority)
	if len(nextTasks) != 1 {
		t.Errorf("Expected 1 next task, got %d", len(nextTasks))
	}

	if len(nextTasks) > 0 && nextTasks[0].Key != "T-E01-F01-002" {
		t.Errorf("Expected T-E01-F01-002 (order=5), got %s (order=%v)",
			nextTasks[0].Key, nextTasks[0].ExecutionOrder)
	}
}

// TestSelectNextTasks_NullOrderSortedByPriority tests that tasks without order are sorted by priority
func TestSelectNextTasks_NullOrderSortedByPriority(t *testing.T) {
	// Create mock tasks without order, different priorities
	now := time.Now()

	tasks := []*models.Task{
		{
			ID:             1,
			Key:            "T-E01-F01-001",
			Title:          "No order, priority 5",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			ExecutionOrder: nil,
			CreatedAt:      now,
		},
		{
			ID:             2,
			Key:            "T-E01-F01-002",
			Title:          "No order, priority 1",
			Status:         models.TaskStatusTodo,
			Priority:       1,
			ExecutionOrder: nil,
			CreatedAt:      now.Add(1 * time.Second),
		},
		{
			ID:             3,
			Key:            "T-E01-F01-003",
			Title:          "No order, priority 3",
			Status:         models.TaskStatusTodo,
			Priority:       3,
			ExecutionOrder: nil,
			CreatedAt:      now.Add(2 * time.Second),
		},
	}

	// Select next tasks
	nextTasks := selectNextTasks(tasks)

	// Should return task with priority 1 (highest priority)
	if len(nextTasks) != 1 {
		t.Errorf("Expected 1 next task, got %d", len(nextTasks))
	}

	if len(nextTasks) > 0 && nextTasks[0].Priority != 1 {
		t.Errorf("Expected task with priority 1, got priority %d (key=%s)",
			nextTasks[0].Priority, nextTasks[0].Key)
	}
}

// TestSelectNextTasks_SamePrioritySortedByCreatedAt tests tiebreaker using created_at
func TestSelectNextTasks_SamePrioritySortedByCreatedAt(t *testing.T) {
	// Create mock tasks with same priority, different created_at times
	now := time.Now()

	tasks := []*models.Task{
		{
			ID:             1,
			Key:            "T-E01-F01-001",
			Title:          "Newer task",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			ExecutionOrder: nil,
			CreatedAt:      now.Add(5 * time.Second),
		},
		{
			ID:             2,
			Key:            "T-E01-F01-002",
			Title:          "Older task",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			ExecutionOrder: nil,
			CreatedAt:      now,
		},
	}

	// Select next tasks
	nextTasks := selectNextTasks(tasks)

	// Should return older task (created first)
	if len(nextTasks) != 1 {
		t.Errorf("Expected 1 next task, got %d", len(nextTasks))
	}

	if len(nextTasks) > 0 && nextTasks[0].Key != "T-E01-F01-002" {
		t.Errorf("Expected T-E01-F01-002 (older task), got %s", nextTasks[0].Key)
	}
}

// TestSelectNextTasks_EmptyList tests handling of empty task list
func TestSelectNextTasks_EmptyList(t *testing.T) {
	tasks := []*models.Task{}

	nextTasks := selectNextTasks(tasks)

	if len(nextTasks) != 0 {
		t.Errorf("Expected 0 tasks for empty input, got %d", len(nextTasks))
	}
}

// Note: selectNextTasks, compareTasksForNext, and bothNil are implemented in task.go
