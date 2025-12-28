package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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

// RelationshipRepositoryInterface defines the interface for relationship repository operations
type RelationshipRepositoryInterface interface {
	GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
	GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
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

// RelationshipWithTask combines relationship and task info for output
type RelationshipWithTask struct {
	RelationshipType string `json:"relationship_type"`
	Direction        string `json:"direction"` // "outgoing" or "incoming"
	TaskKey          string `json:"task_key"`
	TaskTitle        string `json:"task_title"`
	TaskStatus       string `json:"task_status"`
}

// runTaskDeps handles the task deps command
func runTaskDeps(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	filterTypes, _ := cmd.Flags().GetString("type")
	showTree, _ := cmd.Flags().GetBool("tree")
	upstream, _ := cmd.Flags().GetBool("upstream")
	downstream, _ := cmd.Flags().GetBool("downstream")
	maxDepth, _ := cmd.Flags().GetInt("max-depth")

	var typeFilter []string
	if filterTypes != "" {
		typeFilter = strings.Split(filterTypes, ",")
		for i, t := range typeFilter {
			typeFilter[i] = strings.TrimSpace(t)
		}
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

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found", taskKey)
	}

	// If tree mode is enabled, use tree visualization
	if showTree {
		return runTaskDepsTree(ctx, task, taskRepo, relRepo, upstream, downstream, maxDepth)
	}

	// Get all relationships
	allRels, err := relRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get relationships: %w", err)
	}

	// Organize by type and direction
	outgoingByType := make(map[string][]*models.TaskRelationship)
	incomingByType := make(map[string][]*models.TaskRelationship)

	for _, rel := range allRels {
		// Filter by type if specified
		if len(typeFilter) > 0 {
			found := false
			for _, t := range typeFilter {
				if string(rel.RelationshipType) == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if rel.FromTaskID == task.ID {
			// Outgoing relationship
			relType := string(rel.RelationshipType)
			outgoingByType[relType] = append(outgoingByType[relType], rel)
		} else {
			// Incoming relationship
			relType := string(rel.RelationshipType)
			incomingByType[relType] = append(incomingByType[relType], rel)
		}
	}

	// Fetch related task details
	relWithTasks := []RelationshipWithTask{}

	for relType, rels := range outgoingByType {
		for _, rel := range rels {
			relTask, err := taskRepo.GetByID(ctx, rel.ToTaskID)
			if err != nil {
				continue
			}
			relWithTasks = append(relWithTasks, RelationshipWithTask{
				RelationshipType: relType,
				Direction:        "outgoing",
				TaskKey:          relTask.Key,
				TaskTitle:        relTask.Title,
				TaskStatus:       string(relTask.Status),
			})
		}
	}

	for relType, rels := range incomingByType {
		for _, rel := range rels {
			relTask, err := taskRepo.GetByID(ctx, rel.FromTaskID)
			if err != nil {
				continue
			}
			relWithTasks = append(relWithTasks, RelationshipWithTask{
				RelationshipType: relType,
				Direction:        "incoming",
				TaskKey:          relTask.Key,
				TaskTitle:        relTask.Title,
				TaskStatus:       string(relTask.Status),
			})
		}
	}

	// Output results
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":      taskKey,
			"task_title":    task.Title,
			"relationships": relWithTasks,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("%s: %s\n\n", taskKey, task.Title)

	if len(relWithTasks) == 0 {
		fmt.Println("No relationships found")
		return nil
	}

	// Group by type for output
	printed := make(map[string]bool)

	// Print outgoing relationships
	relationshipOrder := []string{"depends_on", "blocks", "related_to", "follows", "spawned_from", "duplicates", "references"}

	for _, relType := range relationshipOrder {
		rels, ok := outgoingByType[relType]
		if !ok || len(rels) == 0 {
			continue
		}

		fmt.Printf("%s (this task → other tasks):\n", getRelationshipLabel(relType, "outgoing"))
		for _, rel := range rels {
			relTask, _ := taskRepo.GetByID(ctx, rel.ToTaskID)
			if relTask != nil {
				status := getStatusIcon(string(relTask.Status))
				fmt.Printf("  %s %s: %s\n", status, relTask.Key, relTask.Title)
			}
		}
		fmt.Println()
		printed[relType] = true
	}

	// Print incoming relationships
	for _, relType := range relationshipOrder {
		rels, ok := incomingByType[relType]
		if !ok || len(rels) == 0 {
			continue
		}

		fmt.Printf("%s (other tasks → this task):\n", getRelationshipLabel(relType, "incoming"))
		for _, rel := range rels {
			relTask, _ := taskRepo.GetByID(ctx, rel.FromTaskID)
			if relTask != nil {
				status := getStatusIcon(string(relTask.Status))
				fmt.Printf("  %s %s: %s\n", status, relTask.Key, relTask.Title)
			}
		}
		fmt.Println()
	}

	fmt.Println("Legend: ✓ completed | • in_progress | ○ todo | ✗ blocked")

	return nil
}

// runTaskBlockedBy shows incoming dependencies
func runTaskBlockedBy(cmd *cobra.Command, args []string) error {
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

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Get outgoing depends_on relationships (this task depends on others)
	deps, err := relRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Fetch task details
	blockers := []RelationshipWithTask{}
	for _, rel := range deps {
		depTask, err := taskRepo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			continue
		}
		blockers = append(blockers, RelationshipWithTask{
			RelationshipType: "depends_on",
			Direction:        "outgoing",
			TaskKey:          depTask.Key,
			TaskTitle:        depTask.Title,
			TaskStatus:       string(depTask.Status),
		})
	}

	// Output results
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":   taskKey,
			"task_title": task.Title,
			"blocked_by": blockers,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("%s: %s\n\n", taskKey, task.Title)

	if len(blockers) == 0 {
		fmt.Println("No blocking dependencies")
		return nil
	}

	fmt.Println("Blocked by (must complete first):")
	for _, blocker := range blockers {
		status := getStatusIcon(blocker.TaskStatus)
		fmt.Printf("  %s %s: %s\n", status, blocker.TaskKey, blocker.TaskTitle)
	}

	fmt.Println("\nLegend: ✓ completed | • in_progress | ○ todo | ✗ blocked")

	return nil
}

// runTaskBlocks shows outgoing blocks
func runTaskBlocks(cmd *cobra.Command, args []string) error {
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

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Get incoming depends_on relationships (others depend on this task)
	blockedTasks, err := relRepo.GetIncoming(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return fmt.Errorf("failed to get blocked tasks: %w", err)
	}

	// Also get outgoing blocks relationships
	explicitBlocks, err := relRepo.GetOutgoing(ctx, task.ID, []string{"blocks"})
	if err != nil {
		return fmt.Errorf("failed to get explicit blocks: %w", err)
	}

	// Combine both
	allBlocked := append(blockedTasks, explicitBlocks...)

	// Fetch task details
	blocked := []RelationshipWithTask{}
	for _, rel := range allBlocked {
		var blockedTask *models.Task
		var err error

		if rel.FromTaskID == task.ID {
			blockedTask, err = taskRepo.GetByID(ctx, rel.ToTaskID)
		} else {
			blockedTask, err = taskRepo.GetByID(ctx, rel.FromTaskID)
		}

		if err != nil {
			continue
		}

		blocked = append(blocked, RelationshipWithTask{
			RelationshipType: string(rel.RelationshipType),
			Direction:        "outgoing",
			TaskKey:          blockedTask.Key,
			TaskTitle:        blockedTask.Title,
			TaskStatus:       string(blockedTask.Status),
		})
	}

	// Output results
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":   taskKey,
			"task_title": task.Title,
			"blocks":     blocked,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	fmt.Printf("%s: %s\n\n", taskKey, task.Title)

	if len(blocked) == 0 {
		fmt.Println("Not blocking any tasks")
		return nil
	}

	fmt.Println("Blocks (waiting on this task):")
	for _, b := range blocked {
		status := getStatusIcon(b.TaskStatus)
		completed := ""
		if task.Status == "completed" {
			completed = " (unblocked)"
		}
		fmt.Printf("  %s %s: %s%s\n", status, b.TaskKey, b.TaskTitle, completed)
	}

	if task.Status == "completed" {
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
	deps, err := relRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
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
	dependents, err := relRepo.GetIncoming(ctx, task.ID, []string{"depends_on"})
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
	// If neither flag is set, show both
	if !showUpstream && !showDownstream {
		showUpstream = true
		showDownstream = true
	}

	var output strings.Builder

	// Show task header
	status := getStatusIcon(string(task.Status))
	output.WriteString(fmt.Sprintf("\n%s %s: %s\n", status, task.Key, task.Title))
	output.WriteString(strings.Repeat("=", 80) + "\n\n")

	// Build and show upstream dependencies tree
	if showUpstream {
		output.WriteString("Upstream Dependencies (Prerequisites):\n\n")
		upstreamTree, err := buildDependencyTree(ctx, taskRepo, relRepo, task, make(map[int64]bool), 0, maxDepth)
		if err != nil {
			return fmt.Errorf("failed to build upstream tree: %w", err)
		}

		if len(upstreamTree.Dependencies) == 0 {
			output.WriteString("  No upstream dependencies\n\n")
		} else {
			treeOutput := renderTree(upstreamTree, "", true)
			output.WriteString(treeOutput + "\n")
		}
	}

	// Build and show downstream dependents tree
	if showDownstream {
		output.WriteString("Downstream Dependents (Tasks waiting on this):\n\n")
		downstreamTree, err := buildDependentsTree(ctx, taskRepo, relRepo, task, make(map[int64]bool), 0, maxDepth)
		if err != nil {
			return fmt.Errorf("failed to build downstream tree: %w", err)
		}

		if len(downstreamTree.Dependents) == 0 {
			output.WriteString("  No downstream dependents\n\n")
		} else {
			treeOutput := renderTree(downstreamTree, "", true)
			output.WriteString(treeOutput + "\n")
		}
	}

	// Show legend
	output.WriteString("Legend: ✓ completed | ⊙ ready_for_review | • in_progress | ○ todo | ✗ blocked\n")

	// Output results
	if cli.GlobalConfig.JSON {
		// For JSON mode, return structured data
		jsonOutput := map[string]interface{}{
			"task_key":    task.Key,
			"task_title":  task.Title,
			"task_status": string(task.Status),
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

	// Human-readable output
	fmt.Print(output.String())
	return nil
}
