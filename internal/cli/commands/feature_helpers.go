package commands

// feature_helpers.go contains presentation/rendering helpers for feature commands.
// These are display-only functions that live in the command layer (not the service layer).
// Some of these helpers require direct repository access for batch data fetching
// (e.g., renderFeatureListTable needs task status breakdowns for health indicators).

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
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/fileops"
	"github.com/jwwelbor/shark-task-manager/internal/formatters"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/pathresolver"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// FeatureWithTaskCount wraps a Feature model with additional task count information
// used for list display.
type FeatureWithTaskCount struct {
	*models.Feature
	TaskCount    int    `json:"task_count"`
	StatusSource string `json:"status_source"`
}

// FeatureListItemJSON is the JSON output structure for feature list items.
type FeatureListItemJSON struct {
	Key            string      `json:"key"`
	Title          string      `json:"title"`
	EpicID         int64       `json:"epic_id"`
	Status         string      `json:"status"`
	StatusOverride bool        `json:"status_override"`
	Health         string      `json:"health"`
	Progress       interface{} `json:"progress"`
	Notes          string      `json:"notes"`
	TaskCount      int         `json:"task_count"`
}

// FeatureTemplateData holds data for rendering a feature markdown template.
type FeatureTemplateData struct {
	EpicKey     string
	FeatureKey  string
	FeatureSlug string
	Title       string
	Description string
	FilePath    string
	Date        string
}

// getRelativePathFeature converts an absolute path to relative path from project root.
func getRelativePathFeature(absPath string, projectRoot string) string {
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return absPath // Fall back to absolute path if conversion fails
	}
	return relPath
}

// backupDatabaseOnForceFeature creates a backup when --force flag is used.
// DEPRECATED: Use CreateBackupIfForce from file_assignment.go instead.
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

// filterFeaturesByCompletedStatus filters out completed features unless showAll is true
// or an explicit status filter is set.
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

// sortFeatures sorts features by the specified field.
func sortFeatures(features []FeatureWithTaskCount, sortBy string) {
	if sortBy == "" || sortBy == "key" {
		sortFeaturesByKey(features)
	} else if sortBy == "progress" {
		sortFeaturesByProgress(features)
	} else if sortBy == "status" {
		sortFeaturesByStatus(features)
	}
}

// sortFeaturesByKey sorts features by key.
func sortFeaturesByKey(features []FeatureWithTaskCount) {
	for i := 0; i < len(features); i++ {
		for j := i + 1; j < len(features); j++ {
			if features[i].Key > features[j].Key {
				features[i], features[j] = features[j], features[i]
			}
		}
	}
}

// sortFeaturesByProgress sorts features by progress (ascending).
func sortFeaturesByProgress(features []FeatureWithTaskCount) {
	for i := 0; i < len(features); i++ {
		for j := i + 1; j < len(features); j++ {
			if features[i].ProgressPct > features[j].ProgressPct {
				features[i], features[j] = features[j], features[i]
			}
		}
	}
}

// sortFeaturesByStatus sorts features by status (draft, active, completed, archived).
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

// renderFeaturePlanning renders a feature in planning mode showing workflow position.
func renderFeaturePlanning(info *services.FeatureDisplayInfo) {
	feature := info.Feature

	// Print feature metadata
	pterm.DefaultSection.Printf("Feature: %s", feature.Key)
	fmt.Println()

	// Build info rows
	featureInfo := [][]string{
		{"Title", feature.Title},
		{"Epic ID", fmt.Sprintf("%d", feature.EpicID)},
		{"Status", fmt.Sprintf("%s (workflow)", string(feature.Status))},
	}

	if info.Phase != "" {
		featureInfo = append(featureInfo, []string{"Phase", info.Phase})
	}

	if info.PhaseDescription != "" {
		featureInfo = append(featureInfo, []string{"Phase Description", info.PhaseDescription})
	}

	if info.ResolvedPath != "" {
		featureInfo = append(featureInfo, []string{"Path", info.ResolvedPath})
	}

	if feature.Description != nil && *feature.Description != "" {
		featureInfo = append(featureInfo, []string{"Description", *feature.Description})
	}

	_ = pterm.DefaultTable.WithData(featureInfo).Render()
	fmt.Println()

	// Workflow position
	if info.WorkflowPosition != nil {
		pterm.DefaultSection.Println("Workflow Position")
		fmt.Println()

		for i, st := range info.WorkflowPosition.Statuses {
			marker := "  "
			if i == info.WorkflowPosition.CurrentIndex {
				marker = "> "
			}
			label := st
			if i < info.WorkflowPosition.CurrentIndex {
				label = fmt.Sprintf("%s (done)", st)
			}
			fmt.Printf("%s%s\n", marker, label)
		}
		fmt.Println()
	}

	// Planning mode message about tasks
	if len(info.Tasks) == 0 {
		pterm.Info.Println("No tasks yet (feature is still being refined)")
	}

	// Display orchestrator action
	displayOrchestratorAction(info.OrchestratorAction)

	// Display related documents
	renderRelatedDocuments(info.RelatedDocs)
}

