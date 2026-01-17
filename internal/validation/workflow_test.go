package validation

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestIsBackwardTransition tests the backward transition detection logic
func TestIsBackwardTransition(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus string
		newStatus     string
		workflow      *config.WorkflowConfig
		expectedValue bool
		description   string
	}{
		// Forward transitions (should return false)
		{
			name:          "forward_planning_to_development",
			currentStatus: "draft",
			newStatus:     "in_development",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "planning → development is forward",
		},
		{
			name:          "forward_development_to_review",
			currentStatus: "in_development",
			newStatus:     "ready_for_code_review",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "development → review is forward",
		},
		{
			name:          "forward_review_to_qa",
			currentStatus: "ready_for_code_review",
			newStatus:     "in_qa",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "review → qa is forward",
		},
		{
			name:          "forward_qa_to_approval",
			currentStatus: "in_qa",
			newStatus:     "ready_for_approval",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "qa → approval is forward",
		},
		{
			name:          "forward_approval_to_done",
			currentStatus: "ready_for_approval",
			newStatus:     "completed",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "approval → done is forward",
		},

		// Backward transitions (should return true)
		{
			name:          "backward_development_to_planning",
			currentStatus: "in_development",
			newStatus:     "draft",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "development → planning is backward (rejection)",
		},
		{
			name:          "backward_review_to_development",
			currentStatus: "ready_for_code_review",
			newStatus:     "in_development",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "review → development is backward (rejection)",
		},
		{
			name:          "backward_qa_to_review",
			currentStatus: "in_qa",
			newStatus:     "ready_for_code_review",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "qa → review is backward (rejection)",
		},
		{
			name:          "backward_approval_to_qa",
			currentStatus: "ready_for_approval",
			newStatus:     "in_qa",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "approval → qa is backward (rejection)",
		},
		{
			name:          "backward_done_to_approval",
			currentStatus: "completed",
			newStatus:     "ready_for_approval",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "done → approval is backward",
		},
		{
			name:          "backward_done_to_development",
			currentStatus: "completed",
			newStatus:     "in_development",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "done → development is very backward",
		},

		// Same-phase transitions (should return false)
		{
			name:          "same_phase_planning",
			currentStatus: "draft",
			newStatus:     "draft",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "same phase = same status is not backward",
		},
		{
			name:          "same_phase_development",
			currentStatus: "in_development",
			newStatus:     "in_development",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "same phase = same status is not backward",
		},
		{
			name:          "same_phase_review",
			currentStatus: "ready_for_code_review",
			newStatus:     "ready_for_code_review",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "same phase = same status is not backward",
		},

		// Special statuses (any phase) - these should return false (ignore)
		{
			name:          "special_blocked_to_planning",
			currentStatus: "blocked",
			newStatus:     "draft",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "blocked (any phase) → planning ignores 'any' phase",
		},
		{
			name:          "special_development_to_blocked",
			currentStatus: "in_development",
			newStatus:     "blocked",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "development → blocked (any phase) ignores 'any' phase",
		},
		{
			name:          "special_blocked_to_blocked",
			currentStatus: "blocked",
			newStatus:     "blocked",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "blocked → blocked is not backward",
		},
		{
			name:          "special_blocked_to_review",
			currentStatus: "blocked",
			newStatus:     "ready_for_code_review",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "blocked (any) → review ignores 'any' phase",
		},

		// Multiple phase jumps
		{
			name:          "multi_jump_backward_qa_to_planning",
			currentStatus: "in_qa",
			newStatus:     "draft",
			workflow:      getTestWorkflow(),
			expectedValue: true,
			description:   "qa → planning skips multiple phases backward",
		},
		{
			name:          "multi_jump_forward_planning_to_qa",
			currentStatus: "draft",
			newStatus:     "in_qa",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "planning → qa skips multiple phases forward",
		},

		// Edge cases with missing metadata
		{
			name:          "missing_metadata_assumes_no_phase",
			currentStatus: "unknown_status_1",
			newStatus:     "unknown_status_2",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "missing metadata treated as no phase (0)",
		},

		// Cancelled/deleted status should be treated carefully
		{
			name:          "to_cancelled_not_backward",
			currentStatus: "in_development",
			newStatus:     "cancelled",
			workflow:      getTestWorkflow(),
			expectedValue: false,
			description:   "transition to cancelled is not a rejection",
		},

		// On-hold transitions
		{
			name:          "on_hold_transition",
			currentStatus: "in_development",
			newStatus:     "on_hold",
			workflow:      getTestWorkflowWithOnHold(),
			expectedValue: false,
			description:   "on_hold is special phase, not backward",
		},
		{
			name:          "on_hold_to_development",
			currentStatus: "on_hold",
			newStatus:     "in_development",
			workflow:      getTestWorkflowWithOnHold(),
			expectedValue: false,
			description:   "on_hold to development ignores any phase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBackwardTransition(tt.currentStatus, tt.newStatus, tt.workflow)
			if result != tt.expectedValue {
				t.Errorf("IsBackwardTransition(%q, %q) = %v, expected %v\n%s",
					tt.currentStatus, tt.newStatus, result, tt.expectedValue, tt.description)
			}
		})
	}
}

