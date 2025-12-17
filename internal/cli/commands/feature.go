package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// FeatureWithTaskCount wraps a Feature with task count
type FeatureWithTaskCount struct {
	*models.Feature
	TaskCount int `json:"task_count"`
}

// featureCmd represents the feature command group
var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage features",
	Long: `Query and manage features within epics.

Examples:
  pm feature list              List all features
  pm feature get E04-F02      Get feature details
  pm feature list --epic=E04  List features in epic E04`,
}

// featureListCmd lists features
var featureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List features",
	Long: `List features with optional filtering by epic.

Examples:
  pm feature list              List all features
  pm feature list --epic=E04   List features in epic E04
  pm feature list --json       Output as JSON
  pm feature list --status=active  Filter by status
  pm feature list --sort-by=progress  Sort by progress`,
	RunE: runFeatureList,
}

// featureGetCmd gets a specific feature
var featureGetCmd = &cobra.Command{
	Use:   "get <feature-key>",
	Short: "Get feature details",
	Long: `Display detailed information about a specific feature including all tasks and progress.

Examples:
  pm feature get E04-F02       Get feature details
  pm feature get E04-F02 --json  Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureGet,
}

// featureCreateCmd creates a new feature
var featureCreateCmd = &cobra.Command{
	Use:   "create --epic=<key> <title>",
	Short: "Create a new feature",
	Long: `Create a new feature with auto-assigned key, folder structure, and database entry.

The feature key is automatically assigned as the next available F## number within the epic.

Examples:
  shark feature create --epic=E01 "OAuth Login Integration"
  shark feature create --epic=E01 "OAuth Login" --description="Add OAuth 2.0 support"`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureCreate,
}

var (
	featureCreateEpic        string
	featureCreateDescription string
)

func init() {
	// Register feature command with root
	cli.RootCmd.AddCommand(featureCmd)

	// Add subcommands
	featureCmd.AddCommand(featureListCmd)
	featureCmd.AddCommand(featureGetCmd)
	featureCmd.AddCommand(featureCreateCmd)

	// Add flags for list command
	featureListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
	featureListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")
	featureListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")

	// Add flags for create command
	featureCreateCmd.Flags().StringVar(&featureCreateEpic, "epic", "", "Epic key (e.g., E01)")
	featureCreateCmd.Flags().StringVar(&featureCreateDescription, "description", "", "Feature description (optional)")
	featureCreateCmd.MarkFlagRequired("epic")
}

// runFeatureList executes the feature list command
func runFeatureList(cmd *cobra.Command, args []string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get flags
	epicFilter, _ := cmd.Flags().GetString("epic")
	statusFilter, _ := cmd.Flags().GetString("status")
	sortBy, _ := cmd.Flags().GetString("sort-by")

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
	featureRepo := repository.NewFeatureRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)

	var features []*models.Feature

	// Apply filters using repository methods
	if epicFilter != "" {
		// Get epic by key
		epic, err := epicRepo.GetByKey(ctx, epicFilter)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicFilter))
			cli.Info("Use 'pm epic list' to see available epics")
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

		featuresWithTaskCount = append(featuresWithTaskCount, FeatureWithTaskCount{
			Feature:   feature,
			TaskCount: taskCount,
		})
	}

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
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Get feature by key
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'pm feature list' to see available features")
		os.Exit(1)
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

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		result := map[string]interface{}{
			"id":               feature.ID,
			"epic_id":          feature.EpicID,
			"key":              feature.Key,
			"title":            feature.Title,
			"description":      feature.Description,
			"status":           feature.Status,
			"progress_pct":     feature.ProgressPct,
			"created_at":       feature.CreatedAt,
			"updated_at":       feature.UpdatedAt,
			"tasks":            tasks,
			"status_breakdown": statusBreakdown,
		}
		return cli.OutputJSON(result)
	}

	// Output as formatted text
	renderFeatureDetails(feature, tasks, statusBreakdown)
	return nil
}

// renderFeatureListTable renders features as a table
func renderFeatureListTable(features []FeatureWithTaskCount, epicFilter string) {
	// Create table data
	tableData := pterm.TableData{
		{"Key", "Title", "Epic ID", "Status", "Progress", "Tasks"},
	}

	for _, feature := range features {
		// Truncate long titles to fit in 80 columns
		title := feature.Title
		if len(title) > 25 {
			title = title[:22] + "..."
		}

		// Format progress with 1 decimal place
		progress := fmt.Sprintf("%.1f%%", feature.ProgressPct)

		tableData = append(tableData, []string{
			feature.Key,
			title,
			fmt.Sprintf("%d", feature.EpicID),
			string(feature.Status),
			progress,
			fmt.Sprintf("%d", feature.TaskCount),
		})
	}

	// Render table
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// renderFeatureDetails renders feature details with tasks table
func renderFeatureDetails(feature *models.Feature, tasks []*models.Task, statusBreakdown map[models.TaskStatus]int) {
	// Print feature metadata
	pterm.DefaultSection.Printf("Feature: %s", feature.Key)
	fmt.Println()

	// Feature info
	info := [][]string{
		{"Title", feature.Title},
		{"Epic ID", fmt.Sprintf("%d", feature.EpicID)},
		{"Status", string(feature.Status)},
		{"Progress", fmt.Sprintf("%.1f%%", feature.ProgressPct)},
	}

	if feature.Description != nil && *feature.Description != "" {
		info = append(info, []string{"Description", *feature.Description})
	}

	// Render info table
	pterm.DefaultTable.WithData(info).Render()
	fmt.Println()

	// Task status breakdown
	if len(statusBreakdown) > 0 {
		pterm.DefaultSection.Println("Task Status Breakdown")
		fmt.Println()
		breakdownData := pterm.TableData{}
		for status, count := range statusBreakdown {
			breakdownData = append(breakdownData, []string{
				string(status),
				fmt.Sprintf("%d", count),
			})
		}
		pterm.DefaultTable.WithData(breakdownData).Render()
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

		tableData = append(tableData, []string{
			task.Key,
			title,
			string(task.Status),
			fmt.Sprintf("%d", task.Priority),
			agent,
		})
	}

	// Render tasks table
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
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

	// Get title from positional argument
	featureTitle := args[0]

	// Validate epic key format
	if !isValidEpicKey(featureCreateEpic) {
		cli.Error("Error: Invalid epic key format. Must be E## (e.g., E01, E02)")
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
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Verify epic exists in database
	epic, err := epicRepo.GetByKey(ctx, featureCreateEpic)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s not found in database", featureCreateEpic))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	// Auto-assign next feature key for this epic
	nextKey, err := getNextFeatureKey(ctx, featureRepo, epic.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to generate feature key: %v", err))
		os.Exit(1)
	}

	// Generate slug from title
	slug := generateSlug(featureTitle)
	featureSlug := fmt.Sprintf("%s-%s-%s", featureCreateEpic, nextKey, slug)

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
		FilePath:    fmt.Sprintf("%s/prd.md", featureDir),
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

	// Write prd.md file
	featureFilePath := fmt.Sprintf("%s/prd.md", featureDir)
	if err := os.WriteFile(featureFilePath, buf.Bytes(), 0644); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to write feature file: %v", err))
		os.Exit(1)
	}

	// Create database entry with key (E##-F##) not full slug
	featureKey := fmt.Sprintf("%s-%s", featureCreateEpic, nextKey)
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       featureTitle,
		Description: &featureCreateDescription,
		Status:      models.FeatureStatusDraft,
		ProgressPct: 0.0,
	}

	if err := featureRepo.Create(ctx, feature); err != nil {
		// Rollback: delete the created directory
		os.RemoveAll(featureDir)
		cli.Error(fmt.Sprintf("Error: Failed to create feature in database: %v", err))
		cli.Info("Rolled back file creation")
		os.Exit(1)
	}

	// Success output
	cli.Success(fmt.Sprintf("Feature created successfully!"))
	fmt.Println()
	fmt.Printf("Feature Key: %s\n", featureSlug)
	fmt.Printf("Epic:        %s\n", featureCreateEpic)
	fmt.Printf("Directory:   %s\n", featureDir)
	fmt.Printf("File:        %s\n", featureFilePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Edit the prd.md file to add details")
	fmt.Printf("2. Create tasks with: shark task create --epic=%s --feature=%s --title=\"Task title\" --agent=backend\n", featureCreateEpic, nextKey)

	return nil
}

// getNextFeatureKey determines the next available feature key (F##) for an epic
func getNextFeatureKey(ctx context.Context, featureRepo *repository.FeatureRepository, epicID int64) (string, error) {
	// Get all features for this epic
	features, err := featureRepo.ListByEpic(ctx, epicID)
	if err != nil {
		return "", fmt.Errorf("failed to list features: %w", err)
	}

	// Find the maximum feature number
	maxNum := 0
	for _, feature := range features {
		// Feature key format in DB is E##-F##, extract the F## part
		var epicNum, featureNum int
		if _, err := fmt.Sscanf(feature.Key, "E%d-F%d", &epicNum, &featureNum); err == nil {
			if featureNum > maxNum {
				maxNum = featureNum
			}
		}
	}

	// Return next available number
	return fmt.Sprintf("F%02d", maxNum+1), nil
}
