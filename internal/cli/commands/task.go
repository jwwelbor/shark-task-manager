package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
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

// taskCmd represents the task command group
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
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
	Use:   "list",
	Short: "List tasks",
	Long: `List tasks with optional filtering by status, epic, feature, or agent.

Examples:
  shark task list                      List all tasks
  shark task list --status=todo        List tasks with status 'todo'
  shark task list --epic=E04           List tasks in epic E04
  shark task list --json               Output as JSON`,
	RunE: runTaskList,
}

// taskGetCmd gets a specific task
var taskGetCmd = &cobra.Command{
	Use:   "get <task-key>",
	Short: "Get task details",
	Long:  `Display detailed information about a specific task.`,
	Args:  cobra.ExactArgs(1),
	RunE: runTaskGet,
}

// taskCreateCmd creates a new task
var taskCreateCmd = &cobra.Command{
	Use:   "create <title> [flags]",
	Short: "Create a new task",
	Long: `Create a new task with automatic key generation and file creation.

The --agent flag is optional and accepts any string value. If not provided, defaults to "general".
The --template flag allows using a custom task template file.

Examples:
  shark task create "Build Login" --epic=E01 --feature=F02
  shark task create "Build Login" --epic=E01 --feature=F02 --agent=frontend
  shark task create "User Service" --epic=E01 --feature=F02 --agent=backend --priority=5
  shark task create "Database task" --epic=E01 --feature=F02 --agent=database-admin
  shark task create "Custom task" --epic=E01 --feature=F02 --template=./my-template.md`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskCreate,
}

// taskStartCmd starts a task
var taskStartCmd = &cobra.Command{
	Use:   "start <task-key>",
	Short: "Start working on a task",
	Long: `Mark a task as in_progress and update timestamps.

Use --force to bypass status transition validation. This allows starting a task
from any status (not just 'todo'). Use with caution as this is an administrative override.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskStart,
}

// taskCompleteCmd marks a task as complete
var taskCompleteCmd = &cobra.Command{
	Use:   "complete <task-key>",
	Short: "Mark task as complete",
	Long: `Mark a task as ready_for_review and update timestamps.

Use --force to bypass status transition validation. This allows marking a task complete
from any status (not just 'in_progress'). Use with caution as this is an administrative override.`,
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

Examples:
  shark task delete T-E04-F01-001`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDelete,
}

// runTaskList executes the task list command
func runTaskList(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get filter flags
	statusStr, _ := cmd.Flags().GetString("status")
	epicKey, _ := cmd.Flags().GetString("epic")
	agentStr, _ := cmd.Flags().GetString("agent")
	priorityMin, _ := cmd.Flags().GetInt("priority-min")
	priorityMax, _ := cmd.Flags().GetInt("priority-max")
	blocked, _ := cmd.Flags().GetBool("blocked")

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

		rows = append(rows, []string{
			task.Key,
			title,
			string(task.Status),
			fmt.Sprintf("%d", task.Priority),
			agentTypeStr,
			execOrder,
		})
	}

	cli.OutputTable(headers, rows)
	return nil
}

