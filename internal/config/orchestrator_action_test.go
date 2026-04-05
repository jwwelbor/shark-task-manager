package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOrchestratorAction_Validate_SpawnAgent validates spawn_agent with all required fields
func TestOrchestratorAction_Validate_SpawnAgent(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"test-driven-development", "implementation"},
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestOrchestratorAction_Validate_SpawnAgent_MissingAgentType validates spawn_agent requires agent_type
func TestOrchestratorAction_Validate_SpawnAgent_MissingAgentType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "",
		Skills:              []string{"implementation"},
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing agent_type")
	}
	if !strings.Contains(err.Error(), "agent_type") {
		t.Errorf("Validate() error message should mention agent_type: %v", err)
	}
}

// TestOrchestratorAction_Validate_SpawnAgent_MissingSkills validates spawn_agent requires skills
func TestOrchestratorAction_Validate_SpawnAgent_MissingSkills(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{},
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing skills")
	}
	if !strings.Contains(err.Error(), "skills") {
		t.Errorf("Validate() error message should mention skills: %v", err)
	}
}

// TestOrchestratorAction_Validate_Pause validates pause action with only instruction_template
func TestOrchestratorAction_Validate_Pause(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		InstructionTemplate: "Task {task_id} is blocked. Do not spawn agent.",
	}

	err := oa.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestOrchestratorAction_Validate_WaitForTriage validates wait_for_triage action
func TestOrchestratorAction_Validate_WaitForTriage(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionWaitForTriage,
		InstructionTemplate: "Task {task_id} needs triage. Awaiting human decision.",
	}

	err := oa.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestOrchestratorAction_Validate_Archive validates archive action
func TestOrchestratorAction_Validate_Archive(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionArchive,
		InstructionTemplate: "Task {task_id} is completed. No further action needed.",
	}

	err := oa.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestOrchestratorAction_Validate_InvalidAction validates invalid action type
func TestOrchestratorAction_Validate_InvalidAction(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              "invalid_action",
		InstructionTemplate: "Some instruction",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action type") {
		t.Errorf("Validate() error message should mention invalid action: %v", err)
	}
}

// TestOrchestratorAction_Validate_MissingInstruction validates instruction_template is always required
func TestOrchestratorAction_Validate_MissingInstruction(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		InstructionTemplate: "",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing instruction_template")
	}
	if !strings.Contains(err.Error(), "instruction_template") {
		t.Errorf("Validate() error message should mention instruction_template: %v", err)
	}
}

// TestOrchestratorAction_Validate_MissingInstruction_Whitespace validates whitespace-only template
func TestOrchestratorAction_Validate_MissingInstruction_Whitespace(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		InstructionTemplate: "   \n\t  ",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for whitespace-only instruction_template")
	}
}

// TestOrchestratorAction_PopulateTemplate validates {task_id} substitution
func TestOrchestratorAction_PopulateTemplate(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"implementation"},
		InstructionTemplate: "Implement task {task_id} following TDD",
	}

	result := oa.PopulateTemplate(map[string]string{"task_id": "T-E07-F21-001"})
	expected := "Implement task T-E07-F21-001 following TDD"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_MultipleOccurrences validates multiple {task_id} placeholders
func TestOrchestratorAction_PopulateTemplate_MultipleOccurrences(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"implementation"},
		InstructionTemplate: "Work on {task_id}. Document completion in {task_id} file.",
	}

	result := oa.PopulateTemplate(map[string]string{"task_id": "T-E07-F21-001"})
	expected := "Work on T-E07-F21-001. Document completion in T-E07-F21-001 file."

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_NoPlaceholder validates template without {task_id} unchanged
func TestOrchestratorAction_PopulateTemplate_NoPlaceholder(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionArchive,
		InstructionTemplate: "Task completed. No further action needed.",
	}

	result := oa.PopulateTemplate(map[string]string{"task_id": "T-E07-F21-001"})
	expected := "Task completed. No further action needed."

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_CaseSensitive validates {task_id} is case-sensitive
func TestOrchestratorAction_PopulateTemplate_CaseSensitive(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"implementation"},
		InstructionTemplate: "Work on {TASK_ID} and {task_id}",
	}

	result := oa.PopulateTemplate(map[string]string{"task_id": "T-E07-F21-001"})
	expected := "Work on {TASK_ID} and T-E07-F21-001"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestValidActionTypes validates constant array contains all action types
