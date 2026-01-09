package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/pathresolver"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// getRelativePathFeature converts an absolute path to relative path from project root
func getRelativePathFeature(absPath string, projectRoot string) string {
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return absPath // Fall back to absolute path if conversion fails
	}
	return relPath
}

// backupDatabaseOnForceFeature creates a backup when --force flag is used
// Returns the backup path and error (if any)
// DEPRECATED: Use CreateBackupIfForce from file_assignment.go instead
func backupDatabaseOnForceFeature(force bool, dbPath string, operation string) (string, error) {
	backupPath, err := CreateBackupIfForce(force, dbPath, operation)
	if err != nil {
		return "", err
	}

	if backupPath != "" && !cli.GlobalConfig.JSON {
		cli.Info(fmt.Sprintf("Database backup created: %s", backupPath))
	}

	return backupPath, nil
}

// FeatureWithTaskCount wraps a Feature with task count
type FeatureWithTaskCount struct {
	*models.Feature
	TaskCount    int    `json:"task_count"`
	StatusSource string `json:"status_source"`
}

// featureCmd represents the feature command group
var featureCmd = &cobra.Command{
	Use:     "feature",
	Short:   "Manage features",
	GroupID: "essentials",
	Long: `Query and manage features within epics.

Examples:
  shark feature list              List all features
  shark feature get E04-F02      Get feature details
  shark feature list --epic=E04  List features in epic E04`,
}

// featureListCmd lists features
var featureListCmd = &cobra.Command{
	Use:   "list [EPIC]",
	Short: "List features",
	Long: `List features with optional filtering by epic.

By default, completed features are hidden. Use --show-all to include them.

Positional Arguments:
  EPIC    Optional epic key (E##) to filter features (e.g., E04)

Examples:
  shark feature list              List all non-completed features
  shark feature list --show-all   List all features including completed
  shark feature list E04          List non-completed features in epic E04
  shark feature list --epic=E04   Same as above (flag syntax still works)
  shark feature list --json       Output as JSON
  shark feature list --status=active  Filter by status
  shark feature list --status=completed  List only completed features
  shark feature list --sort-by=progress  Sort by progress`,
	RunE: runFeatureList,
}

// featureGetCmd gets a specific feature
var featureGetCmd = &cobra.Command{
	Use:   "get <feature-key>",
	Short: "Get feature details",
	Long: `Display detailed information about a specific feature including all tasks and progress.

Supports multiple key formats:
  - Full key: E04-F02
  - Numeric key: F02
  - Slugged key: F02-feature-name
  - Full key with slug: E04-F02-feature-name

Examples:
  shark feature get E04-F02                        Get feature by full key
  shark feature get F02                            Get feature by numeric key
  shark feature get F02-user-auth                  Get feature by slugged key
  shark feature get E04-F02-user-auth              Get feature by full key with slug
  shark feature get E04-F02 --json                 Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureGet,
}

// featureCreateCmd creates a new feature
var featureCreateCmd = &cobra.Command{
	Use:   "create [EPIC] <title> [flags]",
	Short: "Create a new feature",
	Long: `Create a new feature with auto-assigned key, folder structure, and database entry.

The feature key is automatically assigned as the next available F## number within the epic.
By default, the feature file is created at docs/plan/{epic-key}/{feature-key}/feature.md.

Positional Arguments:
  EPIC    Optional epic key (E##) - can also be specified with --epic flag
  TITLE   Feature title (required)

Examples:
  # Positional argument syntax (new, recommended)
  shark feature create E01 "OAuth Login Integration"
  shark feature create E07 "User Authentication" --description="Add OAuth 2.0 support"

  # Flag syntax (still supported for backward compatibility)
  shark feature create --epic=E01 "OAuth Login Integration"
  shark feature create --epic=E01 "OAuth Login" --description="Add OAuth 2.0 support"
  shark feature create --epic=E01 --file="docs/specs/auth.md" "OAuth Login"
  shark feature create --epic=E01 --file="docs/specs/auth.md" --force "OAuth Login"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runFeatureCreate,
}

// featureCompleteCmd completes all tasks in a feature
var featureCompleteCmd = &cobra.Command{
	Use:   "complete <feature-key>",
	Short: "Complete all tasks in a feature",
	Long: `Mark all tasks in a feature as completed, with safeguards against accidental completion.

Without --force, shows a warning summary if any tasks are incomplete and fails.
With --force, completes all tasks regardless of status.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark feature complete E04-F02                   Complete feature by full key
  shark feature complete F02                       Complete feature by numeric key
  shark feature complete F02-user-auth             Complete feature by slugged key
  shark feature complete E04-F02 --force           Force complete all tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureComplete,
}

// featureDeleteCmd deletes a feature
var featureDeleteCmd = &cobra.Command{
	Use:   "delete <feature-key>",
	Short: "Delete a feature",
	Long: `Delete a feature from the database (and all its tasks via CASCADE).

WARNING: This action cannot be undone. All tasks under this feature will also be deleted.
If the feature has tasks, you must use --force to confirm the cascade deletion.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark feature delete E04-F02                     Delete feature with no tasks
  shark feature delete F02                         Delete feature by numeric key
  shark feature delete F02-user-auth               Delete feature by slugged key
  shark feature delete E04-F02 --force             Force delete feature with tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureDelete,
}

// featureUpdateCmd updates a feature's properties
var featureUpdateCmd = &cobra.Command{
	Use:   "update <feature-key>",
	Short: "Update a feature",
	Long: `Update a feature's properties such as title, description, status, execution order, or file path.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark feature update E04-F02 --title "New Title"
  shark feature update F02 --description "New description"
  shark feature update F02-user-auth --status active
  shark feature update E04-F02 --execution-order 2
  shark feature update E04-F02 --filename "docs/specs/feature.md"
  shark feature update E04-F02 --path "docs/custom"`,
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
	// Register feature command with root
	cli.RootCmd.AddCommand(featureCmd)

	// Add subcommands
	featureCmd.AddCommand(featureListCmd)
	featureCmd.AddCommand(featureGetCmd)
	featureCmd.AddCommand(featureCreateCmd)
	featureCmd.AddCommand(featureCompleteCmd)
	featureCmd.AddCommand(featureDeleteCmd)
	featureCmd.AddCommand(featureUpdateCmd)

	// Add flags for list command
	featureListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	featureListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")
	featureListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	featureListCmd.Flags().Bool("show-all", false, "Show all features including completed (by default, completed features are hidden)")

	// Add flags for create command
	featureCreateCmd.Flags().StringVar(&featureCreateEpic, "epic", "", "Epic key (e.g., E01) - can also be specified as first positional argument")
	featureCreateCmd.Flags().StringVar(&featureCreateDescription, "description", "", "Feature description (optional)")
	featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order (optional, 0 = not set)")
	featureCreateCmd.Flags().StringVar(&featureCreateKey, "key", "", "Custom key for the feature (e.g., auth, F00). If not provided, auto-generates next F## number")
	featureCreateCmd.Flags().BoolVar(&featureCreateForce, "force", false, "Force reassignment if file already claimed by another feature or epic")
	featureCreateCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived (default: draft)")

	// File path flags: --file is primary, --filename and --path are hidden aliases
	featureCreateCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/feature.md)")
	featureCreateCmd.Flags().String("filename", "", "Alias for --file")
	featureCreateCmd.Flags().String("path", "", "Alias for --file")
	_ = featureCreateCmd.Flags().MarkHidden("filename")
	_ = featureCreateCmd.Flags().MarkHidden("path")

	// Note: --epic flag is no longer required since it can be specified positionally

	// Add flags for complete command
	featureCompleteCmd.Flags().Bool("force", false, "Force completion of all tasks regardless of status")

	// Add flags for delete command
	featureDeleteCmd.Flags().Bool("force", false, "Force deletion even if feature has tasks")

	// Add flags for update command
	featureUpdateCmd.Flags().String("title", "", "New title for the feature")
	featureUpdateCmd.Flags().String("description", "", "New description for the feature")
	featureUpdateCmd.Flags().String("status", "", "New status: draft, active, completed, archived")
	featureUpdateCmd.Flags().Int("execution-order", -1, "New execution order (-1 = no change)")
	featureUpdateCmd.Flags().String("key", "", "New key for the feature (must be unique, cannot contain spaces)")
	featureUpdateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")

	// File path flags: --file is primary, --filename and --path are hidden aliases
	featureUpdateCmd.Flags().String("file", "", "New file path (e.g., docs/custom/feature.md)")
	featureUpdateCmd.Flags().String("filename", "", "Alias for --file")
	featureUpdateCmd.Flags().String("path", "", "Alias for --file")
	_ = featureUpdateCmd.Flags().MarkHidden("filename")
	_ = featureUpdateCmd.Flags().MarkHidden("path")
}