// runTaskGet executes the task get command
func runTaskGet(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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

	// Create repositories
	repoDb := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)

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

	// Resolve task path (requires feature and epic info)
	var resolvedPath string
	if projectRoot != "" {
		feature, err := featureRepo.GetByID(ctx, task.FeatureID)
		if err == nil {
			epic, err := epicRepo.GetByID(ctx, feature.EpicID)
			if err == nil {
				pathBuilder := utils.NewPathBuilder(projectRoot)
				absPath, err := pathBuilder.ResolveTaskPath(epic.Key, feature.Key, task.Key, task.FilePath, feature.CustomFolderPath, epic.CustomFolderPath)
				if err == nil {
					resolvedPath = getRelativePathTask(absPath, projectRoot)
				}
			}
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

	// Get filename from resolved path
	var filename string
	if resolvedPath != "" {
		filename = filepath.Base(resolvedPath)
	}

	// Output results
	if cli.GlobalConfig.JSON {
		// Create enhanced output with dependency status
		output := map[string]interface{}{
			"task":              task,
			"path":              resolvedPath,
			"filename":          filename,
			"dependency_status": dependencyStatus,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("Task: %s\n", task.Key)
	fmt.Printf("Title: %s\n", task.Title)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Priority: %d\n", task.Priority)

	if resolvedPath != "" {
		fmt.Printf("Path: %s\n", resolvedPath)
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

	if task.FilePath != nil {
		fmt.Printf("File Path: %s\n", *task.FilePath)
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

	return nil
}

// runTaskNext executes the task next command
func runTaskNext(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

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
		if isTaskAvailable(ctx, task, repo) {
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

	// Return highest priority task (priority 1 = highest)
	nextTask := availableTasks[0]

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
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("Next Task: %s\n", nextTask.Key)
	fmt.Printf("Title: %s\n", nextTask.Title)
	fmt.Printf("Priority: %d\n", nextTask.Priority)
	if nextTask.AgentType != nil {
		fmt.Printf("Agent Type: %s\n", *nextTask.AgentType)
	}
	if nextTask.FilePath != nil {
		fmt.Printf("File Path: %s\n", *nextTask.FilePath)
	}

	return nil
}

// isTaskAvailable checks if a task's dependencies are all completed or archived
func isTaskAvailable(ctx context.Context, task *models.Task, repo *repository.TaskRepository) bool {
	if task.DependsOn == nil || *task.DependsOn == "" || *task.DependsOn == "[]" {
		return true // No dependencies
	}

	var deps []string
	if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err != nil {
		return true // Invalid JSON, treat as no dependencies
	}

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

	return true
}

// runTaskCreate executes the task create command
func runTaskCreate(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get title from positional argument or flag
	var title string
	if len(args) > 0 {
		title = args[0]
	} else {
		title, _ = cmd.Flags().GetString("title")
	}

	// Get required flags
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")

	// Validate required flags
	if epicKey == "" || featureKey == "" || title == "" {
		cli.Error("Error: Missing required flags. --epic, --feature, and --title (or positional argument) are required.")
		fmt.Println("\nExamples:")
		fmt.Println("  shark task create \"Build Login\" --epic=E01 --feature=F02")
		fmt.Println("  shark task create \"Build Login\" --epic=E01 --feature=F02 --agent=frontend")
		fmt.Println("  shark task create \"Build Login\" --epic=E01 --feature=F02 --template=./my-template.md")
		os.Exit(1)
	}

	// Get optional flags
	agentType, _ := cmd.Flags().GetString("agent")
	customTemplate, _ := cmd.Flags().GetString("template")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	dependsOn, _ := cmd.Flags().GetString("depends-on")
	executionOrder, _ := cmd.Flags().GetInt("execution-order")
	filename, _ := cmd.Flags().GetString("filename")
	force, _ := cmd.Flags().GetBool("force")

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

	// Get project root (current working directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
		os.Exit(1)
	}

	// Create repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)
	historyRepo := repository.NewTaskHistoryRepository(repoDb)

	// Create task creation components
	keygen := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	validator := taskcreation.NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := taskcreation.NewCreator(repoDb, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, projectRoot)

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
		Filename:       filename,
		Force:          force,
	}

	result, err := creator.CreateTask(ctx, input)
	if err != nil {
		cli.Error(fmt.Sprintf("Failed to create task: %s", err.Error()))
		os.Exit(1)
	}

	// Output result
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result.Task)
	}

	// Human-readable output
	cli.Success(fmt.Sprintf("Created task %s: %s", result.Task.Key, result.Task.Title))
	fmt.Printf("File created at: %s\n", result.FilePath)
	fmt.Printf("Start work with: shark task start %s\n", result.Task.Key)

	return nil
}

// runTaskStart executes the task start command
func runTaskStart(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status is "todo" unless forcing
	if !force && task.Status != models.TaskStatusTodo {
		cli.Error(fmt.Sprintf("Invalid state transition from %s to in_progress. Task must be in 'todo' status.", task.Status))
		cli.Info("Use --force to bypass this validation")
		os.Exit(3)
	}

	// Warn if task has incomplete dependencies
	if !isTaskAvailable(ctx, task, repo) {
		cli.Warning("Warning: Task has incomplete dependencies but proceeding with start.")
	}

	// Get agent identifier
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)

	// Update status
	if err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusInProgress, &agent, nil, force); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-started from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s started. Status changed to in_progress.", taskKey))
	return nil
}

