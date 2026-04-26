package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// taskGetServicer is the narrow interface consumed by runTaskGet.
// It covers only the two methods that command uses, enabling test injection.
type taskGetServicer interface {
	GetTaskWithTags(ctx context.Context, key string) (*models.Task, []string, error)
	GetTaskDisplayData(ctx context.Context, task *models.Task) (*services.TaskDisplayData, error)
}

// taskGetSvcOverride is non-nil only during tests.
var taskGetSvcOverride taskGetServicer

// getTaskGetService returns the service to use for runTaskGet, preferring the
// test override so that unit tests can inject a mock without a real database.
func getTaskGetService() taskGetServicer {
	if taskGetSvcOverride != nil {
		return taskGetSvcOverride
	}
	return cli.GetTaskServiceWithDocs()
}

// displayAutoUnblockedTasks shows which tasks were auto-unblocked after a status change.
func displayAutoUnblockedTasks(unblockedKeys []string) {
	if len(unblockedKeys) > 0 {
		cli.Info(fmt.Sprintf("Auto-unblocked %d dependent task(s):", len(unblockedKeys)))
		for _, key := range unblockedKeys {
			cli.Info(fmt.Sprintf("  - %s (now todo)", key))
		}
	}
}

var taskCmd = &cobra.Command{Use: "task", Short: "Manage tasks", GroupID: "advanced",
	Long: "Task lifecycle operations: list, create, update, and manage task status."}

var taskListCmd = &cobra.Command{
	Use: "list [EPIC] [FEATURE]", Short: "List tasks",
	Long: "List tasks filtered by status, epic, feature, or agent. Completed tasks hidden by default (use --all).",
	RunE: runTaskList,
}

var taskGetCmd = &cobra.Command{
	Use: "get <task-key>", Short: "Get task details", Args: cobra.ExactArgs(1), RunE: runTaskGet,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create [EPIC] [FEATURE] <title> [flags]",
	Short: "Create a new task",
	Long:  "Create a task with auto key generation.\n\nExamples:\n  shark task create E07 F20 \"Implement auth\"\n  shark task create E07-F20 \"User Service\" --agent=backend --order=2",
	Args:  cobra.RangeArgs(1, 3),
	RunE:  runTaskCreate,
}

var taskDeleteCmd = &cobra.Command{
	Use: "delete <task-key>", Short: "Delete a task",
	Long: "Delete a task (WARNING: cannot be undone, history deleted via CASCADE).",
	Args: cobra.ExactArgs(1), RunE: runTaskDelete,
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-key>",
	Short: "Update a task's properties",
	Long:  "Update task properties (title, description, priority, agent, etc).\n\nExamples:\n  shark task update T-E04-F01-001 --title \"New Title\"\n  shark task update T-E04-F01-001 --priority 1 --agent backend",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

var taskSetStatusCmd = &cobra.Command{
	Use: "set-status <task-key> <status>", Short: "Set task to specific status",
	Long: "Set task status with workflow validation. Use --force to bypass validation.",
	Args: cobra.ExactArgs(2), RunE: runTaskSetStatus,
}

// runTaskList lists tasks with optional filters.
func runTaskList(cmd *cobra.Command, args []string) error {
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")
	if len(args) >= 1 {
		epicKey = args[0]
	}
	if len(args) >= 2 {
		featureKey = normalizeFeatureKey(epicKey, args[1])
	}
	status, _ := cmd.Flags().GetString("status")
	agentType, _ := cmd.Flags().GetString("agent")
	showAll, _ := cmd.Flags().GetBool("show-all")
	allFlag, _ := cmd.Flags().GetBool("all")
	showAll = showAll || allFlag
	blocked, _ := cmd.Flags().GetBool("blocked")
	minPriority, _ := cmd.Flags().GetInt("priority-min")
	maxPriority, _ := cmd.Flags().GetInt("priority-max")
	// E28-F05 REQ-F-010: read the repeatable --tag flag.
	// nil when no --tag flags were supplied (AC-T2).
	var tagFilter []string
	if rawTags, err := cmd.Flags().GetStringSlice("tag"); err == nil && len(rawTags) > 0 {
		tagFilter = rawTags
	}

	svc := cli.GetTaskService()
	tasks, err := svc.ListTasks(cmd.Context(), services.TaskFilters{
		EpicKey: epicKey, FeatureKey: featureKey, Status: status,
		AgentType: agentType, ShowAll: showAll, Blocked: blocked,
		MinPriority: minPriority, MaxPriority: maxPriority,
		Tags: tagFilter,
	})
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "task", "")
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(tasks)
	}
	headers := []string{"Key", "Title", "Status", "Agent", "Priority"}
	var rows [][]string
	for _, t := range tasks {
		rows = append(rows, []string{t.Key, t.Title, string(t.Status), derefString(t.AgentType), fmt.Sprintf("%d", t.Priority)})
	}
	cli.OutputTable(headers, rows)
	return nil
}