// runFeatureList executes the feature list command
func runFeatureList(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse positional arguments first
	positionalEpic, err := ParseFeatureListArgs(args)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	// Get flags
	epicFilter, _ := cmd.Flags().GetString("epic")
	statusFilter, _ := cmd.Flags().GetString("status")
	sortBy, _ := cmd.Flags().GetString("sort-by")

	// Positional argument takes priority over flag
	if positionalEpic != nil {
		epicFilter = *positionalEpic
	}

	// Validate status filter using shared parsing function
	if statusFilter != "" {
		validatedStatus, err := ParseFeatureStatus(statusFilter)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		statusFilter = validatedStatus
	}

	// Validate sort-by option
	if sortBy != "" && sortBy != "key" && sortBy != "progress" && sortBy != "status" {
		cli.Error(fmt.Sprintf("Error: Invalid sort-by '%s'. Must be one of: key, progress, status", sortBy))
		os.Exit(1)
	}

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get repositories
	featureRepo := repository.NewFeatureRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)

	var features []*models.Feature

	// Apply filters using repository methods
	if epicFilter != "" {
		// Get epic by key
		epic, err := epicRepo.GetByKey(ctx, epicFilter)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicFilter))
			cli.Info("Use 'shark epic list' to see available epics")
			os.Exit(1)
		}

		// Use combined filter if status is specified
		if statusFilter != "" {
			status := models.FeatureStatus(statusFilter)
			features, err = featureRepo.ListByEpicAndStatus(ctx, epic.ID, status)
		} else {
			features, err = featureRepo.ListByEpic(ctx, epic.ID)
		}

		if err != nil {
			cli.Error("Error: Database error. Run with --verbose for details.")
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Failed to list features: %v\n", err)
			}
			os.Exit(2)
		}
	} else if statusFilter != "" {
		// Use status filter only
		status := models.FeatureStatus(statusFilter)
		features, err = featureRepo.ListByStatus(ctx, status)
		if err != nil {
			cli.Error("Error: Database error. Run with --verbose for details.")
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Failed to list features: %v\n", err)
			}
			os.Exit(2)
		}
	} else {
		// Get all features
		features, err = featureRepo.List(ctx)
		if err != nil {
			cli.Error("Error: Database error. Run with --verbose for details.")
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Failed to list features: %v\n", err)
			}
			os.Exit(2)
		}
	}

	// Handle empty results
	if len(features) == 0 {
		message := "No features found"
		if epicFilter != "" {
			message = fmt.Sprintf("No features found for epic %s", epicFilter)
		}
		if statusFilter != "" {
			message = fmt.Sprintf("No features found with status %s", statusFilter)
		}
		if cli.GlobalConfig.JSON {
			result := map[string]interface{}{
				"results": []interface{}{},
				"count":   0,
			}
			return cli.OutputJSON(result)
		}
		cli.Info(message)
		return nil
	}

	// Update progress and add task count for each feature
	featuresWithTaskCount := make([]FeatureWithTaskCount, 0, len(features))
	for _, feature := range features {
		// Update feature progress (in case it's stale)
		if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update progress for feature %s: %v\n", feature.Key, err)
			}
		}

		// Get updated feature
		feature, err = featureRepo.GetByID(ctx, feature.ID)
		if err != nil {
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get feature %s: %v\n", feature.Key, err)
			}
			continue
		}

		// Get task count using repository method
		taskCount, err := featureRepo.GetTaskCount(ctx, feature.ID)
		if err != nil {
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get task count for feature %s: %v\n", feature.Key, err)
			}
			taskCount = 0
		}

		// Determine status source
		statusSource := "calculated"
		if feature.StatusOverride {
			statusSource = "manual"
		}

		featuresWithTaskCount = append(featuresWithTaskCount, FeatureWithTaskCount{
			Feature:      feature,
			TaskCount:    taskCount,
			StatusSource: statusSource,
		})
	}

	// Filter out completed features by default (unless --show-all or explicit status filter)
	showAll, _ := cmd.Flags().GetBool("show-all")
	featuresWithTaskCount = filterFeaturesByCompletedStatus(featuresWithTaskCount, showAll, statusFilter)

	// Apply sorting
	sortFeatures(featuresWithTaskCount, sortBy)

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		result := map[string]interface{}{
			"results": featuresWithTaskCount,
			"count":   len(featuresWithTaskCount),
		}
		return cli.OutputJSON(result)
	}

	// Output as table
	renderFeatureListTable(featuresWithTaskCount, epicFilter)
	return nil
}

