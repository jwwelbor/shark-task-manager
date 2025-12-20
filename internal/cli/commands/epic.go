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
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// getRelativePath converts an absolute path to relative path from project root
func getRelativePath(absPath string, projectRoot string) string {
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return absPath // Fall back to absolute path if conversion fails
	}
	return relPath
}

// EpicWithProgress wraps an Epic with its calculated progress
type EpicWithProgress struct {
	*models.Epic
	ProgressPct float64 `json:"progress_pct"`
}

// FeatureWithDetails wraps a Feature with task count
type FeatureWithDetails struct {
	*models.Feature
	TaskCount int `json:"task_count"`
}

// epicCmd represents the epic command group
var epicCmd = &cobra.Command{
	Use:   "epic",
	Short: "Manage epics",
	Long: `Query and manage epics with automatic progress calculation.

Examples:
  shark epic list                 List all epics
  shark epic get E04             Get epic details with progress
  shark epic status              Show status of all epics`,
}

// epicListCmd lists epics
var epicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all epics",
	Long: `List all epics with progress information.

Examples:
  shark epic list                 List all epics
  shark epic list --json          Output as JSON`,
	RunE: runEpicList,
}

// epicGetCmd gets a specific epic
var epicGetCmd = &cobra.Command{
	Use:   "get <epic-key>",
	Short: "Get epic details",
	Long: `Display detailed information about a specific epic including all features and progress.

Examples:
  shark epic get E04              Get epic details
  shark epic get E04 --json       Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicGet,
}

// epicStatusCmd shows status of all epics
var epicStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show epic status summary",
	Long:  `Display a summary of all epics with completion percentages and task counts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement in E05-F01 (Status Dashboard)
		cli.Warning("Not yet implemented - coming in E05-F01")
		return nil
	},
}

// epicCompleteCmd completes all tasks in an epic
var epicCompleteCmd = &cobra.Command{
	Use:   "complete <epic-key>",
	Short: "Complete all tasks in an epic",
	Long: `Mark all tasks across all features in an epic as completed, with safeguards against accidental completion.

Without --force, shows a warning summary if any tasks are incomplete and fails.
With --force, completes all tasks regardless of status.

Examples:
  shark epic complete E07       Complete epic (fails if tasks incomplete)
  shark epic complete E07 --force  Force complete all tasks`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicComplete,
}

// epicCreateCmd creates a new epic
var epicCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new epic",
	Long: `Create a new epic with auto-assigned key, folder structure, and database entry.

The epic key is automatically assigned as the next available E## number.

Flags:
  --filename string    Custom file path relative to project root (must end in .md)
  --path string        Custom base folder path for this epic and children (relative to project root)
  --force              Force reassignment if file already claimed by another entity
  --description string Epic description
  --priority string    Priority: high, medium, low (default: medium)
  --business-value string Business value: high, medium, low

Examples:
  shark epic create "User Authentication System"
  shark epic create "User Auth" --description="Add OAuth and MFA"
  shark epic create "Platform Roadmap" --path="docs/specs"
  shark epic create "Platform Roadmap" --filename="docs/roadmap/2025.md"
  shark epic create "Q1 Goals" --filename="docs/roadmap/q1.md" --force`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicCreate,
}

// epicDeleteCmd deletes an epic
var epicDeleteCmd = &cobra.Command{
	Use:   "delete <epic-key>",
	Short: "Delete an epic",
	Long: `Delete an epic from the database (and all its features/tasks via CASCADE).

WARNING: This action cannot be undone. All features and tasks under this epic will also be deleted.
If the epic has features, you must use --force to confirm the cascade deletion.

Examples:
  shark epic delete E05               Delete epic with no features
  shark epic delete E05 --force       Force delete epic with features`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicDelete,
}

var (
	epicCreateDescription string
	epicCreatePath        string
)

