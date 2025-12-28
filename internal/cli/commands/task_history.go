package commands

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/formatters"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var taskHistoryFormat string

// taskHistoryCmd shows the history of a task
var taskHistoryCmd = &cobra.Command{
	Use:   "history <task-key>",
	Short: "Show task history",
	Long: `Display the complete lifecycle history of a task showing all status transitions.

Shows all status changes with timestamps, agents, and notes in chronological order.

Examples:
  shark task history T-E04-F01-001
  shark task history T-E04-F01-001 --json
  shark task history T-E04-F01-001 --format=csv
  shark task history T-E04-F01-001 --format=json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskHistory,
}

// HistoryOutput represents the JSON output structure for task history
type HistoryOutput struct {
	TaskKey string         `json:"task_key"`
	History []HistoryEntry `json:"history"`
}

// HistoryEntry represents a single history record in the output
type HistoryEntry struct {
	Timestamp   string  `json:"timestamp"`
	RelativeAge string  `json:"relative_age"`
	OldStatus   *string `json:"old_status,omitempty"`
	NewStatus   string  `json:"new_status"`
	Agent       *string `json:"agent,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

func runTaskHistory(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	ctx := context.Background()

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Create repository
	dbConn := repository.NewDB(database)
	historyRepo := repository.NewTaskHistoryRepository(dbConn)

	// Get history
	histories, err := historyRepo.GetHistoryByTaskKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task history: %w", err)
	}

	// Check if task exists (empty history could mean no task or no transitions)
	if len(histories) == 0 {
		// Verify task exists
		taskRepo := repository.NewTaskRepository(dbConn)
		_, err := taskRepo.GetByKey(ctx, taskKey)
		if err != nil {
			return fmt.Errorf("task not found: %s", taskKey)
		}

		// Task exists but has no history
		if cli.GlobalConfig.JSON {
			output := HistoryOutput{
				TaskKey: taskKey,
				History: []HistoryEntry{},
			}
			return cli.OutputJSON(output)
		}

		cli.Info("No history found for task %s", taskKey)
		return nil
	}

	// Format output based on --format flag or --json flag
	switch taskHistoryFormat {
	case "csv":
		return outputHistoryCSV(taskKey, histories)
	case "json":
		return outputHistoryJSONExport(taskKey, histories)
	case "":
		// Default behavior: use --json flag or table
		if cli.GlobalConfig.JSON {
			return outputHistoryJSON(taskKey, histories)
		}
		return outputHistoryTable(taskKey, histories)
	default:
		return fmt.Errorf("unsupported format: %s (supported formats: csv, json)", taskHistoryFormat)
	}
}

func outputHistoryJSON(taskKey string, histories []*models.TaskHistory) error {
	entries := make([]HistoryEntry, len(histories))

	for i, h := range histories {
		entries[i] = HistoryEntry{
			Timestamp:   h.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			RelativeAge: utils.FormatRelativeTime(h.Timestamp),
			OldStatus:   h.OldStatus,
			NewStatus:   h.NewStatus,
			Agent:       h.Agent,
			Notes:       h.Notes,
		}
	}

	output := HistoryOutput{
		TaskKey: taskKey,
		History: entries,
	}

	return cli.OutputJSON(output)
}

func outputHistoryTable(taskKey string, histories []*models.TaskHistory) error {
	// Print title
	cli.Title(fmt.Sprintf("Task History: %s", taskKey))
	fmt.Println()

	// Print timeline
	for i, h := range histories {
		// Timeline marker
		if i == 0 {
			fmt.Print(pterm.LightCyan("┌─"))
		} else {
			fmt.Print(pterm.LightCyan("├─"))
		}

		// Timestamp and relative age
		timestamp := h.Timestamp.Format("2006-01-02 15:04:05")
		relativeAge := utils.FormatRelativeTime(h.Timestamp)

		// Status transition
		var statusLine string
		if h.OldStatus != nil {
			statusLine = fmt.Sprintf("%s → %s",
				formatStatus(*h.OldStatus),
				formatStatus(h.NewStatus))
		} else {
			statusLine = fmt.Sprintf("created as %s", formatStatus(h.NewStatus))
		}

		// Print main line
		fmt.Printf(" %s %s (%s)\n",
			pterm.LightCyan(timestamp),
			statusLine,
			pterm.Gray(relativeAge))

		// Print agent if present
		if h.Agent != nil && *h.Agent != "" {
			fmt.Printf(pterm.LightCyan("│  ")+"Agent: %s\n", pterm.LightYellow(*h.Agent))
		}

		// Print notes if present
		if h.Notes != nil && *h.Notes != "" {
			fmt.Printf(pterm.LightCyan("│  ")+"Notes: %s\n", *h.Notes)
		}

		// Add spacing between entries
		if i < len(histories)-1 {
			fmt.Println(pterm.LightCyan("│"))
		}
	}

	// Close timeline
	fmt.Println(pterm.LightCyan("└─"))

	fmt.Println()
	return nil
}

// formatStatus formats a status string with color
func formatStatus(status string) string {
	switch status {
	case "todo":
		return pterm.LightBlue(status)
	case "in_progress":
		return pterm.LightYellow(status)
	case "ready_for_review":
		return pterm.LightMagenta(status)
	case "completed":
		return pterm.LightGreen(status)
	case "blocked":
		return pterm.LightRed(status)
	default:
		return status
	}
}

func outputHistoryCSV(taskKey string, histories []*models.TaskHistory) error {
	// Convert to export records
	records := formatters.ConvertToExportRecords(histories, taskKey)

	// Format as CSV
	csv, err := formatters.FormatHistoryCSV(records)
	if err != nil {
		return fmt.Errorf("failed to format history as CSV: %w", err)
	}

	fmt.Print(csv)
	return nil
}

func outputHistoryJSONExport(taskKey string, histories []*models.TaskHistory) error {
	// Convert to export records
	records := formatters.ConvertToExportRecords(histories, taskKey)

	// Format as JSON
	jsonStr, err := formatters.FormatHistoryJSON(records)
	if err != nil {
		return fmt.Errorf("failed to format history as JSON: %w", err)
	}

	fmt.Println(jsonStr)
	return nil
}

func init() {
	// Register history command
	taskHistoryCmd.Flags().StringVar(&taskHistoryFormat, "format", "", "Output format (csv, json)")
	taskCmd.AddCommand(taskHistoryCmd)
}