// runFeatureGet executes the feature get command
func runFeatureGet(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get project root for WorkflowService
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}

	// Create WorkflowService for status formatting
	workflowService := workflow.NewService(projectRoot)

	// Get repositories
	featureRepo := repository.NewFeatureRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)
	documentRepo := repository.NewDocumentRepository(repoDb)

	// Get feature by key
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'shark feature list' to see available features")
		os.Exit(1)
	}

	// Resolve feature path using PathResolver
	var resolvedPath string
	if projectRoot != "" {
		pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
		absPath, err := pathResolver.ResolveFeaturePath(ctx, feature.Key)
		if err == nil {
			resolvedPath = getRelativePathFeature(absPath, projectRoot)
		} else if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to resolve feature path: %v\n", err)
		}
	}

	// Update feature progress
	if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update progress for feature %s: %v\n", feature.Key, err)
		}
	}

	// Get updated feature
	feature, err = featureRepo.GetByID(ctx, feature.ID)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to get feature: %v\n", err)
		}
		os.Exit(2)
	}

	// Get tasks for this feature
	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to list tasks: %v\n", err)
		}
		os.Exit(2)
	}

	// Get task status breakdown from repository
	statusBreakdown, err := taskRepo.GetStatusBreakdown(ctx, feature.ID)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to get status breakdown: %v\n", err)
		}
		os.Exit(2)
	}

	// Extract directory path and filename
	var dirPath, filename string
	if resolvedPath != "" {
		dirPath = filepath.Dir(resolvedPath) + "/"
		filename = filepath.Base(resolvedPath)
	}

	// Get related documents
	relatedDocs, err := documentRepo.ListForFeature(ctx, feature.ID)
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to fetch related documents: %v\n", err)
	}
	if relatedDocs == nil {
		relatedDocs = []*models.Document{}
	}

	// Determine status source
	statusSource := "calculated"
	if feature.StatusOverride {
		statusSource = "manual"
	}

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		result := map[string]interface{}{
			"id":                feature.ID,
			"epic_id":           feature.EpicID,
			"key":               feature.Key,
			"title":             feature.Title,
			"description":       feature.Description,
			"status":            feature.Status,
			"status_source":     statusSource,
			"status_override":   feature.StatusOverride,
			"progress_pct":      feature.ProgressPct,
			"path":              dirPath,
			"filename":          filename,
			"created_at":        feature.CreatedAt,
			"updated_at":        feature.UpdatedAt,
			"tasks":             tasks,
			"status_breakdown":  statusBreakdown,
			"related_documents": relatedDocs,
		}
		return cli.OutputJSON(result)
	}

	// Output as formatted text
	renderFeatureDetails(feature, tasks, statusBreakdown, dirPath, filename, relatedDocs, workflowService)
	return nil
}

