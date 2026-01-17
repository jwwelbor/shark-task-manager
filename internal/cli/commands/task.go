package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/pathresolver"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/validation"
	"github.com/spf13/cobra"
)

// getRelativePathTask converts an absolute path to relative path from project root
func getRelativePathTask(absPath string, projectRoot string) string {
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return absPath // Fall back to absolute path if conversion fails
	}
	return relPath
}

// triggerStatusCascade triggers cascading status updates for parent feature and epic
// after a task status change. This is informational - errors are logged but don't
// fail the operation.
func triggerStatusCascade(ctx context.Context, dbWrapper *repository.DB, featureID int64) {
	// Load workflow config
	configPath, err := cli.GetConfigPath()
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get config path: %v\n", err)
	}
	cfg, err := config.LoadWorkflowConfig(configPath)
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
	}

	calcService := status.NewCalculationService(dbWrapper, cfg)
	results, err := calcService.CascadeFromFeatureID(ctx, featureID)
	if err != nil {
		cli.Warning(fmt.Sprintf("Status cascade failed: %v", err))
		return
	}

	// Report any status changes
	for _, r := range results {
		if r.WasChanged {
			cli.Info(fmt.Sprintf("  %s %s status: %s -> %s", r.EntityType, r.EntityKey, r.PreviousStatus, r.NewStatus))
		}
	}
}

// taskCmd represents the task command group
var taskCmd = &cobra.Command{
	Use:     "task",
	Short:   "Manage tasks",
	GroupID: "essentials",
	Long: `Task lifecycle operations including listing, creating, updating, and managing task status.

Examples:
  shark task list                 List all tasks
  shark task get T-E01-F01-001   Get task details
  shark task create              Create a new task
  shark task start T-E01-F01-001 Start working on a task
  shark task complete T-E01-F01-001  Mark task as complete`,
}

// taskListCmd lists tasks
var taskListCmd = &cobra.Command{
	Use:   "list [EPIC] [FEATURE]",
	Short: "List tasks",
	Long: `List tasks with optional filtering by status, epic, feature, or agent.

By default, completed tasks are hidden. Use --show-all to include them.

Positional Arguments:
  EPIC      Optional epic key (E##) to filter by epic (e.g., E04)
  FEATURE   Optional feature key (F## or E##-F##) to filter by feature (e.g., F01 or E04-F01)

Examples:
  shark task list                      List all non-completed tasks
  shark task list --show-all           List all tasks including completed
  shark task list E04                  List all non-completed tasks in epic E04
  shark task list E04 F01              List tasks in epic E04, feature F01
  shark task list E04-F01              Same as above (combined format)
  shark task list --status=todo        List tasks with status 'todo'
  shark task list --status=completed   List only completed tasks
  shark task list --epic=E04           Flag syntax (still supported)
  shark task list --json               Output as JSON`,
	RunE: runTaskList,
}

