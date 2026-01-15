package config

import (
	"fmt"
	"strings"
)

// ValidationError is a general validation error type (alias to OrchestratorValidationError for backward compat)
type ValidationError = OrchestratorValidationError

// OrchestratorValidationError provides detailed context for orchestrator action configuration errors
type OrchestratorValidationError struct {
	StatusName   string // Which status has the error (e.g., "ready_for_development")
	FieldName    string // Which field is invalid (e.g., "action", "agent_type")
	Problem      string // What's wrong (e.g., "Invalid action type \"spawn-agent\"")
	SuggestedFix string // How to fix it (e.g., "Use one of: spawn_agent, pause, wait_for_triage, archive")
}

// Error implements the error interface
func (ve *OrchestratorValidationError) Error() string {
	lines := []string{
		fmt.Sprintf("Error: Invalid orchestrator_action in status '%s'", ve.StatusName),
		fmt.Sprintf("  Field: %s", ve.FieldName),
		fmt.Sprintf("  Problem: %s", ve.Problem),
		fmt.Sprintf("  Fix: %s", ve.SuggestedFix),
	}
	return strings.Join(lines, "\n")
}
