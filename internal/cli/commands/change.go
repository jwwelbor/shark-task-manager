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
	CreateChangeCard(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error)
	GetChangeCard(ctx context.Context, key string) (*models.ChangeCard, error)
	// GetChangeCardWithTags returns the change-card along with sorted tag names (REQ-F-014).
	// When TagService is nil or unavailable, tags will be nil (graceful degradation).
	GetChangeCardWithTags(ctx context.Context, key string) (*models.ChangeCard, []string, error)
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

By default, terminal-status change-cards (completed, declined) are hidden.
Use --all to show all change-cards including those in terminal statuses.

Examples:
  shark change list
  shark change list --all
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
	// changeCreateTags and changeUpdateTags back the repeatable --tag flag
	// on `shark change create` and `shark change update`. Declared as
	// distinct variables because they live on different commands; Cobra
	// does not reset flag-backing variables between command executions
	// (test code must zero them out to avoid cross-test bleed).
	changeCreateTags []string
	changeUpdateTags []string
)

// resolveChangeCardID is the EntityKeyResolver used by the `shark change tag`
// subcommand factory. It uses the existing change-card service accessor
// (either the production service or the test override in
// changeCardSvcOverride) to find a change-card by key and return its numeric
// ID.
//
// Split out as a package-level function (not a closure in init()) so the
// E28-F04 entity_tag_cmd.go factory can reference it and so tests can
// observe its behaviour through changeCardSvcOverride.
func resolveChangeCardID(ctx context.Context, key string) (int64, error) {
	svc := getChangeCardService()
	card, err := svc.GetChangeCard(ctx, key)
	if err != nil {
		return 0, err
	}
	return card.ID, nil
}

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

	// E28-F04 T-009: register the shared `tag add|rm` subcommand. Svc
	// override is nil in production so it falls through to
	// cli.GetTagService() at call time.
	changeCmd.AddCommand(makeEntityTagCmd(models.EntityTypeChange, resolveChangeCardID, nil))

	// Create flags
	changeCreateCmd.Flags().StringVar(&changeLinkKey, "link", "", "Link to epic or feature (E## or E##-F##)")
	changeCreateCmd.Flags().StringVar(&changeDescription, "description", "", "Change-card description")
	changeCreateCmd.Flags().IntVar(&changePriority, "priority", 0, "Priority (1-10)")
	changeCreateCmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "Who requested this change")
	// E28-F04 REQ-F-012: repeatable --tag flag. StringSliceVar collects
	// repeated occurrences into a []string; Cobra's comma-separator
	// behaviour means `--tag=foo,bar` becomes ["foo","bar"] — ADR-F04-5
	// accepts this because invalid-name chars (a comma is invalid under
	// the regex) surface as a clear exit-3 validation error.
	changeCreateCmd.Flags().StringSliceVar(&changeCreateTags, "tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	// E07-F42 REQ-F-004: optional size flag (StringVar per Decision D4).
	changeCreateCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")
	changeCreateCmd.Flags().String("content", "", "Pre-populate file body (stdin pipe also accepted)")

	// List flags
	changeListCmd.Flags().StringVar(&changeStatusFilter, "status", "", "Filter by status (proposed, approved, in_progress, completed, declined)")
	changeListCmd.Flags().StringVar(&changeLinkFilter, "link", "", "Filter by linked entity key (E## or E##-F##)")
	changeListCmd.Flags().Bool("all", false, "Show all change-cards including terminal statuses (completed, declined)")
	// E28-F05 REQ-F-010 / REQ-F-018: repeatable --tag flag with AND semantics.
	changeListCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")

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
	// E28-F04 REQ-F-012: `--tag` on update is additive (REQ-F-010). Empty
	// means no change; no way to detach here — use `shark change tag rm`.
	changeUpdateCmd.Flags().StringSliceVar(&changeUpdateTags, "tag", nil,
		"Tag to apply additively (repeatable). Empty = no change; use 'shark change tag rm' to detach.")
	// E07-F42 REQ-F-005: optional size flag with clear-literal support.
	changeUpdateCmd.Flags().String("size", "",
		"Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove on update)")

	// Delete flags
	changeDeleteCmd.Flags().BoolVar(&changeForce, "force", false, "Skip confirmation prompt")

}

// runChangeCreate handles the `shark change create` command.
func runChangeCreate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	title := args[0]
	input := buildCreateChangeCardInput(title)
	// E28-F04 REQ-F-012: read --tag via the flag accessor (not the
	// package-level variable) so test invocations that reset flags between
	// runs see a fresh value each time.
	if tags, err := cmd.Flags().GetStringSlice("tag"); err == nil {
		input.Tags = tags
	}
	// E07-F42 REQ-F-004: parse --size before calling service; reject invalid values early.
	if sizeStr, _ := cmd.Flags().GetString("size"); sizeStr != "" {
		n, sizeErr := models.ParseSize(sizeStr)
		if sizeErr != nil {
			return fmt.Errorf("invalid --size value: %w", sizeErr)
		}
		input.Size = &n
	}
	body, err := cli.ResolveContentInput(cmd)
	if err != nil {
		return err
	}
	input.Body = body

	// Step 2: Call service
	svc := getChangeCardService()
	card, fileWasLinked, err := svc.CreateChangeCard(cmd.Context(), input)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "change", input.Title)
	}

	// Step 3: Format output
	projectRoot, _ := cli.FindProjectRoot()
	cardFilePath := card.GetFilePath()
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cli.FormatEntityCreationJSON("change", card.Key, card.Title, cardFilePath, projectRoot))
	}
	fmt.Print(cli.FormatEntityCreationMessage("change", card.Key, card.Title, cardFilePath, projectRoot, fileWasLinked))
	return nil
}

