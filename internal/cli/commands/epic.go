package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// resolveEpicID is the EntityKeyResolver used by the `shark epic tag`
// subcommand factory. It looks up an epic by key through the existing
// EpicService accessor (cli.GetEpicService) and returns the numeric ID.
//
// Split out as a package-level function (not a closure in init()) so the
// E28-F04 entity_tag_cmd.go factory can reference it directly.
func resolveEpicID(ctx context.Context, key string) (int64, error) {
	svc := cli.GetEpicService()
	epic, err := svc.GetEpic(ctx, key)
	if err != nil {
		return 0, err
	}
	return epic.ID, nil
}

var epicCmd = &cobra.Command{
	Use:     "epic",
	Short:   "Manage epics",
	GroupID: "advanced",
	Long: `Query and manage epics with automatic progress calculation.

Examples:
  shark epic list          List all epics
  shark epic get E04       Get epic details with progress`,
}

var epicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all epics",
	Long: `List all epics with progress information.

Examples:
  shark epic list           List all epics
  shark epic list --json    Output as JSON`,
	RunE: runEpicList,
}

var epicGetCmd = &cobra.Command{
	Use:   "get <epic-key>",
	Short: "Get epic details",
	Long: `Display detailed information about a specific epic including all features and progress.

Accepts numeric keys (E04) or slugged keys (E04-epic-name).

Examples:
  shark epic get E04              Get epic by key
  shark epic get E04-enhancements Get epic by slugged key
  shark epic get E04 --json       Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicGet,
}

var epicStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show epic status summary",
	Long:  `Display a summary of all epics with completion percentages and task counts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cli.Warning("Not yet implemented - coming in E05-F01")
		return nil
	},
}

