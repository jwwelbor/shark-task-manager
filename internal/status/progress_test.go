package status

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestCalculateProgress_AllCompleted tests 100% completion
func TestCalculateProgress_AllCompleted(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"completed": {ProgressWeight: 1.0},
		},
	}

	statusCounts := map[string]int{
		"completed": 5,
	}

	result := CalculateProgress(statusCounts, cfg)

	if result.WeightedPct != 100.0 {
		t.Errorf("expected weighted %v, got %v", 100.0, result.WeightedPct)
	}
	if result.CompletionPct != 100.0 {
		t.Errorf("expected completion %v, got %v", 100.0, result.CompletionPct)
	}
	if result.TotalTasks != 5 {
		t.Errorf("expected total 5, got %v", result.TotalTasks)
	}
	if result.WeightedRatio != "5.0/5" {
		t.Errorf("expected weighted ratio '5.0/5', got '%s'", result.WeightedRatio)
	}
	if result.CompletionRatio != "5/5" {
		t.Errorf("expected completion ratio '5/5', got '%s'", result.CompletionRatio)
	}
}

// TestCalculateProgress_MixedStatuses tests mixed statuses (example from spec)
func TestCalculateProgress_MixedStatuses(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"completed":          {ProgressWeight: 1.0},
			"ready_for_approval": {ProgressWeight: 0.9},
			"in_development":     {ProgressWeight: 0.5},
			"draft":              {ProgressWeight: 0.0},
		},
	}

	// From spec: 2 completed, 1 ready_for_approval, 1 in_development, 1 draft
	statusCounts := map[string]int{
		"completed":          2,
		"ready_for_approval": 1,
		"in_development":     1,
		"draft":              1,
	}

	result := CalculateProgress(statusCounts, cfg)

	// Expected weighted: (2.0 + 0.9 + 0.5 + 0.0) / 5 = 68%
	expectedWeighted := 68.0
	expectedCompletion := 40.0

	if result.WeightedPct != expectedWeighted {
		t.Errorf("expected weighted %v, got %v", expectedWeighted, result.WeightedPct)
	}
	if result.CompletionPct != expectedCompletion {
		t.Errorf("expected completion %v, got %v", expectedCompletion, result.CompletionPct)
	}
	if result.TotalTasks != 5 {
		t.Errorf("expected total 5, got %v", result.TotalTasks)
	}
	if result.WeightedRatio != "3.4/5" {
		t.Errorf("expected weighted ratio '3.4/5', got '%s'", result.WeightedRatio)
	}
	if result.CompletionRatio != "2/5" {
		t.Errorf("expected completion ratio '2/5', got '%s'", result.CompletionRatio)
	}
}

// TestCalculateProgress_EmptyTasks tests empty task list
func TestCalculateProgress_EmptyTasks(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"completed": {ProgressWeight: 1.0},
		},
	}

	statusCounts := map[string]int{}

	result := CalculateProgress(statusCounts, cfg)

	if result.WeightedPct != 0.0 {
		t.Errorf("expected weighted 0.0, got %v", result.WeightedPct)
	}
	if result.CompletionPct != 0.0 {
		t.Errorf("expected completion 0.0, got %v", result.CompletionPct)
	}
	if result.TotalTasks != 0 {
		t.Errorf("expected total 0, got %v", result.TotalTasks)
	}
	if result.WeightedRatio != "0/0" {
		t.Errorf("expected weighted ratio '0/0', got '%s'", result.WeightedRatio)
	}
	if result.CompletionRatio != "0/0" {
		t.Errorf("expected completion ratio '0/0', got '%s'", result.CompletionRatio)
	}
}

