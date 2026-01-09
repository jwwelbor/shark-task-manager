package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// taskContextCmd represents the task context command group
var taskContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage task context data",
	Long: `Manage structured resume context data for tasks.

Context data includes progress tracking, implementation decisions, open questions,
blockers, acceptance criteria status, and related tasks.

Examples:
  shark task context set T-E10-F05-001 --field current_step "Implementing API endpoint"
  shark task context get T-E10-F05-001
  shark task context clear T-E10-F05-001`,
}

// taskContextSetCmd sets a context field
var taskContextSetCmd = &cobra.Command{
	Use:   "set <task-key>",
	Short: "Set or update task context field",
	Long: `Set or update a specific field in task context data.

Supported fields:
  - current_step: String describing current work step
  - completed_steps: JSON array of completed steps
  - remaining_steps: JSON array of remaining steps
  - implementation_decisions: JSON object with decision key-value pairs
  - open_questions: JSON array of question strings
  - blockers: JSON array of blocker objects
  - acceptance_criteria_status: JSON array of criterion objects
  - related_tasks: JSON array of task keys

Examples:
  shark task context set T-E10-F05-001 --field current_step "Implementing dropdown menu"
  shark task context set T-E10-F05-001 --field completed_steps '["Step 1","Step 2"]'
  shark task context set T-E10-F05-001 --field open_questions '["Question 1?"]'`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskContextSet,
}

// taskContextGetCmd gets task context
var taskContextGetCmd = &cobra.Command{
	Use:   "get <task-key>",
	Short: "Get task context data",
	Long:  `Display the current context data for a task.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskContextGet,
}

// taskContextClearCmd clears task context
var taskContextClearCmd = &cobra.Command{
	Use:   "clear <task-key>",
	Short: "Clear task context data",
	Long:  `Remove all context data from a task.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskContextClear,
}

func init() {
	// Add context subcommands to task
	taskCmd.AddCommand(taskContextCmd)
	taskContextCmd.AddCommand(taskContextSetCmd)
	taskContextCmd.AddCommand(taskContextGetCmd)
	taskContextCmd.AddCommand(taskContextClearCmd)

	// Flags for set command
	taskContextSetCmd.Flags().String("field", "", "Context field to update (required)")
	taskContextSetCmd.Flags().String("value", "", "Field value (required)")
	_ = taskContextSetCmd.MarkFlagRequired("field")
	_ = taskContextSetCmd.MarkFlagRequired("value")
}

// runTaskContextSet sets or updates a context field
func runTaskContextSet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]
	field, _ := cmd.Flags().GetString("field")
	value, _ := cmd.Flags().GetString("value")

	// Validate field name
	validFields := map[string]bool{
		"current_step":               true,
		"completed_steps":            true,
		"remaining_steps":            true,
		"implementation_decisions":   true,
		"open_questions":             true,
		"blockers":                   true,
		"acceptance_criteria_status": true,
		"related_tasks":              true,
	}

	if !validFields[field] {
		cli.Error(fmt.Sprintf("Invalid context field: %s", field))
		cli.Info("Supported fields: current_step, completed_steps, remaining_steps, implementation_decisions, open_questions, blockers, acceptance_criteria_status, related_tasks")
		os.Exit(3)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	repo := repository.NewTaskRepository(repoDb)

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		cli.Info("Use 'shark task list' to see available tasks")
		os.Exit(1)
	}

	// Parse existing context data or create new
	var contextData *models.ContextData
	if task.ContextData != nil && *task.ContextData != "" {
		contextData, err = models.FromJSON(*task.ContextData)
		if err != nil {
			return fmt.Errorf("failed to parse existing context data: %w", err)
		}
	} else {
		contextData = &models.ContextData{}
	}

	// Update the specified field
	if err := updateContextField(contextData, field, value); err != nil {
		cli.Error(fmt.Sprintf("Failed to update field: %v", err))
		os.Exit(3)
	}

	// Validate and convert back to JSON
	jsonStr, err := contextData.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert context data to JSON: %w", err)
	}

	// Update task in database
	task.ContextData = &jsonStr
	if err := repo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key": taskKey,
			"field":    field,
			"success":  true,
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Updated context field '%s' for task %s", field, taskKey))
	return nil
}