// renderFeatureListTable renders features as a table.
// Note: This function uses repository.NewTaskRepository for batch status breakdown fetching
// which is needed for health indicators. This is presentation data fetching (display only).
func renderFeatureListTable(features []FeatureWithTaskCount, epicFilter string, ctx context.Context, repoDb *repository.DB) {
	// Create table data with reordered columns (removed Notes, Health next to Status)
	tableData := pterm.TableData{
		{"Key", "Title", "Progress", "Status", "Health"},
	}

	// Get task repository for additional data
	taskRepo := repository.NewTaskRepository(repoDb)

	// Batch fetch status breakdowns for all features to avoid N+1 query
	featureIDs := make([]int64, len(features))
	for i, feature := range features {
		featureIDs[i] = feature.ID
	}
	statusBreakdownBatch, err := taskRepo.GetStatusBreakdownMapBatch(ctx, featureIDs)
	if err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to batch fetch status breakdowns: %v\n", err)
	}
	if statusBreakdownBatch == nil {
		statusBreakdownBatch = make(map[int64]map[models.TaskStatus]int)
	}

	// Load config once for all features
	configPath, cfgErr := cli.GetConfigPath()
	if cfgErr != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get config path: %v\n", cfgErr)
	}
	cfg, cfgErr := config.LoadWorkflowConfig(configPath)
	if cfgErr != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", cfgErr)
	}

	// Get project root for WorkflowService
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}
	workflowService := workflow.NewService(projectRoot)

	for _, feature := range features {
		// Reduce title to 45 characters for 85-90 char terminal width
		title := feature.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}

		// Get status breakdown from batch result
		statusBreakdown := statusBreakdownBatch[feature.ID]
		if statusBreakdown == nil {
			statusBreakdown = make(map[models.TaskStatus]int)
		}

		// Convert to string-keyed map for progress calculation
		statusCounts := make(map[string]int)
		for taskStatus, count := range statusBreakdown {
			statusCounts[string(taskStatus)] = count
		}

		// Calculate health indicator
		health := calculateHealthIndicator(statusCounts, cfg)

		// Calculate progress with weighted ratio
		var progressDisplay string
		if cfg != nil {
			progress := status.CalculateProgress(statusCounts, cfg)
			progressDisplay = fmt.Sprintf("%.0f%% (%s)", progress.WeightedPct, progress.WeightedRatio)
		} else {
			// Fallback to simple percentage if config unavailable
			progressDisplay = fmt.Sprintf("%.0f%%", feature.ProgressPct)
		}

		// Apply color coding using workflow service
		formatted := workflowService.FormatStatusForDisplay(string(feature.Status), !cli.GlobalConfig.NoColor)
		statusDisplay := formatted.Colored
		// Add indicator for manual override
		if feature.StatusOverride {
			statusDisplay += "*"
		}

		tableData = append(tableData, []string{
			feature.Key,
			title,
			progressDisplay,
			statusDisplay,
			health,
		})
	}

	// Render table
	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// calculateHealthIndicator calculates health emoji based on status breakdown.
// Returns: 🔴 (at risk, 3+ blocked), 🟡 (attention needed), or 🟢 (healthy).
func calculateHealthIndicator(statusCounts map[string]int, cfg *config.WorkflowConfig) string {
	if cfg == nil {
		// Fallback to hardcoded behavior if no config
		blockedCount := statusCounts[string(models.TaskStatus("blocked"))]
		if blockedCount >= 3 {
			return "🔴"
		}
		if blockedCount >= 1 || statusCounts["ready_for_approval"] > 0 {
			return "🟡"
		}
		return "🟢"
	}

	// Config-driven approach: check statuses with blocks_feature: true
	blockingCount := 0
	for s, count := range statusCounts {
		if meta, ok := cfg.StatusMetadata[s]; ok && meta.BlocksFeature {
			blockingCount += count
		}
	}

	// At risk: 3 or more blocking tasks
	if blockingCount >= 3 {
		return "🔴"
	}

	// Attention needed: 1+ blocking tasks
	if blockingCount >= 1 {
		return "🟡"
	}

	// Healthy: on track
	return "🟢"
}

