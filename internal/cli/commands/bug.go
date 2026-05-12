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
	CreateBug(ctx context.Context, input services.CreateBugInput) (*models.Bug, bool, error)
	GetBug(ctx context.Context, key string) (*models.Bug, error)
	// GetBugWithTags returns the bug along with the sorted tag names attached to it.
	// When TagService is nil or unavailable, tags will be nil (graceful degradation).
	GetBugWithTags(ctx context.Context, key string) (*models.Bug, []string, error)
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
	// bugCreateTags and bugUpdateTags back the repeatable --tag flag on
	// `shark bug create` and `shark bug update`. Declared as distinct
	// variables because they live on different commands; Cobra does not
	// reset flag-backing variables between command executions (test code
	// must zero them out to avoid cross-test bleed — see bug_test.go).
	bugCreateTags []string
	bugUpdateTags []string
)

// resolveBugID is the EntityKeyResolver used by the `shark bug tag`
// subcommand factory. It uses the existing bug service accessor (either
// the production service or the test override in bugSvcOverride) to find
// a bug by key and return its numeric ID.
//
// Split out as a package-level function (not a closure in init()) so the
// E28-F04 entity_tag_cmd.go factory can reference it and so tests can
// observe its behaviour through bugSvcOverride.
func resolveBugID(ctx context.Context, key string) (int64, error) {
	svc := getBugService()
	bug, err := svc.GetBug(ctx, key)
	if err != nil {
		return 0, err
	}
	return bug.ID, nil
}

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

	// E28-F04 T-005: register the shared `tag add|rm` subcommand. Svc
	// override is nil in production so it falls through to
	// cli.GetTagService() at call time.
	bugCmd.AddCommand(makeEntityTagCmd(models.EntityTypeBug, resolveBugID, nil))

	// Create flags
	bugCreateCmd.Flags().StringVar(&bugSeverity, "severity", "", "Bug severity (critical, high, medium, low; default: medium)")
	bugCreateCmd.Flags().StringVar(&bugLink, "link", "", "Entity key to link (E07, E07-F01, E07-F01-001)")
	bugCreateCmd.Flags().StringVar(&bugFilePath, "file", "", "Custom file path for bug markdown file")
	bugCreateCmd.Flags().BoolVar(&bugForce, "force", false, "Overwrite existing file at target path")
	// E28-F04 REQ-F-012: repeatable --tag flag. StringSliceVar collects
	// repeated occurrences into a []string; Cobra's comma-separator
	// behaviour means `--tag=foo,bar` becomes ["foo","bar"] — ADR-F04-5
	// accepts this because invalid-name chars (a comma is invalid under
	// the regex) surface as a clear exit-3 validation error.
	bugCreateCmd.Flags().StringSliceVar(&bugCreateTags, "tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	// E07-F42 REQ-F-004: optional size flag (StringVar per Decision D4).
	bugCreateCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")
	bugCreateCmd.Flags().String("content", "", "Pre-populate file body (stdin pipe also accepted)")
	bugCreateCmd.Flags().String("description", "", "Bug description (optional)")
	bugCreateCmd.Flags().String("linked-type", "", "Linked entity type: epic, feature, or task (optional)")
	bugCreateCmd.Flags().String("linked-key", "", "Linked entity key (e.g., E07-F01-001) - requires --linked-type")

	// List flags
	bugListCmd.Flags().StringVar(&bugStatus, "status", "", "Filter by status")
	bugListCmd.Flags().StringVar(&bugSeverity, "severity", "", "Filter by severity")
	bugListCmd.Flags().StringVar(&bugLink, "link", "", "Filter by linked entity key")
	bugListCmd.Flags().Bool("all", false, "Show all bugs including terminal statuses (resolved, wont_fix, duplicate)")
	// E28-F05 REQ-F-010 / REQ-F-018: repeatable --tag flag with AND semantics.
	bugListCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")

	// Update flags
	bugUpdateCmd.Flags().StringVar(&bugTitle, "title", "", "New title")
	bugUpdateCmd.Flags().StringVar(&bugSeverity, "severity", "", "New severity")
	bugUpdateCmd.Flags().String("file", "", "New file path")
	bugUpdateCmd.Flags().String("filename", "", "Alias for --file")
	bugUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = bugUpdateCmd.Flags().MarkHidden("filename")
	_ = bugUpdateCmd.Flags().MarkHidden("path")
	// E28-F04 REQ-F-012: `--tag` on update is additive (REQ-F-010). Empty
	// means no change; no way to detach here — use `shark bug tag rm`.
	bugUpdateCmd.Flags().StringSliceVar(&bugUpdateTags, "tag", nil,
		"Tag to apply additively (repeatable). Empty = no change; use 'shark bug tag rm' to detach.")
	// E07-F42 REQ-F-005: optional size flag with clear-literal support.
	bugUpdateCmd.Flags().String("size", "",
		"Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove on update)")

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
	// E28-F04: read --tag via the flag accessor (not the package-level
	// variable) so test invocations that reset flags between runs see a
	// fresh value each time.
	tags, _ := cmd.Flags().GetStringSlice("tag")

	// E07-F42 REQ-F-004: parse --size before calling service; reject invalid values early.
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

	description, _ := cmd.Flags().GetString("description")
	linkedType, _ := cmd.Flags().GetString("linked-type")
	linkedKey, _ := cmd.Flags().GetString("linked-key")

	input := services.CreateBugInput{
		Title:       args[0],
		Description: description,
		Severity:    models.BugSeverity(severity),
		Force:       force,
		Tags:        tags,
		Size:        sizePtr,
		Body:        body,
	}

	if filePath != "" {
		input.FilePath = &filePath
	}

	// The explicit pair takes precedence; partial pairs fall through to
	// --link so the service still gets a complete linkage instead of an
	// empty key paired with a non-empty type.
	if linkedType != "" && linkedKey != "" {
		input.LinkedEntityType = linkedType
		input.LinkedEntityKey = linkedKey
	} else if link != "" {
		entityType, entityKey := parseBugLinkFlag(link)
		input.LinkedEntityType = entityType
		input.LinkedEntityKey = entityKey
	}

	// Step 2: Call service
	svc := getBugService()
	bug, fileWasLinked, err := svc.CreateBug(cmd.Context(), input)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "bug", input.Title)
	}

	// Step 3: Format output
	projectRoot, _ := cli.FindProjectRoot()
	bugFilePath := bug.GetFilePath()
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cli.FormatEntityCreationJSON("bug", bug.Key, bug.Title, bugFilePath, projectRoot))
	}
	fmt.Print(cli.FormatEntityCreationMessage("bug", bug.Key, bug.Title, bugFilePath, projectRoot, fileWasLinked))
	return nil
}

