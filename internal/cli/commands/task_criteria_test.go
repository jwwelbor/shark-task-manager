package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) *repository.DB {
	database, err := db.InitDB(":memory:")
	require.NoError(t, err)
	return &repository.DB{DB: database}
}

// TestTaskCriteriaImport tests importing criteria from a task markdown file
func TestTaskCriteriaImport(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	criteriaRepo := repository.NewTaskCriteriaRepository(dbWrapper)

	// Create test epic
	priority := models.PriorityMedium
	epic := &models.Epic{
		Key:              "E99",
		Title:            "Test Epic",
		Status:           models.EpicStatusActive,
		Priority:         priority,
		BusinessValue:    &priority,
		CustomFolderPath: nil,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create test feature
	execOrder := 1
	feature := &models.Feature{
		EpicID:           epic.ID,
		Key:              "E99-F99",
		Title:            "Test Feature",
		Status:           models.FeatureStatusActive,
		CustomFolderPath: nil,
		ExecutionOrder:   &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create temporary directory for test task file
	tempDir := t.TempDir()
	taskFilePath := filepath.Join(tempDir, "T-E99-F99-001.md")

	// Create task file with acceptance criteria
	taskContent := `---
task_key: T-E99-F99-001
status: todo
title: Test Task
---

# Test Task

## Acceptance Criteria

- [ ] First criterion that is pending
- [x] Second criterion that is complete
- [ ] Third criterion that is pending
- [ ] Fourth criterion with details

Some other content here.
`

	err = os.WriteFile(taskFilePath, []byte(taskContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create task file: %v", err)
	}

	// Create test task
	filePathStr := taskFilePath
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F99-001",
		Title:     "Test Task",
		Status:    models.TaskStatusTodo,
		Priority:  5,
		FilePath:  &filePathStr,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Test: Import criteria
	t.Run("ImportCriteria", func(t *testing.T) {
		// Before import, should have no criteria
		criteriaList, err := criteriaRepo.GetByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get initial criteria: %v", err)
		}
		if len(criteriaList) != 0 {
			t.Errorf("Expected 0 initial criteria, got %d", len(criteriaList))
		}

		// Import criteria from file
		criteria, err := criteriaRepo.GetByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get criteria before import: %v", err)
		}
		if len(criteria) != 0 {
			t.Errorf("Expected 0 criteria before import, got %d", len(criteria))
		}

		// Parse and import from file (simulating the import command)
		parsedCriteria, err := parseCriteriaFromMarkdown(taskContent)
		if err != nil {
			t.Fatalf("Failed to parse criteria: %v", err)
		}

		for _, item := range parsedCriteria {
			criterion := &models.TaskCriteria{
				TaskID:    task.ID,
				Criterion: item.text,
				Status:    item.status,
			}
			err = criteriaRepo.Create(ctx, criterion)
			if err != nil {
				t.Fatalf("Failed to create criterion: %v", err)
			}
		}

		// Verify import
		imported, err := criteriaRepo.GetByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get imported criteria: %v", err)
		}

		if len(imported) != 4 {
			t.Errorf("Expected 4 imported criteria, got %d", len(imported))
		}

		// Verify statuses
		completeCount := 0
		pendingCount := 0
		for _, c := range imported {
			if c.Status == models.CriteriaStatusComplete {
				completeCount++
			} else if c.Status == models.CriteriaStatusPending {
				pendingCount++
			}
		}

		if completeCount != 1 {
			t.Errorf("Expected 1 complete criterion, got %d", completeCount)
		}
		if pendingCount != 3 {
			t.Errorf("Expected 3 pending criteria, got %d", pendingCount)
		}
	})

	// Test: Get summary
	t.Run("GetSummary", func(t *testing.T) {
		summary, err := criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get criteria summary: %v", err)
		}

		if summary.TotalCount != 4 {
			t.Errorf("Expected 4 total criteria, got %d", summary.TotalCount)
		}
		if summary.CompleteCount != 1 {
			t.Errorf("Expected 1 complete criterion, got %d", summary.CompleteCount)
		}
		if summary.PendingCount != 3 {
			t.Errorf("Expected 3 pending criteria, got %d", summary.PendingCount)
		}

		expectedPct := 25.0 // 1 complete out of 4 = 25%
		if summary.CompletionPct != expectedPct {
			t.Errorf("Expected %.0f%% completion, got %.0f%%", expectedPct, summary.CompletionPct)
		}
	})
}