// generateNotesColumn generates status summary notes for feature list.
// Shows counts of tasks in blocking statuses (blocks_feature: true).
func generateNotesColumn(statusCounts map[string]int, cfg *config.WorkflowConfig) string {
	if cfg == nil {
		// Fallback to hardcoded behavior if no config
		parts := []string{}
		if blocked := statusCounts[string(models.TaskStatus("blocked"))]; blocked > 0 {
			parts = append(parts, fmt.Sprintf("%d blocked", blocked))
		}
		if ready := statusCounts["ready_for_approval"]; ready > 0 {
			parts = append(parts, fmt.Sprintf("%d ready", ready))
		}
		if len(parts) == 0 {
			return "[on track]"
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}

	// Config-driven approach: show statuses with blocks_feature: true
	parts := []string{}
	for s, count := range statusCounts {
		if count > 0 {
			if meta, ok := cfg.StatusMetadata[s]; ok && meta.BlocksFeature {
				// Use a friendly label (e.g., "ready_for_approval" -> "ready")
				label := s
				if strings.HasPrefix(s, "ready_for_") {
					label = strings.TrimPrefix(s, "ready_for_")
				}
				parts = append(parts, fmt.Sprintf("%d %s", count, label))
			}
		}
	}

	// Return formatted notes
	if len(parts) == 0 {
		return "[on track]"
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

// renderFeatureDetails renders feature details with tasks table.
// statusBreakdown is workflow-ordered with metadata.
// workflowService is used for color formatting (can be nil for no colors).
// progressInfo, workSummary, and actionItems provide enhanced status information.
func renderFeatureDetails(feature *models.Feature, tasks []*models.Task, statusBreakdown []workflow.StatusCount, path, filename string, relatedDocs []*models.Document, workflowService *workflow.Service, progressInfo *status.ProgressInfo, workSummary *status.WorkSummary, actionItems *status.ActionItems, notes []*models.EntityNote, contextData *models.ContextData) {
	// Determine if colors should be enabled
	colorEnabled := !cli.GlobalConfig.NoColor && workflowService != nil

	// Print feature metadata
	pterm.DefaultSection.Printf("Feature: %s", feature.Key)

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

	// Feature info - use weighted progress for display
	progressDisplay := fmt.Sprintf("%.0f%%", feature.ProgressPct)
	if progressInfo != nil {
		progressDisplay = fmt.Sprintf("%.0f%%", progressInfo.WeightedPct)
	}

	info := [][]string{
		{"Title", feature.Title},
		{"Epic ID", fmt.Sprintf("%d", feature.EpicID)},
		{"Status", featureStatusDisplay},
		{"Progress", progressDisplay},
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
	fmt.Println()
	_ = pterm.DefaultTable.WithData(info).Render()

	// Related documents section
	if len(relatedDocs) > 0 {
		pterm.DefaultSection.Println("Related Documents")
		for _, doc := range relatedDocs {
			fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
		}
	}

	// Task status breakdown (workflow-ordered with colored status names)
	if len(statusBreakdown) > 0 {
		pterm.DefaultSection.Println("Task Status Breakdown")
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
	}

	// Progress breakdown section (weighted vs completion)
	if progressInfo != nil {
		pterm.DefaultSection.Println("Progress Breakdown")
		progressData := pterm.TableData{
			{"Metric", "Value", "Ratio"},
			{"Weighted Progress", fmt.Sprintf("%.0f%%", progressInfo.WeightedPct), progressInfo.WeightedRatio},
			{"Completion", fmt.Sprintf("%.0f%%", progressInfo.CompletionPct), progressInfo.CompletionRatio},
		}
		_ = pterm.DefaultTable.WithHasHeader().WithData(progressData).Render()
	}

	// Work summary section (who's doing what)
	if workSummary != nil && workSummary.TotalTasks > 0 {
		pterm.DefaultSection.Println("Work Summary")
		workData := [][]string{}
		if workSummary.CompletedTasks > 0 {
			workData = append(workData, []string{"✅ Completed", fmt.Sprintf("%d/%d", workSummary.CompletedTasks, workSummary.TotalTasks)})
		}
		if workSummary.AgentWork > 0 {
			workData = append(workData, []string{"🤖 Agent Work", fmt.Sprintf("%d tasks", workSummary.AgentWork)})
		}
		if workSummary.HumanWork > 0 {
			workData = append(workData, []string{"👤 Human Work", fmt.Sprintf("%d tasks", workSummary.HumanWork)})
		}
		if workSummary.BlockedWork > 0 {
			workData = append(workData, []string{"🚫 Blocked", fmt.Sprintf("%d tasks", workSummary.BlockedWork)})
		}
		if workSummary.NotStarted > 0 {
			workData = append(workData, []string{"⏳ Not Started", fmt.Sprintf("%d tasks", workSummary.NotStarted)})
		}
		if len(workData) > 0 {
			_ = pterm.DefaultTable.WithData(workData).Render()
		}
	}

	// Action items section (tasks needing attention)
	if actionItems != nil {
		hasActionItems := len(actionItems.AwaitingApproval) > 0 || len(actionItems.Blocked) > 0 || len(actionItems.InProgress) > 0
		if hasActionItems {
			pterm.DefaultSection.Println("Action Items")

			// Awaiting approval
			if len(actionItems.AwaitingApproval) > 0 {
				fmt.Println()
				pterm.DefaultBox.WithTitle("⏳ Awaiting Approval").Println(fmt.Sprintf("%d tasks", len(actionItems.AwaitingApproval)))
				for _, item := range actionItems.AwaitingApproval {
					ageStr := ""
					if item.AgeDays != nil {
						ageStr = fmt.Sprintf(" (%d days)", *item.AgeDays)
					}
					fmt.Printf("  - %s: %s%s\n", item.TaskKey, item.Title, ageStr)
				}
			}

			// Blocked
			if len(actionItems.Blocked) > 0 {
				fmt.Println()
				pterm.DefaultBox.WithTitle("🚫 Blocked").Println(fmt.Sprintf("%d tasks", len(actionItems.Blocked)))
				for _, item := range actionItems.Blocked {
					reasonStr := ""
					if item.BlockedReason != nil && *item.BlockedReason != "" {
						reasonStr = fmt.Sprintf(" - %s", *item.BlockedReason)
					}
					fmt.Printf("  - %s: %s%s\n", item.TaskKey, item.Title, reasonStr)
				}
			}

			// In progress (summary only)
			if len(actionItems.InProgress) > 0 {
				fmt.Println()
				pterm.DefaultBox.WithTitle("🔄 In Progress").Println(fmt.Sprintf("%d tasks", len(actionItems.InProgress)))
			}
		}
	}

	// Notes section (only if notes exist)
	if len(notes) > 0 {
		maxDisplay := 10
		totalNotes := len(notes)
		if totalNotes > maxDisplay {
			pterm.DefaultSection.Printf("Notes (showing %d of %d)", maxDisplay, totalNotes)
		} else {
			pterm.DefaultSection.Printf("Notes (%d)", totalNotes)
		}
		fmt.Println()

		displayCount := totalNotes
		if displayCount > maxDisplay {
			displayCount = maxDisplay
		}
		for i := totalNotes - displayCount; i < totalNotes; i++ {
			note := notes[i]
			dateStr := note.CreatedAt.Format("2006-01-02")
			content := note.Content
			if len(content) > 80 {
				content = content[:77] + "..."
			}
			fmt.Printf("  [%s] %s  %s\n", note.NoteType, dateStr, content)
		}
		fmt.Println()
	}

	// Context section (only if context data exists and has content)
	if contextData != nil {
		hasContextContent := contextData.Progress != nil ||
			len(contextData.ImplementationDecisions) > 0 ||
			len(contextData.OpenQuestions) > 0 ||
			len(contextData.Blockers) > 0 ||
			len(contextData.AcceptanceCriteriaStatus) > 0
		if hasContextContent {
			pterm.DefaultSection.Println("Context")
			fmt.Println()
			printContextData(contextData)
		}
	}

	// Check if all tasks are completed
	allTasksCompleted := len(tasks) > 0 && feature.ProgressPct >= 100.0
	if allTasksCompleted {
		fmt.Println()
		pterm.Success.Println("All tasks completed! Feature is ready for approval.")
	}

	// Tasks section
	if len(tasks) == 0 {
		fmt.Println()
		pterm.Info.Println("No tasks found for this feature")
		return
	}

	fmt.Println()
	pterm.DefaultSection.Printf("Tasks (%d total)", len(tasks))

	// Use centralized task table formatter
	tableConfig := formatters.FeatureGetTaskTableConfig()
	tableConfig.ColorEnabled = colorEnabled
	_ = formatters.RenderTaskTable(tasks, workflowService, tableConfig)
}

// outputFeatureListJSON renders the feature list as enhanced JSON with health and progress info.
func outputFeatureListJSON(ctx context.Context, featuresWithTaskCount []FeatureWithTaskCount, repoDb *repository.DB) error {
	taskRepo := repository.NewTaskRepository(repoDb)

	// Load workflow config
	configPath, _ := cli.GetConfigPath()
	var cfg *config.WorkflowConfig
	if configPath != "" {
		cfg, _ = config.LoadWorkflowConfig(configPath)
	}

	// Batch fetch status breakdowns for all features to avoid N+1 query
	featureIDs := make([]int64, len(featuresWithTaskCount))
	for i, f := range featuresWithTaskCount {
		featureIDs[i] = f.ID
	}
	statusBreakdownBatch, _ := taskRepo.GetStatusBreakdownMapBatch(ctx, featureIDs)
	if statusBreakdownBatch == nil {
		statusBreakdownBatch = make(map[int64]map[models.TaskStatus]int)
	}

	enhancedResults := make([]FeatureListItemJSON, 0, len(featuresWithTaskCount))
	for _, feature := range featuresWithTaskCount {
		statusBreakdown := statusBreakdownBatch[feature.ID]
		if statusBreakdown == nil {
			statusBreakdown = make(map[models.TaskStatus]int)
		}
		statusCounts := make(map[string]int)
		for taskStatus, count := range statusBreakdown {
			statusCounts[string(taskStatus)] = count
		}

		health := calculateHealthIndicator(statusCounts, cfg)

		var progressInfo interface{}
		if cfg != nil {
			progress := status.CalculateProgress(statusCounts, cfg)
			progressInfo = map[string]interface{}{
				"weighted_pct": progress.WeightedPct, "completion_pct": progress.CompletionPct,
				"weighted_ratio": progress.WeightedRatio, "completion_ratio": progress.CompletionRatio,
				"total_tasks": progress.TotalTasks,
			}
		} else {
			progressInfo = map[string]interface{}{"pct": feature.ProgressPct}
		}

		notes := generateNotesColumn(statusCounts, cfg)

		enhancedResults = append(enhancedResults, FeatureListItemJSON{
			Key: feature.Key, Title: feature.Title, EpicID: feature.EpicID,
			Status: string(feature.Status), StatusOverride: feature.StatusOverride,
			Health: health, Progress: progressInfo, Notes: notes,
			TaskCount: feature.TaskCount,
		})
	}

	return cli.OutputJSON(map[string]interface{}{
		"results": enhancedResults,
		"count":   len(enhancedResults),
	})
}

// FeatureGetData bundles all enriched data needed to render a feature get response.
type FeatureGetData struct {
	Tasks           []*models.Task
	StatusBreakdown []workflow.StatusCount
	DirPath         string
	Filename        string
	RelatedDocs     []*models.Document
	WorkflowService *workflow.Service
	WorkflowCfg     *config.WorkflowConfig
	ProgressInfo    *status.ProgressInfo
	WorkSummary     *status.WorkSummary
	ActionItems     *status.ActionItems
	Notes           []*models.EntityNote
	ContextData     *models.ContextData
}

// buildFeatureGetData gathers all enriched data needed to display a feature.
func buildFeatureGetData(ctx context.Context, feature *models.Feature) (*FeatureGetData, error) {
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return nil, err
	}

	projectRoot, _ := os.Getwd()

	featureRepo := repository.NewFeatureRepository(repoDb)
	epicRepo := repository.NewEpicRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)
	documentRepo := repository.NewDocumentRepository(repoDb)

	featureSvc := cli.GetFeatureService()

	// Update progress
	if err := featureSvc.RecalculateAndSetProgress(ctx, feature.ID); err != nil && cli.GlobalConfig.Verbose {
		fmt.Fprintf(os.Stderr, "Warning: Failed to update progress for feature %s: %v\n", feature.Key, err)
	}
	feature, _ = featureRepo.GetByID(ctx, feature.ID)

	tasks, _ := taskRepo.ListByFeature(ctx, feature.ID)
	statusBreakdown, _ := featureSvc.GetEnrichedTaskStatusBreakdown(ctx, feature.Key)

	// Resolve path
	var dirPath, filename string
	if projectRoot != "" {
		pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
		if absPath, pathErr := pathResolver.ResolveFeaturePath(ctx, feature.Key); pathErr == nil {
			relPath := getRelativePathFeature(absPath, projectRoot)
			dirPath = filepath.Dir(relPath) + "/"
			filename = filepath.Base(relPath)
		}
	}

	relatedDocs, _ := documentRepo.ListForFeature(ctx, feature.ID)
	if relatedDocs == nil {
		relatedDocs = []*models.Document{}
	}

	// Load workflow config for status calculations
	configPath, _ := cli.GetConfigPath()
	var workflowCfg *config.WorkflowConfig
	if configPath != "" {
		workflowCfg, _ = config.LoadWorkflowConfig(configPath)
	}

	// Convert statusBreakdown to map for calculations
	statusCountsMap := make(map[string]int)
	for _, sc := range statusBreakdown {
		statusCountsMap[sc.Status] = sc.Count
	}

	var progressInfo *status.ProgressInfo
	var workSummary *status.WorkSummary
	var actionItems *status.ActionItems
	if workflowCfg != nil {
		progressInfo = status.CalculateProgress(statusCountsMap, workflowCfg)
		workSummary = status.CalculateWorkRemaining(statusCountsMap, workflowCfg)
		actionItems = status.GetActionItems(tasks, workflowCfg)
	} else {
		progressInfo = &status.ProgressInfo{
			WeightedPct: feature.ProgressPct, CompletionPct: feature.ProgressPct,
			TotalTasks: len(tasks),
		}
		workSummary = &status.WorkSummary{TotalTasks: len(tasks), NotStarted: len(tasks)}
		actionItems = &status.ActionItems{
			AwaitingApproval: []*status.TaskActionItem{},
			Blocked:          []*status.TaskActionItem{},
			InProgress:       []*status.TaskActionItem{},
		}
	}

	// Fetch notes and context (graceful degradation)
	var featureNotes []*models.EntityNote
	if noteSvc, noteErr := cli.GetNoteService(ctx); noteErr == nil {
		featureNotes, _ = noteSvc.ListNotes(ctx, models.EntityTypeFeature, feature.Key, nil)
	}
	var featureContext *models.ContextData
	if ctxSvc, ctxErr := cli.GetContextService(ctx); ctxErr == nil {
		featureContext, _ = ctxSvc.GetContext(ctx, models.EntityTypeFeature, feature.Key)
	}

	workflowService := workflow.NewService(projectRoot)

	return &FeatureGetData{
		Tasks:           tasks,
		StatusBreakdown: statusBreakdown,
		DirPath:         dirPath,
		Filename:        filename,
		RelatedDocs:     relatedDocs,
		WorkflowService: workflowService,
		WorkflowCfg:     workflowCfg,
		ProgressInfo:    progressInfo,
		WorkSummary:     workSummary,
		ActionItems:     actionItems,
		Notes:           featureNotes,
		ContextData:     featureContext,
	}, nil
}

// fetchFeaturesWithTaskCount fetches features with progress and task count enrichment.
func fetchFeaturesWithTaskCount(ctx context.Context, repoDb *repository.DB, epicFilter, statusFilter string, showAll bool) ([]FeatureWithTaskCount, error) {
	featureRepo := repository.NewFeatureRepository(repoDb)

	var features []*models.Feature
	var err error
	if epicFilter != "" {
		epicRepo := repository.NewEpicRepository(repoDb)
		epic, epicErr := epicRepo.GetByKey(ctx, epicFilter)
		if epicErr != nil {
			cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicFilter))
			cli.Info("Use 'shark epic list' to see available epics")
			os.Exit(1)
		}
		if statusFilter != "" {
			features, err = featureRepo.ListByEpicAndStatus(ctx, epic.ID, models.FeatureStatus(statusFilter))
		} else {
			features, err = featureRepo.ListByEpic(ctx, epic.ID)
		}
	} else {
		featureSvc := cli.GetFeatureService()
		features, err = featureSvc.ListFeatures(ctx, services.FeatureFilters{Status: statusFilter})
	}
	if err != nil {
		return nil, err
	}

	if len(features) == 0 {
		return nil, nil
	}

	featureSvc := cli.GetFeatureService()
	result := make([]FeatureWithTaskCount, 0, len(features))
	for _, feature := range features {
		if err := featureSvc.RecalculateAndSetProgress(ctx, feature.ID); err != nil && cli.GlobalConfig.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update progress for feature %s: %v\n", feature.Key, err)
		}
		feature, _ = featureRepo.GetByID(ctx, feature.ID)
		if feature == nil {
			continue
		}
		taskCount, _ := featureRepo.GetTaskCount(ctx, feature.ID)
		statusSource := "calculated"
		if feature.StatusOverride {
			statusSource = "manual"
		}
		result = append(result, FeatureWithTaskCount{
			Feature:      feature,
			TaskCount:    taskCount,
			StatusSource: statusSource,
		})
	}

	return filterFeaturesByCompletedStatus(result, showAll, statusFilter), nil
}