// TestCalculateProgress_MissingMetadata tests missing metadata (defaults to 0.0)
func TestCalculateProgress_MissingMetadata(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"completed": {ProgressWeight: 1.0},
		},
	}

	// Status "in_progress" not in metadata
	statusCounts := map[string]int{
		"completed":  2,
		"in_progress": 3,
	}

	result := CalculateProgress(statusCounts, cfg)

	// Only completed tasks count: 2 / 5 = 40%
	expectedCompletion := 40.0
	// Weighted: only 2 out of 5 (missing metadata defaults to 0.0)
	expectedWeighted := 40.0

	if result.WeightedPct != expectedWeighted {
		t.Errorf("expected weighted %v, got %v", expectedWeighted, result.WeightedPct)
	}
	if result.CompletionPct != expectedCompletion {
		t.Errorf("expected completion %v, got %v", expectedCompletion, result.CompletionPct)
	}
	if result.TotalTasks != 5 {
		t.Errorf("expected total 5, got %v", result.TotalTasks)
	}
}

// TestCalculateProgress_SingleTaskReadyForApproval tests single task at 90%
func TestCalculateProgress_SingleTaskReadyForApproval(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"ready_for_approval": {ProgressWeight: 0.9},
		},
	}

	statusCounts := map[string]int{
		"ready_for_approval": 1,
	}

	result := CalculateProgress(statusCounts, cfg)

	if result.WeightedPct != 90.0 {
		t.Errorf("expected weighted 90.0, got %v", result.WeightedPct)
	}
	if result.CompletionPct != 0.0 {
		t.Errorf("expected completion 0.0, got %v", result.CompletionPct)
	}
	if result.TotalTasks != 1 {
		t.Errorf("expected total 1, got %v", result.TotalTasks)
	}
	if result.WeightedRatio != "0.9/1" {
		t.Errorf("expected weighted ratio '0.9/1', got '%s'", result.WeightedRatio)
	}
	if result.CompletionRatio != "0/1" {
		t.Errorf("expected completion ratio '0/1', got '%s'", result.CompletionRatio)
	}
}

// TestCalculateProgress_VariousWeights tests various progress weights
func TestCalculateProgress_VariousWeights(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"completed":      {ProgressWeight: 1.0},
			"in_review":      {ProgressWeight: 0.75},
			"in_progress":    {ProgressWeight: 0.5},
			"blocked":        {ProgressWeight: 0.25},
			"todo":           {ProgressWeight: 0.0},
		},
	}

	statusCounts := map[string]int{
		"completed":   2,
		"in_review":   2,
		"in_progress": 2,
		"blocked":     2,
		"todo":        2,
	}

	result := CalculateProgress(statusCounts, cfg)

	// Weighted: (2*1.0 + 2*0.75 + 2*0.5 + 2*0.25 + 2*0.0) / 10 = 5.0/10 = 50%
	expectedWeighted := 50.0
	// Completion: 2 / 10 = 20%
	expectedCompletion := 20.0

	if result.WeightedPct != expectedWeighted {
		t.Errorf("expected weighted %v, got %v", expectedWeighted, result.WeightedPct)
	}
	if result.CompletionPct != expectedCompletion {
		t.Errorf("expected completion %v, got %v", expectedCompletion, result.CompletionPct)
	}
	if result.TotalTasks != 10 {
		t.Errorf("expected total 10, got %v", result.TotalTasks)
	}
}

// TestCalculateProgress_NilConfig tests with nil WorkflowConfig
func TestCalculateProgress_NilConfig(t *testing.T) {
	var cfg *config.WorkflowConfig

	statusCounts := map[string]int{
		"todo": 5,
	}

	result := CalculateProgress(statusCounts, cfg)

	// With nil config, all statuses default to weight 0.0
	if result.WeightedPct != 0.0 {
		t.Errorf("expected weighted 0.0, got %v", result.WeightedPct)
	}
	if result.CompletionPct != 0.0 {
		t.Errorf("expected completion 0.0, got %v", result.CompletionPct)
	}
	if result.TotalTasks != 5 {
		t.Errorf("expected total 5, got %v", result.TotalTasks)
	}
}