// runBugGet handles the `shark bug get` command.
func runBugGet(cmd *cobra.Command, args []string) error {
	// Step 1: Parse
	key := args[0]
	ctx := cmd.Context()

	// Step 2: Call service — use GetBugWithTags for tag enrichment (REQ-F-014, REQ-F-015).
	svc := getBugService()
	bug, tags, err := svc.GetBugWithTags(ctx, key)
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
		// REQ-F-015: "tags" field always present, never null.
		if tags == nil {
			tags = []string{}
		}
		result["tags"] = tags
		// B032: size/size_label keys are always present (null when unset) so JSON
		// consumers can rely on `--field size` and `jq has("size")` like they can
		// for epics.
		ensureSizeFieldsAlwaysPresent(result, bug)
		return cli.OutputJSON(result)
	}

	basicInfo := buildBugBasicInfo(bug)
	basicInfo = appendTagsToBasicInfo(basicInfo, tags)
	RenderEntity(EntityDisplayOptions{
		EntityType:         "bug",
		Key:                bug.Key,
		Status:             string(bug.Status),
		BasicInfo:          basicInfo,
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
	// E28-F05 REQ-F-010: read the repeatable --tag flag; nil when absent (AC-T2).
	if rawTags, tagErr := cmd.Flags().GetStringSlice("tag"); tagErr == nil && len(rawTags) > 0 {
		filters.Tags = rawTags
	}

	// Step 2: Call service
	svc := getBugService()
	bugs, err := svc.ListBugs(cmd.Context(), filters)
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "bug", "")
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

	// E28-F04 REQ-F-012: read --tag for additive tagging on update.
	// `cmd.Flags().Changed("tag")` guards the at-least-one-flag check
	// below so passing ONLY --tag still counts as a valid update.
	if cmd.Flags().Changed("tag") {
		tags, _ := cmd.Flags().GetStringSlice("tag")
		updates.Tags = tags
	}

	// E07-F42 REQ-F-005: three-way dispatch for --size on update.
	//   empty → no-op; "clear" → ClearSize=true; valid → Size=ptr(n).
	if cmd.Flags().Changed("size") {
		sizePtr, clearSize, sizeErr := parseSizeUpdateFlag(cmd)
		if sizeErr != nil {
			return sizeErr
		}
		updates.Size = sizePtr
		updates.ClearSize = clearSize
	}

	if updates.Title == nil && updates.Severity == nil && updates.FilePath == nil && updates.Tags == nil && updates.Size == nil && !updates.ClearSize {
		return fmt.Errorf("at least one update flag is required (--title, --severity, --file, --tag, or --size)")
	}

	// Step 2: Call service
	svc := getBugService()
	bug, err := svc.UpdateBug(cmd.Context(), key, updates)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "bug", key)
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
	// E07-F42 REQ-F-006: human display uses "<label> (<num>)" or omits the row entirely.
	if bug.Size != nil {
		info = append(info, []string{"Size", formatSize(bug.Size)})
	}
	info = append(info, []string{"Created", bug.CreatedAt.Format(time.RFC3339)})
	info = append(info, []string{"Updated", bug.UpdatedAt.Format(time.RFC3339)})

	return info
}

