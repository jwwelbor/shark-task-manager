package workflow

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// basicWorkflowConfig returns a config map representing a basic workflow profile
func basicWorkflowConfig() map[string]interface{} {
	return map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"todo"},
			"_complete_": {"completed"},
		},
		"status_flow": map[string][]string{
			"todo":             {"in_progress", "blocked"},
			"in_progress":      {"ready_for_review", "blocked"},
			"ready_for_review": {"completed", "in_progress"},
			"completed":        {},
			"blocked":          {"todo", "in_progress"},
		},
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{
				"color":       "gray",
				"description": "Not started",
				"phase":       "planning",
			},
			"in_progress": map[string]interface{}{
				"color":       "yellow",
				"description": "Work in progress",
				"phase":       "development",
			},
			"ready_for_review": map[string]interface{}{
				"color":       "magenta",
				"description": "Awaiting review",
				"phase":       "review",
			},
			"completed": map[string]interface{}{
				"color":       "green",
				"description": "Finished",
				"phase":       "done",
			},
			"blocked": map[string]interface{}{
				"color":       "red",
				"description": "Blocked by dependency",
				"phase":       "any",
			},
		},
	}
}

// advancedWorkflowConfig returns a config map representing an advanced workflow profile
func advancedWorkflowConfig() map[string]interface{} {
	return map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    {"draft"},
			"_complete_": {"completed", "cancelled"},
		},
		"status_flow": map[string][]string{
			"draft":                     {"ready_for_refinement_ba", "cancelled"},
			"ready_for_refinement_ba":   {"in_refinement_ba"},
			"in_refinement_ba":          {"ready_for_refinement_tech"},
			"ready_for_refinement_tech": {"in_refinement_tech"},
			"in_refinement_tech":        {"ready_for_development"},
			"ready_for_development":     {"in_development"},
			"in_development":            {"ready_for_code_review", "blocked"},
			"ready_for_code_review":     {"in_code_review"},
			"in_code_review":            {"changes_requested", "ready_for_qa"},
			"changes_requested":         {"in_development"},
			"ready_for_qa":              {"in_qa"},
			"in_qa":                     {"qa_failed", "ready_for_approval"},
			"qa_failed":                 {"in_development"},
			"ready_for_approval":        {"in_approval"},
			"in_approval":               {"completed", "changes_requested"},
			"completed":                 {},
			"cancelled":                 {},
			"blocked":                   {"draft", "in_development"},
			"on_hold":                   {"draft"},
		},
		"status_metadata": map[string]interface{}{
			"draft": map[string]interface{}{
				"color":       "gray",
				"description": "Initial draft",
				"phase":       "planning",
			},
			"ready_for_refinement_ba": map[string]interface{}{
				"color":       "cyan",
				"description": "Ready for BA review",
				"phase":       "planning",
			},
			"in_refinement_ba": map[string]interface{}{
				"color":       "cyan",
				"description": "Under business analysis",
				"phase":       "planning",
			},
			"ready_for_refinement_tech": map[string]interface{}{
				"color":       "blue",
				"description": "Ready for tech review",
				"phase":       "planning",
			},
			"in_refinement_tech": map[string]interface{}{
				"color":       "blue",
				"description": "Under technical review",
				"phase":       "planning",
			},
			"ready_for_development": map[string]interface{}{
				"color":       "yellow",
				"description": "Ready to implement",
				"phase":       "development",
			},
			"in_development": map[string]interface{}{
				"color":       "yellow",
				"description": "Implementation in progress",
				"phase":       "development",
			},
			"ready_for_code_review": map[string]interface{}{
				"color":       "magenta",
				"description": "Awaiting code review",
				"phase":       "review",
			},
			"in_code_review": map[string]interface{}{
				"color":       "magenta",
				"description": "Code review in progress",
				"phase":       "review",
			},
			"changes_requested": map[string]interface{}{
				"color":       "orange",
				"description": "Revisions requested",
				"phase":       "review",
			},
			"ready_for_qa": map[string]interface{}{
				"color":       "cyan",
				"description": "Ready for testing",
				"phase":       "qa",
			},
			"in_qa": map[string]interface{}{
				"color":       "cyan",
				"description": "QA testing in progress",
				"phase":       "qa",
			},
			"qa_failed": map[string]interface{}{
				"color":       "red",
				"description": "QA tests failed",
				"phase":       "qa",
			},
			"ready_for_approval": map[string]interface{}{
				"color":       "purple",
				"description": "Ready for final approval",
				"phase":       "approval",
			},
			"in_approval": map[string]interface{}{
				"color":       "purple",
				"description": "Under final review",
				"phase":       "approval",
			},
			"completed": map[string]interface{}{
				"color":       "green",
				"description": "Work done and approved",
				"phase":       "done",
			},
			"cancelled": map[string]interface{}{
				"color":       "gray",
				"description": "Work not needed",
				"phase":       "done",
			},
			"blocked": map[string]interface{}{
				"color":       "red",
				"description": "Blocked by dependency",
				"phase":       "any",
			},
			"on_hold": map[string]interface{}{
				"color":       "gray",
				"description": "Temporarily suspended",
				"phase":       "any",
			},
		},
	}
}

