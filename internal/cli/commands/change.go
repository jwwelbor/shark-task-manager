package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
	GetNextStatusForCard(card *models.ChangeCard) *services.NextStatusInfo
	GetOrchestratorAction(card *models.ChangeCard) *config.PopulatedAction
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

	// Notes (generic handlers)
	changeCmd.AddCommand(makeNoteCmd("change"))
	changeCmd.AddCommand(makeNotesCmd("change"))

	// Context (generic handler)
	changeCmd.AddCommand(makeContextCmd("change"))

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
	info := svc.GetNextStatusForCard(card)
	validTransitions := info.TargetStatuses()

	var notes []*models.EntityNote
	if noteSvc, nErr := cli.GetNoteService(ctx); nErr == nil && noteSvc != nil {
		notes, _ = noteSvc.ListNotes(ctx, models.EntityTypeChange, key, nil)
	}

	var contextData *models.ContextData
	if ctxSvc := cli.GetContextService(); ctxSvc != nil {
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

	svc := getChangeCardService()
	result, err := svc.TransitionStatus(cmd.Context(), key, "approved", services.TransitionOptions{})
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
		// Re-fetch the card for full JSON output
		card, getErr := svc.GetChangeCard(cmd.Context(), key)
		if getErr != nil {
			return cli.OutputJSON(result)
		}
		return cli.OutputJSON(card)
	}
	cli.Success(fmt.Sprintf("Approved change-card %s", result.EntityKey))
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
