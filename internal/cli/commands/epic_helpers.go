package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/fileops"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
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
	TaskCount  int    `json:"task_count"`
	IsPlanning bool   `json:"is_planning,omitempty"`
	Phase      string `json:"phase,omitempty"`
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

// EpicGetData holds all data needed to display an epic's details
type EpicGetData struct {
	ResolvedPath         string
	DirPath              string
	Filename             string
	EpicProgress         float64
	FeaturesWithDetails  []FeatureWithDetails
	RelatedDocs          []*models.Document
	FeatureRollup        map[string]int
	TaskRollup           map[string]int
	BlockedTasks         []*models.Task
	ApprovalBacklogCount int
	EpicNotes            []*models.EntityNote
	EpicContext          *models.ContextData
	WorkflowCfg          *config.WorkflowConfig
}

// getRelativePath converts an absolute path to relative path from project root
func getRelativePath(absPath string, projectRoot string) string {
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return absPath // Fall back to absolute path if conversion fails
	}
	return relPath
}

// sortEpics sorts epics by the specified field
func sortEpics(epics []EpicWithProgress, sortBy string) {
	switch sortBy {
	case "progress":
		sort.Slice(epics, func(i, j int) bool {
			return epics[i].ProgressPct < epics[j].ProgressPct
		})
	case "status":
		statusOrder := map[models.EpicStatus]int{
			models.EpicStatusDraft:     1,
			models.EpicStatusActive:    2,
			models.EpicStatusCompleted: 3,
			models.EpicStatusArchived:  4,
		}
		sort.Slice(epics, func(i, j int) bool {
			return statusOrder[epics[i].Status] < statusOrder[epics[j].Status]
		})
	default: // "", "key"
		sort.Slice(epics, func(i, j int) bool {
			return epics[i].Key < epics[j].Key
		})
	}
}

