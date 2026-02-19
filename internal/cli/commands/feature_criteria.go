package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// featureCriteriaCmd shows aggregated criteria for a feature
var featureCriteriaCmd = &cobra.Command{
	Use:   "criteria <feature-key>",
	Short: "Show aggregated acceptance criteria for a feature",
	Long: `Show aggregated acceptance criteria across all tasks in a feature.

Displays:
  - Total criteria count across all feature tasks
  - Breakdown by status: pending, in_progress, complete, failed, na
  - Overall completion percentage
  - Optional per-task breakdown with --by-task flag

Examples:
  shark feature criteria E10-F04
  shark feature criteria E10-F04 --by-task
  shark feature criteria E10-F04 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureCriteria,
}

// runFeatureCriteria handles the feature criteria command
func runFeatureCriteria(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	featureKey := args[0]
	byTask, _ := cmd.Flags().GetBool("by-task")

	// Step 2: Call service
	svc := cli.GetCriteriaService()
	result, err := svc.GetFeatureCriteria(cmd.Context(), featureKey)
	if err != nil {
		return fmt.Errorf("failed to get criteria for feature %s: %w", featureKey, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	// Human-readable output
	if result.Summary.TotalCriteria == 0 {
		fmt.Printf("Feature %s\n", result.FeatureKey)
		fmt.Println("No acceptance criteria found for this feature")
		return nil
	}

	fmt.Printf("Feature %s\n\n", result.FeatureKey)
	fmt.Printf("Overall Progress: %.0f%% complete (%d/%d criteria)\n",
		result.Summary.CompletionPct,
		result.Summary.CompleteCount+result.Summary.NACount,
		result.Summary.TotalCriteria)
	fmt.Printf("  Complete: %d | Pending: %d | In Progress: %d | Failed: %d | N/A: %d\n",
		result.Summary.CompleteCount,
		result.Summary.PendingCount,
		result.Summary.InProgressCount,
		result.Summary.FailedCount,
		result.Summary.NACount)
	fmt.Printf("  Tasks: %d\n", result.Summary.TotalTasks)

	// Show per-task breakdown if requested
	if byTask {
		fmt.Println("\nPer-Task Breakdown:")
		for _, taskResult := range result.Tasks {
			if taskResult.Summary.TotalCount == 0 {
				continue
			}
			fmt.Printf("\n  %s\n", taskResult.TaskKey)
			fmt.Printf("    %.0f%% complete (%d/%d criteria)\n",
				taskResult.Summary.CompletionPct,
				taskResult.Summary.CompleteCount+taskResult.Summary.NACount,
				taskResult.Summary.TotalCount)
			fmt.Printf("    Complete: %d | Pending: %d | In Progress: %d | Failed: %d | N/A: %d\n",
				taskResult.Summary.CompleteCount,
				taskResult.Summary.PendingCount,
				taskResult.Summary.InProgressCount,
				taskResult.Summary.FailedCount,
				taskResult.Summary.NACount)
		}
	}

	return nil
}

func init() {
	// Add criteria subcommand to feature command
	featureCmd.AddCommand(featureCriteriaCmd)

	// Flags for criteria command
	featureCriteriaCmd.Flags().BoolP("by-task", "t", false, "Show per-task breakdown")
}
