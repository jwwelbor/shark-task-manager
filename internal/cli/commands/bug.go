package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// bugServicer is the interface for bug service operations used by CLI commands.
type bugServicer interface {
	CreateBug(ctx context.Context, input services.CreateBugInput) (*models.Bug, error)
	GetBug(ctx context.Context, key string) (*models.Bug, error)
	ListBugs(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error)
	UpdateBug(ctx context.Context, key string, updates services.BugUpdates) (*models.Bug, error)
	DeleteBug(ctx context.Context, key string) error
	TriageBug(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error)
	AdvanceBugStatus(ctx context.Context, key string) (*models.Bug, error)
	SetBugStatus(ctx context.Context, key string, status string, force bool) (*models.Bug, error)
}

// Flag variables for bug commands.
var (
	bugCreateSeverity    string
	bugCreateDescription string
	bugCreateLinkType    string
	bugCreateLinkKey     string

	bugListStatus   string
	bugListSeverity string

	bugUpdateTitle       string
	bugUpdateDescription string
	bugUpdateSeverity    string
	bugUpdateLinkType    string
	bugUpdateLinkKey     string

	bugDeleteForce bool

	bugTriageSeverity string
	bugTriageAssign   string
	bugTriageLinkType string
	bugTriageLinkKey  string

	bugNoteType      string
	bugNoteCreatedBy string
)

// bugCmd is the root command for all bug subcommands.
var bugCmd = &cobra.Command{
	Use:     "bug",
	Short:   "Manage bugs",
	Long:    "Create, view, update, and triage bugs in your project.",
	GroupID: "advanced",
}

// bugCreateCmd creates a new bug.
var bugCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := cli.GetBugService()
		return runBugCreateWithService(cmd.Context(), args, svc, cmd.OutOrStdout())
	},
}

// bugGetCmd retrieves a bug by key.
var bugGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a bug by key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := cli.GetBugService()
		return runBugGetWithService(cmd.Context(), args, svc, cmd.OutOrStdout(), cli.GlobalConfig.JSON)
	},
}

// bugListCmd lists bugs with optional filters.
var bugListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bugs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := cli.GetBugService()
		return runBugListWithService(cmd.Context(), svc, cmd.OutOrStdout(), cli.GlobalConfig.JSON)
	},
}

// bugUpdateCmd updates an existing bug.
var bugUpdateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Update a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := cli.GetBugService()
		return runBugUpdateWithService(cmd.Context(), args, svc, cmd.OutOrStdout())
	},
}

// bugDeleteCmd deletes a bug.
var bugDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := cli.GetBugService()
		return runBugDeleteWithService(cmd.Context(), args, svc, cmd.OutOrStdout())
	},
}

// bugTriageCmd triages a bug (sets severity, assigns, links).
var bugTriageCmd = &cobra.Command{
	Use:   "triage <key>",
	Short: "Triage a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := cli.GetBugService()
		return runBugTriageWithService(cmd.Context(), args, svc, cmd.OutOrStdout())
	},
}

// bugNoteCmd is the parent command for note operations.
var bugNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage bug notes",
}

// bugNoteAddCmd adds a note to a bug.
var bugNoteAddCmd = &cobra.Command{
	Use:   "add <key> <content>",
	Short: "Add a note to a bug",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		content := args[1]

		noteSvc, err := cli.GetNoteService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get note service: %w", err)
		}

		note, err := noteSvc.AddNote(cmd.Context(), models.EntityTypeBug, key, bugNoteType, content, bugNoteCreatedBy)
		if err != nil {
			return fmt.Errorf("failed to add note: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(note)
		}
		cli.Success(fmt.Sprintf("Added %s note to bug %s", bugNoteType, key))
		return nil
	},
}

// bugNotesCmd lists notes for a bug.
var bugNotesCmd = &cobra.Command{
	Use:   "notes <key>",
	Short: "List notes for a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		noteSvc, err := cli.GetNoteService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get note service: %w", err)
		}

		notes, err := noteSvc.ListNotes(cmd.Context(), models.EntityTypeBug, key, nil)
		if err != nil {
			return fmt.Errorf("failed to list notes: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(notes)
		}

		if len(notes) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No notes for bug %s\n", key)
			return nil
		}

		for _, n := range notes {
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", n.CreatedAt.Format("2006-01-02 15:04"), strings.ToUpper(string(n.NoteType)), n.Content)
		}
		return nil
	},
}

// bugContextCmd is the parent command for context operations.
var bugContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage bug context fields",
}

// bugContextSetCmd sets a context field on a bug.
var bugContextSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set a context field on a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		field, _ := cmd.Flags().GetString("field")
		value, _ := cmd.Flags().GetString("value")

		ctxSvc, err := cli.GetContextService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get context service: %w", err)
		}

		if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeBug, key, field, value); err != nil {
			return fmt.Errorf("failed to set context field: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"key": key, "field": field, "value": value})
		}
		cli.Success(fmt.Sprintf("Set context field %q on bug %s", field, key))
		return nil
	},
}