// TestTaskCriteriaCheckAndFail tests marking criteria as complete or failed
func TestTaskCriteriaCheckAndFail(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	criteriaRepo := repository.NewTaskCriteriaRepository(dbWrapper)

	// Create test epic
	priority := models.PriorityMedium
	epic := &models.Epic{
		Key:              "E99",
		Title:            "Test Epic",
		Status:           models.EpicStatusActive,
		Priority:         priority,
		BusinessValue:    &priority,
		CustomFolderPath: nil,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create test feature
	execOrder := 1
	feature := &models.Feature{
		EpicID:           epic.ID,
		Key:              "E99-F99",
		Title:            "Test Feature",
		Status:           models.FeatureStatusActive,
		CustomFolderPath: nil,
		ExecutionOrder:   &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create test task
	task := &models.Task{
		FeatureID: feature.ID,
		Key:       "T-E99-F99-002",
		Title:     "Test Task 2",
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}
	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create test criteria
	criterion1 := &models.TaskCriteria{
		TaskID:    task.ID,
		Criterion: "First criterion",
		Status:    models.CriteriaStatusPending,
	}
	err = criteriaRepo.Create(ctx, criterion1)
	if err != nil {
		t.Fatalf("Failed to create criterion 1: %v", err)
	}

	criterion2 := &models.TaskCriteria{
		TaskID:    task.ID,
		Criterion: "Second criterion",
		Status:    models.CriteriaStatusPending,
	}
	err = criteriaRepo.Create(ctx, criterion2)
	if err != nil {
		t.Fatalf("Failed to create criterion 2: %v", err)
	}

	// Test: Mark criterion as complete (check)
	t.Run("CheckCriterion", func(t *testing.T) {
		note := "Verified with unit tests"
		err := criteriaRepo.UpdateStatus(ctx, criterion1.ID, models.CriteriaStatusComplete, &note)
		if err != nil {
			t.Fatalf("Failed to check criterion: %v", err)
		}

		// Verify update
		updated, err := criteriaRepo.GetByID(ctx, criterion1.ID)
		if err != nil {
			t.Fatalf("Failed to get updated criterion: %v", err)
		}

		if updated.Status != models.CriteriaStatusComplete {
			t.Errorf("Expected status 'complete', got '%s'", updated.Status)
		}
		if updated.VerifiedAt == nil {
			t.Error("Expected verified_at to be set")
		}
		if updated.VerificationNotes == nil || *updated.VerificationNotes != note {
			t.Errorf("Expected verification note '%s', got '%v'", note, updated.VerificationNotes)
		}

		// Verify summary
		summary, err := criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get summary: %v", err)
		}
		if summary.CompleteCount != 1 {
			t.Errorf("Expected 1 complete criterion, got %d", summary.CompleteCount)
		}
		if summary.PendingCount != 1 {
			t.Errorf("Expected 1 pending criterion, got %d", summary.PendingCount)
		}
	})

	// Test: Mark criterion as failed
	t.Run("FailCriterion", func(t *testing.T) {
		failNote := "Performance threshold not met"
		err := criteriaRepo.UpdateStatus(ctx, criterion2.ID, models.CriteriaStatusFailed, &failNote)
		if err != nil {
			t.Fatalf("Failed to fail criterion: %v", err)
		}

		// Verify update
		updated, err := criteriaRepo.GetByID(ctx, criterion2.ID)
		if err != nil {
			t.Fatalf("Failed to get updated criterion: %v", err)
		}

		if updated.Status != models.CriteriaStatusFailed {
			t.Errorf("Expected status 'failed', got '%s'", updated.Status)
		}
		if updated.VerifiedAt == nil {
			t.Error("Expected verified_at to be set")
		}
		if updated.VerificationNotes == nil || *updated.VerificationNotes != failNote {
			t.Errorf("Expected verification note '%s', got '%v'", failNote, updated.VerificationNotes)
		}

		// Verify summary
		summary, err := criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("Failed to get summary: %v", err)
		}
		if summary.CompleteCount != 1 {
			t.Errorf("Expected 1 complete criterion, got %d", summary.CompleteCount)
		}
		if summary.FailedCount != 1 {
			t.Errorf("Expected 1 failed criterion, got %d", summary.FailedCount)
		}
		if summary.PendingCount != 0 {
			t.Errorf("Expected 0 pending criteria, got %d", summary.PendingCount)
		}

		// Completion percentage should be 50% (1 complete out of 2)
		expectedPct := 50.0
		if summary.CompletionPct != expectedPct {
			t.Errorf("Expected %.0f%% completion, got %.0f%%", expectedPct, summary.CompletionPct)
		}
	})
}