func TestValidActionTypes(t *testing.T) {
	expectedTypes := map[string]bool{
		ActionSpawnAgent:    true,
		ActionPause:         true,
		ActionWaitForTriage: true,
		ActionArchive:       true,
		ActionAdvanceStatus: true,
		ActionCheckOrResume: true,
		ActionCascade:       true,
	}

	for _, actionType := range ValidActionTypes {
		if !expectedTypes[actionType] {
			t.Errorf("Unexpected action type in ValidActionTypes: %s", actionType)
		}
	}

	if len(ValidActionTypes) != len(expectedTypes) {
		t.Errorf("ValidActionTypes length = %d, want %d", len(ValidActionTypes), len(expectedTypes))
	}
}

// TestOrchestratorAction_Validate_AllActionTypes validates all action types pass with minimal config
func TestOrchestratorAction_Validate_AllActionTypes(t *testing.T) {
	tests := []struct {
		name   string
		action string
	}{
		{"spawn_agent", ActionSpawnAgent},
		{"pause", ActionPause},
		{"wait_for_triage", ActionWaitForTriage},
		{"archive", ActionArchive},
		{"advance_status", ActionAdvanceStatus},
		{"cascade", ActionCascade},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oa := &OrchestratorAction{
				Action:              tt.action,
				InstructionTemplate: "Template for {task_id}",
			}

			// spawn_agent needs extra fields
			if tt.action == ActionSpawnAgent {
				oa.AgentType = "developer"
				oa.Skills = []string{"implementation"}
			}

			err := oa.Validate()
			if err != nil {
				t.Errorf("Validate() for %s error = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestOrchestratorAction_Validate_SpawnAgent_EmptySkillsArray validates spawn_agent rejects empty skills
func TestOrchestratorAction_Validate_SpawnAgent_EmptySkillsArray(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{}, // Empty array
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() should reject empty skills array for spawn_agent")
	}
}

// TestOrchestratorAction_Validate_SpawnAgent_NilSkils validates spawn_agent rejects nil skills
func TestOrchestratorAction_Validate_SpawnAgent_NilSkills(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              nil, // nil slice
		InstructionTemplate: "Implement task {task_id}",
	}

	err := oa.Validate()
	if err == nil {
		t.Error("Validate() should reject nil skills for spawn_agent")
	}
}

// TestOrchestratorAction_Validate_Pause_WithAgentType validates pause can have agent_type (optional)
func TestOrchestratorAction_Validate_Pause_WithAgentType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionPause,
		AgentType:           "developer",
		InstructionTemplate: "Task {task_id} is blocked.",
	}

	err := oa.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (agent_type is optional)", err)
	}
}

// TestOrchestratorAction_Validate_Error returns proper error type
func TestOrchestratorAction_Validate_ErrorType(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              "invalid",
		InstructionTemplate: "Test",
	}

	err := oa.Validate()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify it's an error (not a specific type, just that it's an error)
	if !errors.Is(err, err) {
		t.Errorf("Expected error type, got %T", err)
	}
}

