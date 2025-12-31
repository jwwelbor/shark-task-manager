package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// listCmd represents the unified list command
var listCmd = &cobra.Command{
	Use:     "list [EPIC] [FEATURE]",
	Short:   "List epics, features, or tasks",
	GroupID: "essentials",
	Long: `Smart list command that dispatches to the appropriate subcommand based on arguments.

Positional Arguments:
  (no args)       List all epics
  EPIC            List features in epic (e.g., E04)
  EPIC FEATURE    List tasks in feature (e.g., E04 F01 or E04-F01)

Examples:
  shark list                      List all epics
  shark list E10                  List features in epic E10
  shark list E10 F01              List tasks in epic E10, feature F01
  shark list E10-F01              List tasks in feature E10-F01 (combined format)
  shark list --json               Output as JSON`,
	RunE: runList,
}

func init() {
	// Register list command with root
	cli.RootCmd.AddCommand(listCmd)

	// Add flags that apply to all list operations
	listCmd.Flags().String("status", "", "Filter by status")
	listCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	listCmd.Flags().Bool("show-all", false, "Show all items including completed (by default, completed items are hidden)")
}

// runList executes the list command dispatcher
func runList(cmd *cobra.Command, args []string) error {
	// Parse arguments to determine which subcommand to invoke
	command, epicKey, featureKey, err := ParseListArgs(args)
	if err != nil {
		return err
	}

	// Get flags
	statusFlag, _ := cmd.Flags().GetString("status")
	sortByFlag, _ := cmd.Flags().GetString("sort-by")
	showAllFlag, _ := cmd.Flags().GetBool("show-all")

	// Dispatch to appropriate subcommand
	switch command {
	case "epic":
		// Call epic list command
		return runEpicListWithFlags(cmd, statusFlag, sortByFlag, showAllFlag)

	case "feature":
		// Call feature list command with epic filter
		return runFeatureListWithFlags(cmd, *epicKey, statusFlag, sortByFlag, showAllFlag)

	case "task":
		// Call task list command with epic and feature filter
		return runTaskListWithFlags(cmd, *epicKey, *featureKey, statusFlag, sortByFlag, showAllFlag)

	default:
		// Should never happen
		return nil
	}
}

// runEpicListWithFlags calls the epic list command with flags
func runEpicListWithFlags(cmd *cobra.Command, statusFilter, sortBy string, showAll bool) error {
	// Set flags on the epic list command
	_ = epicListCmd.Flags().Set("status", statusFilter)
	_ = epicListCmd.Flags().Set("sort-by", sortBy)
	// Note: epic list doesn't have show-all flag, completed epics are always shown

	return runEpicList(epicListCmd, []string{})
}

// runFeatureListWithFlags calls the feature list command with epic filter and flags
func runFeatureListWithFlags(cmd *cobra.Command, epic, statusFilter, sortBy string, showAll bool) error {
	// Set flags on the feature list command
	_ = featureListCmd.Flags().Set("status", statusFilter)
	_ = featureListCmd.Flags().Set("sort-by", sortBy)
	_ = featureListCmd.Flags().Set("show-all", formatBool(showAll))

	return runFeatureList(featureListCmd, []string{epic})
}

// runTaskListWithFlags calls the task list command with epic and feature filter and flags
func runTaskListWithFlags(cmd *cobra.Command, epic, feature, statusFilter, sortBy string, showAll bool) error {
	// Set flags on the task list command
	_ = taskListCmd.Flags().Set("status", statusFilter)
	// Note: task list doesn't have sort-by flag yet
	_ = taskListCmd.Flags().Set("show-all", formatBool(showAll))

	return runTaskList(taskListCmd, []string{epic, feature})
}

// formatBool converts a boolean to string for flag setting
func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