// runTaskGet displays details for a single task with full rich output.
func runTaskGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Use GetTaskWithTags for tag enrichment (REQ-F-014, REQ-F-015).
	// getTaskGetService() returns the test override when set, else the real accessor.
	svc := getTaskGetService()
	task, tags, err := svc.GetTaskWithTags(ctx, taskKey)
	if err != nil {
		return err
	}

	// Gather related data via single-query view (errors non-fatal — display is best-effort)
	displayData, _ := svc.GetTaskDisplayData(ctx, task)
	var relatedDocs []*models.Document
	var blockedBy, blocks []services.RelationshipWithTask
	var deps []*models.Task
	var notes []*models.EntityNote
	if displayData != nil {
		relatedDocs = displayData.RelatedDocs
		blockedBy = displayData.BlockedBy
		blocks = displayData.Blocks
		deps = displayData.Dependencies
		notes = displayData.Notes
	}

	orchestratorAction := cli.GetDisplayService().ResolveTaskAction(ctx, task)

	workflowCfg := cli.GetWorkflowService().ForLevel("task").GetWorkflow()
	validTransitions := GetValidTransitions(string(task.Status), workflowCfg)

	var contextData *models.ContextData
	if ctxSvc := cli.GetContextService(); ctxSvc != nil {
		contextData, _ = ctxSvc.GetContext(ctx, models.EntityTypeTask, taskKey)
	}

	if cli.GlobalConfig.JSON {
		jsonResult := buildTaskGetJSON(task, deps, blockedBy, blocks,
			relatedDocs, validTransitions, orchestratorAction, notes, contextData)
		// REQ-F-015: "tags" field always present in JSON, never null.
		if tags == nil {
			tags = []string{}
		}
		jsonResult["tags"] = tags
		return cli.OutputJSON(jsonResult)
	}

	// appendTagsToBasicInfo handles nil (graceful degradation) and empty (render "(none)").
	basicInfo := buildTaskBasicInfo(task, deps, blockedBy, blocks)
	basicInfo = appendTagsToBasicInfo(basicInfo, tags)
	RenderEntity(EntityDisplayOptions{
		EntityType:         "task",
		Key:                task.Key,
		Status:             string(task.Status),
		BasicInfo:          basicInfo,
		ValidTransitions:   validTransitions,
		OrchestratorAction: orchestratorAction,
		RelatedDocs:        relatedDocs,
		Notes:              notes,
		ContextData:        contextData,
	})
	return nil
}