// taskGetCmd gets a specific task
var taskGetCmd = &cobra.Command{
	Use:   "get <task-key>",
	Short: "Get task details",
	Long: `Display detailed information about a specific task.

Supports multiple key formats:
  - Full key: T-E04-F02-001
  - Numeric key: 001 (if unique within project)
  - Slugged key: T-E04-F02-001-task-name

Examples:
  shark task get T-E04-F02-001                     Get task by full key
  shark task get T-E04-F02-001-user-auth           Get task by slugged key
  shark task get T-E04-F02-001 --json              Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskGet,
}

// taskCreateCmd creates a new task
var taskCreateCmd = &cobra.Command{
	Use:   "create [EPIC] [FEATURE] <title> [flags]",
	Short: "Create a new task",
	Long: `Create a new task with automatic key generation and file creation.

The --agent flag is optional and accepts any string value. If not provided, defaults to "general".
The --template flag allows using a custom task template file.
The --file flag allows specifying a custom file path (relative to project root, must end in .md).
The --create flag creates the file if it doesn't exist (when using --file).

Positional Arguments:
  EPIC      Optional epic key (E##) - can also be specified with --epic flag
  FEATURE   Optional feature key (F## or E##-F##) - can also be specified with --feature flag
  TITLE     Task title (required)

Examples:
  # Positional argument syntax (new, recommended)
  shark task create E07 F20 "Implement user authentication"
  shark task create E07 F20 "Build Login" --agent=frontend
  shark task create E07-F20 "User Service" --agent=backend --priority=5

  # Custom file path examples
  shark task create E01 F02 "Migration task" --file="docs/tasks/existing.md"          # Assigns existing file
  shark task create E01 F02 "New task" --file="docs/tasks/new.md" --create           # Creates new file

  # Flag syntax (still supported for backward compatibility)
  shark task create "Build Login" --epic=E01 --feature=F02
  shark task create "Build Login" --epic=E01 --feature=F02 --agent=frontend
  shark task create "User Service" --epic=E01 --feature=F02 --agent=backend --priority=5
  shark task create "Database task" --epic=E01 --feature=F02 --agent=database-admin
  shark task create "Custom task" --epic=E01 --feature=F02 --template=./my-template.md`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runTaskCreate,
}

// taskStartCmd starts a task
var taskStartCmd = &cobra.Command{
	Use:   "start <task-key>",
	Short: "Start working on a task",
	Long: `Mark a task as in_progress and update timestamps.

Use --force to bypass status transition validation. This allows starting a task
from any status (not just 'todo'). Use with caution as this is an administrative override.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark task start T-E04-F02-001                   Start task by full key
  shark task start T-E04-F02-001-user-auth         Start task by slugged key`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskStart,
}

// taskCompleteCmd marks a task as complete
var taskCompleteCmd = &cobra.Command{
	Use:   "complete <task-key>",
	Short: "Mark task as complete",
	Long: `Mark a task as ready_for_review and update timestamps.

Use --force to bypass status transition validation. This allows marking a task complete
from any status (not just 'in_progress'). Use with caution as this is an administrative override.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark task complete T-E04-F02-001                Complete task by full key
  shark task complete T-E04-F02-001-user-auth      Complete task by slugged key`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskComplete,
}

// taskApproveCmd approves a task for completion
var taskApproveCmd = &cobra.Command{
	Use:   "approve <task-key>",
	Short: "Approve task for completion",
	Long: `Approve a task that is ready for review and mark it as completed.

Use --force to bypass status transition validation. This allows approving a task
from any status (not just 'ready_for_review'). Use with caution as this is an administrative override.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskApprove,
}

// taskBlockCmd blocks a task
var taskBlockCmd = &cobra.Command{
	Use:   "block <task-key>",
	Short: "Block a task",
	Long: `Mark a task as blocked with a required reason.

Use --force to bypass status transition validation. This allows blocking a task
from any status (not just 'todo' or 'in_progress'). Use with caution as this is an administrative override.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskBlock,
}

// taskUnblockCmd unblocks a task
var taskUnblockCmd = &cobra.Command{
	Use:   "unblock <task-key>",
	Short: "Unblock a task",
	Long: `Unblock a task and return it to todo status.

Use --force to bypass status transition validation. This allows unblocking a task
from any status (not just 'blocked'). Use with caution as this is an administrative override.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUnblock,
}

// taskReopenCmd reopens a task
var taskReopenCmd = &cobra.Command{
	Use:   "reopen <task-key>",
	Short: "Reopen a task for rework",
	Long: `Reopen a task from ready_for_review status back to in_progress for additional work.

Use --force to bypass status transition validation. This allows reopening a task
from any status (not just 'ready_for_review'). Use with caution as this is an administrative override.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskReopen,
}

// taskNextCmd finds the next available task
var taskNextCmd = &cobra.Command{
	Use:   "next",
	Short: "Get next available task",
	Long: `Find the next available task based on dependencies, priority, and agent type.

Examples:
  shark task next                     Get next task
  shark task next --agent=frontend    Get next frontend task`,
	RunE: runTaskNext,
}

// taskDeleteCmd deletes a task
var taskDeleteCmd = &cobra.Command{
	Use:   "delete <task-key>",
	Short: "Delete a task",
	Long: `Delete a task from the database (and its history via CASCADE).

WARNING: This action cannot be undone. Task history will also be deleted.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark task delete T-E04-F01-001                  Delete task by full key
  shark task delete T-E04-F01-001-user-auth        Delete task by slugged key`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDelete,
}

// taskUpdateCmd updates a task's properties
var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-key>",
	Short: "Update a task",
	Long: `Update a task's properties such as title, description, priority, agent, or dependencies.

For backward status transitions (e.g., moving from review back to development), you must provide
a --reason flag to explain why the task is being sent back, unless --force is used.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark task update T-E04-F01-001 --title "New Title"
  shark task update T-E04-F01-001-user-auth --description "New description"
  shark task update T-E04-F01-001 --priority 1
  shark task update T-E04-F01-001 --agent backend
  shark task update T-E04-F01-001 --filename "docs/tasks/custom.md"
  shark task update T-E04-F01-001 --depends-on "T-E04-F01-002,T-E04-F01-003"
  shark task update T-E04-F01-001 --status in_development --reason "Missing error handling"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUpdate,
}

// taskSetStatusCmd sets arbitrary task status
var taskSetStatusCmd = &cobra.Command{
	Use:   "set-status <task-key> <status>",
	Short: "Set task to specific status",
	Long: `Set a task to an arbitrary status with workflow validation.

This is the generic status transition command that validates against the configured
workflow. It can transition to any valid status according to the workflow rules.

Use --force to bypass workflow validation. This allows transitioning to any status
regardless of workflow rules. Use with caution as this is an administrative override.

Examples:
  shark task set-status T-E04-F01-001 in_progress
  shark task set-status T-E04-F01-001 ready_for_review --notes "Completed implementation"
  shark task set-status T-E04-F01-001 blocked --notes "Waiting for API" --force`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskSetStatus,
}

// enrichTasksWithOrchestratorActions enriches tasks with orchestrator action metadata
// This function attempts to retrieve and populate orchestrator actions for each task
// based on the task's status. If action retrieval fails for any task, it logs a warning
// and continues processing other tasks (graceful degradation).
//
// Note: This function will be fully implemented when T-E07-F21-006 adds the GetOrchestratorAction
// method to the TaskRepository. Until then, it serves as a placeholder that can be extended.
func enrichTasksWithOrchestratorActions(ctx context.Context, repo *repository.TaskRepository, tasks []*models.Task) {
	if len(tasks) == 0 {
		return
	}

	// TODO: When T-E07-F21-006 is implemented, uncomment the following code:
	// Process each task - errors are logged but don't fail the operation
	// for _, task := range tasks {
	//     action, err := repo.GetOrchestratorAction(ctx, task, string(task.Status))
	//     if err != nil {
	//         cli.Warning(fmt.Sprintf("Failed to get orchestrator action for task %s: %v", task.Key, err))
	//         continue
	//     }
	//     task.OrchestratorAction = action
	// }
}

// runTaskList executes the task list command
func runTaskList(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse positional arguments first
	positionalEpic, positionalFeature, err := ParseTaskListArgs(args)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	repo := repository.NewTaskRepository(repoDb)

	// Get filter flags
	statusStr, _ := cmd.Flags().GetString("status")
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	agentStr, _ := cmd.Flags().GetString("agent")
	priorityMin, _ := cmd.Flags().GetInt("priority-min")
	priorityMax, _ := cmd.Flags().GetInt("priority-max")
	blocked, _ := cmd.Flags().GetBool("blocked")
	withActions, _ := cmd.Flags().GetBool("with-actions")
	hasRejections, _ := cmd.Flags().GetBool("has-rejections")

	// Positional arguments take priority over flags
	if positionalEpic != nil {
		epicKey = *positionalEpic
	}
	if positionalFeature != nil {
		featureKey = *positionalFeature
	}

	// If we have both epic and a feature suffix (F##), construct the full key
	// This applies to both flag-based and positional argument syntax
	if epicKey != "" && featureKey != "" && IsFeatureKeySuffix(featureKey) {
		featureKey = epicKey + "-" + featureKey
	}

	// Build filters
	var status *models.TaskStatus
	var agentType *models.AgentType
	var maxPriority *int

	// Parse status filter (can be multiple, comma-separated)
	var tasks []*models.Task
	if statusStr != "" {
		// For simplicity, handle single status first
		// TODO: Support multiple statuses in a future enhancement
		s := models.TaskStatus(statusStr)
		status = &s
	}

	// Parse agent type filter
	if agentStr != "" {
		a := models.AgentType(agentStr)
		agentType = &a
	}

	// Handle priority filter
	if priorityMax > 0 {
		maxPriority = &priorityMax
	} else if priorityMin > 0 {
		// If only min is specified, max = 10 (highest priority number)
		max := 10
		maxPriority = &max
	}

	// Query tasks based on filters
	if epicKey != "" || status != nil || agentType != nil || maxPriority != nil {
		var epicKeyPtr *string
		if epicKey != "" {
			epicKeyPtr = &epicKey
		}
		tasks, err = repo.FilterCombined(ctx, status, epicKeyPtr, agentType, maxPriority)
	} else {
		tasks, err = repo.List(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter by feature if requested
	if featureKey != "" {
		filteredTasks := []*models.Task{}
		// Get feature ID from the feature key
		featureRepo := repository.NewFeatureRepository(repoDb)
		feature, err := featureRepo.GetByKey(ctx, featureKey)
		if err != nil {
			return fmt.Errorf("failed to find feature: %w", err)
		}
		if feature != nil {
			for _, task := range tasks {
				if task.FeatureID == feature.ID {
					filteredTasks = append(filteredTasks, task)
				}
			}
			tasks = filteredTasks
		} else {
			// Feature doesn't exist - return empty list
			tasks = []*models.Task{}
		}
	}

	// Filter by blocked status if requested
	if blocked {
		filteredTasks := []*models.Task{}
		for _, task := range tasks {
			if task.Status == models.TaskStatusBlocked {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}

	// Filter by rejections if requested
	if hasRejections {
		filteredTasks := []*models.Task{}
		for _, task := range tasks {
			if task.RejectionCount > 0 {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}

	// Filter out completed tasks by default (unless --show-all or explicit status filter)
	showAll, _ := cmd.Flags().GetBool("show-all")
	tasks = filterTasksByCompletedStatus(tasks, showAll, statusStr)

	// Enrich tasks with orchestrator actions if requested
	if withActions && len(tasks) > 0 {
		enrichTasksWithOrchestratorActions(ctx, repo, tasks)
	}

	// Output results
	// TODO: Support multiple output formats (markdown, yaml, csv)
	// See docs/future-enhancements/output-formats.md for implementation plan
	// Future: Replace --json flag with --format=json|table|markdown|yaml|csv
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(tasks)
	}

	// Human-readable table output
	if len(tasks) == 0 {
		cli.Info("No tasks found")
		return nil
	}

	headers := []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"}
	rows := [][]string{}
	for _, task := range tasks {
		agentTypeStr := "-"
		if task.AgentType != nil {
			agentTypeStr = string(*task.AgentType)
		}

		// Truncate title if too long
		title := task.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		// Format execution_order (show "-" if NULL)
		execOrder := "-"
		if task.ExecutionOrder != nil {
			execOrder = fmt.Sprintf("%d", *task.ExecutionOrder)
		}

		// Add rejection indicator to key if task has rejections
		keyDisplay := task.Key
		if task.RejectionCount > 0 {
			keyDisplay = task.Key + " " + formatRejectionIndicator(task.RejectionCount)
		}

		rows = append(rows, []string{
			keyDisplay,
			title,
			string(task.Status),
			fmt.Sprintf("%d", task.Priority),
			agentTypeStr,
			execOrder,
		})
	}

	cli.OutputTable(headers, rows)

	// Show action summaries if --with-actions flag is set
	// Note: This will display orchestrator actions once T-E07-F21-006 adds OrchestratorAction field to Task model
	// TODO: When T-E07-F21-006 is implemented, uncomment this code to display action summaries:
	_ = withActions // Suppress unused variable warning
	/*
		if withActions {
			// hasActions := false
			// for _, task := range tasks {
			//     if task.OrchestratorAction != nil {
			//         hasActions = true
			//         break
			//     }
			// }
			//
			// if hasActions {
			//     cli.Info("") // Blank line
			//     cli.Info("Orchestrator Actions:")
			//     for _, task := range tasks {
			//         if task.OrchestratorAction != nil {
			//             action := task.OrchestratorAction
			//             agentInfo := ""
			//             if action.AgentType != "" {
			//                 agentInfo = fmt.Sprintf(" → %s", action.AgentType)
			//             }
			//             skillsInfo := ""
			//             if len(action.Skills) > 0 {
			//                 skillsInfo = fmt.Sprintf(" (%s)", strings.Join(action.Skills, ", "))
			//             }
			//             cli.Info(fmt.Sprintf("  %s: %s%s%s", task.Key, action.Action, agentInfo, skillsInfo))
			//         }
			//     }
			// }
		}
	*/

	return nil
}

// runTaskGet executes the task get command
func runTaskGet(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repositories
	// Note: repoDb initialized via cli.GetDB()
	taskRepo := repository.NewTaskRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)
	documentRepo := repository.NewDocumentRepository(repoDb)
	relationshipRepo := repository.NewTaskRelationshipRepository(repoDb)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get project root for path resolution
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}

	// Resolve task path using PathResolver
	var resolvedPath string
	if projectRoot != "" {
		pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
		absPath, err := pathResolver.ResolveTaskPath(ctx, task.Key)
		if err == nil {
			resolvedPath = getRelativePathTask(absPath, projectRoot)
		} else if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to resolve task path: %v\n", err)
		}
	}

	// Check dependencies and their status
	dependencyStatus := map[string]string{}
	if task.DependsOn != nil && *task.DependsOn != "" {
		var deps []string
		if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err == nil {
			for _, depKey := range deps {
				depTask, err := taskRepo.GetByKey(ctx, depKey)
				if err == nil {
					dependencyStatus[depKey] = string(depTask.Status)
				} else {
					dependencyStatus[depKey] = "not found"
				}
			}
		}
	}

	// Extract directory path and filename
	var dirPath, filename string
	if resolvedPath != "" {
		dirPath = filepath.Dir(resolvedPath) + "/"
		filename = filepath.Base(resolvedPath)
	}

	// Get related documents
	relatedDocs, err := documentRepo.ListForTask(ctx, task.ID)
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to fetch related documents: %v\n", err)
	}
	if relatedDocs == nil {
		relatedDocs = []*models.Document{}
	}

	// Get blocking relationships
	// Blocked-by: incoming "blocks" relationships (tasks that block this task)
	blockedByRels, err := relationshipRepo.GetIncoming(ctx, task.ID, []string{"blocks"})
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to fetch blocked-by relationships: %v\n", err)
	}
	blockedByKeys := []string{}
	for _, rel := range blockedByRels {
		blocker, err := taskRepo.GetByID(ctx, rel.FromTaskID)
		if err == nil {
			blockedByKeys = append(blockedByKeys, blocker.Key)
		}
	}

	// Blocks: outgoing "blocks" relationships (tasks this task blocks)
	blocksRels, err := relationshipRepo.GetOutgoing(ctx, task.ID, []string{"blocks"})
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to fetch blocks relationships: %v\n", err)
	}
	blocksKeys := []string{}
	for _, rel := range blocksRels {
		blocked, err := taskRepo.GetByID(ctx, rel.ToTaskID)
		if err == nil {
			blocksKeys = append(blocksKeys, blocked.Key)
		}
	}

	// Get rejection history
	noteRepo := repository.NewTaskNoteRepository(repoDb)
	rejectionHistory, err := noteRepo.GetRejectionHistory(ctx, task.ID)
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to fetch rejection history: %v\n", err)
	}
	if rejectionHistory == nil {
		rejectionHistory = make([]*repository.RejectionHistoryEntry, 0)
	}

	// Output results
	if cli.GlobalConfig.JSON {
		// Create enhanced output with dependency status, related docs, and blocking relationships
		output := map[string]interface{}{
			"task":              task,
			"path":              dirPath,
			"filename":          filename,
			"dependency_status": dependencyStatus,
			"related_documents": relatedDocs,
			"blocked_by":        blockedByKeys,
			"blocks":            blocksKeys,
			"rejection_history": rejectionHistory,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("Task: %s\n", task.Key)
	fmt.Printf("Title: %s\n", task.Title)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Priority: %d\n", task.Priority)

	if task.ExecutionOrder != nil {
		fmt.Printf("Order: %d\n", *task.ExecutionOrder)
	}

	if dirPath != "" {
		fmt.Printf("Path: %s\n", dirPath)
	}

	if filename != "" {
		fmt.Printf("Filename: %s\n", filename)
	}

	if task.Description != nil {
		fmt.Printf("Description: %s\n", *task.Description)
	}

	if task.AgentType != nil {
		fmt.Printf("Agent Type: %s\n", *task.AgentType)
	}

	if task.AssignedAgent != nil {
		fmt.Printf("Assigned Agent: %s\n", *task.AssignedAgent)
	}

	if task.BlockedReason != nil {
		fmt.Printf("Blocked Reason: %s\n", *task.BlockedReason)
	}

	// Display timestamps
	fmt.Printf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	if task.StartedAt.Valid {
		fmt.Printf("Started: %s\n", task.StartedAt.Time.Format("2006-01-02 15:04:05"))
	}
	if task.CompletedAt.Valid {
		fmt.Printf("Completed: %s\n", task.CompletedAt.Time.Format("2006-01-02 15:04:05"))
	}
	if task.BlockedAt.Valid {
		fmt.Printf("Blocked: %s\n", task.BlockedAt.Time.Format("2006-01-02 15:04:05"))
	}

	// Display dependencies
	if len(dependencyStatus) > 0 {
		fmt.Println("\nDependencies:")
		for depKey, status := range dependencyStatus {
			fmt.Printf("  - %s: %s\n", depKey, status)
		}
	}

	// Display blocking relationships
	if len(blockedByKeys) > 0 {
		fmt.Println("\nBlocked By:")
		for _, key := range blockedByKeys {
			fmt.Printf("  - %s\n", key)
		}
	}

	if len(blocksKeys) > 0 {
		fmt.Println("\nBlocks:")
		for _, key := range blocksKeys {
			fmt.Printf("  - %s\n", key)
		}
	}

	// Display related documents
	if len(relatedDocs) > 0 {
		fmt.Println("\nRelated Documents:")
		for _, doc := range relatedDocs {
			fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
		}
	}

	// Display completion metadata if flag is set
	completionDetails, _ := cmd.Flags().GetBool("completion-details")
	if completionDetails {
		// Get completion metadata
		metadata, err := taskRepo.GetCompletionMetadata(ctx, taskKey)
		if err != nil {
			fmt.Println("\nCompletion Metadata: Not available")
		} else {
			fmt.Println("\nCompletion Metadata:")

			if metadata.CompletedBy != nil && *metadata.CompletedBy != "" {
				fmt.Printf("  Completed By: %s\n", *metadata.CompletedBy)
			}

			if metadata.CompletedAt != nil {
				fmt.Printf("  Completed At: %s\n", metadata.CompletedAt.Format("2006-01-02 15:04:05"))
			}

			if metadata.VerificationStatus != "" {
				fmt.Printf("  Verification: %s\n", metadata.VerificationStatus)
			}

			if metadata.TestsPassed {
				fmt.Println("  Tests: Passed")
			}

			if len(metadata.FilesChanged) > 0 {
				fmt.Println("  Files Changed:")
				for _, file := range metadata.FilesChanged {
					fmt.Printf("    - %s\n", file)
				}
			}

			if metadata.TimeSpentMinutes != nil && *metadata.TimeSpentMinutes > 0 {
				fmt.Printf("  Time Spent: %d minutes\n", *metadata.TimeSpentMinutes)
			}

			if metadata.CompletionNotes != nil && *metadata.CompletionNotes != "" {
				fmt.Printf("  Notes: %s\n", *metadata.CompletionNotes)
			}
		}
	}

	// Display rejection history if present
	if len(rejectionHistory) > 0 {
		fmt.Printf("\n⚠️  REJECTION HISTORY (%d rejections)\n", len(rejectionHistory))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for _, rejection := range rejectionHistory {
			// Parse timestamp to format nicely
			// Format: [2026-01-15 14:30] Rejected by reviewer-agent-001
			fmt.Printf("\n[%s] Rejected by %s\n", rejection.Timestamp, rejection.RejectedBy)
			fmt.Printf("%s → %s\n", rejection.FromStatus, rejection.ToStatus)

			fmt.Println("\nReason:")
			fmt.Printf("%s\n", rejection.Reason)

			if rejection.ReasonDocument != nil {
				fmt.Printf("\n📄 Related Document: %s\n", *rejection.ReasonDocument)
			}

			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		}
	}

	return nil
}

// runTaskNext executes the task next command
func runTaskNext(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repositories
	dbWrapper := repoDb
	repo := repository.NewTaskRepository(dbWrapper)
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)

	// Get filter flags
	agentStr, _ := cmd.Flags().GetString("agent")
	epicKey, _ := cmd.Flags().GetString("epic")

	// Build filter for todo status
	todoStatus := models.TaskStatusTodo
	var agentType *models.AgentType
	var epicKeyPtr *string

	if agentStr != "" {
		a := models.AgentType(agentStr)
		agentType = &a
	}

	if epicKey != "" {
		epicKeyPtr = &epicKey
	}

	// Get all todo tasks matching filters
	tasks, err := repo.FilterCombined(ctx, &todoStatus, epicKeyPtr, agentType, nil)
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
	}

	// Filter out tasks with incomplete dependencies
	var availableTasks []*models.Task
	for _, task := range tasks {
		if isTaskAvailable(ctx, task, repo, relRepo) {
			availableTasks = append(availableTasks, task)
		}
	}

	if len(availableTasks) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"message": "No available tasks found"})
		}
		cli.Info("No available tasks found")
		return nil
	}

	// Select next task(s) based on execution_order and priority
	nextTasks := selectNextTasks(availableTasks)

	if len(nextTasks) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"message": "No available tasks found"})
		}
		cli.Info("No available tasks found")
		return nil
	}

	// Output result - if multiple tasks, output array; if single task, output object
	if len(nextTasks) > 1 {
		// Multiple tasks with same order (parallel work possible)
		if cli.GlobalConfig.JSON {
			taskOutputs := []map[string]interface{}{}
			for _, task := range nextTasks {
				dependencyStatus := map[string]string{}
				if task.DependsOn != nil && *task.DependsOn != "" {
					var deps []string
					if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err == nil {
						for _, depKey := range deps {
							depTask, err := repo.GetByKey(ctx, depKey)
							if err == nil {
								dependencyStatus[depKey] = string(depTask.Status)
							}
						}
					}
				}

				taskOutputs = append(taskOutputs, map[string]interface{}{
					"key":               task.Key,
					"title":             task.Title,
					"file_path":         task.FilePath,
					"dependencies":      task.DependsOn,
					"dependency_status": dependencyStatus,
					"priority":          task.Priority,
					"agent_type":        task.AgentType,
					"execution_order":   task.ExecutionOrder,
				})
			}

			output := map[string]interface{}{
				"message": "Multiple tasks available for parallel execution",
				"count":   len(nextTasks),
				"tasks":   taskOutputs,
			}
			return cli.OutputJSON(output)
		}

		// Human-readable output for multiple tasks
		fmt.Printf("Multiple tasks available for parallel execution (%d tasks with order=%v):\n\n",
			len(nextTasks), nextTasks[0].ExecutionOrder)
		for i, task := range nextTasks {
			fmt.Printf("%d. %s: %s\n", i+1, task.Key, task.Title)
			fmt.Printf("   Priority: %d\n", task.Priority)
			if task.ExecutionOrder != nil {
				fmt.Printf("   Order: %d\n", *task.ExecutionOrder)
			}
			if task.AgentType != nil {
				fmt.Printf("   Agent Type: %s\n", *task.AgentType)
			}
			if task.FilePath != nil {
				fmt.Printf("   File Path: %s\n", *task.FilePath)
			}
			fmt.Println()
		}

		return nil
	}

	// Single next task
	nextTask := nextTasks[0]

	// Output result
	if cli.GlobalConfig.JSON {
		// Include dependency status
		dependencyStatus := map[string]string{}
		if nextTask.DependsOn != nil && *nextTask.DependsOn != "" {
			var deps []string
			if err := json.Unmarshal([]byte(*nextTask.DependsOn), &deps); err == nil {
				for _, depKey := range deps {
					depTask, err := repo.GetByKey(ctx, depKey)
					if err == nil {
						dependencyStatus[depKey] = string(depTask.Status)
					}
				}
			}
		}

		output := map[string]interface{}{
			"key":               nextTask.Key,
			"title":             nextTask.Title,
			"file_path":         nextTask.FilePath,
			"dependencies":      nextTask.DependsOn,
			"dependency_status": dependencyStatus,
			"priority":          nextTask.Priority,
			"agent_type":        nextTask.AgentType,
			"execution_order":   nextTask.ExecutionOrder,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("Next Task: %s\n", nextTask.Key)
	fmt.Printf("Title: %s\n", nextTask.Title)
	fmt.Printf("Priority: %d\n", nextTask.Priority)
	if nextTask.ExecutionOrder != nil {
		fmt.Printf("Order: %d\n", *nextTask.ExecutionOrder)
	}
	if nextTask.AgentType != nil {
		fmt.Printf("Agent Type: %s\n", *nextTask.AgentType)
	}
	if nextTask.FilePath != nil {
		fmt.Printf("File Path: %s\n", *nextTask.FilePath)
	}

	return nil
}

// selectNextTasks selects the next task(s) to work on based on order and priority
// Returns all tasks with the lowest execution_order value (or highest priority if no order)
// Sorting logic:
// 1. execution_order ascending (nulls last) - tasks with order=1 before order=2
// 2. priority ascending (1 is highest priority, so 1 before 10)
// 3. created_at ascending (oldest first)
func selectNextTasks(tasks []*models.Task) []*models.Task {
	if len(tasks) == 0 {
		return []*models.Task{}
	}

	// Sort tasks by: order (nulls last), priority, created_at
	sortedTasks := make([]*models.Task, len(tasks))
	copy(sortedTasks, tasks)

	// Sort using comparison function
	for i := 0; i < len(sortedTasks)-1; i++ {
		for j := i + 1; j < len(sortedTasks); j++ {
			if compareTasksForNext(sortedTasks[j], sortedTasks[i]) {
				sortedTasks[i], sortedTasks[j] = sortedTasks[j], sortedTasks[i]
			}
		}
	}

	// Find all tasks with the same lowest execution_order
	var result []*models.Task
	firstTask := sortedTasks[0]

	for _, task := range sortedTasks {
		// Check if this task has the same order as the first task
		if bothNil(firstTask.ExecutionOrder, task.ExecutionOrder) {
			// Both have no order - only return the highest priority one
			if task.Priority == firstTask.Priority && task.CreatedAt.Equal(firstTask.CreatedAt) {
				result = append(result, task)
			} else if task.Priority == firstTask.Priority {
				// Same priority, different created_at - only first one
				if task.ID == firstTask.ID {
					result = append(result, task)
				}
			} else {
				// Different priority - only first one
				if task.ID == firstTask.ID {
					result = append(result, task)
				}
			}
		} else if !bothNil(firstTask.ExecutionOrder, task.ExecutionOrder) {
			// One has order, one doesn't - only include ones with same order value
			if firstTask.ExecutionOrder != nil && task.ExecutionOrder != nil {
				if *firstTask.ExecutionOrder == *task.ExecutionOrder {
					result = append(result, task)
				}
			}
		}
	}

	return result
}

// compareTasksForNext returns true if task a should come before task b
func compareTasksForNext(a, b *models.Task) bool {
	// 1. Compare execution_order (nulls last)
	if a.ExecutionOrder == nil && b.ExecutionOrder != nil {
		return false // b comes first (has order)
	}
	if a.ExecutionOrder != nil && b.ExecutionOrder == nil {
		return true // a comes first (has order)
	}
	if a.ExecutionOrder != nil && b.ExecutionOrder != nil {
		if *a.ExecutionOrder != *b.ExecutionOrder {
			return *a.ExecutionOrder < *b.ExecutionOrder
		}
	}

	// 2. Compare priority (1 is highest, so lower number = higher priority)
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}

	// 3. Compare created_at (older first)
	return a.CreatedAt.Before(b.CreatedAt)
}

