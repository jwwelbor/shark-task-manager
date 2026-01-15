package config

import (
	"errors"
	"fmt"
	"regexp"
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

// ValidateWithContext validates the OrchestratorAction with status context for detailed error messages
// statusName is the name of the status (e.g., "ready_for_development") for error context
func (oa *OrchestratorAction) ValidateWithContext(statusName string) error {
	// 1. Validate action enum
	if !stringSliceContains(ValidActionTypes, oa.Action) {
		validActionsStr := strings.Join(ValidActionTypes, ", ")
		return &OrchestratorValidationError{
			StatusName:   statusName,
			FieldName:    "action",
			Problem:      fmt.Sprintf("Invalid action type \"%s\"", oa.Action),
			SuggestedFix: fmt.Sprintf("Use one of: %s", validActionsStr),
		}
	}

	// 2. Validate instruction_template (required for all actions)
	if strings.TrimSpace(oa.InstructionTemplate) == "" {
		return &OrchestratorValidationError{
			StatusName:   statusName,
			FieldName:    "instruction_template",
			Problem:      "Missing required field",
			SuggestedFix: "Add instruction_template with {task_id} placeholder",
		}
	}

	// 3. Validate spawn_agent specific requirements
	if oa.Action == ActionSpawnAgent {
		if strings.TrimSpace(oa.AgentType) == "" {
			return &OrchestratorValidationError{
				StatusName:   statusName,
				FieldName:    "agent_type",
				Problem:      "Missing required field for spawn_agent action",
				SuggestedFix: "Add agent_type (e.g., \"developer\", \"business-analyst\")",
			}
		}

		if len(oa.Skills) == 0 {
			return &OrchestratorValidationError{
				StatusName:   statusName,
				FieldName:    "skills",
				Problem:      "Empty or missing skills array for spawn_agent action",
				SuggestedFix: "Add at least one skill to skills array",
			}
		}

		// Check for empty skill strings
		for i, skill := range oa.Skills {
			if strings.TrimSpace(skill) == "" {
				return &OrchestratorValidationError{
					StatusName:   statusName,
					FieldName:    fmt.Sprintf("skills[%d]", i),
					Problem:      "Empty skill string in skills array",
					SuggestedFix: "Remove empty skill or provide skill name",
				}
			}
		}
	}

	// 4. Validate template syntax (warnings, not errors - log but don't fail)
	// Note: In a real implementation, these would be logged as warnings
	// For now, we just validate the syntax but don't fail
	_ = validateTemplateSyntax(oa.InstructionTemplate)

	return nil
}

// PopulateTemplate replaces template variables with actual values
func (oa *OrchestratorAction) PopulateTemplate(taskID string) string {
	return strings.Replace(oa.InstructionTemplate, "{task_id}", taskID, -1)
}

// sliceContains checks if a string slice contains a target string (deprecated, use contains)
func sliceContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// stringSliceContains checks if a string slice contains a target string
func stringSliceContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// validateTemplateSyntax validates template syntax and returns warnings
// Returns a slice of warning messages (empty if no warnings)
func validateTemplateSyntax(template string) []string {
	warnings := []string{}

	// Check for {task_id} placeholder
	if !strings.Contains(template, "{task_id}") {
		warnings = append(warnings, "Template does not contain {task_id} placeholder")
	}

	// Check for malformed placeholders (unclosed brace)
	if strings.Contains(template, "{") && !strings.Contains(template, "}") {
		warnings = append(warnings, "Malformed placeholder: unclosed brace {")
	}

	// Extract and validate placeholders
	placeholders := extractPlaceholders(template)
	knownPlaceholders := map[string]bool{
		"{task_id}": true,
	}

	for _, placeholder := range placeholders {
		if !knownPlaceholders[placeholder] {
			warnings = append(warnings, fmt.Sprintf("Unknown placeholder %s (currently only {task_id} supported)", placeholder))
		}
	}

	// Check maximum length
	if len(template) > 2000 {
		warnings = append(warnings, "Template exceeds 2000 character limit")
	}

	return warnings
}

// extractPlaceholders extracts all {placeholder} patterns from a template string
// Returns a slice of placeholders found (e.g., ["{task_id}", "{epic_id}"])
func extractPlaceholders(template string) []string {
	re := regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)
	matches := re.FindAllString(template, -1)
	return matches
}

// ValidateAllOrchestratorActions validates all orchestrator actions in status metadata
// Returns a slice of OrchestratorValidationError for all invalid actions (empty if all valid)
func ValidateAllOrchestratorActions(statusMetadata map[string]StatusMetadata) []*OrchestratorValidationError {
	var errors []*OrchestratorValidationError

	for statusName, metadata := range statusMetadata {
		if metadata.OrchestratorAction != nil {
			if err := metadata.OrchestratorAction.ValidateWithContext(statusName); err != nil {
				if valErr, ok := err.(*OrchestratorValidationError); ok {
					errors = append(errors, valErr)
				}
			}
		}
	}

	return errors
}