func init() {
	// Register epic command with root
	cli.RootCmd.AddCommand(epicCmd)

	// Add subcommands
	epicCmd.AddCommand(epicListCmd)
	epicCmd.AddCommand(epicGetCmd)
	epicCmd.AddCommand(epicStatusCmd)
	epicCmd.AddCommand(epicCompleteCmd)
	epicCmd.AddCommand(epicCreateCmd)
	epicCmd.AddCommand(epicDeleteCmd)

	// Add flags for list command
	epicListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	epicListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")

	// Add flags for complete command
	epicCompleteCmd.Flags().Bool("force", false, "Force completion of all tasks regardless of status")

	// Add flags for create command
	epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
	epicCreateCmd.Flags().StringVar(&epicCreatePath, "path", "", "Custom base folder path for this epic and children (relative to project root)")
	epicCreateCmd.Flags().String("filename", "", "Custom filename path (relative to project root, must end in .md)")
	epicCreateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another epic or feature")

	// Add flags for delete command
	epicDeleteCmd.Flags().Bool("force", false, "Force deletion even if epic has features")
}

// runEpicList executes the epic list command
func runEpicList(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get flags
	sortBy, _ := cmd.Flags().GetString("sort-by")
	statusFilter, _ := cmd.Flags().GetString("status")

	// Validate status filter
	if statusFilter != "" {
		validStatuses := []string{"draft", "active", "completed", "archived"}
		valid := false
		for _, s := range validStatuses {
			if statusFilter == s {
				valid = true
				break
			}
		}
		if !valid {
			cli.Error(fmt.Sprintf("Error: Invalid status '%s'. Must be one of: draft, active, completed, archived", statusFilter))
			os.Exit(1)
		}
	}

	// Validate sort-by option
	if sortBy != "" && sortBy != "key" && sortBy != "progress" && sortBy != "status" {
		cli.Error(fmt.Sprintf("Error: Invalid sort-by '%s'. Must be one of: key, progress, status", sortBy))
		os.Exit(1)
	}

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
		return fmt.Errorf("database path error")
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}

	// Get repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)

	// Apply status filter
	var statusPtr *models.EpicStatus
	if statusFilter != "" {
		status := models.EpicStatus(statusFilter)
		statusPtr = &status
	}

	// Get all epics
	epics, err := epicRepo.List(ctx, statusPtr)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to list epics: %v\n", err)
		}
		os.Exit(2)
	}

	// Handle empty results
	if len(epics) == 0 {
		if cli.GlobalConfig.JSON {
			result := map[string]interface{}{
				"results": []interface{}{},
				"count":   0,
			}
			return cli.OutputJSON(result)
		}
		cli.Info("No epics found")
		return nil
	}

	// Calculate progress for each epic
	epicsWithProgress := make([]EpicWithProgress, 0, len(epics))
	for _, epic := range epics {
		progress, err := epicRepo.CalculateProgress(ctx, epic.ID)
		if err != nil {
			if cli.GlobalConfig.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to calculate progress for epic %s: %v\n", epic.Key, err)
			}
			progress = 0.0
		}
		epicsWithProgress = append(epicsWithProgress, EpicWithProgress{
			Epic:        epic,
			ProgressPct: progress,
		})
	}

	// Apply sorting
	sortEpics(epicsWithProgress, sortBy)

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		result := map[string]interface{}{
			"results": epicsWithProgress,
			"count":   len(epicsWithProgress),
		}
		return cli.OutputJSON(result)
	}

	// Output as table
	renderEpicListTable(epicsWithProgress)
	return nil
}

// runEpicGet executes the epic get command
func runEpicGet(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := args[0]

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
		return fmt.Errorf("database path error")
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}

	// Get repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Get epic by key
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	// Get project root for path resolution
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}

	// Resolve epic path
	var resolvedPath string
	if projectRoot != "" {
		pathBuilder := utils.NewPathBuilder(projectRoot)
		absPath, err := pathBuilder.ResolveEpicPath(epic.Key, epic.FilePath, epic.CustomFolderPath)
		if err == nil {
			resolvedPath = getRelativePath(absPath, projectRoot)
		}
	}

	// Calculate epic progress
	epicProgress, err := epicRepo.CalculateProgress(ctx, epic.ID)
	if err != nil {
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to calculate epic progress: %v\n", err)
		}
		epicProgress = 0.0
	}

	// Get features for this epic
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Failed to list features: %v\n", err)
		}
		os.Exit(2)
	}

	// Calculate progress and task count for each feature
	featuresWithDetails := make([]FeatureWithDetails, 0, len(features))
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

		// Get task count
		var taskCount int
		err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE feature_id = ?", feature.ID).Scan(&taskCount)
		if err != nil {
			taskCount = 0
		}

		featuresWithDetails = append(featuresWithDetails, FeatureWithDetails{
			Feature:   feature,
			TaskCount: taskCount,
		})
	}

	// Get filename from resolved path
	var filename string
	if resolvedPath != "" {
		filename = filepath.Base(resolvedPath)
	}

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		result := map[string]interface{}{
			"id":             epic.ID,
			"key":            epic.Key,
			"title":          epic.Title,
			"description":    epic.Description,
			"status":         epic.Status,
			"priority":       epic.Priority,
			"business_value": epic.BusinessValue,
			"progress_pct":   epicProgress,
			"path":           resolvedPath,
			"filename":       filename,
			"created_at":     epic.CreatedAt,
			"updated_at":     epic.UpdatedAt,
			"features":       featuresWithDetails,
		}
		return cli.OutputJSON(result)
	}

	// Output as formatted text
	renderEpicDetails(epic, epicProgress, featuresWithDetails, resolvedPath, filename)
	return nil
}