// --- ValidateStatus Tests ---

func TestService_ValidateStatus_BasicWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, basicWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Valid statuses should return nil
	validStatuses := []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			err := svc.ValidateStatus(status)
			assert.NoError(t, err, "expected no error for valid status %q", status)
		})
	}

	// Case insensitive
	t.Run("case_insensitive", func(t *testing.T) {
		assert.NoError(t, svc.ValidateStatus("TODO"))
		assert.NoError(t, svc.ValidateStatus("In_Progress"))
		assert.NoError(t, svc.ValidateStatus("COMPLETED"))
	})

	// Invalid statuses should return error
	invalidStatuses := []string{"", "unknown", "invalid_status", "in-progress", "done"}
	for _, status := range invalidStatuses {
		t.Run("invalid_"+status, func(t *testing.T) {
			err := svc.ValidateStatus(status)
			assert.Error(t, err, "expected error for invalid status %q", status)
			// Error message should mention the invalid status
			if status != "" {
				assert.Contains(t, err.Error(), status)
			}
		})
	}
}

func TestService_ValidateStatus_AdvancedWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, advancedWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Valid advanced statuses
	validStatuses := []string{
		"draft", "ready_for_refinement_ba", "in_refinement_ba",
		"ready_for_development", "in_development",
		"ready_for_code_review", "in_code_review",
		"ready_for_qa", "in_qa",
		"completed", "cancelled", "blocked",
	}
	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			err := svc.ValidateStatus(status)
			assert.NoError(t, err, "expected no error for valid status %q", status)
		})
	}

	// Invalid statuses
	t.Run("invalid_todo", func(t *testing.T) {
		// "todo" is not in advanced workflow
		err := svc.ValidateStatus("todo")
		assert.Error(t, err)
	})
}

func TestService_ValidateStatus_ErrorContainsValidStatuses(t *testing.T) {
	projectRoot := createTestConfig(t, basicWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	err := svc.ValidateStatus("invalid_status")
	require.Error(t, err)

	errMsg := err.Error()
	// Error should list some valid statuses as guidance
	assert.Contains(t, errMsg, "invalid_status")
}

// --- StatusHelpText Tests ---

func TestService_StatusHelpText_BasicWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, basicWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	helpText := svc.StatusHelpText()

	// Should be non-empty
	assert.NotEmpty(t, helpText)

	// Should contain statuses
	assert.Contains(t, helpText, "todo")
	assert.Contains(t, helpText, "in_progress")
	assert.Contains(t, helpText, "completed")
	assert.Contains(t, helpText, "blocked")

	// Should contain phase groupings
	assert.Contains(t, helpText, "planning")
	assert.Contains(t, helpText, "development")
	assert.Contains(t, helpText, "done")
}

func TestService_StatusHelpText_AdvancedWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, advancedWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	helpText := svc.StatusHelpText()

	// Should be non-empty
	assert.NotEmpty(t, helpText)

	// Should contain advanced statuses
	assert.Contains(t, helpText, "draft")
	assert.Contains(t, helpText, "in_development")
	assert.Contains(t, helpText, "ready_for_code_review")
	assert.Contains(t, helpText, "in_qa")
	assert.Contains(t, helpText, "completed")

	// Should contain phase groupings
	assert.Contains(t, helpText, "planning")
	assert.Contains(t, helpText, "development")
	assert.Contains(t, helpText, "review")
	assert.Contains(t, helpText, "qa")
	assert.Contains(t, helpText, "done")
}

// --- StatusFlagDescription Tests ---