// buildTaskBasicInfo assembles the key-value info table for task display.
func buildTaskBasicInfo(task *models.Task, deps []*models.Task, blockedBy, blocks []services.RelationshipWithTask) [][]string {
	var info [][]string

	info = append(info, []string{"Title", task.Title})
	info = append(info, []string{"Status", string(task.Status)})

	if agent := derefString(task.AgentType); agent != "" {
		info = append(info, []string{"Agent", agent})
	}
	if task.Priority > 0 {
		info = append(info, []string{"Priority", fmt.Sprintf("%d", task.Priority)})
	}
	if task.ExecutionOrder != nil && *task.ExecutionOrder > 0 {
		info = append(info, []string{"Execution Order", fmt.Sprintf("%d", *task.ExecutionOrder)})
	}
	if fp := derefString(task.FilePath); fp != "" {
		info = append(info, []string{"File", fp})
	}
	if desc := derefString(task.Description); desc != "" {
		info = append(info, []string{"Description", desc})
	}
	if reason := derefString(task.BlockedReason); reason != "" {
		info = append(info, []string{"Blocked Reason", reason})
	}

	info = append(info, []string{"Created", task.CreatedAt.Format(time.RFC3339)})
	if task.StartedAt.Valid {
		info = append(info, []string{"Started", task.StartedAt.Time.Format(time.RFC3339)})
	}
	if task.CompletedAt.Valid {
		info = append(info, []string{"Completed", task.CompletedAt.Time.Format(time.RFC3339)})
	}
	if task.BlockedAt.Valid {
		info = append(info, []string{"Blocked At", task.BlockedAt.Time.Format(time.RFC3339)})
	}

	for _, dep := range deps {
		info = append(info, []string{"Depends On", fmt.Sprintf("%s (%s)", dep.Key, string(dep.Status))})
	}
	for _, rel := range blockedBy {
		info = append(info, []string{"Blocked By", fmt.Sprintf("%s (%s)", rel.TaskKey, rel.TaskStatus)})
	}
	for _, rel := range blocks {
		info = append(info, []string{"Blocks", fmt.Sprintf("%s (%s)", rel.TaskKey, rel.TaskStatus)})
	}

	return info
}

// runTaskCreate creates a new task.
func runTaskCreate(cmd *cobra.Command, args []string) error {
	svc := cli.GetTaskService()
	task, err := svc.CreateTask(cmd.Context(), parseCreateTaskInput(cmd, args))
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "task", "")
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Created task %s", task.Key))
	if fp := derefString(task.FilePath); fp != "" {
		cli.Info(fmt.Sprintf("File: %s", fp))
	}
	return nil
}

// runTaskDelete deletes a task.
func runTaskDelete(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	svc := cli.GetTaskService()
	if err := svc.DeleteTask(cmd.Context(), taskKey); err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": taskKey})
	}
	cli.Success(fmt.Sprintf("Deleted task %s", taskKey))
	return nil
}

// runTaskUpdate updates a task's properties.
func runTaskUpdate(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	svc := cli.GetTaskService()
	task, err := svc.UpdateTask(cmd.Context(), taskKey, parseTaskUpdates(cmd))
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "task", taskKey)
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Updated task %s", task.Key))
	return nil
}

// runTaskSetStatus sets a task to a specific status.
func runTaskSetStatus(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	notes, _ := cmd.Flags().GetString("notes")
	force, _ := cmd.Flags().GetBool("force")
	svc := cli.GetTaskService()
	result, err := svc.TransitionStatus(cmd.Context(), taskKey, args[1], services.TransitionOptions{Force: force, Reason: notes})
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}
	if result.Transitioned {
		cli.Success(fmt.Sprintf("Status: %s -> %s", result.FromStatus, result.ToStatus))
	} else {
		cli.Info(result.Message)
	}
	return nil
}

// resolveTaskID is the EntityKeyResolver used by the `shark task tag`
// subcommand factory. It looks up a task by key through the existing
// TaskService accessor (cli.GetTaskService) and returns the numeric ID.
//
// Split out as a package-level function so the E28-F04 entity_tag_cmd.go
// factory can reference it directly.
func resolveTaskID(ctx context.Context, key string) (int64, error) {
	svc := cli.GetTaskService()
	task, err := svc.GetTask(ctx, key)
	if err != nil {
		return 0, err
	}
	return task.ID, nil
}

func init() {
	taskCmd.Hidden = true // Hidden from top-level help; accessible via 'shark task'
	cli.RootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskGetCmd)
	taskGetCmd.Flags().Bool("completion-details", false, "Display completion metadata details")
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskNextStatusCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskSetStatusCmd)
	registerListFlags(taskListCmd)
	registerCreateFlags(taskCreateCmd)
	registerUpdateFlags(taskUpdateCmd)
	taskSetStatusCmd.Flags().Bool("force", false, "Force status change bypassing workflow validation")
	taskSetStatusCmd.Flags().String("notes", "", "Notes to record with status transition")

	// E28-F04 T-006: register the shared `tag add|rm` subcommand. Svc
	// override is nil in production so it falls through to
	// cli.GetTagService() at call time.
	taskCmd.AddCommand(makeEntityTagCmd(models.EntityTypeTask, resolveTaskID, nil))
}
