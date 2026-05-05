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

// techDebtServicer defines the interface for tech-debt service operations used by CLI commands.
type techDebtServicer interface {
	CreateTechDebt(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, bool, error)
	GetTechDebt(ctx context.Context, key string) (*models.TechDebt, error)
	ListTechDebts(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error)
	UpdateTechDebt(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error)
	DeleteTechDebt(ctx context.Context, key string) error
	TriageTechDebt(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error)
	GetNextStatusForTechDebt(td *models.TechDebt) *services.NextStatusInfo
	GetOrchestratorAction(td *models.TechDebt) *config.PopulatedAction
}

// tdSvcOverride is non-nil only during tests.
var tdSvcOverride techDebtServicer

// getTechDebtService returns the service to use, preferring the test override.
func getTechDebtService() techDebtServicer {
	if tdSvcOverride != nil {
		return tdSvcOverride
	}
	return cli.GetTechDebtService()
}

// tdCmd is the parent command for all tech-debt operations.
var tdCmd = &cobra.Command{
	Use:     "td",
	Short:   "Manage technical debt items",
	GroupID: "advanced",
	Long: `Technical debt management operations for tracking and resolving tech debt.

Tech-debt items are assigned keys in format TD-### (e.g., TD-001, TD-042).

Examples:
  shark td list                                  List all tech-debt items
  shark td create "Refactor auth module"         Create a new tech-debt item
  shark td get TD-001                            Get tech-debt details
  shark td triage TD-001 --severity=high         Triage a tech-debt item
  shark td delete TD-001                         Delete a tech-debt item`,
}

// tdCreateCmd creates a new tech-debt item.
var tdCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new tech-debt item",
	Long: `Create a new tech-debt item with auto-generated key (TD-### format).

Examples:
  shark td create "Refactor auth module"
  shark td create "Update dependencies" --category=dependency --severity=high
  shark td create "Add unit tests" --effort-estimate=M --json
  shark td create "Refactor auth module" --size=L`,
	Args: cobra.ExactArgs(1),
	RunE: runTdCreate,
}

// tdGetCmd retrieves a specific tech-debt item by key.
var tdGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get tech-debt details",
	Long: `Display detailed information about a specific tech-debt item.

Examples:
  shark td get TD-001
  shark td get TD-001 --json
  shark td get TD-001 --field severity`,
	Args: cobra.ExactArgs(1),
	RunE: runTdGet,
}

// tdListCmd lists tech-debt items with optional filters.
var tdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tech-debt items",
	Long: `List tech-debt items with optional filtering by status, category, and severity.

By default, terminal-status items (resolved, wont_fix, cancelled) are hidden.
Use --all to show all items including those in terminal statuses.

Examples:
  shark td list
  shark td list --all
  shark td list --status=identified
  shark td list --category=architecture
  shark td list --severity=critical
  shark td list --category=testing --severity=high --json`,
	RunE: runTdList,
}

// tdUpdateCmd updates tech-debt fields.
var tdUpdateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Update a tech-debt item",
	Long: `Update tech-debt fields (title, category, severity, effort-estimate, description, size).

At least one update flag must be provided.
Does NOT accept --status; use "shark status set" instead.

Examples:
  shark td update TD-001 --title="Updated title"
  shark td update TD-001 --category=architecture --severity=critical
  shark td update TD-001 --effort-estimate=L --json
  shark td update TD-001 --size=XL
  shark td update TD-001 --size=clear`,
	Args: cobra.ExactArgs(1),
	RunE: runTdUpdate,
}

// tdDeleteCmd deletes a tech-debt item.
var tdDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a tech-debt item",
	Long: `Delete a tech-debt item and its associated data.

Confirmation is required unless --force is provided.

Examples:
  shark td delete TD-001
  shark td delete TD-001 --force
  shark td delete TD-001 --force --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTdDelete,
}

// tdTriageCmd triages a tech-debt item.
var tdTriageCmd = &cobra.Command{
	Use:   "triage <key>",
	Short: "Triage a tech-debt item",
	Long: `Triage a tech-debt item by setting severity, category, and effort estimate.