// parseCreateFeatureInput parses command args and flags into a CreateFeatureInput plus returns title and projectRoot.
func parseCreateFeatureInput(cmd *cobra.Command, args []string) (services.CreateFeatureInput, string, string, error) {
	var featureTitle string
	positionalEpic, positionalTitle, err := ParseFeatureCreateArgs(args)
	if err == nil && positionalEpic != nil && positionalTitle != nil {
		featureTitle = *positionalTitle
		if featureCreateEpic != "" && featureCreateEpic != *positionalEpic {
			cli.Warning(fmt.Sprintf("Epic key provided both positionally (%s) and via flag (%s). Using positional value.", *positionalEpic, featureCreateEpic))
		}
		featureCreateEpic = *positionalEpic
	} else if len(args) == 1 && featureCreateEpic != "" {
		featureTitle = args[0]
	} else {
		fmt.Println("\nValid syntaxes:")
		fmt.Println("  shark feature create E07 \"Feature Title\"           (recommended)")
		fmt.Println("  shark feature create --epic=E07 \"Feature Title\"     (legacy)")
		return services.CreateFeatureInput{}, "", "", fmt.Errorf("%v", err)
	}

	if !isValidEpicKey(featureCreateEpic) {
		return services.CreateFeatureInput{}, "", "", fmt.Errorf("invalid epic key format. Must be E## (e.g., E01, E02)")
	}

	statusStr, _ := cmd.Flags().GetString("status")
	if statusStr == "" {
		statusStr = "draft"
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return services.CreateFeatureInput{}, "", "", fmt.Errorf("failed to get working directory: %w", err)
	}

	customFile := getFileFlagValue(cmd)
	var filePath *string
	if customFile != "" {
		_, relPath, err := resolveCustomFeatureFilePath(cmd, projectRoot, featureCreateForce)
		if err != nil {
			return services.CreateFeatureInput{}, "", "", err
		}
		filePath = &relPath
	}

	var execOrder *int
	if featureCreateExecutionOrder > 0 {
		execOrder = &featureCreateExecutionOrder
	}

	desc := featureCreateDescription
	input := services.CreateFeatureInput{
		EpicKey:        featureCreateEpic,
		Title:          featureTitle,
		Description:    &desc,
		Status:         statusStr,
		ExecutionOrder: execOrder,
		FilePath:       filePath,
		Force:          featureCreateForce,
	}
	return input, featureTitle, projectRoot, nil
}

