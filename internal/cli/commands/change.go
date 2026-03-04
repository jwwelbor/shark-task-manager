package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// -- Flag variables for change-card commands --

var changeLinkKey string      // --link flag for create and list
var changeStatusFilter string // --status flag for list
var changeLinkFilter string   // --link flag for list (filter)
var changeTitle string        // --title flag for update
var changeForce bool          // --force flag for delete
var changeNoteType string     // --type flag for note add
var changeCtxField string     // --field flag for context set/clear
var changeCtxValue string     // --value flag for context set

// changeCmd is the parent command for all change-card operations.
var changeCmd = &cobra.Command{
	Use:     "change",
	Short:   "Manage change-cards",
	GroupID: "advanced",
	Long: `Change-card management operations for lightweight enhancement tracking.

Change-cards track proposed changes, enhancements, and modifications using keys
in the format CC-### (e.g., CC-001, CC-042).

Examples:
  shark change list                         List all change-cards
  shark change create "Add dark mode"       Create a new change-card
  shark change get CC-001                   Get change-card details
  shark change update CC-001 --title="..."  Update a change-card
  shark change approve CC-001               Approve a change-card
  shark change delete CC-001 --force        Delete a change-card`,
}

// changeCreateCmd creates a new change-card.
var changeCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new change-card",
	Long: `Create a new change-card with an auto-generated key (CC-### format).

Optionally link the change-card to an epic (E##) or feature (E##-F##) using --link.

Examples:
  shark change create "Add dark mode toggle"
  shark change create "Improve search" --link=E07
  shark change create "Cache layer" --link=E07-F03
  shark change create "Fix UI" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeCreate,
}

// changeGetCmd retrieves a change-card by key.
var changeGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get change-card details",
	Long: `Display detailed information about a specific change-card.

Examples:
  shark change get CC-001
  shark change get CC-001 --json
  shark change get CC-001 --field status`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeGet,
}

// changeListCmd lists change-cards with optional filtering.
var changeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List change-cards",
	Long: `List change-cards with optional filtering by status or linked entity.

Examples:
  shark change list                          List all change-cards
  shark change list --status=proposed        Filter by status
  shark change list --link=E07               Filter by linked epic
  shark change list --link=E07-F03           Filter by linked feature
  shark change list --json                   JSON output`,
	Args: cobra.NoArgs,
	RunE: runChangeList,
}

// changeUpdateCmd updates a change-card.
var changeUpdateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Update a change-card",
	Long: `Update fields on an existing change-card. Only specified flags are updated.

Examples:
  shark change update CC-001 --title="New title"
  shark change update CC-001 --title="Better title" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeUpdate,
}

// changeDeleteCmd deletes a change-card.
var changeDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a change-card",
	Long: `Permanently delete a change-card.

Prompts for confirmation unless --force is specified.

Examples:
  shark change delete CC-001
  shark change delete CC-001 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeDelete,
}

// changeApproveCmd approves a change-card.
var changeApproveCmd = &cobra.Command{
	Use:   "approve <key>",
	Short: "Approve a change-card",
	Long: `Approve a change-card, transitioning it from proposed to approved status.

Examples:
  shark change approve CC-001
  shark change approve CC-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeApprove,
}

// changeNoteCmd is the parent command for note operations.
var changeNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage change-card notes",
	Long:  `Manage notes attached to change-cards.`,
}

// changeNoteAddCmd adds a note to a change-card.
var changeNoteAddCmd = &cobra.Command{
	Use:   "add <key> <content>",
	Short: "Add a note to a change-card",
	Long: `Add a note to a change-card. Content can be multiple words without quotes.

Examples:
  shark change note add CC-001 "Approved by stakeholders"
  shark change note add CC-001 --type=decision This was chosen over alternative approach
  shark change note add CC-001 --type=progress Implementation started`,
	Args: cobra.MinimumNArgs(2),
	RunE: runChangeNoteAdd,
}

