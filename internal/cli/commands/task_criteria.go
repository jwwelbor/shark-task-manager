package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// taskCriteriaCmd is the parent command for criteria operations
var taskCriteriaCmd = &cobra.Command{
	Use:   "criteria",
	Short: "Manage task acceptance criteria",
	Long:  `Import, view, and manage acceptance criteria for tasks.`,
}

// taskCriteriaImportCmd imports criteria from task markdown file
var taskCriteriaImportCmd = &cobra.Command{
	Use:   "import <task-key>",
	Short: "Import acceptance criteria from task markdown file",
	Long: `Import acceptance criteria from a task markdown file into the database.

Criteria are extracted from markdown checklist format:
  - [ ] unchecked criterion (pending)
  - [x] checked criterion (complete)

Examples:
  shark task criteria import T-E10-F04-001
  shark task criteria import T-E10-F04-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskCriteriaImport,
}

// taskCriteriaListCmd lists criteria for a task
var taskCriteriaListCmd = &cobra.Command{
	Use:   "list <task-key>",
	Short: "List acceptance criteria for a task",
	Long: `List all acceptance criteria for a task with status summary.

Displays:
  - Total, complete, pending, failed, in_progress, and na counts
  - Each criterion with status icon: ✓ (complete), ✗ (failed), ○ (pending), ◐ (in_progress), − (na)
  - Percentage calculation: "85% complete (6/7 criteria)"
  - Criteria ordered by status (failed first, then pending, in_progress, complete)

Examples:
  shark task criteria list T-E10-F04-001
  shark task criteria list T-E10-F04-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskCriteriaList,
}

// taskCriteriaCheckCmd marks a criterion as complete
var taskCriteriaCheckCmd = &cobra.Command{
	Use:   "check <task-key> <criterion-id>",
	Short: "Mark a criterion as complete",
	Long: `Mark an acceptance criterion as complete (verified).

Updates status to 'complete' and sets verified_at timestamp.
Optional --note parameter adds verification notes.

Examples:
  shark task criteria check T-E10-F04-001 5
  shark task criteria check T-E10-F04-001 5 --note "Verified with unit tests"
  shark task criteria check T-E10-F04-001 5 --json`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskCriteriaCheck,
}

// taskCriteriaFailCmd marks a criterion as failed
var taskCriteriaFailCmd = &cobra.Command{
	Use:   "fail <task-key> <criterion-id>",
	Short: "Mark a criterion as failed",
	Long: `Mark an acceptance criterion as failed.

Updates status to 'failed' and requires a --note explaining the failure reason.
Failed criteria are highlighted in output.

Examples:
  shark task criteria fail T-E10-F04-001 5 --note "Performance threshold not met"
  shark task criteria fail T-E10-F04-001 5 --note "Missing edge case handling" --json`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskCriteriaFail,
}

// runTaskCriteriaImport handles the task criteria import command
func runTaskCriteriaImport(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	taskKey := args[0]

	// Step 2: Call service
	svc := cli.GetCriteriaService()
	importCount, err := svc.ImportCriteriaFromFile(cmd.Context(), taskKey)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if importCount == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"task_key": taskKey,
				"imported": 0,
				"message":  "No criteria found in task file",
			})
		}
		fmt.Printf("No criteria found in task file for %s\n", taskKey)
		return nil
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key": taskKey,
			"imported": importCount,
			"message":  fmt.Sprintf("Imported %d acceptance criteria for %s", importCount, taskKey),
		})
	}

	fmt.Printf("Imported %d acceptance criteria for %s\n", importCount, taskKey)
	return nil
}

// runTaskCriteriaList handles the task criteria list command
func runTaskCriteriaList(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	taskKey := args[0]

	// Step 2: Call service
	svc := cli.GetCriteriaService()
	result, err := svc.GetTaskCriteriaWithSummary(cmd.Context(), taskKey)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key":          result.TaskKey,
			"title":             result.Title,
			"criteria":          result.Criteria,
			"total_count":       result.Summary.TotalCount,
			"pending_count":     result.Summary.PendingCount,
			"in_progress_count": result.Summary.InProgressCount,
			"complete_count":    result.Summary.CompleteCount,
			"failed_count":      result.Summary.FailedCount,
			"na_count":          result.Summary.NACount,
			"completion_pct":    result.Summary.CompletionPct,
		})
	}

	if len(result.Criteria) == 0 {
		fmt.Printf("No criteria found for task %s\n", taskKey)
		return nil
	}

	fmt.Printf("Task %s: %s\n\n", result.TaskKey, result.Title)
	fmt.Printf("Criteria Summary: %.0f%% complete (%d/%d criteria)\n",
		result.Summary.CompletionPct,
		result.Summary.CompleteCount+result.Summary.NACount,
		result.Summary.TotalCount)
	fmt.Printf("  Complete: %d | Pending: %d | In Progress: %d | Failed: %d | N/A: %d\n\n",
		result.Summary.CompleteCount,
		result.Summary.PendingCount,
		result.Summary.InProgressCount,
		result.Summary.FailedCount,
		result.Summary.NACount)

	// Sort criteria: failed first, then pending, in_progress, complete, na
	sortedCriteria := make([]*models.TaskCriteria, 0, len(result.Criteria))

	for _, c := range result.Criteria {
		if c.Status == models.CriteriaStatusFailed {
			sortedCriteria = append(sortedCriteria, c)
		}
	}
	for _, c := range result.Criteria {
		if c.Status == models.CriteriaStatusPending {
			sortedCriteria = append(sortedCriteria, c)
		}
	}
	for _, c := range result.Criteria {
		if c.Status == models.CriteriaStatusInProgress {
			sortedCriteria = append(sortedCriteria, c)
		}
	}
	for _, c := range result.Criteria {
		if c.Status == models.CriteriaStatusComplete {
			sortedCriteria = append(sortedCriteria, c)
		}
	}
	for _, c := range result.Criteria {
		if c.Status == models.CriteriaStatusNA {
			sortedCriteria = append(sortedCriteria, c)
		}
	}

	fmt.Println("Criteria:")
	for _, criterion := range sortedCriteria {
		icon := getCriteriaStatusIcon(criterion.Status)
		fmt.Printf("  [%d] %s %s\n", criterion.ID, icon, criterion.Criterion)

		if criterion.VerificationNotes != nil && *criterion.VerificationNotes != "" {
			fmt.Printf("      Note: %s\n", *criterion.VerificationNotes)
		}
	}

	return nil
}

