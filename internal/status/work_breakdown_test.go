package status

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestCalculateWorkRemaining_AllAgentWork tests that all agent work is categorized correctly
func TestCalculateWorkRemaining_AllAgentWork(t *testing.T) {
	statusCounts := map[string]int{
		"in_development": 5,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"in_development": {
			Responsibility: "agent",
			ProgressWeight: 0.5,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 5 {
		t.Errorf("expected TotalTasks=5, got %d", summary.TotalTasks)
	}
	if summary.AgentWork != 5 {
		t.Errorf("expected AgentWork=5, got %d", summary.AgentWork)
	}
	if summary.HumanWork != 0 {
		t.Errorf("expected HumanWork=0, got %d", summary.HumanWork)
	}
	if summary.BlockedWork != 0 {
		t.Errorf("expected BlockedWork=0, got %d", summary.BlockedWork)
	}
}

// TestCalculateWorkRemaining_AllHumanWork tests that human work (human, qa_team) is categorized correctly
func TestCalculateWorkRemaining_AllHumanWork(t *testing.T) {
	statusCounts := map[string]int{
		"ready_for_review": 3,
		"in_qa":            2,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"ready_for_review": {
			Responsibility: "human",
			ProgressWeight: 0.8,
			BlocksFeature:  false,
		},
		"in_qa": {
			Responsibility: "qa_team",
			ProgressWeight: 0.7,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 5 {
		t.Errorf("expected TotalTasks=5, got %d", summary.TotalTasks)
	}
	if summary.HumanWork != 5 {
		t.Errorf("expected HumanWork=5, got %d", summary.HumanWork)
	}
	if summary.AgentWork != 0 {
		t.Errorf("expected AgentWork=0, got %d", summary.AgentWork)
	}
}

// TestCalculateWorkRemaining_BlockedTasks tests that blocked tasks (blocks_feature=true) are categorized
func TestCalculateWorkRemaining_BlockedTasks(t *testing.T) {
	statusCounts := map[string]int{
		"blocked": 2,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"blocked": {
			Responsibility: "none",
			ProgressWeight: 0.0,
			BlocksFeature:  true,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 2 {
		t.Errorf("expected TotalTasks=2, got %d", summary.TotalTasks)
	}
	if summary.BlockedWork != 2 {
		t.Errorf("expected BlockedWork=2, got %d", summary.BlockedWork)
	}
	if summary.NotStarted != 0 {
		t.Errorf("expected NotStarted=0, got %d", summary.NotStarted)
	}
}

// TestCalculateWorkRemaining_NotStarted tests that not started tasks (progress_weight=0.0) are categorized
func TestCalculateWorkRemaining_NotStarted(t *testing.T) {
	statusCounts := map[string]int{
		"todo": 4,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"todo": {
			Responsibility: "none",
			ProgressWeight: 0.0,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 4 {
		t.Errorf("expected TotalTasks=4, got %d", summary.TotalTasks)
	}
	if summary.NotStarted != 4 {
		t.Errorf("expected NotStarted=4, got %d", summary.NotStarted)
	}
	if summary.BlockedWork != 0 {
		t.Errorf("expected BlockedWork=0, got %d", summary.BlockedWork)
	}
}

// TestCalculateWorkRemaining_CompletedTasks tests that completed tasks (progress_weight>=1.0) are categorized
func TestCalculateWorkRemaining_CompletedTasks(t *testing.T) {
	statusCounts := map[string]int{
		"completed": 3,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"completed": {
			Responsibility: "none",
			ProgressWeight: 1.0,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 3 {
		t.Errorf("expected TotalTasks=3, got %d", summary.TotalTasks)
	}
	if summary.CompletedTasks != 3 {
		t.Errorf("expected CompletedTasks=3, got %d", summary.CompletedTasks)
	}
}

// TestCalculateWorkRemaining_MixedResponsibilities tests that mixed responsibilities are categorized correctly
func TestCalculateWorkRemaining_MixedResponsibilities(t *testing.T) {
	statusCounts := map[string]int{
		"in_development":   5,
		"ready_for_review": 2,
		"blocked":          1,
		"todo":             2,
		"completed":        3,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"in_development": {
			Responsibility: "agent",
			ProgressWeight: 0.5,
			BlocksFeature:  false,
		},
		"ready_for_review": {
			Responsibility: "human",
			ProgressWeight: 0.8,
			BlocksFeature:  false,
		},
		"blocked": {
			Responsibility: "none",
			ProgressWeight: 0.0,
			BlocksFeature:  true,
		},
		"todo": {
			Responsibility: "none",
			ProgressWeight: 0.0,
			BlocksFeature:  false,
		},
		"completed": {
			Responsibility: "none",
			ProgressWeight: 1.0,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 13 {
		t.Errorf("expected TotalTasks=13, got %d", summary.TotalTasks)
	}
	if summary.AgentWork != 5 {
		t.Errorf("expected AgentWork=5, got %d", summary.AgentWork)
	}
	if summary.HumanWork != 2 {
		t.Errorf("expected HumanWork=2, got %d", summary.HumanWork)
	}
	if summary.BlockedWork != 1 {
		t.Errorf("expected BlockedWork=1, got %d", summary.BlockedWork)
	}
	if summary.NotStarted != 2 {
		t.Errorf("expected NotStarted=2, got %d", summary.NotStarted)
	}
	if summary.CompletedTasks != 3 {
		t.Errorf("expected CompletedTasks=3, got %d", summary.CompletedTasks)
	}
}

// TestCalculateWorkRemaining_MissingMetadata tests graceful handling when metadata is missing
func TestCalculateWorkRemaining_MissingMetadata(t *testing.T) {
	statusCounts := map[string]int{
		"unknown_status": 2,
		"in_development": 3,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"in_development": {
			Responsibility: "agent",
			ProgressWeight: 0.5,
			BlocksFeature:  false,
		},
		// "unknown_status" has no metadata
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	// Only the known status should be counted
	if summary.TotalTasks != 3 {
		t.Errorf("expected TotalTasks=3 (skipped unknown), got %d", summary.TotalTasks)
	}
	if summary.AgentWork != 3 {
		t.Errorf("expected AgentWork=3, got %d", summary.AgentWork)
	}
}

// TestCalculateWorkRemaining_PartialProgressWeight tests intermediate progress weights
func TestCalculateWorkRemaining_PartialProgressWeight(t *testing.T) {
	statusCounts := map[string]int{
		"ready_for_approval": 1,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"ready_for_approval": {
			Responsibility: "human",
			ProgressWeight: 0.9,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 1 {
		t.Errorf("expected TotalTasks=1, got %d", summary.TotalTasks)
	}
	if summary.HumanWork != 1 {
		t.Errorf("expected HumanWork=1, got %d", summary.HumanWork)
	}
	if summary.CompletedTasks != 0 {
		t.Errorf("expected CompletedTasks=0 (progress_weight < 1.0), got %d", summary.CompletedTasks)
	}
}

// TestCalculateWorkRemaining_EmptyStatusCounts tests handling of empty status counts
func TestCalculateWorkRemaining_EmptyStatusCounts(t *testing.T) {
	statusCounts := map[string]int{}

	wf := createTestWorkflow(map[string]config.StatusMetadata{})

	summary := CalculateWorkRemaining(statusCounts, wf)

	if summary.TotalTasks != 0 {
		t.Errorf("expected TotalTasks=0, got %d", summary.TotalTasks)
	}
	if summary.AgentWork != 0 {
		t.Errorf("expected AgentWork=0, got %d", summary.AgentWork)
	}
	if summary.HumanWork != 0 {
		t.Errorf("expected HumanWork=0, got %d", summary.HumanWork)
	}
	if summary.BlockedWork != 0 {
		t.Errorf("expected BlockedWork=0, got %d", summary.BlockedWork)
	}
	if summary.NotStarted != 0 {
		t.Errorf("expected NotStarted=0, got %d", summary.NotStarted)
	}
	if summary.CompletedTasks != 0 {
		t.Errorf("expected CompletedTasks=0, got %d", summary.CompletedTasks)
	}
}

// TestCalculateWorkRemaining_OverlapCompletedAndCategory tests that completed status is counted in both
func TestCalculateWorkRemaining_OverlapCompletedAndCategory(t *testing.T) {
	statusCounts := map[string]int{
		"completed": 2,
	}

	wf := createTestWorkflow(map[string]config.StatusMetadata{
		"completed": {
			Responsibility: "agent",
			ProgressWeight: 1.0,
			BlocksFeature:  false,
		},
	})

	summary := CalculateWorkRemaining(statusCounts, wf)

	// Completed tasks should be in both CompletedTasks and their responsibility category
	if summary.TotalTasks != 2 {
		t.Errorf("expected TotalTasks=2, got %d", summary.TotalTasks)
	}
	if summary.CompletedTasks != 2 {
		t.Errorf("expected CompletedTasks=2, got %d", summary.CompletedTasks)
	}
	if summary.AgentWork != 2 {
		t.Errorf("expected AgentWork=2 (completed items are also agents), got %d", summary.AgentWork)
	}
}

// Helper function to create a test workflow with status metadata
func createTestWorkflow(metadata map[string]config.StatusMetadata) *config.WorkflowConfig {
	return &config.WorkflowConfig{
		StatusMetadata: metadata,
	}
}
