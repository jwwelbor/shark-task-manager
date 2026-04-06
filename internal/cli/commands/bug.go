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

// bugServicer defines the interface for bug service operations used by CLI commands.
type bugServicer interface {
	CreateBug(ctx context.Context, input services.CreateBugInput) (*models.Bug, error)
	GetBug(ctx context.Context, key string) (*models.Bug, error)
	ListBugs(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error)
	UpdateBug(ctx context.Context, key string, updates services.BugUpdates) (*models.Bug, error)
	DeleteBug(ctx context.Context, key string) error
	TriageBug(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error)
	GetNextStatusForBug(bug *models.Bug) *services.NextStatusInfo
	GetOrchestratorAction(bug *models.Bug) *config.PopulatedAction
}

// bugSvcOverride is non-nil only during tests.
var bugSvcOverride bugServicer

// getBugService returns the service to use, preferring the test override.
func getBugService() bugServicer {
	if bugSvcOverride != nil {
		return bugSvcOverride
	}
	return cli.GetBugService()
}

// bugCmd is the parent command for all bug operations.
var bugCmd = &cobra.Command{
	Use:     "bug",
	Short:   "Manage bug reports",
	GroupID: "advanced",
	Long: `Bug report management operations for tracking defects and issues.

Bugs are assigned keys in format B### (e.g., B001, B042).

Examples:
  shark bug list                          List all bugs
  shark bug create "Login page crashes"   Create a new bug report
  shark bug get B001                      Get bug details
  shark bug triage B001 --severity=high   Triage a bug
  shark bug delete B001                   Delete a bug`,
}

// bugCreateCmd creates a new bug report.
var bugCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new bug report",
	Long: `Create a new bug report with auto-generated key (B### format).

Examples:
  shark bug create "Login page crashes on submit"
  shark bug create "API timeout" --severity=critical
  shark bug create "Button misaligned" --link=E07-F01
  shark bug create "New bug" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBugCreate,
}

// bugGetCmd retrieves a specific bug by key.
var bugGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get bug details",
	Long: `Display detailed information about a specific bug.

Examples:
  shark bug get B001
  shark bug get B001 --json
  shark bug get B001 --field severity`,
	Args: cobra.ExactArgs(1),
	RunE: runBugGet,
}

// bugListCmd lists bugs with optional filters.
var bugListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bugs",
	Long: `List bugs with optional filtering by status, severity, and linked entity.

By default, terminal-status bugs (resolved, wont_fix, duplicate) are hidden.
Use --all to show all bugs including those in terminal statuses.

Examples:
  shark bug list
  shark bug list --all
  shark bug list --status=reported
  shark bug list --severity=critical
  shark bug list --link=E07-F01
  shark bug list --status=reported --severity=high --json`,
	RunE: runBugList,
}

// bugUpdateCmd updates bug fields.
var bugUpdateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Update a bug",
	Long: `Update bug fields (title, severity).

At least one update flag must be provided.

Examples:
  shark bug update B001 --title="Updated title"
  shark bug update B001 --severity=critical
  shark bug update B001 --title="New" --severity=low --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBugUpdate,
}

// bugDeleteCmd deletes a bug.
var bugDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a bug",
	Long: `Delete a bug and its associated data.

Confirmation is required unless --force is provided.

Examples:
  shark bug delete B001
  shark bug delete B001 --force
  shark bug delete B001 --force --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBugDelete,
}

// bugTriageCmd triages a bug.
var bugTriageCmd = &cobra.Command{
	Use:   "triage <key> --severity=S",
	Short: "Triage a bug",
	Long: `Triage a bug by setting severity, advancing status from 'reported' to 'triaged'.

Examples:
  shark bug triage B001 --severity=high
  shark bug triage B001 --severity=critical --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBugTriage,
}

// Command flag variables
var (
	bugSeverity string
	bugLink     string
	bugTitle    string
	bugStatus   string
	bugForce    bool
	bugFilePath string
)

