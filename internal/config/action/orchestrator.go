package action

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// OrchestratorAction defines the action to take when a task enters a status
type OrchestratorAction struct {
	// Action specifies the type of orchestrator action to perform
	// Valid values: spawn_agent, pause, wait_for_triage, archive, advance_status, check_or_resume
	Action string `json:"action" yaml:"action"`

	// AgentType specifies the type of agent to spawn (required for spawn_agent action)
	AgentType string `json:"agent_type,omitempty" yaml:"agent_type,omitempty"`

	// Provider specifies the AI provider to use for dispatch (e.g., "anthropic", "openai").
	// Optional — when empty, the run controller defaults to "anthropic" (Claude).
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Model specifies the model override for the dispatched agent (e.g., "o3", "claude-opus-4-5").
	// Optional — when empty, the agent uses its default model.
	Model string `json:"model,omitempty" yaml:"model,omitempty"`

	// Effort specifies the reasoning-effort override for the dispatched agent.
	// Valid values: low, medium, high, xhigh (case-insensitive).
	// Optional — when empty, the host uses its default effort level.
	Effort string `json:"effort,omitempty" yaml:"effort,omitempty"`

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
	ActionAdvanceStatus = "advance_status"
	ActionCheckOrResume = "check_or_resume"
	ActionCascade       = "cascade"
)

// ValidActionTypes defines the allowed action types
var ValidActionTypes = []string{
	ActionSpawnAgent,
	ActionPause,
	ActionWaitForTriage,
	ActionArchive,
	ActionAdvanceStatus,
	ActionCheckOrResume,
	ActionCascade,
}

// Reasoning-effort levels a step/action may declare for the dispatched agent.
const (
	EffortLow    = "low"
	EffortMedium = "medium"
	EffortHigh   = "high"
	EffortXHigh  = "xhigh"
)

// ValidEffortLevels defines the allowed values for OrchestratorAction.Effort.
var ValidEffortLevels = []string{EffortLow, EffortMedium, EffortHigh, EffortXHigh}

// Validate validates the OrchestratorAction configuration
func (oa *OrchestratorAction) Validate() error {
	// Check action type is valid
	if !stringSliceContains(ValidActionTypes, oa.Action) {
		return fmt.Errorf("invalid action type: %s (must be one of: %s)",
			oa.Action, strings.Join(ValidActionTypes, ", "))
	}

	// instruction_template is always required
	if strings.TrimSpace(oa.InstructionTemplate) == "" {
		return errors.New("instruction_template is required")
	}

	// effort, when set, must be one of the recognized levels
	if oa.Effort != "" && !stringSliceContains(ValidEffortLevels, strings.ToLower(oa.Effort)) {
		return fmt.Errorf("invalid effort: %s (must be one of: %s)",
			oa.Effort, strings.Join(ValidEffortLevels, ", "))
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
			SuggestedFix: "Add instruction_template with placeholders (e.g., {id}, {title}, {status}, {file_path})",
		}
	}

	// 2b. Validate effort enum, when set
	if oa.Effort != "" && !stringSliceContains(ValidEffortLevels, strings.ToLower(oa.Effort)) {
		validEffortsStr := strings.Join(ValidEffortLevels, ", ")
		return &OrchestratorValidationError{
			StatusName:   statusName,
			FieldName:    "effort",
			Problem:      fmt.Sprintf("Invalid effort \"%s\"", oa.Effort),
			SuggestedFix: fmt.Sprintf("Use one of: %s", validEffortsStr),
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

// ToPopulatedAction constructs a PopulatedAction from this OrchestratorAction,
// populating the instruction template with the given placeholder values.
func (oa *OrchestratorAction) ToPopulatedAction(placeholders map[string]string) *PopulatedAction {
	return &PopulatedAction{
		Action:      oa.Action,
		AgentType:   oa.AgentType,
		Provider:    oa.Provider,
		Model:       oa.Model,
		Effort:      strings.ToLower(strings.TrimSpace(oa.Effort)),
		Skills:      oa.Skills,
		Instruction: oa.PopulateTemplate(placeholders),
	}
}

// PopulateTemplate replaces template placeholders with actual values from the vars map.
// Each key in vars corresponds to a placeholder name (without braces).
// For example, vars["title"] replaces all occurrences of {title} in the template.
// Unknown placeholders are left unchanged in the template.
//
// Detection Logic:
//   - If InstructionTemplate ends with ".tmpl" or ".md", uses OrchestratorRenderer.Render() (template engine)
//   - Otherwise uses legacy string replacement (inline templates)
//   - Template rendering errors log a warning and return empty string (graceful degradation)
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
	// Detect file-reference instruction templates. Shark 2.0 uses .md;
	// legacy workflows used .tmpl. Either extension is routed to the engine.
	if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") || strings.HasSuffix(oa.InstructionTemplate, ".md") {
		engine := templates.GetOrchestratorEngine()
		rendered, err := engine.Render(oa.InstructionTemplate, vars)
		if err != nil {
			// Log error but gracefully degrade - return empty string
			// This allows workflows to continue even if template rendering fails
			slog.Error("template rendering failed", "template", oa.InstructionTemplate, "error", err)
			return ""
		}
		return rendered
	}

	// UNCHANGED: Legacy string replacement for inline templates
	if len(vars) == 0 {
		return oa.InstructionTemplate
	}

	replacements := make([]string, 0, 2*len(vars))
	for key, value := range vars {
		replacements = append(replacements, "{"+key+"}", value)
	}

	return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
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
func validateTemplateSyntax(tmpl string) []string {
	warnings := []string{}

	// Check if template contains at least one placeholder (any {word} pattern)
	placeholders := extractPlaceholders(tmpl)
	if len(placeholders) == 0 {
		warnings = append(warnings, "Template does not contain any placeholder (e.g., {id}, {title}, {status})")
	}

	// Check for malformed placeholders (unclosed brace)
	if strings.Contains(tmpl, "{") && !strings.Contains(tmpl, "}") {
		warnings = append(warnings, "Malformed placeholder: unclosed brace {")
	}

	// Check maximum length
	if len(tmpl) > 2000 {
		warnings = append(warnings, "Template exceeds 2000 character limit")
	}

	return warnings
}

// extractPlaceholders extracts all {placeholder} patterns from a template string
// Returns a slice of placeholders found (e.g., ["{task_id}", "{epic_id}"])
func extractPlaceholders(tmpl string) []string {
	re := regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)
	matches := re.FindAllString(tmpl, -1)
	return matches
}

// ValidateAllOrchestratorActions validates all orchestrator actions in status metadata.
// The input map is keyed by status name and contains the OrchestratorAction pointer for each status
// (nil if no action is defined for that status).
// Returns a slice of OrchestratorValidationError for all invalid actions (empty if all valid).
func ValidateAllOrchestratorActions(actionsByStatus map[string]*OrchestratorAction) []*OrchestratorValidationError {
	var errs []*OrchestratorValidationError

	for statusName, action := range actionsByStatus {
		if action != nil {
			if err := action.ValidateWithContext(statusName); err != nil {
				if valErr, ok := err.(*OrchestratorValidationError); ok {
					errs = append(errs, valErr)
				}
			}
		}
	}

	return errs
}