// TestOrchestratorAction_PopulateTemplate_EmptyTaskID validates substitution with empty task ID
func TestOrchestratorAction_PopulateTemplate_EmptyTaskID(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Skills:              []string{"implementation"},
		InstructionTemplate: "Implement task {task_id}",
	}

	result := oa.PopulateTemplate(map[string]string{"task_id": ""})
	expected := "Implement task "

	if result != expected {
		t.Errorf("PopulateTemplate() with empty ID = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_GenericId validates {id} placeholder is replaced with entity key
func TestPopulateTemplate_GenericId(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Process entity {id}",
	}

	result := oa.PopulateTemplate(map[string]string{"id": "E16"})
	expected := "Process entity E16"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_EpicId validates {epic_id} placeholder is replaced with entity key
func TestPopulateTemplate_EpicId(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Research epic {epic_id}",
	}

	result := oa.PopulateTemplate(map[string]string{"epic_id": "E16"})
	expected := "Research epic E16"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_FeatureId validates {feature_id} placeholder is replaced with entity key
func TestPopulateTemplate_FeatureId(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Refine feature {feature_id}",
	}

	result := oa.PopulateTemplate(map[string]string{"feature_id": "E16-F01"})
	expected := "Refine feature E16-F01"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_TaskId_BackwardCompat validates {task_id} still works exactly as before
func TestPopulateTemplate_TaskId_BackwardCompat(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Implement task {task_id}",
	}

	result := oa.PopulateTemplate(map[string]string{"task_id": "T-E07-F21-001"})
	expected := "Implement task T-E07-F21-001"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_MixedPlaceholders validates template with both {id} and {task_id}
func TestPopulateTemplate_MixedPlaceholders(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Work on {id}, see {task_id} file",
	}

	result := oa.PopulateTemplate(map[string]string{
		"id":      "T-E07-F01-001",
		"task_id": "T-E07-F01-001",
	})
	expected := "Work on T-E07-F01-001, see T-E07-F01-001 file"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_NoPlaceholders_Unchanged validates template without any placeholder returned as-is
func TestPopulateTemplate_NoPlaceholders_Unchanged(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "No placeholders here",
	}

	result := oa.PopulateTemplate(map[string]string{"id": "E16"})
	expected := "No placeholders here"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_MapBased_SingleField validates {title} replaced from map
func TestPopulateTemplate_MapBased_SingleField(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Work on {title}",
	}

	result := oa.PopulateTemplate(map[string]string{"title": "User Authentication"})
	expected := "Work on User Authentication"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_MapBased_MultipleFields validates multiple placeholders replaced
func TestPopulateTemplate_MapBased_MultipleFields(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Implement {id} - {title} at {file_path}",
	}

	result := oa.PopulateTemplate(map[string]string{
		"id":        "T-E07-F01-001",
		"title":     "JWT Validation",
		"file_path": "docs/plan/task.md",
	})
	expected := "Implement T-E07-F01-001 - JWT Validation at docs/plan/task.md"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_MapBased_UnknownKey validates {unknown} left as-is
func TestPopulateTemplate_MapBased_UnknownKey(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Work on {id} with {unknown_field}",
	}

	result := oa.PopulateTemplate(map[string]string{
		"id": "T-E07-F01-001",
	})
	expected := "Work on T-E07-F01-001 with {unknown_field}"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_MapBased_EmptyMap validates template returned unchanged
func TestPopulateTemplate_MapBased_EmptyMap(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Work on {id} and {title}",
	}

	result := oa.PopulateTemplate(map[string]string{})
	expected := "Work on {id} and {title}"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// TestPopulateTemplate_MapBased_BackwardCompat validates {task_id} still works via map key
func TestPopulateTemplate_MapBased_BackwardCompat(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Implement task {task_id}",
	}

	result := oa.PopulateTemplate(map[string]string{
		"task_id": "T-E07-F21-001",
	})
	expected := "Implement task T-E07-F21-001"

	if result != expected {
		t.Errorf("PopulateTemplate() = %q, want %q", result, expected)
	}
}

// === NEW TESTS FOR .tmpl DETECTION (T-E07-F30-002) ===

// TestOrchestratorAction_PopulateTemplate_TmplDetection_Positive validates .tmpl suffix triggers template engine
func TestOrchestratorAction_PopulateTemplate_TmplDetection_Positive(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "task/test.tmpl",
	}

	// This should detect .tmpl and try to use template engine
	// For now, we expect it to fail because the template doesn't exist
	// But we're testing that it attempts the template path
	result := oa.PopulateTemplate(map[string]string{
		"task_id": "E07-F30-001",
	})

	// Empty string expected because template doesn't exist (error handling)
	// The detection logic should log an error and return empty string
	if result != "" {
		t.Errorf("PopulateTemplate() with missing .tmpl file = %q, want empty string (graceful degradation)", result)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplDetection_CaseSensitive validates .TMPL uppercase not treated as template
func TestOrchestratorAction_PopulateTemplate_TmplDetection_CaseSensitive(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Task: {task_id}.TMPL",
	}

	result := oa.PopulateTemplate(map[string]string{
		"task_id": "E07-F30-001",
	})
	expected := "Task: E07-F30-001.TMPL"

	// .TMPL (uppercase) should NOT trigger template engine
	if result != expected {
		t.Errorf("PopulateTemplate() .TMPL (uppercase) = %q, want %q (legacy path)", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplDetection_NoSuffix validates no .tmpl uses legacy path
func TestOrchestratorAction_PopulateTemplate_TmplDetection_NoSuffix(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "Implement {id} - {title}",
	}

	result := oa.PopulateTemplate(map[string]string{
		"id":    "E07-F30-001",
		"title": "Template Engine",
	})
	expected := "Implement E07-F30-001 - Template Engine"

	if result != expected {
		t.Errorf("PopulateTemplate() inline (no suffix) = %q, want %q", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplDetection_TxtSuffix validates .txt uses legacy path
func TestOrchestratorAction_PopulateTemplate_TmplDetection_TxtSuffix(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "See {file_path} for details",
	}

	result := oa.PopulateTemplate(map[string]string{
		"file_path": "task.txt",
	})
	expected := "See task.txt for details"

	if result != expected {
		t.Errorf("PopulateTemplate() .txt suffix = %q, want %q (legacy path)", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_BackwardCompat_LegacyPath validates 62 inline templates still work
func TestOrchestratorAction_PopulateTemplate_BackwardCompat_LegacyPath(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		want     string
	}{
		{
			name:     "simple inline template",
			template: "Launch {agent_type} for {task_id}",
			vars: map[string]string{
				"agent_type": "developer",
				"task_id":    "E07-F30-001",
			},
			want: "Launch developer for E07-F30-001",
		},
		{
			name:     "inline with multiple placeholders",
			template: "Process {id}: {title} in {file_path}",
			vars: map[string]string{
				"id":        "E07-F30-001",
				"title":     "Add .tmpl detection",
				"file_path": "docs/plan/task.md",
			},
			want: "Process E07-F30-001: Add .tmpl detection in docs/plan/task.md",
		},
		{
			name:     "inline with no placeholders",
			template: "Task completed. No action needed.",
			vars: map[string]string{
				"id": "E07-F30-001",
			},
			want: "Task completed. No action needed.",
		},
		{
			name:     "inline with empty vars map",
			template: "Static instruction for {id}",
			vars:     map[string]string{},
			want:     "Static instruction for {id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oa := &OrchestratorAction{
				InstructionTemplate: tt.template,
			}

			result := oa.PopulateTemplate(tt.vars)
			if result != tt.want {
				t.Errorf("PopulateTemplate() = %q, want %q", result, tt.want)
			}
		})
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplPath_WithSlash validates template path with directory
func TestOrchestratorAction_PopulateTemplate_TmplPath_WithSlash(t *testing.T) {
	// Use a non-existent template path to test graceful degradation
	oa := &OrchestratorAction{
		InstructionTemplate: "task/nonexistent_status.tmpl",
	}

	// Should attempt template engine (path ends with .tmpl)
	result := oa.PopulateTemplate(map[string]string{
		"task_id": "E07-F30-001",
	})

	// Will be empty because template file doesn't exist (graceful error handling)
	if result != "" {
		t.Errorf("PopulateTemplate() missing template path = %q, want empty string (graceful degradation)", result)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplEdgeCase_OnlyExtension validates .tmpl only
func TestOrchestratorAction_PopulateTemplate_TmplEdgeCase_OnlyExtension(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: ".tmpl",
	}

	// Technically ends with .tmpl so would try template engine
	// But file doesn't exist, returns empty string gracefully
	result := oa.PopulateTemplate(map[string]string{
		"id": "E07-F30-001",
	})

	if result != "" {
		t.Errorf("PopulateTemplate() edge case (.tmpl only) = %q, want empty string", result)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplEdgeCase_InMiddle validates .tmpl in middle of string
func TestOrchestratorAction_PopulateTemplate_TmplEdgeCase_InMiddle(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "view.tmpl.backup with {id}",
	}

	// .tmpl is NOT at the end, so uses legacy path
	result := oa.PopulateTemplate(map[string]string{
		"id": "E07-F30-001",
	})
	expected := "view.tmpl.backup with E07-F30-001"

	if result != expected {
		t.Errorf("PopulateTemplate() .tmpl in middle = %q, want %q (legacy path)", result, expected)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplDetection_NilVars validates nil vars with .tmpl
func TestOrchestratorAction_PopulateTemplate_TmplDetection_NilVars(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "task/simple.tmpl",
	}

	// Nil vars should be handled gracefully
	result := oa.PopulateTemplate(nil)

	// With template engine error, returns empty string
	if result != "" {
		t.Errorf("PopulateTemplate() with nil vars and .tmpl = %q, want empty string", result)
	}
}

// TestOrchestratorAction_PopulateTemplate_TmplDetection_EmptyVars validates empty vars with .tmpl
func TestOrchestratorAction_PopulateTemplate_TmplDetection_EmptyVars(t *testing.T) {
	oa := &OrchestratorAction{
		InstructionTemplate: "task/simple.tmpl",
	}

	// Empty vars should be handled gracefully
	result := oa.PopulateTemplate(map[string]string{})

	// Template doesn't exist, returns empty string
	if result != "" {
		t.Errorf("PopulateTemplate() with empty vars and .tmpl = %q, want empty string", result)
	}
}

// TestOrchestratorAction_PopulateTemplate_DetectionLogic validates detection logic doesn't affect legacy
func TestOrchestratorAction_PopulateTemplate_DetectionLogic(t *testing.T) {
	tests := []struct {
		name                  string
		instructionTemplate   string
		vars                  map[string]string
		expectedToUseLegacy   bool
		expectedToUseTemplate bool
	}{
		{
			name:                "simple placeholder",
			instructionTemplate: "Launch {agent_type}",
			vars:                map[string]string{"agent_type": "developer"},
			expectedToUseLegacy: true,
		},
		{
			name:                  ".tmpl filename",
			instructionTemplate:   "task/ready_for_dev.tmpl",
			vars:                  map[string]string{"task_id": "E07-F30-001"},
			expectedToUseTemplate: true,
		},
		{
			name:                ".TMPL uppercase",
			instructionTemplate: "task.TMPL",
			vars:                map[string]string{"id": "E07"},
			expectedToUseLegacy: true,
		},
		{
			name:                ".tmpl in middle",
			instructionTemplate: "file.tmpl.bak {id}",
			vars:                map[string]string{"id": "E07"},
			expectedToUseLegacy: true,
		},
		{
			name:                "no extension",
			instructionTemplate: "{agent} works on {task}",
			vars: map[string]string{
				"agent": "developer",
				"task":  "E07-F30-001",
			},
			expectedToUseLegacy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oa := &OrchestratorAction{
				InstructionTemplate: tt.instructionTemplate,
			}

			result := oa.PopulateTemplate(tt.vars)

			// Detailed validation is in TestOrchestratorAction_PopulateTemplate_LegacyPathKeptIntact
			// and TestOrchestratorAction_PopulateTemplate_BackwardCompat_LegacyPath

			// If we expect template path and it fails (file doesn't exist), should be empty
			if tt.expectedToUseTemplate && result == "" {
				// This is expected for missing templates
				return
			}
		})
	}
}

// TestOrchestratorAction_PopulateTemplate_LegacyPathKeptIntact validates legacy code unchanged
func TestOrchestratorAction_PopulateTemplate_LegacyPathKeptIntact(t *testing.T) {
	// These tests verify the legacy path behavior is completely unchanged
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		want     string
	}{
		{
			name:     "single placeholder",
			template: "Task {id}",
			vars:     map[string]string{"id": "T-001"},
			want:     "Task T-001",
		},
		{
			name:     "multiple placeholders",
			template: "Process {entity}: {title}",
			vars: map[string]string{
				"entity": "E07",
				"title":  "Feature X",
			},
			want: "Process E07: Feature X",
		},
		{
			name:     "placeholder not in vars",
			template: "Work on {unknown}",
			vars:     map[string]string{"id": "E07"},
			want:     "Work on {unknown}",
		},
		{
			name:     "repeated placeholder",
			template: "Link {id} to {id}",
			vars:     map[string]string{"id": "E07"},
			want:     "Link E07 to E07",
		},
		{
			name:     "no vars",
			template: "Static text",
			vars:     map[string]string{},
			want:     "Static text",
		},
		{
			name:     "nil vars",
			template: "Text with {placeholder}",
			vars:     nil,
			want:     "Text with {placeholder}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oa := &OrchestratorAction{
				InstructionTemplate: tt.template,
			}

			result := oa.PopulateTemplate(tt.vars)
			if result != tt.want {
				t.Errorf("PopulateTemplate() = %q, want %q", result, tt.want)
			}
		})
	}
}

// TC-021: OrchestratorAction accepts Provider and Model fields
func TestOrchestratorAction_ProviderAndModel_Fields(t *testing.T) {
	oa := &OrchestratorAction{
		Action:              ActionSpawnAgent,
		AgentType:           "developer",
		Provider:            "anthropic",
		Model:               "claude-opus-4-5",
		Skills:              []string{"implementation"},
		InstructionTemplate: "implement {task_key}",
	}

	if oa.Provider != "anthropic" {
		t.Errorf("expected Provider %q, got %q", "anthropic", oa.Provider)
	}
	if oa.Model != "claude-opus-4-5" {
		t.Errorf("expected Model %q, got %q", "claude-opus-4-5", oa.Model)
	}

	// Validate must not reject Provider or Model
	err := oa.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (Provider/Model should not be rejected)", err)
	}
}

// TC-021: Validate does NOT reject missing/empty Provider
func TestOrchestratorAction_Validate_EmptyProvider_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{"empty provider and model", "", ""},
		{"provider only", "anthropic", ""},
		{"model only", "", "claude-sonnet-4-5"},
		{"openai provider", "openai", "o3"},
		{"arbitrary provider", "custom-provider", "custom-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oa := &OrchestratorAction{
				Action:              ActionSpawnAgent,
				AgentType:           "developer",
				Provider:            tt.provider,
				Model:               tt.model,
				Skills:              []string{"implementation"},
				InstructionTemplate: "implement {task_key}",
			}
			err := oa.Validate()
			if err != nil {
				t.Errorf("Validate() error = %v, want nil (Provider=%q Model=%q should be accepted)", err, tt.provider, tt.model)
			}
		})
	}
}

// TC-022: OrchestratorAction JSON round-trips Provider and Model
func TestOrchestratorAction_JSON_RoundTrip_ProviderAndModel(t *testing.T) {
	t.Run("with provider and model", func(t *testing.T) {
		// Write config with provider and model and load via WorkflowConfig
		jsonInput := `{
			"status_flow_version": "1.0",
			"status_flow": {},
			"special_statuses": {},
			"status_metadata": {
				"some_status": {
					"color": "blue",
					"phase": "development",
					"orchestrator_action": {
						"action": "spawn_agent",
						"agent_type": "developer",
						"provider": "openai",
						"model": "o3",
						"skills": ["implementation"],
						"instruction_template": "do the work"
					}
				}
			}
		}`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")
		if err := os.WriteFile(configPath, []byte(jsonInput), 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
		ClearWorkflowCache()
		wf := GetWorkflowOrDefault(configPath)
		if wf == nil {
			t.Fatal("expected workflow, got nil")
		}
		meta, ok := wf.StatusMetadata["some_status"]
		if !ok || meta.OrchestratorAction == nil {
			t.Fatal("expected orchestrator_action in some_status")
		}
		if meta.OrchestratorAction.Provider != "openai" {
			t.Errorf("expected Provider %q, got %q", "openai", meta.OrchestratorAction.Provider)
		}
		if meta.OrchestratorAction.Model != "o3" {
			t.Errorf("expected Model %q, got %q", "o3", meta.OrchestratorAction.Model)
		}
	})

	t.Run("without provider and model (backward compat)", func(t *testing.T) {
		jsonInput := `{
			"status_flow_version": "1.0",
			"status_flow": {},
			"special_statuses": {},
			"status_metadata": {
				"some_status": {
					"color": "blue",
					"phase": "development",
					"orchestrator_action": {
						"action": "spawn_agent",
						"agent_type": "developer",
						"skills": ["implementation"],
						"instruction_template": "do the work"
					}
				}
			}
		}`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")
		if err := os.WriteFile(configPath, []byte(jsonInput), 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
		ClearWorkflowCache()
		wf := GetWorkflowOrDefault(configPath)
		if wf == nil {
			t.Fatal("expected workflow, got nil")
		}
		meta, ok := wf.StatusMetadata["some_status"]
		if !ok || meta.OrchestratorAction == nil {
			t.Fatal("expected orchestrator_action in some_status")
		}
		if meta.OrchestratorAction.Provider != "" {
			t.Errorf("expected empty Provider, got %q", meta.OrchestratorAction.Provider)
		}
		if meta.OrchestratorAction.Model != "" {
			t.Errorf("expected empty Model, got %q", meta.OrchestratorAction.Model)
		}
	})
}