Advances status from 'identified' to 'triaged' if currently in 'identified' status.
If already past 'identified', updates fields without changing status.

Examples:
  shark td triage TD-001 --severity=high
  shark td triage TD-001 --severity=critical --category=architecture
  shark td triage TD-001 --severity=medium --effort-estimate=L --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTdTriage,
}

// Command flag variables for tech-debt commands.
// These globals are bound by list/delete/triage flag registrations only.
// Create/update flag values are read via cmd.Flags().GetString(...) instead
// (see registerTdCreateFlags / registerTdUpdateFlags), so the create/update
// counterparts (tdTitle, tdDescription, tdFilePath) are not needed here.
var (
	tdCategory       string
	tdSeverity       string
	tdEffortEstimate string
	tdStatus         string
	tdForce          bool
)

func init() {
	// Register td command and subcommands
	cli.RootCmd.AddCommand(tdCmd)
	tdCmd.AddCommand(tdCreateCmd)
	tdCmd.AddCommand(tdGetCmd)
	tdCmd.AddCommand(tdListCmd)
	tdCmd.AddCommand(tdUpdateCmd)
	tdCmd.AddCommand(tdDeleteCmd)
	tdCmd.AddCommand(tdTriageCmd)
	tdCmd.AddCommand(makeNoteCmd("tech_debt"))
	tdCmd.AddCommand(makeNotesCmd("tech_debt"))
	tdCmd.AddCommand(makeContextCmd("tech_debt"))

	// Create flags
	registerTdCreateFlags(tdCreateCmd)

	// List flags
	tdListCmd.Flags().StringVar(&tdStatus, "status", "", "Filter by status")
	tdListCmd.Flags().StringVar(&tdCategory, "category", "", "Filter by category")
	tdListCmd.Flags().StringVar(&tdSeverity, "severity", "", "Filter by severity")
	tdListCmd.Flags().Bool("all", false, "Show all items including terminal statuses (resolved, wont_fix, cancelled)")

	registerTdUpdateFlags(tdUpdateCmd)

	// Delete flags
	tdDeleteCmd.Flags().BoolVar(&tdForce, "force", false, "Skip confirmation prompt")

	// Triage flags
	tdTriageCmd.Flags().StringVar(&tdSeverity, "severity", "", "Severity (critical, high, medium, low)")
	tdTriageCmd.Flags().StringVar(&tdCategory, "category", "", "Category (code-quality, architecture, dependency, testing, performance, documentation)")
	tdTriageCmd.Flags().StringVar(&tdEffortEstimate, "effort-estimate", "", "Effort estimate (e.g., XS, S, M, L, XL)")
}

// registerTdCreateFlags registers the create-specific flags on a cobra
// command. Used by both `shark td create` and the unified `shark create
// tech-debt` alias so both surfaces accept identical flags. Reads happen via
// cmd.Flags().GetString — no global bindings.
func registerTdCreateFlags(cmd *cobra.Command) {
	cmd.Flags().String("category", "", "Category (code-quality, architecture, dependency, testing, performance, documentation)")
	cmd.Flags().String("severity", "", "Severity (critical, high, medium, low)")
	cmd.Flags().String("effort-estimate", "", "Effort estimate (e.g., XS, S, M, L, XL)")
	cmd.Flags().String("description", "", "Description of the tech debt")
	cmd.Flags().String("file", "", "Custom file path for tech-debt markdown file")
	cmd.Flags().String("filename", "", "Alias for --file")
	cmd.Flags().String("path", "", "Alias for --file")
	_ = cmd.Flags().MarkHidden("filename")
	_ = cmd.Flags().MarkHidden("path")
	cmd.Flags().Bool("force", false, "Overwrite existing file at target path")
	cmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")
	// Repeatable --tag flag. Tag must be registered in the vocabulary
	// (see `shark tags list` / `shark tags add`).
	cmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	cmd.Flags().String("content", "", "Pre-populate file body (stdin pipe also accepted)")
}