// renderFeatureListTable renders features as a table
func renderFeatureListTable(features []FeatureWithTaskCount, epicFilter string) {
	// Create table data
	tableData := pterm.TableData{
		{"Key", "Title", "Epic ID", "Status", "Progress", "Tasks", "Order"},
	}

	for _, feature := range features {
		// Truncate long titles to fit in 80 columns
		title := feature.Title
		if len(title) > 25 {
			title = title[:22] + "..."
		}

		// Format progress with 1 decimal place
		progress := fmt.Sprintf("%.1f%%", feature.ProgressPct)

		// Format execution_order (show "-" if NULL)
		execOrder := "-"
		if feature.ExecutionOrder != nil {
			execOrder = fmt.Sprintf("%d", *feature.ExecutionOrder)
		}

		// Format status with indicator (* for manual override)
		statusDisplay := string(feature.Status)
		if feature.StatusOverride {
			statusDisplay += "*"
		}

		tableData = append(tableData, []string{
			feature.Key,
			title,
			fmt.Sprintf("%d", feature.EpicID),
			statusDisplay,
			progress,
			fmt.Sprintf("%d", feature.TaskCount),
			execOrder,
		})
	}

	// Render table
	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// renderFeatureDetails renders feature details with tasks table
// statusBreakdown is workflow-ordered with metadata
// workflowService is used for color formatting (can be nil for no colors)
func renderFeatureDetails(feature *models.Feature, tasks []*models.Task, statusBreakdown []workflow.StatusCount, path, filename string, relatedDocs []*models.Document, workflowService *workflow.Service) {
	// Determine if colors should be enabled
	colorEnabled := !cli.GlobalConfig.NoColor && workflowService != nil

	// Print feature metadata
	pterm.DefaultSection.Printf("Feature: %s", feature.Key)
	fmt.Println()

	// Format feature status with color if available
	featureStatusDisplay := string(feature.Status)
	if colorEnabled {
		formatted := workflowService.FormatStatusForDisplay(string(feature.Status), true)
		featureStatusDisplay = formatted.Colored
	}
	if feature.StatusOverride {
		featureStatusDisplay += " (manual override)"
	} else {
		featureStatusDisplay += " (calculated)"
	}

	// Feature info
	info := [][]string{
		{"Title", feature.Title},
		{"Epic ID", fmt.Sprintf("%d", feature.EpicID)},
		{"Status", featureStatusDisplay},
		{"Progress", fmt.Sprintf("%.1f%%", feature.ProgressPct)},
	}

	if path != "" {
		info = append(info, []string{"Path", path})
	}

	if filename != "" {
		info = append(info, []string{"Filename", filename})
	}

	if feature.Description != nil && *feature.Description != "" {
		info = append(info, []string{"Description", *feature.Description})
	}

	// Render info table
	_ = pterm.DefaultTable.WithData(info).Render()
	fmt.Println()

	// Related documents section
	if len(relatedDocs) > 0 {
		pterm.DefaultSection.Println("Related Documents")
		fmt.Println()
		for _, doc := range relatedDocs {
			fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
		}
		fmt.Println()
	}

	// Task status breakdown (workflow-ordered with colored status names)
	if len(statusBreakdown) > 0 {
		pterm.DefaultSection.Println("Task Status Breakdown")
		fmt.Println()
		breakdownData := pterm.TableData{
			{"Status", "Count", "Phase"},
		}
		for _, sc := range statusBreakdown {
			// Format status with color if available
			statusDisplay := sc.Status
			if colorEnabled {
				statusDisplay = workflowService.FormatStatusCount(sc, true)
			}

			breakdownData = append(breakdownData, []string{
				statusDisplay,
				fmt.Sprintf("%d", sc.Count),
				sc.Phase,
			})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithData(breakdownData).Render()
		fmt.Println()
	}

	// Check if all tasks are completed
	allTasksCompleted := len(tasks) > 0 && feature.ProgressPct >= 100.0
	if allTasksCompleted {
		pterm.Success.Println("All tasks completed! Feature is ready for approval.")
		fmt.Println()
	}

	// Tasks section
	if len(tasks) == 0 {
		pterm.Info.Println("No tasks found for this feature")
		return
	}

	pterm.DefaultSection.Printf("Tasks (%d total)", len(tasks))
	fmt.Println()

	// Create tasks table
	tableData := pterm.TableData{
		{"Key", "Title", "Status", "Priority", "Agent"},
	}

	for _, task := range tasks {
		// Truncate long titles
		title := task.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		// Get agent type
		agent := "none"
		if task.AgentType != nil {
			agent = string(*task.AgentType)
		}

		// Format task status with color if available
		taskStatusDisplay := string(task.Status)
		if colorEnabled {
			formatted := workflowService.FormatStatusForDisplay(string(task.Status), true)
			taskStatusDisplay = formatted.Colored
		}

		tableData = append(tableData, []string{
			task.Key,
			title,
			taskStatusDisplay,
			fmt.Sprintf("%d", task.Priority),
			agent,
		})
	}

	// Render tasks table
	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// sortFeatures sorts features by the specified field
func sortFeatures(features []FeatureWithTaskCount, sortBy string) {
	if sortBy == "" || sortBy == "key" {
		// Sort by key (default)
		sortFeaturesByKey(features)
	} else if sortBy == "progress" {
		// Sort by progress
		sortFeaturesByProgress(features)
	} else if sortBy == "status" {
		// Sort by status
		sortFeaturesByStatus(features)
	}
}

// sortFeaturesByKey sorts features by key
func sortFeaturesByKey(features []FeatureWithTaskCount) {
	for i := 0; i < len(features); i++ {
		for j := i + 1; j < len(features); j++ {
			if features[i].Key > features[j].Key {
				features[i], features[j] = features[j], features[i]
			}
		}
	}
}

// sortFeaturesByProgress sorts features by progress (ascending)
func sortFeaturesByProgress(features []FeatureWithTaskCount) {
	for i := 0; i < len(features); i++ {
		for j := i + 1; j < len(features); j++ {
			if features[i].ProgressPct > features[j].ProgressPct {
				features[i], features[j] = features[j], features[i]
			}
		}
	}
}

// sortFeaturesByStatus sorts features by status (draft, active, completed, archived)
func sortFeaturesByStatus(features []FeatureWithTaskCount) {
	statusOrder := map[models.FeatureStatus]int{
		models.FeatureStatusDraft:     1,
		models.FeatureStatusActive:    2,
		models.FeatureStatusCompleted: 3,
		models.FeatureStatusArchived:  4,
	}
	for i := 0; i < len(features); i++ {
		for j := i + 1; j < len(features); j++ {
			if statusOrder[features[i].Status] > statusOrder[features[j].Status] {
				features[i], features[j] = features[j], features[i]
			}
		}
	}
}

// filterFeaturesByCompletedStatus filters out completed features unless showAll is true
// or an explicit status filter is set
func filterFeaturesByCompletedStatus(features []FeatureWithTaskCount, showAll bool, statusFilter string) []FeatureWithTaskCount {
	// If an explicit status filter is set, don't apply default filtering
	if statusFilter != "" {
		return features
	}

	// If showAll is true, return all features
	if showAll {
		return features
	}

	// Default behavior: filter out completed features
	filtered := make([]FeatureWithTaskCount, 0, len(features))
	for _, feature := range features {
		if feature.Status != models.FeatureStatusCompleted {
			filtered = append(filtered, feature)
		}
	}
	return filtered
}

// FeatureTemplateData holds data for feature template rendering
type FeatureTemplateData struct {
	EpicKey     string
	FeatureKey  string
	FeatureSlug string
	Title       string
	Description string
	FilePath    string
	Date        string
}

// runFeatureCreate executes the feature create command
func runFeatureCreate(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse arguments - supports both positional and flag-based syntax
	var featureTitle string
	positionalEpic, positionalTitle, err := ParseFeatureCreateArgs(args)

	if err == nil && positionalEpic != nil && positionalTitle != nil {
		// Positional syntax: shark feature create E07 "Feature Title"
		featureTitle = *positionalTitle
		// Positional epic takes priority over flag (if both provided, use positional)
		if featureCreateEpic != "" && featureCreateEpic != *positionalEpic {
			cli.Warning(fmt.Sprintf("Epic key provided both positionally (%s) and via flag (%s). Using positional value.", *positionalEpic, featureCreateEpic))
		}
		featureCreateEpic = *positionalEpic
	} else if len(args) == 1 && featureCreateEpic != "" {
		// Flag-based syntax: shark feature create --epic=E07 "Feature Title"
		featureTitle = args[0]
	} else {
		// Invalid syntax - show error
		cli.Error(fmt.Sprintf("Error: %v", err))
		fmt.Println("\nValid syntaxes:")
		fmt.Println("  shark feature create E07 \"Feature Title\"           (recommended)")
		fmt.Println("  shark feature create --epic=E07 \"Feature Title\"     (legacy)")
		os.Exit(1)
	}

	// Validate epic key format
	if !isValidEpicKey(featureCreateEpic) {
		cli.Error("Error: Invalid epic key format. Must be E## (e.g., E01, E02)")
		os.Exit(1)
	}

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get repositories
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Verify epic exists in database
	epic, err := epicRepo.GetByKey(ctx, featureCreateEpic)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s not found in database", featureCreateEpic))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	// Get feature key (custom or auto-generated)
	var nextKey string
	if featureCreateKey != "" {
		// Validate custom key using shared validator: no spaces allowed
		if err := ValidateNoSpaces(featureCreateKey, "feature"); err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

		// For custom keys, construct full key as E##-<custom-key>
		// If custom key already has E## prefix, use as-is, otherwise add epic prefix
		if len(featureCreateKey) >= 3 && featureCreateKey[0] == 'F' {
			// Custom key is just F## or similar, prepend epic
			nextKey = fmt.Sprintf("%s-%s", featureCreateEpic, featureCreateKey)
		} else if len(featureCreateKey) >= 7 && featureCreateKey[:3] == featureCreateEpic {
			// Custom key already has epic prefix (e.g., E01-auth)
			nextKey = featureCreateKey
		} else {
			// Custom key is a simple string (e.g., "auth"), construct full key
			nextKey = fmt.Sprintf("%s-%s", featureCreateEpic, featureCreateKey)
		}

		// Check if key already exists
		existing, err := featureRepo.GetByKey(ctx, nextKey)
		if err == nil && existing != nil {
			cli.Error(fmt.Sprintf("Error: Feature with key '%s' already exists", nextKey))
			os.Exit(1)
		}
	} else {
		// Auto-generate next feature key (now includes epic prefix)
		var err error
		nextKey, err = getNextFeatureKey(ctx, featureRepo, epic.ID, epic.Key)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to generate feature key: %v", err))
			os.Exit(1)
		}
	}

	// Generate slug from title
	slug := utils.GenerateSlug(featureTitle)
	featureSlug := fmt.Sprintf("%s-%s", nextKey, slug)

	// Get project root (current working directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get working directory: %v", err))
		os.Exit(1)
	}

	// Use the nextKey which is already in full format (E##-F## or E##-<custom>)
	featureKey := nextKey

	// Set execution_order only if provided (non-zero)
	var executionOrder *int
	if featureCreateExecutionOrder > 0 {
		executionOrder = &featureCreateExecutionOrder
	}

	// Handle custom filename if provided
	var featureFilePath string
	var customFilePath *string

	// Try all three flag aliases: --file, --filename, --path (last one wins)
	file, _ := cmd.Flags().GetString("file")
	filename, _ := cmd.Flags().GetString("filename")
	path, _ := cmd.Flags().GetString("path")

	// Determine which flag was provided (priority: path > filename > file)
	var customFile string
	if path != "" {
		customFile = path
	} else if filename != "" {
		customFile = filename
	} else if file != "" {
		customFile = file
	}

	if customFile != "" {
		// Validate custom filename
		absPath, relPath, err := taskcreation.ValidateCustomFilename(customFile, projectRoot)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Invalid filename: %v", err))
			os.Exit(1)
		}

		// Check for collision with existing features
		existingFeature, err := featureRepo.GetByFilePath(ctx, relPath)
		if err == nil && existingFeature != nil {
			if !featureCreateForce {
				cli.Error(fmt.Sprintf("Error: file '%s' is already claimed by feature %s ('%s'). Use --force to reassign",
					relPath, existingFeature.Key, existingFeature.Title))
				os.Exit(1)
			}
		}

		// Check for collision with existing epics (cross-entity collision)
		existingEpic, err := epicRepo.GetByFilePath(ctx, relPath)
		if err == nil && existingEpic != nil {
			if !featureCreateForce {
				cli.Error(fmt.Sprintf("Error: file '%s' is already claimed by epic %s ('%s'). Use --force to reassign",
					relPath, existingEpic.Key, existingEpic.Title))
				os.Exit(1)
			}
		}

		// Create backup before force reassignment (if any collision exists)
		if (existingFeature != nil || existingEpic != nil) && featureCreateForce {
			dbPath, canBackup, err := cli.GetDatabasePathForBackup()
			if err != nil {
				cli.Error(fmt.Sprintf("Error: failed to get database path for backup: %v", err))
				os.Exit(2)
			}
			if canBackup {
				if _, err := backupDatabaseOnForceFeature(featureCreateForce, dbPath, "force file reassignment"); err != nil {
					cli.Error(fmt.Sprintf("Error: %v", err))
					cli.Info("Aborting operation to prevent data loss")
					os.Exit(2)
				}
			} else {
				// Cloud database - backup is handled by cloud provider
				if cli.GlobalConfig.Verbose {
					cli.Info("Using cloud database - backup handled by provider")
				}
			}
		}

		// Force reassignment: clear the old feature's file path
		if existingFeature != nil && featureCreateForce {
			if err := featureRepo.UpdateFilePath(ctx, existingFeature.Key, nil); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to clear old feature's file path: %v", err))
				os.Exit(1)
			}
		}

		// Force reassignment: clear the old epic's file path
		if existingEpic != nil && featureCreateForce {
			// Force reassignment: clear the old epic's file path
			if err := epicRepo.UpdateFilePath(ctx, existingEpic.Key, nil); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to clear old epic's file path: %v", err))
				os.Exit(1)
			}
		}

		// Create parent directories if they don't exist
		dirPath := filepath.Dir(absPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to create directory structure: %v", err))
			os.Exit(1)
		}

		featureFilePath = absPath
		customFilePath = &relPath
	} else {
		// Default behavior: create feature in standard directory structure
		// Find epic directory
		epicPattern := fmt.Sprintf("docs/plan/%s-*", featureCreateEpic)
		matches, err := filepath.Glob(epicPattern)
		if err != nil || len(matches) == 0 {
			cli.Error(fmt.Sprintf("Error: Epic directory not found for %s", featureCreateEpic))
			cli.Info("The epic exists in the database but the directory structure is missing.")
			os.Exit(1)
		}

		epicDir := matches[0]

		// Create feature directory
		featureDir := fmt.Sprintf("%s/%s", epicDir, featureSlug)

		// Check if feature already exists
		if _, err := os.Stat(featureDir); err == nil {
			cli.Error(fmt.Sprintf("Error: Feature directory already exists: %s", featureDir))
			os.Exit(1)
		}

		// Create feature directory
		if err := os.MkdirAll(featureDir, 0755); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to create feature directory: %v", err))
			os.Exit(1)
		}

		// Set both featureFilePath and customFilePath
		featureFilePath = fmt.Sprintf("%s/feature.md", featureDir)
		relPath := featureFilePath // This is already a relative path from project root
		customFilePath = &relPath
	}

	// Read feature template
	templatePath := "shark-templates/feature.md"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to read feature template: %v", err))
		cli.Info("Make sure you've run 'shark init' to create templates")
		os.Exit(1)
	}

	// Prepare template data
	data := FeatureTemplateData{
		EpicKey:     featureCreateEpic,
		FeatureKey:  nextKey,
		FeatureSlug: featureSlug,
		Title:       featureTitle,
		Description: featureCreateDescription,
		FilePath:    featureFilePath,
		Date:        time.Now().Format("2006-01-02"),
	}

	// Parse and execute template
	tmpl, err := template.New("feature").Parse(string(templateContent))
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to parse feature template: %v", err))
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to render feature template: %v", err))
		os.Exit(1)
	}

	// Write feature file
	if err := os.WriteFile(featureFilePath, buf.Bytes(), 0644); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to write feature file: %v", err))
		os.Exit(1)
	}

	// Parse status flag using shared parsing function (with default "draft")
	statusStr, _ := cmd.Flags().GetString("status")
	if statusStr == "" {
		statusStr = "draft"
	}
	statusStr, err = ParseFeatureStatus(statusStr)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	status := models.FeatureStatus(statusStr)

	// Create feature with custom file path if provided
	feature := &models.Feature{
		EpicID:         epic.ID,
		Key:            featureKey,
		Title:          featureTitle,
		Description:    &featureCreateDescription,
		Status:         status,
		ProgressPct:    0.0,
		ExecutionOrder: executionOrder,
		FilePath:       customFilePath,
	}

	if err := featureRepo.Create(ctx, feature); err != nil {
		// Rollback: delete the created file
		os.Remove(featureFilePath)
		cli.Error(fmt.Sprintf("Error: Failed to create feature in database: %v", err))
		cli.Info("Rolled back file creation")
		os.Exit(1)
	}

	// Success output
	cli.Success("Feature created successfully!")
	fmt.Println()
	fmt.Printf("Feature Key: %s\n", featureSlug)
	fmt.Printf("Epic:        %s\n", featureCreateEpic)
	fmt.Printf("File:        %s\n", featureFilePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Edit the feature file to add details")
	fmt.Printf("2. Create tasks with: shark task create --epic=%s --feature=%s --title=\"Task title\" --agent=backend\n", featureCreateEpic, nextKey)

	return nil
}