// bugContextGetCmd gets context data from a bug.
var bugContextGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get context data from a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		ctxSvc, err := cli.GetContextService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get context service: %w", err)
		}

		contextData, err := ctxSvc.GetContext(cmd.Context(), models.EntityTypeBug, key)
		if err != nil {
			return fmt.Errorf("failed to get context data: %w", err)
		}

		return cli.OutputJSON(contextData)
	},
}

// bugContextClearCmd clears context data from a bug.
var bugContextClearCmd = &cobra.Command{
	Use:   "clear <key>",
	Short: "Clear context data from a bug",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		ctxSvc, err := cli.GetContextService(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get context service: %w", err)
		}

		if err := ctxSvc.ClearContext(cmd.Context(), models.EntityTypeBug, key); err != nil {
			return fmt.Errorf("failed to clear context: %w", err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"key": key, "status": "cleared"})
		}
		cli.Success(fmt.Sprintf("Cleared context for bug %s", key))
		return nil
	},
}

// init registers bug commands with the root command.
func init() {
	// CRUD flags
	bugCreateCmd.Flags().StringVar(&bugCreateSeverity, "severity", "medium", "Bug severity (critical, high, medium, low)")
	bugCreateCmd.Flags().StringVar(&bugCreateDescription, "description", "", "Bug description")
	bugCreateCmd.Flags().StringVar(&bugCreateLinkType, "link-type", "", "Entity type to link (epic, feature, task)")
	bugCreateCmd.Flags().StringVar(&bugCreateLinkKey, "link-key", "", "Entity key to link")

	bugListCmd.Flags().StringVar(&bugListStatus, "status", "", "Filter by status")
	bugListCmd.Flags().StringVar(&bugListSeverity, "severity", "", "Filter by severity (critical, high, medium, low)")

	bugUpdateCmd.Flags().StringVar(&bugUpdateTitle, "title", "", "New title")
	bugUpdateCmd.Flags().StringVar(&bugUpdateDescription, "description", "", "New description")
	bugUpdateCmd.Flags().StringVar(&bugUpdateSeverity, "severity", "", "New severity (critical, high, medium, low)")
	bugUpdateCmd.Flags().StringVar(&bugUpdateLinkType, "link-type", "", "Entity type to link (epic, feature, task)")
	bugUpdateCmd.Flags().StringVar(&bugUpdateLinkKey, "link-key", "", "Entity key to link")

	bugDeleteCmd.Flags().BoolVar(&bugDeleteForce, "force", false, "Force delete without confirmation")

	// Triage flags
	bugTriageCmd.Flags().StringVar(&bugTriageSeverity, "severity", "", "Set severity (critical, high, medium, low)")
	bugTriageCmd.Flags().StringVar(&bugTriageAssign, "assign", "", "Assign to user")
	bugTriageCmd.Flags().StringVar(&bugTriageLinkType, "link-type", "", "Link to entity type (epic, feature, task)")
	bugTriageCmd.Flags().StringVar(&bugTriageLinkKey, "link-key", "", "Link to entity key")

	// Note flags
	bugNoteAddCmd.Flags().StringVar(&bugNoteType, "type", "progress", "Note type (progress, decision, solution, question, implementation)")
	bugNoteAddCmd.Flags().StringVar(&bugNoteCreatedBy, "created-by", "", "Note author")

	// Context flags
	bugContextSetCmd.Flags().String("field", "", "Context field name")
	bugContextSetCmd.Flags().String("value", "", "Context field value")

	// Register note subcommands
	bugNoteCmd.AddCommand(bugNoteAddCmd)

	// Register context subcommands
	bugContextCmd.AddCommand(bugContextSetCmd, bugContextGetCmd, bugContextClearCmd)

	// Register all subcommands under bugCmd
	bugCmd.AddCommand(
		bugCreateCmd,
		bugGetCmd,
		bugListCmd,
		bugUpdateCmd,
		bugDeleteCmd,
		bugTriageCmd,
		bugNoteCmd,
		bugNotesCmd,
		bugContextCmd,
	)

	// Register bugCmd with root
	cli.RootCmd.AddCommand(bugCmd)
}

// runBugCreateWithService is the testable handler for bug create.
func runBugCreateWithService(ctx context.Context, args []string, svc bugServicer, w io.Writer) error {
	title := args[0]

	input := services.CreateBugInput{
		Title:       title,
		Description: bugCreateDescription,
		Severity:    models.BugSeverity(bugCreateSeverity),
	}
	if bugCreateLinkType != "" {
		input.LinkedEntityType = bugCreateLinkType
	}
	if bugCreateLinkKey != "" {
		input.LinkedEntityKey = bugCreateLinkKey
	}

	bug, err := svc.CreateBug(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bug: %w", err)
	}

	if cli.GlobalConfig.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(bug)
	}

	fmt.Fprintf(w, "Created bug %s: %s\n", bug.Key, bug.Title)
	return nil
}