// TestIsBackwardTransitionWithNilWorkflow tests handling of nil workflow
func TestIsBackwardTransitionWithNilWorkflow(t *testing.T) {
	result := IsBackwardTransition("in_development", "draft", nil)
	if result != false {
		t.Errorf("IsBackwardTransition with nil workflow should return false, got %v", result)
	}
}

// TestIsBackwardTransitionWithEmptyWorkflow tests handling of empty workflow
func TestIsBackwardTransitionWithEmptyWorkflow(t *testing.T) {
	emptyWorkflow := &config.WorkflowConfig{
		StatusMetadata: make(map[string]config.StatusMetadata),
	}
	result := IsBackwardTransition("in_development", "draft", emptyWorkflow)
	if result != false {
		t.Errorf("IsBackwardTransition with empty workflow should return false, got %v", result)
	}
}

// TestIsBackwardTransitionPhaseOrdering tests all phase orderings
func TestIsBackwardTransitionPhaseOrdering(t *testing.T) {
	workflow := getTestWorkflow()

	// Define phase order
	phaseOrder := map[string]int{
		"planning":     1,
		"development":  2,
		"review":       3,
		"qa":           4,
		"approval":     5,
		"done":         6,
		"any":          0,
		"cancelled":    0, // terminal status like any
		"on_hold":      0, // special status
	}

	// Get all statuses from workflow
	allStatuses := make([]string, 0)
	for status := range workflow.StatusMetadata {
		allStatuses = append(allStatuses, status)
	}

	// For each pair, verify phase ordering
	for _, fromStatus := range allStatuses {
		for _, toStatus := range allStatuses {
			if fromStatus == toStatus {
				continue // skip same status
			}

			fromMeta, fromExists := workflow.GetStatusMetadata(fromStatus)
			toMeta, toExists := workflow.GetStatusMetadata(toStatus)

			// Default phase to empty string if not found
			fromPhase := ""
			toPhase := ""
			if fromExists {
				fromPhase = fromMeta.Phase
			}
			if toExists {
				toPhase = toMeta.Phase
			}

			fromOrder, fromHasPhase := phaseOrder[fromPhase]
			toOrder, toHasPhase := phaseOrder[toPhase]

			// If either phase is unknown or "any", it's not a backward transition
			if !fromHasPhase || !toHasPhase {
				fromOrder = 0
				toOrder = 0
			}

			result := IsBackwardTransition(fromStatus, toStatus, workflow)

			// Expected: backward if toOrder < fromOrder AND both > 0
			expected := toOrder < fromOrder && toOrder > 0 && fromOrder > 0

			if result != expected {
				t.Errorf(
					"IsBackwardTransition(%q[phase=%s,order=%d], %q[phase=%s,order=%d]) = %v, expected %v",
					fromStatus, fromPhase, fromOrder,
					toStatus, toPhase, toOrder,
					result, expected,
				)
			}
		}
	}
}

// TestIsBackwardTransitionRealisticScenarios tests real-world rejection scenarios
func TestIsBackwardTransitionRealisticScenarios(t *testing.T) {
	workflow := getTestWorkflow()

	tests := []struct {
		name     string
		from     string
		to       string
		backward bool
		scenario string
	}{
		{
			name:     "code_review_rejection",
			from:     "ready_for_code_review",
			to:       "in_development",
			backward: true,
			scenario: "Reviewer rejects code, sends back to development",
		},
		{
			name:     "qa_rejection",
			from:     "in_qa",
			to:       "in_development",
			backward: true,
			scenario: "QA finds bugs, sends back to development",
		},
		{
			name:     "approval_rejection",
			from:     "ready_for_approval",
			to:       "in_development",
			backward: true,
			scenario: "Approver rejects, sends back to development",
		},
		{
			name:     "developer_submitted_for_review",
			from:     "in_development",
			to:       "ready_for_code_review",
			backward: false,
			scenario: "Developer submits code for review",
		},
		{
			name:     "reviewer_approved",
			from:     "ready_for_code_review",
			to:       "in_qa",
			backward: false,
			scenario: "Reviewer approves, sends to QA",
		},
		{
			name:     "qa_approved",
			from:     "in_qa",
			to:       "ready_for_approval",
			backward: false,
			scenario: "QA approves, sends to approval",
		},
		{
			name:     "blocker_encountered",
			from:     "in_development",
			to:       "blocked",
			backward: false,
			scenario: "Developer encounters blocker",
		},
		{
			name:     "blocker_resolved",
			from:     "blocked",
			to:       "in_development",
			backward: false,
			scenario: "Blocker resolved, development resumes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBackwardTransition(tt.from, tt.to, workflow)
			if result != tt.backward {
				t.Errorf(
					"Scenario: %s\nIsBackwardTransition(%q, %q) = %v, expected %v",
					tt.scenario, tt.from, tt.to, result, tt.backward,
				)
			}
		})
	}
}

