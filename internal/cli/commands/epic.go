package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

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
  pm epic list                 List all epics
  pm epic get E04             Get epic details with progress
  pm epic status              Show status of all epics`,
}

// epicListCmd lists epics
var epicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all epics",
	Long: `List all epics with progress information.

Examples:
  pm epic list                 List all epics
  pm epic list --json          Output as JSON`,
	RunE: runEpicList,
}

// epicGetCmd gets a specific epic
var epicGetCmd = &cobra.Command{
	Use:   "get <epic-key>",
	Short: "Get epic details",
	Long: `Display detailed information about a specific epic including all features and progress.

Examples:
  pm epic get E04              Get epic details
  pm epic get E04 --json       Output as JSON`,
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

// epicCreateCmd creates a new epic
var epicCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new epic",
	Long: `Create a new epic with auto-assigned key, folder structure, and database entry.

The epic key is automatically assigned as the next available E## number.

Examples:
  shark epic create "User Authentication System"
  shark epic create "User Auth" --description="Add OAuth and MFA"`,
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
)

func init() {
	// Register epic command with root
	cli.RootCmd.AddCommand(epicCmd)

	// Add subcommands
	epicCmd.AddCommand(epicListCmd)
	epicCmd.AddCommand(epicGetCmd)
	epicCmd.AddCommand(epicStatusCmd)
	epicCmd.AddCommand(epicCreateCmd)
	epicCmd.AddCommand(epicDeleteCmd)

	// Add flags for list command
	epicListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	epicListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")

	// Add flags for create command
	epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")

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
		cli.Info("Use 'pm epic list' to see available epics")
		os.Exit(1)
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
			"created_at":     epic.CreatedAt,
			"updated_at":     epic.UpdatedAt,
			"features":       featuresWithDetails,
		}
		return cli.OutputJSON(result)
	}

	// Output as formatted text
	renderEpicDetails(epic, epicProgress, featuresWithDetails)
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
func renderEpicDetails(epic *models.Epic, progress float64, features []FeatureWithDetails) {
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

	// Get next epic key
	nextKey, err := getNextEpicKey(ctx, epicRepo)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to get next epic key: %v", err))
		os.Exit(1)
	}

	// Generate slug from title
	slug := generateSlug(epicTitle)
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
		EpicSlug:    epicSlug,
		Title:       epicTitle,
		Description: epicCreateDescription,
		FilePath:    fmt.Sprintf("%s/epic.md", epicDir),
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

	// Write epic.md file
	epicFilePath := fmt.Sprintf("%s/epic.md", epicDir)
	if err := os.WriteFile(epicFilePath, buf.Bytes(), 0644); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to write epic file: %v", err))
		os.Exit(1)
	}

	// Create database entry with key (E##) not full slug
	epic := &models.Epic{
		Key:          nextKey,
		Title:        epicTitle,
		Description:  &epicCreateDescription,
		Status:       models.EpicStatusDraft,
		Priority:     models.PriorityMedium,
		BusinessValue: nil,
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to create epic in database: %v", err))
		// Clean up file on DB error
		os.RemoveAll(epicDir)
		os.Exit(1)
	}

	// Success output
	cli.Success(fmt.Sprintf("Epic created successfully!"))
	fmt.Println()
	fmt.Printf("Epic Key:  %s\n", epicSlug)
	fmt.Printf("Directory: %s\n", epicDir)
	fmt.Printf("File:      %s\n", epicFilePath)
	fmt.Printf("Database:  ✓ Epic record created (ID: %d)\n", epic.ID)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Edit the epic.md file to add details")
	fmt.Printf("2. Create features with: shark feature create --epic=%s \"Feature title\"\n", nextKey)

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
