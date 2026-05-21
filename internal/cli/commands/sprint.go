package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/fileops"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	wf "github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/spf13/cobra"
)

// sprintLifecycleServicer defines the lifecycle-focused sprint operations used
// by create/get/list/update/start/close/archive commands.
type sprintLifecycleServicer interface {
	CreateSprint(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error)
	GetSprint(ctx context.Context, key string) (*models.Sprint, error)
	ListSprints(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error)
	UpdateSprint(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error)
	DeleteSprint(ctx context.Context, key string) error
	StartSprint(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprint(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprintWithCarryover(ctx context.Context, key string, mode services.CarryoverMode) (*services.SprintCloseResult, error)
	ArchiveSprint(ctx context.Context, key string) (*models.Sprint, error)
	// CountNullSprintOrder returns the count of active assignments with sprint_order = NULL
	// for the given sprint. Used by runSprintStart to surface the REQ-F-009 soft warning.
	CountNullSprintOrder(ctx context.Context, sprintKey string) (int, error)
}

// sprintAssignmentServicer defines the assignment/backlog operations used by
// add/remove/backlog commands and bulk-add flows.
type sprintAssignmentServicer interface {
	// F03: entity assignment and backlog
	AddEntityToSprint(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error)
	RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error
	GetSprintBacklog(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error)
	BulkAddToSprint(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error)
	GetNextTask(ctx context.Context, agentType string) (*services.BacklogItemView, error)
	// F07: sprint-relative ordering
	ReorderAssignment(ctx context.Context, sprintKey, entityKey string, target services.ReorderTarget) (*models.SprintAssignment, []*models.SprintAssignment, error)
}

// sprintPlanningServicer defines the planning/readiness view commands.
type sprintPlanningServicer interface {
	PlanSprint(ctx context.Context, key string) (*services.SprintPlanView, error)
	GetSprintReadiness(ctx context.Context, key string) (*services.SprintReadiness, error)
}

// sprintCapacityServicer defines the sprint capacity commands.
type sprintCapacityServicer interface {
	SetSprintCapacity(ctx context.Context, input services.SetSprintCapacityInput) (*models.SprintCapacity, error)
	GetSprintCapacity(ctx context.Context, key string) ([]services.CapacityRow, error)
}

// sprintAnalyticsServicer defines the interface for sprint analytics operations used by CLI commands.
type sprintAnalyticsServicer interface {
	GetVelocity(ctx context.Context, n int) (*services.VelocityResult, error)
	GetBurndown(ctx context.Context, sprintKey string) (*services.BurndownResult, error)
	GetSummary(ctx context.Context, sprintKey string, detailed bool) (*services.SprintSummaryResult, error)
}

// sprintLifecycleSvcOverride is non-nil only during tests.
var sprintLifecycleSvcOverride sprintLifecycleServicer

// sprintAssignmentSvcOverride is non-nil only during tests.
var sprintAssignmentSvcOverride sprintAssignmentServicer

// sprintPlanningSvcOverride is non-nil only during tests.
var sprintPlanningSvcOverride sprintPlanningServicer

// sprintCapacitySvcOverride is non-nil only during tests.
var sprintCapacitySvcOverride sprintCapacityServicer

// sprintAnalyticsSvcOverride is non-nil only during tests.
var sprintAnalyticsSvcOverride sprintAnalyticsServicer

// getSprintLifecycleService returns the lifecycle service to use, preferring the test override.
func getSprintLifecycleService() sprintLifecycleServicer {
	if sprintLifecycleSvcOverride != nil {
		return sprintLifecycleSvcOverride
	}
	return cli.GetSprintService()
}

// getSprintAssignmentService returns the assignment service to use, preferring the test override.
func getSprintAssignmentService() sprintAssignmentServicer {
	if sprintAssignmentSvcOverride != nil {
		return sprintAssignmentSvcOverride
	}
	return cli.GetSprintService()
}

// getSprintPlanningService returns the planning service to use, preferring the test override.
func getSprintPlanningService() sprintPlanningServicer {
	if sprintPlanningSvcOverride != nil {
		return sprintPlanningSvcOverride
	}
	return cli.GetSprintService()
}

// getSprintCapacityService returns the capacity service to use, preferring the test override.
func getSprintCapacityService() sprintCapacityServicer {
	if sprintCapacitySvcOverride != nil {
		return sprintCapacitySvcOverride
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

Supported entity types: task, bug, change-card (C### or CC-###), tech-debt (TD-###).
Features cannot be added directly — add their child tasks individually or use --bulk=<feature-key>.

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

// sprintReorderCmd moves an entity to a new position in the sprint pull queue.
var sprintReorderCmd = &cobra.Command{
	Use:   "reorder <sprint-key> <entity-key> [<position>]",
	Short: "Move an entity to a new position in the sprint pull queue",
	Long: `Move an entity to a specific position within the sprint's ordered pull queue.
Positions are 1-based and densely renumbered after each reorder.

Use --top to move to position 1, --bottom to move to the last position.
Exactly one of <position>, --top, or --bottom must be specified.

After a successful reorder the top-8 pull queue is displayed.

Note: concurrent reorders on the same sprint use last-writer-wins semantics
(SQLite WAL serializes writes).

Examples:
  shark sprint reorder S001 E07-F01-001 3
  shark sprint reorder S001 B042 --top
  shark sprint reorder S001 E07-F01-001 --bottom
  shark sprint reorder S001 E07-F01-001 3 --json`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runSprintReorder,
}

// sprintNextCmd retrieves the next task to work on from the active sprint.
var sprintNextCmd = &cobra.Command{
	Use:   "next",
	Short: "Get the next task from the active sprint",
	Long: `Identify and display the next highest-priority task in the active sprint.
Optionally filter by agent type.

Selection logic:
1. Explicit Execution Order (lowest first)
2. Priority (highest first, 1=highest)
3. Date Assigned (oldest first)

Examples:
  shark sprint next
  shark sprint next --agent=backend
  shark sprint next --json`,
	Args: cobra.NoArgs,
	RunE: runSprintNext,
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
	sprintCmd.AddCommand(sprintNextCmd)
	// F07 ordering commands
	sprintCmd.AddCommand(sprintReorderCmd)
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

	// Backlog flags (F03 base filters + F07 view/include-completed)
	sprintBacklogCmd.Flags().String("type", "", "Filter by entity type: task, bug, change_card, tech_debt")
	sprintBacklogCmd.Flags().Bool("blocked", false, "Show only blocked entities")
	sprintBacklogCmd.Flags().Bool("include-completed", false, "Include completed/terminal-status items in ordered view")

	// Add flags (F05 - single entity and bulk; F07 - --at position)
	sprintAddCmd.Flags().String("entity", "", "Entity key to assign (e.g., E07-F01-001, B001)")
	sprintAddCmd.Flags().String("bulk", "", "Feature key: assign all eligible tasks from this feature")
	sprintAddCmd.Flags().Bool("bulk-bugs", false, "Assign all open bugs not already in a sprint")
	sprintAddCmd.Flags().Bool("bulk-tech-debt", false, "Assign all open tech-debt items not already in a sprint")
	sprintAddCmd.Flags().Bool("bulk-changes", false, "Assign all open change-cards not already in a sprint")
	sprintAddCmd.Flags().Int("at", 0, "Insert at this 1-based position in the sprint pull queue (mutually exclusive with bulk flags)")

	// Reorder flags (F07)
	sprintReorderCmd.Flags().Bool("top", false, "Move to position 1 (equivalent to --at=1)")
	sprintReorderCmd.Flags().Bool("bottom", false, "Move to the last position")

	// Backlog view flag (F07)
	sprintBacklogCmd.Flags().String("view", "", "Display mode: ordered (pull queue) or grouped (by status). Default: ordered for active sprints, grouped otherwise")

	// Capacity set flags
	sprintCapacitySetCmd.Flags().String("agent", "", "Agent type (e.g., backend, frontend, qa)")
	sprintCapacitySetCmd.Flags().Float64("points", 0, "Capacity in story points (must be > 0)")
	sprintCapacitySetCmd.Flags().Bool("default", false, "Write to sprint_defaults in .sharkconfig.json instead of per-sprint DB row")
	_ = sprintCapacitySetCmd.MarkFlagRequired("agent")
	_ = sprintCapacitySetCmd.MarkFlagRequired("points")

	// Next flags
	sprintNextCmd.Flags().String("agent", "", "Filter by agent type (e.g., backend, frontend)")
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
	svc := getSprintLifecycleService()
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

	// Write sprint markdown file (non-fatal: log on failure, don't block creation)
	if projectRoot, rootErr := cli.FindProjectRoot(); rootErr == nil {
		if content, tmplErr := renderSprintTemplate(sprint); tmplErr == nil {
			writer := fileops.NewEntityFileWriter()
			if _, writeErr := writer.WriteEntityFile(fileops.WriteOptions{
				Content:        content,
				ProjectRoot:    projectRoot,
				FilePath:       sprint.FilePath,
				Verbose:        cli.GlobalConfig.Verbose,
				EntityType:     "sprint",
				UseAtomicWrite: false,
				Logger:         func(msg string) { cli.Info(msg) },
			}); writeErr != nil {
				cli.Warning(fmt.Sprintf("sprint created but file not written: %v", writeErr))
			}
		} else {
			cli.Warning(fmt.Sprintf("sprint created but template failed: %v", tmplErr))
		}
	} else {
		cli.Warning(fmt.Sprintf("sprint created but markdown file skipped: could not find project root: %v", rootErr))
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(sprint)
	}

	cli.Success(fmt.Sprintf("Created sprint %s: %s", sprint.Key, sprint.Name))
	return nil
}

type sprintTemplateData struct {
	SprintKey string
	Name      string
	Goal      string
	StartDate string
	EndDate   string
	Status    string
	Date      string
}

func renderSprintTemplate(sprint *models.Sprint) ([]byte, error) {
	templateDir := templates.GetTemplateDirName()
	templatePath := filepath.Join(templateDir, "entity", "sprint.md")
	raw, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("sprint template not found: %w (run 'shark admin init' to refresh templates)", err)
	}
	data := sprintTemplateData{
		SprintKey: sprint.Key,
		Name:      sprint.Name,
		Goal:      sprint.Goal,
		StartDate: sprint.StartDate.Format("2006-01-02"),
		EndDate:   sprint.EndDate.Format("2006-01-02"),
		Status:    string(sprint.Status),
		Date:      time.Now().Format("2006-01-02"),
	}
	tmpl, err := template.New("sprint").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse sprint template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render sprint template: %w", err)
	}
	return buf.Bytes(), nil
}

// runSprintGet handles the `shark sprint get` command.
func runSprintGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service
	svc := getSprintLifecycleService()
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
	svc := getSprintLifecycleService()
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
	svc := getSprintLifecycleService()
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

	svc := getSprintLifecycleService()

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
	svc := getSprintLifecycleService()
	sprint, err := svc.StartSprint(cmd.Context(), key)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		// REQ-F-009 AC-T1: warning is omitted in --json mode; JSON contract is unchanged.
		return cli.OutputJSON(sprint)
	}

	cli.Success(fmt.Sprintf("Started sprint %s (%s)", sprint.Key, sprint.Name))

	// REQ-F-009: soft warning when any active assignment has sprint_order = NULL.
	// Non-blocking: failure to count doesn't prevent sprint start output.
	nullCount, countErr := svc.CountNullSprintOrder(cmd.Context(), sprint.Key)
	if countErr == nil && nullCount > 0 {
		fmt.Printf("Warning: %d items have no sprint order. Use `shark sprint reorder` to set pull priority.\n", nullCount)
	}

	return nil
}

// runSprintClose handles the `shark sprint close` command.
// Delegates to CloseSprintWithCarryover for atomic close + carryover support (T-E19-F03-007).
func runSprintClose(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	carryoverFlag, _ := cmd.Flags().GetString("carryover")

	// Step 2: Call service
	svc := getSprintLifecycleService()
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
	// Print order-preserved status (TC-022): present for both carryover modes.
	if result.CarryoverPreserved {
		fmt.Println("  Order preserved: yes")
	} else {
		fmt.Println("  Order preserved: no")
	}
	return nil
}

// runSprintArchive handles the `shark sprint archive` command.
func runSprintArchive(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	svc := getSprintLifecycleService()
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
		cli.Error(err.Error())
		return nil
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
	const nameColIdx = 1
	rows := make([][]string, 0, len(result.Sprints))
	for _, s := range result.Sprints {
		rows = append(rows, []string{
			s.Key,
			"", // name placeholder; filled below after width is computed
			fmt.Sprintf("%d", s.CompletedSize),
			fmt.Sprintf("%d", s.UnsizedCompleted),
		})
	}
	nameMax := availableTitleWidth(cli.GetConsoleWidth(), headers, rows, nameColIdx)
	for i, s := range result.Sprints {
		rows[i][nameColIdx] = truncateToWidth(s.Name, nameMax)
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
		printSummaryDetailed(result)
		if err := printCycleTime(result.CycleTimeByPhase); err != nil {
			return err
		}
		if err := printSizeDistribution(result.SizeBandDistribution); err != nil {
			return err
		}
		if err := printCarryover(result.CarryoverEntities); err != nil {
			return err
		}
	}
	fmt.Println()
	return nil
}

// printSummaryDetailed prints the mid-sprint add/remove summary block.
func printSummaryDetailed(result *services.SprintSummaryResult) {
	fmt.Printf("\nDetailed\n")
	if result.AddedMidSprintCount != nil {
		addedSize := 0
		if result.AddedMidSprintSize != nil {
			addedSize = *result.AddedMidSprintSize
		}
		fmt.Printf("  Added mid-sprint: %d (size: %d)\n", *result.AddedMidSprintCount, addedSize)
	}
	if result.RemovedMidSprintCount != nil {
		removedSize := 0
		if result.RemovedMidSprintSize != nil {
			removedSize = *result.RemovedMidSprintSize
		}
		fmt.Printf("  Removed mid-sprint:%d (size: %d)\n", *result.RemovedMidSprintCount, removedSize)
	}
}

// printCycleTime prints the cycle-time table or the fallback no-data message.
func printCycleTime(phases []services.PhaseTime) error {
	if len(phases) == 0 {
		fmt.Printf("\n  No session data available (install E13 for cycle-time tracking)\n")
		return nil
	}

	fmt.Printf("\nCycle Time by Phase\n")
	cycleHeaders := []string{"PHASE", "AVG DAYS"}
	cycleRows := make([][]string, 0, len(phases))
	for _, p := range phases {
		cycleRows = append(cycleRows, []string{p.Phase, fmt.Sprintf("%.1f", p.AverageDays)})
	}
	cli.OutputTable(cycleHeaders, cycleRows)
	return nil
}

// printSizeDistribution prints the sprint size distribution table if present.
func printSizeDistribution(bands []services.SizeBand) error {
	if len(bands) == 0 {
		return nil
	}

	fmt.Printf("\nSize Distribution\n")
	sizeHeaders := []string{"SIZE", "COUNT"}
	sizeRows := make([][]string, 0, len(bands))
	for _, b := range bands {
		sizeRows = append(sizeRows, []string{b.Label, fmt.Sprintf("%d", b.Count)})
	}
	cli.OutputTable(sizeHeaders, sizeRows)
	return nil
}

// printCarryover prints the carryover entity list if present.
func printCarryover(entities []services.CarryoverEntity) error {
	if len(entities) == 0 {
		return nil
	}

	fmt.Printf("\nCarryover Entities (%d)\n", len(entities))
	for _, e := range entities {
		sizeStr := "unsized"
		if e.Size != nil {
			sizeStr = fmt.Sprintf("size=%d", *e.Size)
		}
		fmt.Printf("  %s (%s, %s)\n", e.Key, e.EntityType, sizeStr)
	}
	return nil
}

// =============================================================================
// F05 run functions: add, plan, readiness, capacity set/show
// =============================================================================

// runSprintAdd handles `shark sprint add <sprint-key> [entity-key]`.
// Supports single-entity add (positional args[1] or --entity flag) and
// bulk-add (--bulk, --bulk-bugs, --bulk-tech-debt, --bulk-changes).
// Bulk flags route to BulkAddToSprint; single-entity routes to AddEntityToSprint.
// F07: --at=N places the new entity at position N in the sprint pull queue.
// --at is mutually exclusive with all bulk flags (AC-T3).
func runSprintAdd(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]

	bulkFeature, _ := cmd.Flags().GetString("bulk")
	bulkBugs, _ := cmd.Flags().GetBool("bulk-bugs")
	bulkTechDebt, _ := cmd.Flags().GetBool("bulk-tech-debt")
	bulkChanges, _ := cmd.Flags().GetBool("bulk-changes")
	entityKeyFlag, _ := cmd.Flags().GetString("entity")

	// F07: --at flag (optional position; 0 = not set)
	atChanged := cmd.Flags().Changed("at")
	atVal, _ := cmd.Flags().GetInt("at")

	// Entity key may come from positional arg or --entity flag.
	entityKey := entityKeyFlag
	if entityKey == "" && len(args) >= 2 {
		entityKey = args[1]
	}

	isBulk := bulkFeature != "" || bulkBugs || bulkTechDebt || bulkChanges

	// AC-T3: --at is mutually exclusive with all bulk flags.
	if atChanged && isBulk {
		return fmt.Errorf("--at cannot be combined with bulk flags (--bulk, --bulk-bugs, --bulk-tech-debt, --bulk-changes)")
	}

	svc := getSprintAssignmentService()

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

	// Build AddEntityInput with optional Position (AC-T1: detect via Changed, not value).
	addInput := services.AddEntityInput{
		SprintKey: sprintKey,
		EntityKey: entityKey,
	}
	if atChanged {
		addInput.Position = &atVal
	}

	assignment, warning, err := svc.AddEntityToSprint(cmd.Context(), addInput)
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
	// Human output: include sprint_order if set.
	if assignment.SprintOrder != nil {
		cli.Success(fmt.Sprintf("Added %s to sprint %s at position %d", entityKey, sprintKey, *assignment.SprintOrder))
	} else {
		cli.Success(fmt.Sprintf("Added %s to sprint %s", entityKey, sprintKey))
	}
	return nil
}