// getNextFeatureKey determines the next available feature key (E##-F##) for an epic
// If epicKey is empty, it will attempt to extract from existing features
func getNextFeatureKey(ctx context.Context, featureRepo *repository.FeatureRepository, epicID int64, epicKey ...string) (string, error) {
	// Get all features for this epic
	features, err := featureRepo.ListByEpic(ctx, epicID)
	if err != nil {
		return "", fmt.Errorf("failed to list features: %w", err)
	}

	// Find the maximum feature number and extract epic key from existing features
	maxNum := 0
	extractedEpicKey := ""
	for _, feature := range features {
		// Feature key format in DB is E##-F##, extract both parts
		var epicNum, featureNum int
		if _, err := fmt.Sscanf(feature.Key, "E%d-F%d", &epicNum, &featureNum); err == nil {
			if extractedEpicKey == "" {
				extractedEpicKey = fmt.Sprintf("E%02d", epicNum)
			}
			if featureNum > maxNum {
				maxNum = featureNum
			}
		}
	}

	// Determine which epic key to use: provided parameter or extracted from features
	finalEpicKey := extractedEpicKey
	if len(epicKey) > 0 && epicKey[0] != "" {
		finalEpicKey = epicKey[0]
	}

	// If still no epic key, we can't proceed
	if finalEpicKey == "" {
		return "", fmt.Errorf("unable to determine epic key - no existing features and no epic key provided")
	}

	// Return full feature key with epic prefix
	return fmt.Sprintf("%s-F%02d", finalEpicKey, maxNum+1), nil
}

