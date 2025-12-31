package validation

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestStatusValidator_ValidateStatus(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":                 {"ready_for_refinement"},
			"ready_for_refinement":  {"in_refinement"},
			"in_refinement":         {"ready_for_development"},
			"ready_for_development": {"in_development"},
			"in_development":        {"ready_for_code_review", "blocked"},
			"ready_for_code_review": {"in_code_review"},
			"in_code_review":        {"ready_for_qa", "in_development"},
			"ready_for_qa":          {"in_qa"},
			"in_qa":                 {"ready_for_approval", "in_development", "blocked"},
			"ready_for_approval":    {"in_approval"},
			"in_approval":           {"completed", "ready_for_qa"},
			"blocked":               {"ready_for_development"},
			"completed":             {},
		},
	}

	validator := NewStatusValidator(workflow)

	tests := []struct {
		name      string
		status    string
		wantError bool
	}{
		{"valid status draft", "draft", false},
		{"valid status in_development", "in_development", false},
		{"valid status completed", "completed", false},
		{"invalid status", "invalid_status", true},
		{"empty status", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStatus(tt.status)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStatus(%q) error = %v, wantError %v", tt.status, err, tt.wantError)
			}
		})
	}
}

func TestStatusValidator_ValidateTransition(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":                 {"ready_for_refinement"},
			"ready_for_refinement":  {"in_refinement"},
			"in_refinement":         {"ready_for_development"},
			"ready_for_development": {"in_development"},
			"in_development":        {"ready_for_code_review", "blocked"},
			"ready_for_code_review": {"in_code_review"},
			"in_code_review":        {"ready_for_qa", "in_development"},
			"ready_for_qa":          {"in_qa"},
			"in_qa":                 {"ready_for_approval"},
			"ready_for_approval":    {"in_approval"},
			"in_approval":           {"completed"},
			"blocked":               {"in_development"},
			"completed":             {},
		},
	}

	validator := NewStatusValidator(workflow)

	tests := []struct {
		name       string
		fromStatus string
		toStatus   string
		wantError  bool
	}{
		{"valid transition", "in_development", "ready_for_code_review", false},
		{"valid transition to blocked", "in_development", "blocked", false},
		{"valid backward transition", "in_code_review", "in_development", false},
		{"invalid transition", "draft", "in_development", true},
		{"transition from terminal status", "completed", "in_development", true},
		{"self transition", "draft", "draft", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTransition(tt.fromStatus, tt.toStatus)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTransition(%q, %q) error = %v, wantError %v",
					tt.fromStatus, tt.toStatus, err, tt.wantError)
			}
		})
	}
}

func TestStatusValidator_CanTransition(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"in_development":        {"ready_for_code_review", "blocked"},
			"ready_for_code_review": {"in_code_review"},
			"in_code_review":        {"ready_for_qa", "in_development"},
			"ready_for_qa":          {"in_qa"},
			"in_qa":                 {"ready_for_approval"},
			"ready_for_approval":    {"in_approval"},
			"in_approval":           {"completed"},
			"blocked":               {"in_development"},
			"completed":             {},
		},
	}

	validator := NewStatusValidator(workflow)

	if !validator.CanTransition("in_development", "ready_for_code_review") {
		t.Error("Should allow transition from in_development to ready_for_code_review")
	}

	if validator.CanTransition("in_development", "completed") {
		t.Error("Should not allow transition from in_development to completed")
	}
}

func TestStatusValidator_GetAllStatuses(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":          {"ready_for_refinement"},
			"in_development": {"ready_for_code_review"},
			"completed":      {},
		},
	}

	validator := NewStatusValidator(workflow)
	statuses := validator.GetAllStatuses()

	if len(statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(statuses))
	}

	// Check that statuses are sorted
	expected := []string{"completed", "draft", "in_development"}
	for i, status := range statuses {
		if status != expected[i] {
			t.Errorf("Status at index %d: got %q, want %q", i, status, expected[i])
		}
	}
}

func TestStatusValidator_GetStartStatuses(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":                 {"ready_for_refinement"},
			"ready_for_development": {"in_development"},
		},
		SpecialStatuses: map[string][]string{
			"_start_": {"draft", "ready_for_development"},
		},
	}

	validator := NewStatusValidator(workflow)
	startStatuses := validator.GetStartStatuses()

	if len(startStatuses) != 2 {
		t.Errorf("Expected 2 start statuses, got %d", len(startStatuses))
	}

	if !validator.IsStartStatus("draft") {
		t.Error("draft should be a start status")
	}

	if !validator.IsStartStatus("ready_for_development") {
		t.Error("ready_for_development should be a start status")
	}

	if validator.IsStartStatus("in_development") {
		t.Error("in_development should not be a start status")
	}
}