func init() {
	// Register bug command and subcommands
	cli.RootCmd.AddCommand(bugCmd)
	bugCmd.AddCommand(bugCreateCmd)
	bugCmd.AddCommand(bugGetCmd)
	bugCmd.AddCommand(bugListCmd)
	bugCmd.AddCommand(bugUpdateCmd)
	bugCmd.AddCommand(bugDeleteCmd)
	bugCmd.AddCommand(bugTriageCmd)
	bugCmd.AddCommand(makeNoteCmd("bug"))
	bugCmd.AddCommand(makeNotesCmd("bug"))
	bugCmd.AddCommand(makeContextCmd("bug"))

	// Create flags
	bugCreateCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (critical, high, medium, low)")
	bugCreateCmd.Flags().StringVar(&bugLink, "link", "", "Entity key to link (E07, E07-F01, E07-F01-001)")
	bugCreateCmd.Flags().StringVar(&bugFilePath, "file", "", "Custom file path for bug markdown file")
	bugCreateCmd.Flags().BoolVar(&bugForce, "force", false, "Overwrite existing file at target path")

	// List flags
	bugListCmd.Flags().StringVar(&bugStatus, "status", "", "Filter by status")
	bugListCmd.Flags().StringVar(&bugSeverity, "severity", "", "Filter by severity")
	bugListCmd.Flags().StringVar(&bugLink, "link", "", "Filter by linked entity key")
	bugListCmd.Flags().Bool("all", false, "Show all bugs including terminal statuses (resolved, wont_fix, duplicate)")

	// Update flags
	bugUpdateCmd.Flags().StringVar(&bugTitle, "title", "", "New title")
	bugUpdateCmd.Flags().StringVar(&bugSeverity, "severity", "", "New severity")
	bugUpdateCmd.Flags().String("file", "", "New file path")
	bugUpdateCmd.Flags().String("filename", "", "Alias for --file")
	bugUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = bugUpdateCmd.Flags().MarkHidden("filename")
	_ = bugUpdateCmd.Flags().MarkHidden("path")

	// Delete flags
	bugDeleteCmd.Flags().BoolVar(&bugForce, "force", false, "Skip confirmation prompt")

	// Triage flags
	bugTriageCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (required)")
	_ = bugTriageCmd.MarkFlagRequired("severity")

}

// runBugCreate handles the `shark bug create` command.
func runBugCreate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	severity, _ := cmd.Flags().GetString("severity")
	link, _ := cmd.Flags().GetString("link")
	filePath, _ := cmd.Flags().GetString("file")
	force, _ := cmd.Flags().GetBool("force")

	input := services.CreateBugInput{
		Title:    args[0],
		Severity: models.BugSeverity(severity),
		Force:    force,
	}

	if filePath != "" {
		input.FilePath = &filePath
	}

	// Parse --link into linked entity type and key
	if link != "" {
		entityType, entityKey := parseBugLinkFlag(link)
		input.LinkedEntityType = entityType
		input.LinkedEntityKey = entityKey
	}

	// Step 2: Call service
	svc := getBugService()
	bug, err := svc.CreateBug(cmd.Context(), input)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}
	cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
	if fp := bug.GetFilePath(); fp != "" {
		cli.Info(fmt.Sprintf("File: %s", fp))
	}
	return nil
}

// runBugGet handles the `shark bug get` command.
func runBugGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service
	svc := getBugService()
	bug, err := svc.GetBug(ctx, key)
	if err != nil {
		return err
	}

	// Gather enrichment data (best-effort)
	orchestratorAction := svc.GetOrchestratorAction(bug)
	info := svc.GetNextStatusForBug(bug)
	validTransitions := info.TargetStatuses()

	var notes []*models.EntityNote
	if noteSvc, nErr := cli.GetNoteService(ctx); nErr == nil && noteSvc != nil {
		notes, _ = noteSvc.ListNotes(ctx, models.EntityTypeBug, key, nil)
	}

	var contextData *models.ContextData
	if ctxSvc := cli.GetContextService(); ctxSvc != nil {
		contextData, _ = ctxSvc.GetContext(ctx, models.EntityTypeBug, key)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		result, err := buildEnrichedJSON(bug, orchestratorAction, validTransitions)
		if err != nil {
			return err
		}
		if notes != nil {
			result["notes"] = notes
		}
		if contextData != nil {
			result["context_data"] = contextData
		}
		return cli.OutputJSON(result)
	}

	RenderEntity(EntityDisplayOptions{
		EntityType:         "bug",
		Key:                bug.Key,
		Status:             string(bug.Status),
		BasicInfo:          buildBugBasicInfo(bug),
		ValidTransitions:   validTransitions,
		OrchestratorAction: orchestratorAction,
		Notes:              notes,
		ContextData:        contextData,
	})
	return nil
}

// runBugList handles the `shark bug list` command.
func runBugList(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	statusStr, _ := cmd.Flags().GetString("status")
	severityStr, _ := cmd.Flags().GetString("severity")
	linkStr, _ := cmd.Flags().GetString("link")
	allFlag, _ := cmd.Flags().GetBool("all")

	filters := services.BugFilters{
		ShowAll: allFlag || statusStr != "",
	}
	if statusStr != "" {
		s := models.BugStatus(statusStr)
		filters.Status = &s
	}
	if severityStr != "" {
		sv := models.BugSeverity(severityStr)
		filters.Severity = &sv
	}
	if linkStr != "" {
		filters.LinkedEntityKey = &linkStr
	}

	// Step 2: Call service
	svc := getBugService()
	bugs, err := svc.ListBugs(cmd.Context(), filters)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bugs)
	}

	if len(bugs) == 0 {
		cli.Info("No bugs found")
		return nil
	}
	return printBugTable(bugs)
}