// runFeatureComplete executes the feature complete command
func runFeatureComplete(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get repositories
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Get feature by key
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'shark feature list' to see available features")
		os.Exit(1)
	}

	// Get all tasks in feature
	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to list tasks: %v", err))
		os.Exit(2)
	}

	// If no tasks, set feature status to completed and inform user
	if len(tasks) == 0 {
		// Set feature status to completed even with no tasks
		feature.Status = models.FeatureStatusCompleted
		if err := featureRepo.Update(ctx, feature); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update feature status: %v", err))
			os.Exit(2)
		}

		if cli.GlobalConfig.JSON {
			result := map[string]interface{}{
				"feature_key":      featureKey,
				"completed_count":  0,
				"total_count":      0,
				"status_breakdown": map[string]int{},
				"affected_tasks":   []string{},
			}
			return cli.OutputJSON(result)
		}
		cli.Success(fmt.Sprintf("Feature %s completed (no tasks)", featureKey))
		return nil
	}

	// Get status breakdown using new workflow-aware method
	statusBreakdownSlice, err := taskRepo.GetStatusBreakdown(ctx, feature.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get task status: %v", err))
		os.Exit(2)
	}

	// Convert to map for efficient lookup
	statusBreakdown := make(map[models.TaskStatus]int)
	for _, sc := range statusBreakdownSlice {
		statusBreakdown[models.TaskStatus(sc.Status)] = sc.Count
	}

	// Count completed and reviewed tasks (tasks that don't need completion)
	completedCount := statusBreakdown[models.TaskStatusCompleted]
	reviewedCount := statusBreakdown[models.TaskStatusReadyForReview]
	allDoneCount := completedCount + reviewedCount

	// Identify incomplete tasks (any status NOT in {completed, ready_for_review})
	var incompleteTasks []*models.Task
	for _, task := range tasks {
		if task.Status != models.TaskStatusCompleted && task.Status != models.TaskStatusReadyForReview {
			incompleteTasks = append(incompleteTasks, task)
		}
	}

	hasIncomplete := len(incompleteTasks) > 0

	// Show warning if incomplete tasks exist and no --force
	if hasIncomplete && !force {
		// Build status breakdown summary
		statusSummary := ""
		todoCount := statusBreakdown[models.TaskStatusTodo]
		inProgressCount := statusBreakdown[models.TaskStatusInProgress]
		blockedCount := statusBreakdown[models.TaskStatusBlocked]

		statusSummary = fmt.Sprintf("%d todo, %d in_progress, %d blocked, %d ready_for_review",
			todoCount, inProgressCount, blockedCount, reviewedCount)

		cli.Warning("Cannot complete feature with incomplete tasks")
		fmt.Printf("  Status breakdown: %s\n", statusSummary)

		// Show affected tasks (up to 10)
		fmt.Println("\nAffected tasks:")
		maxTasks := 10
		if len(incompleteTasks) < maxTasks {
			maxTasks = len(incompleteTasks)
		}
		for i := 0; i < maxTasks; i++ {
			task := incompleteTasks[i]
			fmt.Printf("  - %s (%s)\n", task.Key, task.Status)
		}

		if len(incompleteTasks) > 10 {
			fmt.Printf("  ... and %d more\n", len(incompleteTasks)-10)
		}

		cli.Info("Use --force to complete all tasks regardless of status")

		// If JSON output requested, return error with details
		if cli.GlobalConfig.JSON {
			// Convert status breakdown to map with string keys
			breakdown := make(map[string]int)
			breakdown["todo"] = todoCount
			breakdown["in_progress"] = inProgressCount
			breakdown["blocked"] = blockedCount
			breakdown["ready_for_review"] = reviewedCount
			breakdown["completed"] = completedCount

			affectedKeys := make([]string, len(incompleteTasks))
			for i, task := range incompleteTasks {
				affectedKeys[i] = task.Key
			}

			result := map[string]interface{}{
				"feature_key":      featureKey,
				"completed_count":  allDoneCount,
				"total_count":      len(tasks),
				"status_breakdown": breakdown,
				"affected_tasks":   affectedKeys,
				"requires_force":   true,
			}
			return cli.OutputJSON(result)
		}

		os.Exit(3)
	}

	// Create backup before force completing tasks
	if force && hasIncomplete {
		dbPath, canBackup, err := cli.GetDatabasePathForBackup()
		if err != nil {
			cli.Error(fmt.Sprintf("Error: failed to get database path for backup: %v", err))
			os.Exit(2)
		}
		if canBackup {
			if _, err := backupDatabaseOnForceFeature(force, dbPath, "force complete feature"); err != nil {
				cli.Error(fmt.Sprintf("Error: %v", err))
				cli.Info("Aborting operation to prevent data loss")
				os.Exit(2)
			}
		} else {
			// Cloud database - backup is handled by cloud provider
			if cli.GlobalConfig.Verbose {
				cli.Info("Using cloud database - backup handled by provider")
			}
		}
	}

	// Complete all tasks in a transaction
	agent := getAgentIdentifier("")
	numCompleted := 0
	affectedTaskKeys := make([]string, 0)

	for _, task := range tasks {
		// Skip already completed tasks
		if task.Status == models.TaskStatusCompleted {
			continue
		}

		// Mark as completed
		if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to complete task %s: %v", task.Key, err))
			os.Exit(2)
		}
		numCompleted++
		affectedTaskKeys = append(affectedTaskKeys, task.Key)
	}

	// Update feature progress (which now auto-completes at 100%)
	if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to update feature progress: %v", err))
		os.Exit(2)
	}

	// Fetch updated feature to get the new status
	feature, err = featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to fetch updated feature: %v", err))
		os.Exit(2)
	}

	// Output results
	if cli.GlobalConfig.JSON {
		// Convert status breakdown to map with string keys
		breakdown := make(map[string]int)
		breakdown["todo"] = statusBreakdown[models.TaskStatusTodo]
		breakdown["in_progress"] = statusBreakdown[models.TaskStatusInProgress]
		breakdown["blocked"] = statusBreakdown[models.TaskStatusBlocked]
		breakdown["ready_for_review"] = statusBreakdown[models.TaskStatusReadyForReview]
		breakdown["completed"] = statusBreakdown[models.TaskStatusCompleted]

		result := map[string]interface{}{
			"feature_key":      featureKey,
			"completed_count":  len(tasks), // All tasks are now completed
			"total_count":      len(tasks),
			"status_breakdown": breakdown,
			"affected_tasks":   affectedTaskKeys,
		}
		return cli.OutputJSON(result)
	}

	// Human-readable output
	statusMsg := ""
	if feature.Status == models.FeatureStatusCompleted {
		statusMsg = " (feature marked as completed)"
	}

	if hasIncomplete && force {
		// Show what was force-completed
		todoCount := statusBreakdown[models.TaskStatusTodo]
		inProgressCount := statusBreakdown[models.TaskStatusInProgress]
		blockedCount := statusBreakdown[models.TaskStatusBlocked]
		reviewedCount := statusBreakdown[models.TaskStatusReadyForReview]

		statusCounts := fmt.Sprintf("%d todo, %d in_progress, %d blocked, %d ready_for_review",
			todoCount, inProgressCount, blockedCount, reviewedCount)

		cli.Success(fmt.Sprintf("Feature %s completed: Force-completed %d tasks (%s)%s",
			featureKey, numCompleted, statusCounts, statusMsg))
	} else {
		// All tasks were already completed or in review
		cli.Success(fmt.Sprintf("Feature %s completed: %d/%d tasks completed%s", featureKey, len(tasks), len(tasks), statusMsg))
	}

	return nil
}

