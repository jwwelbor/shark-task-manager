package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/spf13/cobra"
)

// resolveFeatureID is the EntityKeyResolver used by the `shark feature tag`
// subcommand factory. It looks up a feature by key through the existing
// FeatureService accessor (cli.GetFeatureService) and returns the numeric
// ID.
//
// Split out as a package-level function (not a closure in init()) so the
// E28-F04 entity_tag_cmd.go factory can reference it directly.
func resolveFeatureID(ctx context.Context, key string) (int64, error) {
	svc := cli.GetFeatureService()
	feature, err := svc.GetFeature(ctx, key)
	if err != nil {
		return 0, err
	}
	return feature.ID, nil
}

// featureCmd represents the feature command group.
var featureCmd = &cobra.Command{
	Use:     "feature",
	Short:   "Manage features",
	GroupID: "advanced",
	Long: `Query and manage features within epics.

Examples:
  shark feature list              List all features
  shark feature get E04-F02      Get feature details
  shark feature list --epic=E04  List features in epic E04`,
}

// featureListCmd lists features.
var featureListCmd = &cobra.Command{
	Use:   "list [EPIC]",
	Short: "List features",
	Long: `List features with optional filtering by epic.

Examples:
  shark feature list              List all non-completed features
  shark feature list --all        List all features including completed
  shark feature list E04          List features in epic E04
  shark feature list --json       Output as JSON
  shark feature list --sort-by=progress  Sort by progress`,
	RunE: runFeatureList,
}

// featureGetCmd gets a specific feature.
var featureGetCmd = &cobra.Command{
	Use:   "get <feature-key>",
	Short: "Get feature details",
	Long: `Display detailed information about a specific feature.

Supports key formats: E04-F02, F02, F02-feature-name, E04-F02-feature-name

Examples:
  shark feature get E04-F02       Get feature by full key
  shark feature get F02           Get feature by numeric key
  shark feature get E04-F02 --json  Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureGet,
}

// featureCreateCmd creates a new feature.
var featureCreateCmd = &cobra.Command{
	Use:   "create [EPIC] <title> [flags]",
	Short: "Create a new feature",
	Long: `Create a new feature with auto-assigned key and folder structure.

Examples:
  shark feature create E01 "OAuth Login Integration"
  shark feature create E07 "User Authentication" --description="Add OAuth 2.0 support"
  shark feature create --epic=E01 "OAuth Login"
  shark feature create --epic=E01 --file="docs/specs/auth.md" "OAuth Login"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runFeatureCreate,
}

// featureCompleteCmd completes all tasks in a feature.
var featureCompleteCmd = &cobra.Command{
	Use:   "complete <feature-key>",
	Short: "Complete all tasks in a feature",
	Long: `Mark all tasks in a feature as completed.

Without --force, fails if any tasks are incomplete.
With --force, completes all tasks regardless of status.

Examples:
  shark feature complete E04-F02        Complete feature by full key
  shark feature complete E04-F02 --force  Force complete all tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureComplete,
}

// featureDeleteCmd deletes a feature.
var featureDeleteCmd = &cobra.Command{
	Use:   "delete <feature-key>",
	Short: "Delete a feature",
	Long: `Delete a feature from the database (CASCADE deletes all tasks).

WARNING: This action cannot be undone.

Examples:
  shark feature delete E04-F02          Delete feature with no tasks
  shark feature delete E04-F02 --force  Force delete with tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureDelete,
}

// featureUpdateCmd updates a feature's properties.
var featureUpdateCmd = &cobra.Command{
	Use:   "update <feature-key>",
	Short: "Update a feature",
	Long: `Update a feature's properties such as title, description, or execution order.

Use 'shark status set' to change status.

Examples:
  shark feature update E04-F02 --title "New Title"
  shark feature update E04-F02 --order 2`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureUpdate,
}

var (
	featureCreateEpic           string
	featureCreateDescription    string
	featureCreateExecutionOrder int
	featureCreateForce          bool
	featureCreateKey            string
)

