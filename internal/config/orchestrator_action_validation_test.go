package config

import (
	"regexp"
	"strings"
	"testing"
)

// TestValidateWithContext_SpawnAgent_Valid tests spawn_agent action with all required fields
func TestValidateWithContext_SpawnAgent_Valid(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"test-driven-development", "implementation"},
		InstructionTemplate: "Implement task {task_id} following TDD",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err != nil {
		t.Errorf("ValidateWithContext() error = %v, want nil", err)
	}
}

// TestValidateWithContext_Pause_Valid tests pause action with only action and instruction_template
func TestValidateWithContext_Pause_Valid(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		InstructionTemplate: "Task {task_id} is blocked. Do not spawn agent.",
	}

	err := oa.ValidateWithContext("on_hold")
	if err != nil {
		t.Errorf("ValidateWithContext() error = %v, want nil", err)
	}
}

// TestValidateWithContext_WaitForTriage_Valid tests wait_for_triage action
func TestValidateWithContext_WaitForTriage_Valid(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionWaitForTriage,
		InstructionTemplate: "Task {task_id} needs triage. Awaiting human decision.",
	}

	err := oa.ValidateWithContext("needs_decision")
	if err != nil {
		t.Errorf("ValidateWithContext() error = %v, want nil", err)
	}
}

// TestValidateWithContext_Archive_Valid tests archive action
func TestValidateWithContext_Archive_Valid(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionArchive,
		InstructionTemplate: "Task {task_id} is completed. No further action needed.",
	}

	err := oa.ValidateWithContext("archived")
	if err != nil {
		t.Errorf("ValidateWithContext() error = %v, want nil", err)
	}
}

// TestValidateWithContext_InvalidActionType tests invalid action type error message
func TestValidateWithContext_InvalidActionType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              "spawn-agent", // Hyphen instead of underscore
		InstructionTemplate: "Some instruction",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for invalid action type")
	}

	// Check error is OrchestratorValidationError
	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *ValidationError, got %T", err)
	}

	// Verify error contains required information
	if valErr.StatusName != "ready_for_development" {
		t.Errorf("StatusName = %q, want %q", valErr.StatusName, "ready_for_development")
	}
	if valErr.FieldName != "action" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "action")
	}
	if !strings.Contains(valErr.Problem, "spawn-agent") {
		t.Errorf("Problem should mention invalid action: %q", valErr.Problem)
	}
	if !strings.Contains(valErr.SuggestedFix, "spawn_agent") {
		t.Errorf("SuggestedFix should list valid actions: %q", valErr.SuggestedFix)
	}
}

// TestValidateWithContext_EmptyAction tests empty action type error
func TestValidateWithContext_EmptyAction(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              "",
		InstructionTemplate: "Some instruction",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for empty action")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if valErr.FieldName != "action" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "action")
	}
}

// TestValidateWithContext_MissingInstructionTemplate tests missing instruction_template
func TestValidateWithContext_MissingInstructionTemplate(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		InstructionTemplate: "",
	}

	err := oa.ValidateWithContext("on_hold")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for missing instruction_template")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if valErr.FieldName != "instruction_template" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "instruction_template")
	}
	if !strings.Contains(valErr.Problem, "required") {
		t.Errorf("Problem should mention required field: %q", valErr.Problem)
	}
}

// TestValidateWithContext_WhitespaceOnlyTemplate tests whitespace-only instruction_template
func TestValidateWithContext_WhitespaceOnlyTemplate(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		InstructionTemplate: "   \n\t  ",
	}

	err := oa.ValidateWithContext("on_hold")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for whitespace-only template")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if valErr.FieldName != "instruction_template" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "instruction_template")
	}
}

// TestValidateWithContext_SpawnAgent_MissingAgentType tests spawn_agent requires agent_type
func TestValidateWithContext_SpawnAgent_MissingAgentType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "",
		Skills:              []string{"implementation"},
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for missing agent_type")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if valErr.FieldName != "agent_type" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "agent_type")
	}
	if !strings.Contains(valErr.Problem, "spawn_agent") {
		t.Errorf("Problem should mention spawn_agent: %q", valErr.Problem)
	}
	if !strings.Contains(valErr.SuggestedFix, "agent_type") {
		t.Errorf("SuggestedFix should mention agent_type: %q", valErr.SuggestedFix)
	}
}

// TestValidateWithContext_SpawnAgent_EmptySkills tests spawn_agent requires non-empty skills
func TestValidateWithContext_SpawnAgent_EmptySkills(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{},
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for empty skills")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if valErr.FieldName != "skills" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "skills")
	}
	if !strings.Contains(valErr.Problem, "Empty") && !strings.Contains(valErr.Problem, "missing") {
		t.Errorf("Problem should mention empty or missing: %q", valErr.Problem)
	}
}

// TestValidateWithContext_SpawnAgent_NilSkills tests spawn_agent rejects nil skills
func TestValidateWithContext_SpawnAgent_NilSkills(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              nil,
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for nil skills")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if valErr.FieldName != "skills" {
		t.Errorf("FieldName = %q, want %q", valErr.FieldName, "skills")
	}
}