// getFileFlagValue returns the value from --file, --filename, or --path flags (in that priority order).
func getFileFlagValue(cmd *cobra.Command) string {
	path, _ := cmd.Flags().GetString("path")
	if path != "" {
		return path
	}
	filename, _ := cmd.Flags().GetString("filename")
	if filename != "" {
		return filename
	}
	file, _ := cmd.Flags().GetString("file")
	return file
}

// resolveCustomFeatureFilePath resolves and validates a custom file path for a feature.
func resolveCustomFeatureFilePath(cmd *cobra.Command, projectRoot string, force bool) (string, string, error) {
	customFile := getFileFlagValue(cmd)
	if customFile == "" {
		return "", "", fmt.Errorf("no custom file path provided")
	}
	absPath, relPath, err := taskcreation.ValidateCustomFilename(customFile, projectRoot)
	if err != nil {
		return "", "", fmt.Errorf("invalid filename: %w", err)
	}
	dirPath := filepath.Dir(absPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create directory structure: %w", err)
	}
	return absPath, relPath, nil
}

// renderFeatureTemplate reads and renders the feature markdown template.
func renderFeatureTemplate(epicKey, featureKey, featureSlug, title, description, filePath string) ([]byte, error) {
	templatePath := "shark-templates/feature.md"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read feature template: %w (run 'shark init' to create templates)", err)
	}

	data := FeatureTemplateData{
		EpicKey:     epicKey,
		FeatureKey:  featureKey,
		FeatureSlug: featureSlug,
		Title:       title,
		Description: description,
		FilePath:    filePath,
		Date:        time.Now().Format("2006-01-02"),
	}

	tmpl, err := template.New("feature").Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse feature template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render feature template: %w", err)
	}
	return buf.Bytes(), nil
}

