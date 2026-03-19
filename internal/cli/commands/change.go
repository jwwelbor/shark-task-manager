package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// changeCardServicer is the interface used by change-card CLI commands.
// Defined here so tests can inject a mock without touching the global CLI layer.
type changeCardServicer interface {
	CreateChangeCard(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error)
	GetChangeCard(ctx context.Context, key string) (*models.ChangeCard, error)
	ListChangeCards(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error)
	UpdateChangeCard(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error)
	DeleteChangeCard(ctx context.Context, key string) error
	ApproveChangeCard(ctx context.Context, key string) (*models.ChangeCard, error)
	SetChangeCardStatus(ctx context.Context, key, targetStatus string) (*models.ChangeCard, error)
	AdvanceChangeCardStatus(ctx context.Context, key string) (*models.ChangeCard, error)
	GetOrchestratorAction(card *models.ChangeCard) *config.PopulatedAction
	GetValidTransitions(status string) []string
}

// changeCardSvcOverride is non-nil only during tests.
var changeCardSvcOverride changeCardServicer

// getChangeCardService returns the service to use, preferring the test override.
func getChangeCardService() changeCardServicer {
	if changeCardSvcOverride != nil {
		return changeCardSvcOverride
	}
	return cli.GetChangeCardService()
}

// changeCmd is the parent command group for change-card management.
var changeCmd = &cobra.Command{
	Use:     "change",
	Short:   "Manage change-cards",
	GroupID: "advanced",
	Long: `Change-card management operations for lightweight enhancement tracking.

Change-cards are used to track proposed and approved changes to the system.
Keys are in format C-### (e.g., C-001).

Examples:
  shark change list                            List all change-cards
  shark change create "Add dark mode"          Create a new change-card
  shark change get C-001                       Get change-card details
  shark change update C-001 --title="New title"
  shark change delete C-001                    Delete a change-card
  shark change approve C-001                   Approve a change-card`,
}

// changeCreateCmd creates a new change-card.
var changeCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new change-card",
	Long: `Create a new change-card with the given title and optional metadata.

Examples:
  shark change create "Add dark mode toggle"
  shark change create "Improve search" --link=E07
  shark change create "Fix navigation" --link=E07-F03 --description="..."
  shark change create "Performance fix" --requested-by=alice --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeCreate,
}

// changeGetCmd retrieves a specific change-card.
var changeGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get change-card details",
	Long: `Display detailed information about a specific change-card.

Examples:
  shark change get C-001
  shark change get C-001 --json
  shark change get C-001 --field status`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeGet,
}

// changeListCmd lists change-cards with optional filters.
var changeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List change-cards",
	Long: `List change-cards with optional filtering by status or linked entity.

By default, terminal-status change-cards (completed, declined) are hidden
unless --status is specified.

Examples:
  shark change list
  shark change list --status=proposed
  shark change list --link=E07
  shark change list --status=approved --json`,
	Args: cobra.NoArgs,
	RunE: runChangeList,
}

// changeUpdateCmd updates an existing change-card.
var changeUpdateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Update a change-card",
	Long: `Update properties of an existing change-card.

Examples:
  shark change update C-001 --title="New title"
  shark change update C-001 --description="Updated description"
  shark change update C-001 --assigned-to=alice
  shark change update C-001 --priority=8 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeUpdate,
}

// changeDeleteCmd deletes a change-card.
var changeDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a change-card",
	Long: `Delete a change-card by key. Requires confirmation unless --force is provided.

Examples:
  shark change delete C-001
  shark change delete C-001 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeDelete,
}

// changeApproveCmd approves a change-card.
var changeApproveCmd = &cobra.Command{
	Use:   "approve <key>",
	Short: "Approve a change-card",
	Long: `Approve a change-card, transitioning it to the 'approved' status.

The change-card must be in a status that allows transition to 'approved'.

Examples:
  shark change approve C-001
  shark change approve C-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeApprove,
}