// renderEpicListTable renders epics as a table
func renderEpicListTable(epics []EpicWithProgress) {
	tableData := pterm.TableData{
		{"Key", "Title", "Status", "Progress", "Priority"},
	}

	for _, epic := range epics {
		title := epic.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		progress := fmt.Sprintf("%.0f%%", epic.ProgressPct)
		tableData = append(tableData, []string{
			epic.Key,
			title,
			string(epic.Status),
			progress,
			string(epic.Priority),
		})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// buildEpicPlanningBasicInfo assembles the key-value info table for epic planning mode display.
func buildEpicPlanningBasicInfo(info *services.EpicDisplayInfo) [][]string {
	epic := info.Epic

	basicInfo := [][]string{
		{"Title", epic.Title},
		{"Status", fmt.Sprintf("%s (workflow)", string(epic.Status))},
	}

	if info.Phase != "" {
		basicInfo = append(basicInfo, []string{"Phase", info.Phase})
	}

	if info.PhaseDescription != "" {
		basicInfo = append(basicInfo, []string{"Phase Description", info.PhaseDescription})
	}

	if epic.Priority != "" {
		basicInfo = append(basicInfo, []string{"Priority", string(epic.Priority)})
	}

	if info.ResolvedPath != "" {
		basicInfo = append(basicInfo, []string{"Path", info.ResolvedPath})
	}

	if epic.Description != nil && *epic.Description != "" {
		basicInfo = append(basicInfo, []string{"Description", *epic.Description})
	}

	if epic.BusinessValue != nil {
		basicInfo = append(basicInfo, []string{"Business Value", string(*epic.BusinessValue)})
	}
	// E07-F42 REQ-F-006: human display uses "<label> (<num>)" or omits the row entirely.
	if epic.Size != nil {
		basicInfo = append(basicInfo, []string{"Size", formatSize(epic.Size)})
	}

	return basicInfo
}

// renderEpicPlanningSpecific renders epic-specific sections for planning mode:
// workflow position and planning features table.
func renderEpicPlanningSpecific(info *services.EpicDisplayInfo) {
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

	// Planning mode features
	if len(info.Features) == 0 {
		pterm.Info.Println("No features yet (epic is still being refined)")
	} else {
		fmt.Println()
		pterm.DefaultSection.Printf("Features (%d total)", len(info.Features))
		fmt.Println()

		tableData := pterm.TableData{
			{"Key", "Title", "Status", "Tasks", "Phase"},
		}
		for _, fi := range info.Features {
			phase := fi.Phase
			if phase == "" {
				phase = "-"
			}
			tableData = append(tableData, []string{
				fi.Feature.Key,
				fi.Feature.Title,
				string(fi.Feature.Status),
				fmt.Sprintf("%d", fi.TaskCount),
				phase,
			})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	}
}

// renderEpicPlanningWithTags renders an epic in planning mode with optional tag display.
// tags==nil means tagSvc unavailable (no Tags line). Empty slice renders "Tags: (none)".
func renderEpicPlanningWithTags(info *services.EpicDisplayInfo, tags []string) {
	basicInfo := buildEpicPlanningBasicInfo(info)
	basicInfo = appendTagsToBasicInfo(basicInfo, tags)
	RenderEntity(EntityDisplayOptions{
		EntityType:         "epic",
		Key:                info.Epic.Key,
		Status:             string(info.Epic.Status),
		BasicInfo:          basicInfo,
		ValidTransitions:   info.ValidTransitions,
		OrchestratorAction: info.OrchestratorAction,
		RelatedDocs:        info.RelatedDocs,
		Notes:              info.Notes,
		ContextData:        info.ContextData,
		RenderSpecific: func() {
			renderEpicPlanningSpecific(info)
		},
	})
}

// buildEpicAggregationBasicInfo assembles the key-value info table for epic aggregation mode display.
func buildEpicAggregationBasicInfo(epic *models.Epic, progress float64, path, filename string) [][]string {
	info := [][]string{
		{"Title", epic.Title},
		{"Status", fmt.Sprintf("%s (calculated)", string(epic.Status))},
		{"Priority", string(epic.Priority)},
		{"Progress", fmt.Sprintf("%.0f%%", progress)},
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
	// E07-F42 REQ-F-006: human display uses "<label> (<num>)" or omits the row entirely.
	if epic.Size != nil {
		info = append(info, []string{"Size", formatSize(epic.Size)})
	}

	return info
}

// renderEpicAggregationSpecific renders epic-specific sections for aggregation mode:
// feature status rollup, task rollup, impediments, and features table.
func renderEpicAggregationSpecific(featureRollup map[string]int, taskRollup map[string]int, blockedTasks []*models.Task, approvalBacklogCount int, features []FeatureWithDetails) {
	if len(featureRollup) > 0 {
		pterm.DefaultSection.Println("Feature Status Rollup")
		fmt.Println()
		rollupInfo := [][]string{}
		for st, count := range featureRollup {
			rollupInfo = append(rollupInfo, []string{strings.ToTitle(st), fmt.Sprintf("%d", count)})
		}
		_ = pterm.DefaultTable.WithData(rollupInfo).Render()
		fmt.Println()
	}

	if len(taskRollup) > 0 {
		pterm.DefaultSection.Println("Task Rollup")
		fmt.Println()
		rollupInfo := [][]string{}
		for st, count := range taskRollup {
			rollupInfo = append(rollupInfo, []string{strings.ToTitle(st), fmt.Sprintf("%d", count)})
		}
		_ = pterm.DefaultTable.WithData(rollupInfo).Render()
		fmt.Println()
	}

	if len(blockedTasks) > 0 || approvalBacklogCount > 0 {
		pterm.DefaultSection.Println("Impediments & Risks")
		fmt.Println()

		if len(blockedTasks) > 0 {
			fmt.Printf("Blocked Tasks (%d):\n", len(blockedTasks))
			for _, task := range blockedTasks {
				reason := ""
				if task.BlockedReason != nil && *task.BlockedReason != "" {
					reason = fmt.Sprintf(" - %s", *task.BlockedReason)
				}
				age := ""
				if task.BlockedAt.Valid && !task.BlockedAt.Time.IsZero() {
					ageDuration := time.Since(task.BlockedAt.Time)
					if ageDuration.Hours() < 1 {
						age = fmt.Sprintf(" (<%d min old)", int(ageDuration.Minutes()))
					} else if ageDuration.Hours() < 24 {
						age = fmt.Sprintf(" (%.1f hours old)", ageDuration.Hours())
					} else {
						age = fmt.Sprintf(" (%.1f days old)", ageDuration.Hours()/24)
					}
				}
				fmt.Printf("  - %s: %s%s%s\n", task.Key, task.Title, age, reason)
			}
		}

		if approvalBacklogCount > 0 {
			fmt.Printf("Approval Backlog: %d task(s) waiting for review\n", approvalBacklogCount)
		}
		fmt.Println()
	}

	if len(features) == 0 {
		pterm.Info.Println("No features found for this epic")
		return
	}

	pterm.DefaultSection.Println("Features")
	fmt.Println()

	tableData := pterm.TableData{
		{"Key", "Title", "Status", "Progress", "Tasks"},
	}

	for _, feature := range features {
		title := feature.Title
		if len(title) > 35 {
			title = title[:32] + "..."
		}

		progress := fmt.Sprintf("%.0f%%", feature.ProgressPct)
		taskDisplay := fmt.Sprintf("%d", feature.TaskCount)
		if feature.IsPlanning {
			progress = "(planning)"
			taskDisplay = "-"
		}

		tableData = append(tableData, []string{
			feature.Key,
			title,
			string(feature.Status),
			progress,
			taskDisplay,
		})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// renderEpicDetails renders epic details using the unified RenderEntity pattern with callbacks.
// renderEpicDetailsWithTags renders epic aggregation details with optional tag display.
// tags==nil means tagSvc unavailable (no Tags line). Empty slice renders "Tags: (none)".
func renderEpicDetailsWithTags(epic *models.Epic, data *EpicGetData, orchestratorAction *config.PopulatedAction, tags []string) {
	validTransitions := GetValidTransitions(string(epic.Status), data.WorkflowCfg)

	basicInfo := buildEpicAggregationBasicInfo(epic, data.EpicProgress, data.DirPath, data.Filename)
	basicInfo = appendTagsToBasicInfo(basicInfo, tags)
	RenderEntity(EntityDisplayOptions{
		EntityType:         "epic",
		Key:                epic.Key,
		Status:             string(epic.Status),
		BasicInfo:          basicInfo,
		ValidTransitions:   validTransitions,
		OrchestratorAction: orchestratorAction,
		RelatedDocs:        data.RelatedDocs,
		Notes:              data.EpicNotes,
		ContextData:        data.EpicContext,
		RenderSpecific: func() {
			renderEpicAggregationSpecific(data.FeatureRollup, data.TaskRollup, data.BlockedTasks, data.ApprovalBacklogCount, data.FeaturesWithDetails)
		},
	})
}

// buildEpicsWithProgress attaches progress percentages to a list of epics.
func buildEpicsWithProgress(ctx context.Context, epics []*models.Epic) []EpicWithProgress {
	epicSvc := cli.GetEpicService()
	result := make([]EpicWithProgress, 0, len(epics))
	for _, epic := range epics {
		progress, err := epicSvc.CalculateProgress(ctx, epic.ID)
		if err != nil {
			if cli.GlobalConfig.Verbose {
				slog.Warn("Failed to calculate progress for epic", "epic", epic.Key, "error", err)
			}
			progress = 0.0
		}
		result = append(result, EpicWithProgress{Epic: epic, ProgressPct: progress})
	}
	return result
}

// buildEpicGetData retrieves all data needed to display an epic.
// Uses batch queries via EpicService.GetEpicDisplayData and parallelizes
// independent ancillary queries (docs, notes, context) to minimize Turso latency.
func buildEpicGetData(ctx context.Context, epic *models.Epic) (*EpicGetData, error) {
	epicSvc := cli.GetEpicService()
	displaySvc := cli.GetDisplayService()

	projectRoot, _ := cli.FindProjectRoot()

	// Single DB query via epic_display_data view fetches everything
	displayData, err := epicSvc.GetEpicDisplayData(ctx, epic, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic display data: %w", err)
	}

	// Workflow config (file read, no DB query)
	var workflowCfg *config.WorkflowConfig
	configPath, _ := cli.GetConfigPath()
	if configPath != "" {
		cfg, cfgErr := config.LoadWorkflowConfig(configPath)
		if cfgErr == nil {
			workflowCfg = cfg
		}
	}

	// Extract context from epic.ContextData (already on the epic row, no DB query)
	var epicContext *models.ContextData
	if epic.ContextData != nil && *epic.ContextData != "" {
		epicContext = &models.ContextData{}
		_ = json.Unmarshal([]byte(*epic.ContextData), epicContext)
	}

	// Map resolved path to dir/filename
	resolvedPath := displayData.RelPath
	var dirPath, filename string
	if resolvedPath != "" {
		dirPath = filepath.Dir(resolvedPath) + "/"
		filename = filepath.Base(resolvedPath)
	}

	// Build features with details from view data (0 additional queries)
	featuresWithDetails := make([]FeatureWithDetails, 0, len(displayData.Features))
	for _, feature := range displayData.Features {
		taskCount := displayData.FeatureTaskCounts[feature.ID]

		featureMode := displaySvc.DetermineFeatureDisplayMode(feature)
		isPlanning := featureMode == services.DisplayModePlanning
		featurePhase := ""
		if isPlanning {
			featurePhase = displaySvc.GetFeaturePhase(string(feature.Status))
		}

		featuresWithDetails = append(featuresWithDetails, FeatureWithDetails{
			Feature:    feature,
			TaskCount:  taskCount,
			IsPlanning: isPlanning,
			Phase:      featurePhase,
		})
	}

	// Approval backlog from task rollup
	approvalBacklogCount := 0
	if approvalCount, ok := displayData.TaskStatusRollup[string(models.TaskStatus("ready_for_review"))]; ok {
		approvalBacklogCount = approvalCount
	}

	return &EpicGetData{
		ResolvedPath:         resolvedPath,
		DirPath:              dirPath,
		Filename:             filename,
		EpicProgress:         displayData.Progress,
		FeaturesWithDetails:  featuresWithDetails,
		RelatedDocs:          displayData.RelatedDocs,
		FeatureRollup:        displayData.FeatureRollup,
		TaskRollup:           displayData.TaskStatusRollup,
		BlockedTasks:         displayData.BlockedTasks,
		ApprovalBacklogCount: approvalBacklogCount,
		EpicNotes:            displayData.Notes,
		EpicContext:          epicContext,
		WorkflowCfg:          workflowCfg,
	}, nil
}

// buildEpicGetJSON builds the JSON response for epic get
func buildEpicGetJSON(epic *models.Epic, data *EpicGetData, orchestratorAction interface{}) map[string]interface{} {
	// Build impediments list
	impediments := make([]map[string]interface{}, 0)
	for _, task := range data.BlockedTasks {
		blockReason := ""
		if task.BlockedReason != nil {
			blockReason = *task.BlockedReason
		}
		blockedSince := interface{}(nil)
		if task.BlockedAt.Valid {
			blockedSince = task.BlockedAt.Time
		}
		impediments = append(impediments, map[string]interface{}{
			"task_key":      task.Key,
			"title":         task.Title,
			"blocked_since": blockedSince,
			"reason":        blockReason,
		})
	}

	validTransitions := GetValidTransitions(string(epic.Status), data.WorkflowCfg)

	epicJSON := map[string]interface{}{
		"id":                     epic.ID,
		"key":                    epic.Key,
		"title":                  epic.Title,
		"description":            epic.Description,
		"status":                 epic.Status,
		"status_source":          "calculated",
		"priority":               epic.Priority,
		"business_value":         epic.BusinessValue,
		"slug":                   epic.Slug,
		"progress_pct":           data.EpicProgress,
		"path":                   data.DirPath,
		"filename":               data.Filename,
		"file_path":              epic.FilePath,
		"created_at":             epic.CreatedAt,
		"updated_at":             epic.UpdatedAt,
		"features":               data.FeaturesWithDetails,
		"related_documents":      data.RelatedDocs,
		"feature_status_rollup":  data.FeatureRollup,
		"task_status_rollup":     data.TaskRollup,
		"impediments":            impediments,
		"approval_backlog_count": data.ApprovalBacklogCount,
		"notes":                  data.EpicNotes,
		"context_data":           data.EpicContext,
		"orchestrator_action":    orchestratorAction,
		"valid_transitions":      validTransitions,
	}
	// E07-F42 REQ-F-006/007: size (numeric) and size_label (t-shirt label) in JSON output.
	if epic.Size != nil {
		epicJSON["size"] = *epic.Size
		if label, err := models.SizeLabel(*epic.Size); err == nil {
			epicJSON["size_label"] = label
		}
	}
	return epicJSON
}

// performEpicCreate handles the core logic of creating an epic
func performEpicCreate(ctx context.Context, epicTitle string, cmd *cobra.Command) error {
	// Try all three flag aliases: --file, --filename, --path (last one wins)
	file, _ := cmd.Flags().GetString("file")
	filename, _ := cmd.Flags().GetString("filename")
	path, _ := cmd.Flags().GetString("path")

	var customFile string
	if path != "" {
		customFile = path
	} else if filename != "" {
		customFile = filename
	} else if file != "" {
		customFile = file
	}

	force, _ := cmd.Flags().GetBool("force")

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		cli.Error(fmt.Sprintf("Failed to find project root: %s", err.Error()))
		os.Exit(1)
	}

	// Parse priority flag
	priorityStr, _ := cmd.Flags().GetString("priority")
	if priorityStr == "" {
		priorityStr = "medium"
	}
	priorityStr, err = ParseEpicPriority(priorityStr)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	// Parse business-value flag
	businessValueStr, _ := cmd.Flags().GetString("business-value")
	var businessValuePtr *string
	if businessValueStr != "" {
		businessValueStr, err = ParseEpicPriority(businessValueStr)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Invalid business-value: %v", err))
			os.Exit(1)
		}
		businessValuePtr = &businessValueStr
	}

	// Parse status flag
	statusStr, _ := cmd.Flags().GetString("status")
	if statusStr == "" {
		statusStr = "draft"
	}
	statusStr, err = ParseEpicStatus(statusStr)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	if epicCreateKey != "" {
		if err := ValidateNoSpaces(epicCreateKey, "epic"); err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
	}

	// Resolve custom file path for template rendering
	var customFilePath *string
	var actualFilePath string

	if customFile != "" {
		absPath, relPath, err := taskcreation.ValidateCustomFilename(customFile, projectRoot)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Invalid filename: %v", err))
			os.Exit(1)
		}
		customFilePath = &relPath
		actualFilePath = absPath
	}

	// E28-F04 REQ-F-012: read --tag via the flag accessor so repeated test
	// invocations see a fresh value each time.
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

	// Build CreateEpicInput and delegate key generation, collision checks, and DB creation to service
	input := services.CreateEpicInput{
		Title:         epicTitle,
		Description:   &epicCreateDescription,
		Status:        statusStr,
		Priority:      priorityStr,
		BusinessValue: businessValuePtr,
		CustomKey:     epicCreateKey,
		Force:         force,
		Tags:          tags,
		Size:          sizePtr,
	}
	if customFilePath != nil {
		input.FilePath = customFilePath
	}

	epicSvc := cli.GetEpicService()
	epic, err := epicSvc.CreateEpic(ctx, input)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "epic", epicTitle)
	}

	nextKey := epic.Key

	// Determine the actual file path for template rendering and output
	if actualFilePath == "" {
		slug := utils.GenerateSlug(epicTitle)
		epicSlug := fmt.Sprintf("%s-%s", nextKey, slug)
		epicDir := fmt.Sprintf("docs/plan/%s", epicSlug)

		if err := os.MkdirAll(epicDir, 0755); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to create epic directory: %v", err))
			os.Exit(1)
		}

		actualFilePath = fmt.Sprintf("%s/epic.md", epicDir)
	}

	// Read and render template (uses cached template dir from PersistentPreRunE)
	templateDir := templates.GetTemplateDirName()
	templatePath := filepath.Join(templateDir, "entity", "epic.md")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to read epic template: %v", err))
		cli.Info("Make sure you've run 'shark init' to create templates")
		os.Exit(1)
	}

	data := EpicTemplateData{
		EpicKey:     nextKey,
		EpicSlug:    nextKey,
		Title:       epicTitle,
		Description: epicCreateDescription,
		FilePath:    actualFilePath,
		Date:        time.Now().Format("2006-01-02"),
	}

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

	writer := fileops.NewEntityFileWriter()
	result, err := writer.WriteEntityFile(fileops.WriteOptions{
		Content:        buf.Bytes(),
		ProjectRoot:    projectRoot,
		FilePath:       actualFilePath,
		Verbose:        cli.GlobalConfig.Verbose,
		EntityType:     "epic",
		UseAtomicWrite: false,
		Logger: func(message string) {
			cli.Info(message)
		},
	})
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	fileWasLinked := result.Linked

	requiredSections := cli.GetRequiredSectionsForEntityType("epic")
	if cli.GlobalConfig.JSON {
		jsonOutput := cli.FormatEntityCreationJSON("epic", nextKey, epicTitle, actualFilePath, projectRoot, requiredSections)
		return cli.OutputJSON(jsonOutput)
	}

	message := cli.FormatEntityCreationMessage("epic", nextKey, epicTitle, actualFilePath, projectRoot, fileWasLinked, requiredSections)
	fmt.Print(message)
	return nil
}

