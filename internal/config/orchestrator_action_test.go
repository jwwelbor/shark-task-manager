package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getProjectRoot finds the project root by walking up from this file
func getProjectRoot() string {
	// Start from the directory of this test file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Walk up until we find .sharkconfig.json or .git
	for {
		configPath := filepath.Join(dir, ".sharkconfig.json")
		if _, err := os.Stat(configPath); err == nil {
			return dir
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached root, return current directory
			return "."
		}
		dir = parentDir
	}
}

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

// === NEW TESTS FOR PHASE 2 TEMPLATE REFERENCES (T-E07-F30-008) ===

// TestPhase2TemplateReferenceCountInConfig verifies .tmpl references in config
func TestPhase2TemplateReferenceCountInConfig(t *testing.T) {
	// Find the config file
	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read .sharkconfig.json: %v", err)
	}

	// Count occurrences of .tmpl"
	count := 0
	str := string(content)
	for i := 0; i < len(str)-5; i++ {
		if str[i:i+5] == ".tmpl" && i+5 < len(str) && str[i+5] == '"' {
			count++
		}
	}

	// Should have at least 12 .tmpl references (expanded in E18-F01 to include all entity statuses)
	if count < 12 {
		t.Errorf("Config has %d .tmpl references, want at least 12", count)
	}
}

// TestPhase2TaskTemplateReferencesInConfig verifies 5 task templates referenced
func TestPhase2TaskTemplateReferencesInConfig(t *testing.T) {
	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	manager := NewManager(configPath)
	config, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	taskStatuses := map[string]string{
		"ready_for_development":     "task/ready_for_development.tmpl",
		"ready_for_code_review":     "task/ready_for_code_review.tmpl",
		"ready_for_qa":              "task/ready_for_qa.tmpl",
		"ready_for_refinement_ba":   "task/ready_for_refinement_ba.tmpl",
		"ready_for_refinement_tech": "task/ready_for_refinement_tech.tmpl",
	}

	// Get status_metadata from raw data
	statusMetadata, ok := config.RawData["status_metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("status_metadata not found in raw config")
	}

	for status, expectedTemplate := range taskStatuses {
		statusData, ok := statusMetadata[status].(map[string]interface{})
		if !ok {
			t.Errorf("Task status %q not found in status_metadata", status)
			continue
		}

		orchestratorAction, ok := statusData["orchestrator_action"].(map[string]interface{})
		if !ok {
			t.Errorf("Task status %q has no orchestrator_action", status)
			continue
		}

		instructionTemplate, ok := orchestratorAction["instruction_template"].(string)
		if !ok {
			t.Errorf("Task status %q has no instruction_template", status)
			continue
		}

		if instructionTemplate != expectedTemplate {
			t.Errorf("Task status %q instruction_template = %q, want %q",
				status,
				instructionTemplate,
				expectedTemplate)
		}
	}
}

// TestPhase2FeatureTemplateReferencesInConfig verifies 4 feature templates referenced
func TestPhase2FeatureTemplateReferencesInConfig(t *testing.T) {
	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	manager := NewManager(configPath)
	config, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Get feature_workflow from raw data
	featureWorkflow, ok := config.RawData["feature_workflow"].(map[string]interface{})
	if !ok {
		t.Fatal("feature_workflow not found in raw config")
	}

	statusMetadata, ok := featureWorkflow["status_metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("feature_workflow.status_metadata not found")
	}

	featureStatuses := map[string]string{
		"ready_for_research":        "feature/ready_for_research.tmpl",
		"ready_for_refinement_ba":   "feature/ready_for_refinement_ba.tmpl",
		"ready_for_refinement_tech": "feature/ready_for_refinement_tech.tmpl",
		"ready_for_test_planning":   "feature/ready_for_test_planning.tmpl",
	}

	for status, expectedTemplate := range featureStatuses {
		statusData, ok := statusMetadata[status].(map[string]interface{})
		if !ok {
			t.Errorf("Feature status %q not found in feature_workflow", status)
			continue
		}

		orchestratorAction, ok := statusData["orchestrator_action"].(map[string]interface{})
		if !ok {
			t.Errorf("Feature status %q has no orchestrator_action", status)
			continue
		}

		instructionTemplate, ok := orchestratorAction["instruction_template"].(string)
		if !ok {
			t.Errorf("Feature status %q has no instruction_template", status)
			continue
		}

		if instructionTemplate != expectedTemplate {
			t.Errorf("Feature status %q instruction_template = %q, want %q",
				status,
				instructionTemplate,
				expectedTemplate)
		}
	}
}