// runFeatureDelete executes the feature delete command
func runFeatureDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get repositories
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Get feature by key to verify it exists
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'shark feature list' to see available features")
		os.Exit(1)
	}

	// Check for child tasks
	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to check for tasks: %v", err))
		os.Exit(1)
	}

	// If there are tasks, require --force flag
	if len(tasks) > 0 && !force {
		cli.Error(fmt.Sprintf("Error: Feature %s has %d task(s)", featureKey, len(tasks)))
		cli.Warning("This will CASCADE DELETE all tasks and their history")
		cli.Info(fmt.Sprintf("Use --force to confirm deletion: shark feature delete %s --force", featureKey))
		os.Exit(1)
	}

	// Create backup before cascade delete (when feature has tasks)
	if len(tasks) > 0 {
		dbPath, canBackup, err := cli.GetDatabasePathForBackup()
		if err != nil {
			cli.Error(fmt.Sprintf("Error: failed to get database path for backup: %v", err))
			os.Exit(2)
		}
		if canBackup {
			backupPath, err := db.BackupDatabase(dbPath)
			if err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to create backup before deletion: %v", err))
				cli.Info("Aborting deletion to prevent data loss")
				os.Exit(2)
			}
			if !cli.GlobalConfig.JSON {
				cli.Info(fmt.Sprintf("Database backup created: %s", backupPath))
			}
		} else {
			// Cloud database - backup is handled by cloud provider
			if cli.GlobalConfig.Verbose {
				cli.Info("Using cloud database - backup handled by provider")
			}
		}
	}

	// Delete feature from database (CASCADE will handle tasks)
	if err := featureRepo.Delete(ctx, feature.ID); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to delete feature: %v", err))
		os.Exit(1)
	}

	cli.Success(fmt.Sprintf("Feature %s deleted successfully", featureKey))
	if len(tasks) > 0 {
		cli.Warning(fmt.Sprintf("Cascade deleted %d task(s) and their history", len(tasks)))
	}
	return nil
}

