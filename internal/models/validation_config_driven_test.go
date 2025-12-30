package models

import (
	"strings"
	"testing"
)

// TestValidateTaskStatus_HardcodedLimitation demonstrates the problem with hardcoded status validation.
// This test SHOULD FAIL initially because ValidateTaskStatus only accepts the 6 hardcoded statuses.
func TestValidateTaskStatus_HardcodedLimitation(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		shouldPass  bool
		description string
	}{
		{
			name:        "new workflow status rejected",
			status:      "in_development",
			shouldPass:  true, // We WANT this to pass, but it will FAIL with hardcoded validation
			description: "New workflow status from .sharkconfig.json should be valid",
		},
		{
			name:        "ready_for_refinement rejected",
			status:      "ready_for_refinement",
			shouldPass:  true, // We WANT this to pass, but it will FAIL with hardcoded validation
			description: "Status from 14-status workflow should be valid",
		},
		{
			name:        "in_refinement rejected",
			status:      "in_refinement",
			shouldPass:  true, // We WANT this to pass, but it will FAIL with hardcoded validation
			description: "Status from 14-status workflow should be valid",
		},
		{
			name:        "ready_for_development rejected",
			status:      "ready_for_development",
			shouldPass:  true, // We WANT this to pass, but it will FAIL with hardcoded validation
			description: "Status from 14-status workflow should be valid",
		},
		{
			name:        "in_code_review rejected",
			status:      "in_code_review",
			shouldPass:  true, // We WANT this to pass, but it will FAIL with hardcoded validation
			description: "Status from 14-status workflow should be valid",
		},
		{
			name:        "in_qa rejected",
			status:      "in_qa",
			shouldPass:  true, // We WANT this to pass, but it will FAIL with hardcoded validation
			description: "Status from 14-status workflow should be valid",
		},
		{
			name:        "hardcoded status accepted",
			status:      "todo",
			shouldPass:  true,
			description: "Old hardcoded status should still work",
		},
		{
			name:        "invalid status rejected",
			status:      "invalid_status",
			shouldPass:  false,
			description: "Truly invalid status should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskStatus(tt.status)

			if tt.shouldPass && err != nil {
				t.Errorf("EXPECTED FAILURE (demonstrates hardcoded limitation): %s\n"+
					"Status '%s' should be valid but was rejected: %v\n"+
					"This test shows that hardcoded validation prevents using new workflow statuses.",
					tt.description, tt.status, err)
			}

			if !tt.shouldPass && err == nil {
				t.Errorf("Status '%s' should be rejected but was accepted", tt.status)
			}
		})
	}
}

// TestValidateTaskStatus_ConfigDrivenDesired shows the DESIRED behavior after refactoring.
// This test documents what we want to achieve: validation based on workflow config.
func TestValidateTaskStatus_ConfigDrivenDesired(t *testing.T) {
	t.Skip("This test documents desired behavior - will be implemented after refactoring")

	// DESIRED: ValidateTaskStatus should accept a workflow config parameter
	// Example of desired API:
	//
	// workflow := &config.WorkflowConfig{
	//     StatusFlow: map[string][]string{
	//         "draft": {"ready_for_refinement"},
	//         "ready_for_refinement": {"in_refinement"},
	//         "in_refinement": {"ready_for_development"},
	//         "ready_for_development": {"in_development"},
	//         "in_development": {"ready_for_code_review"},
	//         "ready_for_code_review": {"in_code_review"},
	//         "in_code_review": {"ready_for_qa"},
	//         "ready_for_qa": {"in_qa"},
	//         "in_qa": {"ready_for_approval"},
	//         "ready_for_approval": {"in_approval"},
	//         "in_approval": {"completed"},
	//         "completed": {},
	//     },
	// }
	//
	// err := ValidateTaskStatusWithWorkflow("in_development", workflow)
	// if err != nil {
	//     t.Errorf("Status 'in_development' should be valid in custom workflow: %v", err)
	// }
	//
	// err = ValidateTaskStatusWithWorkflow("invalid_status", workflow)
	// if err == nil {
	//     t.Error("Status 'invalid_status' should be rejected")
	// }
}

// TestTaskStatusConstants_ShouldBeDeprecated documents that TaskStatus constants should be removed.
// These constants create a false sense that only these 6 statuses are valid.
func TestTaskStatusConstants_ShouldBeDeprecated(t *testing.T) {
	// Document the problem: these constants suggest a fixed set of statuses
	hardcodedStatuses := []TaskStatus{
		TaskStatusTodo,
		TaskStatusInProgress,
		TaskStatusBlocked,
		TaskStatusReadyForReview,
		TaskStatusCompleted,
		TaskStatusArchived,
	}

	t.Logf("DESIGN ISSUE: Found %d hardcoded TaskStatus constants", len(hardcodedStatuses))
	t.Logf("These constants should be deprecated in favor of config-driven status definitions")
	t.Logf("Problem: Code using these constants assumes only these 6 statuses exist")
	t.Logf("Solution: Replace with workflow config lookups")

	for _, status := range hardcodedStatuses {
		t.Logf("  - TaskStatus%s = %q",
			// Convert status to proper case name
			map[TaskStatus]string{
				TaskStatusTodo:           "Todo",
				TaskStatusInProgress:     "InProgress",
				TaskStatusBlocked:        "Blocked",
				TaskStatusReadyForReview: "ReadyForReview",
				TaskStatusCompleted:      "Completed",
				TaskStatusArchived:       "Archived",
			}[status],
			status,
		)
	}
}

// TestErrorMessage_ConfigDrivenStatusList verifies error messages are now config-driven
func TestErrorMessage_ConfigDrivenStatusList(t *testing.T) {
	err := ValidateTaskStatus("custom_status")

	if err == nil {
		t.Fatal("Expected error for invalid status")
	}

	// New error message should mention workflow config, not hardcoded list
	errMsg := err.Error()
	if !strings.Contains(errMsg, "workflow") {
		t.Errorf("Error message should mention workflow config.\nGot: %v", err)
	}

	t.Logf("SUCCESS: Error message is now config-driven: %v", err)
	t.Logf("Error message correctly guides users to check workflow configuration")
}
