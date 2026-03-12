package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/spf13/cobra"
)

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

	// List flags
	featureListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	featureListCmd.Flags().String("status", "", "Filter by status")
	featureListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status")
	featureListCmd.Flags().Bool("show-all", false, "Show all features including completed")
	_ = featureListCmd.Flags().MarkDeprecated("show-all", "use --all instead")
	featureListCmd.Flags().Bool("all", false, "Show all features including completed")

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
}

// runFeatureList lists features with optional epic and status filtering.
func runFeatureList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicFilter, statusFilter, sortBy, showAll, err := parseFeatureListFlags(cmd, args)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	featuresWithTaskCount, err := fetchFeaturesWithTaskCount(ctx, epicFilter, statusFilter, showAll)
	if err != nil {
		handleServiceError(err, "feature", "")
		return nil
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
	featureSvc := cli.GetFeatureService()
	feature, err := featureSvc.GetFeature(ctx, featureKey)
	if err != nil {
		handleServiceError(err, "feature", featureKey)
		return nil
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
			return cli.OutputJSON(info)
		}
		renderFeaturePlanning(info)
		return nil
	}

	data, err := buildFeatureGetData(ctx, feature)
	if err != nil {
		handleServiceError(err, "feature", featureKey)
		return nil
	}

	orchestratorAction := displaySvc.ResolveFeatureAction(ctx, feature)
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(buildFeatureGetJSON(feature, data, orchestratorAction))
	}
	renderFeatureAggregation(feature, data, orchestratorAction)
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
		handleServiceError(err, "feature", input.EpicKey)
		return nil
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

// runFeatureUpdate updates a feature's properties.
func runFeatureUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]

	if err := performFeatureUpdate(ctx, featureKey, cmd); err != nil {
		handleServiceError(err, "feature", featureKey)
	}
	return nil
}
