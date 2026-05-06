package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// sprintServicer defines the interface for sprint service operations used by CLI commands.
type sprintServicer interface {
	CreateSprint(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error)
	GetSprint(ctx context.Context, key string) (*models.Sprint, error)
	ListSprints(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error)
	UpdateSprint(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error)
	DeleteSprint(ctx context.Context, key string) error
	StartSprint(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprint(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprintWithCarryover(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error)
	ArchiveSprint(ctx context.Context, key string) (*models.Sprint, error)
	// F03: entity assignment and backlog
	AddEntityToSprint(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error)
	RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error
	GetSprintBacklog(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error)
	// F05: planning view, readiness, capacity, bulk-add
	PlanSprint(ctx context.Context, key string) (*services.SprintPlanView, error)
	GetSprintReadiness(ctx context.Context, key string) (*services.SprintReadiness, error)
	SetSprintCapacity(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error)
	GetSprintCapacity(ctx context.Context, key string) ([]services.CapacityRow, error)
	BulkAddToSprint(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error)
}

// sprintAnalyticsServicer defines the interface for sprint analytics operations used by CLI commands.
type sprintAnalyticsServicer interface {
	GetVelocity(ctx context.Context, n int) (*services.VelocityResult, error)
	GetBurndown(ctx context.Context, sprintKey string) (*services.BurndownResult, error)
	GetSummary(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error)
}

// sprintSvcOverride is non-nil only during tests.
var sprintSvcOverride sprintServicer

// sprintAnalyticsSvcOverride is non-nil only during tests.
var sprintAnalyticsSvcOverride sprintAnalyticsServicer

// getSprintService returns the service to use, preferring the test override.
func getSprintService() sprintServicer {
	if sprintSvcOverride != nil {
		return sprintSvcOverride
	}
	return cli.GetSprintService()
}

// getSprintAnalyticsService returns the analytics service to use, preferring the test override.
func getSprintAnalyticsService() sprintAnalyticsServicer {
	if sprintAnalyticsSvcOverride != nil {
		return sprintAnalyticsSvcOverride
	}
	return cli.GetSprintAnalyticsService()
}

// sprintCmd is the parent command for all sprint operations.
var sprintCmd = &cobra.Command{
	Use:     "sprint",
	Short:   "Manage sprints",
	GroupID: "advanced",
	Long: `Sprint management operations for planning and executing work sprints.

Sprints are assigned keys in format S### (e.g., S001, S042).

Examples:
  shark sprint create "Sprint 1" --start=2026-03-18 --end=2026-04-01
  shark sprint list
  shark sprint get S001
  shark sprint start S001
  shark sprint close S001`,
}

// sprintCreateCmd creates a new sprint.
var sprintCreateCmd = &cobra.Command{
	Use:   "create <name> --start=DATE --end=DATE [--goal=text]",
	Short: "Create a new sprint",
	Long: `Create a new sprint with auto-generated key (S### format).

Start and end dates must be in YYYY-MM-DD format.
End date must be after start date.

Examples:
  shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01
  shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01 --goal="Implement auth"
  shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintCreate,
}

// sprintGetCmd retrieves a specific sprint by key.
var sprintGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get sprint details",
	Long: `Display detailed information about a specific sprint.

Examples:
  shark sprint get S001
  shark sprint get S001 --json
  shark sprint get S001 --field status`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintGet,
}

// sprintListCmd lists sprints with optional filters.
var sprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sprints",
	Long: `List sprints with optional filtering by status.

Examples:
  shark sprint list
  shark sprint list --status=in_progress
  shark sprint list --status=completed --json`,
	RunE: runSprintList,
}

// sprintUpdateCmd updates sprint fields.
var sprintUpdateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Update a sprint",
	Long: `Update sprint fields (name, goal, end date).

At least one update flag must be provided.

Examples:
  shark sprint update S001 --name="Sprint 24 (Extended)"
  shark sprint update S001 --goal="Complete authentication"
  shark sprint update S001 --end=2026-04-08
  shark sprint update S001 --name="Updated" --goal="New goal" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintUpdate,
}

// sprintDeleteCmd deletes a sprint.
var sprintDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a sprint",
	Long: `Delete a sprint. Only sprints in planning status can be deleted.

Confirmation is required unless --force is provided.

Examples:
  shark sprint delete S001
  shark sprint delete S001 --force
  shark sprint delete S001 --force --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintDelete,
}

// sprintStartCmd starts a sprint (transitions to in_progress).
var sprintStartCmd = &cobra.Command{
	Use:   "start <key>",
	Short: "Start a sprint",
	Long: `Transition a sprint to in_progress status.

Only one sprint can be in_progress at a time.

Examples:
  shark sprint start S001
  shark sprint start S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintStart,
}

// sprintCloseCmd closes a sprint with optional carryover of incomplete entities.
var sprintCloseCmd = &cobra.Command{
	Use:   "close <key>",
	Short: "Close a sprint with carryover",
	Long: `Close an active sprint, optionally carrying over incomplete entities.

Carryover modes:
  next    — Move incomplete entities to the next planning sprint (auto-created if none exists).
  backlog — Soft-delete incomplete assignments, returning entities to the backlog.

The default mode is read from sprint_defaults.carryover_behavior in .sharkconfig.json
(defaults to "next" when absent).

Examples:
  shark sprint close S001
  shark sprint close S001 --carryover=next
  shark sprint close S001 --carryover=backlog
  shark sprint close S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintClose,
}

// sprintArchiveCmd archives a sprint (transitions to completed).
var sprintArchiveCmd = &cobra.Command{
	Use:   "archive <key>",
	Short: "Archive a sprint",
	Long: `Transition a sprint to completed status.

Examples:
  shark sprint archive S001
  shark sprint archive S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintArchive,
}

// sprintVelocityCmd shows velocity data for recent completed sprints.
var sprintVelocityCmd = &cobra.Command{
	Use:   "velocity",
	Short: "Show sprint velocity history",
	Long: `Show velocity data for the last N completed sprints.

Velocity is the total completed story-point size across the sprint.
Unsized entities are counted separately and do not contribute to velocity.

Examples:
  shark sprint velocity
  shark sprint velocity --sprints=10
  shark sprint velocity --json`,
	Args: cobra.NoArgs,
	RunE: runSprintVelocity,
}

// sprintBurndownCmd shows burndown data for a sprint.
var sprintBurndownCmd = &cobra.Command{
	Use:   "burndown [SPRINT_KEY]",
	Short: "Show sprint burndown chart",
	Long: `Show burndown data for a sprint.

When SPRINT_KEY is omitted, the current active sprint is used.
Future days show — in the Actual column.

Examples:
  shark sprint burndown
  shark sprint burndown S001
  shark sprint burndown S001 --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSprintBurndown,
}

// sprintSummaryCmd shows a summary report for a completed or archived sprint.
var sprintSummaryCmd = &cobra.Command{
	Use:   "summary <SPRINT_KEY>",
	Short: "Show sprint summary report",
	Long: `Show a summary report for a completed or archived sprint.

Summary includes: planned/completed size and count, velocity metrics,
and optionally detailed cycle-time, size-band distribution, and carryover entities.

Use --detailed to include extended analytics.

Examples:
  shark sprint summary S001
  shark sprint summary S001 --detailed
  shark sprint summary S001 --json
  shark sprint summary S001 --detailed --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintSummary,
}

// sprintAddCmd assigns one entity (or a bulk set) to a sprint.
var sprintAddCmd = &cobra.Command{
	Use:   "add <sprint-key> [entity-key]",
	Short: "Add an entity (or bulk entities) to a sprint",
	Long: `Assign one entity or a group of entities to a sprint.

Single-entity add (positional):
  shark sprint add S001 E07-F01-001

Single-entity add (flag):
  shark sprint add S001 --entity=E07-F01-001

Bulk-add all eligible tasks from a feature:
  shark sprint add S001 --bulk=E07-F34

Bulk-add all open bugs/tech-debt/change-cards not yet in a sprint:
  shark sprint add S001 --bulk-bugs
  shark sprint add S001 --bulk-tech-debt
  shark sprint add S001 --bulk-changes

Examples:
  shark sprint add S001 E07-F01-001
  shark sprint add S001 B001
  shark sprint add S001 --entity=B001
  shark sprint add S001 --bulk=E07-F34
  shark sprint add S001 --bulk-bugs --json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runSprintAdd,
}

// sprintRemoveCmd removes an entity from a sprint.
var sprintRemoveCmd = &cobra.Command{
	Use:   "remove <sprint-key> <entity-key>",
	Short: "Remove an entity from a sprint",
	Long: `Remove an entity assignment from a sprint.

Examples:
  shark sprint remove S001 E07-F01-001
  shark sprint remove S001 B001
  shark sprint remove S001 CC-001
  shark sprint remove S001 TD-001 --json`,
	Args: cobra.ExactArgs(2),
	RunE: runSprintRemove,
}

// sprintBacklogCmd shows all entities assigned to a sprint.
var sprintBacklogCmd = &cobra.Command{
	Use:   "backlog <sprint-key> [--type=task|bug|change_card|tech_debt] [--blocked]",
	Short: "View all entities assigned to a sprint",
	Long: `Display all entities assigned to a sprint, grouped by status.

Use --type to filter by entity type. Use --blocked to show only blocked entities.

Examples:
  shark sprint backlog S001
  shark sprint backlog S001 --type=task
  shark sprint backlog S001 --blocked
  shark sprint backlog S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintBacklog,
}

// sprintPlanCmd shows the composite planning view for a sprint.
var sprintPlanCmd = &cobra.Command{
	Use:   "plan <sprint-key>",
	Short: "Show sprint planning view",
	Long: `Display the composite planning view for a sprint.

The planning view shows three sections:
  Backlog   — unassigned entities eligible for assignment
  Capacity  — per-agent-type capacity vs. allocated story points
  Readiness — 0-100 readiness score with 6-factor breakdown

Examples:
  shark sprint plan S001
  shark sprint plan S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintPlan,
}

// sprintReadinessCmd shows the readiness score for a sprint.
var sprintReadinessCmd = &cobra.Command{
	Use:   "readiness <sprint-key>",
	Short: "Show sprint readiness score",
	Long: `Display the readiness score (0-100) and 6-factor breakdown for a sprint.

Examples:
  shark sprint readiness S001
  shark sprint readiness S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintReadiness,
}

// sprintCapacityCmd is the parent for capacity sub-commands.
var sprintCapacityCmd = &cobra.Command{
	Use:   "capacity",
	Short: "Manage sprint capacity",
	Long: `Set or show per-sprint agent-type capacity configuration.

Examples:
  shark sprint capacity set S001 --agent=backend --points=21
  shark sprint capacity show S001
  shark sprint capacity set --default --agent=backend --points=21`,
}

// sprintCapacitySetCmd sets or updates agent-type capacity for a sprint (or default).
var sprintCapacitySetCmd = &cobra.Command{
	Use:   "set [sprint-key]",
	Short: "Set capacity for a sprint or update the default",
	Long: `Create or update capacity for a (sprint, agent-type) pair.

With --default, writes the value to sprint_defaults.capacity in .sharkconfig.json
and does NOT modify the sprint_capacity table. New sprints will inherit the value.

Without --default, upserts the sprint_capacity row for the given sprint key.

Examples:
  shark sprint capacity set S001 --agent=backend --points=21
  shark sprint capacity set --default --agent=backend --points=21`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSprintCapacitySet,
}

// sprintCapacityShowCmd displays capacity vs. allocation for a sprint.
var sprintCapacityShowCmd = &cobra.Command{
	Use:   "show <sprint-key>",
	Short: "Show capacity vs. allocation for a sprint",
	Long: `Display a table of agent-type capacity vs. allocated story points.

Columns: AgentType | Capacity | Allocated | Remaining | Unsized

Examples:
  shark sprint capacity show S001
  shark sprint capacity show S001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintCapacityShow,
}

// Command flag variables
var (
	sprintStartDate string
	sprintEndDate   string
	sprintGoal      string
	sprintStatus    string
	sprintName      string
	sprintForce     bool
)

func init() {
	// Register sprint command and subcommands
	cli.RootCmd.AddCommand(sprintCmd)
	sprintCmd.AddCommand(sprintCreateCmd)
	sprintCmd.AddCommand(sprintGetCmd)
	sprintCmd.AddCommand(sprintListCmd)
	sprintCmd.AddCommand(sprintUpdateCmd)
	sprintCmd.AddCommand(sprintDeleteCmd)
	sprintCmd.AddCommand(sprintStartCmd)
	sprintCmd.AddCommand(sprintCloseCmd)
	sprintCmd.AddCommand(sprintArchiveCmd)
	sprintCmd.AddCommand(sprintVelocityCmd)
	sprintCmd.AddCommand(sprintBurndownCmd)
	sprintCmd.AddCommand(sprintSummaryCmd)
	// F03 assignment commands
	sprintCmd.AddCommand(sprintAddCmd)
	sprintCmd.AddCommand(sprintRemoveCmd)
	sprintCmd.AddCommand(sprintBacklogCmd)
	// F05 commands
	sprintCmd.AddCommand(sprintPlanCmd)
	sprintCmd.AddCommand(sprintReadinessCmd)
	sprintCapacityCmd.AddCommand(sprintCapacitySetCmd)
	sprintCapacityCmd.AddCommand(sprintCapacityShowCmd)
	sprintCmd.AddCommand(sprintCapacityCmd)

	// Create flags
	sprintCreateCmd.Flags().StringVar(&sprintStartDate, "start", "", "Sprint start date (YYYY-MM-DD)")
	sprintCreateCmd.Flags().StringVar(&sprintEndDate, "end", "", "Sprint end date (YYYY-MM-DD)")
	sprintCreateCmd.Flags().StringVar(&sprintGoal, "goal", "", "Sprint goal (optional)")
	_ = sprintCreateCmd.MarkFlagRequired("start")
	_ = sprintCreateCmd.MarkFlagRequired("end")

	// List flags
	sprintListCmd.Flags().StringVar(&sprintStatus, "status", "", "Filter by status")

	// Update flags
	sprintUpdateCmd.Flags().StringVar(&sprintName, "name", "", "New sprint name")
	sprintUpdateCmd.Flags().StringVar(&sprintGoal, "goal", "", "New sprint goal")
	sprintUpdateCmd.Flags().StringVar(&sprintEndDate, "end", "", "New sprint end date (YYYY-MM-DD)")

	// Delete flags
	sprintDeleteCmd.Flags().BoolVar(&sprintForce, "force", false, "Skip confirmation prompt")

	// Close flags (T-E19-F03-007)
	sprintCloseCmd.Flags().String("carryover", "", "Carryover mode: next or backlog (default from config)")

	// Velocity flags
	sprintVelocityCmd.Flags().Int("sprints", 5, "Number of recent sprints to include (1–100, default 5)")

	// Summary flags
	sprintSummaryCmd.Flags().Bool("detailed", false, "Include detailed cycle-time, size-band distribution, and carryover entities")

	// Backlog flags
	sprintBacklogCmd.Flags().String("type", "", "Filter by entity type: task, bug, change_card, tech_debt")
	sprintBacklogCmd.Flags().Bool("blocked", false, "Show only blocked entities")

	// Add flags (F05 - single entity and bulk)
	sprintAddCmd.Flags().String("entity", "", "Entity key to assign (e.g., E07-F01-001, B001)")
	sprintAddCmd.Flags().String("bulk", "", "Feature key: assign all eligible tasks from this feature")
	sprintAddCmd.Flags().Bool("bulk-bugs", false, "Assign all open bugs not already in a sprint")
	sprintAddCmd.Flags().Bool("bulk-tech-debt", false, "Assign all open tech-debt items not already in a sprint")
	sprintAddCmd.Flags().Bool("bulk-changes", false, "Assign all open change-cards not already in a sprint")

	// Capacity set flags
	sprintCapacitySetCmd.Flags().String("agent", "", "Agent type (e.g., backend, frontend, qa)")
	sprintCapacitySetCmd.Flags().Float64("points", 0, "Capacity in story points (must be > 0)")
	sprintCapacitySetCmd.Flags().Bool("default", false, "Write to sprint_defaults in .sharkconfig.json instead of per-sprint DB row")
	_ = sprintCapacitySetCmd.MarkFlagRequired("agent")
	_ = sprintCapacitySetCmd.MarkFlagRequired("points")
}

// runSprintCreate handles the `shark sprint create` command.
func runSprintCreate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("sprint name cannot be empty")
	}

	startStr, _ := cmd.Flags().GetString("start")
	endStr, _ := cmd.Flags().GetString("end")
	goal, _ := cmd.Flags().GetString("goal")

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
	}

	// Step 2: Call service
	svc := getSprintService()
	input := services.CreateSprintInput{
		Name:      name,
		Goal:      goal,
		StartDate: startDate,
		EndDate:   endDate,
	}

	sprint, err := svc.CreateSprint(cmd.Context(), input)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprint)
	}

	cli.Success(fmt.Sprintf("Created sprint %s: %s", sprint.Key, sprint.Name))
	return nil
}

// runSprintGet handles the `shark sprint get` command.
func runSprintGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service
	svc := getSprintService()
	sprint, err := svc.GetSprint(ctx, key)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprint)
	}

	// Human-readable output
	info := [][]string{
		{"Key", sprint.Key},
		{"Name", sprint.Name},
		{"Status", string(sprint.Status)},
		{"Start Date", sprint.StartDate.Format("2006-01-02")},
		{"End Date", sprint.EndDate.Format("2006-01-02")},
	}

	if sprint.Goal != "" {
		info = append(info, []string{"Goal", sprint.Goal})
	}

	info = append(info, []string{"Created", sprint.CreatedAt.Format(time.RFC3339)})
	info = append(info, []string{"Updated", sprint.UpdatedAt.Format(time.RFC3339)})

	RenderEntity(EntityDisplayOptions{
		EntityType: "sprint",
		Key:        sprint.Key,
		Status:     string(sprint.Status),
		BasicInfo:  info,
	})
	return nil
}

// runSprintList handles the `shark sprint list` command.
func runSprintList(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	statusStr, _ := cmd.Flags().GetString("status")

	var filters *services.SprintListFilters
	if statusStr != "" {
		filters = &services.SprintListFilters{
			Status: statusStr,
		}
	}

	// Step 2: Call service
	svc := getSprintService()
	sprints, err := svc.ListSprints(cmd.Context(), filters)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprints)
	}

	if len(sprints) == 0 {
		cli.Info("No sprints found")
		return nil
	}

	return printSprintTable(sprints)
}

// runSprintUpdate handles the `shark sprint update` command.
func runSprintUpdate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	updates := services.UpdateSprintInput{}

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		updates.Name = &name
	}

	if cmd.Flags().Changed("goal") {
		goal, _ := cmd.Flags().GetString("goal")
		updates.Goal = &goal
	}

	if cmd.Flags().Changed("end") {
		endStr, _ := cmd.Flags().GetString("end")
		endDate, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
		}
		updates.EndDate = &endDate
	}

	if updates.Name == nil && updates.Goal == nil && updates.EndDate == nil {
		return fmt.Errorf("at least one update flag is required (--name, --goal, or --end)")
	}

	// Step 2: Call service
	svc := getSprintService()
	sprint, err := svc.UpdateSprint(cmd.Context(), key, updates)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprint)
	}

	cli.Success(fmt.Sprintf("Updated sprint %s", sprint.Key))
	return nil
}

// runSprintDelete handles the `shark sprint delete` command.
func runSprintDelete(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	force, _ := cmd.Flags().GetBool("force")

	svc := getSprintService()

	// Confirm deletion unless --force
	if !force {
		sprint, err := svc.GetSprint(cmd.Context(), key)
		if err != nil {
			return err
		}
		if !confirmSprintDelete(sprint) {
			cli.Info("Delete cancelled")
			return nil
		}
	}

	// Step 2: Call service
	if err := svc.DeleteSprint(cmd.Context(), key); err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": key})
	}

	cli.Success(fmt.Sprintf("Deleted sprint %s", key))
	return nil
}

// runSprintStart handles the `shark sprint start` command.
func runSprintStart(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	svc := getSprintService()
	sprint, err := svc.StartSprint(cmd.Context(), key)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprint)
	}

	cli.Success(fmt.Sprintf("Started sprint %s (%s)", sprint.Key, sprint.Name))
	return nil
}

// runSprintClose handles the `shark sprint close` command.
// Delegates to CloseSprintWithCarryover for atomic close + carryover support (T-E19-F03-007).
func runSprintClose(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	carryoverFlag, _ := cmd.Flags().GetString("carryover")

	// Step 2: Call service
	svc := getSprintService()
	result, err := svc.CloseSprintWithCarryover(cmd.Context(), key, services.CarryoverMode(carryoverFlag))
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("Closed sprint %s (%s)", result.Sprint.Key, result.Sprint.Name))
	fmt.Printf("  Completed: %d  Carried over: %d  Dropped: %d\n",
		result.CompletedCount, result.CarriedOverCount, result.DroppedCount)
	if result.NextSprintKey != "" {
		fmt.Printf("  Incomplete entities moved to: %s\n", result.NextSprintKey)
	}
	return nil
}

// runSprintArchive handles the `shark sprint archive` command.
func runSprintArchive(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	svc := getSprintService()
	sprint, err := svc.ArchiveSprint(cmd.Context(), key)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprint)
	}

	cli.Success(fmt.Sprintf("Archived sprint %s (%s)", sprint.Key, sprint.Name))
	return nil
}

// runSprintVelocity handles the `shark sprint velocity` command.
func runSprintVelocity(cmd *cobra.Command, _ []string) error {
	// Step 1: Parse
	n, _ := cmd.Flags().GetInt("sprints")

	// Step 2: Call service (validation of n range 1–100 happens in service)
	svc := getSprintAnalyticsService()
	result, err := svc.GetVelocity(cmd.Context(), n)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	return printVelocityTable(result)
}

// runSprintBurndown handles the `shark sprint burndown [SPRINT_KEY]` command.
func runSprintBurndown(cmd *cobra.Command, args []string) error {
	// Step 1: Parse — key is optional; empty string means "active sprint"
	var sprintKey string
	if len(args) > 0 {
		sprintKey = args[0]
	}

	// Step 2: Call service
	svc := getSprintAnalyticsService()
	result, err := svc.GetBurndown(cmd.Context(), sprintKey)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	return printBurndownTable(result)
}

// runSprintSummary handles the `shark sprint summary <SPRINT_KEY>` command.
func runSprintSummary(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]
	detailed, _ := cmd.Flags().GetBool("detailed")

	// Step 2: Call service
	svc := getSprintAnalyticsService()
	result, err := svc.GetSummary(cmd.Context(), sprintKey, detailed)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	return printSummaryTable(result, detailed)
}

// printVelocityTable formats and prints velocity data as a human-readable table.
func printVelocityTable(result *services.VelocityResult) error {
	if result.InsufficientData {
		cli.Warning(fmt.Sprintf("Insufficient data: need at least 3 completed sprints (have %d)", result.SprintCount))
	}

	if len(result.Sprints) == 0 {
		cli.Info("No completed sprints found")
		return nil
	}

	fmt.Printf("\nSprint Velocity (last %d sprints)\n", result.SprintCount)

	headers := []string{"KEY", "NAME", "COMPLETED", "UNSIZED"}
	rows := make([][]string, 0, len(result.Sprints))
	for _, s := range result.Sprints {
		rows = append(rows, []string{
			s.Key,
			s.Name,
			fmt.Sprintf("%d", s.CompletedSize),
			fmt.Sprintf("%d", s.UnsizedCompleted),
		})
	}
	cli.OutputTable(headers, rows)

	fmt.Printf("Trailing Average: %.1f\n\n", result.TrailingAverage)
	return nil
}

// printBurndownTable formats and prints burndown data as a human-readable ASCII table.
// Uses ASCII-only characters except for em-dash (U+2014) for future days (AC-B-6).
func printBurndownTable(result *services.BurndownResult) error {
	if len(result.DataPoints) == 0 {
		cli.Info(fmt.Sprintf("No burndown data available for sprint %s (planning status has no data)", result.SprintKey))
		return nil
	}

	fmt.Printf("\nBurndown: %s (%s)\n", result.SprintKey, result.SprintName)
	fmt.Printf("Total Size: %d  Unsized: %d\n\n", result.TotalSize, result.UnsizedTotal)

	headers := []string{"DAY", "IDEAL", "ACTUAL", "UNSIZED"}
	rows := make([][]string, 0, len(result.DataPoints))
	for _, dp := range result.DataPoints {
		var actualStr string
		if dp.ActualRemaining != nil {
			actualStr = fmt.Sprintf("%.1f", *dp.ActualRemaining)
		} else {
			actualStr = "—" // em-dash U+2014 for future days (AC-B-6 allows em-dash)
		}
		rows = append(rows, []string{
			dp.Date.Format("2006-01-02"),
			fmt.Sprintf("%.1f", dp.IdealRemaining),
			actualStr,
			fmt.Sprintf("%d", dp.UnsizedRemaining),
		})
	}
	cli.OutputTable(headers, rows)
	fmt.Println()
	return nil
}

// printSummaryTable formats and prints sprint summary data as a human-readable display.
func printSummaryTable(result *services.SprintSummaryResult, detailed bool) error {
	fmt.Printf("\nSprint Summary: %s (%s)\n\n", result.SprintKey, result.SprintName)
	fmt.Printf("  Planned size:     %d\n", result.PlannedSize)
	fmt.Printf("  Completed size:   %d  (%.1f%%)\n", result.CompletedSize, result.CompletionPctBySize)
	fmt.Printf("  Planned count:    %d\n", result.PlannedCount)
	fmt.Printf("  Completed count:  %d\n", result.CompletedCount)
	fmt.Printf("  Unsized planned:  %d\n", result.UnsizedPlanned)
	fmt.Printf("  Unsized completed:%d\n", result.UnsizedCompleted)
	fmt.Printf("\nVelocity\n")
	fmt.Printf("  This sprint:      %d\n", result.VelocityThisSprint)
	fmt.Printf("  Trailing average: %.1f\n", result.TrailingAvgVelocity)
	fmt.Printf("  Delta:            %+.1f  (%+.1f%%)\n", result.VelocityDelta, result.VelocityDeltaPct)

	if detailed {
		fmt.Printf("\nDetailed\n")
		if result.AddedMidSprintCount != nil {
			fmt.Printf("  Added mid-sprint: %d (size: %d)\n", *result.AddedMidSprintCount, *result.AddedMidSprintSize)
		}
		if result.RemovedMidSprintCount != nil {
			fmt.Printf("  Removed mid-sprint:%d (size: %d)\n", *result.RemovedMidSprintCount, *result.RemovedMidSprintSize)
		}

		if len(result.CycleTimeByPhase) > 0 {
			fmt.Printf("\nCycle Time by Phase\n")
			cycleHeaders := []string{"PHASE", "AVG DAYS"}
			cycleRows := make([][]string, 0, len(result.CycleTimeByPhase))
			for _, p := range result.CycleTimeByPhase {
				cycleRows = append(cycleRows, []string{p.Phase, fmt.Sprintf("%.1f", p.AverageDays)})
			}
			cli.OutputTable(cycleHeaders, cycleRows)
		} else {
			fmt.Printf("\n  No session data available (install E13 for cycle-time tracking)\n")
		}

		if len(result.SizeBandDistribution) > 0 {
			fmt.Printf("\nSize Distribution\n")
			sizeHeaders := []string{"SIZE", "COUNT"}
			sizeRows := make([][]string, 0, len(result.SizeBandDistribution))
			for _, b := range result.SizeBandDistribution {
				sizeRows = append(sizeRows, []string{b.Label, fmt.Sprintf("%d", b.Count)})
			}
			cli.OutputTable(sizeHeaders, sizeRows)
		}

		if len(result.CarryoverEntities) > 0 {
			fmt.Printf("\nCarryover Entities (%d)\n", len(result.CarryoverEntities))
			for _, e := range result.CarryoverEntities {
				sizeStr := "unsized"
				if e.Size != nil {
					sizeStr = fmt.Sprintf("size=%d", *e.Size)
				}
				fmt.Printf("  %s (%s, %s)\n", e.Key, e.EntityType, sizeStr)
			}
		}
	}
	fmt.Println()
	return nil
}

// =============================================================================
// F05 run functions: add, plan, readiness, capacity set/show
// =============================================================================

// runSprintAdd handles `shark sprint add <sprint-key> [entity-key]`.
// Supports single-entity add (positional args[1] or --entity flag) and
// bulk-add (--bulk, --bulk-bugs, --bulk-tech-debt, --bulk-changes).
// Bulk flags route to BulkAddToSprint; single-entity routes to AddEntityToSprint.
func runSprintAdd(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]

	bulkFeature, _ := cmd.Flags().GetString("bulk")
	bulkBugs, _ := cmd.Flags().GetBool("bulk-bugs")
	bulkTechDebt, _ := cmd.Flags().GetBool("bulk-tech-debt")
	bulkChanges, _ := cmd.Flags().GetBool("bulk-changes")
	entityKeyFlag, _ := cmd.Flags().GetString("entity")

	// Entity key may come from positional arg or --entity flag.
	entityKey := entityKeyFlag
	if entityKey == "" && len(args) >= 2 {
		entityKey = args[1]
	}

	svc := getSprintService()

	// Step 2: Route — bulk paths call BulkAddToSprint; single path calls AddEntityToSprint.
	if bulkFeature != "" {
		input := services.BulkAddInput{SprintKey: sprintKey, FeatureKey: bulkFeature}
		result, err := svc.BulkAddToSprint(cmd.Context(), input)
		if err != nil {
			return err
		}
		return formatBulkAddResult(result)
	}

	if bulkBugs || bulkTechDebt || bulkChanges {
		var entityTypes []string
		if bulkBugs {
			entityTypes = append(entityTypes, "bug")
		}
		if bulkTechDebt {
			entityTypes = append(entityTypes, "tech_debt")
		}
		if bulkChanges {
			entityTypes = append(entityTypes, "change_card")
		}
		input := services.BulkAddInput{SprintKey: sprintKey, EntityTypes: entityTypes}
		result, err := svc.BulkAddToSprint(cmd.Context(), input)
		if err != nil {
			return err
		}
		return formatBulkAddResult(result)
	}

	// Single-entity add
	if entityKey == "" {
		return fmt.Errorf("provide an entity key (positional arg or --entity=<key>), or use --bulk/--bulk-bugs/--bulk-tech-debt/--bulk-changes for bulk add")
	}
	assignment, warning, err := svc.AddEntityToSprint(cmd.Context(), services.AddEntityInput{
		SprintKey: sprintKey,
		EntityKey: entityKey,
	})
	if err != nil {
		return err
	}

	// Step 3: Format
	if warning != nil {
		cli.Warning(fmt.Sprintf("Capacity warning: %s is over capacity (%.0f/%.0f pts allocated)",
			warning.AgentType, warning.Allocated, warning.Capacity))
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(assignment)
	}
	cli.Success(fmt.Sprintf("Added %s to sprint %s", entityKey, sprintKey))
	return nil
}

// runSprintRemove handles `shark sprint remove <sprint-key> <entity-key>`.
func runSprintRemove(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]
	entityKey := args[1]

	// Step 2: Call service
	svc := getSprintService()
	if err := svc.RemoveEntityFromSprint(cmd.Context(), sprintKey, entityKey); err != nil {
		return err
	}

	// Step 3: Format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"ok":         true,
			"sprint_key": sprintKey,
			"entity_key": entityKey,
		})
	}
	cli.Success(fmt.Sprintf("Removed %s from sprint %s", entityKey, sprintKey))
	return nil
}

// runSprintBacklog handles `shark sprint backlog <sprint-key>`.
func runSprintBacklog(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]
	entityType, _ := cmd.Flags().GetString("type")
	blockedOnly, _ := cmd.Flags().GetBool("blocked")

	// Step 2: Call service
	svc := getSprintService()
	backlog, err := svc.GetSprintBacklog(cmd.Context(), sprintKey, services.BacklogOptions{
		EntityType:  entityType,
		BlockedOnly: blockedOnly,
	})
	if err != nil {
		return err
	}

	// Step 3: Format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(backlog)
	}
	return printBacklog(backlog)
}

// printBacklog prints the sprint backlog as human-readable grouped sections.
func printBacklog(backlog *services.SprintBacklog) error {
	fmt.Printf("\nBacklog: %s (%s)  —  %.0f%% complete (%d/%d)\n\n",
		backlog.SprintKey, backlog.SprintName,
		backlog.CompletionPercent, backlog.CompletedCount, backlog.TotalCount)

	if backlog.TotalCount == 0 {
		cli.Info("No entities assigned to this sprint.")
		return nil
	}

	for _, group := range backlog.Groups {
		fmt.Printf("--- %s (%d) ---\n", group.StatusCategory, len(group.Items))
		if len(group.Items) == 0 {
			continue
		}
		fmt.Printf("  %-15s %-12s %-12s  %s\n", "KEY", "TYPE", "STATUS", "TITLE")
		fmt.Printf("  %-15s %-12s %-12s  %s\n",
			strings.Repeat("-", 15), strings.Repeat("-", 12), strings.Repeat("-", 12), strings.Repeat("-", 30))
		for _, item := range group.Items {
			// Display label uses brackets (display-only); JSON uses raw entity_type string.
			typeLabel := fmt.Sprintf("[%s]", item.EntityType)
			title := item.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			fmt.Printf("  %-15s %-12s %-12s  %s\n", item.Key, typeLabel, item.Status, title)
		}
		fmt.Println()
	}
	return nil
}

// formatBulkAddResult formats and outputs a BulkAddResult (JSON or human).
func formatBulkAddResult(result *services.BulkAddResult) error {
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}
	// Human output: summarise per-type counts
	totalAdded := 0
	for _, n := range result.AddedByType {
		totalAdded += n
	}
	totalSkipped := 0
	for _, n := range result.SkippedByType {
		totalSkipped += n
	}
	cli.Success(fmt.Sprintf("Bulk add complete: %d added, %d skipped", totalAdded, totalSkipped))
	for t, n := range result.AddedByType {
		if n > 0 {
			cli.Info(fmt.Sprintf("  %s: %d added", t, n))
		}
	}
	for _, w := range result.CapacityWarnings {
		cli.Warning(fmt.Sprintf("Capacity warning: %s over capacity (%.0f/%.0f pts)", w.AgentType, w.Allocated, w.Capacity))
	}
	return nil
}

// runSprintPlan handles `shark sprint plan <sprint-key>`.
func runSprintPlan(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]

	// Step 2: Call service
	svc := getSprintService()
	view, err := svc.PlanSprint(cmd.Context(), sprintKey)
	if err != nil {
		return err
	}

	// Step 3: Format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(view)
	}
	return printSprintPlanView(view)
}

// printSprintPlanView prints the three planning sections for human output.
func printSprintPlanView(view *services.SprintPlanView) error {
	sprintLabel := ""
	if view.Sprint != nil {
		sprintLabel = fmt.Sprintf(" %s (%s)", view.Sprint.Key, view.Sprint.Name)
	}
	fmt.Printf("\nSprint Planning View%s\n\n", sprintLabel)

	// Section 1: Backlog
	fmt.Printf("=== Backlog (%d unassigned entities) ===\n", len(view.Backlog))
	if len(view.Backlog) == 0 {
		fmt.Println("  (no unassigned entities)")
	} else {
		fmt.Printf("%-15s %-10s %-8s %s\n", "KEY", "TYPE", "SIZE", "TITLE")
		fmt.Printf("%-15s %-10s %-8s %s\n",
			strings.Repeat("-", 15), strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 30))
		for _, item := range view.Backlog {
			sizeStr := "-"
			if item.Size != nil {
				sizeStr = fmt.Sprintf("%d", *item.Size)
			}
			title := item.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			fmt.Printf("%-15s %-10s %-8s %s\n", item.Key, item.EntityType, sizeStr, title)
		}
	}

	// Section 2: Capacity
	fmt.Printf("\n=== Capacity ===\n")
	if len(view.Capacity) == 0 {
		fmt.Println("  (no capacity configured — use `shark sprint capacity set`)")
	} else {
		fmt.Printf("%-12s %10s %10s %10s %8s\n", "AGENT", "CAPACITY", "ALLOCATED", "REMAINING", "UNSIZED")
		fmt.Printf("%-12s %10s %10s %10s %8s\n",
			strings.Repeat("-", 12), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8))
		for _, row := range view.Capacity {
			fmt.Printf("%-12s %10.0f %10.0f %10.0f %8d\n",
				row.AgentType, row.CapacityPoints, row.AllocatedPoints, row.Remaining, row.UnsizedAssigned)
		}
	}

	// Section 3: Readiness
	fmt.Printf("\n=== Readiness ===\n")
	if view.Readiness != nil {
		fmt.Printf("Overall Score: %d / 100\n", view.Readiness.OverallScore)
		if len(view.Readiness.Factors) > 0 {
			fmt.Printf("%-30s %6s %8s  %s\n", "FACTOR", "SCORE", "MAX", "DETAIL")
			fmt.Printf("%-30s %6s %8s  %s\n",
				strings.Repeat("-", 30), strings.Repeat("-", 6), strings.Repeat("-", 8), strings.Repeat("-", 30))
			for _, f := range view.Readiness.Factors {
				fmt.Printf("%-30s %6d %8d  %s\n", f.Name, f.Score, f.MaxScore, f.Detail)
			}
		}
	}
	fmt.Println()
	return nil
}

// runSprintReadiness handles `shark sprint readiness <sprint-key>`.
func runSprintReadiness(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]

	// Step 2: Call service
	svc := getSprintService()
	readiness, err := svc.GetSprintReadiness(cmd.Context(), sprintKey)
	if err != nil {
		return err
	}

	// Step 3: Format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(readiness)
	}
	return printReadiness(sprintKey, readiness)
}

// printReadiness formats the readiness score and factor table for human output.
func printReadiness(sprintKey string, r *services.SprintReadiness) error {
	fmt.Printf("\nSprint Readiness: %s\n", sprintKey)
	fmt.Printf("Overall Score: %d / 100\n\n", r.OverallScore)

	if len(r.Factors) > 0 {
		fmt.Printf("%-30s %6s %8s  %s\n", "FACTOR", "SCORE", "MAX", "DETAIL")
		fmt.Printf("%-30s %6s %8s  %s\n",
			strings.Repeat("-", 30), strings.Repeat("-", 6), strings.Repeat("-", 8), strings.Repeat("-", 30))
		for _, f := range r.Factors {
			fmt.Printf("%-30s %6d %8d  %s\n", f.Name, f.Score, f.MaxScore, f.Detail)
		}
	}

	if len(r.UnsizedEntities) > 0 {
		fmt.Printf("\nUnsized entities (%d):\n", len(r.UnsizedEntities))
		for _, e := range r.UnsizedEntities {
			fmt.Printf("  %s  %s\n", e.Key, e.Title)
		}
	}
	if len(r.OversizedEntities) > 0 {
		fmt.Printf("\nOversized entities (%d):\n", len(r.OversizedEntities))
		for _, e := range r.OversizedEntities {
			fmt.Printf("  %s  %s\n", e.Key, e.Title)
		}
	}
	fmt.Println()
	return nil
}

// runSprintCapacitySet handles `shark sprint capacity set [sprint-key]`.
// With --default, writes to .sharkconfig.json and does NOT call the service.
// Without --default, calls SetSprintCapacity on the service for the sprint.
func runSprintCapacitySet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	agentType, _ := cmd.Flags().GetString("agent")
	points, _ := cmd.Flags().GetFloat64("points")
	isDefault, _ := cmd.Flags().GetBool("default")

	// Step 2: Route based on --default flag
	if isDefault {
		// Write to config file only — do NOT call SetSprintCapacity service
		configPath, err := cli.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to find config path: %w", err)
		}
		mgr := config.NewManager(configPath)
		if err := mgr.SetSprintCapacityDefault(agentType, points); err != nil {
			return fmt.Errorf("failed to update sprint defaults: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"agent_type": agentType,
				"points":     points,
				"updated":    "sprint_defaults.capacity",
			})
		}
		cli.Success(fmt.Sprintf("Updated sprint default capacity: %s = %.0f pts", agentType, points))
		return nil
	}

	// Per-sprint set
	if len(args) == 0 {
		return fmt.Errorf("sprint key required (or use --default to set config default)")
	}
	sprintKey := args[0]

	svc := getSprintService()
	cap, err := svc.SetSprintCapacity(cmd.Context(), services.SetSprintCapacityInput{
		SprintKey: sprintKey,
		AgentType: agentType,
		Points:    points,
	})
	if err != nil {
		return err
	}

	// Step 3: Format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cap)
	}
	cli.Success(fmt.Sprintf("Set capacity for %s in sprint %s: %.0f pts", agentType, sprintKey, points))
	return nil
}

// runSprintCapacityShow handles `shark sprint capacity show <sprint-key>`.
func runSprintCapacityShow(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]

	// Step 2: Call service
	svc := getSprintService()
	rows, err := svc.GetSprintCapacity(cmd.Context(), sprintKey)
	if err != nil {
		return err
	}

	// Step 3: Format
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(rows)
	}
	return printCapacityTable(sprintKey, rows)
}

// printCapacityTable formats and prints capacity rows for human output.
func printCapacityTable(sprintKey string, rows []services.CapacityRow) error {
	fmt.Printf("\nCapacity: %s\n\n", sprintKey)
	if len(rows) == 0 {
		cli.Info("No capacity configured. Use `shark sprint capacity set` to configure agent capacity.")
		return nil
	}
	fmt.Printf("%-12s %10s %10s %10s %8s\n", "AGENT", "CAPACITY", "ALLOCATED", "REMAINING", "UNSIZED")
	fmt.Printf("%-12s %10s %10s %10s %8s\n",
		strings.Repeat("-", 12), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8))
	for _, row := range rows {
		fmt.Printf("%-12s %10.0f %10.0f %10.0f %8d\n",
			row.AgentType, row.CapacityPoints, row.AllocatedPoints, row.Remaining, row.UnsizedAssigned)
	}
	fmt.Println()
	return nil
}

// --- Helper functions ---

// confirmSprintDelete prompts the user to confirm deletion of a sprint.
func confirmSprintDelete(sprint *models.Sprint) bool {
	fmt.Printf("Delete sprint %s (%s)? This action cannot be undone. (yes/no): ", sprint.Key, sprint.Name)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "yes"
}

// printSprintTable formats and prints sprints as a table.
func printSprintTable(sprints []*models.Sprint) error {
	headers := []string{"KEY", "NAME", "STATUS", "START DATE", "END DATE"}
	rows := make([][]string, 0, len(sprints))

	for _, s := range sprints {
		rows = append(rows, []string{
			s.Key,
			s.Name,
			string(s.Status),
			s.StartDate.Format("2006-01-02"),
			s.EndDate.Format("2006-01-02"),
		})
	}

	cli.OutputTable(headers, rows)
	return nil
}
