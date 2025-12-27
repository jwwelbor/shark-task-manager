package commands

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// featureCriteriaCmd shows aggregated criteria for a feature
var featureCriteriaCmd = &cobra.Command{
	Use:   "criteria <feature-key>",
	Short: "Show aggregated acceptance criteria for a feature",
	Long: `Show aggregated acceptance criteria across all tasks in a feature.

Displays:
  - Total criteria count across all feature tasks
  - Breakdown by status: pending, in_progress, complete, failed, na
  - Overall completion percentage
  - Optional per-task breakdown with --by-task flag

Examples:
  shark feature criteria E10-F04
  shark feature criteria E10-F04 --by-task
  shark feature criteria E10-F04 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureCriteria,
}

// TaskCriteriaSummary represents criteria summary for a single task
type TaskCriteriaSummary struct {
	TaskKey         string  `json:"task_key"`
	TaskTitle       string  `json:"task_title"`
	TotalCount      int     `json:"total_count"`
	PendingCount    int     `json:"pending_count"`
	InProgressCount int     `json:"in_progress_count"`
	CompleteCount   int     `json:"complete_count"`
	FailedCount     int     `json:"failed_count"`
	NACount         int     `json:"na_count"`
	CompletionPct   float64 `json:"completion_pct"`
}

// FeatureCriteriaSummary represents aggregated criteria for a feature
type FeatureCriteriaSummary struct {
	FeatureKey      string                `json:"feature_key"`
	FeatureTitle    string                `json:"feature_title"`
	TaskCount       int                   `json:"task_count"`
	TotalCount      int                   `json:"total_count"`
	PendingCount    int                   `json:"pending_count"`
	InProgressCount int                   `json:"in_progress_count"`
	CompleteCount   int                   `json:"complete_count"`
	FailedCount     int                   `json:"failed_count"`
	NACount         int                   `json:"na_count"`
	CompletionPct   float64               `json:"completion_pct"`
	TaskSummaries   []TaskCriteriaSummary `json:"task_summaries,omitempty"`
}

// runFeatureCriteria handles the feature criteria command
func runFeatureCriteria(cmd *cobra.Command, args []string) error {
	featureKey := args[0]
	byTask, _ := cmd.Flags().GetBool("by-task")

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

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	criteriaRepo := repository.NewTaskCriteriaRepository(dbWrapper)

	// Get feature by key
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature %s not found", featureKey)
	}

	// Get all tasks for the feature
	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return fmt.Errorf("failed to get tasks for feature: %w", err)
	}

	if len(tasks) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"feature_key": featureKey,
				"message":     "No tasks found for feature",
			})
		}
		fmt.Printf("No tasks found for feature %s\n", featureKey)
		return nil
	}

	// Aggregate criteria across all tasks
	featureSummary := FeatureCriteriaSummary{
		FeatureKey:    featureKey,
		FeatureTitle:  feature.Title,
		TaskCount:     len(tasks),
		TaskSummaries: make([]TaskCriteriaSummary, 0),
	}

	for _, task := range tasks {
		summary, err := criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
		if err != nil {
			return fmt.Errorf("failed to get criteria summary for task %s: %w", task.Key, err)
		}

		// Aggregate counts
		featureSummary.TotalCount += summary.TotalCount
		featureSummary.PendingCount += summary.PendingCount
		featureSummary.InProgressCount += summary.InProgressCount
		featureSummary.CompleteCount += summary.CompleteCount
		featureSummary.FailedCount += summary.FailedCount
		featureSummary.NACount += summary.NACount

		// Add task summary if requested
		if byTask && summary.TotalCount > 0 {
			featureSummary.TaskSummaries = append(featureSummary.TaskSummaries, TaskCriteriaSummary{
				TaskKey:         task.Key,
				TaskTitle:       task.Title,
				TotalCount:      summary.TotalCount,
				PendingCount:    summary.PendingCount,
				InProgressCount: summary.InProgressCount,
				CompleteCount:   summary.CompleteCount,
				FailedCount:     summary.FailedCount,
				NACount:         summary.NACount,
				CompletionPct:   summary.CompletionPct,
			})
		}
	}

	// Calculate overall completion percentage
	if featureSummary.TotalCount > 0 {
		featureSummary.CompletionPct = float64(featureSummary.CompleteCount+featureSummary.NACount) / float64(featureSummary.TotalCount) * 100.0
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(featureSummary)
	}

	// Human-readable output
	if featureSummary.TotalCount == 0 {
		fmt.Printf("Feature %s: %s\n", featureKey, feature.Title)
		fmt.Println("No acceptance criteria found for this feature")
		return nil
	}

	fmt.Printf("Feature %s: %s\n\n", featureKey, feature.Title)
	fmt.Printf("Overall Progress: %.0f%% complete (%d/%d criteria)\n",
		featureSummary.CompletionPct,
		featureSummary.CompleteCount+featureSummary.NACount,
		featureSummary.TotalCount)
	fmt.Printf("  Complete: %d | Pending: %d | In Progress: %d | Failed: %d | N/A: %d\n",
		featureSummary.CompleteCount,
		featureSummary.PendingCount,
		featureSummary.InProgressCount,
		featureSummary.FailedCount,
		featureSummary.NACount)
	fmt.Printf("  Tasks: %d\n", featureSummary.TaskCount)

	// Show per-task breakdown if requested
	if byTask && len(featureSummary.TaskSummaries) > 0 {
		fmt.Println("\nPer-Task Breakdown:")
		for _, taskSummary := range featureSummary.TaskSummaries {
			fmt.Printf("\n  %s: %s\n", taskSummary.TaskKey, taskSummary.TaskTitle)
			fmt.Printf("    %.0f%% complete (%d/%d criteria)\n",
				taskSummary.CompletionPct,
				taskSummary.CompleteCount+taskSummary.NACount,
				taskSummary.TotalCount)
			fmt.Printf("    Complete: %d | Pending: %d | In Progress: %d | Failed: %d | N/A: %d\n",
				taskSummary.CompleteCount,
				taskSummary.PendingCount,
				taskSummary.InProgressCount,
				taskSummary.FailedCount,
				taskSummary.NACount)
		}
	}

	return nil
}

func init() {
	// Add criteria subcommand to feature command
	featureCmd.AddCommand(featureCriteriaCmd)

	// Flags for criteria command
	featureCriteriaCmd.Flags().BoolP("by-task", "t", false, "Show per-task breakdown")
}
