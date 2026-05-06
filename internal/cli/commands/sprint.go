package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
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
	cli.Info(fmt.Sprintf("  Completed: %d  Carried over: %d  Dropped: %d",
		result.CompletedCount, result.CarriedOverCount, result.DroppedCount))
	if result.NextSprintKey != "" {
		cli.Info(fmt.Sprintf("  Incomplete entities moved to: %s", result.NextSprintKey))
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
	fmt.Printf("%-10s %-30s %10s %10s\n", "KEY", "NAME", "COMPLETED", "UNSIZED")
	fmt.Printf("%-10s %-30s %10s %10s\n", strings.Repeat("-", 10), strings.Repeat("-", 30), strings.Repeat("-", 10), strings.Repeat("-", 10))

	for _, s := range result.Sprints {
		name := s.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		fmt.Printf("%-10s %-30s %10d %10d\n", s.Key, name, s.CompletedSize, s.UnsizedCompleted)
	}

	fmt.Printf("%-10s %-30s %10s %10s\n", strings.Repeat("-", 10), strings.Repeat("-", 30), strings.Repeat("-", 10), strings.Repeat("-", 10))
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
	fmt.Printf("%-12s %10s %10s %8s\n", "DAY", "IDEAL", "ACTUAL", "UNSIZED")
	fmt.Printf("%-12s %10s %10s %8s\n", strings.Repeat("-", 12), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8))

	for _, dp := range result.DataPoints {
		dayStr := dp.Date.Format("2006-01-02")
		idealStr := fmt.Sprintf("%.1f", dp.IdealRemaining)
		var actualStr string
		if dp.ActualRemaining != nil {
			actualStr = fmt.Sprintf("%.1f", *dp.ActualRemaining)
		} else {
			actualStr = "—" // em-dash U+2014 for future days (AC-B-6 allows em-dash)
		}
		fmt.Printf("%-12s %10s %10s %8d\n", dayStr, idealStr, actualStr, dp.UnsizedRemaining)
	}
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
			for _, p := range result.CycleTimeByPhase {
				fmt.Printf("  %-20s %.1f days\n", p.Phase, p.AverageDays)
			}
		} else {
			fmt.Printf("\n  No session data available (install E13 for cycle-time tracking)\n")
		}

		if len(result.SizeBandDistribution) > 0 {
			fmt.Printf("\nSize Distribution\n")
			for _, b := range result.SizeBandDistribution {
				fmt.Printf("  %3s: %d\n", b.Label, b.Count)
			}
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