// writeFeatureFile writes a feature file to disk and returns whether it was linked and any error.
func writeFeatureFile(content []byte, featureFilePath, projectRoot string) (bool, error) {
	writer := fileops.NewEntityFileWriter()
	writeResult, err := writer.WriteEntityFile(fileops.WriteOptions{
		Content:        content,
		ProjectRoot:    projectRoot,
		FilePath:       featureFilePath,
		Verbose:        cli.GlobalConfig.Verbose,
		EntityType:     "feature",
		UseAtomicWrite: false,
		Logger:         func(message string) { cli.Info(message) },
	})
	if err != nil {
		return false, err
	}
	return writeResult.Linked, nil
}

// completeFeatureWithTasks handles the main logic of completing a feature that has tasks.
// Extracted from runFeatureComplete to keep the command handler thin.
func completeFeatureWithTasks(ctx context.Context, featureKey string, feature *models.Feature, tasks []*models.Task,
	featureRepo *repository.FeatureRepository, taskRepo *repository.TaskRepository, force bool) error {

	statusBreakdown, _ := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
	completedCount := statusBreakdown[models.TaskStatus("completed")]
	reviewedCount := statusBreakdown[models.TaskStatus("ready_for_review")]
	allDoneCount := completedCount + reviewedCount

	var incompleteTasks []*models.Task
	for _, task := range tasks {
		if task.Status != models.TaskStatus("completed") && task.Status != models.TaskStatus("ready_for_review") {
			incompleteTasks = append(incompleteTasks, task)
		}
	}
	hasIncomplete := len(incompleteTasks) > 0

	if hasIncomplete && !force {
		todoCount := statusBreakdown[models.TaskStatus("todo")]
		inProgressCount := statusBreakdown[models.TaskStatus("in_progress")]
		blockedCount := statusBreakdown[models.TaskStatus("blocked")]
		cli.Warning("Cannot complete feature with incomplete tasks")
		fmt.Printf("  Status breakdown: %d todo, %d in_progress, %d blocked, %d ready_for_review\n",
			todoCount, inProgressCount, blockedCount, reviewedCount)
		fmt.Println("\nAffected tasks:")
		maxTasks := 10
		if len(incompleteTasks) < maxTasks {
			maxTasks = len(incompleteTasks)
		}
		for i := 0; i < maxTasks; i++ {
			fmt.Printf("  - %s (%s)\n", incompleteTasks[i].Key, incompleteTasks[i].Status)
		}
		if len(incompleteTasks) > 10 {
			fmt.Printf("  ... and %d more\n", len(incompleteTasks)-10)
		}
		cli.Info("Use --force to complete all tasks regardless of status")
		if cli.GlobalConfig.JSON {
			taskKeys := make([]string, len(incompleteTasks))
			for i, t := range incompleteTasks {
				taskKeys[i] = t.Key
			}
			return cli.OutputJSON(map[string]interface{}{
				"feature_key": featureKey, "completed_count": allDoneCount,
				"total_count": len(tasks),
				"status_breakdown": map[string]int{
					"todo": todoCount, "in_progress": inProgressCount,
					"blocked": blockedCount, "ready_for_review": reviewedCount, "completed": completedCount,
				},
				"affected_tasks": taskKeys,
				"requires_force": true,
			})
		}
		os.Exit(3)
	}

	if force && hasIncomplete {
		dbPath, canBackup, err := cli.GetDatabasePathForBackup()
		if err != nil {
			cli.Error(fmt.Sprintf("Error: failed to get database path for backup: %v", err))
			os.Exit(2)
		}
		if canBackup {
			if _, err := backupDatabaseOnForceFeature(force, dbPath, "force complete feature"); err != nil {
				cli.Error(fmt.Sprintf("Error: %v", err))
				os.Exit(2)
			}
		}
	}

	agent := getAgentIdentifier("")
	numCompleted := 0
	affectedTaskKeys := make([]string, 0)
	for _, task := range tasks {
		if task.Status == models.TaskStatus("completed") {
			continue
		}
		if err := taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("completed"), &agent, nil, nil, nil, true); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to complete task %s: %v", task.Key, err))
			os.Exit(2)
		}
		numCompleted++
		affectedTaskKeys = append(affectedTaskKeys, task.Key)
	}

	featureSvc := cli.GetFeatureService()
	if err := featureSvc.RecalculateAndSetProgress(ctx, feature.ID); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to update feature progress: %v", err))
		os.Exit(2)
	}
	feature, _ = featureRepo.GetByKey(ctx, featureKey)

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"feature_key":     featureKey,
			"completed_count": len(tasks),
			"total_count":     len(tasks),
			"status_breakdown": map[string]int{
				"todo": statusBreakdown[models.TaskStatus("todo")], "in_progress": statusBreakdown[models.TaskStatus("in_progress")],
				"blocked": statusBreakdown[models.TaskStatus("blocked")], "ready_for_review": reviewedCount,
				"completed": completedCount,
			},
			"affected_tasks": affectedTaskKeys,
		})
	}

	statusMsg := ""
	if feature != nil && feature.Status == models.FeatureStatusCompleted {
		statusMsg = " (feature marked as completed)"
	}
	if hasIncomplete && force {
		todoCount := statusBreakdown[models.TaskStatus("todo")]
		inProgressCount := statusBreakdown[models.TaskStatus("in_progress")]
		blockedCount := statusBreakdown[models.TaskStatus("blocked")]
		cli.Success(fmt.Sprintf("Feature %s completed: Force-completed %d tasks (%d todo, %d in_progress, %d blocked, %d ready_for_review)%s",
			featureKey, numCompleted, todoCount, inProgressCount, blockedCount, reviewedCount, statusMsg))
	} else {
		cli.Success(fmt.Sprintf("Feature %s completed: %d/%d tasks completed%s", featureKey, len(tasks), len(tasks), statusMsg))
	}
	return nil
}