// performEpicComplete handles the core logic of completing an epic.
// It delegates business logic to EpicService.CompleteEpic and handles CLI output.
func performEpicComplete(ctx context.Context, epicKey string, force bool) error {
	epicSvc := cli.GetEpicService()
	agentID := getAgentIdentifier("")

	// First call: check if force is required (no mutations yet when RequiresForce is returned)
	result, err := epicSvc.CompleteEpic(ctx, epicKey, force, agentID)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(2)
	}

	if result.FeatureCount == 0 {
		cli.Info(fmt.Sprintf("Epic %s has no features to complete", epicKey))
		return nil
	}

	if result.TotalCount == 0 {
		cli.Info(fmt.Sprintf("Epic %s has no tasks to complete", epicKey))
		return nil
	}

	// Service returned RequiresForce — display warning and incomplete details
	if result.RequiresForce {
		cli.Warning("Cannot complete epic with incomplete tasks")
		fmt.Println()
		fmt.Printf("Total tasks: %d\n", result.TotalCount)
		fmt.Print("Status breakdown: ")
		breakdownParts := []string{}
		for _, st := range []string{"todo", "in_progress", "blocked", "ready_for_review"} {
			if count, ok := result.StatusBreakdown[st]; ok && count > 0 {
				breakdownParts = append(breakdownParts, fmt.Sprintf("%d %s", count, st))
			}
		}
		fmt.Println(strings.Join(breakdownParts, ", "))
		fmt.Println()

		fmt.Println("Feature breakdown:")
		for featureKey, details := range result.IncompleteDetails {
			if details.IncompleteCount == 0 {
				fmt.Printf("  %s: %d tasks (all ready_for_review or completed)\n", featureKey, details.TotalTasks)
			} else {
				fmt.Printf("  %s: %d tasks (%d incomplete) ", featureKey, details.TotalTasks, details.IncompleteCount)
				parts := []string{}
				for _, st := range []string{"todo", "in_progress", "blocked"} {
					if count, ok := details.StatusBreakdown[st]; ok && count > 0 {
						parts = append(parts, fmt.Sprintf("%d %s", count, st))
					}
				}
				if len(parts) > 0 {
					fmt.Printf("(%s)", strings.Join(parts, ", "))
				}
				fmt.Println()
			}
		}
		fmt.Println()
		cli.Info("Use --force to complete all tasks regardless of status")
		os.Exit(3)
	}

	// If force was used to complete incomplete tasks, create backup first
	if result.ForceCompleted {
		dbPath, canBackup, backupErr := cli.GetDatabasePathForBackup()
		if backupErr != nil {
			cli.Error(fmt.Sprintf("Error: failed to get database path for backup: %v", backupErr))
			os.Exit(2)
		}
		if canBackup {
			backupPath, backupCreateErr := CreateBackupIfForce(force, dbPath, "force complete epic")
			if backupCreateErr != nil {
				cli.Error(fmt.Sprintf("Error: %v", backupCreateErr))
				cli.Info("Aborting operation to prevent data loss")
				os.Exit(2)
			}
			if backupPath != "" && !cli.GlobalConfig.JSON {
				cli.Info(fmt.Sprintf("Database backup created: %s", backupPath))
			}
		} else if cli.GlobalConfig.Verbose {
			cli.Info("Using cloud database - backup handled by provider")
		}
	}

	// Recalculate progress for all features (CLI concern — calls FeatureService)
	featureSvc := cli.GetFeatureService()
	features, listErr := epicSvc.GetFeatures(ctx, epicKey)
	if listErr == nil {
		for _, f := range features {
			if recalcErr := featureSvc.RecalculateAndSetProgress(ctx, f.ID); recalcErr != nil {
				if cli.GlobalConfig.Verbose {
					slog.Warn("Failed to update progress for feature", "feature", f.Key, "error", recalcErr)
				}
			}
		}
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	if result.ForceCompleted {
		todoCount := result.StatusBreakdown["todo"]
		inProgressCount := result.StatusBreakdown["in_progress"]
		blockedCount := result.StatusBreakdown["blocked"]
		readyCount := result.StatusBreakdown["ready_for_review"]
		breakdownStr := fmt.Sprintf("%d todo, %d in_progress, %d blocked, %d ready_for_review", todoCount, inProgressCount, blockedCount, readyCount)
		cli.Success(fmt.Sprintf("Epic %s completed: Force-completed %d tasks (%s)", epicKey, len(result.AffectedTasks), breakdownStr))
	} else {
		cli.Success(fmt.Sprintf("Epic %s completed: %d/%d tasks completed across %d feature(s)", epicKey, result.TotalCount, result.TotalCount, result.FeatureCount))
	}

	return nil
}