// runTaskCriteriaCheck handles the task criteria check command
func runTaskCriteriaCheck(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	taskKey := args[0]
	var criterionID int64
	if _, err := fmt.Sscanf(args[1], "%d", &criterionID); err != nil {
		return fmt.Errorf("invalid criterion ID: %s", args[1])
	}

	note, _ := cmd.Flags().GetString("note")
	var notePtr *string
	if note != "" {
		notePtr = &note
	}

	// Step 2: Call service
	svc := cli.GetCriteriaService()
	result, err := svc.CheckCriterionForTask(cmd.Context(), taskKey, criterionID, notePtr)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key":       result.TaskKey,
			"criterion_id":   result.CriterionID,
			"status":         "complete",
			"total_count":    result.Summary.TotalCount,
			"complete_count": result.Summary.CompleteCount,
			"completion_pct": result.Summary.CompletionPct,
		})
	}

	fmt.Printf("Criterion %d marked as complete\n", criterionID)
	fmt.Printf("Progress: %.0f%% complete (%d/%d criteria)\n",
		result.Summary.CompletionPct,
		result.Summary.CompleteCount+result.Summary.NACount,
		result.Summary.TotalCount)

	return nil
}

// runTaskCriteriaFail handles the task criteria fail command
func runTaskCriteriaFail(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	taskKey := args[0]
	var criterionID int64
	if _, err := fmt.Sscanf(args[1], "%d", &criterionID); err != nil {
		return fmt.Errorf("invalid criterion ID: %s", args[1])
	}

	note, _ := cmd.Flags().GetString("note")
	if note == "" {
		return fmt.Errorf("--note flag is required for failed criteria")
	}

	// Step 2: Call service
	svc := cli.GetCriteriaService()
	result, err := svc.FailCriterionForTask(cmd.Context(), taskKey, criterionID, &note)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key":       result.TaskKey,
			"criterion_id":   result.CriterionID,
			"status":         "failed",
			"note":           note,
			"total_count":    result.Summary.TotalCount,
			"failed_count":   result.Summary.FailedCount,
			"completion_pct": result.Summary.CompletionPct,
		})
	}

	fmt.Printf("Criterion %d marked as failed\n", criterionID)
	fmt.Printf("Reason: %s\n", note)
	fmt.Printf("Progress: %.0f%% complete (%d/%d criteria)\n",
		result.Summary.CompletionPct,
		result.Summary.CompleteCount+result.Summary.NACount,
		result.Summary.TotalCount)

	return nil
}

// getCriteriaStatusIcon returns the icon for a criterion status
func getCriteriaStatusIcon(status models.CriteriaStatus) string {
	switch status {
	case models.CriteriaStatusComplete:
		return "✓"
	case models.CriteriaStatusFailed:
		return "✗"
	case models.CriteriaStatusPending:
		return "○"
	case models.CriteriaStatusInProgress:
		return "◐"
	case models.CriteriaStatusNA:
		return "−"
	default:
		return "?"
	}
}

func init() {
	// Add criteria subcommand to task command
	taskCmd.AddCommand(taskCriteriaCmd)

	// Add subcommands to criteria command
	taskCriteriaCmd.AddCommand(taskCriteriaImportCmd)
	taskCriteriaCmd.AddCommand(taskCriteriaListCmd)
	taskCriteriaCmd.AddCommand(taskCriteriaCheckCmd)
	taskCriteriaCmd.AddCommand(taskCriteriaFailCmd)

	// Flags for check command
	taskCriteriaCheckCmd.Flags().StringP("note", "n", "", "Verification notes (optional)")

	// Flags for fail command
	taskCriteriaFailCmd.Flags().StringP("note", "n", "", "Failure reason (required)")
	_ = taskCriteriaFailCmd.MarkFlagRequired("note")
}
