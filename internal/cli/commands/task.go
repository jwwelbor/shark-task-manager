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

// displayAutoUnblockedTasks shows which tasks were auto-unblocked after a status change.
func displayAutoUnblockedTasks(unblockedKeys []string) {
	if len(unblockedKeys) > 0 {
		cli.Info(fmt.Sprintf("Auto-unblocked %d dependent task(s):", len(unblockedKeys)))
		for _, key := range unblockedKeys {
			cli.Info(fmt.Sprintf("  - %s (now todo)", key))
		}
	}
}

var taskCmd = &cobra.Command{Use: "task", Short: "Manage tasks", GroupID: "entities",
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

var taskStartCmd = &cobra.Command{
	Use: "start <task-key>", Short: "Start working on a task", Args: cobra.ExactArgs(1), RunE: runTaskStart,
}

var taskCompleteCmd = &cobra.Command{
	Use: "complete <task-key>", Short: "Mark task as complete (ready for review)", Args: cobra.ExactArgs(1), RunE: runTaskComplete,
}

var taskApproveCmd = &cobra.Command{
	Use: "approve <task-key>", Short: "Approve task for completion", Args: cobra.ExactArgs(1), RunE: runTaskApprove,
}

var taskBlockCmd = &cobra.Command{
	Use: "block <task-key>", Short: "Block a task", Args: cobra.ExactArgs(1), RunE: runTaskBlock,
}

var taskUnblockCmd = &cobra.Command{
	Use: "unblock <task-key>", Short: "Unblock a task", Args: cobra.ExactArgs(1), RunE: runTaskUnblock,
}

var taskReopenCmd = &cobra.Command{
	Use: "reopen <task-key>", Short: "Reopen a task for rework", Args: cobra.ExactArgs(1), RunE: runTaskReopen,
}

var taskNextCmd = &cobra.Command{
	Use: "next", Short: "Get next available task", RunE: runTaskNext,
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

	svc := cli.GetTaskService()
	tasks, err := svc.ListTasks(cmd.Context(), services.TaskFilters{
		EpicKey: epicKey, FeatureKey: featureKey, Status: status,
		AgentType: agentType, ShowAll: showAll, Blocked: blocked,
		MinPriority: minPriority, MaxPriority: maxPriority,
	})
	if err != nil {
		return err
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

	svc := cli.GetTaskServiceWithDeps()
	task, err := svc.GetTask(ctx, taskKey)
	if err != nil {
		return err
	}

	// Gather related data (errors non-fatal — display is best-effort)
	relatedDocs, _ := svc.ListRelatedDocuments(ctx, taskKey)
	blockedBy, _ := svc.GetTaskBlockedBy(ctx, taskKey)
	blocks, _ := svc.GetTaskBlocks(ctx, taskKey)
	deps, _ := svc.ListDependencies(ctx, taskKey)

	orchestratorAction := cli.GetDisplayService().ResolveTaskAction(ctx, task)

	workflowCfg := cli.GetWorkflowService().ForLevel("task").GetWorkflow()
	validTransitions := GetValidTransitions(string(task.Status), workflowCfg)

	var notes []*models.EntityNote
	if noteSvc, err := cli.GetNoteService(ctx); err == nil && noteSvc != nil {
		notes, _ = noteSvc.ListNotes(ctx, models.EntityTypeTask, taskKey, nil)
	}

	var contextData *models.ContextData
	if ctxSvc, err := cli.GetContextService(ctx); err == nil && ctxSvc != nil {
		contextData, _ = ctxSvc.GetContext(ctx, models.EntityTypeTask, taskKey)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(buildTaskGetJSON(task, deps, blockedBy, blocks,
			relatedDocs, validTransitions, orchestratorAction, notes, contextData))
	}

	RenderEntity(EntityDisplayOptions{
		EntityType:         "task",
		Key:                task.Key,
		Status:             string(task.Status),
		BasicInfo:          buildTaskBasicInfo(task, deps, blockedBy, blocks),
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
		return err
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

// runTaskNext finds the next available task.
func runTaskNext(cmd *cobra.Command, args []string) error {
	agentType, _ := cmd.Flags().GetString("agent")
	epicKey, _ := cmd.Flags().GetString("epic")
	svc := cli.GetTaskService()
	task, err := svc.GetNextTask(cmd.Context(), services.NextTaskFilters{AgentType: agentType, EpicKey: epicKey})
	if err != nil {
		return err
	}
	if task == nil {
		cli.Info("No available tasks found")
		return nil
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Next task: %s", task.Key))
	cli.Info(fmt.Sprintf("Title: %s", task.Title))
	cli.Info(fmt.Sprintf("Status: %s", task.Status))
	if agent := derefString(task.AgentType); agent != "" {
		cli.Info(fmt.Sprintf("Agent: %s", agent))
	}
	return nil
}

// runTaskStart starts a task.
func runTaskStart(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	agentID, _ := cmd.Flags().GetString("agent")
	svc := cli.GetTaskService()
	task, err := svc.StartTask(cmd.Context(), taskKey, agentID)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Started task %s (status: %s)", task.Key, task.Status))
	return nil
}

// runTaskComplete marks a task as complete (ready for review).
func runTaskComplete(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	notes, _ := cmd.Flags().GetString("notes")
	svc := cli.GetTaskService()
	task, err := svc.CompleteTask(cmd.Context(), taskKey, notes)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Completed task %s (status: %s)", task.Key, task.Status))
	return nil
}

// runTaskApprove approves a task for completion.
func runTaskApprove(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	notes, _ := cmd.Flags().GetString("notes")
	svc := cli.GetTaskService()
	task, err := svc.ApproveTask(cmd.Context(), taskKey, notes)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Approved task %s (status: %s)", task.Key, task.Status))
	return nil
}

// runTaskBlock blocks a task with a reason.
func runTaskBlock(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	reason, _ := cmd.Flags().GetString("reason")
	svc := cli.GetTaskService()
	task, err := svc.BlockTask(cmd.Context(), taskKey, reason)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Blocked task %s", task.Key))
	return nil
}

// runTaskUnblock unblocks a task.
func runTaskUnblock(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	svc := cli.GetTaskService()
	task, unblockedKeys, err := svc.UnblockTask(cmd.Context(), taskKey)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Unblocked task %s (status: %s)", task.Key, task.Status))
	displayAutoUnblockedTasks(unblockedKeys)
	return nil
}

// runTaskReopen reopens a task for rework.
func runTaskReopen(cmd *cobra.Command, args []string) error {
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}
	notes, _ := cmd.Flags().GetString("notes")
	svc := cli.GetTaskService()
	task, err := svc.ReopenTask(cmd.Context(), taskKey, notes)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(task)
	}
	cli.Success(fmt.Sprintf("Reopened task %s (status: %s)", task.Key, task.Status))
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
		return err
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

func init() {
	cli.RootCmd.AddCommand(taskCmd)
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
	registerListFlags(taskListCmd)
	registerCreateFlags(taskCreateCmd)
	registerTransitionFlags()
	registerUpdateFlags(taskUpdateCmd)
	taskNextCmd.Flags().StringP("agent", "a", "", "Agent type to match")
	taskNextCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	defaultStatus := cli.GetWorkflowService().GetDefaultStatus()
	taskUnblockCmd.Long = fmt.Sprintf("Unblock a task and return it to %s status.", defaultStatus)
	taskSetStatusCmd.Flags().Bool("force", false, "Force status change bypassing workflow validation")
	taskSetStatusCmd.Flags().String("notes", "", "Notes to record with status transition")
}
