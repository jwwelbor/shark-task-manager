package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status dashboard",
	Long: `Display a comprehensive status dashboard showing project progress, active tasks, and blocked items.

Examples:
  shark status                       Show full project dashboard
  shark status --epic=E05            Show status for specific epic
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

	// Get flags
	epicKey, _ := cmd.Flags().GetString("epic")
	recentWindow, _ := cmd.Flags().GetString("recent")
	includeArchived, _ := cmd.Flags().GetBool("include-archived")

	// Get database path
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create repositories and service
	dbWrapper := repository.NewDB(database)
	service := status.NewStatusService(dbWrapper)

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

	// For now, just output JSON even in non-JSON mode
	// T04 will implement rich terminal output
	return outputStatusJSON(dashboard)
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
