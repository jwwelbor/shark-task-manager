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

// sprintSvcOverride is non-nil only during tests.
var sprintSvcOverride sprintServicer

// getSprintService returns the service to use, preferring the test override.
func getSprintService() sprintServicer {
	if sprintSvcOverride != nil {
		return sprintSvcOverride
	}
	return cli.GetSprintService()
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