func init() {
	featureCmd.Hidden = true // Hidden from top-level help; accessible via 'shark feature'
	cli.RootCmd.AddCommand(featureCmd)

	featureCmd.AddCommand(featureListCmd)
	featureCmd.AddCommand(featureGetCmd)
	featureCmd.AddCommand(featureCreateCmd)
	featureCmd.AddCommand(featureCompleteCmd)
	featureCmd.AddCommand(featureDeleteCmd)
	featureCmd.AddCommand(featureUpdateCmd)

	// E28-F04 T-007: register the shared `tag add|rm` subcommand. Svc
	// override is nil in production so it falls through to
	// cli.GetTagService() at call time.
	featureCmd.AddCommand(makeEntityTagCmd(models.EntityTypeFeature, resolveFeatureID, nil))

	// List flags
	featureListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	featureListCmd.Flags().String("status", "", "Filter by status")
	featureListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status")
	featureListCmd.Flags().Bool("show-all", false, "Show all features including completed")
	_ = featureListCmd.Flags().MarkDeprecated("show-all", "use --all instead")
	featureListCmd.Flags().Bool("all", false, "Show all features including completed")
	// E28-F05 REQ-F-010 / REQ-F-018: repeatable --tag flag with AND semantics.
	featureListCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")

	// Create flags
	featureCreateCmd.Flags().StringVar(&featureCreateEpic, "epic", "", "Epic key (e.g., E01)")
	featureCreateCmd.Flags().StringVar(&featureCreateDescription, "description", "", "Feature description")
	featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order")
	_ = featureCreateCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
	featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "order", 0, "Execution order (lower runs first)")
	featureCreateCmd.Flags().StringVar(&featureCreateKey, "key", "", "Custom key for the feature")
	featureCreateCmd.Flags().BoolVar(&featureCreateForce, "force", false, "Force reassignment if file already claimed")
	featureCreateCmd.Flags().String("status", "draft", "Status (default: draft)")
	featureCreateCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/feature.md)")
	featureCreateCmd.Flags().String("filename", "", "Alias for --file")
	featureCreateCmd.Flags().String("path", "", "Alias for --file")
	_ = featureCreateCmd.Flags().MarkHidden("filename")
	_ = featureCreateCmd.Flags().MarkHidden("path")
	// E28-F04 REQ-F-012: repeatable --tag flag. Tag must be registered in
	// the vocabulary (see `shark tags list` / `shark tags add`).
	featureCreateCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")

	// Complete flags
	featureCompleteCmd.Flags().Bool("force", false, "Force completion of all tasks")

	// Delete flags
	featureDeleteCmd.Flags().Bool("force", false, "Force deletion even if feature has tasks")

	// Update flags
	featureUpdateCmd.Flags().String("title", "", "New title")
	featureUpdateCmd.Flags().String("description", "", "New description")
	featureUpdateCmd.Flags().Int("execution-order", -1, "New execution order (-1 = no change)")
	_ = featureUpdateCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
	featureUpdateCmd.Flags().Int("order", -1, "New execution order (-1 = no change)")
	featureUpdateCmd.Flags().String("key", "", "New key (must be unique, no spaces)")
	featureUpdateCmd.Flags().String("file", "", "New file path")
	featureUpdateCmd.Flags().String("filename", "", "Alias for --file")
	featureUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = featureUpdateCmd.Flags().MarkHidden("filename")
	_ = featureUpdateCmd.Flags().MarkHidden("path")
	// E28-F04 REQ-F-012 / REQ-F-010: `--tag` on update is ADDITIVE (no
	// detach). Use `shark feature tag rm` to detach a single tag.
	featureUpdateCmd.Flags().StringSlice("tag", nil,
		"Tag to apply additively (repeatable). Empty = no change; use 'shark feature tag rm' to detach.")
}

// runFeatureList lists features with optional epic and status filtering.
func runFeatureList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicFilter, statusFilter, sortBy, showAll, tagFilter, err := parseFeatureListFlags(cmd, args)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	featuresWithTaskCount, err := fetchFeaturesWithTaskCount(ctx, epicFilter, statusFilter, showAll, tagFilter)
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "feature", "")
	}

	if len(featuresWithTaskCount) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{"results": []interface{}{}, "count": 0})
		}
		message := "No features found"
		if epicFilter != "" {
			message = fmt.Sprintf("No features found for epic %s", epicFilter)
		} else if statusFilter != "" {
			message = fmt.Sprintf("No features found with status %s", statusFilter)
		}
		cli.Info(message)
		return nil
	}

	sortFeatures(featuresWithTaskCount, sortBy)

	if cli.GlobalConfig.JSON {
		return outputFeatureListJSON(ctx, featuresWithTaskCount)
	}
	renderFeatureListTable(featuresWithTaskCount, epicFilter, ctx)
	return nil
}