// changeNoteCmd is the parent for note operations.
var changeNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage change-card notes",
	Long:  `Add and manage typed notes for change-cards.`,
}

// changeNoteAddCmd adds a note to a change-card.
var changeNoteAddCmd = &cobra.Command{
	Use:   "add <key> <content>",
	Short: "Add a typed note to a change-card",
	Long: `Add a typed note to a change-card for context, decisions, and documentation.

Note Types:
  comment        - General observation
  decision       - Why we chose X over Y
  blocker        - What is blocking progress
  solution       - How we solved a problem
  reference      - External links, documentation
  implementation - What we actually built
  question       - Unanswered questions

Examples:
  shark change note add C-001 "Needs design approval" --type=comment
  shark change note add C-001 "Chose dark mode over light" --type=decision
  shark change note add C-001 --type=blocker "Waiting on UX sign-off"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runChangeNoteAdd,
}

// changeNotesCmd lists notes for a change-card.
var changeNotesCmd = &cobra.Command{
	Use:   "notes <key>",
	Short: "List notes for a change-card",
	Long: `List all notes for a change-card, optionally filtered by type.

Examples:
  shark change notes C-001
  shark change notes C-001 --type=decision
  shark change notes C-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeNotes,
}

// changeContextCmd is the parent for context operations.
var changeContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage change-card context data",
	Long:  `Get, set, and clear structured context data for change-cards.`,
}

// changeContextSetCmd sets a context field on a change-card.
var changeContextSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set a context field on a change-card",
	Long: `Set or update a specific field in change-card context data.

Examples:
  shark change context set C-001 --field current_step --value "Awaiting approval"
  shark change context set C-001 --field completed_steps --value '["Design","Review"]'`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeContextSet,
}

// changeContextGetCmd gets context data for a change-card.
var changeContextGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get context data for a change-card",
	Long: `Display the current context data for a change-card.

Examples:
  shark change context get C-001
  shark change context get C-001 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeContextGet,
}

// changeContextClearCmd clears context data for a change-card.
var changeContextClearCmd = &cobra.Command{
	Use:   "clear <key>",
	Short: "Clear context data for a change-card",
	Long: `Remove all context data from a change-card.