// renderEpicListTable renders epics as a table
func renderEpicListTable(epics []EpicWithProgress) {
	// Create table data
	tableData := pterm.TableData{
		{"Key", "Title", "Status", "Progress", "Priority"},
	}

	for _, epic := range epics {
		// Truncate long titles to fit in 80 columns
		title := epic.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		// Format progress with 1 decimal place
		progress := fmt.Sprintf("%.1f%%", epic.ProgressPct)

		tableData = append(tableData, []string{
			epic.Key,
			title,
			string(epic.Status),
			progress,
			string(epic.Priority),
		})
	}

	// Render table
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// renderEpicDetails renders epic details with features table
func renderEpicDetails(epic *models.Epic, progress float64, features []FeatureWithDetails, path, filename string) {
	// Print epic metadata
	pterm.DefaultSection.Printf("Epic: %s", epic.Key)
	fmt.Println()

	// Epic info
	info := [][]string{
		{"Title", epic.Title},
		{"Status", string(epic.Status)},
		{"Priority", string(epic.Priority)},
		{"Progress", fmt.Sprintf("%.1f%%", progress)},
	}

	if path != "" {
		info = append(info, []string{"Path", path})
	}

	if filename != "" {
		info = append(info, []string{"Filename", filename})
	}

	if epic.Description != nil && *epic.Description != "" {
		info = append(info, []string{"Description", *epic.Description})
	}

	if epic.BusinessValue != nil {
		info = append(info, []string{"Business Value", string(*epic.BusinessValue)})
	}

	// Render info table
	pterm.DefaultTable.WithData(info).Render()
	fmt.Println()

	// Features section
	if len(features) == 0 {
		pterm.Info.Println("No features found for this epic")
		return
	}

	pterm.DefaultSection.Println("Features")
	fmt.Println()

	// Create features table
	tableData := pterm.TableData{
		{"Key", "Title", "Status", "Progress", "Tasks"},
	}

	for _, feature := range features {
		// Truncate long titles
		title := feature.Title
		if len(title) > 35 {
			title = title[:32] + "..."
		}

		// Format progress with 1 decimal place
		progress := fmt.Sprintf("%.1f%%", feature.ProgressPct)

		tableData = append(tableData, []string{
			feature.Key,
			title,
			string(feature.Status),
			progress,
			fmt.Sprintf("%d", feature.TaskCount),
		})
	}

	// Render features table
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// truncateString truncates a string to maxLen and adds ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}
	return s[:maxLen-3] + "..."
}

// sortEpics sorts epics by the specified field
func sortEpics(epics []EpicWithProgress, sortBy string) {
	if sortBy == "" || sortBy == "key" {
		// Sort by key (default)
		sortEpicsByKey(epics)
	} else if sortBy == "progress" {
		// Sort by progress
		sortEpicsByProgress(epics)
	} else if sortBy == "status" {
		// Sort by status
		sortEpicsByStatus(epics)
	}
}

// sortEpicsByKey sorts epics by key
func sortEpicsByKey(epics []EpicWithProgress) {
	for i := 0; i < len(epics); i++ {
		for j := i + 1; j < len(epics); j++ {
			if epics[i].Key > epics[j].Key {
				epics[i], epics[j] = epics[j], epics[i]
			}
		}
	}
}

// sortEpicsByProgress sorts epics by progress (ascending)
func sortEpicsByProgress(epics []EpicWithProgress) {
	for i := 0; i < len(epics); i++ {
		for j := i + 1; j < len(epics); j++ {
			if epics[i].ProgressPct > epics[j].ProgressPct {
				epics[i], epics[j] = epics[j], epics[i]
			}
		}
	}
}

// sortEpicsByStatus sorts epics by status (draft, active, completed, archived)
func sortEpicsByStatus(epics []EpicWithProgress) {
	statusOrder := map[models.EpicStatus]int{
		models.EpicStatusDraft:     1,
		models.EpicStatusActive:    2,
		models.EpicStatusCompleted: 3,
		models.EpicStatusArchived:  4,
	}
	for i := 0; i < len(epics); i++ {
		for j := i + 1; j < len(epics); j++ {
			if statusOrder[epics[i].Status] > statusOrder[epics[j].Status] {
				epics[i], epics[j] = epics[j], epics[i]
			}
		}
	}
}

// EpicTemplateData holds data for epic template rendering
type EpicTemplateData struct {
	EpicKey     string
	EpicSlug    string
	Title       string
	Description string
	FilePath    string
	Date        string
}

// runEpicCreate executes the epic create command
func runEpicCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get title from args
	epicTitle := args[0]

	// Get optional flags
	filename, _ := cmd.Flags().GetString("filename")
	force, _ := cmd.Flags().GetBool("force")

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
		return fmt.Errorf("database path error")
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}

	// Get project root (current working directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
		os.Exit(1)
	}

	// Validate and process custom path if provided
	var customFolderPath *string
	if epicCreatePath != "" {
		_, relPath, err := utils.ValidateFolderPath(epicCreatePath, projectRoot)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		customFolderPath = &relPath
	}

	// Get repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Get next epic key
	nextKey, err := getNextEpicKey(ctx, epicRepo)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get next epic key: %v", err))
		os.Exit(1)
	}

	// Validate and process custom filename if provided
	var customFilePath *string
	var actualFilePath string // The path where the file will be created

	if filename != "" {
		// Validate custom filename
		absPath, relPath, err := taskcreation.ValidateCustomFilename(filename, projectRoot)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Invalid filename: %v", err))
			os.Exit(1)
		}

		// Collision detection
		existingEpic, err := epicRepo.GetByFilePath(ctx, relPath)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to check for file collision: %v", err))
			os.Exit(2)
		}

		// Check if feature owns the file
		existingFeature, _ := featureRepo.GetByFilePath(ctx, relPath)

		// Handle collision
		if existingEpic != nil && !force {
			cli.Error(fmt.Sprintf("Error: file '%s' is already claimed by epic %s ('%s'). Use --force to reassign",
				relPath, existingEpic.Key, existingEpic.Title))
			os.Exit(1)
		}

		if existingFeature != nil && !force {
			cli.Error(fmt.Sprintf("Error: file '%s' is already claimed by feature %s ('%s'). Use --force to reassign",
				relPath, existingFeature.Key, existingFeature.Title))
			os.Exit(1)
		}

		// Force reassignment if collision exists and --force is set
		if existingEpic != nil && force {
			if err := epicRepo.UpdateFilePath(ctx, existingEpic.Key, nil); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to reassign file from epic %s: %v", existingEpic.Key, err))
				os.Exit(2)
			}
			cli.Warning(fmt.Sprintf("Reassigned file from epic %s ('%s')", existingEpic.Key, existingEpic.Title))
		}

		if existingFeature != nil && force {
			if err := featureRepo.UpdateFilePath(ctx, existingFeature.Key, nil); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to reassign file from feature %s: %v", existingFeature.Key, err))
				os.Exit(2)
			}
			cli.Warning(fmt.Sprintf("Reassigned file from feature %s ('%s')", existingFeature.Key, existingFeature.Title))
		}

		customFilePath = &relPath
		actualFilePath = absPath
	} else {
		// Default behavior: use docs/plan/{epic-key}-{slug}/epic.md
		// Generate slug from title
		slug := utils.GenerateSlug(epicTitle)
		epicSlug := fmt.Sprintf("%s-%s", nextKey, slug)

		// Create folder path
		epicDir := fmt.Sprintf("docs/plan/%s", epicSlug)

		// Check if epic already exists (shouldn't happen with auto-increment)
		if _, err := os.Stat(epicDir); err == nil {
			cli.Error(fmt.Sprintf("Error: Epic directory already exists: %s", epicDir))
			os.Exit(1)
		}

		// Create epic directory
		if err := os.MkdirAll(epicDir, 0755); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to create epic directory: %v", err))
			os.Exit(1)
		}

		actualFilePath = fmt.Sprintf("%s/epic.md", epicDir)
	}

	// Read epic template
	templatePath := "shark-templates/epic.md"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to read epic template: %v", err))
		cli.Info("Make sure you've run 'shark init' to create templates")
		os.Exit(1)
	}

	// Prepare template data
	data := EpicTemplateData{
		EpicKey:     nextKey,
		EpicSlug:    nextKey,
		Title:       epicTitle,
		Description: epicCreateDescription,
		FilePath:    actualFilePath,
		Date:        time.Now().Format("2006-01-02"),
	}

	// Parse and execute template
	tmpl, err := template.New("epic").Parse(string(templateContent))
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to parse epic template: %v", err))
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to render epic template: %v", err))
		os.Exit(1)
	}

	// Create parent directories if needed (for custom paths)
	if filename != "" {
		parentDir := filepath.Dir(actualFilePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to create parent directories: %v", err))
			os.Exit(1)
		}
	}

	// Write epic file
	if err := os.WriteFile(actualFilePath, buf.Bytes(), 0644); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to write epic file: %v", err))
		os.Exit(1)
	}

	// Create database entry with key (E##) not full slug
	epic := &models.Epic{
		Key:               nextKey,
		Title:             epicTitle,
		Description:       &epicCreateDescription,
		Status:            models.EpicStatusDraft,
		Priority:          models.PriorityMedium,
		BusinessValue:     nil,
		FilePath:          customFilePath,
		CustomFolderPath:  customFolderPath,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to create epic in database: %v", err))
		// Clean up file on DB error
		os.Remove(actualFilePath)
		os.Exit(1)
	}

	// Success output
	cli.Success(fmt.Sprintf("Created epic %s '%s' at %s", nextKey, epicTitle, actualFilePath))
	if customFilePath != nil {
		fmt.Printf("Start work with: shark epic get %s\n", nextKey)
	} else {
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Edit the epic.md file to add details")
		fmt.Printf("2. Create features with: shark feature create --epic=%s \"Feature title\"\n", nextKey)
	}

	return nil
}