// bothNil returns true if both pointers are nil
func bothNil(a, b *int) bool {
	return a == nil && b == nil
}

// isTaskAvailable checks if a task's dependencies are all completed or archived
func isTaskAvailable(ctx context.Context, task *models.Task, repo *repository.TaskRepository, relRepo *repository.TaskRelationshipRepository) bool {
	// First, check the old depends_on field for backward compatibility
	if task.DependsOn != nil && *task.DependsOn != "" && *task.DependsOn != "[]" {
		var deps []string
		if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err == nil {
			// Check each dependency
			for _, depKey := range deps {
				depTask, err := repo.GetByKey(ctx, depKey)
				if err != nil {
					return false // Dependency not found
				}

				// Dependency must be completed or archived
				if depTask.Status != models.TaskStatusCompleted && depTask.Status != models.TaskStatusArchived {
					return false
				}
			}
		}
	}

	// Second, check task_relationships for depends_on relationships
	// Get outgoing depends_on relationships
	rels, err := relRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		// If error getting relationships, assume available (graceful degradation)
		return true
	}

	// Check each dependency relationship
	for _, rel := range rels {
		depTask, err := repo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			return false // Dependency not found
		}

		// Dependency must be completed or archived
		if depTask.Status != models.TaskStatusCompleted && depTask.Status != models.TaskStatusArchived {
			return false // Incomplete dependency blocks this task
		}
	}

	return true
}

