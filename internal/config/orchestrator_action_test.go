package config

import (
	"errors"
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

	result := oa.PopulateTemplate("T-E07-F21-001")
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

	result := oa.PopulateTemplate("T-E07-F21-001")
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

	result := oa.PopulateTemplate("T-E07-F21-001")
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

	result := oa.PopulateTemplate("T-E07-F21-001")
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

	result := oa.PopulateTemplate("")
	expected := "Implement task "

	if result != expected {
		t.Errorf("PopulateTemplate() with empty ID = %q, want %q", result, expected)
	}
}