Examples:
  shark change context clear C-001`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeContextClear,
}

// Command flag variables
var (
	changeLinkKey      string
	changeStatusFilter string
	changeLinkFilter   string
	changeTitle        string
	changeDescription  string
	changePriority     int
	changeRequestedBy  string
	changeAssignedTo   string
	changeForce        bool
	changeNoteType     string
	changeCtxField     string
	changeCtxValue     string
)

func init() {
	// Register change command group
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

	// Create flags
	changeCreateCmd.Flags().StringVar(&changeLinkKey, "link", "", "Link to epic or feature (E## or E##-F##)")
	changeCreateCmd.Flags().StringVar(&changeDescription, "description", "", "Change-card description")
	changeCreateCmd.Flags().IntVar(&changePriority, "priority", 0, "Priority (1-10)")
	changeCreateCmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "Who requested this change")

	// List flags
	changeListCmd.Flags().StringVar(&changeStatusFilter, "status", "", "Filter by status (proposed, approved, in_progress, completed, declined)")
	changeListCmd.Flags().StringVar(&changeLinkFilter, "link", "", "Filter by linked entity key (E## or E##-F##)")

	// Update flags
	changeUpdateCmd.Flags().StringVar(&changeTitle, "title", "", "New title")
	changeUpdateCmd.Flags().StringVar(&changeDescription, "description", "", "New description")
	changeUpdateCmd.Flags().IntVar(&changePriority, "priority", 0, "New priority (1-10)")
	changeUpdateCmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "Update requested by")
	changeUpdateCmd.Flags().StringVar(&changeAssignedTo, "assigned-to", "", "Assign to a person")
	changeUpdateCmd.Flags().String("file", "", "New file path")
	changeUpdateCmd.Flags().String("filename", "", "Alias for --file")
	changeUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = changeUpdateCmd.Flags().MarkHidden("filename")
	_ = changeUpdateCmd.Flags().MarkHidden("path")

	// Delete flags
	changeDeleteCmd.Flags().BoolVar(&changeForce, "force", false, "Skip confirmation prompt")

	// Note flags
	changeNoteAddCmd.Flags().StringVar(&changeNoteType, "type", "comment", "Note type (comment, decision, blocker, solution, reference, implementation, question)")

	// Context flags
	changeContextSetCmd.Flags().StringVar(&changeCtxField, "field", "", "Context field name (required)")
	changeContextSetCmd.Flags().StringVar(&changeCtxValue, "value", "", "Context field value (required)")
	_ = changeContextSetCmd.MarkFlagRequired("field")
	_ = changeContextSetCmd.MarkFlagRequired("value")

	changeContextClearCmd.Flags().StringVar(&changeCtxField, "field", "", "Specific field to clear (optional; clears all if omitted)")
}

// runChangeCreate handles the `shark change create` command.
func runChangeCreate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	title := args[0]
	input := buildCreateChangeCardInput(title)

	// Step 2: Call service
	svc := getChangeCardService()
	card, err := svc.CreateChangeCard(cmd.Context(), input)
	if err != nil {
		return fmt.Errorf("failed to create change-card: %w", err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
	return nil
}

// runChangeGet handles the `shark change get` command.
func runChangeGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service
	svc := getChangeCardService()
	card, err := svc.GetChangeCard(ctx, key)
	if err != nil {
		return fmt.Errorf("change-card %s not found: %w", key, err)
	}

	// Gather enrichment data (best-effort)
	orchestratorAction := svc.GetOrchestratorAction(card)
	validTransitions := svc.GetValidTransitions(string(card.Status))

	var notes []*models.EntityNote
	if noteSvc, nErr := cli.GetNoteService(ctx); nErr == nil && noteSvc != nil {
		notes, _ = noteSvc.ListNotes(ctx, models.EntityTypeChange, key, nil)
	}

	var contextData *models.ContextData
	if ctxSvc, cErr := cli.GetContextService(ctx); cErr == nil && ctxSvc != nil {
		contextData, _ = ctxSvc.GetContext(ctx, models.EntityTypeChange, key)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		result, err := buildEnrichedJSON(card, orchestratorAction, validTransitions)
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
		EntityType:         "change-card",
		Key:                card.Key,
		Status:             string(card.Status),
		BasicInfo:          buildChangeCardBasicInfo(card),
		ValidTransitions:   validTransitions,
		OrchestratorAction: orchestratorAction,
		Notes:              notes,
		ContextData:        contextData,
	})
	return nil
}

// runChangeList handles the `shark change list` command.
func runChangeList(cmd *cobra.Command, args []string) error {
	// Step 1: Parse filters
	filters := services.ChangeCardFilters{
		Status:  changeStatusFilter,
		ShowAll: changeStatusFilter != "",
	}
	// Route --link to FeatureKey when format is E##-F## (feature), otherwise EpicKey
	if changeLinkFilter != "" {
		if strings.Contains(changeLinkFilter, "-F") {
			filters.FeatureKey = changeLinkFilter
		} else {
			filters.EpicKey = changeLinkFilter
		}
	}

	// Step 2: Call service
	svc := getChangeCardService()
	cards, err := svc.ListChangeCards(cmd.Context(), filters)
	if err != nil {
		return fmt.Errorf("failed to list change-cards: %w", err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cards)
	}
	return printChangeCardList(cards)
}

// runChangeUpdate handles the `shark change update` command.
func runChangeUpdate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	updates := buildChangeCardUpdates(cmd)

	// Step 2: Call service
	svc := getChangeCardService()
	card, err := svc.UpdateChangeCard(cmd.Context(), key, updates)
	if err != nil {
		return fmt.Errorf("failed to update change-card %s: %w", key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Updated change-card %s", card.Key))
	return nil
}

// runChangeDelete handles the `shark change delete` command.
func runChangeDelete(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Optionally confirm
	svc := getChangeCardService()
	if !changeForce {
		card, err := svc.GetChangeCard(cmd.Context(), key)
		if err != nil {
			return fmt.Errorf("change-card %s not found: %w", key, err)
		}
		if !confirmChangeDelete(card) {
			return nil
		}
	}

	// Step 3: Call service
	if err := svc.DeleteChangeCard(cmd.Context(), key); err != nil {
		return fmt.Errorf("failed to delete change-card %s: %w", key, err)
	}

	// Step 4: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"status": "deleted", "key": key})
	}
	cli.Success(fmt.Sprintf("Deleted change-card %s", key))
	return nil
}

// runChangeApprove handles the `shark change approve` command.
func runChangeApprove(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	svc := getChangeCardService()
	card, err := svc.ApproveChangeCard(cmd.Context(), key)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			cli.Error(fmt.Sprintf("Change-card not found: %s", key))
			os.Exit(1)
		}
		if strings.Contains(errMsg, "cannot approve") || strings.Contains(errMsg, "invalid transition") ||
			strings.Contains(errMsg, "invalid status") || strings.Contains(errMsg, "cannot transition") {
			cli.Error(fmt.Sprintf("Cannot approve change-card %s: %s", key, errMsg))
			os.Exit(3)
		}
		cli.Error(fmt.Sprintf("Failed to approve change-card %s: %s", key, errMsg))
		os.Exit(2)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Approved change-card %s", card.Key))
	return nil
}

// runChangeNoteAdd handles the `shark change note add` command.
func runChangeNoteAdd(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	content := strings.Join(args[1:], " ")
	if content == "" {
		return fmt.Errorf("note content is required")
	}

	// Step 2: Call services
	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get note service: %w", err)
	}

	note, err := noteSvc.AddNote(cmd.Context(), models.EntityTypeChange, key, changeNoteType, content, "")
	if err != nil {
		return fmt.Errorf("failed to add note to change-card %s: %w", key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}

	ts := note.CreatedAt
	if ts.IsZero() {
		ts = time.Now()
	}
	creator := ""
	if note.CreatedBy != nil {
		creator = *note.CreatedBy
	}

	fmt.Printf("Note added to change-card %s\n\n", key)
	if creator != "" {
		fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(changeNoteType), ts.Format("2006-01-02 15:04"), creator)
	} else {
		fmt.Printf("[%s] %s\n", strings.ToUpper(changeNoteType), ts.Format("2006-01-02 15:04"))
	}
	fmt.Println(content)
	return nil
}

// runChangeNotes handles the `shark change notes` command.
func runChangeNotes(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get note service: %w", err)
	}

	notes, err := noteSvc.ListNotes(cmd.Context(), models.EntityTypeChange, key, nil)
	if err != nil {
		return fmt.Errorf("failed to list notes for change-card %s: %w", key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(notes)
	}

	if len(notes) == 0 {
		fmt.Printf("No notes found for change-card %s\n", key)
		return nil
	}

	fmt.Printf("Notes for change-card %s\n\n", key)
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

// runChangeContextSet handles the `shark change context set` command.
func runChangeContextSet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	field, _ := cmd.Flags().GetString("field")
	value, _ := cmd.Flags().GetString("value")

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeChange, key, field, value); err != nil {
		return fmt.Errorf("failed to set context field for change-card %s: %w", key, err)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_type": "change",
			"entity_key":  key,
			"field":       field,
			"success":     true,
		})
	}
	cli.Success(fmt.Sprintf("Updated context field '%s' for change-card %s", field, key))
	return nil
}

// runChangeContextGet handles the `shark change context get` command.
func runChangeContextGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	contextData, err := ctxSvc.GetContext(cmd.Context(), models.EntityTypeChange, key)
	if err != nil {
		return fmt.Errorf("failed to get context for change-card %s: %w", key, err)
	}

	if contextData == nil {
		contextData = &models.ContextData{}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_type":  "change",
			"entity_key":   key,
			"context_data": contextData,
		})
	}

	fmt.Printf("Context for change-card %s\n\n", key)
	printContextData(contextData)
	return nil
}

// runChangeContextClear handles the `shark change context clear` command.
func runChangeContextClear(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	field, _ := cmd.Flags().GetString("field")

	// Step 2: Call service
	ctxSvc, err := cli.GetContextService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to initialize context service: %w", err)
	}

	// Clear specific field or all context depending on --field flag
	if field != "" {
		if err := ctxSvc.SetContextField(cmd.Context(), models.EntityTypeChange, key, field, ""); err != nil {
			return fmt.Errorf("failed to clear context field for change-card %s: %w", key, err)
		}
	} else {
		if err := ctxSvc.ClearContext(cmd.Context(), models.EntityTypeChange, key); err != nil {
			return fmt.Errorf("failed to clear context for change-card %s: %w", key, err)
		}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_type": "change",
			"entity_key":  key,
			"success":     true,
		})
	}
	if field != "" {
		cli.Success(fmt.Sprintf("Cleared context field '%s' for change-card %s", field, key))
	} else {
		cli.Success(fmt.Sprintf("Cleared context data for change-card %s", key))
	}
	return nil
}

// --- Helper functions ---

// buildCreateChangeCardInput constructs a CreateChangeCardInput from flag values.
func buildCreateChangeCardInput(title string) services.CreateChangeCardInput {
	input := services.CreateChangeCardInput{
		Title:       title,
		Description: changeDescription,
		RequestedBy: changeRequestedBy,
	}
	// Parse --link into EpicKey or FeatureKey
	if changeLinkKey != "" {
		if strings.Contains(changeLinkKey, "-F") {
			input.FeatureKey = changeLinkKey
		} else {
			input.EpicKey = changeLinkKey
		}
	}
	if changePriority > 0 {
		input.Priority = changePriority
	}
	return input
}

// buildChangeCardUpdates constructs a ChangeCardUpdates from changed flags.
func buildChangeCardUpdates(cmd *cobra.Command) services.ChangeCardUpdates {
	updates := services.ChangeCardUpdates{}
	if cmd.Flags().Changed("title") {
		updates.Title = &changeTitle
	}
	if cmd.Flags().Changed("description") {
		updates.Description = &changeDescription
	}
	if cmd.Flags().Changed("priority") {
		updates.Priority = &changePriority
	}
	if cmd.Flags().Changed("requested-by") {
		updates.RequestedBy = &changeRequestedBy
	}
	if cmd.Flags().Changed("assigned-to") {
		updates.AssignedTo = &changeAssignedTo
	}
	if v := getFileFlagValue(cmd); v != "" {
		updates.FilePath = &v
	}
	return updates
}

// confirmChangeDelete prompts for confirmation before deleting a change-card.
func confirmChangeDelete(card *models.ChangeCard) bool {
	fmt.Printf("Are you sure you want to delete change-card %s: %s? (yes/no): ", card.Key, card.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.EqualFold(response, "yes") || strings.EqualFold(response, "y")
}

// printChangeCardList renders a table of change-cards.
func printChangeCardList(cards []*models.ChangeCard) error {
	if len(cards) == 0 {
		fmt.Println("No change-cards found")
		return nil
	}

	headers := []string{"Key", "Title", "Status", "Linked Entity", "Created"}
	rows := make([][]string, len(cards))
	for i, c := range cards {
		linkedEntity := "--"
		if c.FeatureID != nil {
			linkedEntity = fmt.Sprintf("F#%d", *c.FeatureID)
		} else if c.EpicID != nil {
			linkedEntity = fmt.Sprintf("E#%d", *c.EpicID)
		}
		rows[i] = []string{
			c.Key,
			c.Title,
			string(c.Status),
			linkedEntity,
			c.CreatedAt.Format("2006-01-02"),
		}
	}
	cli.OutputTable(headers, rows)
	return nil
}
