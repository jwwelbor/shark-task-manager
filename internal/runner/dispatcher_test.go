package runner

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// Compile-time interface satisfiability check (INT-F01-1)
// This causes a compile error if ClaudeDispatcher does not satisfy AgentDispatcher.
var _ AgentDispatcher = &ClaudeDispatcher{}

// TC-001: DispatchInput struct has all required fields
func TestDispatchInput_Fields(t *testing.T) {
	input := DispatchInput{
		Instruction:     "implement the feature",
		WorkingDir:      "/tmp/work",
		EntityKey:       "E07-F01-001",
		EntityType:      "task",
		Status:          "in_development",
		AgentType:       "developer",
		Model:           "claude-opus-4-5",
		MaxTurns:        10,
		AllowedTools:    []string{"Read", "Write"},
		DisallowedTools: []string{"Bash(rm*)"},
	}

	if input.Instruction != "implement the feature" {
		t.Errorf("Instruction = %q, want %q", input.Instruction, "implement the feature")
	}
	if input.WorkingDir != "/tmp/work" {
		t.Errorf("WorkingDir = %q, want %q", input.WorkingDir, "/tmp/work")
	}
	if input.EntityKey != "E07-F01-001" {
		t.Errorf("EntityKey = %q, want %q", input.EntityKey, "E07-F01-001")
	}
	if input.EntityType != "task" {
		t.Errorf("EntityType = %q, want %q", input.EntityType, "task")
	}
	if input.Status != "in_development" {
		t.Errorf("Status = %q, want %q", input.Status, "in_development")
	}
	if input.AgentType != "developer" {
		t.Errorf("AgentType = %q, want %q", input.AgentType, "developer")
	}
	if input.Model != "claude-opus-4-5" {
		t.Errorf("Model = %q, want %q", input.Model, "claude-opus-4-5")
	}
	if input.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want %d", input.MaxTurns, 10)
	}
	if len(input.AllowedTools) != 2 || input.AllowedTools[0] != "Read" || input.AllowedTools[1] != "Write" {
		t.Errorf("AllowedTools = %v, want [Read Write]", input.AllowedTools)
	}
	if len(input.DisallowedTools) != 1 || input.DisallowedTools[0] != "Bash(rm*)" {
		t.Errorf("DisallowedTools = %v, want [Bash(rm*)]", input.DisallowedTools)
	}
}

// TC-001 edge cases: zero-value fields are valid
func TestDispatchInput_ZeroValues(t *testing.T) {
	// Zero MaxTurns (means no limit), empty slices, empty model — all valid
	input := DispatchInput{
		Instruction: "test",
		EntityKey:   "E07-F01-001",
	}
	if input.MaxTurns != 0 {
		t.Errorf("expected zero MaxTurns, got %d", input.MaxTurns)
	}
	if input.AllowedTools != nil {
		t.Errorf("expected nil AllowedTools, got %v", input.AllowedTools)
	}
	if input.DisallowedTools != nil {
		t.Errorf("expected nil DisallowedTools, got %v", input.DisallowedTools)
	}
	if input.Model != "" {
		t.Errorf("expected empty Model, got %q", input.Model)
	}
}

// TC-002: DispatchResult struct has all required fields including Duration
func TestDispatchResult_Fields(t *testing.T) {
	result := &DispatchResult{
		ExitCode: 0,
		Stdout:   "agent output here",
		Stderr:   "",
		Duration: 3 * time.Second,
		Command:  `claude -p "instruction" --output-format json`,
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "agent output here" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "agent output here")
	}
	if result.Stderr != "" {
		t.Errorf("Stderr = %q, want empty", result.Stderr)
	}
	if result.Duration != 3*time.Second {
		t.Errorf("Duration = %v, want 3s", result.Duration)
	}
	if result.Command != `claude -p "instruction" --output-format json` {
		t.Errorf("Command = %q, unexpected", result.Command)
	}
}

// TC-002 edge cases
func TestDispatchResult_EdgeCases(t *testing.T) {
	// Negative exit code (process killed by signal)
	r1 := &DispatchResult{ExitCode: -1}
	if r1.ExitCode != -1 {
		t.Errorf("expected ExitCode -1, got %d", r1.ExitCode)
	}

	// Zero duration is valid
	r2 := &DispatchResult{Duration: 0}
	if r2.Duration != 0 {
		t.Errorf("expected zero Duration, got %v", r2.Duration)
	}

	// Both Stdout and Stderr can be empty
	r3 := &DispatchResult{ExitCode: 0}
	if r3.Stdout != "" || r3.Stderr != "" {
		t.Errorf("expected empty Stdout and Stderr")
	}
}