var epicCompleteCmd = &cobra.Command{
	Use:   "complete <epic-key>",
	Short: "Complete all tasks in an epic",
	Long: `Mark all tasks in an epic as completed.

Without --force, fails if any tasks are incomplete. Accepts numeric or slugged keys.

Examples:
  shark epic complete E07         Complete epic by key
  shark epic complete E07 --force Force complete all tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicComplete,
}

var epicCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new epic",
	Long: `Create a new epic with auto-assigned key, folder structure, and database entry.

Examples:
  shark epic create "User Auth"
  shark epic create "Platform Roadmap" --file="docs/specs/roadmap.md"
  shark epic create "Q1 Goals" --file="docs/roadmap/q1.md" --force`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicCreate,
}

var epicDeleteCmd = &cobra.Command{
	Use:   "delete <epic-key>",
	Short: "Delete an epic",
	Long: `Delete an epic (and all its features/tasks via CASCADE). Use --force if epic has features.

Examples:
  shark epic delete E05           Delete epic with no features
  shark epic delete E05 --force   Force delete epic with features`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicDelete,
}

var epicUpdateCmd = &cobra.Command{
	Use:   "update <epic-key>",
	Short: "Update an epic",
	Long: `Update an epic's properties: title, description, priority, or file path.

Use 'shark status set' to change status.

Examples:
  shark epic update E01 --title "New Title"
  shark epic update E01 --priority high
  shark epic update E01 --file "docs/roadmap/2025.md"`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicUpdate,
}

var (
	epicCreateDescription string
	epicCreateKey         string
)

func init() {
	epicCmd.Hidden = true // Hidden from top-level help; accessible via 'shark epic'
	cli.RootCmd.AddCommand(epicCmd)
	epicCmd.AddCommand(epicListCmd, epicGetCmd, epicStatusCmd, epicCompleteCmd, epicCreateCmd, epicDeleteCmd, epicUpdateCmd)

	// E28-F04 T-008: register the shared `tag add|rm` subcommand. Svc
	// override is nil in production so it falls through to
	// cli.GetTagService() at call time.
	epicCmd.AddCommand(makeEntityTagCmd(models.EntityTypeEpic, resolveEpicID, nil))

	epicListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	epicListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")
	// E28-F05 REQ-F-010 / REQ-F-018: repeatable --tag flag with AND semantics.
	epicListCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")

	epicCompleteCmd.Flags().Bool("force", false, "Force completion of all tasks regardless of status")

	epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
	epicCreateCmd.Flags().StringVar(&epicCreateKey, "key", "", "Custom key (e.g., E00, bugs). Defaults to auto-generated next E## number")
	epicCreateCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/epic.md)")
	epicCreateCmd.Flags().String("filename", "", "Alias for --file")
	epicCreateCmd.Flags().String("path", "", "Alias for --file")
	_ = epicCreateCmd.Flags().MarkHidden("filename")
	_ = epicCreateCmd.Flags().MarkHidden("path")
	epicCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
	epicCreateCmd.Flags().String("priority", "medium", "Priority: low, medium, high (default: medium)")
	epicCreateCmd.Flags().String("business-value", "", "Business value: low, medium, high (optional)")
	epicCreateCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived (default: draft)")
	// E28-F04 REQ-F-012: repeatable --tag flag. Tag must be registered in
	// the vocabulary (see `shark tags list` / `shark tags add`).
	epicCreateCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	// E07-F42 REQ-F-004: optional size flag (StringVar per Decision D4).
	epicCreateCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")
	epicCreateCmd.Flags().String("content", "", "Pre-populate file body (stdin pipe also accepted)")

	epicDeleteCmd.Flags().Bool("force", false, "Force deletion even if epic has features")

	epicUpdateCmd.Flags().String("title", "", "New title for the epic")
	epicUpdateCmd.Flags().String("description", "", "New description for the epic")
	epicUpdateCmd.Flags().String("priority", "", "New priority: low, medium, high")
	epicUpdateCmd.Flags().String("business-value", "", "New business value: low, medium, high")
	epicUpdateCmd.Flags().String("key", "", "New key for the epic (must be unique, cannot contain spaces)")
	epicUpdateCmd.Flags().String("file", "", "New file path (e.g., docs/custom/epic.md)")
	epicUpdateCmd.Flags().String("filename", "", "Alias for --file")
	epicUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = epicUpdateCmd.Flags().MarkHidden("filename")
	_ = epicUpdateCmd.Flags().MarkHidden("path")
	// E28-F04 REQ-F-012 / REQ-F-010: `--tag` on update is ADDITIVE (no
	// detach). Use `shark epic tag rm` to detach a single tag.
	epicUpdateCmd.Flags().StringSlice("tag", nil,
		"Tag to apply additively (repeatable). Empty = no change; use 'shark epic tag rm' to detach.")
	// E07-F42 REQ-F-005: optional size flag with clear-literal support.
	epicUpdateCmd.Flags().String("size", "",
		"Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove on update)")
}

// runEpicList lists all epics with progress information.
func runEpicList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sortBy, _ := cmd.Flags().GetString("sort-by")
	statusFilter, _ := cmd.Flags().GetString("status")

	if statusFilter != "" {
		validated, err := ParseEpicStatus(statusFilter)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		statusFilter = validated
	}

	if sortBy != "" && sortBy != "key" && sortBy != "progress" && sortBy != "status" {
		cli.Error(fmt.Sprintf("Error: Invalid sort-by '%s'. Must be one of: key, progress, status", sortBy))
		os.Exit(1)
	}

	// E28-F05 REQ-F-010: read the repeatable --tag flag; nil when absent (AC-T2).
	var tagFilter []string
	if rawTags, tagErr := cmd.Flags().GetStringSlice("tag"); tagErr == nil && len(rawTags) > 0 {
		tagFilter = rawTags
	}
	epics, err := cli.GetEpicService().ListEpics(ctx, services.EpicFilters{Status: statusFilter, Tags: tagFilter})
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "epic", "")
	}

	if len(epics) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{"results": []interface{}{}, "count": 0})
		}
		cli.Info("No epics found")
		return nil
	}

	epicsWithProgress := buildEpicsWithProgress(ctx, epics)
	sortEpics(epicsWithProgress, sortBy)

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{"results": epicsWithProgress, "count": len(epicsWithProgress)})
	}
	renderEpicListTable(epicsWithProgress)
	return nil
}

// runEpicGet retrieves and displays a specific epic with its details.
func runEpicGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := args[0]
	// Use GetEpicWithTags for tag enrichment (REQ-F-014, REQ-F-015).
	epicSvc := cli.GetEpicService()
	epic, tags, err := epicSvc.GetEpicWithTags(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	// REQ-F-015: JSON always has a "tags" key, never null.
	jsonTags := tags
	if jsonTags == nil {
		jsonTags = []string{}
	}

	displaySvc := cli.GetDisplayService()
	displayMode := displaySvc.DetermineEpicDisplayMode(epic)

	if displayMode == services.DisplayModePlanning {
		info, err := displaySvc.GetEpicDisplayInfo(ctx, epicKey)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to get epic display info: %v", err))
			os.Exit(2)
		}
		info.ResolvedPath = resolveEpicPlanningPath(ctx, epic.Key)

		// Populate valid transitions for planning mode
		workflowCfg := cli.GetWorkflowService().ForLevel("epic").GetWorkflow()
		info.ValidTransitions = GetValidTransitions(string(epic.Status), workflowCfg)
		info.OrchestratorAction = displaySvc.ResolveEpicAction(ctx, epic)

		if cli.GlobalConfig.JSON {
			// Add "tags" to the planning JSON envelope.
			infoJSON, marshalErr := json.Marshal(info)
			if marshalErr != nil {
				return marshalErr
			}
			var infoMap map[string]interface{}
			if unmarshalErr := json.Unmarshal(infoJSON, &infoMap); unmarshalErr != nil {
				return unmarshalErr
			}
			infoMap["tags"] = jsonTags
			// B032: always surface size and size_label so `shark get <epic> --json`
			// is consistent across entity types (null when unset).
			// E07-F42 REQ-F-006/007: inject size and size_label at the top level so
			// that --field size and --field size_label work for planning-mode epics.
			// The struct marshals size inside the nested "epic" key; we mirror
			// the aggregation-mode pattern by also surfacing them at the top level.
			if epic.Size != nil {
				infoMap["size"] = *epic.Size
				if label, err := models.SizeLabel(*epic.Size); err == nil {
					infoMap["size_label"] = label
				} else {
					infoMap["size_label"] = nil
				}
			} else {
				infoMap["size"] = nil
				infoMap["size_label"] = nil
			}
			return cli.OutputJSON(infoMap)
		}
		renderEpicPlanningWithTags(info, tags)
		return nil
	}

	data, err := buildEpicGetData(ctx, epic)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			slog.Error("Failed to build epic data", "error", err)
		}
		os.Exit(2)
	}

	orchestratorAction := displaySvc.ResolveEpicAction(ctx, epic)
	if cli.GlobalConfig.JSON {
		result := buildEpicGetJSON(epic, data, orchestratorAction)
		result["tags"] = jsonTags
		return cli.OutputJSON(result)
	}
	renderEpicDetailsWithTags(epic, data, orchestratorAction, tags)
	return nil
}

// runEpicCreate creates a new epic.
func runEpicCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return performEpicCreate(ctx, args[0], cmd)
}

// runEpicComplete marks all tasks in an epic as completed.
func runEpicComplete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	force, _ := cmd.Flags().GetBool("force")
	return performEpicComplete(ctx, args[0], force)
}

// runEpicDelete deletes an epic and all its children.
func runEpicDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	force, _ := cmd.Flags().GetBool("force")
	return performEpicDelete(ctx, args[0], force)
}

// runEpicUpdate updates an epic's properties.
func runEpicUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return performEpicUpdate(ctx, args[0], cmd)
}