// getNextEpicKey finds the next available epic key
func getNextEpicKey(ctx context.Context, epicRepo *repository.EpicRepository) (string, error) {
	epics, err := epicRepo.List(ctx, nil)
	if err != nil {
		return "", err
	}

	maxNum := 0
	for _, epic := range epics {
		// Extract number from key in DB (E01 -> 1, E02 -> 2, etc.)
		var num int
		if _, err := fmt.Sscanf(epic.Key, "E%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}

	return fmt.Sprintf("E%02d", maxNum+1), nil
}

// runEpicComplete executes the epic complete command
func runEpicComplete(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
		return fmt.Errorf("database path error")
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}
	defer database.Close()

	// Get repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Get epic by key
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	// Get all features in epic
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to list features: %v", err))
		os.Exit(2)
	}

	// If no features, inform user
	if len(features) == 0 {
		cli.Info(fmt.Sprintf("Epic %s has no features to complete", epicKey))
		return nil
	}

	// Collect all tasks from all features with per-feature tracking
	var allTasks []*models.Task
	totalStatusBreakdown := make(map[models.TaskStatus]int)
	featureTaskBreakdown := make(map[string]map[models.TaskStatus]int) // feature.Key -> status breakdown
	featureTaskCounts := make(map[string]int)                            // feature.Key -> total task count

	for _, feature := range features {
		tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to list tasks in feature %s: %v", feature.Key, err))
			os.Exit(2)
		}

		allTasks = append(allTasks, tasks...)
		featureTaskCounts[feature.Key] = len(tasks)

		// Aggregate status breakdown
		statusBreakdown, err := taskRepo.GetStatusBreakdown(ctx, feature.ID)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to get status breakdown for feature %s: %v", feature.Key, err))
			os.Exit(2)
		}

		featureTaskBreakdown[feature.Key] = statusBreakdown
		for status, count := range statusBreakdown {
			totalStatusBreakdown[status] += count
		}
	}

	// If no tasks, inform user
	if len(allTasks) == 0 {
		cli.Info(fmt.Sprintf("Epic %s has no tasks to complete", epicKey))
		return nil
	}

	// Count completed and reviewed tasks
	completedCount := totalStatusBreakdown[models.TaskStatusCompleted]
	reviewedCount := totalStatusBreakdown[models.TaskStatusReadyForReview]
	allDoneCount := completedCount + reviewedCount

	// Check if all tasks are already completed/reviewed
	hasIncomplete := allDoneCount < len(allTasks)

	// Show warning if incomplete tasks exist
	if hasIncomplete && !force {
		cli.Warning("Cannot complete epic with incomplete tasks")
		fmt.Println()

		// Overall status breakdown
		fmt.Printf("Total tasks: %d\n", len(allTasks))
		fmt.Print("Status breakdown: ")
		breakdownParts := []string{}
		for _, status := range []models.TaskStatus{
			models.TaskStatusTodo,
			models.TaskStatusInProgress,
			models.TaskStatusBlocked,
			models.TaskStatusReadyForReview,
		} {
			if count, ok := totalStatusBreakdown[status]; ok && count > 0 {
				breakdownParts = append(breakdownParts, fmt.Sprintf("%d %s", count, status))
			}
		}
		fmt.Println(strings.Join(breakdownParts, ", "))
		fmt.Println()

		// Per-feature breakdown
		fmt.Println("Feature breakdown:")
		for _, feature := range features {
			breakdown := featureTaskBreakdown[feature.Key]
			totalInFeature := featureTaskCounts[feature.Key]
			completedInFeature := breakdown[models.TaskStatusCompleted] + breakdown[models.TaskStatusReadyForReview]

			if completedInFeature == totalInFeature {
				fmt.Printf("  %s: %d tasks (all ready_for_review or completed)\n", feature.Key, totalInFeature)
			} else {
				incompleteInFeature := totalInFeature - completedInFeature
				fmt.Printf("  %s: %d tasks (%d incomplete) ", feature.Key, totalInFeature, incompleteInFeature)

				// Show breakdown for this feature
				parts := []string{}
				for _, status := range []models.TaskStatus{
					models.TaskStatusTodo,
					models.TaskStatusInProgress,
					models.TaskStatusBlocked,
				} {
					if count, ok := breakdown[status]; ok && count > 0 {
						parts = append(parts, fmt.Sprintf("%d %s", count, status))
					}
				}
				if len(parts) > 0 {
					fmt.Printf("(%s)", strings.Join(parts, ", "))
				}
				fmt.Println()
			}
		}
		fmt.Println()

		// List most problematic tasks (blocked first, then other incomplete)
		fmt.Println("Most problematic tasks:")
		problematicTasks := make([]*models.Task, 0)

		// First, collect all blocked tasks
		for _, task := range allTasks {
			if task.Status == models.TaskStatusBlocked {
				problematicTasks = append(problematicTasks, task)
			}
		}

		// Then, collect other incomplete tasks
		for _, task := range allTasks {
			if task.Status != models.TaskStatusBlocked && task.Status != models.TaskStatusCompleted && task.Status != models.TaskStatusReadyForReview {
				problematicTasks = append(problematicTasks, task)
			}
		}

		// Show up to 15 most problematic tasks
		maxTasks := 15
		if len(problematicTasks) > maxTasks {
			problematicTasks = problematicTasks[:maxTasks]
		}

		for _, task := range problematicTasks {
			if task.Status == models.TaskStatusBlocked {
				reason := ""
				if task.BlockedReason != nil && *task.BlockedReason != "" {
					reason = fmt.Sprintf(" - %s", *task.BlockedReason)
				}
				fmt.Printf("  - %s (%s)%s\n", task.Key, task.Status, reason)
			} else {
				fmt.Printf("  - %s (%s)\n", task.Key, task.Status)
			}
		}
		fmt.Println()

		cli.Info("Use --force to complete all tasks regardless of status")
		os.Exit(3)
	}

	// Complete all tasks in a transaction
	agent := getAgentIdentifier("")
	completedTaskCount := 0
	var affectedTaskKeys []string

	for _, task := range allTasks {
		// Skip already completed tasks
		if task.Status == models.TaskStatusCompleted {
			completedTaskCount++
			continue
		}

		// Mark as completed
		if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, nil, true); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to complete task %s: %v", task.Key, err))
			os.Exit(2)
		}
		completedTaskCount++
		affectedTaskKeys = append(affectedTaskKeys, task.Key)
	}

	// Update progress for all features
	for _, feature := range features {
		if err := featureRepo.UpdateProgress(ctx, feature.ID); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update progress for feature %s: %v", feature.Key, err))
			os.Exit(2)
		}
	}

	// Epic progress is calculated on-demand, no need to update

	// Output results
	if cli.GlobalConfig.JSON {
		// Build status breakdown map for JSON
		statusBreakdownMap := make(map[string]int)
		statusBreakdownMap["todo"] = totalStatusBreakdown[models.TaskStatusTodo]
		statusBreakdownMap["in_progress"] = totalStatusBreakdown[models.TaskStatusInProgress]
		statusBreakdownMap["blocked"] = totalStatusBreakdown[models.TaskStatusBlocked]
		statusBreakdownMap["ready_for_review"] = totalStatusBreakdown[models.TaskStatusReadyForReview]
		statusBreakdownMap["completed"] = completedCount + completedTaskCount

		result := map[string]interface{}{
			"epic_key":           epicKey,
			"feature_count":      len(features),
			"total_task_count":   len(allTasks),
			"completed_count":    completedCount + completedTaskCount,
			"status_breakdown":   statusBreakdownMap,
			"affected_tasks":     affectedTaskKeys,
			"force_completed":    force && hasIncomplete,
		}
		return cli.OutputJSON(result)
	}

	// Human-readable output
	if force && hasIncomplete {
		// Had to force complete
		todoCount := totalStatusBreakdown[models.TaskStatusTodo]
		inProgressCount := totalStatusBreakdown[models.TaskStatusInProgress]
		blockedCount := totalStatusBreakdown[models.TaskStatusBlocked]
		readyCount := totalStatusBreakdown[models.TaskStatusReadyForReview]

		breakdownStr := fmt.Sprintf("%d todo, %d in_progress, %d blocked, %d ready_for_review", todoCount, inProgressCount, blockedCount, readyCount)
		cli.Success(fmt.Sprintf("Epic %s completed: Force-completed %d tasks (%s)", epicKey, completedTaskCount, breakdownStr))
	} else {
		// All tasks were already completed or ready for review
		cli.Success(fmt.Sprintf("Epic %s completed: %d/%d tasks completed across %d feature(s)", epicKey, len(allTasks), len(allTasks), len(features)))
	}

	return nil
}

// runEpicDelete executes the epic delete command
func runEpicDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
		return fmt.Errorf("database path error")
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		cli.Error("Error: Database error. Run with --verbose for details.")
		if cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
		}
		os.Exit(2)
	}

	// Get repositories
	repoDb := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Get epic by key to verify it exists
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	// Check for child features
	features, err := featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to check for features: %v", err))
		os.Exit(1)
	}

	// If there are features, require --force flag
	if len(features) > 0 && !force {
		cli.Error(fmt.Sprintf("Error: Epic %s has %d feature(s)", epicKey, len(features)))
		cli.Warning("This will CASCADE DELETE all features and their tasks")
		cli.Info(fmt.Sprintf("Use --force to confirm deletion: shark epic delete %s --force", epicKey))
		os.Exit(1)
	}

	// Delete epic from database (CASCADE will handle features/tasks)
	if err := epicRepo.Delete(ctx, epic.ID); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to delete epic: %v", err))
		os.Exit(1)
	}

	cli.Success(fmt.Sprintf("Epic %s deleted successfully", epicKey))
	if len(features) > 0 {
		cli.Warning(fmt.Sprintf("Cascade deleted %d feature(s) and their tasks", len(features)))
	}
	return nil
}
