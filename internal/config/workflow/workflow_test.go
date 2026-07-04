package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Test workflow schema defaults
func TestWorkflowConfigDefaults(t *testing.T) {
	workflow := &WorkflowConfig{}

	if workflow.Version != "" {
		t.Errorf("expected empty version, got %s", workflow.Version)
	}
}

// Test default workflow
//
// The task workflow is now route-based (loaded from the embedded
// workflow/task.yaml, see internal/config/workflow/steps.go
// buildWorkflowMapsFromSteps): step name = status name, "terminal" steps
// have no outgoing transitions, "parking" steps (blocked/on_hold) expose
// every workable step as a return target, and StatusFlow/StatusMetadata are
// derived from each step's `outcomes:` map. Statuses are draft/development/
// completed/blocked/on_hold/cancelled -- NOT the legacy
// todo/in_progress/ready_for_review/completed/blocked shape.
func TestDefaultWorkflow(t *testing.T) {
	workflow := DefaultWorkflow()

	// Check version
	if workflow.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", workflow.Version)
	}

	// Check all expected statuses exist
	expectedStatuses := []string{"draft", "development", "completed", "blocked", "on_hold", "cancelled"}
	for _, status := range expectedStatuses {
		if _, exists := workflow.StatusFlow[status]; !exists {
			t.Errorf("expected status %s to exist in default workflow", status)
		}
	}

	// Check transitions match current behavior (derived from task.yaml's
	// outcomes maps; parking steps expose all workable steps as return targets)
	testCases := []struct {
		from     string
		expected []string
	}{
		{"draft", []string{"development", "draft", "blocked", "cancelled", "on_hold"}},
		{"development", []string{"completed", "draft", "blocked", "on_hold"}},
		{"completed", []string{}},
		{"blocked", []string{"development", "draft"}},
		{"on_hold", []string{"development", "draft"}},
		{"cancelled", []string{}},
	}

	for _, tc := range testCases {
		transitions := workflow.StatusFlow[tc.from]
		if len(transitions) != len(tc.expected) {
			t.Errorf("status %s: expected %d transitions, got %d", tc.from, len(tc.expected), len(transitions))
			continue
		}

		for i, exp := range tc.expected {
			if transitions[i] != exp {
				t.Errorf("status %s: expected transition[%d] = %s, got %s", tc.from, i, exp, transitions[i])
			}
		}
	}

	// Check special statuses
	startStatuses := workflow.SpecialStatuses[StartStatusKey]
	if len(startStatuses) != 1 || startStatuses[0] != "draft" {
		t.Errorf("expected _start_ = [draft], got %v", startStatuses)
	}

	completeStatuses := workflow.SpecialStatuses[CompleteStatusKey]
	if len(completeStatuses) != 2 || completeStatuses[0] != "cancelled" || completeStatuses[1] != "completed" {
		t.Errorf("expected _complete_ = [cancelled completed], got %v", completeStatuses)
	}

	// Check metadata exists for all statuses
	for _, status := range expectedStatuses {
		if _, exists := workflow.StatusMetadata[status]; !exists {
			t.Errorf("expected metadata for status %s", status)
		}
	}
}

// Test default workflow is valid
func TestDefaultWorkflowIsValid(t *testing.T) {
	workflow := DefaultWorkflow()
	err := ValidateWorkflow(workflow)
	if err != nil {
		t.Errorf("default workflow should be valid, got error: %v", err)
	}
}

// Test workflow parser - missing file
func TestLoadWorkflowConfig_MissingFile(t *testing.T) {
	workflow, err := LoadWorkflowConfig("/nonexistent/config.json")
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
	if workflow != nil {
		t.Error("expected nil workflow for missing file")
	}
}