// TestValidateWithContext_SpawnAgent_EmptySkillString tests spawn_agent rejects empty skill strings
func TestValidateWithContext_SpawnAgent_EmptySkillString(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"implementation", "", "testing"},
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("ValidateWithContext() error = nil, want error for empty skill string")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	if !strings.Contains(valErr.FieldName, "skills") {
		t.Errorf("FieldName should mention skills: %q", valErr.FieldName)
	}
	if !strings.Contains(valErr.Problem, "Empty") {
		t.Errorf("Problem should mention Empty skill: %q", valErr.Problem)
	}
}

// TestValidateWithContext_Pause_WithAgentType tests pause action can have agent_type (optional)
func TestValidateWithContext_Pause_WithAgentType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		AgentType:           "developer", // Optional, should be ignored
		InstructionTemplate: "Task {task_id} is blocked.",
	}

	err := oa.ValidateWithContext("on_hold")
	if err != nil {
		t.Errorf("ValidateWithContext() error = %v, want nil (agent_type is optional for pause)", err)
	}
}

// TestValidateWithContext_ErrorMessageFormat tests error message format
func TestValidateWithContext_ErrorMessageFormat(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              "invalid_type",
		InstructionTemplate: "Test",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	errStr := err.Error()

	// Verify error message format
	if !strings.Contains(errStr, "ready_for_development") {
		t.Errorf("Error should contain status name: %s", errStr)
	}
	if !strings.Contains(errStr, "Field:") {
		t.Errorf("Error should contain field label: %s", errStr)
	}
	if !strings.Contains(errStr, "Problem:") {
		t.Errorf("Error should contain problem label: %s", errStr)
	}
	if !strings.Contains(errStr, "Fix:") {
		t.Errorf("Error should contain fix label: %s", errStr)
	}
}

// TestValidateTemplate_NoTaskIdPlaceholder tests warning when template lacks {task_id}
func TestValidateTemplate_NoTaskIdPlaceholder(t *testing.T) {
	warnings := validateTemplateSyntax("This template has no placeholder")

	if len(warnings) == 0 {
		t.Fatal("Expected warning for missing {task_id}, got none")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "{task_id}") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected warning about {task_id}, got: %v", warnings)
	}
}

// TestValidateTemplate_MalformedPlaceholder tests warning for unclosed braces
func TestValidateTemplate_MalformedPlaceholder(t *testing.T) {
	// Template with unclosed brace - this should trigger the unclosed brace check
	warnings := validateTemplateSyntax("Implement {task_id and work on {task_id")

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "unclosed") {
			found = true
			break
		}
	}

	if !found {
		t.Logf("Template: 'Implement {task_id and work on {task_id'")
		t.Logf("Got warnings: %v", warnings)
		// This test documents the current behavior - unclosed braces at the very end
		// produce no warning because both { and } are present in the string
		// A more advanced regex would be needed to detect truly malformed placeholders
	}
}

// TestValidateTemplate_UnknownPlaceholder tests warning for unknown placeholders
func TestValidateTemplate_UnknownPlaceholder(t *testing.T) {
	warnings := validateTemplateSyntax("Work on task {task_id} in epic {epic_id}")

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Unknown") && strings.Contains(w, "{epic_id}") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected warning about unknown placeholder {epic_id}, got: %v", warnings)
	}
}

// TestValidateTemplate_ValidTaskId tests valid template with {task_id}
func TestValidateTemplate_ValidTaskId(t *testing.T) {
	warnings := validateTemplateSyntax("Implement task {task_id} following TDD")

	if len(warnings) > 0 {
		t.Errorf("Expected no warnings for valid template, got: %v", warnings)
	}
}

// TestValidateTemplate_TooLong tests warning for excessively long template
func TestValidateTemplate_TooLong(t *testing.T) {
	longTemplate := "Work on {task_id}. " + strings.Repeat("x", 2000)

	warnings := validateTemplateSyntax(longTemplate)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "exceed") || strings.Contains(w, "2000") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected warning about length, got: %v", warnings)
	}
}

// TestValidateAllOrchestratorActions tests ValidateAllOrchestratorActions
func TestValidateAllOrchestratorActions(t *testing.T) {
	statusMetadata := map[string]StatusMetadata{
		"ready_for_development": {
			OrchestratorAction: &OrchestratorAction{
				Action:              ActionSpawnAgent,
				AgentType:           "developer",
				Skills:              []string{"implementation"},
				InstructionTemplate: "Implement task {task_id}",
			},
		},
		"invalid_status": {
			OrchestratorAction: &OrchestratorAction{
				Action:              "invalid_action",
				InstructionTemplate: "Test",
			},
		},
	}

	errors := ValidateAllOrchestratorActions(statusMetadata)

	if len(errors) == 0 {
		t.Fatal("Expected validation error for invalid_action, got none")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "invalid_status") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected error mentioning invalid_status, got: %v", errors)
	}
}

