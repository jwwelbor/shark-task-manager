package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// filterTasksByCompletedStatus filters out completed tasks unless showAll is true
// or an explicit status filter is set (in which case, pass through as-is).
func filterTasksByCompletedStatus(tasks []*models.Task, showAll bool, statusFilter string) []*models.Task {
	// If an explicit status filter is set, pass through without additional filtering.
	if statusFilter != "" {
		return tasks
	}
	// If showAll, return everything.
	if showAll {
		return tasks
	}
	// Default: hide completed tasks.
	var filtered []*models.Task
	for _, t := range tasks {
		if t.Status != models.TaskStatus("completed") {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// formatRejectionIndicator returns an emoji-based indicator string for rejection counts.
// Returns "" if rejectionCount is 0.
func formatRejectionIndicator(rejectionCount int) string {
	if rejectionCount <= 0 {
		return ""
	}
	return fmt.Sprintf("⚠️(%d)", rejectionCount)
}

// selectNextTasks selects the next tasks to work on from a list of available tasks.
// Tasks with a numeric ExecutionOrder take precedence over unordered tasks.
// Among tasks with orders, the lowest order group is returned (parallel work).
// Among unordered tasks, the single highest-priority task is returned.
// Ties in priority are broken by CreatedAt (older first).
func selectNextTasks(tasks []*models.Task) []*models.Task {
	if len(tasks) == 0 {
		return nil
	}

	// Separate ordered vs unordered tasks.
	var ordered []*models.Task
	var unordered []*models.Task
	for _, t := range tasks {
		if t.ExecutionOrder != nil {
			ordered = append(ordered, t)
		} else {
			unordered = append(unordered, t)
		}
	}

	// If there are ordered tasks, find and return the lowest-order group.
	if len(ordered) > 0 {
		sort.Slice(ordered, func(i, j int) bool {
			return *ordered[i].ExecutionOrder < *ordered[j].ExecutionOrder
		})
		minOrder := *ordered[0].ExecutionOrder
		var group []*models.Task
		for _, t := range ordered {
			if *t.ExecutionOrder == minOrder {
				group = append(group, t)
			}
		}
		// Sort the group: lowest priority number first (1 = highest priority).
		sort.Slice(group, func(i, j int) bool {
			if group[i].Priority != group[j].Priority {
				return group[i].Priority < group[j].Priority
			}
			return group[i].CreatedAt.Before(group[j].CreatedAt)
		})
		return group
	}

	// No ordered tasks: pick the single highest-priority unordered task.
	sort.Slice(unordered, func(i, j int) bool {
		if unordered[i].Priority != unordered[j].Priority {
			return unordered[i].Priority < unordered[j].Priority
		}
		return unordered[i].CreatedAt.Before(unordered[j].CreatedAt)
	})
	return unordered[:1]
}

// normalizeFeatureKey ensures the feature key includes the epic prefix when needed.
// If featureKey already contains a "-", it is returned as-is; otherwise epicKey is prepended.
func normalizeFeatureKey(epicKey, featureKey string) string {
	if strings.Contains(featureKey, "-") {
		return featureKey
	}
	return epicKey + "-" + featureKey
}

// parseCreateTaskInput builds a CreateTaskInput from CLI flags and positional args.
func parseCreateTaskInput(cmd *cobra.Command, args []string) services.CreateTaskInput {
	epicFlag, _ := cmd.Flags().GetString("epic")
	featureFlag, _ := cmd.Flags().GetString("feature")
	var epicKey, featureKey, title string
	switch len(args) {
	case 3:
		epicKey, featureKey, title = args[0], args[1], args[2]
	case 2:
		if strings.Contains(args[0], "-") {
			parts := strings.SplitN(args[0], "-", 3)
			if len(parts) >= 2 {
				epicKey = parts[0]
				featureKey = args[0]
			}
		} else {
			epicKey = args[0]
		}
		title = args[1]
	case 1:
		title, epicKey, featureKey = args[0], epicFlag, featureFlag
	}
	if epicKey == "" {
		epicKey = epicFlag
	}
	if featureKey == "" {
		featureKey = featureFlag
	}
	agentType, _ := cmd.Flags().GetString("agent")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	order, _ := cmd.Flags().GetInt("order")
	if execOrder, _ := cmd.Flags().GetInt("execution-order"); execOrder > 0 && order == 0 {
		order = execOrder
	}
	dependsOnStr, _ := cmd.Flags().GetString("depends-on")
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		filePath, _ = cmd.Flags().GetString("filename")
	}
	if filePath == "" {
		filePath, _ = cmd.Flags().GetString("path")
	}
	createFile, _ := cmd.Flags().GetBool("create")
	force, _ := cmd.Flags().GetBool("force")
	var dependsOn []string
	for _, d := range strings.Split(dependsOnStr, ",") {
		if d = strings.TrimSpace(d); d != "" {
			dependsOn = append(dependsOn, d)
		}
	}
	return services.CreateTaskInput{
		EpicKey: epicKey, FeatureKey: featureKey, Title: title,
		AgentType: agentType, Description: description, Priority: priority,
		ExecutionOrder: order, DependsOn: dependsOn, FilePath: filePath,
		CreateFile: createFile, Force: force,
	}
}

// parseTaskUpdates builds a TaskUpdates struct from CLI flags for the update command.
func parseTaskUpdates(cmd *cobra.Command) services.TaskUpdates {
	updates := services.TaskUpdates{}
	if cmd.Flags().Changed("title") {
		v, _ := cmd.Flags().GetString("title")
		updates.Title = &v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		updates.Description = &v
	}
	if cmd.Flags().Changed("priority") {
		v, _ := cmd.Flags().GetInt("priority")
		if v >= 0 {
			updates.Priority = &v
		}
	}
	if cmd.Flags().Changed("agent") {
		v, _ := cmd.Flags().GetString("agent")
		updates.AgentType = &v
	}
	if cmd.Flags().Changed("order") {
		v, _ := cmd.Flags().GetInt("order")
		if v >= 0 {
			updates.ExecutionOrder = &v
		}
	}
	if cmd.Flags().Changed("filename") {
		v, _ := cmd.Flags().GetString("filename")
		updates.FilePath = &v
	}
	return updates
}

// registerListFlags adds flags for the task list command.
func registerListFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("status", "s", "", cli.GetWorkflowService().StatusFlagDescription())
	cmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	cmd.Flags().StringP("feature", "f", "", "Filter by feature key")
	cmd.Flags().StringP("agent", "a", "", "Filter by assigned agent")
	cmd.Flags().IntP("priority-min", "", 0, "Minimum priority (1=highest priority)")
	cmd.Flags().IntP("priority-max", "", 0, "Maximum priority (10=lowest priority)")
	cmd.Flags().BoolP("blocked", "b", false, "Show only blocked tasks")
	cmd.Flags().Bool("show-all", false, "Show all tasks including completed")
	cmd.Flags().Bool("with-actions", false, "Include orchestrator actions with each task")
	cmd.Flags().Bool("has-rejections", false, "Filter tasks that have rejections")
}

// registerCreateFlags adds flags for the task create command.
func registerCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("epic", "e", "", "Epic key")
	cmd.Flags().StringP("feature", "f", "", "Feature key")
	cmd.Flags().StringP("agent", "a", "", "Agent type")
	cmd.Flags().StringP("description", "d", "", "Detailed description")
	cmd.Flags().Int("order", 0, "Execution order (lower runs first)")
	cmd.Flags().Int("execution-order", 0, "Execution order (alias for --order)")
	cmd.Flags().IntP("priority", "p", 5, "Priority level 1-10 (default: 5)")
	cmd.Flags().String("depends-on", "", "Comma-separated dependency task keys")
	cmd.Flags().String("key", "", "Custom task key")
	cmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
	cmd.Flags().Bool("create", false, "Create file if it doesn't exist")
	cmd.Flags().String("file", "", "Full file path (e.g., docs/custom/task.md)")
	cmd.Flags().String("filename", "", "Alias for --file")
	cmd.Flags().String("path", "", "Alias for --file")
	_ = cmd.Flags().MarkHidden("filename")
	_ = cmd.Flags().MarkHidden("path")
}

