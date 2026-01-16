package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/spf13/cobra"
)

// workflowValidateActionsCmd validates all orchestrator actions in the workflow
var workflowValidateActionsCmd = &cobra.Command{
	Use:   "validate-actions",
	Short: "Validate workflow orchestrator actions",
	Long: `Validate that all orchestrator actions in the workflow configuration are properly defined.

This command checks:
- Action schema correctness (valid action types, required fields)
- Completeness (actionable statuses have actions defined)
- spawn_agent actions have required agent_type and skills
- instruction_templates are non-empty and syntactically valid

Use --strict to fail on warnings (any missing actions).

Exit codes:
  0 - Validation passed (or passed with warnings in non-strict mode)
  1 - Validation failed (errors found or warnings in --strict mode)

Examples:
  shark workflow validate-actions         Validate with warnings
  shark workflow validate-actions --strict Fail on any warnings
  shark workflow validate-actions --json  JSON output`,
	RunE: runWorkflowValidateActions,
}

// Flags
var (
	validateActionsStrict bool
)

func init() {
	workflowCmd.AddCommand(workflowValidateActionsCmd)
	workflowValidateActionsCmd.Flags().BoolVar(&validateActionsStrict, "strict", false,
		"Fail with exit code 1 if any status lacks an orchestrator action")
}

// ValidationReport contains the validation results
type ValidationReport struct {
	Valid         bool                     `json:"valid"`
	StrictMode    bool                     `json:"strict_mode"`
	TotalStatuses int                      `json:"total_statuses"`
	ValidCount    int                      `json:"valid_count"`
	WarningCount  int                      `json:"warning_count"`
	ErrorCount    int                      `json:"error_count"`
	Results       []StatusValidationResult `json:"results"`
}

// StatusValidationResult contains validation result for a single status
type StatusValidationResult struct {
	Status         string   `json:"status"`
	Valid          bool     `json:"valid"`
	Severity       string   `json:"severity,omitempty"` // "error" or "warning"
	Message        string   `json:"message,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	ActionType     string   `json:"action_type,omitempty"`
	AgentType      string   `json:"agent_type,omitempty"`
	Skills         []string `json:"skills,omitempty"`
}

// runWorkflowValidateActions implements the validate-actions command
func runWorkflowValidateActions(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Context available for future use

	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load workflow config
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// If no custom workflow, validate default
	if workflow == nil {
		workflow = config.DefaultWorkflow()
		if !cli.GlobalConfig.JSON {
			cli.Warning("No custom workflow configured in .sharkconfig.json, validating default workflow")
		}
	}

	// Perform validation
	report := validateWorkflowActions(workflow, validateActionsStrict)

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(report)
	}

	// Human-readable output
	displayValidationReport(report)

	// Determine exit code
	if !report.Valid {
		os.Exit(1)
	}

	return nil
}

// validateWorkflowActions validates all orchestrator actions in the workflow
func validateWorkflowActions(workflow *config.WorkflowConfig, strict bool) *ValidationReport {
	report := &ValidationReport{
		StrictMode: strict,
		Results:    make([]StatusValidationResult, 0),
	}

	// Get all statuses from status_metadata (ordered for consistency)
	statuses := make([]string, 0, len(workflow.StatusMetadata))
	for status := range workflow.StatusMetadata {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	// Validate each status
	for _, status := range statuses {
		metadata := workflow.StatusMetadata[status]
		result := validateStatusAction(status, &metadata, strict)
		report.Results = append(report.Results, result)

		if result.Severity == "error" {
			report.ErrorCount++
		} else if result.Severity == "warning" {
			report.WarningCount++
		} else if result.Valid {
			report.ValidCount++
		}
	}

	report.TotalStatuses = len(statuses)
	report.Valid = report.ErrorCount == 0 && (!strict || report.WarningCount == 0)

	return report
}

// validateStatusAction validates a single status's orchestrator_action
func validateStatusAction(status string, metadata *config.StatusMetadata, strict bool) StatusValidationResult {
	result := StatusValidationResult{
		Status: status,
		Valid:  true,
	}

	// No orchestrator_action defined
	if metadata.OrchestratorAction == nil {
		isActionable := strings.HasPrefix(status, "ready_for_")

		if isActionable {
			result.Valid = false
			result.Severity = "warning"
			result.Message = "Missing orchestrator_action (actionable status)"
			result.Recommendation = "Add spawn_agent or wait_for_triage action"
		} else if strict {
			result.Valid = false
			result.Severity = "warning"
			result.Message = "Missing orchestrator_action"
		}

		return result
	}

	// Validate orchestrator_action schema using the existing validator
	if err := metadata.OrchestratorAction.ValidateWithContext(status); err != nil {
		result.Valid = false
		result.Severity = "error"
		result.Message = err.Error()
		return result
	}

	// Valid - populate action details
	result.ActionType = metadata.OrchestratorAction.Action
	result.AgentType = metadata.OrchestratorAction.AgentType
	result.Skills = metadata.OrchestratorAction.Skills

	return result
}

// displayValidationReport displays the validation report in human-readable format
func displayValidationReport(report *ValidationReport) {
	fmt.Println("Validating workflow configuration...")

	// Display results
	for _, result := range report.Results {
		if result.Valid && result.Severity == "" {
			// Valid status
			fmt.Printf("✅ Status \"%s\": Valid\n", result.Status)
			if result.ActionType != "" {
				fmt.Printf("   - Action: %s\n", result.ActionType)
				if result.AgentType != "" {
					fmt.Printf("   - Agent: %s\n", result.AgentType)
				}
				if len(result.Skills) > 0 {
					skillsList := strings.Join(result.Skills, ", ")
					if len(skillsList) > 50 {
						skillsList = skillsList[:50] + "..."
					}
					fmt.Printf("   - Skills: %s\n", skillsList)
				}
			} else {
				// No action but valid (non-actionable status)
				fmt.Printf("   - No orchestrator action (not actionable)\n")
			}
		} else if result.Severity == "warning" {
			// Warning
			fmt.Printf("⚠️  Status \"%s\": Missing orchestrator_action\n", result.Status)
			fmt.Printf("   - %s\n", result.Message)
			if result.Recommendation != "" {
				fmt.Printf("   - Recommendation: %s\n", result.Recommendation)
			}
		} else if result.Severity == "error" {
			// Error
			fmt.Printf("❌ Status \"%s\": Invalid orchestrator_action\n", result.Status)
			fmt.Printf("   - %s\n", result.Message)
		}

		fmt.Println()
	}

	// Display summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Validation Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total statuses: %d\n", report.TotalStatuses)
	fmt.Printf("Valid: %d\n", report.ValidCount)
	if report.WarningCount > 0 {
		fmt.Printf("Warnings: %d\n", report.WarningCount)
	}
	if report.ErrorCount > 0 {
		fmt.Printf("Errors: %d\n", report.ErrorCount)
	}
	fmt.Println()

	if report.Valid {
		cli.Success("Validation passed")
	} else {
		if report.ErrorCount > 0 {
			cli.Error("Validation failed with errors")
		} else {
			cli.Warning("Validation completed with warnings")
			if !report.StrictMode {
				fmt.Println("Run with --strict to fail on warnings")
			}
		}
	}
}
