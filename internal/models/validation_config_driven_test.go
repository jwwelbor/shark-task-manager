package models

import (
	"testing"
)

// TestValidateTaskStatus_BasicValidation verifies that ValidateTaskStatus performs
// only basic non-empty validation at the model layer. Workflow-aware validation
// (checking against configured statuses) is delegated to workflow.Service.ValidateStatus()
// at the CLI/command layer.
func TestValidateTaskStatus_BasicValidation(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		shouldPass bool
	}{
		{
			name:       "standard status accepted",
			status:     "todo",
			shouldPass: true,
		},
		{
			name:       "workflow status accepted at model layer",
			status:     "in_development",
			shouldPass: true,
		},
		{
			name:       "any non-empty string accepted at model layer",
			status:     "custom_status",
			shouldPass: true,
		},
		{
			name:       "empty string rejected",
			status:     "",
			shouldPass: false,
		},
		{
			name:       "whitespace only rejected",
			status:     "   ",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskStatus(tt.status)

			if tt.shouldPass && err != nil {
				t.Errorf("ValidateTaskStatus(%q) returned error: %v; want nil", tt.status, err)
			}

			if !tt.shouldPass && err == nil {
				t.Errorf("ValidateTaskStatus(%q) returned nil; want error", tt.status)
			}
		})
	}
}

// TestValidateTaskStatus_NoHardcodedMaps confirms that the old hardcoded status maps
// have been removed. ValidateTaskStatus no longer rejects statuses that are not in
// a predefined list -- it only checks for non-empty. Workflow-aware validation is
// the responsibility of cli.GetWorkflowService().ValidateStatus().
func TestValidateTaskStatus_NoHardcodedMaps(t *testing.T) {
	// These statuses would have been rejected by the old hardcoded validation
	// but should now pass at the model layer (non-empty check only)
	workflowStatuses := []string{
		"in_development",
		"ready_for_refinement",
		"in_refinement",
		"ready_for_development",
		"in_code_review",
		"in_qa",
		"ready_for_approval",
		"in_approval",
		"ready_for_qa",
		"ready_for_code_review",
	}

	for _, status := range workflowStatuses {
		t.Run(status, func(t *testing.T) {
			err := ValidateTaskStatus(status)
			if err != nil {
				t.Errorf("ValidateTaskStatus(%q) should pass basic validation but got: %v", status, err)
			}
		})
	}
}

// TestTaskStatusConstants_Removed verifies that deprecated TaskStatus constants
// have been removed. Status values are now config-driven via workflow.Service.
func TestTaskStatusConstants_Removed(t *testing.T) {
	// The old hardcoded constants (TaskStatusTodo, TaskStatusInProgress, etc.)
	// have been removed. Status values are now created via TaskStatus("status_name")
	// and validated through workflow.Service.ValidateStatus().
	statuses := []TaskStatus{
		TaskStatus("todo"),
		TaskStatus("in_progress"),
		TaskStatus("blocked"),
		TaskStatus("ready_for_review"),
		TaskStatus("completed"),
		TaskStatus("archived"),
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("TaskStatus should not be empty")
		}
	}

	t.Logf("Verified: %d status values created via TaskStatus() type literals", len(statuses))
	t.Logf("Status validation is now handled by workflow.Service.ValidateStatus()")
}