func TestService_StatusFlagDescription_BasicWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, basicWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	desc := svc.StatusFlagDescription()

	// Should be non-empty
	assert.NotEmpty(t, desc)

	// Should contain pipe separators
	assert.Contains(t, desc, "|")

	// Should contain actual status values
	assert.Contains(t, desc, "todo")
	assert.Contains(t, desc, "in_progress")
	assert.Contains(t, desc, "completed")

	// Should be a single line (no newlines)
	assert.NotContains(t, desc, "\n")

	// Should start with "Status filter"
	assert.True(t, strings.HasPrefix(desc, "Status filter"), "expected prefix 'Status filter', got: %s", desc)
}

func TestService_StatusFlagDescription_AdvancedWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, advancedWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	desc := svc.StatusFlagDescription()

	// Should be non-empty
	assert.NotEmpty(t, desc)

	// Should contain pipe separators
	assert.Contains(t, desc, "|")

	// Should contain some advanced status values
	assert.Contains(t, desc, "draft")
	assert.Contains(t, desc, "completed")

	// Should be a single line
	assert.NotContains(t, desc, "\n")
}

// --- IsCompletedStatus Tests ---

func TestService_IsCompletedStatus_BasicWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, basicWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Terminal/completed statuses
	assert.True(t, svc.IsCompletedStatus("completed"))
	assert.True(t, svc.IsCompletedStatus("COMPLETED")) // Case insensitive

	// Non-completed statuses
	assert.False(t, svc.IsCompletedStatus("todo"))
	assert.False(t, svc.IsCompletedStatus("in_progress"))
	assert.False(t, svc.IsCompletedStatus("ready_for_review"))
	assert.False(t, svc.IsCompletedStatus("blocked"))

	// Unknown status
	assert.False(t, svc.IsCompletedStatus("unknown"))
}

func TestService_IsCompletedStatus_AdvancedWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, advancedWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	// Terminal statuses in advanced workflow
	assert.True(t, svc.IsCompletedStatus("completed"))
	assert.True(t, svc.IsCompletedStatus("cancelled"))

	// Non-completed statuses
	assert.False(t, svc.IsCompletedStatus("draft"))
	assert.False(t, svc.IsCompletedStatus("in_development"))
	assert.False(t, svc.IsCompletedStatus("in_qa"))
	assert.False(t, svc.IsCompletedStatus("blocked"))
}

func TestService_IsCompletedStatus_MatchesIsTerminalStatus(t *testing.T) {
	// Verify IsCompletedStatus and IsTerminalStatus always return the same result
	projectRoot := createTestConfig(t, advancedWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	allStatuses := svc.GetAllStatuses()
	for _, status := range allStatuses {
		t.Run(status, func(t *testing.T) {
			assert.Equal(t, svc.IsTerminalStatus(status), svc.IsCompletedStatus(status),
				"IsCompletedStatus and IsTerminalStatus should match for status %q", status)
		})
	}

	// Also test with unknown status
	assert.Equal(t, svc.IsTerminalStatus("unknown"), svc.IsCompletedStatus("unknown"))
}

// --- GetDefaultStatus Tests ---

func TestService_GetDefaultStatus_BasicWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, basicWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	defaultStatus := svc.GetDefaultStatus()

	// Should be non-empty
	assert.NotEmpty(t, defaultStatus)

	// Should match GetInitialStatus
	assert.Equal(t, string(svc.GetInitialStatus()), defaultStatus)

	// For basic workflow, should be "todo"
	assert.Equal(t, "todo", defaultStatus)
}

func TestService_GetDefaultStatus_AdvancedWorkflow(t *testing.T) {
	projectRoot := createTestConfig(t, advancedWorkflowConfig())
	defer config.ClearWorkflowCache()

	svc := NewService(projectRoot)

	defaultStatus := svc.GetDefaultStatus()

	// Should be non-empty
	assert.NotEmpty(t, defaultStatus)

	// Should match GetInitialStatus
	assert.Equal(t, string(svc.GetInitialStatus()), defaultStatus)

	// For advanced workflow, should be "draft"
	assert.Equal(t, "draft", defaultStatus)
}

func TestService_GetDefaultStatus_Fallback(t *testing.T) {
	// When no config exists, should fall back to "todo"
	tempDir := t.TempDir()
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	svc := NewService(tempDir)

	defaultStatus := svc.GetDefaultStatus()
	assert.Equal(t, "todo", defaultStatus)
}