// filterTasksByCompletedStatus filters out completed tasks unless showAll is true
// or an explicit status filter is set
// formatRejectionIndicator formats the rejection indicator symbol and count
func formatRejectionIndicator(count int) string {
	if count == 0 {
		return ""
	}
	if count > 9 {
		return "⚠️(9+)"
	}
	return fmt.Sprintf("⚠️(%d)", count)
}

func filterTasksByCompletedStatus(tasks []*models.Task, showAll bool, statusFilter string) []*models.Task {
	// If an explicit status filter is set, don't apply default filtering
	// The status filter will be handled by the repository query
	if statusFilter != "" {
		return tasks
	}

	// If showAll is true, return all tasks
	if showAll {
		return tasks
	}

	// Default behavior: filter out completed tasks
	filtered := make([]*models.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Status != models.TaskStatusCompleted {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// runTaskCreate executes the task create command
func runTaskCreate(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse positional arguments (supports multiple syntaxes)
	var title, epicKey, featureKey string
	positionalEpic, positionalFeature, positionalTitle, err := ParseTaskCreateArgs(args)

	// If positional parsing succeeds, use those values
	if err == nil && positionalEpic != nil && positionalFeature != nil && positionalTitle != nil {
		// Positional syntax: shark task create E07 F20 "Task Title" or shark task create E07-F20 "Task Title"
		title = *positionalTitle

		// Get flag values
		epicFlag, _ := cmd.Flags().GetString("epic")
		featureFlag, _ := cmd.Flags().GetString("feature")

		// If flags are also provided, positional args take priority (with warning)
		if epicFlag == "" {
			epicKey = *positionalEpic
		} else if epicFlag != *positionalEpic {
			cli.Warning(fmt.Sprintf("Epic key mismatch: positional=%s, flag=%s. Using positional value.", *positionalEpic, epicFlag))
			epicKey = *positionalEpic
		} else {
			epicKey = *positionalEpic
		}

		if featureFlag == "" {
			featureKey = *positionalFeature
		} else if featureFlag != *positionalFeature {
			cli.Warning(fmt.Sprintf("Feature key mismatch: positional=%s, flag=%s. Using positional value.", *positionalFeature, featureFlag))
			featureKey = *positionalFeature
		} else {
			featureKey = *positionalFeature
		}
	} else if len(args) == 1 {
		// Old flag syntax: shark task create "Task Title" --epic=E07 --feature=F20
		title = args[0]
		epicKey, _ = cmd.Flags().GetString("epic")
		featureKey, _ = cmd.Flags().GetString("feature")
	} else {
		// Invalid syntax - show error
		cli.Error(fmt.Sprintf("Error: %v", err))
		fmt.Println("\nValid syntaxes:")
		fmt.Println("  shark task create E07 F20 \"Task Title\"")
		fmt.Println("  shark task create E07-F20 \"Task Title\"")
		fmt.Println("  shark task create \"Task Title\" --epic=E07 --feature=F20")
		os.Exit(1)
	}

	// Validate required fields
	if epicKey == "" || featureKey == "" || title == "" {
		cli.Error("Error: Missing required arguments. Epic, feature, and title are all required.")
		fmt.Println("\nExamples:")
		fmt.Println("  shark task create E07 F20 \"Build Login\"")
		fmt.Println("  shark task create E07-F20 \"Build Login\"")
		fmt.Println("  shark task create \"Build Login\" --epic=E01 --feature=F02")
		os.Exit(1)
	}

	// Get optional flags
	agentType, _ := cmd.Flags().GetString("agent")
	customTemplate, _ := cmd.Flags().GetString("template")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	dependsOn, _ := cmd.Flags().GetString("depends-on")
	executionOrder, _ := cmd.Flags().GetInt("execution-order")
	order, _ := cmd.Flags().GetInt("order")
	// Use --order if specified, otherwise fall back to --execution-order
	if order != 0 {
		executionOrder = order
	}
	customKey, _ := cmd.Flags().GetString("key")
	force, _ := cmd.Flags().GetBool("force")

	// Get filename flag for custom file path
	// Try all three flag aliases: --file, --filename, --path (priority: path > filename > file)
	fileFlag, _ := cmd.Flags().GetString("file")
	filenameFlag, _ := cmd.Flags().GetString("filename")
	pathFlag, _ := cmd.Flags().GetString("path")

	var filename string
	if pathFlag != "" {
		filename = pathFlag
	} else if filenameFlag != "" {
		filename = filenameFlag
	} else if fileFlag != "" {
		filename = fileFlag
	}

	create, _ := cmd.Flags().GetBool("create")

	// Validate custom key if provided
	if customKey != "" && containsSpace(customKey) {
		cli.Error("Error: Task key cannot contain spaces")
		os.Exit(1)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get project root (current working directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
		os.Exit(1)
	}

	// Create repositories
	// Note: repoDb initialized via cli.GetDB()
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)
	historyRepo := repository.NewTaskHistoryRepository(repoDb)

	// Create task creation components
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	// Pass nil for workflowService - Creator will create one automatically from projectRoot
	creator := taskcreation.NewCreator(repoDb, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot, nil)

	// Create task
	input := taskcreation.CreateTaskInput{
		EpicKey:        epicKey,
		FeatureKey:     featureKey,
		Title:          title,
		Description:    description,
		AgentType:      agentType,
		CustomTemplate: customTemplate,
		Priority:       priority,
		DependsOn:      dependsOn,
		ExecutionOrder: executionOrder,
		CustomKey:      customKey,
		Filename:       filename,
		Force:          force,
		Create:         create,
	}

	result, err := creator.CreateTask(ctx, input)
	if err != nil {
		cli.Error(fmt.Sprintf("Failed to create task: %s", err.Error()))
		os.Exit(1)
	}

	// Output result
	if cli.GlobalConfig.JSON {
		// JSON output with enhanced messaging
		requiredSections := cli.GetRequiredSectionsForEntityType("task")
		jsonOutput := cli.FormatEntityCreationJSON("task", result.Task.Key, result.Task.Title, result.FilePath, projectRoot, requiredSections)
		// Merge with task data
		jsonOutput["task"] = result.Task
		return cli.OutputJSON(jsonOutput)
	}

	// Human-readable output with improved messaging
	requiredSections := cli.GetRequiredSectionsForEntityType("task")
	message := cli.FormatEntityCreationMessage("task", result.Task.Key, result.Task.Title, result.FilePath, projectRoot, result.FileWasLinked, requiredSections)
	fmt.Print(message)

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, repoDb, result.Task.FeatureID)

	return nil
}

// runTaskStart executes the task start command
func runTaskStart(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository with workflow support
	dbWrapper := repoDb

	// Load workflow config
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// Create task repository with workflow
	var repo *repository.TaskRepository
	if workflow != nil {
		repo = repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)
	} else {
		repo = repository.NewTaskRepository(dbWrapper)
	}

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Note: Workflow validation now handled by repository layer, not here

	// Warn if task has incomplete dependencies
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)
	if !isTaskAvailable(ctx, task, repo, relRepo) {
		cli.Warning("Warning: Task has incomplete dependencies but proceeding with start.")
	}

	// Get agent identifier
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)

	// Update status and get orchestrator action for in_progress status
	updatedTask, orchestratorAction, err := repo.UpdateStatusWithAction(ctx, taskKey, string(models.TaskStatusInProgress))
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Create work session
	sessionRepo := repository.NewWorkSessionRepository(dbWrapper)
	session := &models.WorkSession{
		TaskID:    task.ID,
		AgentID:   &agent,
		StartedAt: time.Now(),
	}
	if err := sessionRepo.Create(ctx, session); err != nil {
		// Log warning but don't fail the command
		cli.Warning(fmt.Sprintf("Failed to create work session: %v", err))
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-started from %s status", taskKey, task.Status))
	}

	// Output result
	if cli.GlobalConfig.JSON {
		// Create response with orchestrator action (omitted if nil)
		response := struct {
			*models.Task
			OrchestratorAction *config.PopulatedAction `json:"orchestrator_action,omitempty"`
		}{
			Task:               updatedTask,
			OrchestratorAction: orchestratorAction,
		}
		return cli.OutputJSON(response)
	}

	// Human-readable output
	cli.Success(fmt.Sprintf("Task %s started successfully", taskKey))
	fmt.Printf("Status: %s → in_progress\n", task.Status)
	if updatedTask.StartedAt.Valid {
		fmt.Printf("Started: %s\n", updatedTask.StartedAt.Time.Format("2006-01-02 15:04:05"))
	}

	// Display orchestrator action if present
	if orchestratorAction != nil {
		fmt.Println("\nNext Action:")
		fmt.Printf("  Type: %s\n", orchestratorAction.Action)
		if orchestratorAction.AgentType != "" {
			fmt.Printf("  Agent: %s\n", orchestratorAction.AgentType)
		}
		if len(orchestratorAction.Skills) > 0 {
			fmt.Printf("  Skills: %s\n", strings.Join(orchestratorAction.Skills, ", "))
		}
		fmt.Printf("\nInstruction: %s\n", orchestratorAction.Instruction)
	} else {
		fmt.Println("\nNext Action: None configured")
	}

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, task.FeatureID)

	return nil
}