// runFeatureUpdate executes the feature update command
func runFeatureUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := args[0]

	// Get database connection (cloud-aware)
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	// Get repositories
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Get feature by key to verify it exists
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'shark feature list' to see available features")
		os.Exit(1)
	}

	// Track if any changes were made
	changed := false

	// Update title if provided
	title, _ := cmd.Flags().GetString("title")
	if title != "" {
		feature.Title = title
		changed = true
	}

	// Update description if provided
	description, _ := cmd.Flags().GetString("description")
	if description != "" {
		feature.Description = &description
		changed = true
	}

	// Update status if provided (using shared validation)
	// Special handling for "auto" to enable calculated status
	statusFlag, _ := cmd.Flags().GetString("status")
	if statusFlag != "" {
		if strings.ToLower(statusFlag) == "auto" {
			// Clear status override and recalculate status
			if err := featureRepo.SetStatusOverride(ctx, feature.ID, false); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to clear status override: %v", err))
				os.Exit(1)
			}

			// Recalculate status from tasks
			calcService := status.NewCalculationService(repoDb)
			result, err := calcService.RecalculateFeatureStatus(ctx, feature.ID)
			if err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to recalculate status: %v", err))
				os.Exit(1)
			}

			cli.Success(fmt.Sprintf("Feature %s status recalculated: %s (calculated from tasks)", feature.Key, result.NewStatus))
			return nil
		}

		// Regular status update - set override and apply status
		validatedStatus, err := ParseFeatureStatus(statusFlag)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

		// Set override to true for manual status
		if err := featureRepo.SetStatusOverride(ctx, feature.ID, true); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to set status override: %v", err))
			os.Exit(1)
		}

		feature.Status = models.FeatureStatus(validatedStatus)
		changed = true
	}

	// Update execution order if provided
	execOrder, _ := cmd.Flags().GetInt("execution-order")
	if execOrder != -1 {
		feature.ExecutionOrder = &execOrder
		changed = true
	}

	// Apply core field updates if any changed
	if changed {
		if err := featureRepo.Update(ctx, feature); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update feature: %v", err))
			os.Exit(1)
		}
	}

	// Handle key update separately (requires unique validation)
	newKey, _ := cmd.Flags().GetString("key")
	if newKey != "" {
		// Validate new key using shared validator: no spaces allowed
		if err := ValidateNoSpaces(newKey, "feature"); err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

		// Check if new key already exists (and is different from current key)
		if newKey != featureKey {
			existing, err := featureRepo.GetByKey(ctx, newKey)
			if err == nil && existing != nil {
				cli.Error(fmt.Sprintf("Error: Feature with key '%s' already exists", newKey))
				os.Exit(1)
			}

			// Update the key
			if err := featureRepo.UpdateKey(ctx, featureKey, newKey); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to update feature key: %v", err))
				os.Exit(1)
			}
			changed = true
		}
	}

	// Handle filename update separately
	// Try all three flag aliases: --file, --filename, --path (last one wins)
	file, _ := cmd.Flags().GetString("file")
	filename, _ := cmd.Flags().GetString("filename")
	path, _ := cmd.Flags().GetString("path")

	// Determine which flag was provided (priority: path > filename > file)
	var customFile string
	if path != "" {
		customFile = path
	} else if filename != "" {
		customFile = filename
	} else if file != "" {
		customFile = file
	}

	if customFile != "" {
		if err := featureRepo.UpdateFilePath(ctx, featureKey, &customFile); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update feature file path: %v", err))
			os.Exit(1)
		}
		changed = true
	}

	if !changed {
		cli.Warning("No changes specified. Use --help to see available flags.")
		return nil
	}

	cli.Success(fmt.Sprintf("Feature %s updated successfully", featureKey))
	return nil
}