// runChangeGet handles the `shark change get` command.
func runChangeGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service — use GetChangeCardWithTags for tag enrichment (REQ-F-014, REQ-F-015).
	svc := getChangeCardService()
	card, tags, err := svc.GetChangeCardWithTags(ctx, key)
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
		// REQ-F-015: "tags" field always present, never null.
		if tags == nil {
			tags = []string{}
		}
		result["tags"] = tags
		return cli.OutputJSON(result)
	}

	basicInfo := buildChangeCardBasicInfo(card)
	basicInfo = appendTagsToBasicInfo(basicInfo, tags)
	RenderEntity(EntityDisplayOptions{
		EntityType:         "change-card",
		Key:                card.Key,
		Status:             string(card.Status),
		BasicInfo:          basicInfo,
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
	allFlag, _ := cmd.Flags().GetBool("all")
	filters := services.ChangeCardFilters{
		Status:  changeStatusFilter,
		ShowAll: allFlag || changeStatusFilter != "",
	}
	// Route --link to FeatureKey when format is E##-F## (feature), otherwise EpicKey
	if changeLinkFilter != "" {
		if strings.Contains(changeLinkFilter, "-F") {
			filters.FeatureKey = changeLinkFilter
		} else {
			filters.EpicKey = changeLinkFilter
		}
	}
	// E28-F05 REQ-F-010: read the repeatable --tag flag; nil when absent (AC-T2).
	if rawTags, tagErr := cmd.Flags().GetStringSlice("tag"); tagErr == nil && len(rawTags) > 0 {
		filters.Tags = rawTags
	}

	// Step 2: Call service
	svc := getChangeCardService()
	cards, err := svc.ListChangeCards(cmd.Context(), filters)
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "change", "")
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
	updates, err := buildChangeCardUpdates(cmd)
	if err != nil {
		return err
	}

	// Step 2: Call service
	svc := getChangeCardService()
	card, err := svc.UpdateChangeCard(cmd.Context(), key, updates)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "change", key)
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
// Note: Tags are threaded in by the caller (runChangeCreate) via the flag
// accessor rather than here, to keep this function usable from tests that
// don't build a cobra.Command (E28-F04 T-009).
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
func buildChangeCardUpdates(cmd *cobra.Command) (services.ChangeCardUpdates, error) {
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
	// E28-F04 REQ-F-012: read --tag for additive tagging on update.
	// `cmd.Flags().Changed("tag")` ensures we only attach the slice when
	// the user actually passed the flag (vs. an empty default).
	if cmd.Flags().Changed("tag") {
		if tags, err := cmd.Flags().GetStringSlice("tag"); err == nil {
			updates.Tags = tags
		}
	}
	// E07-F42 REQ-F-005: three-way dispatch for --size on update.
	//   empty → no-op; "clear" → ClearSize=true; valid → Size=ptr(n).
	if cmd.Flags().Changed("size") {
		sizePtr, clearSize, sizeErr := parseSizeUpdateFlag(cmd)
		if sizeErr != nil {
			return updates, sizeErr
		}
		updates.Size = sizePtr
		updates.ClearSize = clearSize
	}
	return updates, nil
}

// confirmChangeDelete prompts for confirmation before deleting a change-card.
func confirmChangeDelete(card *models.ChangeCard) bool {
	fmt.Printf("Are you sure you want to delete change-card %s: %s? (yes/no): ", card.Key, card.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.EqualFold(response, "yes") || strings.EqualFold(response, "y")
}

// buildChangeCardListRows converts a slice of change-cards to table rows for list display.
// Extracted for testability (E07-F42 F4 coverage requirement).
func buildChangeCardListRows(cards []*models.ChangeCard) [][]string {
	rows := make([][]string, 0, len(cards))
	for _, c := range cards {
		linkedEntity := "--"
		if c.FeatureID != nil {
			linkedEntity = fmt.Sprintf("F#%d", *c.FeatureID)
		} else if c.EpicID != nil {
			linkedEntity = fmt.Sprintf("E#%d", *c.EpicID)
		}
		rows = append(rows, []string{
			c.Key,
			c.Title,
			string(c.Status),
			linkedEntity,
			c.CreatedAt.Format("2006-01-02"),
			formatSize(c.Size), // E07-F42 REQ-F-006: Size column
		})
	}
	return rows
}

// printChangeCardList renders a table of change-cards.
func printChangeCardList(cards []*models.ChangeCard) error {
	if len(cards) == 0 {
		fmt.Println("No change-cards found")
		return nil
	}

	// E07-F42: Size column added to change-card list table (REQ-F-006).
	headers := []string{"Key", "Title", "Status", "Linked Entity", "Created", "Size"}
	cli.OutputTable(headers, buildChangeCardListRows(cards))
	return nil
}
