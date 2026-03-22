package commands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// DependencyTree represents a hierarchical tree of task dependencies
type DependencyTree struct {
	Task         *models.Task      `json:"task"`
	Dependencies []*DependencyTree `json:"dependencies,omitempty"`
	Dependents   []*DependencyTree `json:"dependents,omitempty"`
	Depth        int               `json:"depth"`
	HasCycle     bool              `json:"has_cycle,omitempty"`
}

// TaskRepositoryInterfaceWithID extends TaskRepositoryInterface with GetByID
type TaskRepositoryInterfaceWithID interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
}

// RelationshipRepositoryInterface defines the interface for relationship repository operations.
// This is now implemented by entityRelRepoAdapter which wraps EntityRelationshipService.
type RelationshipRepositoryInterface interface {
	GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
	GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
}

// entityRelRepoAdapter adapts EntityRelationshipService to RelationshipRepositoryInterface
// for use by the tree-building code, which expects TaskRelationship results.
type entityRelRepoAdapter struct {
	svc *services.EntityRelationshipService
}

func (a *entityRelRepoAdapter) GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error) {
	rels, err := a.svc.GetOutgoing(ctx, models.EntityTypeTask, taskID, toEntityRelTypes(relTypes))
	if err != nil {
		return nil, err
	}
	return entityRelsToTaskRels(rels), nil
}

func (a *entityRelRepoAdapter) GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error) {
	rels, err := a.svc.GetIncoming(ctx, models.EntityTypeTask, taskID, toEntityRelTypes(relTypes))
	if err != nil {
		return nil, err
	}
	return entityRelsToTaskRels(rels), nil
}

// toEntityRelTypes converts a string slice to EntityRelationshipType slice.
func toEntityRelTypes(relTypes []string) []models.EntityRelationshipType {
	result := make([]models.EntityRelationshipType, len(relTypes))
	for i, rt := range relTypes {
		result[i] = models.EntityRelationshipType(rt)
	}
	return result
}

// entityRelsToTaskRels converts EntityRelationship slice to TaskRelationship slice
// for task-to-task relationships only.
func entityRelsToTaskRels(rels []*models.EntityRelationship) []*models.TaskRelationship {
	result := make([]*models.TaskRelationship, 0, len(rels))
	for _, rel := range rels {
		if rel.FromEntityType == models.EntityTypeTask && rel.ToEntityType == models.EntityTypeTask {
			result = append(result, &models.TaskRelationship{
				ID:               rel.ID,
				FromTaskID:       rel.FromEntityID,
				ToTaskID:         rel.ToEntityID,
				RelationshipType: models.RelationshipType(rel.RelationshipType),
			})
		}
	}
	return result
}

// taskDepsCmd shows all relationships for a task
var taskDepsCmd = &cobra.Command{
	Use:   "deps <task-key>",
	Short: "Show all relationships for a task",
	Long: `Show all relationships for a task (incoming and outgoing).

Shows dependencies, blocks, related tasks, and other relationships.

Examples:
  shark task deps T-E10-F03-004                              Show all relationships
  shark task deps T-E10-F03-004 --tree                       Show as dependency tree
  shark task deps T-E10-F03-004 --tree --upstream            Show upstream dependencies tree
  shark task deps T-E10-F03-004 --tree --downstream          Show downstream dependents tree
  shark task deps T-E10-F03-004 --type depends_on,blocks     Filter by types
  shark task deps T-E10-F03-004 --json                       Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDeps,
}

// taskBlockedByCmd shows what blocks this task
var taskBlockedByCmd = &cobra.Command{
	Use:   "blocked-by <task-key>",
	Short: "Show what blocks this task (incoming dependencies)",
	Long: `Show all tasks that this task depends on (incoming dependencies).

Examples:
  shark task blocked-by T-E10-F03-004        Show blocking tasks
  shark task blocked-by T-E10-F03-004 --json Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskBlockedBy,
}

