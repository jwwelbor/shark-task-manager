package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

const (
	recentMaxLimit       = 10_000
	recentBuiltInDefault = 5
)

var recentCmd = &cobra.Command{
	Use:     "recent [N]",
	Short:   "List most recently created entities across all entity types",
	GroupID: "inspect",
	Long: `List the most recently created entities across all entity types
(tasks, features, epics, bugs, change-cards, ideas, and tech-debt).

The optional positional argument N (or --limit=N flag) controls how many items
to return. When both are given, the flag wins and a warning is printed (unless
--json mode is active).

Type filter flags narrow the result to specific entity kinds; without any type
flag all entity types are included.

Examples:
  shark recent
  shark recent 10
  shark recent --limit=20
  shark recent --tasks --features
  shark recent 5 --epics --json
  shark recent --bugs --changes
  shark recent --ideas --tech-debt`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRecent,
}

func init() {
	cli.RootCmd.AddCommand(recentCmd)
	recentCmd.Flags().Int("limit", 0, "Limit results (overrides positional argument and config default)")
	recentCmd.Flags().Bool("tasks", false, "Include only tasks")
	recentCmd.Flags().Bool("features", false, "Include only features")
	recentCmd.Flags().Bool("epics", false, "Include only epics")
	recentCmd.Flags().Bool("bugs", false, "Include only bugs")
	recentCmd.Flags().Bool("changes", false, "Include only change-cards")
	recentCmd.Flags().Bool("ideas", false, "Include only ideas")
	recentCmd.Flags().Bool("tech-debt", false, "Include only tech-debt items")
}

func runRecent(cmd *cobra.Command, args []string) error {
	cfgLimit := recentBuiltInDefault
	if cfg, err := cli.GetConfig(); err == nil && cfg != nil {
		cfgLimit = cfg.GetRecentDefaultLimit()
	}

	filters, err := parseRecentFiltersWithConfig(cmd, args, cfgLimit)
	if err != nil {
		return err
	}

	svc := cli.GetRecentService()
	return runRecentWithSvc(cmd.Context(), filters, svc)
}

func runRecentWithSvc(ctx context.Context, filters services.RecentFilters, svc recentServicerInternal) error {
	items, err := svc.ListRecent(ctx, filters)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(items)
	}
	return renderRecentTable(items)
}

type recentServicerInternal interface {
	ListRecent(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error)
}

// parseRecentFiltersWithConfig resolves the effective limit.
// Precedence: --limit flag > positional arg > cfgLimit.
func parseRecentFiltersWithConfig(cmd *cobra.Command, args []string, cfgLimit int) (services.RecentFilters, error) {
	flagLimit, _ := cmd.Flags().GetInt("limit")
	includeTasks, _ := cmd.Flags().GetBool("tasks")
	includeFeatures, _ := cmd.Flags().GetBool("features")
	includeEpics, _ := cmd.Flags().GetBool("epics")
	includeBugs, _ := cmd.Flags().GetBool("bugs")
	includeChanges, _ := cmd.Flags().GetBool("changes")
	includeIdeas, _ := cmd.Flags().GetBool("ideas")
	includeTechDebt, _ := cmd.Flags().GetBool("tech-debt")

	limit := cfgLimit
	if limit <= 0 {
		limit = recentBuiltInDefault
	}

	hasPositional := len(args) > 0
	hasFlag := cmd.Flags().Changed("limit")

	if hasPositional {
		positional, err := strconv.Atoi(args[0])
		if err != nil || positional <= 0 || positional > recentMaxLimit {
			return services.RecentFilters{}, fmt.Errorf(
				"exit code 3: invalid limit %q: must be a positive integer <= %d", args[0], recentMaxLimit,
			)
		}
		limit = positional
	}

	if hasFlag {
		if flagLimit <= 0 || flagLimit > recentMaxLimit {
			return services.RecentFilters{}, fmt.Errorf(
				"exit code 3: invalid --limit %d: must be a positive integer <= %d", flagLimit, recentMaxLimit,
			)
		}
		if hasPositional && !cli.GlobalConfig.JSON {
			cli.Warning(fmt.Sprintf("both positional (%s) and --limit flag (%d) provided; --limit wins", args[0], flagLimit))
		}
		limit = flagLimit
	}

	return services.RecentFilters{
		Limit:           limit,
		IncludeTasks:    includeTasks,
		IncludeFeatures: includeFeatures,
		IncludeEpics:    includeEpics,
		IncludeBugs:     includeBugs,
		IncludeChanges:  includeChanges,
		IncludeIdeas:    includeIdeas,
		IncludeTechDebt: includeTechDebt,
	}, nil
}

var recentTableHeaders = []string{"Type", "Key", "Title", "Created", "Status"}

const recentTitleColIdx = 2

func renderRecentTable(items []services.RecentItem) error {
	if len(items) == 0 {
		fmt.Println("No recent items found.")
		return nil
	}

	cli.OutputTable(recentTableHeaders, buildRecentRows(items))
	return nil
}

// buildRecentRows converts recent items to table rows for list display.
// Extracted for testability (CC-036 follow-up: console_width coverage).
//
// The title column is sized from the actual rendered widths of the other
// columns so the table fills the configured console_width even when keys
// and statuses are short (e.g., "TD-052"/"identified") rather than
// reserving worst-case space for them.
func buildRecentRows(items []services.RecentItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Type,
			item.Key,
			"", // title placeholder; filled below after width is computed
			item.CreatedAt.UTC().Format("2006-01-02 15:04"),
			item.Status,
		})
	}

	titleMax := availableTitleWidth(cli.GetConsoleWidth(), recentTableHeaders, rows, recentTitleColIdx)
	for i, item := range items {
		rows[i][recentTitleColIdx] = truncateToWidth(item.Title, titleMax)
	}
	return rows
}