// runBugGetWithService is the testable handler for bug get.
func runBugGetWithService(ctx context.Context, args []string, svc bugServicer, w io.Writer, jsonMode bool) error {
	key := args[0]

	bug, err := svc.GetBug(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(bug)
	}

	fmt.Fprintf(w, "Bug: %s\n", bug.Key)
	fmt.Fprintf(w, "Title: %s\n", bug.Title)
	fmt.Fprintf(w, "Status: %s\n", bug.Status)
	fmt.Fprintf(w, "Severity: %s\n", bug.Severity)
	if bug.Description != nil {
		fmt.Fprintf(w, "Description: %s\n", *bug.Description)
	}
	if bug.LinkedEntityKey != nil {
		fmt.Fprintf(w, "Linked: %s/%s\n", *bug.LinkedEntityType, *bug.LinkedEntityKey)
	}
	return nil
}

// runBugListWithService is the testable handler for bug list.
func runBugListWithService(ctx context.Context, svc bugServicer, w io.Writer, jsonMode bool) error {
	filters := services.BugFilters{}
	if bugListStatus != "" {
		status := models.BugStatus(bugListStatus)
		filters.Status = &status
	}
	if bugListSeverity != "" {
		severity := models.BugSeverity(bugListSeverity)
		filters.Severity = &severity
	}

	bugs, err := svc.ListBugs(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to list bugs: %w", err)
	}

	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(bugs)
	}

	if len(bugs) == 0 {
		fmt.Fprintf(w, "No bugs found\n")
		return nil
	}

	fmt.Fprintf(w, "%-12s %-40s %-20s %-10s\n", "Key", "Title", "Status", "Severity")
	fmt.Fprintf(w, "%-12s %-40s %-20s %-10s\n", "---", "-----", "------", "--------")
	for _, b := range bugs {
		title := b.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Fprintf(w, "%-12s %-40s %-20s %-10s\n", b.Key, title, string(b.Status), string(b.Severity))
	}
	return nil
}

// runBugUpdateWithService is the testable handler for bug update.
func runBugUpdateWithService(ctx context.Context, args []string, svc bugServicer, w io.Writer) error {
	key := args[0]

	updates := services.BugUpdates{}
	if bugUpdateTitle != "" {
		updates.Title = &bugUpdateTitle
	}
	if bugUpdateDescription != "" {
		updates.Description = &bugUpdateDescription
	}
	if bugUpdateSeverity != "" {
		severity := models.BugSeverity(bugUpdateSeverity)
		updates.Severity = &severity
	}
	if bugUpdateLinkType != "" {
		updates.LinkedEntityType = &bugUpdateLinkType
	}
	if bugUpdateLinkKey != "" {
		updates.LinkedEntityKey = &bugUpdateLinkKey
	}

	bug, err := svc.UpdateBug(ctx, key, updates)
	if err != nil {
		return fmt.Errorf("failed to update bug %s: %w", key, err)
	}

	if cli.GlobalConfig.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(bug)
	}

	fmt.Fprintf(w, "Updated bug %s\n", bug.Key)
	return nil
}

// runBugDeleteWithService is the testable handler for bug delete.
func runBugDeleteWithService(ctx context.Context, args []string, svc bugServicer, w io.Writer) error {
	key := args[0]

	if err := svc.DeleteBug(ctx, key); err != nil {
		return fmt.Errorf("failed to delete bug %s: %w", key, err)
	}

	if cli.GlobalConfig.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"key": key, "status": "deleted"})
	}

	fmt.Fprintf(w, "Deleted bug %s\n", key)
	return nil
}

// runBugTriageWithService is the testable handler for bug triage.
func runBugTriageWithService(ctx context.Context, args []string, svc bugServicer, w io.Writer) error {
	key := args[0]

	input := services.TriageBugInput{}
	if bugTriageSeverity != "" {
		severity := models.BugSeverity(bugTriageSeverity)
		input.Severity = &severity
	}
	if bugTriageAssign != "" {
		input.AssignedTo = &bugTriageAssign
	}
	if bugTriageLinkType != "" {
		input.LinkedEntityType = &bugTriageLinkType
	}
	if bugTriageLinkKey != "" {
		input.LinkedEntityKey = &bugTriageLinkKey
	}

	bug, err := svc.TriageBug(ctx, key, input)
	if err != nil {
		return fmt.Errorf("failed to triage bug %s: %w", key, err)
	}

	if cli.GlobalConfig.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(bug)
	}

	fmt.Fprintf(w, "Triaged bug %s (severity: %s)\n", bug.Key, bug.Severity)
	return nil
}