// runTaskComplete executes the task complete command
func runTaskComplete(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status is "in_progress" unless forcing
	if !force && task.Status != models.TaskStatusInProgress {
		cli.Error(fmt.Sprintf("Invalid state transition from %s to ready_for_review. Task must be in 'in_progress' status.", task.Status))
		cli.Info("Use --force to bypass this validation")
		os.Exit(3)
	}

	// Get agent identifier and optional notes
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)
	notesFlag, _ := cmd.Flags().GetString("notes")
	var notes *string
	if notesFlag != "" {
		notes = &notesFlag
	}

	// Update status
	if err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusReadyForReview, &agent, notes, force); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-completed from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s marked ready for review. Status changed to ready_for_review.", taskKey))
	return nil
}

// runTaskApprove executes the task approve command
func runTaskApprove(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status is "ready_for_review" unless forcing
	if !force && task.Status != models.TaskStatusReadyForReview {
		cli.Error(fmt.Sprintf("Invalid state transition from %s to completed. Task must be in 'ready_for_review' status.", task.Status))
		cli.Info("Use --force to bypass this validation")
		os.Exit(3)
	}

	// Get agent identifier and optional notes
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)
	notesFlag, _ := cmd.Flags().GetString("notes")
	var notes *string
	if notesFlag != "" {
		notes = &notesFlag
	}

	// Update status
	if err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, notes, force); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-approved from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s approved and completed.", taskKey))
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

	taskKey := args[0]

	// Get required reason flag
	reason, _ := cmd.Flags().GetString("reason")
	if reason == "" {
		cli.Error("Error: --reason is required when blocking a task. Explain why the task cannot proceed.")
		os.Exit(1)
	}

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status is "todo" or "in_progress" unless forcing
	if !force && task.Status != models.TaskStatusTodo && task.Status != models.TaskStatusInProgress {
		cli.Error(fmt.Sprintf("Invalid state transition from %s to blocked. Task must be in 'todo' or 'in_progress' status.", task.Status))
		cli.Info("Use --force to bypass this validation")
		os.Exit(3)
	}

	// Get agent identifier
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)

	// Block the task atomically
	if err := repo.BlockTaskForced(ctx, task.ID, reason, &agent, force); err != nil {
		return fmt.Errorf("failed to block task: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-blocked from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s blocked. Reason: %s", taskKey, reason))
	return nil
}

// runTaskUnblock executes the task unblock command
func runTaskUnblock(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

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
	return nil
}

// runTaskReopen executes the task reopen command
func runTaskReopen(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get task by key
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Get force flag
	force, _ := cmd.Flags().GetBool("force")

	// Validate current status is "ready_for_review" unless forcing
	if !force && task.Status != models.TaskStatusReadyForReview {
		cli.Error(fmt.Sprintf("Invalid state transition from %s to in_progress. Task must be in 'ready_for_review' status.", task.Status))
		cli.Info("Use --force to bypass this validation")
		os.Exit(3)
	}

	// Get agent identifier and optional notes
	agentFlag, _ := cmd.Flags().GetString("agent")
	agent := getAgentIdentifier(agentFlag)
	notesFlag, _ := cmd.Flags().GetString("notes")
	var notes *string
	if notesFlag != "" {
		notes = &notesFlag
	}

	// Reopen the task atomically
	if err := repo.ReopenTaskForced(ctx, task.ID, &agent, notes, force); err != nil {
		return fmt.Errorf("failed to reopen task: %w", err)
	}

	if force {
		cli.Warning(fmt.Sprintf("Task %s force-reopened from %s status", taskKey, task.Status))
	}

	cli.Success(fmt.Sprintf("Task %s reopened for rework.", taskKey))
	return nil
}