// runBugUpdate handles the `shark bug update` command.
func runBugUpdate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	updates := services.BugUpdates{}
	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		updates.Title = &title
	}
	if cmd.Flags().Changed("severity") {
		severity, _ := cmd.Flags().GetString("severity")
		sv := models.BugSeverity(severity)
		updates.Severity = &sv
	}

	if v := getFileFlagValue(cmd); v != "" {
		updates.FilePath = &v
	}

	if updates.Title == nil && updates.Severity == nil && updates.FilePath == nil {
		return fmt.Errorf("at least one update flag is required (--title, --severity, or --file)")
	}

	// Step 2: Call service
	svc := getBugService()
	bug, err := svc.UpdateBug(cmd.Context(), key, updates)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}
	cli.Success(fmt.Sprintf("Updated bug %s", bug.Key))
	return nil
}

// runBugDelete handles the `shark bug delete` command.
func runBugDelete(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	force, _ := cmd.Flags().GetBool("force")

	svc := getBugService()

	// Confirm deletion unless --force
	if !force {
		bug, err := svc.GetBug(cmd.Context(), key)
		if err != nil {
			return err
		}
		if !confirmBugDelete(bug) {
			cli.Info("Delete cancelled")
			return nil
		}
	}

	// Step 2: Call service
	if err := svc.DeleteBug(cmd.Context(), key); err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": key})
	}
	cli.Success(fmt.Sprintf("Deleted bug %s", key))
	return nil
}

// runBugTriage handles the `shark bug triage` command.
func runBugTriage(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	severity, _ := cmd.Flags().GetString("severity")

	input := services.TriageBugInput{
		Severity: severity,
	}

	// Step 2: Call service
	svc := getBugService()
	bug, err := svc.TriageBug(cmd.Context(), key, input)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}
	cli.Success(fmt.Sprintf("Triaged bug %s (severity: %s, status: %s)", bug.Key, bug.Severity, bug.Status))
	return nil
}

// --- Helper functions ---

// parseBugLinkFlag parses a --link flag value into entity type and entity key.
// It handles epic keys (E07), feature keys (E07-F01), and task keys (E07-F01-001).
func parseBugLinkFlag(link string) (entityType, entityKey string) {
	parts := strings.Split(link, "-")
	switch {
	case len(parts) >= 3 && strings.HasPrefix(parts[0], "E") && strings.HasPrefix(parts[1], "F"):
		return "task", link
	case len(parts) >= 2 && strings.HasPrefix(parts[0], "E") && strings.HasPrefix(parts[1], "F"):
		return "feature", link
	default:
		return "epic", link
	}
}

// buildBugBasicInfo assembles the key-value info table for bug display.
func buildBugBasicInfo(bug *models.Bug) [][]string {
	var info [][]string

	info = append(info, []string{"Title", bug.Title})
	info = append(info, []string{"Status", string(bug.Status)})
	info = append(info, []string{"Severity", string(bug.Severity)})

	if bug.LinkedEntityKey != nil && bug.LinkedEntityType != nil {
		info = append(info, []string{"Linked To", fmt.Sprintf("%s (%s)", *bug.LinkedEntityKey, *bug.LinkedEntityType)})
	}
	if bug.FilePath != nil && *bug.FilePath != "" {
		info = append(info, []string{"File", *bug.FilePath})
	}
	if bug.Description != nil && *bug.Description != "" {
		info = append(info, []string{"Description", *bug.Description})
	}
	info = append(info, []string{"Created", bug.CreatedAt.Format(time.RFC3339)})
	info = append(info, []string{"Updated", bug.UpdatedAt.Format(time.RFC3339)})

	return info
}

// printBugTable renders a table for bug list output.
func printBugTable(bugs []*models.Bug) error {
	headers := []string{"KEY", "TITLE", "STATUS", "SEVERITY", "LINKED TO"}
	rows := make([][]string, 0, len(bugs))
	for _, b := range bugs {
		linkedTo := ""
		if b.LinkedEntityKey != nil {
			linkedTo = *b.LinkedEntityKey
		}
		rows = append(rows, []string{
			b.Key,
			truncateBugString(b.Title, 45),
			string(b.Status),
			string(b.Severity),
			linkedTo,
		})
	}
	cli.OutputTable(headers, rows)
	return nil
}

// confirmBugDelete prompts for confirmation before deleting a bug.
func confirmBugDelete(bug *models.Bug) bool {
	fmt.Printf("Delete bug %s: %s? [y/N] ", bug.Key, bug.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}

// truncateBugString truncates a string to maxLen characters, appending "..." if truncated.
func truncateBugString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
