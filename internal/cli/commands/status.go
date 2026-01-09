package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:     "status [EPIC] [FEATURE]",
	Short:   "Show project status dashboard",
	GroupID: "status",
	Long: `Display a comprehensive status dashboard showing project progress, active tasks, and blocked items.

Positional Arguments:
  (no args)       Show full project dashboard
  EPIC            Show status for specific epic (e.g., E04)
  EPIC FEATURE    Show status for specific feature (e.g., E04 F01 or E04-F01)

Examples:
  shark status                       Show full project dashboard
  shark status E05                   Show status for epic E05
  shark status E05 F02               Show status for feature E05-F02
  shark status E05-F02               Show status for feature E05-F02 (combined format)
  shark status --epic=E05            Flag syntax (still supported)
  shark status --recent=7d           Include recent completions (7 days)
  shark status --json                Output as JSON`,
	RunE: runStatus,
}

func init() {
	// Register status command with root
	cli.RootCmd.AddCommand(statusCmd)

	// Add flags
	statusCmd.Flags().String("epic", "", "Filter by epic key")
	statusCmd.Flags().String("recent", "", "Recent completion window (24h, 7d, 30d, 90d)")
	statusCmd.Flags().Bool("include-archived", false, "Include archived epics/features")
}

// runStatus executes the status command
func runStatus(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parse positional arguments first
	_, positionalEpic, _, err := ParseListArgs(args)
	if err != nil {
		return err
	}

	// Get flags
	epicKeyFlag, _ := cmd.Flags().GetString("epic")
	recentWindow, _ := cmd.Flags().GetString("recent")
	includeArchived, _ := cmd.Flags().GetBool("include-archived")

	// Positional argument takes priority over flag
	epicKey := epicKeyFlag
	if positionalEpic != nil {
		epicKey = *positionalEpic
	}

	// For now, if a feature is specified, we treat it as epic-level status
	// Future enhancement could add feature-specific status view

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create service
	service := status.NewStatusService(repoDb)

	// Build request
	req := &status.StatusRequest{
		EpicKey:         epicKey,
		RecentWindow:    recentWindow,
		IncludeArchived: includeArchived,
	}

	// Get dashboard
	dashboard, err := service.GetDashboard(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Output
	if cli.GlobalConfig.JSON {
		return outputStatusJSON(dashboard)
	}

	// Rich terminal output
	return outputStatusTerminal(dashboard)
}

// outputStatusJSON outputs the dashboard as JSON
func outputStatusJSON(dashboard *status.StatusDashboard) error {
	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// outputStatusTerminal outputs the dashboard with rich terminal formatting
func outputStatusTerminal(dashboard *status.StatusDashboard) error {
	// Use the formatter from the status package
	output := status.FormatDashboard(dashboard, cli.GlobalConfig.NoColor)
	fmt.Print(output)
	return nil
}