// Helper function to create a test workflow
func getTestWorkflow() *config.WorkflowConfig {
	return &config.WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"draft":                    {"in_development", "blocked", "cancelled"},
			"in_development":           {"ready_for_code_review", "blocked", "cancelled"},
			"ready_for_code_review":    {"in_qa", "in_development", "blocked", "cancelled"},
			"in_qa":                    {"ready_for_approval", "in_development", "blocked", "cancelled"},
			"ready_for_approval":       {"completed", "in_development", "blocked", "cancelled"},
			"completed":                {},
			"blocked":                  {"draft", "in_development", "ready_for_code_review", "in_qa", "ready_for_approval"},
			"cancelled":                {},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"draft": {
				Color:       "gray",
				Phase:       "planning",
				Description: "Task draft",
			},
			"in_development": {
				Color:       "yellow",
				Phase:       "development",
				Description: "In development",
			},
			"ready_for_code_review": {
				Color:       "magenta",
				Phase:       "review",
				Description: "Ready for review",
			},
			"in_qa": {
				Color:       "cyan",
				Phase:       "qa",
				Description: "In QA",
			},
			"ready_for_approval": {
				Color:       "purple",
				Phase:       "approval",
				Description: "Ready for approval",
			},
			"completed": {
				Color:       "green",
				Phase:       "done",
				Description: "Completed",
			},
			"blocked": {
				Color:       "red",
				Phase:       "any",
				Description: "Blocked",
			},
			"cancelled": {
				Color:       "gray",
				Phase:       "any",
				Description: "Cancelled",
			},
		},
		SpecialStatuses: map[string][]string{
			"_start_":    {"draft"},
			"_complete_": {"completed", "cancelled"},
		},
	}
}

// Helper function to create a test workflow with on_hold status
func getTestWorkflowWithOnHold() *config.WorkflowConfig {
	workflow := getTestWorkflow()
	workflow.StatusMetadata["on_hold"] = config.StatusMetadata{
		Color:       "orange",
		Phase:       "any",
		Description: "On hold",
	}
	workflow.StatusFlow["on_hold"] = []string{"draft", "in_development", "ready_for_code_review"}
	workflow.StatusFlow["in_development"] = append(workflow.StatusFlow["in_development"], "on_hold")
	return workflow
}

// TestValidateReasonForStatusTransition tests the reason validation for status transitions
func TestValidateReasonForStatusTransition(t *testing.T) {
	workflow := getTestWorkflow()

	tests := []struct {
		name        string
		newStatus   string
		currentStatus string
		reason      string
		force       bool
		shouldErr   bool
	}{
		{
			name:         "no_status_change",
			newStatus:    "",
			currentStatus: "in_development",
			reason:       "",
			force:        false,
			shouldErr:    false,
		},
		{
			name:         "backward_without_reason",
			newStatus:    "in_development",
			currentStatus: "ready_for_code_review",
			reason:       "",
			force:        false,
			shouldErr:    true,
		},
		{
			name:         "backward_with_reason",
			newStatus:    "in_development",
			currentStatus: "ready_for_code_review",
			reason:       "Need to fix bugs",
			force:        false,
			shouldErr:    false,
		},
		{
			name:         "backward_with_force",
			newStatus:    "in_development",
			currentStatus: "ready_for_code_review",
			reason:       "",
			force:        true,
			shouldErr:    false,
		},
		{
			name:         "forward_without_reason",
			newStatus:    "ready_for_code_review",
			currentStatus: "in_development",
			reason:       "",
			force:        false,
			shouldErr:    false,
		},
		{
			name:         "same_phase_without_reason",
			newStatus:    "blocked",
			currentStatus: "in_development",
			reason:       "",
			force:        false,
			shouldErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReasonForStatusTransition(tt.newStatus, tt.currentStatus, tt.reason, tt.force, workflow)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateReasonForStatusTransition() error = %v, shouldErr %v", err, tt.shouldErr)
			}
			if tt.shouldErr && err != nil && err != ErrReasonRequired {
				t.Errorf("ValidateReasonForStatusTransition() got error %v, expected ErrReasonRequired", err)
			}
		})
	}
}
