package commands

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/spf13/cobra"
)

// workflowCmd represents the workflow command group
var workflowCmd = &cobra.Command{
	Use:     "workflow",
	Short:   "Manage workflow configuration",
	GroupID: "setup",
	Long: `Workflow configuration operations including listing, validation, and migration.

The workflow system allows customizing task status transitions via .sharkconfig.json.

Examples:
  shark workflow list      Display configured workflow
  shark workflow validate  Validate workflow configuration`,
}

// workflowListCmd displays the configured workflow
var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "Display configured workflow",
	Long: `Display the configured status workflow from .sharkconfig.json.

Shows all statuses and their valid transitions, highlighting special statuses
(_start_ and _complete_). If no custom workflow is configured, displays the
default workflow.

Examples:
  shark workflow list         Display workflow (human-readable)
  shark workflow list --json  Display workflow (JSON format)`,
	RunE: runWorkflowList,
}

// workflowValidateCmd validates the workflow configuration
var workflowValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate workflow configuration",
	Long: `Validate the workflow configuration in .sharkconfig.json for correctness.

Checks all validation rules:
- Required special statuses (_start_, _complete_) are defined
- All status references in transitions are defined
- All statuses are reachable from _start_ statuses
- All statuses have a path to _complete_ statuses
- No circular references with no terminal path

Exit codes:
  0 - Configuration is valid
  2 - Configuration is invalid (specific errors displayed)

Examples:
  shark workflow validate         Validate configuration
  shark workflow validate --json  Validate with JSON output`,
	RunE: runWorkflowValidate,
}

func init() {
	cli.RootCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowValidateCmd)
}

// runWorkflowList implements the workflow list command
func runWorkflowList(cmd *cobra.Command, args []string) error {
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

	// If no custom workflow, use default
	if workflow == nil {
		workflow = config.DefaultWorkflow()
		if !cli.GlobalConfig.JSON {
			cli.Warning("No custom workflow configured in .sharkconfig.json, using default workflow")
		}
	}

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(workflow)
	}

	// Human-readable output
	return displayWorkflowHumanReadable(workflow)
}

// displayWorkflowHumanReadable displays the workflow in a human-readable format
func displayWorkflowHumanReadable(workflow *config.WorkflowConfig) error {
	// Display header
	fmt.Printf("Workflow Configuration (version: %s)\n\n", workflow.Version)

	// Display special statuses
	fmt.Println("Special Statuses:")
	if startStatuses, ok := workflow.SpecialStatuses[config.StartStatusKey]; ok && len(startStatuses) > 0 {
		fmt.Printf("  %s (entry points):  %s\n", config.StartStatusKey, strings.Join(startStatuses, ", "))
	}
	if completeStatuses, ok := workflow.SpecialStatuses[config.CompleteStatusKey]; ok && len(completeStatuses) > 0 {
		fmt.Printf("  %s (exit points): %s\n", config.CompleteStatusKey, strings.Join(completeStatuses, ", "))
	}
	fmt.Println()

	// Display status flow
	fmt.Println("Status Transitions:")

	// Sort statuses for consistent output
	statuses := make([]string, 0, len(workflow.StatusFlow))
	for status := range workflow.StatusFlow {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	for _, status := range statuses {
		transitions := workflow.StatusFlow[status]

		// Get metadata if available
		metadata, hasMetadata := workflow.StatusMetadata[status]

		// Display status with metadata
		statusDisplay := status
		if hasMetadata && metadata.Description != "" {
			statusDisplay = fmt.Sprintf("%s (%s)", status, metadata.Description)
		}

		// Display transitions
		if len(transitions) == 0 {
			fmt.Printf("  %s\n    → (terminal - no transitions)\n", statusDisplay)
		} else {
			fmt.Printf("  %s\n", statusDisplay)
			for _, nextStatus := range transitions {
				fmt.Printf("    → %s\n", nextStatus)
			}
		}

		// Display additional metadata if present
		if hasMetadata {
			var metaInfo []string
			if metadata.Phase != "" {
				metaInfo = append(metaInfo, fmt.Sprintf("phase: %s", metadata.Phase))
			}
			if len(metadata.AgentTypes) > 0 {
				metaInfo = append(metaInfo, fmt.Sprintf("agents: %s", strings.Join(metadata.AgentTypes, ", ")))
			}
			if metadata.Color != "" {
				metaInfo = append(metaInfo, fmt.Sprintf("color: %s", metadata.Color))
			}
			if len(metaInfo) > 0 {
				fmt.Printf("      [%s]\n", strings.Join(metaInfo, " | "))
			}
		}
		fmt.Println()
	}

	return nil
}

// levelValidationResult holds validation results for a single workflow level.
type levelValidationResult struct {
	Level       string `json:"level"`
	Valid       bool   `json:"valid"`
	Source      string `json:"source"` // "custom" or "default"
	Statuses    int    `json:"statuses"`
	Transitions int    `json:"transitions"`
	Error       string `json:"error,omitempty"`
}

// runWorkflowValidate implements the workflow validate command.
// Validates all three workflow levels (epic, feature, task).
func runWorkflowValidate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Context available for future use

	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load multi-level workflow (nil fields = using default)
	multi, err := config.LoadMultiLevelWorkflow(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}
	if multi == nil {
		multi = &config.MultiLevelWorkflow{}
	}

	// Validate each level
	levels := []struct {
		name     string
		raw      *config.WorkflowConfig // nil means default
		resolved *config.WorkflowConfig // always non-nil (with default fallback)
	}{
		{"epic", multi.Epic, multi.GetWorkflowForLevel("epic")},
		{"feature", multi.Feature, multi.GetWorkflowForLevel("feature")},
		{"task", multi.Task, multi.GetWorkflowForLevel("task")},
	}

	allValid := true
	var results []levelValidationResult

	for _, lvl := range levels {
		source := "default"
		if lvl.raw != nil {
			source = "custom"
		}

		validationErr := config.ValidateWorkflow(lvl.resolved)

		transitionCount := 0
		for _, transitions := range lvl.resolved.StatusFlow {
			transitionCount += len(transitions)
		}

		lr := levelValidationResult{
			Level:       lvl.name,
			Valid:       validationErr == nil,
			Source:      source,
			Statuses:    len(lvl.resolved.StatusFlow),
			Transitions: transitionCount,
		}

		if validationErr != nil {
			lr.Error = validationErr.Error()
			allValid = false
		}

		results = append(results, lr)
	}

	// Prepare combined result
	result := map[string]interface{}{
		"valid":       allValid,
		"config_path": configPath,
		"levels":      results,
	}

	// JSON output
	if cli.GlobalConfig.JSON {
		if !allValid {
			_ = cli.OutputJSON(result)
			return fmt.Errorf("validation failed")
		}
		return cli.OutputJSON(result)
	}

	// Human-readable output
	fmt.Println()
	for _, lr := range results {
		titleCase := cases.Title(language.English)
		if lr.Valid {
			fmt.Printf("  %s workflow: valid (%d statuses, %s)\n", titleCase.String(lr.Level), lr.Statuses, lr.Source)
		} else {
			fmt.Printf("  %s workflow: INVALID (%s)\n", titleCase.String(lr.Level), lr.Source)
			fmt.Printf("    Error: %s\n", lr.Error)
		}
	}
	fmt.Println()

	if allValid {
		cli.Success("All workflow levels are valid")
		return nil
	}

	cli.Error("One or more workflow levels have validation errors")
	return fmt.Errorf("workflow configuration is invalid")
}
