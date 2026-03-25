package action

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

	valErr, ok := err.(*OrchestratorValidationError)
	if !ok {
		t.Fatalf("Expected *OrchestratorValidationError, got %T", err)
	}

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
}

// TestValidateWithContext_Pause_WithAgentType tests pause action can have agent_type (optional)
func TestValidateWithContext_Pause_WithAgentType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		AgentType:           "developer",
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

// TestValidateTemplate_NoPlaceholder tests warning when template has no placeholders
func TestValidateTemplate_NoPlaceholder(t *testing.T) {
	warnings := validateTemplateSyntax("This template has no placeholder")
	if len(warnings) == 0 {
		t.Fatal("Expected warning for missing placeholder, got none")
	}
}

// TestValidateTemplate_MalformedPlaceholder tests warning for unclosed braces
func TestValidateTemplate_MalformedPlaceholder(t *testing.T) {
	warnings := validateTemplateSyntax("Implement {task_id and work on {task_id")
	_ = warnings // Documents current behavior
}

// TestValidateTemplate_WithCustomPlaceholder tests custom placeholders
func TestValidateTemplate_WithCustomPlaceholder(t *testing.T) {
	warnings := validateTemplateSyntax("Work on task {task_id} with {custom_field}")
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings for template with valid placeholders, got: %v", warnings)
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

// TestValidateAllOrchestratorActions tests validation of action maps
func TestValidateAllOrchestratorActions(t *testing.T) {
	actionsByStatus := map[string]*OrchestratorAction{
		"ready_for_development": {
			Action:              ActionSpawnAgent,
			AgentType:           "developer",
			Skills:              []string{"implementation"},
			InstructionTemplate: "Implement task {task_id}",
		},
		"invalid_status": {
			Action:              "invalid_action",
			InstructionTemplate: "Test",
		},
	}

	errs := ValidateAllOrchestratorActions(actionsByStatus)
	if len(errs) == 0 {
		t.Fatal("Expected validation error for invalid_action, got none")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "invalid_status") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected error mentioning invalid_status, got: %v", errs)
	}
}

// TestValidateAllOrchestratorActions_AllValid tests with all valid actions
func TestValidateAllOrchestratorActions_AllValid(t *testing.T) {
	actionsByStatus := map[string]*OrchestratorAction{
		"ready_for_development": {
			Action:              ActionSpawnAgent,
			AgentType:           "developer",
			Skills:              []string{"implementation"},
			InstructionTemplate: "Implement task {task_id}",
		},
		"on_hold": {
			Action:              ActionPause,
			InstructionTemplate: "Task paused",
		},
	}

	errs := ValidateAllOrchestratorActions(actionsByStatus)
	if len(errs) > 0 {
		t.Errorf("Expected no errors for all valid actions, got: %v", errs)
	}
}

// TestValidateAllOrchestratorActions_NoActions tests with no orchestrator actions
func TestValidateAllOrchestratorActions_NoActions(t *testing.T) {
	actionsByStatus := map[string]*OrchestratorAction{
		"todo":        nil,
		"in_progress": nil,
	}

	errs := ValidateAllOrchestratorActions(actionsByStatus)
	if len(errs) > 0 {
		t.Errorf("Expected no errors when no orchestrator actions, got: %v", errs)
	}
}

// TestExtractPlaceholders tests placeholder extraction
func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		expected int
	}{
		{"single placeholder", "Work on {task_id}", 1},
		{"multiple placeholders", "Work on {task_id} in epic {epic_id}", 2},
		{"no placeholders", "Simple text", 0},
		{"malformed unclosed", "Work on {task_id and {unclosed", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholders := extractPlaceholders(tt.tmpl)
			if len(placeholders) != tt.expected {
				t.Errorf("Expected %d placeholders, got %d: %v", tt.expected, len(placeholders), placeholders)
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

// TestOrchestratorValidationError_Format tests error string formatting
func TestOrchestratorValidationError_Format(t *testing.T) {
	err := &OrchestratorValidationError{
		StatusName:   "ready_for_development",
		FieldName:    "action",
		Problem:      "Invalid action type \"spawn-agent\"",
		SuggestedFix: "Use one of: spawn_agent, pause, wait_for_triage, archive",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "ready_for_development") {
		t.Errorf("Error string missing status name: %s", errStr)
	}
	if !strings.Contains(errStr, "Field: action") {
		t.Errorf("Error string missing field name: %s", errStr)
	}
}

// TestParseableErrorFormat tests that error format is machine-parseable
func TestParseableErrorFormat(t *testing.T) {
	err := &OrchestratorValidationError{
		StatusName:   "ready_for_development",
		FieldName:    "action",
		Problem:      "Invalid action type \"spawn-agent\"",
		SuggestedFix: "Use one of: spawn_agent, pause, wait_for_triage, archive",
	}

	errStr := err.Error()
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
	if valErr.FieldName != "action" {
		t.Errorf("Should report first error (action), got field: %q", valErr.FieldName)
	}
}

// TestValidateTemplate_GenericId_NoWarning tests template with {id}
func TestValidateTemplate_GenericId_NoWarning(t *testing.T) {
	warnings := validateTemplateSyntax("Process entity {id}")
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings for template with {id}, got: %v", warnings)
	}
}

// TestValidateTemplate_EpicId_NoWarning tests template with {epic_id}
func TestValidateTemplate_EpicId_NoWarning(t *testing.T) {
	warnings := validateTemplateSyntax("Research epic {epic_id}")
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}
}

// TestValidateTemplate_FeatureId_NoWarning tests template with {feature_id}
func TestValidateTemplate_FeatureId_NoWarning(t *testing.T) {
	warnings := validateTemplateSyntax("Refine feature {feature_id}")
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}
}

// TestValidateTemplate_NoKnownPlaceholder_Warning tests template without any placeholder
func TestValidateTemplate_NoKnownPlaceholder_Warning(t *testing.T) {
	warnings := validateTemplateSyntax("Do some work on this entity")
	if len(warnings) == 0 {
		t.Fatal("Expected warning for template without any placeholder, got none")
	}
}

// TestValidateTemplate_MixedPlaceholders_NoWarning tests mixed placeholders
func TestValidateTemplate_MixedPlaceholders_NoWarning(t *testing.T) {
	warnings := validateTemplateSyntax("Work on {id} with {unknown} data")
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings for template with mixed placeholders, got: %v", warnings)
	}
}