// runTaskDelete executes the task delete command
func runTaskDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey := args[0]

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
	repo := repository.NewTaskRepository(repository.NewDB(database))

	// Get task by key to verify it exists
	task, err := repo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task not found: %s", taskKey))
		os.Exit(1)
	}

	// Delete task from database (CASCADE will handle history)
	if err := repo.Delete(ctx, task.ID); err != nil {
		cli.Error(fmt.Sprintf("Failed to delete task: %v", err))
		os.Exit(1)
	}

	cli.Success(fmt.Sprintf("Task %s deleted successfully", taskKey))
	return nil
}

func init() {
	// Register task command with root
	cli.RootCmd.AddCommand(taskCmd)

	// Add subcommands to task
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskGetCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskApproveCmd)
	taskCmd.AddCommand(taskBlockCmd)
	taskCmd.AddCommand(taskUnblockCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskNextCmd)
	taskCmd.AddCommand(taskDeleteCmd)

	// Add flags for list command
	taskListCmd.Flags().StringP("status", "s", "", "Filter by status (todo, in_progress, completed, blocked)")
	taskListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	taskListCmd.Flags().StringP("feature", "f", "", "Filter by feature key")
	taskListCmd.Flags().StringP("agent", "a", "", "Filter by assigned agent")
	taskListCmd.Flags().IntP("priority-min", "", 0, "Minimum priority (1-10)")
	taskListCmd.Flags().IntP("priority-max", "", 0, "Maximum priority (1-10)")
	taskListCmd.Flags().BoolP("blocked", "b", false, "Show only blocked tasks")

	// Add flags for create command
	taskCreateCmd.Flags().StringP("epic", "e", "", "Epic key (e.g., E01) (required)")
	taskCreateCmd.MarkFlagRequired("epic")
	taskCreateCmd.Flags().StringP("feature", "f", "", "Feature key (e.g., F02 or E01-F02) (required)")
	taskCreateCmd.MarkFlagRequired("feature")
	taskCreateCmd.Flags().StringP("agent", "a", "", "Agent type (optional, accepts any string)")
	taskCreateCmd.Flags().StringP("template", "", "", "Path to custom task template (optional)")
	taskCreateCmd.Flags().StringP("description", "d", "", "Detailed description (optional)")
	taskCreateCmd.Flags().IntP("priority", "p", 5, "Priority (1-10, default 5)")
	taskCreateCmd.Flags().String("depends-on", "", "Comma-separated dependency task keys (optional)")
	taskCreateCmd.Flags().Int("execution-order", 0, "Execution order (optional, 0 = not set)")
	taskCreateCmd.Flags().String("filename", "", "Custom filename path (relative to project root, must include .md extension)")
	taskCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another task")

	// Add flags for next command
	taskNextCmd.Flags().StringP("agent", "a", "", "Agent type to match")
	taskNextCmd.Flags().StringP("epic", "e", "", "Filter by epic key")

	// Add flags for state transition commands
	taskStartCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskStartCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskCompleteCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskCompleteCmd.Flags().StringP("notes", "n", "", "Completion notes")
	taskCompleteCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskApproveCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskApproveCmd.Flags().StringP("notes", "n", "", "Approval notes")
	taskApproveCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Add flags for exception handling commands
	taskBlockCmd.Flags().StringP("reason", "r", "", "Reason for blocking (required)")
	taskBlockCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskBlockCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskUnblockCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskUnblockCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
	taskReopenCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	taskReopenCmd.Flags().StringP("notes", "n", "", "Rework notes")
	taskReopenCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")
}