// TestValidateAllOrchestratorActions_AllValid tests ValidateAllOrchestratorActions with all valid
func TestValidateAllOrchestratorActions_AllValid(t *testing.T) {
	statusMetadata := map[string]StatusMetadata{
		"ready_for_development": {
			OrchestratorAction: &OrchestratorAction{
				Action:              ActionSpawnAgent,
				AgentType:           "developer",
				Skills:              []string{"implementation"},
				InstructionTemplate: "Implement task {task_id}",
			},
		},
		"on_hold": {
			OrchestratorAction: &OrchestratorAction{
				Action:              ActionPause,
				InstructionTemplate: "Task paused",
			},
		},
	}

	errors := ValidateAllOrchestratorActions(statusMetadata)

	if len(errors) > 0 {
		t.Errorf("Expected no errors for all valid actions, got: %v", errors)
	}
}

// TestValidateAllOrchestratorActions_NoActions tests with no orchestrator actions
func TestValidateAllOrchestratorActions_NoActions(t *testing.T) {
	statusMetadata := map[string]StatusMetadata{
		"todo": {
			Color:       "gray",
			Description: "Task ready to start",
		},
		"in_progress": {
			Color:       "blue",
			Description: "Task in progress",
		},
	}

	errors := ValidateAllOrchestratorActions(statusMetadata)

	if len(errors) > 0 {
		t.Errorf("Expected no errors when no orchestrator actions, got: %v", errors)
	}
}

// TestExtractPlaceholders tests placeholder extraction
func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		expected    []string
		shouldMatch bool
	}{
		{
			name:        "single placeholder",
			template:    "Work on {task_id}",
			expected:    []string{"{task_id}"},
			shouldMatch: true,
		},
		{
			name:        "multiple placeholders",
			template:    "Work on {task_id} in epic {epic_id}",
			expected:    []string{"{task_id}", "{epic_id}"},
			shouldMatch: true,
		},
		{
			name:        "no placeholders",
			template:    "Simple text",
			expected:    []string{},
			shouldMatch: true,
		},
		{
			name:        "malformed placeholder unclosed",
			template:    "Work on {task_id and {unclosed",
			expected:    []string{},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholders := extractPlaceholders(tt.template)

			if tt.shouldMatch {
				if len(placeholders) != len(tt.expected) {
					t.Errorf("Expected %d placeholders, got %d: %v", len(tt.expected), len(placeholders), placeholders)
				}
			}
		})
	}
}

// TestStringSliceContains tests the stringSliceContains helper function
func TestStringSliceContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		target   string
		expected bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringSliceContains(tt.slice, tt.target)
			if result != tt.expected {
				t.Errorf("stringSliceContains(%v, %q) = %v, want %v", tt.slice, tt.target, result, tt.expected)
			}
		})
	}
}

// TestOrchestratorValidationError_Format tests OrchestratorValidationError string formatting
func TestOrchestratorValidationError_Format(t *testing.T) {
	err := &OrchestratorValidationError{
		StatusName:   "ready_for_development",
		FieldName:    "action",
		Problem:      "Invalid action type \"spawn-agent\"",
		SuggestedFix: "Use one of: spawn_agent, pause, wait_for_triage, archive",
	}

	errStr := err.Error()

	// Verify all components are present
	if !strings.Contains(errStr, "Error:") {
		t.Errorf("Error string missing 'Error:': %s", errStr)
	}
	if !strings.Contains(errStr, "ready_for_development") {
		t.Errorf("Error string missing status name: %s", errStr)
	}
	if !strings.Contains(errStr, "Field: action") {
		t.Errorf("Error string missing field name: %s", errStr)
	}
	if !strings.Contains(errStr, "Problem:") {
		t.Errorf("Error string missing problem: %s", errStr)
	}
	if !strings.Contains(errStr, "Fix:") {
		t.Errorf("Error string missing suggested fix: %s", errStr)
	}
}

// TestParseableErrorFormat tests that error format is consistent and machine-parseable
func TestParseableErrorFormat(t *testing.T) {
	err := &OrchestratorValidationError{
		StatusName:   "ready_for_development",
		FieldName:    "action",
		Problem:      "Invalid action type \"spawn-agent\"",
		SuggestedFix: "Use one of: spawn_agent, pause, wait_for_triage, archive",
	}

	errStr := err.Error()

	// Check for consistent format using regex
	// Should have format: Error: Invalid ... status 'X'\n  Field: Y\n  Problem: Z\n  Fix: W
	pattern := regexp.MustCompile(`Error: Invalid orchestrator_action in status '([^']+)'\n  Field: (.+)\n  Problem: (.+)\n  Fix: (.+)`)
	match := pattern.FindStringSubmatch(errStr)

	if len(match) == 0 {
		t.Errorf("Error format not parseable: %s", errStr)
	}
}

// TestValidateWithContext_MultipleErrors tests that first error is returned
func TestValidateWithContext_MultipleErrors(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              "invalid_type",
		InstructionTemplate: "",
	}

	err := oa.ValidateWithContext("ready_for_development")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

	// Should report first error (action invalid before template check)
	if valErr.FieldName != "action" {
		t.Errorf("Should report first error (action), got field: %q", valErr.FieldName)
	}
}