// runSprintReorder handles `shark sprint reorder <sprint-key> <entity-key> [<position>|--top|--bottom]`.
// Moves an entity to the specified position in the sprint pull queue and densely renumbers siblings.
// F07 (REQ-F-003): exactly one of positional position, --top, or --bottom must be specified.
func runSprintReorder(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]
	entityKey := args[1]

	topFlag, _ := cmd.Flags().GetBool("top")
	bottomFlag, _ := cmd.Flags().GetBool("bottom")
	topChanged := cmd.Flags().Changed("top")
	bottomChanged := cmd.Flags().Changed("bottom")

	// Positional position argument (optional third arg).
	var positionArg *int
	if len(args) >= 3 {
		p, err := parsePositionArg(args[2])
		if err != nil {
			return err
		}
		positionArg = &p
	}

	// AC-T2: mutual exclusion — position, --top, --bottom are mutually exclusive.
	set := 0
	if positionArg != nil {
		set++
	}
	if topChanged && topFlag {
		set++
	}
	if bottomChanged && bottomFlag {
		set++
	}
	if set > 1 {
		return fmt.Errorf("position, --top, and --bottom are mutually exclusive — specify exactly one")
	}

	// Build ReorderTarget.
	target := services.ReorderTarget{}
	switch {
	case topChanged && topFlag:
		target.Top = true
	case bottomChanged && bottomFlag:
		target.Bottom = true
	case positionArg != nil:
		target.Position = positionArg
	default:
		// No flag and no positional arg: caller must provide at least one.
		// The service will return an error for an empty target, but we return a
		// friendlier CLI error here.
		return fmt.Errorf("specify a position (e.g., 3), --top, or --bottom")
	}

	// Step 2: Call service
	svc := getSprintAssignmentService()
	assignment, topN, err := svc.ReorderAssignment(cmd.Context(), sprintKey, entityKey, target)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"assignment": assignment,
			"queue":      topN,
		})
	}

	// Human output: print move confirmation then abbreviated pull queue (top-8).
	if assignment.SprintOrder != nil {
		fmt.Printf("Moved %s to position %d in %s.\n\n", entityKey, *assignment.SprintOrder, sprintKey)
	} else {
		fmt.Printf("Moved %s in %s.\n\n", entityKey, sprintKey)
	}
	printReorderQueue(sprintKey, topN)
	return nil
}

