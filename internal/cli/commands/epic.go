package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
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

func init() {
	// Register epic command with root
	cli.RootCmd.AddCommand(epicCmd)

	// Add subcommands
	epicCmd.AddCommand(epicListCmd)
	epicCmd.AddCommand(epicGetCmd)
	epicCmd.AddCommand(epicStatusCmd)

	// Add flags for list command
	epicListCmd.Flags().String("sort-by", "", "Sort by: key, progress, status (default: key)")
	epicListCmd.Flags().String("status", "", "Filter by status: draft, active, completed, archived")
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
