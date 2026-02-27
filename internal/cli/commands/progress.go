package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/spf13/cobra"
)

// progressCmd is the dedicated command for viewing entity progress rollups,
// health indicators, task breakdowns, and action items.
//
// Once E17-F07 is complete, "shark status <id>" becomes a deprecated alias
// for this command, resolving the namespace collision with status transitions.
var progressCmd = &cobra.Command{
	Use:     "progress [EPIC] [FEATURE]",
	Short:   "Show progress, health indicators, and task breakdown",
	GroupID: "inspect",
	Long: `Display a progress dashboard showing project progress, health indicators,
active tasks, and blocked items.

Positional Arguments:
  (no args)       Show full project progress dashboard
  EPIC            Show progress for specific epic (e.g., E04)
  EPIC FEATURE    Show progress for specific feature (e.g., E04 F01 or E04-F01)

Examples:
  shark progress                     Show full project progress dashboard
  shark progress E05                 Show progress for epic E05
  shark progress E05 F02             Show progress for feature E05-F02
  shark progress E05-F02             Show progress for feature E05-F02 (combined format)
  shark progress --epic=E05          Flag syntax (still supported)
  shark progress --recent=7d         Include recent completions (7 days)
  shark progress --json              Output as JSON`,
	RunE: runProgress,
}

func init() {
	cli.RootCmd.AddCommand(progressCmd)
	progressCmd.Flags().String("epic", "", "Filter by epic key")
	progressCmd.Flags().String("recent", "", "Recent completion window (24h, 7d, 30d, 90d)")
	progressCmd.Flags().Bool("include-archived", false, "Include archived epics/features")
}

// runProgress executes the progress command.
// Pattern: parse -> call service -> format output. No business logic here.
func runProgress(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	req, err := parseProgressRequest(cmd, args)
	if err != nil {
		return err
	}

	// Ensure a usable context with timeout when none is provided (e.g., in tests)
	ctx := cmd.Context()
	if ctx == nil {
		var cancel func()
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	// Step 2: Call service
	dashboard, err := cli.GetStatusService().GetDashboard(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get progress: %w", err)
	}

	// Enrich with display mode metadata (uses DisplayService, no additional DB queries)
	enrichEpicSummaries(dashboard.Epics, cli.GetDisplayService())

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return outputProgressJSON(dashboard)
	}
	return outputProgressTerminal(dashboard)
}

// parseProgressRequest builds a StatusRequest from command arguments and flags.
// Supports three invocation forms:
//
//	shark progress                      -> no filter
//	shark progress E05                  -> epic filter
//	shark progress E05 F02              -> epic + feature filter
//	shark progress E05-F02              -> combined format (feature filter)
func parseProgressRequest(cmd *cobra.Command, args []string) (*status.StatusRequest, error) {
	_, positionalEpic, _, err := ParseListArgs(args)
	if err != nil {
		return nil, err
	}

	epicKeyFlag, _ := cmd.Flags().GetString("epic")
	recentWindow, _ := cmd.Flags().GetString("recent")
	includeArchived, _ := cmd.Flags().GetBool("include-archived")

	// Positional argument takes precedence over flag (matches status.go behavior)
	epicKey := epicKeyFlag
	if positionalEpic != nil {
		epicKey = *positionalEpic
	}

	return &status.StatusRequest{
		EpicKey:         epicKey,
		RecentWindow:    recentWindow,
		IncludeArchived: includeArchived,
	}, nil
}

// outputProgressJSON marshals the dashboard to JSON via cli.OutputJSON,
// which handles --field extraction and consistent formatting.
func outputProgressJSON(dashboard *status.StatusDashboard) error {
	return cli.OutputJSON(dashboard)
}

// outputProgressTerminal renders the dashboard with rich terminal formatting.
func outputProgressTerminal(dashboard *status.StatusDashboard) error {
	output := status.FormatDashboard(dashboard, cli.GlobalConfig.NoColor)
	fmt.Print(output)
	return nil
}

// enrichEpicSummaries populates DisplayMode, IsPlanning, and Phase fields
// on each EpicSummary using the DisplayService to determine planning vs aggregation mode.
// Uses in-memory workflow config lookup (no additional DB queries).
func enrichEpicSummaries(epics []*status.EpicSummary, displaySvc *services.DisplayService) {
	for _, epic := range epics {
		mode := displaySvc.DetermineEpicDisplayModeByStatus(epic.Status)
		epic.DisplayMode = string(mode)
		if mode == services.DisplayModePlanning {
			epic.IsPlanning = true
			epic.Phase = displaySvc.GetEpicPhase(epic.Status)
		}
	}
}