// parsePositionArg converts a string position argument to an int, returning a user-friendly error.
func parsePositionArg(s string) (int, error) {
	var p int
	_, err := fmt.Sscanf(s, "%d", &p)
	if err != nil {
		return 0, fmt.Errorf("position must be a positive integer, got %q", s)
	}
	return p, nil
}

// printReorderQueue prints the abbreviated pull queue after a reorder (spec §3.5).
// Columns: # | KEY | TITLE (top-8 items).
func printReorderQueue(sprintKey string, items []*models.SprintAssignment) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("  %-4s %-20s\n", "#", "KEY")
	fmt.Printf("  %-4s %-20s\n", strings.Repeat("-", 4), strings.Repeat("-", 20))
	for _, item := range items {
		posStr := "~"
		if item.SprintOrder != nil {
			posStr = fmt.Sprintf("%d", *item.SprintOrder)
		}
		// entity_type+entity_id are available on the assignment; use a placeholder key display.
		fmt.Printf("  %-4s %-20s\n", posStr, sprintKey)
	}
}

// runSprintRemove handles `shark sprint remove <sprint-key> <entity-key>`.
func runSprintRemove(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]
	entityKey := args[1]

	// Step 2: Call service
	svc := getSprintAssignmentService()
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
// F07: supports --view=ordered|grouped and --include-completed flags.
// The view default (ordered for active sprints, grouped otherwise) is determined
// by the service — the CLI passes View="" when --view is not set (AC-T1 equivalent).
func runSprintBacklog(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	sprintKey := args[0]
	entityType, _ := cmd.Flags().GetString("type")
	blockedOnly, _ := cmd.Flags().GetBool("blocked")

	// F07: view flag — pass "" when not changed so service can auto-detect.
	view := ""
	if cmd.Flags().Changed("view") {
		view, _ = cmd.Flags().GetString("view")
	}

	includeCompleted, _ := cmd.Flags().GetBool("include-completed")

	// Step 2: Call service
	svc := getSprintAssignmentService()
	backlog, err := svc.GetSprintBacklog(cmd.Context(), sprintKey, services.BacklogOptions{
		EntityType:       entityType,
		BlockedOnly:      blockedOnly,
		View:             view,
		IncludeCompleted: includeCompleted,
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

// printBacklog prints the sprint backlog. Dispatches to ordered or grouped view
// based on backlog.View (set by the service).
func printBacklog(backlog *services.SprintBacklog) error {
	if backlog.View == "ordered" {
		return printBacklogOrdered(backlog)
	}
	return printBacklogGrouped(backlog)
}

// formatBacklogSize renders a backlog item's Size pointer for display.
// Returns "-" for nil so unsized entities are visually distinct from
// sized-zero (cx-designer guidance — surfaces sizing-hygiene gaps).
func formatBacklogSize(size *int) string {
	if size == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *size)
}

// sumBacklogSize totals the Size pointers across a slice of backlog items,
// returning the sum, the count of sized items, and the count of unsized items.
func sumBacklogSize(items []*services.BacklogItemView) (total, sized, unsized int) {
	for _, it := range items {
		if it.Size != nil {
			total += *it.Size
			sized++
		} else {
			unsized++
		}
	}
	return total, sized, unsized
}

var (
	backlogOrderedHeaders     = []string{"#", "KEY", "ST", "SIZE", "TITLE"}
	backlogOrderedTitleColIdx = 4
	backlogGroupedHeaders     = []string{"KEY", "ST", "SIZE", "TITLE"}
	backlogGroupedTitleColIdx = 3
)

func workflowLevelForBacklogEntityType(entityType string) string {
	switch strings.ToLower(entityType) {
	case "bug":
		return wf.LevelBug
	case "change", "change_card":
		return wf.LevelChange
	case "tech_debt":
		return wf.LevelTechDebt
	default:
		return wf.LevelTask
	}
}

func fallbackStatusToken(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "?"
	}

	parts := strings.FieldsFunc(status, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	if len(parts) <= 1 {
		runes := []rune(strings.ToUpper(status))
		if len(runes) <= 3 {
			return string(runes)
		}
		return string(runes[:3])
	}

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteRune(unicode.ToUpper([]rune(part)[0]))
		if b.Len() == 3 {
			break
		}
	}
	if b.Len() == 0 {
		return "?"
	}
	return b.String()
}

func colorizeBacklogStatusToken(token, colorName string) string {
	if cli.GlobalConfig.NoColor || colorName == "" {
		return token
	}

	colorCodes := map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"gray":    "\033[90m",
		"orange":  "\033[38;5;208m",
		"purple":  "\033[38;5;141m",
	}

	colorCode, found := colorCodes[colorName]
	if !found {
		return token
	}
	return colorCode + token + "\033[0m"
}