// Test workflow parser - valid config
func TestLoadWorkflowConfig_ValidConfig(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configContent := `{
		"status_flow_version": "1.0",
		"status_flow": {
			"todo": ["in_progress"],
			"in_progress": ["done"],
			"done": []
		},
		"status_metadata": {
			"todo": {
				"color": "gray",
				"description": "To do"
			}
		},
		"special_statuses": {
			"_start_": ["todo"],
			"_complete_": ["done"]
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Clear cache before test
	ClearWorkflowCache()

	workflow, err := LoadWorkflowConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if workflow == nil {
		t.Fatal("expected workflow, got nil")
	}

	// Check parsed values
	if workflow.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", workflow.Version)
	}

	if len(workflow.StatusFlow) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(workflow.StatusFlow))
	}

	if workflow.StatusFlow["todo"][0] != "in_progress" {
		t.Errorf("expected todo → in_progress transition")
	}
}

// Test workflow parser - invalid JSON
func TestLoadWorkflowConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	invalidJSON := `{"status_flow": {invalid json}}`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	ClearWorkflowCache()

	_, err := LoadWorkflowConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Test workflow parser - no status_flow section
func TestLoadWorkflowConfig_NoStatusFlow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configContent := `{"color_enabled": true}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	ClearWorkflowCache()

	workflow, err := LoadWorkflowConfig(configPath)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if workflow != nil {
		t.Error("expected nil workflow when status_flow missing")
	}
}