// changeNotesCmd lists notes on a change-card.
var changeNotesCmd = &cobra.Command{
	Use:   "notes <key>",
	Short: "List notes for a change-card",
	Long: `Display all notes attached to a change-card.

Examples:
  shark change notes CC-001
  shark change notes CC-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeNotes,
}

// changeContextCmd is the parent command for context operations.
var changeContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage change-card context data",
	Long:  `Manage structured context data attached to change-cards.`,
}

// changeContextSetCmd sets a context field on a change-card.
var changeContextSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set a context field on a change-card",
	Long: `Set a named context field on a change-card.

Examples:
  shark change context set CC-001 --field current_step --value "Reviewing impact"
  shark change context set CC-001 --field assignee --value "alice"`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeContextSet,
}

// changeContextGetCmd retrieves context data for a change-card.
var changeContextGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get context data for a change-card",
	Long: `Display all context data attached to a change-card.

Examples:
  shark change context get CC-001
  shark change context get CC-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeContextGet,
}

// changeContextClearCmd clears context data from a change-card.
var changeContextClearCmd = &cobra.Command{
	Use:   "clear <key>",
	Short: "Clear context data from a change-card",
	Long: `Remove all context data from a change-card.

Examples:
  shark change context clear CC-001`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeContextClear,
}

func init() {
	cli.RootCmd.AddCommand(changeCmd)

	// CRUD + lifecycle subcommands
	changeCmd.AddCommand(changeCreateCmd)
	changeCmd.AddCommand(changeGetCmd)
	changeCmd.AddCommand(changeListCmd)
	changeCmd.AddCommand(changeUpdateCmd)
	changeCmd.AddCommand(changeDeleteCmd)
	changeCmd.AddCommand(changeApproveCmd)

	// Notes
	changeCmd.AddCommand(changeNoteCmd)
	changeNoteCmd.AddCommand(changeNoteAddCmd)
	changeCmd.AddCommand(changeNotesCmd)

	// Context
	changeCmd.AddCommand(changeContextCmd)
	changeContextCmd.AddCommand(changeContextSetCmd)
	changeContextCmd.AddCommand(changeContextGetCmd)
	changeContextCmd.AddCommand(changeContextClearCmd)

	// Flags: create
	changeCreateCmd.Flags().StringVar(&changeLinkKey, "link", "", "Link to epic or feature (E## or E##-F##)")

	// Flags: list
	changeListCmd.Flags().StringVar(&changeStatusFilter, "status", "", "Filter by status (e.g. proposed, approved, in_progress, completed, declined)")
	changeListCmd.Flags().StringVar(&changeLinkFilter, "link", "", "Filter by linked entity key (E## or E##-F##)")

	// Flags: update
	changeUpdateCmd.Flags().StringVar(&changeTitle, "title", "", "New title for the change-card")

	// Flags: delete
	changeDeleteCmd.Flags().BoolVar(&changeForce, "force", false, "Skip confirmation prompt")

	// Flags: note add
	changeNoteAddCmd.Flags().StringVar(&changeNoteType, "type", "comment", "Note type (comment, decision, progress, question, solution, implementation)")

	// Flags: context set
	changeContextSetCmd.Flags().StringVar(&changeCtxField, "field", "", "Context field name to set")
	changeContextSetCmd.Flags().StringVar(&changeCtxValue, "value", "", "Context field value to set")
	_ = changeContextSetCmd.MarkFlagRequired("field")
	_ = changeContextSetCmd.MarkFlagRequired("value")

	// Flags: context clear
	changeContextClearCmd.Flags().StringVar(&changeCtxField, "field", "", "Context field to clear (clears all if omitted)")
}

// -- Handler functions --

func runChangeCreate(cmd *cobra.Command, args []string) error {
	title := args[0]

	input := services.CreateChangeCardInput{
		Title: title,
	}
	// --link may be an epic key (E##) or feature key (E##-F##)
	if changeLinkKey != "" {
		if strings.Contains(changeLinkKey, "-F") {
			input.FeatureKey = changeLinkKey
		} else {
			input.EpicKey = changeLinkKey
		}
	}

	svc := cli.GetChangeCardService()
	card, err := svc.CreateChangeCard(cmd.Context(), input)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
	return nil
}

func runChangeGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	svc := cli.GetChangeCardService()
	card, err := svc.GetChangeCard(cmd.Context(), key)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}

	printChangeCardDetail(card)
	return nil
}

func runChangeList(cmd *cobra.Command, args []string) error {
	filters := services.ChangeCardFilters{
		Status: changeStatusFilter,
	}
	if changeLinkFilter != "" {
		if strings.Contains(changeLinkFilter, "-F") {
			filters.EpicKey = changeLinkFilter
		} else {
			filters.EpicKey = changeLinkFilter
		}
	}

	svc := cli.GetChangeCardService()
	cards, err := svc.ListChangeCards(cmd.Context(), filters)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cards)
	}

	return printChangeCardList(cards)
}

func runChangeUpdate(cmd *cobra.Command, args []string) error {
	key := args[0]

	updates := services.ChangeCardUpdates{}
	if changeTitle != "" {
		updates.Title = &changeTitle
	}

	svc := cli.GetChangeCardService()
	card, err := svc.UpdateChangeCard(cmd.Context(), key, updates)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Updated change-card %s", card.Key))
	return nil
}

func runChangeDelete(cmd *cobra.Command, args []string) error {
	key := args[0]

	svc := cli.GetChangeCardService()
	card, err := svc.GetChangeCard(cmd.Context(), key)
	if err != nil {
		return err
	}

	if !changeForce {
		if !confirmChangeDelete(card) {
			return nil
		}
	}

	if err := svc.DeleteChangeCard(cmd.Context(), key); err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"status": "deleted", "key": key})
	}
	cli.Success(fmt.Sprintf("Deleted change-card %s", key))
	return nil
}

func runChangeApprove(cmd *cobra.Command, args []string) error {
	key := args[0]

	svc := cli.GetChangeCardService()
	card, err := svc.ApproveChangeCard(cmd.Context(), key)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Approved change-card %s", card.Key))
	return nil
}

func runChangeNoteAdd(cmd *cobra.Command, args []string) error {
	key := args[0]
	content := strings.Join(args[1:], " ")

	// Resolve the change-card to confirm it exists
	svc := cli.GetChangeCardService()
	_, err := svc.GetChangeCard(cmd.Context(), key)
	if err != nil {
		return err
	}

	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize note service: %w", err)
	}

	note, err := noteSvc.AddNote(cmd.Context(), models.EntityTypeChange, key, changeNoteType, content, "")
	if err != nil {
		return fmt.Errorf("failed to add note: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}
	cli.Success(fmt.Sprintf("Added %s note to change-card %s", changeNoteType, key))
	return nil
}

func runChangeNotes(cmd *cobra.Command, args []string) error {
	key := args[0]

	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize note service: %w", err)
	}

	notes, err := noteSvc.ListNotes(cmd.Context(), models.EntityTypeChange, key, nil)
	if err != nil {
		return fmt.Errorf("failed to list notes: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(notes)
	}

	if len(notes) == 0 {
		cli.Info(fmt.Sprintf("No notes found for change-card %s", key))
		return nil
	}

	for _, note := range notes {
		fmt.Printf("[%s] %s: %s\n", note.CreatedAt.Format("2006-01-02 15:04"), note.NoteType, note.Content)
	}
	return nil
}

func runChangeContextSet(cmd *cobra.Command, args []string) error {
	key := args[0]

	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeChange, key, changeCtxField, changeCtxValue); err != nil {
		return fmt.Errorf("failed to set context field: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"key": key, "field": changeCtxField, "value": changeCtxValue})
	}
	cli.Success(fmt.Sprintf("Set context field %q on change-card %s", changeCtxField, key))
	return nil
}

func runChangeContextGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	contextData, err := ctxSvc.GetContext(cmd.Context(), models.EntityTypeChange, key)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(contextData)
	}

	if contextData == nil {
		cli.Info(fmt.Sprintf("No context data found for change-card %s", key))
		return nil
	}

	fmt.Printf("Context for change-card %s:\n", key)
	if contextData.Progress != nil {
		if contextData.Progress.CurrentStep != nil && *contextData.Progress.CurrentStep != "" {
			fmt.Printf("  current_step: %s\n", *contextData.Progress.CurrentStep)
		}
		if len(contextData.Progress.CompletedSteps) > 0 {
			fmt.Printf("  completed_steps: %s\n", strings.Join(contextData.Progress.CompletedSteps, ", "))
		}
	}
	for k, v := range contextData.ImplementationDecisions {
		fmt.Printf("  %s: %s\n", k, v)
	}
	return nil
}