// runTaskComplete executes the task complete command
func runTaskComplete(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository with workflow support
	dbWrapper := repoDb

	// Load workflow config
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// Create task repository with workflow
	var repo *repository.TaskRepository
	if workflow != nil {
		repo = repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)
	} else {
		repo = repository.NewTaskRepository(dbWrapper)
	}

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Note: Workflow validation now handled by repository layer, not CLI

	// Get agent identifier and optional notes
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)
	notesFlag, _ := cmd.Flags().GetString("notes")
	var notes *string
	if notesFlag != "" {
		notes = &notesFlag
	}

	// Update status with orchestrator action (repository handles workflow validation)
	updatedTask, orchestratorAction, err := repo.UpdateStatusWithAction(ctx, taskKey, string(models.TaskStatusReadyForReview))
	if err != nil {
		// Display error with workflow suggestion
		cli.Error(fmt.Sprintf("Failed to update task status: %s", err.Error()))
		if !force {
			cli.Info("Use --force to bypass workflow validation")
		}
		os.Exit(3)
	}

	// End active work session
	sessionRepo := repository.NewWorkSessionRepository(dbWrapper)
	activeSession, err := sessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
	if err == nil && activeSession != nil {
		if err := sessionRepo.EndSession(ctx, activeSession.ID, models.SessionOutcomeCompleted, notes); err != nil {
			cli.Warning(fmt.Sprintf("Failed to end work session: %v", err))
		}
	}

	// Process completion metadata flags
	filesCreated, _ := cmd.Flags().GetStringSlice("files-created")
	filesModified, _ := cmd.Flags().GetStringSlice("files-modified")
	testsFlag, _ := cmd.Flags().GetString("tests")
	summaryFlag, _ := cmd.Flags().GetString("summary")
	verified, _ := cmd.Flags().GetBool("verified")
	agentIDFlag, _ := cmd.Flags().GetString("agent-id")
	timeSpent, _ := cmd.Flags().GetInt("time-spent")

	// Combine files-created and files-modified into single array
	var allFiles []string
	allFiles = append(allFiles, filesCreated...)
	allFiles = append(allFiles, filesModified...)

	// Only update completion metadata if at least one metadata flag was provided
	if len(allFiles) > 0 || testsFlag != "" || summaryFlag != "" || verified || agentIDFlag != "" || timeSpent > 0 {
		// Build completion metadata
		metadata := models.NewCompletionMetadata()
		metadata.FilesChanged = allFiles

		if testsFlag != "" {
			// Store test summary in completion notes if not already provided
			if notes == nil && testsFlag != "" {
				combinedNotes := fmt.Sprintf("Tests: %s", testsFlag)
				if summaryFlag != "" {
					combinedNotes = fmt.Sprintf("%s\n\n%s", combinedNotes, summaryFlag)
				}
				metadata.CompletionNotes = &combinedNotes
			}
		}

		// Store completed_by from agent
		metadata.CompletedBy = &agent

		// Set tests_passed if tests flag provided
		if testsFlag != "" {
			metadata.TestsPassed = true // Assume tests passed if summary provided
		}

		// Set verification status
		if verified {
			metadata.VerificationStatus = models.VerificationStatusVerified
		} else {
			metadata.VerificationStatus = models.VerificationStatusPending
		}

		// Set time spent if provided
		if timeSpent > 0 {
			metadata.TimeSpentMinutes = &timeSpent
		}

		// Update completion metadata in database
		if err := repo.UpdateCompletionMetadata(ctx, taskKey, metadata); err != nil {
			cli.Warning(fmt.Sprintf("Failed to save completion metadata: %v", err))
		}

		// Show warning if verified but no tests specified
		if verified && testsFlag == "" {
			cli.Warning("Task marked verified but no tests specified (use --tests to document test coverage)")
		}
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-completed from %s status", taskKey, task.Status))
	}

	// Return JSON output if requested
	if cli.GlobalConfig.JSON {
		// Get feature to get epic_id
		featureRepo := repository.NewFeatureRepository(dbWrapper)
		feature, err := featureRepo.GetByID(ctx, updatedTask.FeatureID)
		var epicID int64
		if err == nil && feature != nil {
			epicID = feature.EpicID
		}

		// Create response with orchestrator_action
		response := map[string]interface{}{
			"id":          updatedTask.ID,
			"key":         updatedTask.Key,
			"slug":        updatedTask.Slug,
			"feature_id":  updatedTask.FeatureID,
			"epic_id":     epicID,
			"title":       updatedTask.Title,
			"description": updatedTask.Description,
			"status":      updatedTask.Status,
			"priority":    updatedTask.Priority,
			"agent_type":  updatedTask.AgentType,
			"depends_on":  updatedTask.DependsOn,
			"file_path":   updatedTask.FilePath,
			"created_at":  updatedTask.CreatedAt,
			"updated_at":  updatedTask.UpdatedAt,
		}

		// Include orchestrator_action if it exists
		if orchestratorAction != nil {
			response["orchestrator_action"] = orchestratorAction
		}

		return cli.OutputJSON(response)
	}

	// Human-readable output
	cli.Success(fmt.Sprintf("Task %s marked ready for review. Status changed to ready_for_review.", taskKey))

	// Display orchestrator action summary
	displayOrchestratorAction(orchestratorAction)

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, task.FeatureID)

	return nil
}

