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
Use --level to validate only a specific entity level (epic, feature, task).

Exit codes:
  0 - Validation passed (or passed with warnings in non-strict mode)
  1 - Validation failed (errors found or warnings in --strict mode)

Examples:
  shark admin workflow validate-actions                  Validate all levels
  shark admin workflow validate-actions --level=task     Validate only task workflow
  shark admin workflow validate-actions --level=epic     Validate only epic workflow
  shark admin workflow validate-actions --strict         Fail on any warnings
  shark admin workflow validate-actions --json           JSON output`,
	RunE: runWorkflowValidateActions,
}

// Flags
var (
	validateActionsStrict bool
	validateActionsLevel  string
)

func init() {
	workflowCmd.AddCommand(workflowValidateActionsCmd)
	workflowValidateActionsCmd.Flags().BoolVar(&validateActionsStrict, "strict", false,
		"Fail with exit code 1 if any status lacks an orchestrator action")
	workflowValidateActionsCmd.Flags().StringVar(&validateActionsLevel, "level", "",
		"Filter by entity level (epic, feature, task)")
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

// MultiLevelValidationReport contains validation results for all entity levels
type MultiLevelValidationReport struct {
	Valid         bool              `json:"valid"`
	StrictMode    bool              `json:"strict_mode"`
	EpicReport    *ValidationReport `json:"epic_report,omitempty"`
	FeatureReport *ValidationReport `json:"feature_report,omitempty"`
	TaskReport    *ValidationReport `json:"task_report,omitempty"`
	BugReport     *ValidationReport `json:"bug_report,omitempty"`
	ChangeReport  *ValidationReport `json:"change_report,omitempty"`
}

// runWorkflowValidateActions implements the validate-actions command
func runWorkflowValidateActions(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Context available for future use

	// Validate --level flag
	if validateActionsLevel != "" {
		validLevels := map[string]bool{"epic": true, "feature": true, "task": true, "bug": true, "change": true}
		if !validLevels[validateActionsLevel] {
			return fmt.Errorf("invalid level %q: must be one of: epic, feature, task, bug, change", validateActionsLevel)
		}
	}

	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load multi-level workflow config
	multiWorkflow := config.LoadMultiLevelWorkflowOrDefault(configPath)

	// If --level is specified, validate only that level (backward compat)
	if validateActionsLevel != "" {
		workflow := multiWorkflow.GetWorkflowForLevel(validateActionsLevel)
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

	// No --level specified: validate all levels
	multiReport := &MultiLevelValidationReport{
		Valid:      true,
		StrictMode: validateActionsStrict,
	}

	// Validate epic workflow
	epicWorkflow := multiWorkflow.GetWorkflowForLevel("epic")
	multiReport.EpicReport = validateWorkflowActions(epicWorkflow, validateActionsStrict)
	if !multiReport.EpicReport.Valid {
		multiReport.Valid = false
	}

	// Validate feature workflow
	featureWorkflow := multiWorkflow.GetWorkflowForLevel("feature")
	multiReport.FeatureReport = validateWorkflowActions(featureWorkflow, validateActionsStrict)
	if !multiReport.FeatureReport.Valid {
		multiReport.Valid = false
	}

	// Validate task workflow
	taskWorkflow := multiWorkflow.GetWorkflowForLevel("task")
	multiReport.TaskReport = validateWorkflowActions(taskWorkflow, validateActionsStrict)
	if !multiReport.TaskReport.Valid {
		multiReport.Valid = false
	}

	// Validate bug workflow
	bugWorkflow := multiWorkflow.GetWorkflowForLevel("bug")
	multiReport.BugReport = validateWorkflowActions(bugWorkflow, validateActionsStrict)
	if !multiReport.BugReport.Valid {
		multiReport.Valid = false
	}

	// Validate change workflow
	changeWorkflow := multiWorkflow.GetWorkflowForLevel("change")
	multiReport.ChangeReport = validateWorkflowActions(changeWorkflow, validateActionsStrict)
	if !multiReport.ChangeReport.Valid {
		multiReport.Valid = false
	}

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(multiReport)
	}

	// Human-readable output
	displayMultiLevelValidationReport(multiReport)

	// Determine exit code
	if !multiReport.Valid {
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

	// Soft-validate dispatch fields: spawn_agent / check_or_resume should set
	// `provider` so consumers of `shark next` don't have to guess. Empty is
	// still legal at the schema layer (run controller may default), so this is
	// a warning, not an error.
	dispatches := metadata.OrchestratorAction.Action == config.ActionSpawnAgent ||
		metadata.OrchestratorAction.Action == config.ActionCheckOrResume
	if dispatches && strings.TrimSpace(metadata.OrchestratorAction.Provider) == "" {
		result.Valid = false
		result.Severity = "warning"
		result.Message = fmt.Sprintf("%s action missing provider", metadata.OrchestratorAction.Action)
		result.Recommendation = "Set provider (e.g. anthropic, openai) so `shark next` returns a populated provider field"
		// Still populate action details so the report is informative.
		result.ActionType = metadata.OrchestratorAction.Action
		result.AgentType = metadata.OrchestratorAction.AgentType
		result.Skills = metadata.OrchestratorAction.Skills
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

// displayMultiLevelValidationReport displays the multi-level validation report in human-readable format
func displayMultiLevelValidationReport(report *MultiLevelValidationReport) {
	fmt.Println("Validating workflow orchestrator actions...")
	fmt.Println()

	// Display epic workflow results
	if report.EpicReport != nil {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("--- Epic Workflow ---")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		displaySingleLevelResults(report.EpicReport)
		fmt.Println()
	}

	// Display feature workflow results
	if report.FeatureReport != nil {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("--- Feature Workflow ---")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		displaySingleLevelResults(report.FeatureReport)
		fmt.Println()
	}

	// Display task workflow results
	if report.TaskReport != nil {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("--- Task Workflow ---")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		displaySingleLevelResults(report.TaskReport)
		fmt.Println()
	}

	// Display bug workflow results
	if report.BugReport != nil {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("--- Bug Workflow ---")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		displaySingleLevelResults(report.BugReport)
		fmt.Println()
	}

	// Display change workflow results
	if report.ChangeReport != nil {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("--- Change Workflow ---")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		displaySingleLevelResults(report.ChangeReport)
		fmt.Println()
	}

	// Display overall summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Overall Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if report.EpicReport != nil {
		epicStatus := "VALID"
		if !report.EpicReport.Valid {
			epicStatus = "INVALID"
		}
		fmt.Printf("Epic:    %s (%d actions validated)\n", epicStatus, report.EpicReport.TotalStatuses)
	}

	if report.FeatureReport != nil {
		featureStatus := "VALID"
		if !report.FeatureReport.Valid {
			featureStatus = "INVALID"
		}
		fmt.Printf("Feature: %s (%d actions validated)\n", featureStatus, report.FeatureReport.TotalStatuses)
	}

	if report.TaskReport != nil {
		taskStatus := "VALID"
		if !report.TaskReport.Valid {
			taskStatus = "INVALID"
		}
		fmt.Printf("Task:    %s (%d actions validated)\n", taskStatus, report.TaskReport.TotalStatuses)
	}

	if report.BugReport != nil {
		bugStatus := "VALID"
		if !report.BugReport.Valid {
			bugStatus = "INVALID"
		}
		fmt.Printf("Bug:     %s (%d actions validated)\n", bugStatus, report.BugReport.TotalStatuses)
	}

	if report.ChangeReport != nil {
		changeStatus := "VALID"
		if !report.ChangeReport.Valid {
			changeStatus = "INVALID"
		}
		fmt.Printf("Change:  %s (%d actions validated)\n", changeStatus, report.ChangeReport.TotalStatuses)
	}

	fmt.Println()

	if report.Valid {
		cli.Success("Overall: VALID")
	} else {
		cli.Error("Overall: INVALID")
		if report.StrictMode {
			fmt.Println("(Failed with --strict mode)")
		}
	}
}

// displaySingleLevelResults displays validation results for a single level (without header/summary)
func displaySingleLevelResults(report *ValidationReport) {
	// Show note if no actions configured
	if report.TotalStatuses == 0 {
		fmt.Println("  (Using default workflow with no custom orchestrator actions)")
		return
	}

	// Display results
	for _, result := range report.Results {
		if result.Valid && result.Severity == "" {
			// Valid status
			fmt.Printf("  ✅ Status \"%s\": Valid\n", result.Status)
			if result.ActionType != "" {
				fmt.Printf("     - Action: %s\n", result.ActionType)
				if result.AgentType != "" {
					fmt.Printf("     - Agent: %s\n", result.AgentType)
				}
				if len(result.Skills) > 0 {
					skillsList := strings.Join(result.Skills, ", ")
					if len(skillsList) > 50 {
						skillsList = skillsList[:50] + "..."
					}
					fmt.Printf("     - Skills: %s\n", skillsList)
				}
			} else {
				// No action but valid (non-actionable status)
				fmt.Printf("     - No orchestrator action (not actionable)\n")
			}
		} else if result.Severity == "warning" {
			// Warning
			fmt.Printf("  ⚠️  Status \"%s\": Missing orchestrator_action\n", result.Status)
			fmt.Printf("     - %s\n", result.Message)
			if result.Recommendation != "" {
				fmt.Printf("     - Recommendation: %s\n", result.Recommendation)
			}
		} else if result.Severity == "error" {
			// Error
			fmt.Printf("  ❌ Status \"%s\": Invalid orchestrator_action\n", result.Status)
			fmt.Printf("     - %s\n", result.Message)
		}
	}

	// Brief summary line for this level
	fmt.Printf("\n  Summary: %d valid, %d warnings, %d errors\n",
		report.ValidCount, report.WarningCount, report.ErrorCount)
}
