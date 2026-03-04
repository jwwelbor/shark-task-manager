package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

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

Examples:
  shark bug list
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
	Use:   "triage <key> --severity=S [--assign=AGENT]",
	Short: "Triage a bug",
	Long: `Triage a bug by setting severity and optionally assigning an agent.
Advances bug status from 'reported' to 'triaged'.

Examples:
  shark bug triage B001 --severity=high
  shark bug triage B001 --severity=medium --assign=developer
  shark bug triage B001 --severity=critical --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBugTriage,
}

// bugNoteCmd is the parent for note subcommands.
var bugNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage notes on a bug",
	Long:  `Add and manage notes attached to a bug.`,
}

// bugNoteAddCmd adds a note to a bug.
var bugNoteAddCmd = &cobra.Command{
	Use:   "add <key> <content>",
	Short: "Add a note to a bug",
	Long: `Add a note to a bug with a specified type.

Note types: comment, decision, progress, blocker, reference, implementation, future

Examples:
  shark bug note add B001 --type=comment "Reproduced on Safari 17.2"
  shark bug note add B001 --type=decision "Root cause is race condition"`,
	Args: cobra.ExactArgs(2),
	RunE: runBugNoteAdd,
}

// bugNotesCmd lists notes for a bug.
var bugNotesCmd = &cobra.Command{
	Use:   "notes <key>",
	Short: "List notes for a bug",
	Long: `Display all notes for a specific bug.

Examples:
  shark bug notes B001
  shark bug notes B001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBugNotes,
}

// bugContextCmd is the parent for context subcommands.
var bugContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage context fields on a bug",
	Long:  `Get, set, and clear context fields on a bug.`,
}

// bugContextSetCmd sets a context field on a bug.
var bugContextSetCmd = &cobra.Command{
	Use:   "set <key> --field F --value V",
	Short: "Set a context field on a bug",
	Args:  cobra.ExactArgs(1),
	RunE:  runBugContextSet,
}

// bugContextGetCmd retrieves context for a bug.
var bugContextGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get context for a bug",
	Args:  cobra.ExactArgs(1),
	RunE:  runBugContextGet,
}

// bugContextClearCmd clears a context field on a bug.
var bugContextClearCmd = &cobra.Command{
	Use:   "clear <key> --field F",
	Short: "Clear a context field on a bug",
	Args:  cobra.ExactArgs(1),
	RunE:  runBugContextClear,
}

// Command flag variables
var (
	bugSeverity string
	bugLink     string
	bugTitle    string
	bugAssign   string
	bugStatus   string
	bugForce    bool
	bugField    string
	bugValue    string
	bugNoteType string
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
	bugCmd.AddCommand(bugNoteCmd)
	bugCmd.AddCommand(bugNotesCmd)
	bugCmd.AddCommand(bugContextCmd)

	// Note subcommands
	bugNoteCmd.AddCommand(bugNoteAddCmd)

	// Context subcommands
	bugContextCmd.AddCommand(bugContextSetCmd)
	bugContextCmd.AddCommand(bugContextGetCmd)
	bugContextCmd.AddCommand(bugContextClearCmd)

	// Create flags
	bugCreateCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (critical, high, medium, low)")
	bugCreateCmd.Flags().StringVar(&bugLink, "link", "", "Entity key to link (E07, E07-F01, E07-F01-001)")

	// List flags
	bugListCmd.Flags().StringVar(&bugStatus, "status", "", "Filter by status")
	bugListCmd.Flags().StringVar(&bugSeverity, "severity", "", "Filter by severity")
	bugListCmd.Flags().StringVar(&bugLink, "link", "", "Filter by linked entity key")

	// Update flags
	bugUpdateCmd.Flags().StringVar(&bugTitle, "title", "", "New title")
	bugUpdateCmd.Flags().StringVar(&bugSeverity, "severity", "", "New severity")

	// Delete flags
	bugDeleteCmd.Flags().BoolVar(&bugForce, "force", false, "Skip confirmation prompt")

	// Triage flags
	bugTriageCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (required)")
	_ = bugTriageCmd.MarkFlagRequired("severity")
	bugTriageCmd.Flags().StringVar(&bugAssign, "assign", "", "Agent to assign")

	// Note add flags
	bugNoteAddCmd.Flags().StringVar(&bugNoteType, "type", "", "Note type (comment, decision, progress, blocker, reference, implementation, future)")
	_ = bugNoteAddCmd.MarkFlagRequired("type")

	// Context set flags
	bugContextSetCmd.Flags().StringVar(&bugField, "field", "", "Context field name")
	bugContextSetCmd.Flags().StringVar(&bugValue, "value", "", "Context field value")
	_ = bugContextSetCmd.MarkFlagRequired("field")
	_ = bugContextSetCmd.MarkFlagRequired("value")

	// Context clear flags
	bugContextClearCmd.Flags().StringVar(&bugField, "field", "", "Context field to clear")
	_ = bugContextClearCmd.MarkFlagRequired("field")
}

// runBugCreate handles the `shark bug create` command.
func runBugCreate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	severity, _ := cmd.Flags().GetString("severity")
	link, _ := cmd.Flags().GetString("link")

	input := services.CreateBugInput{
		Title:    args[0],
		Severity: models.BugSeverity(severity),
	}

	// Parse --link into linked entity type and key
	if link != "" {
		entityType, entityKey := parseBugLinkFlag(link)
		input.LinkedEntityType = entityType
		input.LinkedEntityKey = entityKey
	}

	// Step 2: Call service
	svc := cli.GetBugService()
	bug, err := svc.CreateBug(cmd.Context(), input)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}
	cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
	return nil
}