var bugListHeaders = []string{"KEY", "TITLE", "STATUS", "SEVERITY", "LINKED TO", "SIZE"}

const bugListTitleColIdx = 1

// buildBugListRows converts a slice of bugs to table rows for list display.
// Extracted for testability (E07-F42 F4 coverage requirement).
//
// Title is truncated to a width derived from the actual rendered widths
// of the other columns, so wider terminals show more title without
// padding the table out artificially when content is short.
func buildBugListRows(bugs []*models.Bug) [][]string {
	rows := make([][]string, 0, len(bugs))
	for _, b := range bugs {
		linkedTo := ""
		if b.LinkedEntityKey != nil {
			linkedTo = *b.LinkedEntityKey
		}
		rows = append(rows, []string{
			b.Key,
			"", // title placeholder; filled below after width is computed
			string(b.Status),
			string(b.Severity),
			linkedTo,
			formatSize(b.Size), // E07-F42 REQ-F-006: Size column
		})
	}

	titleMax := availableTitleWidth(cli.GetConsoleWidth(), bugListHeaders, rows, bugListTitleColIdx)
	for i, b := range bugs {
		rows[i][bugListTitleColIdx] = truncateToWidth(b.Title, titleMax)
	}
	return rows
}

// printBugTable renders a table for bug list output.
func printBugTable(bugs []*models.Bug) error {
	// E07-F42: Size column added to bug list table (REQ-F-006).
	cli.OutputTable(bugListHeaders, buildBugListRows(bugs))
	return nil
}

// confirmBugDelete prompts for confirmation before deleting a bug.
func confirmBugDelete(bug *models.Bug) bool {
	fmt.Printf("Delete bug %s: %s? [y/N] ", bug.Key, bug.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}