// runFeatureGet displays detailed information about a feature.
func runFeatureGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]
	// Use GetFeatureWithTags for tag enrichment (REQ-F-014, REQ-F-015).
	featureSvc := cli.GetFeatureService()
	feature, tags, err := featureSvc.GetFeatureWithTags(ctx, featureKey)
	if err != nil {
		handleServiceError(err, "feature", featureKey)
		return nil
	}

	// REQ-F-015: JSON always has a "tags" key, never null.
	jsonTags := tags
	if jsonTags == nil {
		jsonTags = []string{}
	}

	displaySvc := cli.GetDisplayService()

	if displaySvc.DetermineFeatureDisplayMode(feature) == "planning" {
		info, err := displaySvc.GetFeatureDisplayInfo(ctx, featureKey)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to get feature display info: %v", err))
			os.Exit(2)
		}
		info.ResolvedPath = resolveFeaturePath(ctx, feature)
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
			return cli.OutputJSON(infoMap)
		}
		renderFeaturePlanningWithTags(info, tags)
		return nil
	}

	data, err := buildFeatureGetData(ctx, feature)
	if err != nil {
		handleServiceError(err, "feature", featureKey)
		return nil
	}

	orchestratorAction := displaySvc.ResolveFeatureAction(ctx, feature)
	if cli.GlobalConfig.JSON {
		result := buildFeatureGetJSON(feature, data, orchestratorAction)
		result["tags"] = jsonTags
		return cli.OutputJSON(result)
	}
	renderFeatureAggregationWithTags(feature, data, orchestratorAction, tags)
	return nil
}

// runFeatureCreate creates a new feature and its markdown file.
func runFeatureCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input, featureTitle, projectRoot, err := parseCreateFeatureInput(cmd, args)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	featureSvc := cli.GetFeatureService()
	feature, err := featureSvc.CreateFeature(ctx, input)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "feature", input.EpicKey)
	}

	featureFilePath := resolveFeatureFilePath(feature, input.EpicKey, projectRoot)
	featureSlug := fmt.Sprintf("%s-%s", feature.Key, utils.GenerateSlug(featureTitle))
	content, err := renderFeatureTemplate(input.EpicKey, feature.Key, featureSlug, featureTitle, featureCreateDescription, featureFilePath)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	fileWasLinked, err := writeFeatureFile(content, featureFilePath, projectRoot)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	requiredSections := cli.GetRequiredSectionsForEntityType("feature")
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(cli.FormatEntityCreationJSON("feature", feature.Key, featureTitle, featureFilePath, projectRoot, requiredSections))
	}
	fmt.Print(cli.FormatEntityCreationMessage("feature", feature.Key, featureTitle, featureFilePath, projectRoot, fileWasLinked, requiredSections))
	return nil
}

// runFeatureComplete completes all tasks in a feature.
func runFeatureComplete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]
	force, _ := cmd.Flags().GetBool("force")

	if err := performFeatureComplete(ctx, featureKey, force); err != nil {
		handleServiceError(err, "feature", featureKey)
	}
	return nil
}

// runFeatureDelete deletes a feature and all its tasks.
func runFeatureDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]
	force, _ := cmd.Flags().GetBool("force")

	if err := performFeatureDelete(ctx, featureKey, force); err != nil {
		handleServiceError(err, "feature", featureKey)
	}
	return nil
}

// featureUpdateImpl is the function called by runFeatureUpdate to perform the
// update. It is a package-level variable so tests can override it without
// touching the real database.
var featureUpdateImpl = func(ctx context.Context, featureKey string, cmd *cobra.Command) error {
	return performFeatureUpdate(ctx, featureKey, cmd)
}

// runFeatureUpdate updates a feature's properties.
func runFeatureUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]

	return featureUpdateImpl(ctx, featureKey, cmd)
}
