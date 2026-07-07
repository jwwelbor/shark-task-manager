package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// listCmd represents the unified list command
var listCmd = &cobra.Command{
	Use:     "list [EPIC] [FEATURE]",
	Short:   "List epics, features, tasks, ideas, bugs, changes, or tech-debt",
	GroupID: "inspect",
	Long: `Smart list command that dispatches to the appropriate subcommand based on arguments.

Positional Arguments:
  (no args)              List all epics
  EPIC                   List features in epic (e.g., E04)
  EPIC FEATURE           List tasks in feature (e.g., E04 F01 or E04-F01)
  idea / ideas           List all ideas
  bug / bugs             List all bugs
  change / changes       List all change cards (also: change-card / change_card)
  tech-debt / td         List all tech-debt items (also: tech_debt / techdebt)

Examples:
  shark list                      List all epics
  shark list E10                  List features in epic E10
  shark list E10 F01              List tasks in epic E10, feature F01
  shark list E10-F01              List tasks in feature E10-F01 (combined format)
  shark list idea                 List all ideas
  shark list bugs                 List all bugs
  shark list changes              List all change cards
  shark list tech-debt            List all tech-debt items
  shark list td                   List all tech-debt items (alias)
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
	_ = listCmd.Flags().MarkDeprecated("show-all", "use --all instead")
	listCmd.Flags().Bool("all", false, "Show all items including completed (by default, completed items are hidden)")
	// Idea-specific flags (only used when 'idea' or 'ideas' keyword is provided)
	listCmd.Flags().Int("priority", 0, "Filter ideas by priority (1-10)")
	// E28-F05 REQ-F-009: repeatable --tag flag with AND semantics.
	listCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")
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
	allFlag, _ := cmd.Flags().GetBool("all")
	showAllFlag = showAllFlag || allFlag
	// E28-F05 REQ-F-009: read the repeatable --tag flag.
	// nil when no --tag flags were supplied (not an empty slice).
	var tagFlags []string
	if rawTags, err := cmd.Flags().GetStringSlice("tag"); err == nil && len(rawTags) > 0 {
		tagFlags = rawTags
	}

	// Dispatch to appropriate subcommand
	switch command {
	case "epic":
		// Call epic list command
		return runEpicListWithFlags(cmd, statusFlag, sortByFlag, showAllFlag, tagFlags)

	case "feature":
		// Call feature list command with epic filter
		return runFeatureListWithFlags(cmd, *epicKey, statusFlag, sortByFlag, showAllFlag, tagFlags)

	case "task":
		// Call task list command with epic and feature filter
		return runTaskListWithFlags(cmd, *epicKey, *featureKey, statusFlag, sortByFlag, showAllFlag, tagFlags)

	case "idea":
		// Wire idea-specific flags from listCmd into package-level idea vars
		ideaStatus, _ = cmd.Flags().GetString("status")
		if p, err := cmd.Flags().GetInt("priority"); err == nil {
			ideaPriority = p
		}
		forwardTagFlags(ideaListCmd, tagFlags)
		// Propagate context and delegate to idea list handler
		ideaListCmd.SetContext(cmd.Context())
		return runIdeaList(ideaListCmd, []string{})

	case "bug":
		bugListCmd.SetContext(cmd.Context())
		_ = bugListCmd.Flags().Set("status", statusFlag)
		_ = bugListCmd.Flags().Set("all", formatBool(showAllFlag))
		forwardTagFlags(bugListCmd, tagFlags)
		return runBugList(bugListCmd, []string{})

	case "change":
		changeListCmd.SetContext(cmd.Context())
		changeStatusFilter = statusFlag
		_ = changeListCmd.Flags().Set("status", statusFlag)
		_ = changeListCmd.Flags().Set("all", formatBool(showAllFlag))
		forwardTagFlags(changeListCmd, tagFlags)
		return runChangeList(changeListCmd, []string{})

	case "tech_debt":
		// Forward --all flag to tech-debt list command.
		// --tag is forwarded as nil per REQ-F-009 (tech_debt out of scope).
		tdListCmd.SetContext(cmd.Context())
		_ = tdListCmd.Flags().Set("status", statusFlag)
		_ = tdListCmd.Flags().Set("all", formatBool(showAllFlag))
		return runTdList(tdListCmd, []string{})

	case "sprint":
		sprintListCmd.SetContext(cmd.Context())
		if statusFlag != "" {
			_ = sprintListCmd.Flags().Set("status", statusFlag)
		}
		if epicKey != nil {
			// epicKey holds the sprint key when `shark list S001` was used
			return runSprintBacklog(cmd, []string{*epicKey})
		}
		return runSprintList(sprintListCmd, []string{})

	default:
		// Should never happen
		return nil
	}
}

// forwardTagFlags forwards a repeatable --tag slice from runList to a delegated
// subcommand. nil/empty slice is a no-op so callers can pass through unconditionally.
func forwardTagFlags(target *cobra.Command, tags []string) {
	for _, t := range tags {
		_ = target.Flags().Set("tag", t)
	}
}

// runEpicListWithFlags calls the epic list command with flags.
// tags is nil when no --tag flags were supplied (REQ-F-009).
func runEpicListWithFlags(cmd *cobra.Command, statusFilter, sortBy string, showAll bool, tags []string) error {
	_ = epicListCmd.Flags().Set("status", statusFilter)
	_ = epicListCmd.Flags().Set("sort-by", sortBy)
	// Note: epic list doesn't have show-all flag, completed epics are always shown
	forwardTagFlags(epicListCmd, tags)
	epicListCmd.SetContext(cmd.Context())
	return runEpicList(epicListCmd, []string{})
}

// runFeatureListWithFlags calls the feature list command with epic filter and flags.
// tags is nil when no --tag flags were supplied (REQ-F-009).
func runFeatureListWithFlags(cmd *cobra.Command, epic, statusFilter, sortBy string, showAll bool, tags []string) error {
	_ = featureListCmd.Flags().Set("status", statusFilter)
	_ = featureListCmd.Flags().Set("sort-by", sortBy)
	_ = featureListCmd.Flags().Set("all", formatBool(showAll))
	forwardTagFlags(featureListCmd, tags)
	featureListCmd.SetContext(cmd.Context())
	return runFeatureList(featureListCmd, []string{epic})
}

// runTaskListWithFlags calls the task list command with epic and feature filter and flags.
// tags is nil when no --tag flags were supplied (REQ-F-009).
func runTaskListWithFlags(cmd *cobra.Command, epic, feature, statusFilter, sortBy string, showAll bool, tags []string) error {
	_ = taskListCmd.Flags().Set("status", statusFilter)
	// Note: task list doesn't have sort-by flag yet
	_ = taskListCmd.Flags().Set("all", formatBool(showAll))
	forwardTagFlags(taskListCmd, tags)
	taskListCmd.SetContext(cmd.Context())
	return runTaskList(taskListCmd, []string{epic, feature})
}

// formatBool converts a boolean to string for flag setting
func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