// runTaskApprove executes the task approve command
func runTaskApprove(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository with workflow support
	dbWrapper := repoDb

	// Load workflow config
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// Create task repository with workflow
	var repo *repository.TaskRepository
	if workflow != nil {
		repo = repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)
	} else {
		repo = repository.NewTaskRepository(dbWrapper)
	}

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Note: Workflow validation now handled by repository layer, not CLI

	// Get agent identifier and optional notes
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)
	notesFlag, _ := cmd.Flags().GetString("notes")
	var notes *string
	if notesFlag != "" {
		notes = &notesFlag
	}

	// Update status (repository handles workflow validation)
	if err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, notes, nil, force); err != nil {
		// Display error with workflow suggestion
		cli.Error(fmt.Sprintf("Failed to update task status: %s", err.Error()))
		if !force {
			cli.Info("Use --force to bypass workflow validation")
		}
		os.Exit(3)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-approved from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s approved and completed.", taskKey))

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, task.FeatureID)

	return nil
}

// getAgentIdentifier returns the agent identifier from flag, environment variable, or default
func getAgentIdentifier(agentFlag string) string {
	if agentFlag != "" {
		return agentFlag
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "unknown"
}

// runTaskBlock executes the task block command
func runTaskBlock(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get required reason flag
	reason, _ := cmd.Flags().GetString("reason")
	if reason == "" {
		cli.Error("Error: --reason is required when blocking a task. Explain why the task cannot proceed.")
		os.Exit(1)
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
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status allows transition to blocked unless forcing
	// Use workflow config to determine valid transitions
	if !force {
		workflow := repo.GetWorkflow()
		if workflow != nil && workflow.StatusFlow != nil {
			allowedTransitions := workflow.StatusFlow[string(task.Status)]
			canBlock := false
			for _, nextStatus := range allowedTransitions {
				if nextStatus == "blocked" {
					canBlock = true
					break
				}
			}
			if !canBlock {
				cli.Error(fmt.Sprintf("Invalid state transition from %s to blocked.", task.Status))
				cli.Info(fmt.Sprintf("Workflow does not allow blocking from status '%s'", task.Status))
				cli.Info("Use --force to bypass this validation")
				os.Exit(3)
			}
		}
	}

	// Get agent identifier
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)

	// Block the task atomically
	if err := repo.BlockTaskForced(ctx, task.ID, reason, &agent, force); err != nil {
		return fmt.Errorf("failed to block task: %w", err)
	}

	// End active work session with blocked outcome
	dbWrapper := repoDb
	sessionRepo := repository.NewWorkSessionRepository(dbWrapper)
	activeSession, err := sessionRepo.GetActiveSessionByTaskID(ctx, task.ID)
	if err == nil && activeSession != nil {
		if err := sessionRepo.EndSession(ctx, activeSession.ID, models.SessionOutcomeBlocked, &reason); err != nil {
			cli.Warning(fmt.Sprintf("Failed to end work session: %v", err))
		}
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-blocked from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s blocked. Reason: %s", taskKey, reason))

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, task.FeatureID)

	return nil
}

// runTaskUnblock executes the task unblock command
func runTaskUnblock(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	dbWrapper := repoDb
	repo := repository.NewTaskRepository(dbWrapper)

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status is "blocked" unless forcing
	if !force && task.Status != models.TaskStatusBlocked {
		cli.Error(fmt.Sprintf("Invalid state transition from %s to todo. Task must be in 'blocked' status.", task.Status))
		cli.Info("Use --force to bypass this validation")
		os.Exit(3)
	}

	// Get agent identifier
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)

	// Unblock the task atomically
	if err := repo.UnblockTaskForced(ctx, task.ID, &agent, force); err != nil {
		return fmt.Errorf("failed to unblock task: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-unblocked from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s unblocked and returned to todo queue", taskKey))

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, task.FeatureID)

	return nil
}

