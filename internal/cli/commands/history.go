package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/formatters"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
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
	Use:     "history [KEY]",
	Short:   "View entity or project-wide history",
	GroupID: "manage",
	Long: `View status change history for a specific entity or project-wide activity log.

When given a specific entity key, shows the status history for that entity.
When given no arguments or epic/feature filters, shows project-wide task history.

Entity keys are auto-detected from format:
  E##               Epic history
  E##-F##           Feature history
  E##-F##-###       Task history (also T-E##-F##-###)
  B###              Bug history
  CC-###            Change-card history

Project-wide filtering:
  (no args)         View history for all tasks
  EPIC FEATURE      View history for tasks in specific feature (e.g., E04 F01)`,
	Example: `  # View history for a specific task
  shark history T-E21-F10-001
  shark history E21-F10-001

  # View history for a specific feature
  shark history E07-F01

  # View history for a specific epic
  shark history E07

  # View history for a bug or change-card
  shark history B001
  shark history CC-001

  # Project-wide history
  shark history --epic=E05 --limit=20

  # Project-wide with filters
  shark history E05 F02
  shark history --agent=backend --since="2025-12-27T10:00:00Z"

  # Output as JSON
  shark history E07-F01-001 --json

  # Pagination
  shark history --limit=10 --offset=10`,
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

	historyCmd.Hidden = true
}

// detectHistoryEntityKey checks if the args represent a single entity key.
// Returns (entityType, key, true) if an entity key is detected,
// or ("", "", false) if args should be handled as project-wide history filters.
func detectHistoryEntityKey(args []string) (entityType string, key string, isEntity bool) {
	if len(args) != 1 {
		return "", "", false
	}

	detected := DetectEntityType(args[0])
	if detected == "unknown" {
		return "", "", false
	}

	return detected, args[0], true
}

// runEntityHistory handles `shark history <key>` for a specific entity.
// It uses ParseGetArgs for key normalization (e.g., E21-F10-001 -> T-E21-F10-001).
func runEntityHistory(cmd *cobra.Command, entityType string, key string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Normalize entity type (ADR-1: change_card -> change)
	if entityType == "change_card" {
		entityType = "change"
	}

	svc := cli.GetEntityHistoryService()
	history, err := svc.GetHistory(ctx, models.EntityType(entityType), key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			cli.Error(fmt.Sprintf("%s %s not found", entityType, key))
			os.Exit(1)
		}
		return fmt.Errorf("failed to get history: %w", err)
	}

	// Apply limit
	if historyLimit > 0 && len(history) > historyLimit {
		history = history[:historyLimit]
	}

	entries := make([]StatusHistoryEntry, 0, len(history))
	for _, h := range history {
		entry := StatusHistoryEntry{
			Timestamp: h.ChangedAt.Format(time.RFC3339),
			NewStatus: h.ToStatus,
		}
		if h.FromStatus != nil {
			entry.OldStatus = *h.FromStatus
		}
		if h.ChangedBy != nil {
			entry.Agent = *h.ChangedBy
		}
		if h.Notes != nil {
			entry.Notes = *h.Notes
		}
		entries = append(entries, entry)
	}

	result := &StatusHistoryResult{
		EntityType: entityType,
		EntityKey:  key,
		History:    entries,
		Total:      len(entries),
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	if len(entries) == 0 {
		cli.Info(fmt.Sprintf("No history found for %s %s", entityType, key))
		return nil
	}

	fmt.Printf("\nHistory for %s %s (%d entries)\n", entityType, key, len(entries))
	fmt.Println(strings.Repeat("-", 80))

	headers := []string{"Timestamp", "Old Status", "New Status", "Agent", "Notes"}
	// CC-036: Notes column width scales with the resolved console width.
	// Reserved width (~75 cols) accounts for the actual chrome: timestamp
	// (RFC3339, ≤25) + old status (≤12) + new status (≤12) + agent (≤12)
	// + four " | " separators (12), plus a small margin. Notes are
	// right-padded via fitColumn so pterm renders the column at full
	// width instead of shrinking it to the widest actual note.
	notesMax := cli.TitleColumnWidth(75)
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		oldStatus := e.OldStatus
		if oldStatus == "" {
			oldStatus = "(initial)"
		}
		rows = append(rows, []string{e.Timestamp, oldStatus, e.NewStatus, e.Agent, fitColumn(e.Notes, notesMax)})
	}

	cli.OutputTable(headers, rows)
	return nil
}

func runHistory(cmd *cobra.Command, args []string) error {
	// Check if the argument is a specific entity key (task, feature, epic, bug, change-card)
	if _, _, isEntity := detectHistoryEntityKey(args); isEntity {
		// Use ParseGetArgs for proper key normalization (e.g., E21-F10-001 -> T-E21-F10-001)
		entityType, key, err := ParseGetArgs(args)
		if err != nil {
			return fmt.Errorf("invalid key format: %w", err)
		}
		return runEntityHistory(cmd, entityType, key)
	}

	ctx := cmd.Context()

	// Parse positional arguments for project-wide history (0 args or 2 args: EPIC FEATURE)
	_, positionalEpic, positionalFeature, err := ParseListArgs(args)
	if err != nil {
		return err
	}

	// Build service-layer filters
	filters := services.HistoryFilters{
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

	// Positional arguments take priority over flags for epic and feature
	epicKey := historyEpic
	if positionalEpic != nil {
		epicKey = *positionalEpic
	}
	if epicKey != "" {
		filters.EpicKey = &epicKey
	}

	featureKey := historyFeature
	if positionalFeature != nil {
		featureKey = *positionalFeature
	}
	if featureKey != "" {
		filters.FeatureKey = &featureKey
	}
	if historyOldStatus != "" {
		filters.OldStatus = &historyOldStatus
	}
	if historyNewStatus != "" {
		filters.NewStatus = &historyNewStatus
	}

	// Retrieve history via service
	svc := cli.GetTaskServiceWithHistory()
	histories, err := svc.ListHistory(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to retrieve history: %w", err)
	}

	// Handle format-based export
	if historyFormat != "" {
		return outputHistoryExport(ctx, histories, svc, historyFormat)
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
		task, err := svc.GetTaskByID(ctx, h.TaskID)
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
	// CC-036: Notes column width scales with the resolved console width.
	// Reserved width (~85 cols) accounts for the actual chrome: timestamp
	// (19 for "YYYY-MM-DD HH:MM:SS") + task key (~13 for T-EXX-FYY-ZZZ) +
	// old status (≤12) + new status (≤12) + agent (≤12) + five " | "
	// separators (15), plus a small margin. Notes are right-padded via
	// fitColumn so pterm renders the column at full width instead of
	// shrinking it to the widest actual note.
	notesMax := cli.TitleColumnWidth(85)
	var rows [][]string
	for _, record := range displayRecords {
		rows = append(rows, []string{
			record.Timestamp,
			record.TaskKey,
			record.OldStatus,
			record.NewStatus,
			record.Agent,
			fitColumn(record.Notes, notesMax),
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

// taskByIDGetter is a minimal interface for retrieving a task by database ID.
// Used by outputHistoryExport to resolve task keys from history records.
type taskByIDGetter interface {
	GetTaskByID(ctx context.Context, id int64) (*models.Task, error)
}

func outputHistoryExport(ctx context.Context, histories []*models.TaskHistory, svc taskByIDGetter, format string) error {
	// Build export records with task keys
	var historyWithTasks []formatters.HistoryWithTask
	for _, h := range histories {
		task, err := svc.GetTaskByID(ctx, h.TaskID)
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