func TestStatusValidator_GetCompleteStatuses(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"in_approval": {"completed", "cancelled"},
			"completed":   {},
			"cancelled":   {},
		},
		SpecialStatuses: map[string][]string{
			"_complete_": {"completed", "cancelled"},
		},
	}

	validator := NewStatusValidator(workflow)
	completeStatuses := validator.GetCompleteStatuses()

	if len(completeStatuses) != 2 {
		t.Errorf("Expected 2 complete statuses, got %d", len(completeStatuses))
	}

	if !validator.IsCompleteStatus("completed") {
		t.Error("completed should be a complete status")
	}

	if !validator.IsCompleteStatus("cancelled") {
		t.Error("cancelled should be a complete status")
	}

	if validator.IsCompleteStatus("in_approval") {
		t.Error("in_approval should not be a complete status")
	}
}

func TestStatusValidator_GetAllowedTransitions(t *testing.T) {
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"in_development": {"ready_for_code_review", "blocked", "ready_for_refinement"},
		},
	}

	validator := NewStatusValidator(workflow)
	transitions := validator.GetAllowedTransitions("in_development")

	if len(transitions) != 3 {
		t.Errorf("Expected 3 allowed transitions, got %d", len(transitions))
	}

	// Verify all expected transitions are present
	expectedTransitions := map[string]bool{
		"ready_for_code_review": true,
		"blocked":               true,
		"ready_for_refinement":  true,
	}

	for _, transition := range transitions {
		if !expectedTransitions[transition] {
			t.Errorf("Unexpected transition: %s", transition)
		}
	}
}

func TestStatusValidator_DefaultWorkflow(t *testing.T) {
	// Test that validator works with default workflow when nil is passed
	validator := NewStatusValidator(nil)

	// Default workflow should have at least the basic statuses
	statuses := validator.GetAllStatuses()
	if len(statuses) == 0 {
		t.Error("Default workflow should have statuses defined")
	}

	// Verify default workflow is usable - default has "todo", not "draft"
	if err := validator.ValidateStatus("todo"); err != nil {
		t.Errorf("Default workflow should support 'todo' status: %v", err)
	}

	// Verify default workflow supports in_progress
	if err := validator.ValidateStatus("in_progress"); err != nil {
		t.Errorf("Default workflow should support 'in_progress' status: %v", err)
	}

	// Verify default workflow supports completed
	if err := validator.ValidateStatus("completed"); err != nil {
		t.Errorf("Default workflow should support 'completed' status: %v", err)
	}
}

func TestStatusValidator_NilWorkflowHandling(t *testing.T) {
	// Create validator with explicit nil config
	validator := &StatusValidator{workflow: nil}

	// Should handle nil gracefully in all methods
	err := validator.ValidateStatus("draft")
	if err == nil {
		t.Error("Expected error when workflow is nil")
	}

	if validator.IsValidStatus("draft") {
		t.Error("Should return false when workflow is nil")
	}

	if len(validator.GetAllStatuses()) != 0 {
		t.Error("Should return empty slice when workflow is nil")
	}
}

func TestStatusValidator_ValidateTransition_UnrecognizedCurrentStatus(t *testing.T) {
	// Test that when current status is not in workflow config,
	// error message explains how to list workflow statuses and use --force
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"todo":        {"in_progress"},
			"in_progress": {"completed"},
			"completed":   {},
		},
	}

	validator := NewStatusValidator(workflow)

	// Try to transition from a status that doesn't exist in the workflow
	// This simulates a task with an old status after workflow config changed
	err := validator.ValidateTransition("old_status", "todo")

	if err == nil {
		t.Fatal("Expected error when transitioning from unrecognized status")
	}

	errMsg := err.Error()

	// Error message should contain helpful information
	expectedPhrases := []string{
		"current status",      // Explain it's the current status that's the problem
		"not recognized",      // Clear about what the issue is
		"shark workflow list", // How to see available statuses
		"--force",             // How to bypass the check
		"old_status",          // Show the actual unrecognized status
	}

	for _, phrase := range expectedPhrases {
		if !containsIgnoreCase(errMsg, phrase) {
			t.Errorf("Error message missing expected phrase %q.\nGot: %s", phrase, errMsg)
		}
	}
}

func TestStatusValidator_ValidateTransition_UnrecognizedTargetStatus(t *testing.T) {
	// Test that when target status is not in workflow config, error message is clear
	workflow := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"todo":        {"in_progress"},
			"in_progress": {"completed"},
			"completed":   {},
		},
	}

	validator := NewStatusValidator(workflow)

	// Try to transition to a status that doesn't exist in the workflow
	err := validator.ValidateTransition("todo", "invalid_target")

	if err == nil {
		t.Fatal("Expected error when transitioning to unrecognized status")
	}

	errMsg := err.Error()

	// Error should mention it's an invalid "to status" (wrapped error from ValidateStatus)
	if !strings.Contains(errMsg, "invalid to status") {
		t.Errorf("Error message should mention 'invalid to status'.\nGot: %s", errMsg)
	}
}

// Helper function for case-insensitive substring check
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
