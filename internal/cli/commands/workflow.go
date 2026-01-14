package commands

import (
	"fmt"
	"sort"
	"strings"

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

// runWorkflowValidate implements the workflow validate command
func runWorkflowValidate(cmd *cobra.Command, args []string) error {
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

	// Validate workflow
	validationErr := config.ValidateWorkflow(workflow)

	// Prepare validation result
	result := map[string]interface{}{
		"valid":       validationErr == nil,
		"config_path": configPath,
	}

	if validationErr == nil {
		// Count statistics
		statusCount := len(workflow.StatusFlow)
		startCount := len(workflow.SpecialStatuses[config.StartStatusKey])
		completeCount := len(workflow.SpecialStatuses[config.CompleteStatusKey])

		// Count total transitions
		transitionCount := 0
		for _, transitions := range workflow.StatusFlow {
			transitionCount += len(transitions)
		}

		result["statistics"] = map[string]interface{}{
			"statuses":          statusCount,
			"transitions":       transitionCount,
			"start_statuses":    startCount,
			"complete_statuses": completeCount,
		}

		// JSON output
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		// Human-readable output
		cli.Success("✓ Workflow configuration is valid")
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("  - %d statuses defined\n", statusCount)
		fmt.Printf("  - %d transitions configured\n", transitionCount)
		fmt.Printf("  - %d start statuses (_start_)\n", startCount)
		fmt.Printf("  - %d terminal statuses (_complete_)\n", completeCount)
		fmt.Printf("  - All statuses are reachable\n")

		return nil
	}

	// Validation failed
	result["error"] = validationErr.Error()

	// JSON output
	if cli.GlobalConfig.JSON {
		_ = cli.OutputJSON(result)
		return fmt.Errorf("validation failed")
	}

	// Human-readable output
	cli.Error(fmt.Sprintf("✗ Workflow validation failed\n\n%s", validationErr.Error()))
	return fmt.Errorf("workflow configuration is invalid")
}