// runTaskReopen executes the task reopen command
func runTaskReopen(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	dbWrapper := repoDb
	repo := repository.NewTaskRepository(dbWrapper)

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status allows reopening (typically means transitioning back to an earlier workflow stage)
	// Use workflow config to determine valid transitions
	if !force {
		workflow := repo.GetWorkflow()
		if workflow != nil && workflow.StatusFlow != nil {
			allowedTransitions := workflow.StatusFlow[string(task.Status)]
			canReopen := false
			// Reopen typically means going back to a development/refinement status
			reopenTargets := []string{"in_development", "in_progress", "ready_for_development", "ready_for_refinement", "in_refinement"}
			for _, nextStatus := range allowedTransitions {
				for _, target := range reopenTargets {
					if nextStatus == target {
						canReopen = true
						break
					}
				}
				if canReopen {
					break
				}
			}
			if !canReopen {
				cli.Error(fmt.Sprintf("Invalid state transition from %s.", task.Status))
				cli.Info(fmt.Sprintf("Workflow does not allow reopening from status '%s'", task.Status))
				cli.Info(fmt.Sprintf("Allowed transitions from '%s': %v", task.Status, allowedTransitions))
				cli.Info("Use --force to bypass this validation")
				os.Exit(3)
			}
		}
	}

	// Get agent identifier and optional notes
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)
	notesFlag, _ := cmd.Flags().GetString("notes")
	var notes *string
	if notesFlag != "" {
		notes = &notesFlag
	}

	// Get rejection reason for backward transitions
	rejectionReasonFlag, _ := cmd.Flags().GetString("rejection-reason")
	var rejectionReason *string
	if rejectionReasonFlag != "" {
		rejectionReason = &rejectionReasonFlag
	}

	// Reopen the task atomically
	if err := repo.ReopenTaskForced(ctx, task.ID, &agent, notes, rejectionReason, force); err != nil {
		return fmt.Errorf("failed to reopen task: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-reopened from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s reopened for rework.", taskKey))

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, task.FeatureID)

	return nil
}

// runTaskDelete executes the task delete command
func runTaskDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	dbWrapper := repoDb
	repo := repository.NewTaskRepository(dbWrapper)

	// Get task by key to verify it exists
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Capture feature ID before deletion for cascade
	featureID := task.FeatureID

	// Delete task from database (CASCADE will handle history)
	if err := repo.Delete(ctx, task.ID); err != nil {
		cli.Error(fmt.Sprintf("Failed to delete task: %v", err))
		os.Exit(1)
	}

	cli.Success(fmt.Sprintf("Task %s deleted successfully", taskKey))

	// Trigger cascading status updates for parent feature and epic
	triggerStatusCascade(ctx, dbWrapper, featureID)

	return nil
}

func init() {
	// Register task command with root
	cli.RootCmd.AddCommand(taskCmd)

	// Add subcommands to task
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskGetCmd)
	taskGetCmd.Flags().Bool("completion-details", false, "Display completion metadata details")
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskApproveCmd)
	taskCmd.AddCommand(taskBlockCmd)
	taskCmd.AddCommand(taskUnblockCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskNextCmd)
	taskCmd.AddCommand(taskNextStatusCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskSetStatusCmd)

	// Add flags for list command
	taskListCmd.Flags().StringP("status", "s", "", "Filter by status (todo, in_progress, completed, blocked)")
	taskListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	taskListCmd.Flags().StringP("feature", "f", "", "Filter by feature key")
	taskListCmd.Flags().StringP("agent", "a", "", "Filter by assigned agent")
	taskListCmd.Flags().IntP("priority-min", "", 0, "Minimum priority (1=highest priority)")
	taskListCmd.Flags().IntP("priority-max", "", 0, "Maximum priority (10=lowest priority)")
	taskListCmd.Flags().BoolP("blocked", "b", false, "Show only blocked tasks")
	taskListCmd.Flags().Bool("show-all", false, "Show all tasks including completed (by default, completed tasks are hidden)")
	taskListCmd.Flags().Bool("with-actions", false, "Include orchestrator actions with each task (for batch orchestrator polling)")
	taskListCmd.Flags().Bool("has-rejections", false, "Filter tasks that have rejections")

	// Add flags for create command
	taskCreateCmd.Flags().StringP("epic", "e", "", "Epic key (e.g., E01) - can also be specified as first positional argument")
	taskCreateCmd.Flags().StringP("feature", "f", "", "Feature key (e.g., F02 or E01-F02) - can also be specified as second positional argument")
	taskCreateCmd.Flags().StringP("agent", "a", "", "Agent type (optional, accepts any string)")
	taskCreateCmd.Flags().StringP("template", "", "", "Path to custom task template (optional)")
	taskCreateCmd.Flags().StringP("description", "d", "", "Detailed description (optional)")
	taskCreateCmd.Flags().IntP("priority", "p", 5, "Priority (1=highest, 10=lowest, default 5)")
	taskCreateCmd.Flags().String("depends-on", "", "Comma-separated dependency task keys (optional)")
	taskCreateCmd.Flags().Int("execution-order", 0, "Execution order (optional, 0 = not set)")
	taskCreateCmd.Flags().Int("order", 0, "Execution order (alias for --execution-order)")
	taskCreateCmd.Flags().String("key", "", "Custom key for the task (e.g., T-E01-F01-custom). If not provided, auto-generates next sequence number")
	taskCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another task")
	taskCreateCmd.Flags().Bool("create", false, "Create file if it doesn't exist when using --file flag")

	// Note: --epic and --feature flags are no longer required since they can be specified positionally

	// File path flags: --file is primary, --filename and --path are hidden aliases
	taskCreateCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/task.md)")
	taskCreateCmd.Flags().String("filename", "", "Alias for --file")
	taskCreateCmd.Flags().String("path", "", "Alias for --file")
	_ = taskCreateCmd.Flags().MarkHidden("filename")
	_ = taskCreateCmd.Flags().MarkHidden("path")

	// Add flags for next command
	taskNextCmd.Flags().StringP("agent", "a", "", "Agent type to match")
	taskNextCmd.Flags().StringP("epic", "e", "", "Filter by epic key")

	// Add flags for state transition commands
	taskStartCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskStartCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskCompleteCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskCompleteCmd.Flags().StringP("notes", "n", "", "Completion notes")
	taskCompleteCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Completion metadata flags
	taskCompleteCmd.Flags().StringSlice("files-created", []string{}, "Files created during task (repeatable)")
	taskCompleteCmd.Flags().StringSlice("files-modified", []string{}, "Files modified during task (repeatable)")
	taskCompleteCmd.Flags().String("tests", "", "Test status summary (e.g., '16/16 passing')")
	taskCompleteCmd.Flags().String("summary", "", "Completion summary describing what was delivered")
	taskCompleteCmd.Flags().Bool("verified", false, "Mark task as verified")
	taskCompleteCmd.Flags().String("agent-id", "", "Agent execution ID for traceability")
	taskCompleteCmd.Flags().Int("time-spent", 0, "Time spent in minutes")
	taskApproveCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskApproveCmd.Flags().StringP("notes", "n", "", "Approval notes")
	taskApproveCmd.Flags().String("rejection-reason", "", "Reason for rejection or feedback on the task")
	taskApproveCmd.Flags().String("reason-doc", "", "Path to document containing rejection reason (relative to project root)")
	taskApproveCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Add flags for exception handling commands
	taskBlockCmd.Flags().StringP("reason", "r", "", "Reason for blocking (required)")
	taskBlockCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskBlockCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskUnblockCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskUnblockCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskReopenCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskReopenCmd.Flags().StringP("notes", "n", "", "Rework notes")
	taskReopenCmd.Flags().String("rejection-reason", "", "Reason for rejection or sending task back")
	taskReopenCmd.Flags().String("reason-doc", "", "Path to document containing rejection reason (relative to project root)")
	taskReopenCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Add flags for update command
	taskUpdateCmd.Flags().String("title", "", "New title for the task")
	taskUpdateCmd.Flags().StringP("description", "d", "", "New description for the task")
	taskUpdateCmd.Flags().IntP("priority", "p", -1, "New priority (1=highest, 10=lowest, -1=no change)")
	taskUpdateCmd.Flags().StringP("agent", "a", "", "New agent type")
	taskUpdateCmd.Flags().String("key", "", "New key for the task (must be unique, cannot contain spaces)")
	taskUpdateCmd.Flags().String("filename", "", "New file path (relative to project root, must end in .md)")
	taskUpdateCmd.Flags().String("depends-on", "", "New comma-separated dependency task keys")
	taskUpdateCmd.Flags().Int("order", -1, "New execution order (-1 = no change)")
	taskUpdateCmd.Flags().String("status", "", "New status for the task (uses workflow validation)")
	taskUpdateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed or bypass workflow validation for status changes")
	taskUpdateCmd.Flags().String("reason", "", "Reason for backward status transitions (required unless --force is used)")

	// Add flags for set-status command
	taskSetStatusCmd.Flags().Bool("force", false, "Force status change bypassing workflow validation (use with caution)")
	taskSetStatusCmd.Flags().String("notes", "", "Notes to record with this status transition")
}

