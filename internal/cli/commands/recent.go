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
	Short:   "List most recently created entities across tasks, features, and epics",
	GroupID: "inspect",
	Long: `List the most recently created entities across tasks, features, and epics.

The optional positional argument N (or --limit=N flag) controls how many items
to return. When both are given, the flag wins and a warning is printed (unless
--json mode is active).

Type filter flags narrow the result to specific entity kinds; without any type
flag all three types are included.

Examples:
  shark recent
  shark recent 10
  shark recent --limit=20
  shark recent --tasks --features
  shark recent 5 --epics --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRecent,
}

func init() {
	cli.RootCmd.AddCommand(recentCmd)
	recentCmd.Flags().Int("limit", 0, "Limit results (overrides positional argument and config default)")
	recentCmd.Flags().Bool("tasks", false, "Include only tasks")
	recentCmd.Flags().Bool("features", false, "Include only features")
	recentCmd.Flags().Bool("epics", false, "Include only epics")
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
	}, nil
}

func renderRecentTable(items []services.RecentItem) error {
	if len(items) == 0 {
		fmt.Println("No recent items found.")
		return nil
	}

	headers := []string{"Type", "Key", "Title", "Created", "Status"}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Type,
			item.Key,
			item.Title,
			item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			item.Status,
		})
	}
	cli.OutputTable(headers, rows)
	return nil
}