// getNextFeatureKey generates the next available feature key for an epic.
// Used by idea.go and other commands that need to generate feature keys.
func getNextFeatureKey(ctx context.Context, featureRepo *repository.FeatureRepository, epicID int64, epicKey ...string) (string, error) {
	features, err := featureRepo.ListByEpic(ctx, epicID)
	if err != nil {
		return "", fmt.Errorf("failed to list features: %w", err)
	}

	maxNum := 0
	extractedEpicKey := ""
	for _, feature := range features {
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

	finalEpicKey := extractedEpicKey
	if len(epicKey) > 0 && epicKey[0] != "" {
		finalEpicKey = epicKey[0]
	}

	if finalEpicKey == "" {
		return "", fmt.Errorf("unable to determine epic key - no existing features and no epic key provided")
	}

	return fmt.Sprintf("%s-F%02d", finalEpicKey, maxNum+1), nil
}

// resolveFeaturePath resolves the relative path to a feature file.
func resolveFeaturePath(ctx context.Context, feature *models.Feature) string {
	projectRoot, err := os.Getwd()
	if err != nil || projectRoot == "" {
		return ""
	}
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return ""
	}
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)
	pathResolver := pathresolver.NewPathResolver(epicRepo, featureRepo, taskRepo, projectRoot)
	absPath, err := pathResolver.ResolveFeaturePath(ctx, feature.Key)
	if err != nil {
		return ""
	}
	return getRelativePathFeature(absPath, projectRoot)
}

// resolveFeatureFilePath returns the absolute file path for a new feature.
func resolveFeatureFilePath(feature *models.Feature, epicKey, projectRoot string) string {
	if feature.FilePath != nil {
		return filepath.Join(projectRoot, *feature.FilePath)
	}
	return filepath.Join(projectRoot, fmt.Sprintf("docs/plan/%s/%s/feature.md", epicKey, feature.Key))
}

// performFeatureDelete handles the core delete logic for a feature.
func performFeatureDelete(ctx context.Context, featureKey string, force bool) error {
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return err
	}
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'shark feature list' to see available features")
		os.Exit(1)
	}

	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to check for tasks: %v", err))
		os.Exit(1)
	}

	if len(tasks) > 0 && !force {
		cli.Error(fmt.Sprintf("Error: Feature %s has %d task(s)", featureKey, len(tasks)))
		cli.Warning("This will CASCADE DELETE all tasks and their history")
		cli.Info(fmt.Sprintf("Use --force to confirm deletion: shark feature delete %s --force", featureKey))
		os.Exit(1)
	}

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
				os.Exit(2)
			}
			if !cli.GlobalConfig.JSON {
				cli.Info(fmt.Sprintf("Database backup created: %s", backupPath))
			}
		}
	}

	featureSvc := cli.GetFeatureService()
	if err := featureSvc.DeleteFeature(ctx, featureKey); err != nil {
		return err
	}

	cli.Success(fmt.Sprintf("Feature %s deleted successfully", featureKey))
	if len(tasks) > 0 {
		cli.Warning(fmt.Sprintf("Cascade deleted %d task(s) and their history", len(tasks)))
	}
	return nil
}