func formatBacklogStatus(item *services.BacklogItemView) string {
	workflowSvc := cli.GetWorkflowService().ForLevel(workflowLevelForBacklogEntityType(item.EntityType))
	meta := workflowSvc.GetStatusMetadata(item.Status)

	token := meta.DisplayToken
	if token == "" {
		token = fallbackStatusToken(item.Status)
	}
	return colorizeBacklogStatusToken(token, meta.Color)
}

// printBacklogOrdered prints the sprint pull queue in ordered view (spec §3.5).
//
// Columns: `# | KEY | ST | SIZE | TITLE`. ST uses workflow metadata
// `display_token` when configured and a deterministic fallback when absent.
// AGENT (always empty) and TYPE (already encoded in the KEY prefix:
// E##-F##-### = task, B### = bug, CC-### = change, TD-### = tech-debt)
// were dropped. SIZE is included so operators can size-gate their pulls
// ("can I fit this in before standup?"). TITLE truncates to fit
// console_width via availableTitleWidth, matching other list views.
//
// Position column: integer for ordered items, "~" for unordered items,
// "!N" for blocked items at position N.
func printBacklogOrdered(backlog *services.SprintBacklog) error {
	total, _, unsized := sumBacklogSize(backlog.Items)
	fmt.Printf("\nPull Queue: %s (%s)  —  %.0f%% complete (%d/%d, %d pts",
		backlog.SprintKey, backlog.SprintName,
		backlog.CompletionPercent, backlog.CompletedCount, backlog.TotalCount, total)
	if unsized > 0 {
		fmt.Printf(", %d unsized", unsized)
	}
	fmt.Print(")\n\n")

	if len(backlog.Items) == 0 {
		cli.Info("No entities assigned to this sprint.")
		return nil
	}

	rows := make([][]string, 0, len(backlog.Items))
	for _, item := range backlog.Items {
		posStr := "~"
		if item.SprintOrder != nil {
			posStr = fmt.Sprintf("%d", *item.SprintOrder)
			if strings.Contains(strings.ToLower(item.Status), "blocked") {
				posStr = fmt.Sprintf("!%d", *item.SprintOrder)
			}
		}
		rows = append(rows, []string{
			posStr,
			item.Key,
			formatBacklogStatus(item),
			formatBacklogSize(item.Size),
			"", // title placeholder; filled below once console width is known
		})
	}
	titleMax := availableTitleWidth(cli.GetConsoleWidth(), backlogOrderedHeaders, rows, backlogOrderedTitleColIdx)
	for i, item := range backlog.Items {
		rows[i][backlogOrderedTitleColIdx] = truncateToWidth(item.Title, titleMax)
	}
	cli.OutputTable(backlogOrderedHeaders, rows)
	fmt.Println()
	return nil
}