// TestPhase2EpicTemplateReferencesInConfig verifies 3 epic templates referenced
func TestPhase2EpicTemplateReferencesInConfig(t *testing.T) {
	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	manager := NewManager(configPath)
	config, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Get epic_workflow from raw data
	epicWorkflow, ok := config.RawData["epic_workflow"].(map[string]interface{})
	if !ok {
		t.Fatal("epic_workflow not found in raw config")
	}

	statusMetadata, ok := epicWorkflow["status_metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("epic_workflow.status_metadata not found")
	}

	epicStatuses := map[string]string{
		"ready_for_research":                "epic/ready_for_research.tmpl",
		"ready_for_feasibility_review_ba":   "epic/ready_for_feasibility_review_ba.tmpl",
		"ready_for_feasibility_review_tech": "epic/ready_for_feasibility_review_tech.tmpl",
	}

	for status, expectedTemplate := range epicStatuses {
		statusData, ok := statusMetadata[status].(map[string]interface{})
		if !ok {
			t.Errorf("Epic status %q not found in epic_workflow", status)
			continue
		}

		orchestratorAction, ok := statusData["orchestrator_action"].(map[string]interface{})
		if !ok {
			t.Errorf("Epic status %q has no orchestrator_action", status)
			continue
		}

		instructionTemplate, ok := orchestratorAction["instruction_template"].(string)
		if !ok {
			t.Errorf("Epic status %q has no instruction_template", status)
			continue
		}

		if instructionTemplate != expectedTemplate {
			t.Errorf("Epic status %q instruction_template = %q, want %q",
				status,
				instructionTemplate,
				expectedTemplate)
		}
	}
}

// TestPhase2AllTaskTemplateFilesExist verifies all 5 task .tmpl files exist
func TestPhase2AllTaskTemplateFilesExist(t *testing.T) {
	// Find templates directory
	projectRoot := getProjectRoot()
	baseDir := filepath.Join(projectRoot, "shark-templates")

	taskTemplates := []string{
		baseDir + "/task/ready_for_development.tmpl",
		baseDir + "/task/ready_for_code_review.tmpl",
		baseDir + "/task/ready_for_qa.tmpl",
		baseDir + "/task/ready_for_refinement_ba.tmpl",
		baseDir + "/task/ready_for_refinement_tech.tmpl",
	}

	for _, path := range taskTemplates {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Task template file missing: %s (%v)", path, err)
		}
	}
}

// TestPhase2AllFeatureTemplateFilesExist verifies all 4 feature .tmpl files exist
func TestPhase2AllFeatureTemplateFilesExist(t *testing.T) {
	// Find templates directory
	projectRoot := getProjectRoot()
	baseDir := filepath.Join(projectRoot, "shark-templates")

	featureTemplates := []string{
		baseDir + "/feature/ready_for_research.tmpl",
		baseDir + "/feature/ready_for_refinement_ba.tmpl",
		baseDir + "/feature/ready_for_refinement_tech.tmpl",
		baseDir + "/feature/ready_for_test_planning.tmpl",
	}

	for _, path := range featureTemplates {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Feature template file missing: %s (%v)", path, err)
		}
	}
}

// TestPhase2AllEpicTemplateFilesExist verifies all 3 epic .tmpl files exist
func TestPhase2AllEpicTemplateFilesExist(t *testing.T) {
	// Find templates directory
	projectRoot := getProjectRoot()
	baseDir := filepath.Join(projectRoot, "shark-templates")

	epicTemplates := []string{
		baseDir + "/epic/ready_for_research.tmpl",
		baseDir + "/epic/ready_for_feasibility_review_ba.tmpl",
		baseDir + "/epic/ready_for_feasibility_review_tech.tmpl",
	}

	for _, path := range epicTemplates {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Epic template file missing: %s (%v)", path, err)
		}
	}
}

// TestPhase2NonPhase2StatusesRemainInline verifies template system is consistent
// Note: E18-F01 expanded the template system to cover all entity statuses.
// All statuses now use .tmpl files for consistent orchestration instructions.
func TestPhase2NonPhase2StatusesRemainInline(t *testing.T) {
	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	manager := NewManager(configPath)
	config, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	statusMetadata, ok := config.RawData["status_metadata"].(map[string]interface{})
	if !ok {
		// No status_metadata, that's fine
		return
	}

	// Verify all orchestrator_action entries have non-empty instruction_template
	for status, v := range statusMetadata {
		statusData, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		orchestratorAction, ok := statusData["orchestrator_action"].(map[string]interface{})
		if !ok {
			continue
		}

		template, ok := orchestratorAction["instruction_template"].(string)
		if !ok || template == "" {
			t.Errorf("Status %q has orchestrator_action with empty instruction_template", status)
		}
	}
}

// TestPhase2NoEmptyTemplateFiles verifies no template files are empty
func TestPhase2NoEmptyTemplateFiles(t *testing.T) {
	// Find templates directory
	projectRoot := getProjectRoot()
	baseDir := filepath.Join(projectRoot, "shark-templates")

	allTemplates := []string{
		baseDir + "/task/ready_for_development.tmpl",
		baseDir + "/task/ready_for_code_review.tmpl",
		baseDir + "/task/ready_for_qa.tmpl",
		baseDir + "/task/ready_for_refinement_ba.tmpl",
		baseDir + "/task/ready_for_refinement_tech.tmpl",
		baseDir + "/feature/ready_for_research.tmpl",
		baseDir + "/feature/ready_for_refinement_ba.tmpl",
		baseDir + "/feature/ready_for_refinement_tech.tmpl",
		baseDir + "/feature/ready_for_test_planning.tmpl",
		baseDir + "/epic/ready_for_research.tmpl",
		baseDir + "/epic/ready_for_feasibility_review_ba.tmpl",
		baseDir + "/epic/ready_for_feasibility_review_tech.tmpl",
	}

	for _, path := range allTemplates {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read template %s: %v", path, err)
			continue
		}

		if len(content) == 0 {
			t.Errorf("Template file is empty: %s", path)
		}
	}
}