// taskBlocksCmd shows what this task blocks
var taskBlocksCmd = &cobra.Command{
	Use:   "blocks <task-key>",
	Short: "Show what this task blocks (outgoing blockers)",
	Long: `Show all tasks that depend on this task completing (outgoing blockers).

Examples:
  shark task blocks T-E10-F03-003          Show blocked tasks
  shark task blocks T-E10-F03-003 --json   Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskBlocks,
}

func init() {
	taskDepsCmd.Flags().String("type", "", "Filter by relationship types (comma-separated)")
	taskDepsCmd.Flags().Bool("tree", false, "Show dependency tree visualization")
	taskDepsCmd.Flags().Bool("upstream", false, "Show upstream dependencies (prerequisites)")
	taskDepsCmd.Flags().Bool("downstream", false, "Show downstream dependents (tasks waiting on this)")
	taskDepsCmd.Flags().Int("max-depth", 10, "Maximum tree depth")

	taskCmd.AddCommand(taskDepsCmd)
	taskCmd.AddCommand(taskBlockedByCmd)
	taskCmd.AddCommand(taskBlocksCmd)
}

// taskDepsOptions holds parsed flags for the task deps command.
type taskDepsOptions struct {
	typeFilter []string
	showTree   bool
	upstream   bool
	downstream bool
	maxDepth   int
}

// parseTaskDepsOptions reads all flags for the task deps command.
func parseTaskDepsOptions(cmd *cobra.Command) taskDepsOptions {
	filterTypes, _ := cmd.Flags().GetString("type")
	showTree, _ := cmd.Flags().GetBool("tree")
	upstream, _ := cmd.Flags().GetBool("upstream")
	downstream, _ := cmd.Flags().GetBool("downstream")
	maxDepth, _ := cmd.Flags().GetInt("max-depth")
	return taskDepsOptions{
		typeFilter: parseTypeFilter(filterTypes),
		showTree:   showTree,
		upstream:   upstream,
		downstream: downstream,
		maxDepth:   maxDepth,
	}
}

// runTaskDeps handles the task deps command
func runTaskDeps(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	opts := parseTaskDepsOptions(cmd)

	taskSvc := cli.GetTaskService()
	relSvc := cli.GetEntityRelationshipService()

	task, err := taskSvc.GetTask(cmd.Context(), taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found: %w", taskKey, err)
	}

	if opts.showTree {
		adapter := &entityRelRepoAdapter{svc: relSvc}
		taskRepo := taskSvc.GetTaskRepository()
		return runTaskDepsTree(cmd.Context(), task, taskRepo, adapter, opts.upstream, opts.downstream, opts.maxDepth)
	}

	relWithTasks, err := relSvc.GetTaskRelationships(cmd.Context(), task.ID, opts.typeFilter)
	if err != nil {
		cli.Error(fmt.Sprintf("Error retrieving relationships for %s", taskKey))
		return fmt.Errorf("error retrieving relationships for %s: %w", taskKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key": taskKey, "task_title": task.Title, "relationships": relWithTasks,
		})
	}

	return printTaskDeps(taskKey, task.Title, relWithTasks)
}

// parseTypeFilter splits a comma-separated type filter string into a slice.
func parseTypeFilter(filterTypes string) []string {
	if filterTypes == "" {
		return nil
	}
	parts := strings.Split(filterTypes, ",")
	for i, t := range parts {
		parts[i] = strings.TrimSpace(t)
	}
	return parts
}

// printTaskDeps prints human-readable relationship output for a task.
func printTaskDeps(taskKey, taskTitle string, relWithTasks []services.RelationshipWithTask) error {
	fmt.Printf("%s: %s\n\n", taskKey, taskTitle)

	if len(relWithTasks) == 0 {
		fmt.Println("No relationships found")
		return nil
	}

	outgoingByType := make(map[string][]services.RelationshipWithTask)
	incomingByType := make(map[string][]services.RelationshipWithTask)
	for _, rel := range relWithTasks {
		if rel.Direction == "outgoing" {
			outgoingByType[rel.RelationshipType] = append(outgoingByType[rel.RelationshipType], rel)
		} else {
			incomingByType[rel.RelationshipType] = append(incomingByType[rel.RelationshipType], rel)
		}
	}

	relationshipOrder := []string{
		models.RelDependsOn, models.RelBlocks, models.RelRelatedTo, models.RelFollows,
		models.RelSpawnedFrom, models.RelDuplicates, models.RelReferences,
	}
	printRelationshipGroup(outgoingByType, relationshipOrder, "outgoing")
	printRelationshipGroup(incomingByType, relationshipOrder, "incoming")
	fmt.Println("Legend: ✓ completed | • in_progress | ○ todo | ✗ blocked")
	return nil
}

// printRelationshipGroup prints one direction of relationships (outgoing or incoming).
func printRelationshipGroup(byType map[string][]services.RelationshipWithTask, order []string, direction string) {
	suffix := map[string]string{"outgoing": "(this task → other tasks)", "incoming": "(other tasks → this task)"}
	for _, relType := range order {
		rels, ok := byType[relType]
		if !ok || len(rels) == 0 {
			continue
		}
		fmt.Printf("%s %s:\n", getRelationshipLabel(relType, direction), suffix[direction])
		for _, rel := range rels {
			fmt.Printf("  %s %s: %s\n", getStatusIcon(rel.TaskStatus), rel.TaskKey, rel.TaskTitle)
		}
		fmt.Println()
	}
}

// runTaskBlockedBy shows incoming dependencies
func runTaskBlockedBy(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	taskSvc := cli.GetTaskService()
	relSvc := cli.GetEntityRelationshipService()

	task, err := taskSvc.GetTask(cmd.Context(), taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found: %w", taskKey, err)
	}

	blockers, err := relSvc.GetTaskBlockedBy(cmd.Context(), task.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error retrieving dependencies for %s", taskKey))
		return fmt.Errorf("error retrieving dependencies for %s: %w", taskKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key": taskKey, "task_title": task.Title, "blocked_by": blockers,
		})
	}

	return printBlockedBy(taskKey, task.Title, blockers)
}

// printBlockedBy prints the human-readable blocked-by output.
func printBlockedBy(taskKey, taskTitle string, blockers []services.RelationshipWithTask) error {
	fmt.Printf("%s: %s\n\n", taskKey, taskTitle)
	if len(blockers) == 0 {
		fmt.Println("No blocking dependencies")
		return nil
	}
	fmt.Println("Blocked by (must complete first):")
	for _, blocker := range blockers {
		fmt.Printf("  %s %s: %s\n", getStatusIcon(blocker.TaskStatus), blocker.TaskKey, blocker.TaskTitle)
	}
	fmt.Println("\nLegend: ✓ completed | • in_progress | ○ todo | ✗ blocked")
	return nil
}

// runTaskBlocks shows outgoing blocks
func runTaskBlocks(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	taskSvc := cli.GetTaskService()
	relSvc := cli.GetEntityRelationshipService()

	task, err := taskSvc.GetTask(cmd.Context(), taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found: %w", taskKey, err)
	}

	blocked, err := relSvc.GetTaskBlocks(cmd.Context(), task.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error retrieving blocks for %s", taskKey))
		return fmt.Errorf("error retrieving blocks for %s: %w", taskKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key": taskKey, "task_title": task.Title, "blocks": blocked,
		})
	}

	return printBlocks(taskKey, task.Title, string(task.Status), blocked)
}

// printBlocks prints the human-readable blocks output.
func printBlocks(taskKey, taskTitle, taskStatus string, blocked []services.RelationshipWithTask) error {
	fmt.Printf("%s: %s\n\n", taskKey, taskTitle)
	if len(blocked) == 0 {
		fmt.Println("Not blocking any tasks")
		return nil
	}
	ws := cli.GetWorkflowService()
	isTerminal := ws.IsTerminalStatus(taskStatus)
	fmt.Println("Blocks (waiting on this task):")
	for _, b := range blocked {
		suffix := ""
		if isTerminal {
			suffix = " (unblocked)"
		}
		fmt.Printf("  %s %s: %s%s\n", getStatusIcon(b.TaskStatus), b.TaskKey, b.TaskTitle, suffix)
	}
	if isTerminal {
		fmt.Println("\nThis task is completed - all downstream tasks are unblocked.")
	}
	fmt.Println("\nLegend: ✓ completed | • in_progress | ○ todo | ✗ blocked")
	return nil
}

// getStatusIcon returns a unicode icon for task status
func getStatusIcon(status string) string {
	switch status {
	case "completed":
		return "✓"
	case "in_progress":
		return "•"
	case "blocked":
		return "✗"
	case "ready_for_review":
		return "⊙"
	default:
		return "○"
	}
}

// getRelationshipLabel returns a human-readable label for relationship type
func getRelationshipLabel(relType, direction string) string {
	labels := map[string]string{
		"depends_on":   "Dependencies",
		"blocks":       "Blocks",
		"related_to":   "Related Tasks",
		"follows":      "Follows",
		"spawned_from": "Spawned From",
		"duplicates":   "Duplicates",
		"references":   "References",
	}

	label, ok := labels[relType]
	if !ok {
		return relType
	}

	return label
}

// buildDependencyTree recursively builds a dependency tree for a task
// visited map prevents infinite loops in case of circular dependencies
// maxDepth limits recursion depth (default: 10)
func buildDependencyTree(
	ctx context.Context,
	taskRepo TaskRepositoryInterfaceWithID,
	relRepo RelationshipRepositoryInterface,
	task *models.Task,
	visited map[int64]bool,
	depth int,
	maxDepth int,
) (*DependencyTree, error) {
	// Prevent infinite recursion
	if depth > maxDepth {
		return &DependencyTree{
			Task:     task,
			Depth:    depth,
			HasCycle: true,
		}, nil
	}

	// Check if we've already visited this task (circular dependency)
	if visited[task.ID] {
		return &DependencyTree{
			Task:     task,
			Depth:    depth,
			HasCycle: true,
		}, nil
	}

	// Mark as visited
	visited[task.ID] = true

	tree := &DependencyTree{
		Task:         task,
		Dependencies: []*DependencyTree{},
		Depth:        depth,
	}

	// Get dependencies (tasks this task depends on)
	deps, err := relRepo.GetOutgoing(ctx, task.ID, []string{models.RelDependsOn})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Build subtrees for each dependency
	for _, rel := range deps {
		depTask, err := taskRepo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			continue // Skip if task not found
		}

		subtree, err := buildDependencyTree(ctx, taskRepo, relRepo, depTask, visited, depth+1, maxDepth)
		if err != nil {
			return nil, err
		}

		tree.Dependencies = append(tree.Dependencies, subtree)
	}

	// Unmark visited for other branches
	visited[task.ID] = false

	return tree, nil
}

// buildDependentsTree recursively builds a tree of tasks that depend on this task
func buildDependentsTree(
	ctx context.Context,
	taskRepo TaskRepositoryInterfaceWithID,
	relRepo RelationshipRepositoryInterface,
	task *models.Task,
	visited map[int64]bool,
	depth int,
	maxDepth int,
) (*DependencyTree, error) {
	// Prevent infinite recursion
	if depth > maxDepth {
		return &DependencyTree{
			Task:     task,
			Depth:    depth,
			HasCycle: true,
		}, nil
	}

	// Check if we've already visited this task
	if visited[task.ID] {
		return &DependencyTree{
			Task:     task,
			Depth:    depth,
			HasCycle: true,
		}, nil
	}

	// Mark as visited
	visited[task.ID] = true

	tree := &DependencyTree{
		Task:       task,
		Dependents: []*DependencyTree{},
		Depth:      depth,
	}

	// Get dependents (tasks that depend on this task)
	dependents, err := relRepo.GetIncoming(ctx, task.ID, []string{models.RelDependsOn})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}

	// Build subtrees for each dependent
	for _, rel := range dependents {
		depTask, err := taskRepo.GetByID(ctx, rel.FromTaskID)
		if err != nil {
			continue // Skip if task not found
		}

		subtree, err := buildDependentsTree(ctx, taskRepo, relRepo, depTask, visited, depth+1, maxDepth)
		if err != nil {
			return nil, err
		}

		tree.Dependents = append(tree.Dependents, subtree)
	}

	// Unmark visited for other branches
	visited[task.ID] = false

	return tree, nil
}

// renderTree renders a dependency tree in ASCII format
func renderTree(tree *DependencyTree, prefix string, isLast bool) string {
	if tree == nil {
		return ""
	}

	var output strings.Builder

	// Draw the tree branch
	if prefix == "" {
		// Root node - no prefix
		status := getStatusIcon(string(tree.Task.Status))
		cycleMarker := ""
		if tree.HasCycle {
			cycleMarker = " [CIRCULAR]"
		}
		output.WriteString(fmt.Sprintf("%s %s: %s%s\n", status, tree.Task.Key, tree.Task.Title, cycleMarker))
	} else {
		// Child node - show branch
		if isLast {
			output.WriteString(prefix + "└── ")
		} else {
			output.WriteString(prefix + "├── ")
		}

		// Add status icon and task info
		status := getStatusIcon(string(tree.Task.Status))
		cycleMarker := ""
		if tree.HasCycle {
			cycleMarker = " [CIRCULAR]"
		}
		output.WriteString(fmt.Sprintf("%s %s: %s%s\n", status, tree.Task.Key, tree.Task.Title, cycleMarker))
	}

	// Render dependencies
	for i, dep := range tree.Dependencies {
		var newPrefix string
		if prefix == "" {
			// First level children get simple indentation
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}
		isLastDep := i == len(tree.Dependencies)-1
		// Always use a prefix for first level to show tree structure
		if prefix == "" {
			output.WriteString(renderTree(dep, " ", isLastDep))
		} else {
			output.WriteString(renderTree(dep, newPrefix, isLastDep))
		}
	}

	// Render dependents
	for i, dep := range tree.Dependents {
		var newPrefix string
		if prefix == "" {
			// First level children get simple indentation
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}
		isLastDep := i == len(tree.Dependents)-1
		// Always use a prefix for first level to show tree structure
		if prefix == "" {
			output.WriteString(renderTree(dep, " ", isLastDep))
		} else {
			output.WriteString(renderTree(dep, newPrefix, isLastDep))
		}
	}

	return output.String()
}

// runTaskDepsTree handles tree visualization mode for task deps
func runTaskDepsTree(
	ctx context.Context,
	task *models.Task,
	taskRepo TaskRepositoryInterfaceWithID,
	relRepo RelationshipRepositoryInterface,
	showUpstream bool,
	showDownstream bool,
	maxDepth int,
) error {
	if !showUpstream && !showDownstream {
		showUpstream = true
		showDownstream = true
	}

	if cli.GlobalConfig.JSON {
		return outputDepsTreeJSON(ctx, task, taskRepo, relRepo, showUpstream, showDownstream, maxDepth)
	}

	return printDepsTree(ctx, task, taskRepo, relRepo, showUpstream, showDownstream, maxDepth)
}

// outputDepsTreeJSON renders the dependency tree as JSON.
func outputDepsTreeJSON(
	ctx context.Context, task *models.Task,
	taskRepo TaskRepositoryInterfaceWithID, relRepo RelationshipRepositoryInterface,
	showUpstream, showDownstream bool, maxDepth int,
) error {
	jsonOutput := map[string]interface{}{
		"task_key": task.Key, "task_title": task.Title, "task_status": string(task.Status),
	}
	if showUpstream {
		upstreamTree, _ := buildDependencyTree(ctx, taskRepo, relRepo, task, make(map[int64]bool), 0, maxDepth)
		jsonOutput["upstream"] = upstreamTree
	}
	if showDownstream {
		downstreamTree, _ := buildDependentsTree(ctx, taskRepo, relRepo, task, make(map[int64]bool), 0, maxDepth)
		jsonOutput["downstream"] = downstreamTree
	}
	return cli.OutputJSON(jsonOutput)
}

// printDepsTree renders the dependency tree in human-readable ASCII format.
func printDepsTree(
	ctx context.Context, task *models.Task,
	taskRepo TaskRepositoryInterfaceWithID, relRepo RelationshipRepositoryInterface,
	showUpstream, showDownstream bool, maxDepth int,
) error {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n%s %s: %s\n", getStatusIcon(string(task.Status)), task.Key, task.Title))
	output.WriteString(strings.Repeat("=", 80) + "\n\n")

	if showUpstream {
		output.WriteString("Upstream Dependencies (Prerequisites):\n\n")
		upstreamTree, err := buildDependencyTree(ctx, taskRepo, relRepo, task, make(map[int64]bool), 0, maxDepth)
		if err != nil {
			return fmt.Errorf("failed to build upstream tree: %w", err)
		}
		if len(upstreamTree.Dependencies) == 0 {
			output.WriteString("  No upstream dependencies\n\n")
		} else {
			output.WriteString(renderTree(upstreamTree, "", true) + "\n")
		}
	}

	if showDownstream {
		output.WriteString("Downstream Dependents (Tasks waiting on this):\n\n")
		downstreamTree, err := buildDependentsTree(ctx, taskRepo, relRepo, task, make(map[int64]bool), 0, maxDepth)
		if err != nil {
			return fmt.Errorf("failed to build downstream tree: %w", err)
		}
		if len(downstreamTree.Dependents) == 0 {
			output.WriteString("  No downstream dependents\n\n")
		} else {
			output.WriteString(renderTree(downstreamTree, "", true) + "\n")
		}
	}

	output.WriteString("Legend: ✓ completed | ⊙ ready_for_review | • in_progress | ○ todo | ✗ blocked\n")
	fmt.Print(output.String())
	return nil
}