// runBugGet handles the `shark bug get` command.
func runBugGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	svc := cli.GetBugService()
	bug, err := svc.GetBug(cmd.Context(), key)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}
	return printBugDetail(bug)
}

// runBugList handles the `shark bug list` command.
func runBugList(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	statusStr, _ := cmd.Flags().GetString("status")
	severityStr, _ := cmd.Flags().GetString("severity")

	filters := services.BugFilters{}
	if statusStr != "" {
		s := models.BugStatus(statusStr)
		filters.Status = &s
	}
	if severityStr != "" {
		sv := models.BugSeverity(severityStr)
		filters.Severity = &sv
	}

	// Step 2: Call service
	svc := cli.GetBugService()
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

	if updates.Title == nil && updates.Severity == nil {
		return fmt.Errorf("at least one update flag is required (--title or --severity)")
	}

	// Step 2: Call service
	svc := cli.GetBugService()
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

	svc := cli.GetBugService()

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
	if cmd.Flags().Changed("assign") {
		assign, _ := cmd.Flags().GetString("assign")
		input.Assign = &assign
	}

	// Step 2: Call service
	svc := cli.GetBugService()
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

// runBugNoteAdd handles the `shark bug note add` command.
func runBugNoteAdd(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	content := args[1]
	noteType, _ := cmd.Flags().GetString("type")

	// Step 2: Call service
	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get note service: %w", err)
	}

	if _, err := noteSvc.AddNote(cmd.Context(), models.EntityTypeBug, key, noteType, content, ""); err != nil {
		return fmt.Errorf("failed to add note to bug %s: %w", key, err)
	}

	// Step 3: Format output
	cli.Success(fmt.Sprintf("Note added to bug %s", key))
	return nil
}

// runBugNotes handles the `shark bug notes` command.
func runBugNotes(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get note service: %w", err)
	}

	notes, err := noteSvc.ListNotes(cmd.Context(), models.EntityTypeBug, key, nil)
	if err != nil {
		return fmt.Errorf("failed to list notes for bug %s: %w", key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(notes)
	}

	if len(notes) == 0 {
		fmt.Printf("No notes found for bug %s\n", key)
		return nil
	}

	fmt.Printf("Notes for bug %s\n\n", key)
	for _, n := range notes {
		creator := ""
		if n.CreatedBy != nil {
			creator = " (" + *n.CreatedBy + ")"
		}
		fmt.Printf("[%s] %s%s\n", strings.ToUpper(string(n.NoteType)), n.CreatedAt.Format("2006-01-02 15:04"), creator)
		fmt.Printf("  %s\n\n", n.Content)
	}
	return nil
}

// runBugContextSet handles the `shark bug context set` command.
func runBugContextSet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	field, _ := cmd.Flags().GetString("field")
	value, _ := cmd.Flags().GetString("value")

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeBug, key, field, value); err != nil {
		return fmt.Errorf("failed to set context field for bug %s: %w", key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_type": "bug",
			"entity_key":  key,
			"field":       field,
			"success":     true,
		})
	}
	cli.Success(fmt.Sprintf("Updated context field '%s' for bug %s", field, key))
	return nil
}

// runBugContextGet handles the `shark bug context get` command.
func runBugContextGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	contextData, err := ctxSvc.GetContext(cmd.Context(), models.EntityTypeBug, key)
	if err != nil {
		return fmt.Errorf("failed to get context for bug %s: %w", key, err)
	}

	if contextData == nil {
		contextData = &models.ContextData{}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_type":  "bug",
			"entity_key":   key,
			"context_data": contextData,
		})
	}

	fmt.Printf("Context for bug %s\n\n", key)
	printContextData(contextData)
	return nil
}

// runBugContextClear handles the `shark bug context clear` command.
func runBugContextClear(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	field, _ := cmd.Flags().GetString("field")

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	// Clear the specific field (or all context if field is empty)
	if field != "" {
		// Set field to empty to clear it
		if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeBug, key, field, ""); err != nil {
			return fmt.Errorf("failed to clear context field for bug %s: %w", key, err)
		}
	} else {
		if err := ctxSvc.ClearContext(cmd.Context(), models.EntityTypeBug, key); err != nil {
			return fmt.Errorf("failed to clear context for bug %s: %w", key, err)
		}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_type": "bug",
			"entity_key":  key,
			"success":     true,
		})
	}
	if field != "" {
		cli.Success(fmt.Sprintf("Cleared context field '%s' for bug %s", field, key))
	} else {
		cli.Success(fmt.Sprintf("Cleared context data for bug %s", key))
	}
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

// printBugDetail renders a formatted bug detail view for non-JSON output.
func printBugDetail(bug *models.Bug) error {
	fmt.Printf("Bug %s\n", bug.Key)
	fmt.Printf("  Title:     %s\n", bug.Title)
	fmt.Printf("  Status:    %s\n", bug.Status)
	fmt.Printf("  Severity:  %s\n", bug.Severity)
	if bug.LinkedEntityKey != nil && bug.LinkedEntityType != nil {
		fmt.Printf("  Linked To: %s (%s)\n", *bug.LinkedEntityKey, *bug.LinkedEntityType)
	}
	if bug.Description != nil && *bug.Description != "" {
		fmt.Printf("  Desc:      %s\n", *bug.Description)
	}
	fmt.Printf("  Created:   %s\n", bug.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Updated:   %s\n", bug.UpdatedAt.Format(time.RFC3339))
	return nil
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