// performEpicDelete handles the core logic of deleting an epic
func performEpicDelete(ctx context.Context, epicKey string, force bool) error {
	epicSvc := cli.GetEpicService()

	epic, err := epicSvc.GetEpic(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	features, err := epicSvc.GetFeatures(ctx, epic.Key)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to check for features: %v", err))
		os.Exit(1)
	}

	if len(features) > 0 && !force {
		cli.Error(fmt.Sprintf("Error: Epic %s has %d feature(s)", epicKey, len(features)))
		cli.Warning("This will CASCADE DELETE all features and their tasks")
		cli.Info(fmt.Sprintf("Use --force to confirm deletion: shark epic delete %s --force", epicKey))
		os.Exit(1)
	}

	if len(features) > 0 {
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
		} else if cli.GlobalConfig.Verbose {
			cli.Info("Using cloud database - backup handled by provider")
		}
	}

	if err := epicSvc.DeleteEpic(ctx, epic.Key); err != nil {
		cli.Error(fmt.Sprintf("Error: Failed to delete epic: %v", err))
		os.Exit(1)
	}

	cli.Success(fmt.Sprintf("Epic %s deleted successfully", epicKey))
	if len(features) > 0 {
		cli.Warning(fmt.Sprintf("Cascade deleted %d feature(s) and their tasks", len(features)))
	}
	return nil
}