// performFeatureUpdate handles the core update logic for a feature.
func performFeatureUpdate(ctx context.Context, featureKey string, cmd *cobra.Command) error {
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return err
	}
	featureRepo := repository.NewFeatureRepository(repoDb)

	changed := false
	updates := services.FeatureUpdates{}

	if title, _ := cmd.Flags().GetString("title"); title != "" {
		updates.Title = &title
		changed = true
	}
	if description, _ := cmd.Flags().GetString("description"); description != "" {
		updates.Description = &description
		changed = true
	}

	statusFlag, _ := cmd.Flags().GetString("status")
	force, _ := cmd.Flags().GetBool("force")

	if statusFlag != "" {
		if err := applyFeatureStatusUpdate(ctx, featureRepo, featureKey, statusFlag, force, &updates, &changed); err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
	}

	if execOrder, _ := cmd.Flags().GetInt("execution-order"); execOrder != -1 {
		updates.ExecutionOrder = &execOrder
		changed = true
	}

	if changed {
		featureSvc := cli.GetFeatureService()
		if _, err := featureSvc.UpdateFeature(ctx, featureKey, updates); err != nil {
			return err
		}
	}

	// Handle key update
	if newKey, _ := cmd.Flags().GetString("key"); newKey != "" {
		if err := ValidateNoSpaces(newKey, "feature"); err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		if newKey != featureKey {
			existing, err := featureRepo.GetByKey(ctx, newKey)
			if err == nil && existing != nil {
				cli.Error(fmt.Sprintf("Error: Feature with key '%s' already exists", newKey))
				os.Exit(1)
			}
			if err := featureRepo.UpdateKey(ctx, featureKey, newKey); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to update feature key: %v", err))
				os.Exit(1)
			}
			changed = true
		}
	}

	// Handle file path update
	if customFile := getFileFlagValue(cmd); customFile != "" {
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

// applyFeatureStatusUpdate handles status-related logic for feature update.
func applyFeatureStatusUpdate(ctx context.Context, featureRepo *repository.FeatureRepository, featureKey, statusFlag string, force bool, updates *services.FeatureUpdates, changed *bool) error {
	if strings.ToLower(statusFlag) == "auto" {
		feature, err := featureRepo.GetByKey(ctx, featureKey)
		if err != nil {
			return fmt.Errorf("feature %s does not exist", featureKey)
		}
		if err := featureRepo.SetStatusOverride(ctx, feature.ID, false); err != nil {
			return fmt.Errorf("failed to clear status override: %w", err)
		}
		featureSvc := cli.GetFeatureService()
		if err := featureSvc.RecalculateAndSetProgressByKey(ctx, featureKey); err != nil {
			return fmt.Errorf("failed to recalculate status: %w", err)
		}
		cli.Success(fmt.Sprintf("Feature %s status recalculated from tasks", featureKey))
		os.Exit(0)
	}

	validatedStatus, err := ParseFeatureStatus(statusFlag)
	if err != nil {
		return err
	}
	s := models.FeatureStatus(validatedStatus)
	updates.Status = &s
	*changed = true

	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature %s does not exist", featureKey)
	}
	if err := featureRepo.SetStatusOverride(ctx, feature.ID, true); err != nil {
		return fmt.Errorf("failed to set status override: %w", err)
	}
	if force && s == models.FeatureStatusCompleted {
		if err := featureRepo.CascadeStatusToTasks(ctx, feature.ID, models.TaskStatus("completed")); err != nil {
			return fmt.Errorf("failed to cascade status to tasks: %w", err)
		}
	}
	return nil
}

// parseFeatureListFlags parses and validates the feature list command flags/args.
// Returns epicFilter, statusFilter, sortBy, showAll, or error.
func parseFeatureListFlags(cmd *cobra.Command, args []string) (epicFilter, statusFilter, sortBy string, showAll bool, err error) {
	positionalEpic, parseErr := ParseFeatureListArgs(args)
	if parseErr != nil {
		return "", "", "", false, parseErr
	}
	epicFilter, _ = cmd.Flags().GetString("epic")
	statusFilter, _ = cmd.Flags().GetString("status")
	sortBy, _ = cmd.Flags().GetString("sort-by")
	showAll, _ = cmd.Flags().GetBool("show-all")
	if positionalEpic != nil {
		epicFilter = *positionalEpic
	}
	if statusFilter != "" {
		validatedStatus, sErr := ParseFeatureStatus(statusFilter)
		if sErr != nil {
			return "", "", "", false, sErr
		}
		statusFilter = validatedStatus
	}
	if sortBy != "" && sortBy != "key" && sortBy != "progress" && sortBy != "status" {
		return "", "", "", false, fmt.Errorf("invalid sort-by '%s'. Must be one of: key, progress, status", sortBy)
	}
	return epicFilter, statusFilter, sortBy, showAll, nil
}

// buildFeatureGetJSON builds the JSON map for feature get output.
func buildFeatureGetJSON(feature *models.Feature, data *FeatureGetData, orchestratorAction interface{}) map[string]interface{} {
	validTransitions := GetValidTransitions(string(feature.Status), data.WorkflowCfg)
	statusSource := "calculated"
	if feature.StatusOverride {
		statusSource = "manual"
	}
	return map[string]interface{}{
		"id": feature.ID, "epic_id": feature.EpicID, "key": feature.Key,
		"title": feature.Title, "description": feature.Description,
		"status": feature.Status, "status_source": statusSource,
		"status_override": feature.StatusOverride, "progress_pct": feature.ProgressPct,
		"path": data.DirPath, "filename": data.Filename,
		"created_at": feature.CreatedAt, "updated_at": feature.UpdatedAt,
		"tasks": data.Tasks, "status_breakdown": data.StatusBreakdown,
		"related_documents": data.RelatedDocs, "progress": data.ProgressInfo,
		"work_summary": data.WorkSummary, "action_items": data.ActionItems,
		"notes": data.Notes, "context_data": data.ContextData,
		"orchestrator_action": orchestratorAction,
		"valid_transitions":   validTransitions,
	}
}

// performFeatureComplete handles the core logic for the feature complete command.
func performFeatureComplete(ctx context.Context, featureKey string, force bool) error {
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return err
	}

	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Feature %s does not exist", featureKey))
		cli.Info("Use 'shark feature list' to see available features")
		os.Exit(1)
	}

	tasks, err := taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to list tasks: %v", err))
		os.Exit(2)
	}

	if len(tasks) == 0 {
		return performFeatureCompleteEmpty(ctx, featureKey, feature, featureRepo)
	}

	return completeFeatureWithTasks(ctx, featureKey, feature, tasks, featureRepo, taskRepo, force)
}

// performFeatureCompleteEmpty handles completing a feature that has no tasks.
func performFeatureCompleteEmpty(ctx context.Context, featureKey string, feature *models.Feature, featureRepo *repository.FeatureRepository) error {
	feature.Status = models.FeatureStatusCompleted
	if err := featureRepo.Update(ctx, feature); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to update feature status: %v", err))
		os.Exit(2)
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"feature_key": featureKey, "completed_count": 0, "total_count": 0,
			"status_breakdown": map[string]int{}, "affected_tasks": []string{},
		})
	}
	cli.Success(fmt.Sprintf("Feature %s completed (no tasks)", featureKey))
	return nil
}
