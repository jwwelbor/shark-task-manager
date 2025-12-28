package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/formatters"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

var (
	historyAgent     string
	historySince     string
	historyEpic      string
	historyFeature   string
	historyOldStatus string
	historyNewStatus string
	historyLimit     int
	historyOffset    int
	historyFormat    string
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View project-wide task history",
	Long: `View project-wide task activity log with optional filtering.

Displays recent status changes, agent assignments, and task transitions across all tasks in the project.`,
	Example: `  # View recent 50 events (default)
  shark history

  # View history for specific agent
  shark history --agent=backend-agent-1

  # View history since timestamp
  shark history --since="2025-12-27T10:00:00Z"

  # View history for specific epic
  shark history --epic=E05

  # Filter by status transition
  shark history --old-status=todo --new-status=in_progress

  # Combine filters
  shark history --agent=backend --epic=E05 --limit=20

  # Pagination
  shark history --limit=10 --offset=10

  # Output as JSON
  shark history --json

  # Export as CSV
  shark history --format=csv

  # Export as JSON (alternative to --json)
  shark history --format=json`,
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().StringVar(&historyAgent, "agent", "", "Filter by agent ID")
	historyCmd.Flags().StringVar(&historySince, "since", "", "Filter by timestamp (ISO 8601 format)")
	historyCmd.Flags().StringVar(&historyEpic, "epic", "", "Filter by epic key")
	historyCmd.Flags().StringVar(&historyFeature, "feature", "", "Filter by feature key")
	historyCmd.Flags().StringVar(&historyOldStatus, "old-status", "", "Filter by old status")
	historyCmd.Flags().StringVar(&historyNewStatus, "new-status", "", "Filter by new status")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 50, "Maximum number of records to return")
	historyCmd.Flags().IntVar(&historyOffset, "offset", 0, "Number of records to skip")
	historyCmd.Flags().StringVar(&historyFormat, "format", "", "Output format (csv, json)")

	cli.RootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize database
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbConn := repository.NewDB(database)
	historyRepo := repository.NewTaskHistoryRepository(dbConn)
	taskRepo := repository.NewTaskRepository(dbConn)

	// Build filters
	filters := repository.HistoryFilters{
		Limit:  historyLimit,
		Offset: historyOffset,
	}

	// Parse timestamp if provided
	if historySince != "" {
		sinceTime, err := time.Parse(time.RFC3339, historySince)
		if err != nil {
			return fmt.Errorf("invalid timestamp format, expected ISO 8601 (RFC3339): %w", err)
		}
		filters.Since = &sinceTime
	}

	// Set optional filters
	if historyAgent != "" {
		filters.Agent = &historyAgent
	}
	if historyEpic != "" {
		filters.EpicKey = &historyEpic
	}
	if historyFeature != "" {
		filters.FeatureKey = &historyFeature
	}
	if historyOldStatus != "" {
		filters.OldStatus = &historyOldStatus
	}
	if historyNewStatus != "" {
		filters.NewStatus = &historyNewStatus
	}

	// Retrieve history
	histories, err := historyRepo.ListWithFilters(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to retrieve history: %w", err)
	}

	// Handle format-based export
	if historyFormat != "" {
		return outputHistoryExport(ctx, histories, taskRepo, historyFormat)
	}

	// Output based on format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(histories)
	}

	// Human-readable table output
	if len(histories) == 0 {
		cli.Info("No history records found")
		return nil
	}

	// Build enhanced history records with task information
	type HistoryDisplay struct {
		Timestamp string `json:"timestamp"`
		TaskKey   string `json:"task_key"`
		OldStatus string `json:"old_status,omitempty"`
		NewStatus string `json:"new_status"`
		Agent     string `json:"agent,omitempty"`
		Notes     string `json:"notes,omitempty"`
	}

	var displayRecords []HistoryDisplay
	for _, h := range histories {
		task, err := taskRepo.GetByID(ctx, h.TaskID)
		if err != nil {
			continue // Skip if task not found
		}

		record := HistoryDisplay{
			Timestamp: h.Timestamp.Format("2006-01-02 15:04:05"),
			TaskKey:   task.Key,
			NewStatus: h.NewStatus,
		}

		if h.OldStatus != nil {
			record.OldStatus = *h.OldStatus
		} else {
			record.OldStatus = "(initial)"
		}

		if h.Agent != nil {
			record.Agent = *h.Agent
		}

		if h.Notes != nil {
			record.Notes = *h.Notes
		}

		displayRecords = append(displayRecords, record)
	}

	// Print table
	headers := []string{"Timestamp", "Task", "Old Status", "New Status", "Agent", "Notes"}
	var rows [][]string
	for _, record := range displayRecords {
		rows = append(rows, []string{
			record.Timestamp,
			record.TaskKey,
			record.OldStatus,
			record.NewStatus,
			record.Agent,
			record.Notes,
		})
	}

	cli.OutputTable(headers, rows)

	// Print summary
	if len(histories) == historyLimit {
		cli.Info(fmt.Sprintf("Showing %d records (limit reached, use --offset to see more)", len(histories)))
	} else {
		cli.Info(fmt.Sprintf("Showing %d records", len(histories)))
	}

	return nil
}

func outputHistoryExport(ctx context.Context, histories []*models.TaskHistory, taskRepo *repository.TaskRepository, format string) error {
	// Build export records with task keys
	var historyWithTasks []formatters.HistoryWithTask
	for _, h := range histories {
		task, err := taskRepo.GetByID(ctx, h.TaskID)
		if err != nil {
			continue // Skip if task not found
		}

		historyWithTasks = append(historyWithTasks, formatters.HistoryWithTask{
			History: h,
			TaskKey: task.Key,
		})
	}

	// Convert to export records
	records := formatters.ConvertMultipleTasksToExportRecords(historyWithTasks)

	// Format based on requested format
	switch format {
	case "csv":
		csv, err := formatters.FormatHistoryCSV(records)
		if err != nil {
			return fmt.Errorf("failed to format history as CSV: %w", err)
		}
		fmt.Print(csv)
		return nil
	case "json":
		jsonStr, err := formatters.FormatHistoryJSON(records)
		if err != nil {
			return fmt.Errorf("failed to format history as JSON: %w", err)
		}
		fmt.Println(jsonStr)
		return nil
	default:
		return fmt.Errorf("unsupported format: %s (supported formats: csv, json)", format)
	}
}