// performEpicUpdate handles the core logic of updating an epic
func performEpicUpdate(ctx context.Context, epicKey string, cmd *cobra.Command) error {
	epicSvc := cli.GetEpicService()

	_, err := epicSvc.GetEpic(ctx, epicKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Error: Epic %s does not exist", epicKey))
		cli.Info("Use 'shark epic list' to see available epics")
		os.Exit(1)
	}

	changed := false
	updates := services.EpicUpdates{}

	title, _ := cmd.Flags().GetString("title")
	if title != "" {
		updates.Title = &title
		changed = true
	}

	description, _ := cmd.Flags().GetString("description")
	if description != "" {
		updates.Description = &description
		changed = true
	}

	priority, _ := cmd.Flags().GetString("priority")
	if priority != "" {
		validatedPriority, err := ParseEpicPriority(priority)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		p := models.Priority(validatedPriority)
		updates.Priority = &p
		changed = true
	}

	businessValue, _ := cmd.Flags().GetString("business-value")
	if businessValue != "" {
		validatedBV, err := ParseEpicPriority(businessValue)
		if err != nil {
			cli.Error(fmt.Sprintf("Error: Invalid business-value: %v", err))
			os.Exit(1)
		}
		bv := models.Priority(validatedBV)
		updates.BusinessValue = &bv
		changed = true
	}

	// E28-F04 REQ-F-012 / REQ-F-010: `--tag` on update is ADDITIVE only.
	// Guard with `Changed` so only explicit --tag usage sets Tags (nil
	// otherwise). The service-layer hook skips when len(Tags)==0.
	if cmd.Flags().Changed("tag") {
		tags, _ := cmd.Flags().GetStringSlice("tag")
		updates.Tags = tags
		changed = true
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
		changed = true
	}

	if changed && (updates.Title != nil || updates.Description != nil || updates.Status != nil || updates.Priority != nil || updates.BusinessValue != nil || len(updates.Tags) > 0 || updates.Size != nil || updates.ClearSize) {
		if _, err := epicSvc.UpdateEpic(ctx, epicKey, updates); err != nil {
			return handleEntityServiceError(cmd, resolveTagService(nil), err, "epic", epicKey)
		}
	}

	newKey, _ := cmd.Flags().GetString("key")
	if newKey != "" {
		if err := ValidateNoSpaces(newKey, "epic"); err != nil {
			cli.Error(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

		if newKey != epicKey {
			existing, _ := epicSvc.GetEpic(ctx, newKey)
			if existing != nil {
				cli.Error(fmt.Sprintf("Error: Epic with key '%s' already exists", newKey))
				os.Exit(1)
			}

			if err := epicSvc.RenameKey(ctx, epicKey, newKey); err != nil {
				cli.Error(fmt.Sprintf("Error: Failed to update epic key: %v", err))
				os.Exit(1)
			}
			changed = true
		}
	}

	file, _ := cmd.Flags().GetString("file")
	filename, _ := cmd.Flags().GetString("filename")
	path, _ := cmd.Flags().GetString("path")

	var customFile string
	if path != "" {
		customFile = path
	} else if filename != "" {
		customFile = filename
	} else if file != "" {
		customFile = file
	}

	if customFile != "" {
		updates.FilePath = &customFile
		if _, err := epicSvc.UpdateEpic(ctx, epicKey, services.EpicUpdates{FilePath: &customFile}); err != nil {
			cli.Error(fmt.Sprintf("Error: Failed to update epic file path: %v", err))
			os.Exit(1)
		}
		changed = true
	}

	if !changed {
		cli.Warning("No changes specified. Use --help to see available flags.")
		return nil
	}

	cli.Success(fmt.Sprintf("Epic %s updated successfully", epicKey))
	return nil
}

// resolveEpicPlanningPath resolves the file path for an epic in planning mode.
// Returns the relative path or empty string if resolution fails.
func resolveEpicPlanningPath(ctx context.Context, epicKey string) string {
	projectRoot, _ := cli.FindProjectRoot()
	if projectRoot == "" {
		return ""
	}
	epicSvc := cli.GetEpicService()
	absPath, err := epicSvc.ResolveEpicPath(ctx, epicKey, projectRoot)
	if err != nil {
		return ""
	}
	return getRelativePath(absPath, projectRoot)
}