// TC-003: ToolNotFoundError message format
func TestToolNotFoundError_Message(t *testing.T) {
	tests := []struct {
		tool    string
		wantMsg string
	}{
		{"claude", "claude CLI not found on PATH. Install claude to continue."},
		{"codex", "codex CLI not found on PATH. Install codex to continue."},
		{"", " CLI not found on PATH. Install  to continue."},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			err := &ToolNotFoundError{Tool: tt.tool}
			if err.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// Negative case: must include tool name in message
func TestToolNotFoundError_MessageContainsTool(t *testing.T) {
	err := &ToolNotFoundError{Tool: "claude"}
	msg := err.Error()
	if msg == "not found" || msg == "" {
		t.Errorf("Error() must not be generic: %q", msg)
	}
	if !containsString(msg, "claude") {
		t.Errorf("Error() must contain tool name 'claude': %q", msg)
	}
}

// TC-004: ToolNotFoundError matchable with errors.As
func TestToolNotFoundError_ErrorsAs(t *testing.T) {
	err := &ToolNotFoundError{Tool: "claude"}
	wrapped := fmt.Errorf("dispatch failed: %w", err)

	var target *ToolNotFoundError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should match *ToolNotFoundError in wrapped error")
	}
	if target.Tool != "claude" {
		t.Errorf("target.Tool = %q, want %q", target.Tool, "claude")
	}
}

// TC-004 negative: wrong type does not match
func TestToolNotFoundError_ErrorsAs_WrongType(t *testing.T) {
	err := &ToolNotFoundError{Tool: "claude"}
	wrapped := fmt.Errorf("dispatch failed: %w", err)

	var target *AgentFailedError
	if errors.As(wrapped, &target) {
		t.Error("errors.As should NOT match *AgentFailedError for ToolNotFoundError")
	}
}

// TC-005: AgentFailedError message format includes exit code and stderr
func TestAgentFailedError_Message(t *testing.T) {
	err := &AgentFailedError{
		ExitCode: 1,
		Stdout:   "some output",
		Stderr:   "permission denied",
		Command:  "claude -p ...",
	}

	msg := err.Error()
	if msg == "" {
		t.Fatal("Error() must not be empty")
	}
	if !containsString(msg, "1") {
		t.Errorf("Error() must contain exit code '1': %q", msg)
	}
	if !containsString(msg, "permission denied") {
		t.Errorf("Error() must contain stderr 'permission denied': %q", msg)
	}
}

// TC-005 edge cases: various exit codes and empty stderr
func TestAgentFailedError_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		stderr   string
	}{
		{"exit code 2", 2, "misuse of shell builtins"},
		{"exit code 137 (SIGKILL)", 137, "killed"},
		{"empty stderr", 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &AgentFailedError{
				ExitCode: tt.exitCode,
				Stderr:   tt.stderr,
				Command:  "claude ...",
			}
			msg := err.Error()
			if msg == "" {
				t.Error("Error() must not be empty")
			}
			// Must contain the exit code
			exitCodeStr := fmt.Sprintf("%d", tt.exitCode)
			if !containsString(msg, exitCodeStr) {
				t.Errorf("Error() must contain exit code %q: got %q", exitCodeStr, msg)
			}
		})
	}
}

// TC-006: AgentFailedError matchable with errors.As
func TestAgentFailedError_ErrorsAs(t *testing.T) {
	err := &AgentFailedError{
		ExitCode: 1,
		Stderr:   "error",
		Command:  "claude ...",
	}
	wrapped := fmt.Errorf("stage failed: %w", err)

	var target *AgentFailedError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should match *AgentFailedError in wrapped error")
	}
	if target.ExitCode != 1 {
		t.Errorf("target.ExitCode = %d, want 1", target.ExitCode)
	}
}

// TC-006 negative: wrong type does not match
func TestAgentFailedError_ErrorsAs_WrongType(t *testing.T) {
	err := &AgentFailedError{ExitCode: 1, Command: "claude ..."}
	wrapped := fmt.Errorf("stage failed: %w", err)

	var target *ToolNotFoundError
	if errors.As(wrapped, &target) {
		t.Error("errors.As should NOT match *ToolNotFoundError for AgentFailedError")
	}
}

// INT-F01-2: DefaultDisallowedTools package variable is testable
func TestDefaultDisallowedTools(t *testing.T) {
	expectedTools := []string{
		"Bash(shark status advance*)",
		"Bash(shark task next-status*)",
		"Bash(shark status set*)",
		"Bash(shark task set-status*)",
		"Bash(shark feature next-status*)",
		"Bash(shark epic next-status*)",
	}

	if len(DefaultDisallowedTools) != 6 {
		t.Errorf("expected 6 default disallowed tools, got %d", len(DefaultDisallowedTools))
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range DefaultDisallowedTools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultDisallowedTools missing: %q", expected)
		}
	}
}

// containsString is a helper to check if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