// registerTdUpdateFlags registers update-specific flags. Mirrors
// registerTdCreateFlags but uses additive --tag semantics matching the
// bug/change-card update pattern.
func registerTdUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("category", "", "New category")
	cmd.Flags().String("severity", "", "New severity")
	cmd.Flags().String("effort-estimate", "", "New effort estimate")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().String("file", "", "New file path")
	cmd.Flags().String("filename", "", "Alias for --file")
	cmd.Flags().String("path", "", "Alias for --file")
	_ = cmd.Flags().MarkHidden("filename")
	_ = cmd.Flags().MarkHidden("path")
	cmd.Flags().String("size", "",
		"Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove)")
	// `--tag` on update is ADDITIVE only; detach via `shark td tag rm`.
	cmd.Flags().StringSlice("tag", nil,
		"Tag to apply additively (repeatable). Empty = no change; use 'shark td tag rm' to detach.")
}

// runTdCreate handles the `shark td create` command.
func runTdCreate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	category, _ := cmd.Flags().GetString("category")
	severity, _ := cmd.Flags().GetString("severity")
	effortEstimate, _ := cmd.Flags().GetString("effort-estimate")
	description, _ := cmd.Flags().GetString("description")
	filePath := getFileFlagValue(cmd)
	force, _ := cmd.Flags().GetBool("force")
	tags, _ := cmd.Flags().GetStringSlice("tag")

	// Apply defaults for category and severity if not provided
	if category == "" {
		category = string(models.TechDebtCategoryCodeQuality)
	}
	if severity == "" {
		severity = string(models.TechDebtSeverityMedium)
	}

	// Parse --size before calling the service so invalid values are rejected early.
	var sizePtr *int
	if sizeStr, _ := cmd.Flags().GetString("size"); sizeStr != "" {
		n, sizeErr := models.ParseSize(sizeStr)
		if sizeErr != nil {
			return fmt.Errorf("invalid --size value: %w", sizeErr)
		}
		sizePtr = &n
	}

	body, err := cli.ResolveContentInput(cmd)
	if err != nil {
		return err
	}

	input := services.CreateTechDebtInput{
		Title:          args[0],
		Category:       models.TechDebtCategory(category),
		Severity:       models.TechDebtSeverity(severity),
		EffortEstimate: effortEstimate,
		Description:    description,
		Force:          force,
		Size:           sizePtr,
		Tags:           tags,
		Body:           body,
	}

	if filePath != "" {
		input.FilePath = &filePath
	}

	// Step 2: Call service
	svc := getTechDebtService()
	td, fileWasLinked, err := svc.CreateTechDebt(cmd.Context(), input)
	if err != nil {
		return err
	}

	// Step 3: Format output
	projectRoot, _ := cli.FindProjectRoot()
	tdFilePath := td.GetFilePath()
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cli.FormatEntityCreationJSON("tech-debt", td.Key, td.Title, tdFilePath, projectRoot))
	}
	fmt.Print(cli.FormatEntityCreationMessage("tech-debt", td.Key, td.Title, tdFilePath, projectRoot, fileWasLinked))
	return nil
}

