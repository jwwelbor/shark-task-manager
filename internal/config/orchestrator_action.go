package config

import (
	"errors"
	"fmt"
	"strings"
)

// OrchestratorAction defines the action to take when a task enters a status
type OrchestratorAction struct {
	// Action specifies the type of orchestrator action to perform
	// Valid values: spawn_agent, pause, wait_for_triage, archive
	Action string `json:"action" yaml:"action"`

	// AgentType specifies the type of agent to spawn (required for spawn_agent action)
	AgentType string `json:"agent_type,omitempty" yaml:"agent_type,omitempty"`

	// Skills lists the skills required for the agent (required for spawn_agent action)
	Skills []string `json:"skills,omitempty" yaml:"skills,omitempty"`

	// InstructionTemplate contains the template string with {task_id} placeholder
	// This field is required for all action types
	InstructionTemplate string `json:"instruction_template" yaml:"instruction_template"`
}

// Action type constants
const (
	ActionSpawnAgent    = "spawn_agent"
	ActionPause         = "pause"
	ActionWaitForTriage = "wait_for_triage"
	ActionArchive       = "archive"
)

// ValidActionTypes defines the allowed action types
var ValidActionTypes = []string{
	ActionSpawnAgent,
	ActionPause,
	ActionWaitForTriage,
	ActionArchive,
}

// Validate validates the OrchestratorAction configuration
func (oa *OrchestratorAction) Validate() error {
	// Check action type is valid
	if !sliceContains(ValidActionTypes, oa.Action) {
		return fmt.Errorf("invalid action type: %s (must be one of: %s)",
			oa.Action, strings.Join(ValidActionTypes, ", "))
	}

	// instruction_template is always required
	if strings.TrimSpace(oa.InstructionTemplate) == "" {
		return errors.New("instruction_template is required")
	}

	// spawn_agent requires agent_type and skills
	if oa.Action == ActionSpawnAgent {
		if strings.TrimSpace(oa.AgentType) == "" {
			return errors.New("agent_type is required for spawn_agent action")
		}
		if len(oa.Skills) == 0 {
			return errors.New("skills array is required and must not be empty for spawn_agent action")
		}
	}

	return nil
}

// PopulateTemplate replaces template variables with actual values
func (oa *OrchestratorAction) PopulateTemplate(taskID string) string {
	return strings.Replace(oa.InstructionTemplate, "{task_id}", taskID, -1)
}

// sliceContains checks if a string slice contains a target string
func sliceContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