func runChangeContextClear(cmd *cobra.Command, args []string) error {
	key := args[0]

	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.ClearContext(cmd.Context(), models.EntityTypeChange, key); err != nil {
		return fmt.Errorf("failed to clear context: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{"key": key, "success": true, "message": "Context cleared"})
	}
	cli.Success(fmt.Sprintf("Cleared context data for change-card %s", key))
	return nil
}

// -- Output helpers --

// printChangeCardDetail prints a human-readable detail view of a change-card.
func printChangeCardDetail(card *models.ChangeCard) {
	fmt.Printf("Change-Card: %s\n", card.Key)
	fmt.Printf("  Title:   %s\n", card.Title)
	fmt.Printf("  Status:  %s\n", string(card.Status))
	if card.Description != nil && *card.Description != "" {
		fmt.Printf("  Desc:    %s\n", *card.Description)
	}
	if card.RequestedBy != nil && *card.RequestedBy != "" {
		fmt.Printf("  Requested By: %s\n", *card.RequestedBy)
	}
	if card.AssignedTo != nil && *card.AssignedTo != "" {
		fmt.Printf("  Assigned To:  %s\n", *card.AssignedTo)
	}
	linkedEntity := resolveChangeCardLink(card)
	fmt.Printf("  Linked:  %s\n", linkedEntity)
	if card.FilePath != "" {
		fmt.Printf("  File:    %s\n", card.FilePath)
	}
	fmt.Printf("  Created: %s\n", card.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Updated: %s\n", card.UpdatedAt.Format("2006-01-02 15:04:05"))
}

// printChangeCardList prints a table of change-cards.
func printChangeCardList(cards []*models.ChangeCard) error {
	if len(cards) == 0 {
		cli.Info("No change-cards found")
		return nil
	}

	headers := []string{"Key", "Title", "Status", "Linked Entity", "Created"}
	var rows [][]string
	for _, card := range cards {
		linkedEntity := resolveChangeCardLink(card)
		rows = append(rows, []string{
			card.Key,
			truncateChangeCardTitle(card.Title, 40),
			string(card.Status),
			linkedEntity,
			card.CreatedAt.Format("2006-01-02"),
		})
	}

	cli.OutputTable(headers, rows)
	return nil
}

// resolveChangeCardLink returns a human-readable linked entity string.
// Returns "(none)" when no link exists, or the entity key hint.
func resolveChangeCardLink(card *models.ChangeCard) string {
	// The card model stores EpicID and FeatureID as foreign keys.
	// For display we show what we can infer from the IDs being set.
	if card.FeatureID != nil {
		return fmt.Sprintf("feature:%d", *card.FeatureID)
	}
	if card.EpicID != nil {
		return fmt.Sprintf("epic:%d", *card.EpicID)
	}
	return "(none)"
}

// truncateChangeCardTitle truncates s to maxLen characters, appending "..." if truncated.
func truncateChangeCardTitle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// confirmChangeDelete shows a confirmation prompt; returns true if user confirms.
func confirmChangeDelete(card *models.ChangeCard) bool {
	fmt.Printf("Are you sure you want to permanently delete change-card %s: %s? (yes/no): ", card.Key, card.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.EqualFold(response, "yes") || strings.EqualFold(response, "y")
}