// runTdGet handles the `shark td get` command.
func runTdGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service
	svc := getTechDebtService()
	td, err := svc.GetTechDebt(ctx, key)
	if err != nil {
		return err
	}

	// Gather enrichment data (best-effort)
	orchestratorAction := svc.GetOrchestratorAction(td)
	info := svc.GetNextStatusForTechDebt(td)
	validTransitions := info.TargetStatuses()

	var notes []*models.EntityNote
	if noteSvc, nErr := cli.GetNoteService(ctx); nErr == nil && noteSvc != nil {
		notes, _ = noteSvc.ListNotes(ctx, models.EntityTypeTechDebt, key, nil)
	}

	var contextData *models.ContextData
	if ctxSvc := cli.GetContextService(); ctxSvc != nil {
		contextData, _ = ctxSvc.GetContext(ctx, models.EntityTypeTechDebt, key)
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		result, err := buildEnrichedJSON(td, orchestratorAction, validTransitions)
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
		EntityType:         "tech_debt",
		Key:                td.Key,
		Status:             string(td.Status),
		BasicInfo:          buildTechDebtBasicInfo(td),
		ValidTransitions:   validTransitions,
		OrchestratorAction: orchestratorAction,
		Notes:              notes,
		ContextData:        contextData,
	})
	return nil
}

// runTdList handles the `shark td list` command.
func runTdList(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	statusStr, _ := cmd.Flags().GetString("status")
	categoryStr, _ := cmd.Flags().GetString("category")
	severityStr, _ := cmd.Flags().GetString("severity")
	allFlag, _ := cmd.Flags().GetBool("all")

	filters := services.TechDebtFilters{
		ShowAll: allFlag || statusStr != "",
	}
	if statusStr != "" {
		filters.Status = &statusStr
	}
	if categoryStr != "" {
		cat := models.TechDebtCategory(categoryStr)
		filters.Category = &cat
	}
	if severityStr != "" {
		sev := models.TechDebtSeverity(severityStr)
		filters.Severity = &sev
	}

	// Step 2: Call service
	svc := getTechDebtService()
	items, err := svc.ListTechDebts(cmd.Context(), filters)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(items)
	}

	if len(items) == 0 {
		cli.Info("No tech-debt items found")
		return nil
	}
	return printTechDebtTable(items)
}

// runTdUpdate handles the `shark td update` command.
func runTdUpdate(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]

	updates := services.TechDebtUpdates{}
	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		updates.Title = &title
	}
	if cmd.Flags().Changed("category") {
		category, _ := cmd.Flags().GetString("category")
		cat := models.TechDebtCategory(category)
		updates.Category = &cat
	}
	if cmd.Flags().Changed("severity") {
		severity, _ := cmd.Flags().GetString("severity")
		sev := models.TechDebtSeverity(severity)
		updates.Severity = &sev
	}
	if cmd.Flags().Changed("effort-estimate") {
		effort, _ := cmd.Flags().GetString("effort-estimate")
		updates.EffortEstimate = &effort
	}
	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		updates.Description = &desc
	}

	if v := getFileFlagValue(cmd); v != "" {
		updates.FilePath = &v
	}

	// Three-way dispatch for --size on update.
	//   empty → no-op; "clear" → ClearSize=true; valid → Size=ptr(n).
	if cmd.Flags().Changed("size") {
		sizePtr, clearSize, sizeErr := parseSizeUpdateFlag(cmd)
		if sizeErr != nil {
			return sizeErr
		}
		updates.Size = sizePtr
		updates.ClearSize = clearSize
	}

	// `--tag` on update is additive only. Guard with Changed so only
	// explicit --tag usage sets Tags (nil otherwise; empty = no change).
	if cmd.Flags().Changed("tag") {
		tags, _ := cmd.Flags().GetStringSlice("tag")
		updates.Tags = tags
	}

	if updates.Title == nil && updates.Category == nil && updates.Severity == nil &&
		updates.EffortEstimate == nil && updates.Description == nil && updates.FilePath == nil &&
		updates.Size == nil && !updates.ClearSize && len(updates.Tags) == 0 {
		return fmt.Errorf("at least one update flag is required (--title, --category, --severity, --effort-estimate, --description, --file, --size, or --tag)")
	}

	// Step 2: Call service
	svc := getTechDebtService()
	td, err := svc.UpdateTechDebt(cmd.Context(), key, updates)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(td)
	}
	cli.Success(fmt.Sprintf("Updated tech-debt %s", td.Key))
	return nil
}