// updateContextField updates a specific field in the context data
func updateContextField(cd *models.ContextData, field, value string) error {
	switch field {
	case "current_step":
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.CurrentStep = &value

	case "completed_steps":
		var steps []string
		if err := json.Unmarshal([]byte(value), &steps); err != nil {
			return fmt.Errorf("invalid JSON for completed_steps: %w", err)
		}
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.CompletedSteps = steps

	case "remaining_steps":
		var steps []string
		if err := json.Unmarshal([]byte(value), &steps); err != nil {
			return fmt.Errorf("invalid JSON for remaining_steps: %w", err)
		}
		if cd.Progress == nil {
			cd.Progress = &models.ProgressContext{}
		}
		cd.Progress.RemainingSteps = steps

	case "implementation_decisions":
		var decisions map[string]string
		if err := json.Unmarshal([]byte(value), &decisions); err != nil {
			return fmt.Errorf("invalid JSON for implementation_decisions: %w", err)
		}
		if cd.ImplementationDecisions == nil {
			cd.ImplementationDecisions = make(map[string]string)
		}
		// Merge decisions
		for k, v := range decisions {
			cd.ImplementationDecisions[k] = v
		}

	case "open_questions":
		var questions []string
		if err := json.Unmarshal([]byte(value), &questions); err != nil {
			return fmt.Errorf("invalid JSON for open_questions: %w", err)
		}
		cd.OpenQuestions = questions

	case "blockers":
		var blockers []models.BlockerContext
		if err := json.Unmarshal([]byte(value), &blockers); err != nil {
			return fmt.Errorf("invalid JSON for blockers: %w", err)
		}
		cd.Blockers = blockers

	case "acceptance_criteria_status":
		var criteria []models.AcceptanceCriterionContext
		if err := json.Unmarshal([]byte(value), &criteria); err != nil {
			return fmt.Errorf("invalid JSON for acceptance_criteria_status: %w", err)
		}
		cd.AcceptanceCriteriaStatus = criteria

	case "related_tasks":
		var tasks []string
		if err := json.Unmarshal([]byte(value), &tasks); err != nil {
			return fmt.Errorf("invalid JSON for related_tasks: %w", err)
		}
		cd.RelatedTasks = tasks
	}

	return nil
}

// runTaskContextGet retrieves and displays task context
func runTaskContextGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	repo := repository.NewTaskRepository(repoDb)

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		os.Exit(1)
	}

	// Parse context data
	var contextData *models.ContextData
	if task.ContextData != nil && *task.ContextData != "" {
		contextData, err = models.FromJSON(*task.ContextData)
		if err != nil {
			return fmt.Errorf("failed to parse context data: %w", err)
		}
	} else {
		contextData = &models.ContextData{}
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":     taskKey,
			"context_data": contextData,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("Context for Task %s\n\n", taskKey)

	// Progress
	if contextData.Progress != nil {
		fmt.Println("Progress:")
		if contextData.Progress.CurrentStep != nil {
			fmt.Printf("  Current Step: %s\n", *contextData.Progress.CurrentStep)
		}
		if len(contextData.Progress.CompletedSteps) > 0 {
			fmt.Println("  Completed Steps:")
			for _, step := range contextData.Progress.CompletedSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		if len(contextData.Progress.RemainingSteps) > 0 {
			fmt.Println("  Remaining Steps:")
			for _, step := range contextData.Progress.RemainingSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		fmt.Println()
	}

	// Implementation Decisions
	if len(contextData.ImplementationDecisions) > 0 {
		fmt.Println("Implementation Decisions:")
		for k, v := range contextData.ImplementationDecisions {
			fmt.Printf("  %s: %s\n", k, v)
		}
		fmt.Println()
	}

	// Open Questions
	if len(contextData.OpenQuestions) > 0 {
		fmt.Println("Open Questions:")
		for _, q := range contextData.OpenQuestions {
			fmt.Printf("  - %s\n", q)
		}
		fmt.Println()
	}

	// Blockers
	if len(contextData.Blockers) > 0 {
		fmt.Println("Blockers:")
		for _, b := range contextData.Blockers {
			fmt.Printf("  - %s (%s) - blocked since %s\n", b.Description, b.BlockerType, b.BlockedSince.Format("2006-01-02 15:04"))
		}
		fmt.Println()
	}

	// Acceptance Criteria
	if len(contextData.AcceptanceCriteriaStatus) > 0 {
		fmt.Println("Acceptance Criteria:")
		for _, ac := range contextData.AcceptanceCriteriaStatus {
			fmt.Printf("  [%s] %s\n", ac.Status, ac.Criterion)
		}
		fmt.Println()
	}

	// Related Tasks
	if len(contextData.RelatedTasks) > 0 {
		fmt.Println("Related Tasks:")
		for _, t := range contextData.RelatedTasks {
			fmt.Printf("  - %s\n", t)
		}
		fmt.Println()
	}

	if contextData.Progress == nil && len(contextData.ImplementationDecisions) == 0 &&
		len(contextData.OpenQuestions) == 0 && len(contextData.Blockers) == 0 &&
		len(contextData.AcceptanceCriteriaStatus) == 0 && len(contextData.RelatedTasks) == 0 {
		fmt.Println("No context data available for this task.")
	}

	return nil
}

// runTaskContextClear clears task context data
func runTaskContextClear(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	repo := repository.NewTaskRepository(repoDb)

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		os.Exit(1)
	}

	// Clear context data
	emptyJSON := "{}"
	task.ContextData = &emptyJSON

	// Update task in database
	if err := repo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key": taskKey,
			"success":  true,
			"message":  "Context data cleared",
		}
		return cli.OutputJSON(output)
	}

	cli.Success(fmt.Sprintf("Cleared context data for task %s", taskKey))
	return nil
}