// runTaskUpdate executes the task update command
func runTaskUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository
	repo := repository.NewTaskRepository(repoDb)

	// Get task by key to verify it exists
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Track if any changes were made
	changed := false

	// Update title if provided
	title, _ := cmd.Flags().GetString("title")
	if title != "" {
		task.Title = title
		changed = true
	}

	// Update description if provided
	description, _ := cmd.Flags().GetString("description")
	if description != "" {
		task.Description = &description
		changed = true
	}

	// Update priority if provided
	priority, _ := cmd.Flags().GetInt("priority")
	if priority != -1 {
		task.Priority = priority
		changed = true
	}

	// Update agent type if provided
	agent, _ := cmd.Flags().GetString("agent")
	if agent != "" {
		agentType := models.AgentType(agent)
		task.AgentType = &agentType
		changed = true
	}

	// Update dependencies if provided
	dependsOn, _ := cmd.Flags().GetString("depends-on")
	if dependsOn != "" {
		// Parse dependencies - split by comma and trim whitespace
		deps := []string{}
		for _, part := range strings.Split(dependsOn, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				deps = append(deps, trimmed)
			}
		}
		depsJSON, err := json.Marshal(deps)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to marshal dependencies: %v", err))
			os.Exit(1)
		}
		depsStr := string(depsJSON)
		task.DependsOn = &depsStr
		changed = true
	}

	// Update execution order if provided
	order, _ := cmd.Flags().GetInt("order")
	if order != -1 {
		if order == 0 {
			// 0 means clear the execution order
			task.ExecutionOrder = nil
		} else {
			task.ExecutionOrder = &order
		}
		changed = true
	}

	// Apply core field updates if any changed
	if changed {
		if err := repo.Update(ctx, task); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update task: %v", err))
			os.Exit(1)
		}
	}

	// Handle key update separately (requires unique validation)
	newKey, _ := cmd.Flags().GetString("key")
	if newKey != "" {
		// Validate new key: no spaces allowed
		if containsSpace(newKey) {
			cli.Error("Error: Task key cannot contain spaces")
			os.Exit(1)
		}

		// Check if new key already exists (and is different from current key)
		if newKey != taskKey {
			existing, err := repo.GetByKey(ctx, newKey)
			if err == nil && existing != nil {
				cli.Error(fmt.Sprintf("Error: Task with key '%s' already exists", newKey))
				os.Exit(1)
			}

			// Update the key
			if err := repo.UpdateKey(ctx, taskKey, newKey); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to update task key: %v", err))
				os.Exit(1)
			}
			changed = true
		}
	}

	// Handle filename update separately
	// Try all three flag aliases: --file, --filename, --path (last one wins)
	file, _ := cmd.Flags().GetString("file")
	filename, _ := cmd.Flags().GetString("filename")
	path, _ := cmd.Flags().GetString("path")

	// Determine which flag was provided (priority: path > filename > file)
	var customFile string
	if path != "" {
		customFile = path
	} else if filename != "" {
		customFile = filename
	} else if file != "" {
		customFile = file
	}

	if customFile != "" {
		if err := repo.UpdateFilePath(ctx, taskKey, &customFile); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update task file path: %v", err))
			os.Exit(1)
		}
		changed = true
	}

	// Handle status update separately (requires workflow validation)
	status, _ := cmd.Flags().GetString("status")
	if status != "" {
		// Load workflow config for status validation
		configPath := cli.GlobalConfig.ConfigFile
		if configPath == "" {
			configPath = ".sharkconfig.json"
		}
		workflow, err := config.LoadWorkflowConfig(configPath)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to load workflow config: %v", err))
			os.Exit(1)
		}

		// Get force flag and reason flag
		force, _ := cmd.Flags().GetBool("force")
		reason, _ := cmd.Flags().GetString("reason")

		// Validate that backward transitions have a reason (unless --force is used)
		if err := validation.ValidateReasonForStatusTransition(status, string(task.Status), reason, force, workflow); err != nil {
			cli.Error(fmt.Sprintf("Error: %s", err.Error()))
			cli.Info(fmt.Sprintf("Use --reason to provide a reason, or use --force to bypass this requirement"))
			cli.Info(fmt.Sprintf("Example: shark task update %s --status %s --reason \"Reason for transition\"", taskKey, status))
			os.Exit(3) // Exit code 3 for invalid state
		}

		// Create repository with workflow support
		dbWrapper := repoDb
		var workflowRepo *repository.TaskRepository
		if workflow != nil {
			workflowRepo = repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)
		} else {
			workflowRepo = repository.NewTaskRepository(dbWrapper)
		}

		// Convert status string to TaskStatus
		newStatus := models.TaskStatus(status)

		// Pass reason as rejectionReason parameter (not notes)
		var rejectionReasonPtr *string
		if reason != "" {
			rejectionReasonPtr = &reason
		}

		// Update status with workflow validation (unless forcing)
		err = workflowRepo.UpdateStatusForced(ctx, task.ID, newStatus, nil, nil, rejectionReasonPtr, force)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update task status: %s", err.Error()))

			// If this is a validation error, suggest using --force
			if !force && (strings.Contains(err.Error(), "invalid status transition") || strings.Contains(err.Error(), "transition")) {
				cli.Info("Use --force to bypass workflow validation")
			}

			os.Exit(3) // Exit code 3 for invalid state
		}

		// Display warning if force was used
		if force && !cli.GlobalConfig.JSON {
			cli.Warning(fmt.Sprintf("⚠️  Forced transition from %s to %s (bypassed workflow validation)", task.Status, newStatus))
		}

		changed = true
	}

	if !changed {
		cli.Warning("No changes specified. Use --help to see available flags.")
		return nil
	}

	cli.Success(fmt.Sprintf("Task %s updated successfully", taskKey))
	return nil
}

// runTaskSetStatus executes the set-status command
func runTaskSetStatus(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Normalize task key to support short format (E##-F##-###) and case insensitivity
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	newStatus := args[1]

	// Get flags
	force, _ := cmd.Flags().GetBool("force")
	notes, _ := cmd.Flags().GetString("notes")

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Create repository with workflow support
	dbWrapper := repoDb

	// Load workflow config for repository
	configPath := cli.GlobalConfig.ConfigFile
	if configPath == "" {
		configPath = ".sharkconfig.json"
	}
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// Create task repository with workflow
	var repo *repository.TaskRepository
	if workflow != nil {
		repo = repository.NewTaskRepositoryWithWorkflow(dbWrapper, workflow)
	} else {
		repo = repository.NewTaskRepository(dbWrapper)
	}

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Convert status string to TaskStatus
	taskStatus := models.TaskStatus(newStatus)

	// Prepare notes pointer
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	// Update status with workflow validation (unless forcing)
	err = repo.UpdateStatusForced(ctx, task.ID, taskStatus, nil, notesPtr, nil, force)
	if err != nil {
		// Extract validation error message if available
		cli.Error(fmt.Sprintf("Failed to update task status: %s", err.Error()))

		// If this is a validation error, suggest using --force
		if !force && (strings.Contains(err.Error(), "invalid status transition") || strings.Contains(err.Error(), "transition")) {
			cli.Info("Use --force to bypass workflow validation")
		}

		os.Exit(3) // Exit code 3 for invalid state
	}

	// Display warning if force was used
	if force && !cli.GlobalConfig.JSON {
		cli.Warning(fmt.Sprintf("⚠️  Forced transition from %s to %s (bypassed workflow validation)", task.Status, newStatus))
	}

	// Output result
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":        taskKey,
			"previous_status": task.Status,
			"new_status":      newStatus,
			"forced":          force,
		}
		if notes != "" {
			output["notes"] = notes
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	cli.Success(fmt.Sprintf("Task %s status updated: %s → %s", taskKey, task.Status, newStatus))
	if notes != "" {
		fmt.Printf("Notes: %s\n", notes)
	}

	return nil
}

// displayOrchestratorAction displays the orchestrator action summary in human-readable format
func displayOrchestratorAction(action *config.PopulatedAction) {
	if action == nil {
		fmt.Println("Next Action: None configured")
		return
	}

	fmt.Println("\nNext Action:")
	fmt.Printf("  Type: %s\n", action.Action)
	if action.AgentType != "" {
		fmt.Printf("  Agent: %s\n", action.AgentType)
	}
	if len(action.Skills) > 0 {
		fmt.Printf("  Skills: %s\n", strings.Join(action.Skills, ", "))
	}
	fmt.Printf("\nInstruction: %s\n", action.Instruction)
}

// Note: Custom string functions removed - now using standard library:
// - strings.Contains() replaces containsString()
// - strings.Split() and strings.TrimSpace() replace splitDependencies()