// runTdDelete handles the `shark td delete` command.
func runTdDelete(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	force, _ := cmd.Flags().GetBool("force")

	svc := getTechDebtService()

	// Confirm deletion unless --force
	if !force {
		td, err := svc.GetTechDebt(cmd.Context(), key)
		if err != nil {
			return err
		}
		if !confirmTdDelete(td) {
			cli.Info("Delete cancelled")
			return nil
		}
	}

	// Step 2: Call service
	if err := svc.DeleteTechDebt(cmd.Context(), key); err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": key})
	}
	cli.Success(fmt.Sprintf("Deleted tech-debt %s", key))
	return nil
}

// runTdTriage handles the `shark td triage` command.
func runTdTriage(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	severity, _ := cmd.Flags().GetString("severity")
	category, _ := cmd.Flags().GetString("category")
	effortEstimate, _ := cmd.Flags().GetString("effort-estimate")

	input := services.TriageTechDebtInput{
		Severity:       severity,
		Category:       category,
		EffortEstimate: effortEstimate,
	}

	// Step 2: Call service
	svc := getTechDebtService()
	td, err := svc.TriageTechDebt(cmd.Context(), key, input)
	if err != nil {
		return err
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(td)
	}
	cli.Success(fmt.Sprintf("Triaged tech-debt %s (severity: %s, category: %s, status: %s)", td.Key, td.Severity, td.Category, td.Status))
	return nil
}

// --- Helper functions ---

// buildTechDebtBasicInfo assembles the key-value info table for tech-debt display.
func buildTechDebtBasicInfo(td *models.TechDebt) [][]string {
	var info [][]string

	info = append(info, []string{"Title", td.Title})
	info = append(info, []string{"Status", string(td.Status)})
	info = append(info, []string{"Category", string(td.Category)})
	info = append(info, []string{"Severity", string(td.Severity)})

	if td.EffortEstimate != nil && *td.EffortEstimate != "" {
		info = append(info, []string{"Effort Estimate", *td.EffortEstimate})
	}
	if td.Size != nil {
		info = append(info, []string{"Size", formatSize(td.Size)})
	}
	if td.FilePath != nil && *td.FilePath != "" {
		info = append(info, []string{"File", *td.FilePath})
	}
	if td.Description != nil && *td.Description != "" {
		info = append(info, []string{"Description", *td.Description})
	}
	info = append(info, []string{"Created", td.CreatedAt.Format(time.RFC3339)})
	info = append(info, []string{"Updated", td.UpdatedAt.Format(time.RFC3339)})

	return info
}

// printTechDebtTable renders a table for tech-debt list output.
func printTechDebtTable(items []*models.TechDebt) error {
	headers := []string{"KEY", "TITLE", "STATUS", "CATEGORY", "SEVERITY"}
	const titleColIdx = 1

	rows := make([][]string, 0, len(items))
	for _, td := range items {
		rows = append(rows, []string{
			td.Key,
			"", // title placeholder; filled below after width is computed
			string(td.Status),
			string(td.Category),
			string(td.Severity),
		})
	}

	titleMax := availableTitleWidth(cli.GetConsoleWidth(), headers, rows, titleColIdx)
	for i, td := range items {
		rows[i][titleColIdx] = truncateToWidth(td.Title, titleMax)
	}

	cli.OutputTable(headers, rows)
	return nil
}

// confirmTdDelete prompts for confirmation before deleting a tech-debt item.
func confirmTdDelete(td *models.TechDebt) bool {
	fmt.Printf("Delete tech-debt %s: %s? [y/N] ", td.Key, td.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}

// truncateTdString truncates a string to maxLen characters, appending "..." if truncated.
func truncateTdString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