// TestFeatureCriteriaAggregation tests aggregating criteria across all tasks in a feature
func TestFeatureCriteriaAggregation(t *testing.T) {
	// Create test database
	dbWrapper := setupTestDB(t)
	defer dbWrapper.Close()

	ctx := context.Background()
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	criteriaRepo := repository.NewTaskCriteriaRepository(dbWrapper)

	// Create test epic
	priority := models.PriorityMedium
	epic := &models.Epic{
		Key:              "E99",
		Title:            "Test Epic",
		Status:           models.EpicStatusActive,
		Priority:         priority,
		BusinessValue:    &priority,
		CustomFolderPath: nil,
	}
	err := epicRepo.Create(ctx, epic)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Create test feature
	execOrder := 1
	feature := &models.Feature{
		EpicID:           epic.ID,
		Key:              "E99-F99",
		Title:            "Test Feature",
		Status:           models.FeatureStatusActive,
		CustomFolderPath: nil,
		ExecutionOrder:   &execOrder,
	}
	err = featureRepo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create test tasks with criteria
	for i := 1; i <= 3; i++ {
		task := &models.Task{
			FeatureID: feature.ID,
			Key:       fmt.Sprintf("T-E99-F99-%03d", i),
			Title:     fmt.Sprintf("Test Task %d", i),
			Status:    models.TaskStatusTodo,
			Priority:  5,
		}
		err = taskRepo.Create(ctx, task)
		if err != nil {
			t.Fatalf("Failed to create task %d: %v", i, err)
		}

		// Create criteria for each task
		// Task 1: 3 criteria (2 complete, 1 pending)
		// Task 2: 2 criteria (1 complete, 1 failed)
		// Task 3: 4 criteria (all pending)
		var criteriaCount int
		var completeCount int
		var failedCount int

		if i == 1 {
			criteriaCount = 3
			completeCount = 2
		} else if i == 2 {
			criteriaCount = 2
			completeCount = 1
			failedCount = 1
		} else {
			criteriaCount = 4
		}

		for j := 1; j <= criteriaCount; j++ {
			status := models.CriteriaStatusPending
			if j <= completeCount {
				status = models.CriteriaStatusComplete
			} else if completeCount > 0 && j == completeCount+1 && failedCount > 0 {
				status = models.CriteriaStatusFailed
			}

			criterion := &models.TaskCriteria{
				TaskID:    task.ID,
				Criterion: fmt.Sprintf("Criterion %d for task %d", j, i),
				Status:    status,
			}
			err = criteriaRepo.Create(ctx, criterion)
			if err != nil {
				t.Fatalf("Failed to create criterion %d for task %d: %v", j, i, err)
			}
		}
	}

	// Test: Aggregate criteria across feature
	t.Run("AggregateFeatureCriteria", func(t *testing.T) {
		// Get all tasks for the feature
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Failed to get tasks: %v", err)
		}

		if len(tasks) != 3 {
			t.Fatalf("Expected 3 tasks, got %d", len(tasks))
		}

		// Aggregate criteria
		var totalCount int
		var completeCount int
		var pendingCount int
		var failedCount int

		for _, task := range tasks {
			summary, err := criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
			if err != nil {
				t.Fatalf("Failed to get summary for task %s: %v", task.Key, err)
			}

			totalCount += summary.TotalCount
			completeCount += summary.CompleteCount
			pendingCount += summary.PendingCount
			failedCount += summary.FailedCount
		}

		// Verify aggregation
		// Total: 3 + 2 + 4 = 9 criteria
		// Complete: 2 + 1 + 0 = 3
		// Pending: 1 + 0 + 4 = 5
		// Failed: 0 + 1 + 0 = 1
		if totalCount != 9 {
			t.Errorf("Expected 9 total criteria, got %d", totalCount)
		}
		if completeCount != 3 {
			t.Errorf("Expected 3 complete criteria, got %d", completeCount)
		}
		if pendingCount != 5 {
			t.Errorf("Expected 5 pending criteria, got %d", pendingCount)
		}
		if failedCount != 1 {
			t.Errorf("Expected 1 failed criterion, got %d", failedCount)
		}

		// Completion percentage: (3 complete + 0 na) / 9 total = 33.33%
		expectedPct := 33.33
		actualPct := float64(completeCount) / float64(totalCount) * 100.0
		if fmt.Sprintf("%.2f", actualPct) != fmt.Sprintf("%.2f", expectedPct) {
			t.Errorf("Expected %.2f%% completion, got %.2f%%", expectedPct, actualPct)
		}
	})
}

// Helper types and functions for testing

type criterionItem struct {
	text   string
	status models.CriteriaStatus
}

// parseCriteriaFromMarkdown is a simple parser for test purposes
func parseCriteriaFromMarkdown(content string) ([]criterionItem, error) {
	var criteria []criterionItem

	// Simple line-by-line parsing
	lines := splitLines(content)
	for _, line := range lines {
		// Match markdown checkboxes: - [ ] or - [x]
		if len(line) >= 6 && line[0] == '-' && line[1] == ' ' && line[2] == '[' && line[4] == ']' {
			status := models.CriteriaStatusPending
			if line[3] == 'x' || line[3] == 'X' {
				status = models.CriteriaStatusComplete
			}

			// Extract text after checkbox
			text := ""
			if len(line) > 5 {
				text = trimSpaceCriteria(line[5:])
			}

			if text != "" {
				criteria = append(criteria, criterionItem{
					text:   text,
					status: status,
				})
			}
		}
	}

	return criteria, nil
}

// Helper functions

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpaceCriteria(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