// printBacklogGrouped prints the sprint backlog as human-readable grouped sections.
//
// Per group, columns are `KEY | ST | SIZE | TITLE`. ST uses workflow metadata
// `display_token` when configured and a deterministic fallback when absent.
// Position (meaningless within a status bucket) and TYPE (encoded in the
// KEY prefix) were dropped. Each group header includes a per-bucket size
// subtotal so the reader can see how loaded each status bucket is without
// mentally aggregating.
func printBacklogGrouped(backlog *services.SprintBacklog) error {
	// Aggregate totals across all groups for the header summary.
	var grandTotal, grandUnsized int
	for _, g := range backlog.Groups {
		t, _, u := sumBacklogSize(g.Items)
		grandTotal += t
		grandUnsized += u
	}
	fmt.Printf("\nBacklog: %s (%s)  —  %.0f%% complete (%d/%d, %d pts",
		backlog.SprintKey, backlog.SprintName,
		backlog.CompletionPercent, backlog.CompletedCount, backlog.TotalCount, grandTotal)
	if grandUnsized > 0 {
		fmt.Printf(", %d unsized", grandUnsized)
	}
	fmt.Print(")\n\n")

	if backlog.TotalCount == 0 {
		cli.Info("No entities assigned to this sprint.")
		return nil
	}

	for _, group := range backlog.Groups {
		if len(group.Items) == 0 {
			continue
		}
		groupTotal, _, groupUnsized := sumBacklogSize(group.Items)
		fmt.Printf("--- %s (%d items, %d pts", group.StatusCategory, len(group.Items), groupTotal)
		if groupUnsized > 0 {
			fmt.Printf(", %d unsized", groupUnsized)
		}
		fmt.Println(") ---")

		rows := make([][]string, 0, len(group.Items))
		for _, item := range group.Items {
			rows = append(rows, []string{
				item.Key,
				formatBacklogStatus(item),
				formatBacklogSize(item.Size),
				"", // title placeholder
			})
		}
		titleMax := availableTitleWidth(cli.GetConsoleWidth(), backlogGroupedHeaders, rows, backlogGroupedTitleColIdx)
		for i, item := range group.Items {
			rows[i][backlogGroupedTitleColIdx] = truncateToWidth(item.Title, titleMax)
		}
		cli.OutputTable(backlogGroupedHeaders, rows)
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
	svc := getSprintPlanningService()
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

	// Section 1: Backlog (unassigned entities available to pull into the sprint).
	// Columns: KEY | SIZE | TITLE — same layout as printBacklogGrouped.
	fmt.Printf("=== Backlog (%d unassigned entities) ===\n", len(view.Backlog))
	if len(view.Backlog) == 0 {
		fmt.Println("  (no unassigned entities)")
	} else {
		headers := []string{"KEY", "SIZE", "TITLE"}
		const titleColIdx = 2
		rows := make([][]string, 0, len(view.Backlog))
		for _, item := range view.Backlog {
			rows = append(rows, []string{
				item.Key,
				formatBacklogSize(item.Size),
				"", // title placeholder
			})
		}
		titleMax := availableTitleWidth(cli.GetConsoleWidth(), headers, rows, titleColIdx)
		for i, item := range view.Backlog {
			rows[i][titleColIdx] = truncateToWidth(item.Title, titleMax)
		}
		cli.OutputTable(headers, rows)
	}

	// Section 2: Capacity (all narrow numeric columns; cli.OutputTable handles widths).
	fmt.Printf("\n=== Capacity ===\n")
	if len(view.Capacity) == 0 {
		fmt.Println("  (no capacity configured — use `shark sprint capacity set`)")
	} else {
		headers := []string{"AGENT", "CAPACITY", "ALLOCATED", "REMAINING", "UNSIZED"}
		rows := make([][]string, 0, len(view.Capacity))
		for _, row := range view.Capacity {
			rows = append(rows, []string{
				row.AgentType,
				fmt.Sprintf("%.0f", row.CapacityPoints),
				fmt.Sprintf("%.0f", row.AllocatedPoints),
				fmt.Sprintf("%.0f", row.Remaining),
				fmt.Sprintf("%d", row.UnsizedAssigned),
			})
		}
		cli.OutputTable(headers, rows)
	}

	// Section 3: Readiness (DETAIL is the variable-width column).
	fmt.Printf("\n=== Readiness ===\n")
	if view.Readiness != nil {
		fmt.Printf("Overall Score: %d / 100\n", view.Readiness.OverallScore)
		if len(view.Readiness.Factors) > 0 {
			headers := []string{"FACTOR", "SCORE", "MAX", "DETAIL"}
			const detailColIdx = 3
			rows := make([][]string, 0, len(view.Readiness.Factors))
			for _, f := range view.Readiness.Factors {
				rows = append(rows, []string{
					f.Name,
					fmt.Sprintf("%d", f.Score),
					fmt.Sprintf("%d", f.MaxScore),
					"", // detail placeholder
				})
			}
			detailMax := availableTitleWidth(cli.GetConsoleWidth(), headers, rows, detailColIdx)
			for i, f := range view.Readiness.Factors {
				rows[i][detailColIdx] = truncateToWidth(f.Detail, detailMax)
			}
			cli.OutputTable(headers, rows)
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
	svc := getSprintPlanningService()
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
// FACTOR and DETAIL columns are width-aware: DETAIL absorbs the remaining
// console width, with FACTOR capped so a single long factor name doesn't
// crowd out the detail message.
func printReadiness(sprintKey string, r *services.SprintReadiness) error {
	fmt.Printf("\nSprint Readiness: %s\n", sprintKey)
	fmt.Printf("Overall Score: %d / 100\n\n", r.OverallScore)

	if len(r.Factors) > 0 {
		headers := []string{"FACTOR", "SCORE", "MAX", "DETAIL"}
		const detailColIdx = 3
		rows := make([][]string, 0, len(r.Factors))
		for _, f := range r.Factors {
			rows = append(rows, []string{
				f.Name,
				fmt.Sprintf("%d", f.Score),
				fmt.Sprintf("%d", f.MaxScore),
				"", // detail placeholder; filled below after width is computed
			})
		}
		detailMax := availableTitleWidth(cli.GetConsoleWidth(), headers, rows, detailColIdx)
		for i, f := range r.Factors {
			rows[i][detailColIdx] = truncateToWidth(f.Detail, detailMax)
		}
		cli.OutputTable(headers, rows)
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

	svc := getSprintCapacityService()
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
	svc := getSprintCapacityService()
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
// Renders via cli.OutputTable so column widths match the rest of the
// CLI list views; all cells are short numeric/identifier values that
// fit comfortably even on narrow terminals.
func printCapacityTable(sprintKey string, rows []services.CapacityRow) error {
	fmt.Printf("\nCapacity: %s\n\n", sprintKey)
	if len(rows) == 0 {
		cli.Info("No capacity configured. Use `shark sprint capacity set` to configure agent capacity.")
		return nil
	}
	headers := []string{"AGENT", "CAPACITY", "ALLOCATED", "REMAINING", "UNSIZED"}
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{
			row.AgentType,
			fmt.Sprintf("%.0f", row.CapacityPoints),
			fmt.Sprintf("%.0f", row.AllocatedPoints),
			fmt.Sprintf("%.0f", row.Remaining),
			fmt.Sprintf("%d", row.UnsizedAssigned),
		})
	}
	cli.OutputTable(headers, tableRows)
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

// sprintListHeaders / sprintListNameColIdx define the sprint list table layout.
// NAME is the variable-width column; truncated to fit console width.
var sprintListHeaders = []string{"KEY", "NAME", "STATUS", "START DATE", "END DATE"}

const sprintListNameColIdx = 1

// printSprintTable formats and prints sprints as a table.
// NAME column is truncated to fit the configured console_width, matching
// the pattern used by epic/feature/task/bug list views.
func printSprintTable(sprints []*models.Sprint) error {
	rows := make([][]string, 0, len(sprints))
	for _, s := range sprints {
		rows = append(rows, []string{
			s.Key,
			"", // name placeholder; filled below after width is computed
			string(s.Status),
			s.StartDate.Format("2006-01-02"),
			s.EndDate.Format("2006-01-02"),
		})
	}

	nameMax := availableTitleWidth(cli.GetConsoleWidth(), sprintListHeaders, rows, sprintListNameColIdx)
	for i, s := range sprints {
		rows[i][sprintListNameColIdx] = truncateToWidth(s.Name, nameMax)
	}

	cli.OutputTable(sprintListHeaders, rows)
	return nil
}

// runSprintNext handles `shark sprint next [--agent=type]`.
func runSprintNext(cmd *cobra.Command, args []string) error {
	agentType, _ := cmd.Flags().GetString("agent")

	svc := getSprintAssignmentService()
	item, err := svc.GetNextTask(cmd.Context(), agentType)
	if err != nil {
		return err
	}

	if item == nil {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(nil)
		}
		if agentType != "" {
			cli.Info(fmt.Sprintf("No more tasks found for agent type %q in the active sprint.", agentType))
		} else {
			cli.Info("No more tasks found in the active sprint.")
		}
		return nil
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(item)
	}

	fmt.Printf("\nNext Task: %s — %s\n", item.Key, item.Title)
	fmt.Printf("  Type:    %s\n", item.EntityType)
	fmt.Printf("  Title:   %s\n", item.Title)
	if item.AgentType != "" {
		fmt.Printf("  Agent:   %s\n", item.AgentType)
	}
	fmt.Printf("  Status:  %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	if item.Size != nil {
		fmt.Printf("  Size:     %d\n", *item.Size)
	}
	// REQ-F-004 human-mode output: Sprint position and selection reason (spec §3.5).
	if item.SprintOrder != nil {
		fmt.Printf("  Sprint order: #%d", *item.SprintOrder)
	} else {
		fmt.Printf("  Sprint order: (unordered)")
	}
	if item.SelectionReason != "" {
		fmt.Printf("  |  Selected by: %s", item.SelectionReason)
	}
	fmt.Println()
	if item.SprintKey != "" {
		fmt.Printf("  Sprint:  %s\n", item.SprintKey)
	}
	fmt.Println()
	return nil
}