// Test workflow parser - unsupported version
func TestLoadWorkflowConfig_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configContent := `{
		"status_flow_version": "2.0",
		"status_flow": {
			"todo": ["done"],
			"done": []
		},
		"special_statuses": {
			"_start_": ["todo"],
			"_complete_": ["done"]
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	ClearWorkflowCache()

	_, err := LoadWorkflowConfig(configPath)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

// Test workflow parser - supported 1.x versions
func TestLoadWorkflowConfig_Supported1xVersions(t *testing.T) {
	versions := []string{"1.0", "1.1", "1.2", "1.99"}
	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".sharkconfig.json")

			configContent := fmt.Sprintf(`{
				"status_flow_version": %q,
				"status_flow": {
					"todo": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`, version)

			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			ClearWorkflowCache()

			wf, err := LoadWorkflowConfig(configPath)
			if err != nil {
				t.Errorf("version %s should be supported, got error: %v", version, err)
			}
			if wf != nil && wf.Version != version {
				t.Errorf("expected version %s, got %s", version, wf.Version)
			}
		})
	}
}

// Test GetWorkflowOrDefault
func TestGetWorkflowOrDefault(t *testing.T) {
	ClearWorkflowCache()

	// Should return default for missing file
	workflow := GetWorkflowOrDefault("/nonexistent/config.json")
	if workflow == nil {
		t.Fatal("expected default workflow, got nil")
	}

	// Verify it's the default workflow (route-based task.yaml has 6 steps:
	// draft/development/completed/blocked/on_hold/cancelled)
	if len(workflow.StatusFlow) != 6 {
		t.Errorf("expected 6 default statuses, got %d", len(workflow.StatusFlow))
	}
}

// Table-driven tests for workflow validation
func TestValidateWorkflow(t *testing.T) {
	testCases := []struct {
		name        string
		workflow    *WorkflowConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil workflow",
			workflow:    nil,
			expectError: true,
			errorMsg:    "workflow config is nil",
		},
		{
			name: "missing _start_ status",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo": {"done"},
					"done": {},
				},
				SpecialStatuses: map[string][]string{
					CompleteStatusKey: {"done"},
				},
			},
			expectError: true,
			errorMsg:    "_start_",
		},
		{
			name: "missing _complete_ status",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo": {"done"},
					"done": {},
				},
				SpecialStatuses: map[string][]string{
					StartStatusKey: {"todo"},
				},
			},
			expectError: true,
			errorMsg:    "_complete_",
		},
		{
			name: "undefined status reference",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo":        {"in_progress"},
					"in_progress": {"undefined_status"},
				},
				SpecialStatuses: map[string][]string{
					StartStatusKey:    {"todo"},
					CompleteStatusKey: {"in_progress"},
				},
			},
			expectError: true,
			errorMsg:    "undefined status",
		},
		{
			name: "unreachable status",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo":        {"done"},
					"done":        {},
					"unreachable": {"done"},
				},
				SpecialStatuses: map[string][]string{
					StartStatusKey:    {"todo"},
					CompleteStatusKey: {"done"},
				},
			},
			expectError: true,
			errorMsg:    "unreachable",
		},
		{
			name: "dead-end status",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo":     {"dead_end", "done"},
					"dead_end": {}, // Dead end - can't reach "done"
					"done":     {},
				},
				SpecialStatuses: map[string][]string{
					StartStatusKey:    {"todo"},
					CompleteStatusKey: {"done"},
				},
			},
			expectError: true,
			errorMsg:    "dead-end",
		},
		{
			name: "valid simple workflow",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo": {"done"},
					"done": {},
				},
				SpecialStatuses: map[string][]string{
					StartStatusKey:    {"todo"},
					CompleteStatusKey: {"done"},
				},
			},
			expectError: false,
		},
		{
			name: "valid complex workflow with cycles",
			workflow: &WorkflowConfig{
				Version: "1.0",
				StatusFlow: map[string][]string{
					"todo":        {"in_progress"},
					"in_progress": {"review", "todo"},      // Can go back to todo
					"review":      {"done", "in_progress"}, // Can go back to in_progress
					"done":        {},
				},
				SpecialStatuses: map[string][]string{
					StartStatusKey:    {"todo"},
					CompleteStatusKey: {"done"},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWorkflow(tc.workflow)

			if tc.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tc.errorMsg != "" && !workflowStringContains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// Test ValidateTransition
func TestValidateTransition(t *testing.T) {
	workflow := &WorkflowConfig{
		StatusFlow: map[string][]string{
			"todo":        {"in_progress", "blocked"},
			"in_progress": {"done"},
			"done":        {},
			"blocked":     {"todo"},
		},
	}

	testCases := []struct {
		name        string
		from        string
		to          string
		expectError bool
	}{
		{"valid transition", "todo", "in_progress", false},
		{"valid transition 2", "todo", "blocked", false},
		{"invalid transition", "todo", "done", true},
		{"from terminal status", "done", "todo", true},
		{"undefined status", "invalid", "todo", true},
		{"valid to terminal", "in_progress", "done", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTransition(workflow, tc.from, tc.to)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for transition %s → %s", tc.from, tc.to)
				}
			} else {
				if err != nil {
					t.Errorf("expected valid transition %s → %s, got error: %v", tc.from, tc.to, err)
				}
			}
		})
	}
}

// Test validation error messages
func TestValidationErrorMessages(t *testing.T) {
	err := &WorkflowValidationError{
		Message: "test error",
		Fix:     "do this to fix",
	}

	expected := "test error. Fix: do this to fix"
	if err.Error() != expected {
		t.Errorf("expected error message: %s, got: %s", expected, err.Error())
	}

	errNoFix := &WorkflowValidationError{
		Message: "test error",
	}

	if errNoFix.Error() != "test error" {
		t.Errorf("expected error message without fix: 'test error', got: %s", errNoFix.Error())
	}
}

// Test workflow cache
func TestWorkflowCache(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configContent := `{
		"status_flow_version": "1.0",
		"status_flow": {
			"todo": ["done"],
			"done": []
		},
		"special_statuses": {
			"_start_": ["todo"],
			"_complete_": ["done"]
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	ClearWorkflowCache()

	// First load - should read from file
	workflow1, err := LoadWorkflowConfig(configPath)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	// Second load - should use cache (same pointer)
	workflow2, err := LoadWorkflowConfig(configPath)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	// Should be same instance (from cache)
	if workflow1 != workflow2 {
		t.Error("expected cached workflow instance")
	}

	// Clear cache
	ClearWorkflowCache()

	// Third load - should read from file again (new instance)
	workflow3, err := LoadWorkflowConfig(configPath)
	if err != nil {
		t.Fatalf("third load failed: %v", err)
	}

	// Should be different instance (cache was cleared)
	if workflow1 == workflow3 {
		t.Error("expected new workflow instance after cache clear")
	}
}

// Test loading a complex workflow config from a temp file
// (Replaces old TestLoadActualSharkConfig which read the real project config)
func TestLoadComplexWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create a multi-status workflow config similar to real project configs
	configContent := `{
		"status_flow_version": "1.0",
		"status_flow": {
			"draft": ["ready_for_refinement"],
			"ready_for_refinement": ["in_refinement", "blocked"],
			"in_refinement": ["ready_for_development", "blocked"],
			"ready_for_development": ["in_development", "blocked"],
			"in_development": ["ready_for_code_review", "blocked"],
			"ready_for_code_review": ["in_code_review"],
			"in_code_review": ["ready_for_qa", "in_development"],
			"ready_for_qa": ["in_qa"],
			"in_qa": ["ready_for_approval", "in_development"],
			"ready_for_approval": ["in_approval"],
			"in_approval": ["completed", "in_development"],
			"blocked": ["draft", "ready_for_refinement", "in_refinement", "ready_for_development", "in_development"],
			"on_hold": ["draft"],
			"completed": [],
			"cancelled": []
		},
		"special_statuses": {
			"_start_": ["draft"],
			"_complete_": ["completed", "cancelled"]
		},
		"status_metadata": {
			"draft": {"color": "gray", "phase": "planning"},
			"ready_for_refinement": {"color": "cyan", "phase": "planning"},
			"in_refinement": {"color": "cyan", "phase": "planning"},
			"ready_for_development": {"color": "blue", "phase": "development"},
			"in_development": {"color": "blue", "phase": "development"},
			"ready_for_code_review": {"color": "yellow", "phase": "review"},
			"in_code_review": {"color": "yellow", "phase": "review"},
			"ready_for_qa": {"color": "magenta", "phase": "qa"},
			"in_qa": {"color": "magenta", "phase": "qa"},
			"ready_for_approval": {"color": "cyan", "phase": "approval"},
			"in_approval": {"color": "cyan", "phase": "approval"},
			"blocked": {"color": "red", "phase": "any"},
			"on_hold": {"color": "gray", "phase": "any"},
			"completed": {"color": "green", "phase": "done"},
			"cancelled": {"color": "gray", "phase": "done"}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	ClearWorkflowCache()

	workflow, err := LoadWorkflowConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if workflow == nil {
		t.Fatal("expected workflow config, got nil")
	}

	expectedStatuses := []string{
		"draft", "ready_for_refinement", "in_refinement",
		"ready_for_development", "in_development",
		"ready_for_code_review", "in_code_review",
		"ready_for_qa", "in_qa",
		"ready_for_approval", "in_approval",
		"blocked", "on_hold",
		"completed", "cancelled",
	}

	if len(workflow.StatusFlow) != len(expectedStatuses) {
		t.Errorf("expected %d statuses, got %d", len(expectedStatuses), len(workflow.StatusFlow))
	}

	for _, status := range expectedStatuses {
		if _, exists := workflow.StatusFlow[status]; !exists {
			t.Errorf("expected status %s to exist in workflow", status)
		}
	}

	// Verify special statuses
	startStatuses := workflow.SpecialStatuses[StartStatusKey]
	if len(startStatuses) == 0 {
		t.Error("expected _start_ statuses to be defined")
	}

	completeStatuses := workflow.SpecialStatuses[CompleteStatusKey]
	if len(completeStatuses) == 0 {
		t.Error("expected _complete_ statuses to be defined")
	}
}

// Helper function
func workflowStringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && workflowStringContainsHelper(s, substr)))
}

func workflowStringContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test IsBackwardTransition with default workflow
//
// The route-based task workflow (task.yaml) uses phases planning (draft),
// development (development), done (completed/cancelled), blocked (blocked),
// and an unmapped "paused" phase (on_hold) that -- like "blocked" -- doesn't
// participate in backward-transition ordering (getPhaseOrder returns -1 for
// unknown phases, schema.go).
func TestIsBackwardTransition_DefaultWorkflow(t *testing.T) {
	workflow := DefaultWorkflow()

	testCases := []struct {
		name        string
		fromStatus  string
		toStatus    string
		expectError bool
		isBackward  bool
	}{
		// Forward transitions (phase increases)
		{"planning to development", "draft", "development", false, false},
		{"development to done", "development", "completed", false, false},

		// Backward transitions (phase decreases) -- mirrors the "fail" outcome
		// routing development back to draft.
		{"development back to planning", "development", "draft", false, true},

		// Lateral/special transitions (blocked phase is special, never backward)
		{"planning to blocked", "draft", "blocked", false, false},
		{"development to blocked (special)", "development", "blocked", false, false},
		{"blocked back to planning", "blocked", "draft", false, false},
		{"blocked back to development", "blocked", "development", false, false},

		// on_hold's "paused" phase is unmapped, so it behaves like "blocked":
		// never participates in backward-transition detection.
		{"development to on_hold (special)", "development", "on_hold", false, false},
		{"on_hold back to development", "on_hold", "development", false, false},

		// Same status (no phase change)
		{"same status", "draft", "draft", false, false},

		// Undefined statuses
		{"undefined from status", "nonexistent", "draft", true, false},
		{"undefined to status", "draft", "nonexistent", true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isBackward, err := workflow.IsBackwardTransition(tc.fromStatus, tc.toStatus)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}

				if isBackward != tc.isBackward {
					t.Errorf("expected isBackward=%v, got %v for transition %s → %s",
						tc.isBackward, isBackward, tc.fromStatus, tc.toStatus)
				}
			}
		})
	}
}

// Test IsBackwardTransition with custom workflow
func TestIsBackwardTransition_CustomWorkflow(t *testing.T) {
	// Create custom workflow with multiple phases
	workflow := &WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"draft":     {"review", "discard"},
			"review":    {"approved", "draft"},
			"approved":  {"published", "review"},
			"published": {},
			"discard":   {},
		},
		StatusMetadata: map[string]StatusMetadata{
			"draft":     {Phase: "planning"},
			"review":    {Phase: "review"},
			"approved":  {Phase: "qa"},
			"published": {Phase: "done"},
			"discard":   {Phase: "any"},
		},
	}

	testCases := []struct {
		name       string
		from       string
		to         string
		isBackward bool
	}{
		// Forward transitions
		{"draft to review", "draft", "review", false},
		{"review to approved", "review", "approved", false},
		{"approved to published", "approved", "published", false},

		// Backward transitions
		{"review to draft", "review", "draft", true},
		{"approved to review", "approved", "review", true},
		{"published to approved", "published", "approved", true},

		// Transitions to "any" phase (special - any phase never participates in backward detection)
		{"draft to discard (any phase)", "draft", "discard", false},
		{"published to discard (any phase)", "published", "discard", false}, // "any" phase doesn't participate in backward
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isBackward, err := workflow.IsBackwardTransition(tc.from, tc.to)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if isBackward != tc.isBackward {
				t.Errorf("expected isBackward=%v, got %v for transition %s → %s",
					tc.isBackward, isBackward, tc.from, tc.to)
			}
		})
	}
}

// Test IsBackwardTransition with missing metadata
func TestIsBackwardTransition_MissingMetadata(t *testing.T) {
	// Workflow with some statuses missing phase info in metadata
	workflow := &WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"todo": {"done"},
			"done": {},
		},
		StatusMetadata: map[string]StatusMetadata{
			"todo": {Phase: "planning"},
			"done": {Phase: ""}, // Status exists but has no phase
		},
	}

	// Status with missing phase information - should be treated as not backward
	isBackward, err := workflow.IsBackwardTransition("todo", "done")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if isBackward {
		t.Errorf("expected forward transition when phase metadata missing")
	}
}

// Test IsBackwardTransition phase ordering
func TestIsBackwardTransition_PhaseOrdering(t *testing.T) {
	// Verify specific phase order (planning < development < review < qa < approval < done)
	workflow := &WorkflowConfig{
		Version: "1.0",
		StatusMetadata: map[string]StatusMetadata{
			"s1": {Phase: "planning"},    // Order 0
			"s2": {Phase: "development"}, // Order 1
			"s3": {Phase: "review"},      // Order 2
			"s4": {Phase: "qa"},          // Order 3
			"s5": {Phase: "approval"},    // Order 4
			"s6": {Phase: "done"},        // Order 5
			"s7": {Phase: "any"},         // Order 6 (any phase)
		},
	}

	testCases := []struct {
		name       string
		from       string
		to         string
		isBackward bool
	}{
		// Forward through ordered phases
		{"s1 to s2", "s1", "s2", false},
		{"s2 to s3", "s2", "s3", false},
		{"s3 to s4", "s3", "s4", false},
		{"s4 to s5", "s4", "s5", false},
		{"s5 to s6", "s5", "s6", false},

		// Backward through ordered phases
		{"s6 to s5", "s6", "s5", true},
		{"s5 to s4", "s5", "s4", true},
		{"s4 to s3", "s4", "s3", true},
		{"s3 to s2", "s3", "s2", true},
		{"s2 to s1", "s2", "s1", true},

		// Skip forward (still forward)
		{"s1 to s3", "s1", "s3", false},
		{"s1 to s6", "s1", "s6", false},

		// Transition to "any" phase
		{"s1 to s7 (any)", "s1", "s7", false},
		{"s6 to s7 (any)", "s6", "s7", false},
		{"s7 to s1 (from any)", "s7", "s1", false}, // "any" doesn't define backward
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isBackward, err := workflow.IsBackwardTransition(tc.from, tc.to)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if isBackward != tc.isBackward {
				t.Errorf("expected isBackward=%v, got %v for transition %s → %s",
					tc.isBackward, isBackward, tc.from, tc.to)
			}
		})
	}
}

// TestWorkflowConfig_RequireRejectionReason_Default tests that require_rejection_reason defaults to true
// when unmarshaling from JSON without the field specified
func TestWorkflowConfig_RequireRejectionReason_Default(t *testing.T) {
	// When JSON is unmarshaled without require_rejection_reason field, should default to true
	var workflow WorkflowConfig
	jsonData := []byte(`{"status_flow_version": "1.0"}`)
	if err := json.Unmarshal(jsonData, &workflow); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Default should be true (require rejection reasons)
	if !workflow.RequireRejectionReason {
		t.Error("expected RequireRejectionReason to default to true when unmarshaling")
	}
}

// TestWorkflowConfig_RequireRejectionReason_DefaultWorkflow tests default workflow setting
func TestWorkflowConfig_RequireRejectionReason_DefaultWorkflow(t *testing.T) {
	workflow := DefaultWorkflow()

	// Default workflow should require rejection reasons
	if !workflow.RequireRejectionReason {
		t.Error("expected default workflow RequireRejectionReason to be true")
	}
}

// TestLoadWorkflowConfig_RequireRejectionReason_Explicit tests loading explicit config value
func TestLoadWorkflowConfig_RequireRejectionReason_Explicit(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		{
			name: "require_rejection_reason = true",
			config: `{
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				},
				"require_rejection_reason": true
			}`,
			expected: true,
		},
		{
			name: "require_rejection_reason = false",
			config: `{
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				},
				"require_rejection_reason": false
			}`,
			expected: false,
		},
		{
			name: "require_rejection_reason omitted (defaults to true)",
			config: `{
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			expected: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unique file path for each test to avoid cache issues
			configPath := filepath.Join(tmpDir, fmt.Sprintf(".sharkconfig-%d.json", i))
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			ClearWorkflowCache()

			workflow, err := LoadWorkflowConfig(configPath)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if workflow == nil {
				t.Fatal("expected workflow, got nil")
			}

			if workflow.RequireRejectionReason != tt.expected {
				t.Errorf("expected RequireRejectionReason=%v, got %v", tt.expected, workflow.RequireRejectionReason)
			}
		})
	}
}

// TestWorkflowConfig_RequireRejectionReason_JSON tests JSON marshaling/unmarshaling
func TestWorkflowConfig_RequireRejectionReason_JSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		{
			name:     "true value",
			json:     `{"require_rejection_reason": true}`,
			expected: true,
		},
		{
			name:     "false value",
			json:     `{"require_rejection_reason": false}`,
			expected: false,
		},
		{
			name:     "missing field defaults to true",
			json:     `{}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config WorkflowConfig
			if err := json.Unmarshal([]byte(tt.json), &config); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			if config.RequireRejectionReason != tt.expected {
				t.Errorf("expected RequireRejectionReason=%v, got %v", tt.expected, config.RequireRejectionReason)
			}
		})
	}
}
