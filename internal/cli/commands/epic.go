package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

var epicCmd = &cobra.Command{
	Use:     "epic",
	Short:   "Manage epics",
	GroupID: "advanced",
	Long: `Query and manage epics with automatic progress calculation.

Examples:
  shark epic list          List all epics
  shark epic get E04       Get epic details with progress`,
}

var epicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all epics",
	Long: `List all epics with progress information.

Examples:
  shark epic list           List all epics
  shark epic list --json    Output as JSON`,
	RunE: runEpicList,
}

var epicGetCmd = &cobra.Command{
	Use:   "get <epic-key>",
	Short: "Get epic details",
	Long: `Display detailed information about a specific epic including all features and progress.

Accepts numeric keys (E04) or slugged keys (E04-epic-name).

Examples:
  shark epic get E04              Get epic by key
  shark epic get E04-enhancements Get epic by slugged key
  shark epic get E04 --json       Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicGet,
}

var epicStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show epic status summary",
	Long:  `Display a summary of all epics with completion percentages and task counts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cli.Warning("Not yet implemented - coming in E05-F01")
		return nil
	},
}

var epicCompleteCmd = &cobra.Command{
	Use:   "complete <epic-key>",
	Short: "Complete all tasks in an epic",
	Long: `Mark all tasks in an epic as completed.

Without --force, fails if any tasks are incomplete. Accepts numeric or slugged keys.

Examples:
  shark epic complete E07         Complete epic by key
  shark epic complete E07 --force Force complete all tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicComplete,
}

var epicCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new epic",
	Long: `Create a new epic with auto-assigned key, folder structure, and database entry.

Examples:
  shark epic create "User Auth"
  shark epic create "Platform Roadmap" --file="docs/specs/roadmap.md"
  shark epic create "Q1 Goals" --file="docs/roadmap/q1.md" --force`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicCreate,
}

var epicDeleteCmd = &cobra.Command{
	Use:   "delete <epic-key>",
	Short: "Delete an epic",
	Long: `Delete an epic (and all its features/tasks via CASCADE). Use --force if epic has features.

Examples:
  shark epic delete E05           Delete epic with no features
  shark epic delete E05 --force   Force delete epic with features`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicDelete,
}

var epicUpdateCmd = &cobra.Command{
	Use:   "update <epic-key>",
	Short: "Update an epic",
	Long: `Update an epic's properties: title, description, priority, or file path.

Use 'shark status set' to change status.

Examples:
  shark epic update E01 --title "New Title"
  shark epic update E01 --priority high
  shark epic update E01 --file "docs/roadmap/2025.md"`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicUpdate,
}

var (
	epicCreateDescription string
	epicCreateKey         string
)

func init() {
	cli.RootCmd.AddCommand(epicCmd)
	epicCmd.AddCommand(epicListCmd, epicGetCmd, epicStatusCmd, epicCompleteCmd, epicCreateCmd, epicDeleteCmd, epicUpdateCmd)

	epicListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	epicListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")

	epicCompleteCmd.Flags().Bool("force", false, "Force completion of all tasks regardless of status")

	epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
	epicCreateCmd.Flags().StringVar(&epicCreateKey, "key", "", "Custom key (e.g., E00, bugs). Defaults to auto-generated next E## number")
	epicCreateCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/epic.md)")
	epicCreateCmd.Flags().String("filename", "", "Alias for --file")
	epicCreateCmd.Flags().String("path", "", "Alias for --file")
	_ = epicCreateCmd.Flags().MarkHidden("filename")
	_ = epicCreateCmd.Flags().MarkHidden("path")
	epicCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
	epicCreateCmd.Flags().String("priority", "medium", "Priority: low, medium, high (default: medium)")
	epicCreateCmd.Flags().String("business-value", "", "Business value: low, medium, high (optional)")
	epicCreateCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived (default: draft)")

	epicDeleteCmd.Flags().Bool("force", false, "Force deletion even if epic has features")

	epicUpdateCmd.Flags().String("title", "", "New title for the epic")
	epicUpdateCmd.Flags().String("description", "", "New description for the epic")
	epicUpdateCmd.Flags().String("priority", "", "New priority: low, medium, high")
	epicUpdateCmd.Flags().String("business-value", "", "New business value: low, medium, high")
	epicUpdateCmd.Flags().String("key", "", "New key for the epic (must be unique, cannot contain spaces)")
	epicUpdateCmd.Flags().String("file", "", "New file path (e.g., docs/custom/epic.md)")
	epicUpdateCmd.Flags().String("filename", "", "Alias for --file")
	epicUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = epicUpdateCmd.Flags().MarkHidden("filename")
	_ = epicUpdateCmd.Flags().MarkHidden("path")
}

// runEpicList lists all epics with progress information.
func runEpicList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sortBy, _ := cmd.Flags().GetString("sort-by")
	statusFilter, _ := cmd.Flags().GetString("status")

	if statusFilter != "" {
		validated, err := ParseEpicStatus(statusFilter)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		statusFilter = validated
	}

	if sortBy != "" && sortBy != "key" && sortBy != "progress" && sortBy != "status" {
		cli.Error(fmt.Sprintf("Error: Invalid sort-by '%s'. Must be one of: key, progress, status", sortBy))
		os.Exit(1)
	}

	epics, err := cli.GetEpicService().ListEpics(ctx, services.EpicFilters{Status: statusFilter})
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to list epics: %v\n", err)
		}
		os.Exit(2)
	}

	if len(epics) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{"results": []interface{}{}, "count": 0})
		}
		cli.Info("No epics found")
		return nil
	}

	epicsWithProgress := buildEpicsWithProgress(ctx, epics)
	sortEpics(epicsWithProgress, sortBy)

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{"results": epicsWithProgress, "count": len(epicsWithProgress)})
	}
	renderEpicListTable(epicsWithProgress)
	return nil
}

// runEpicGet retrieves and displays a specific epic with its details.
func runEpicGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := args[0]
	epic, err := cli.GetEpicService().GetEpic(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	displaySvc := cli.GetDisplayService()
	displayMode := displaySvc.DetermineEpicDisplayMode(epic)

	if displayMode == services.DisplayModePlanning {
		info, err := displaySvc.GetEpicDisplayInfo(ctx, epicKey)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to get epic display info: %v", err))
			os.Exit(2)
		}
		info.ResolvedPath = resolveEpicPlanningPath(ctx, epic.Key)
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(info)
		}
		renderEpicPlanning(info)
		return nil
	}

	data, err := buildEpicGetData(ctx, epic)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to build epic data: %v\n", err)
		}
		os.Exit(2)
	}

	orchestratorAction := displaySvc.ResolveEpicAction(ctx, epic)
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(buildEpicGetJSON(epic, data, orchestratorAction))
	}
	renderEpicDetails(epic, data.EpicProgress, data.FeaturesWithDetails, data.DirPath, data.Filename, data.RelatedDocs, data.FeatureRollup, data.TaskRollup, data.BlockedTasks, data.ApprovalBacklogCount, data.EpicNotes, data.EpicContext)
	displayOrchestratorAction(orchestratorAction)
	return nil
}

// runEpicCreate creates a new epic.
func runEpicCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return performEpicCreate(ctx, args[0], cmd)
}

// runEpicComplete marks all tasks in an epic as completed.
func runEpicComplete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	force, _ := cmd.Flags().GetBool("force")
	return performEpicComplete(ctx, args[0], force)
}

// runEpicDelete deletes an epic and all its children.
func runEpicDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	force, _ := cmd.Flags().GetBool("force")
	return performEpicDelete(ctx, args[0], force)
}

// runEpicUpdate updates an epic's properties.
func runEpicUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return performEpicUpdate(ctx, args[0], cmd)
}