// registerTransitionFlags adds flags for task lifecycle transition commands.
func registerTransitionFlags() {
	taskStartCmd.Flags().StringP("agent", "", "", "Agent identifier")
	taskStartCmd.Flags().Bool("force", false, "Force status change bypassing validation")

	taskCompleteCmd.Flags().StringP("agent", "", "", "Agent identifier")
	taskCompleteCmd.Flags().StringP("notes", "n", "", "Completion notes")
	taskCompleteCmd.Flags().Bool("force", false, "Force status change bypassing validation")
	taskCompleteCmd.Flags().StringSlice("files-created", []string{}, "Files created during task")
	taskCompleteCmd.Flags().StringSlice("files-modified", []string{}, "Files modified during task")
	taskCompleteCmd.Flags().String("tests", "", "Test status summary")
	taskCompleteCmd.Flags().String("summary", "", "Completion summary")
	taskCompleteCmd.Flags().Bool("verified", false, "Mark task as verified")
	taskCompleteCmd.Flags().String("agent-id", "", "Agent execution ID for traceability")
	taskCompleteCmd.Flags().Int("time-spent", 0, "Time spent in minutes")

	taskApproveCmd.Flags().StringP("agent", "", "", "Agent identifier")
	taskApproveCmd.Flags().StringP("notes", "n", "", "Approval notes")
	taskApproveCmd.Flags().String("rejection-reason", "", "Reason for rejection")
	taskApproveCmd.Flags().String("reason-doc", "", "Path to rejection reason document")
	taskApproveCmd.Flags().Bool("force", false, "Force status change bypassing validation")

	taskBlockCmd.Flags().StringP("reason", "r", "", "Reason for blocking (required)")
	taskBlockCmd.Flags().StringP("agent", "", "", "Agent identifier")
	taskBlockCmd.Flags().Bool("force", false, "Force status change bypassing validation")

	taskUnblockCmd.Flags().StringP("agent", "", "", "Agent identifier")
	taskUnblockCmd.Flags().Bool("force", false, "Force status change bypassing validation")

	taskReopenCmd.Flags().StringP("agent", "", "", "Agent identifier")
	taskReopenCmd.Flags().StringP("notes", "n", "", "Rework notes")
	taskReopenCmd.Flags().String("rejection-reason", "", "Reason for rejection")
	taskReopenCmd.Flags().String("reason-doc", "", "Path to rejection reason document")
	taskReopenCmd.Flags().Bool("force", false, "Force status change bypassing validation")
}

// registerUpdateFlags adds flags for the task update command.
func registerUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().StringP("description", "d", "", "New description")
	cmd.Flags().IntP("priority", "p", -1, "New priority (1-10, -1=no change)")
	cmd.Flags().StringP("agent", "a", "", "New agent type")
	cmd.Flags().String("key", "", "New key")
	cmd.Flags().String("filename", "", "New file path (relative to project root)")
	cmd.Flags().String("depends-on", "", "New comma-separated dependency task keys")
	cmd.Flags().Int("order", -1, "New execution order (-1=no change)")
	cmd.Flags().String("status", "", "New status (uses workflow validation)")
	cmd.Flags().Bool("force", false, "Force reassignment or bypass workflow validation")
	cmd.Flags().String("reason", "", "Reason for backward status transitions")
	cmd.Flags().String("reason-doc", "", "Path to rejection reason document")
}
