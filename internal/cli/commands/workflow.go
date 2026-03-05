package commands

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/spf13/cobra"
)

// MultiLevelWorkflowDisplay holds the display data for all workflow levels.
type MultiLevelWorkflowDisplay struct {
	EpicWorkflow    *LevelWorkflowDisplay `json:"epic_workflow"`
	FeatureWorkflow *LevelWorkflowDisplay `json:"feature_workflow"`
	TaskWorkflow    *LevelWorkflowDisplay `json:"task_workflow"`
	BugWorkflow     *LevelWorkflowDisplay `json:"bug_workflow"`
	ChangeWorkflow  *LevelWorkflowDisplay `json:"change_workflow"`
	ConfigPath      string                `json:"config_path"`
}

// LevelWorkflowDisplay holds the display data for a single workflow level.
type LevelWorkflowDisplay struct {
	Level           string              `json:"level"`
	Source          string              `json:"source"` // "custom" or "default"
	Version         string              `json:"version"`
	Statuses        []StatusDisplay     `json:"statuses"`
	SpecialStatuses map[string][]string `json:"special_statuses"`
	StatusCount     int                 `json:"status_count"`
	TransitionCount int                 `json:"transition_count"`
}

// StatusDisplay holds the display data for a single status.
type StatusDisplay struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	Phase          string   `json:"phase,omitempty"`
	Color          string   `json:"color,omitempty"`
	IsPlanning     bool     `json:"is_planning,omitempty"`
	AggregatesFrom string   `json:"aggregates_from,omitempty"`
	Transitions    []string `json:"transitions"`
	AgentTypes     []string `json:"agent_types,omitempty"`
}

// workflowCmd represents the workflow command group
var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage workflow configuration",

	Long: `Workflow configuration operations including listing, validation, and migration.

The workflow system allows customizing task status transitions via .sharkconfig.json.

Examples:
  shark workflow list      Display configured workflow
  shark workflow validate  Validate workflow configuration`,
}

// workflowListCmd displays the configured workflow
var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "Display configured workflow",
	Long: `Display the configured status workflow from .sharkconfig.json.

Shows all statuses and their valid transitions, highlighting special statuses
(_start_ and _complete_). If no custom workflow is configured, displays the
default workflow.

Examples:
  shark workflow list         Display workflow (human-readable)
  shark workflow list --json  Display workflow (JSON format)`,
	RunE: runWorkflowList,
}

// workflowValidateCmd validates the workflow configuration
var workflowValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate workflow configuration",
	Long: `Validate the workflow configuration in .sharkconfig.json for correctness.

Checks all validation rules:
- Required special statuses (_start_, _complete_) are defined
- All status references in transitions are defined
- All statuses are reachable from _start_ statuses
- All statuses have a path to _complete_ statuses
- No circular references with no terminal path

Exit codes:
  0 - Configuration is valid
  2 - Configuration is invalid (specific errors displayed)

Examples:
  shark workflow validate         Validate configuration
  shark workflow validate --json  Validate with JSON output`,
	RunE: runWorkflowValidate,
}

func init() {
	adminCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowValidateCmd)
}

// runWorkflowList implements the workflow list command
func runWorkflowList(cmd *cobra.Command, args []string) error {
	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load multi-level workflow (nil fields = using default)
	multi, err := config.LoadMultiLevelWorkflow(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// Build display structs
	display := buildMultiLevelWorkflowDisplay(multi, configPath)

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(display)
	}

	// Human-readable output
	return displayMultiLevelWorkflowHumanReadable(display)
}

// buildMultiLevelWorkflowDisplay builds display structs for all workflow levels.
func buildMultiLevelWorkflowDisplay(multi *config.MultiLevelWorkflow, configPath string) *MultiLevelWorkflowDisplay {
	return &MultiLevelWorkflowDisplay{
		EpicWorkflow:    buildLevelWorkflowDisplay("epic", multi.Epic, multi.GetWorkflowForLevel("epic")),
		FeatureWorkflow: buildLevelWorkflowDisplay("feature", multi.Feature, multi.GetWorkflowForLevel("feature")),
		TaskWorkflow:    buildLevelWorkflowDisplay("task", multi.Task, multi.GetWorkflowForLevel("task")),
		BugWorkflow:     buildLevelWorkflowDisplay("bug", multi.Bug, multi.GetWorkflowForLevel("bug")),
		ChangeWorkflow:  buildLevelWorkflowDisplay("change", multi.Change, multi.GetWorkflowForLevel("change")),
		ConfigPath:      configPath,
	}
}

// buildLevelWorkflowDisplay builds the display struct for a single workflow level.
// raw is the custom config (nil if using default), resolved is the effective config (never nil).
func buildLevelWorkflowDisplay(level string, raw *config.WorkflowConfig, resolved *config.WorkflowConfig) *LevelWorkflowDisplay {
	source := "default"
	if raw != nil {
		source = "custom"
	}

	// Count transitions
	transitionCount := 0
	for _, transitions := range resolved.StatusFlow {
		transitionCount += len(transitions)
	}

	// Build sorted status list
	statusNames := make([]string, 0, len(resolved.StatusFlow))
	for s := range resolved.StatusFlow {
		statusNames = append(statusNames, s)
	}
	sort.Strings(statusNames)

	statuses := make([]StatusDisplay, 0, len(statusNames))
	for _, name := range statusNames {
		sd := StatusDisplay{
			Name:        name,
			Transitions: resolved.StatusFlow[name],
		}
		if sd.Transitions == nil {
			sd.Transitions = []string{}
		}
		if meta, ok := resolved.StatusMetadata[name]; ok {
			sd.Description = meta.Description
			sd.Phase = meta.Phase
			sd.Color = meta.Color
			sd.IsPlanning = meta.IsPlanning
			sd.AggregatesFrom = meta.AggregatesFrom
			if len(meta.AgentTypes) > 0 {
				sd.AgentTypes = meta.AgentTypes
			}
		}
		statuses = append(statuses, sd)
	}

	version := resolved.Version
	if version == "" {
		version = config.DefaultWorkflowVersion
	}

	return &LevelWorkflowDisplay{
		Level:           level,
		Source:          source,
		Version:         version,
		Statuses:        statuses,
		SpecialStatuses: resolved.SpecialStatuses,
		StatusCount:     len(resolved.StatusFlow),
		TransitionCount: transitionCount,
	}
}

// displayMultiLevelWorkflowHumanReadable renders all three workflow levels in human-readable format.
func displayMultiLevelWorkflowHumanReadable(display *MultiLevelWorkflowDisplay) error {
	fmt.Println("Workflow Configuration")
	fmt.Println("================================================================")

	displayWorkflowLevelSection(display.EpicWorkflow)
	displayWorkflowLevelSection(display.FeatureWorkflow)
	displayWorkflowLevelSection(display.TaskWorkflow)
	displayWorkflowLevelSection(display.BugWorkflow)
	displayWorkflowLevelSection(display.ChangeWorkflow)

	// Legend
	fmt.Println("Legend:")
	fmt.Println("  [status] = aggregation threshold (progress derived from children)")
	fmt.Println("  [planning] = entity has its own workflow status (not aggregating)")
	fmt.Println("  [aggregates: X] = status aggregates progress from children of type X")
	fmt.Println()

	return nil
}

// displayWorkflowLevelSection renders a single workflow level section.
func displayWorkflowLevelSection(level *LevelWorkflowDisplay) {
	titleCase := cases.Title(language.English)
	fmt.Printf("\n--- %s Workflow (%s) ---\n", titleCase.String(level.Level), level.Source)
	fmt.Printf("  Version: %s\n\n", level.Version)

	// Special statuses
	displaySpecialStatuses(level.SpecialStatuses)

	// Status transitions
	fmt.Println("  Status Transitions:")
	for _, status := range level.Statuses {
		displayStatusWithMarkers(status)
	}
}

// displaySpecialStatuses renders the special statuses section.
func displaySpecialStatuses(specials map[string][]string) {
	if len(specials) == 0 {
		return
	}
	fmt.Println("  Special Statuses:")
	// Show in a consistent order
	keys := []struct {
		key   string
		label string
	}{
		{config.StartStatusKey, "entry points"},
		{config.CompleteStatusKey, "exit points"},
		{config.AggregationStatusKey, "threshold"},
	}
	for _, k := range keys {
		if vals, ok := specials[k.key]; ok && len(vals) > 0 {
			fmt.Printf("    %s (%s):  %s\n", k.key, k.label, strings.Join(vals, ", "))
		}
	}
	fmt.Println()
}

// displayStatusWithMarkers renders a single status with planning/aggregation markers.
func displayStatusWithMarkers(status StatusDisplay) {
	// Build status name display: use [brackets] for aggregation statuses
	nameDisplay := status.Name
	if status.AggregatesFrom != "" {
		nameDisplay = fmt.Sprintf("[%s]", status.Name)
	}

	// Build description part
	descPart := ""
	if status.Description != "" {
		descPart = fmt.Sprintf(" (%s)", status.Description)
	}

	// Build marker suffix
	var markers []string
	if status.IsPlanning {
		markers = append(markers, "[planning]")
	}
	if status.AggregatesFrom != "" {
		markers = append(markers, fmt.Sprintf("[aggregates: %s]", status.AggregatesFrom))
	}
	markerSuffix := ""
	if len(markers) > 0 {
		markerSuffix = "  " + strings.Join(markers, " ")
	}

	fmt.Printf("    %s%s%s\n", nameDisplay, descPart, markerSuffix)

	// Transitions
	if len(status.Transitions) == 0 {
		fmt.Printf("      -> (terminal - no transitions)\n")
	} else {
		for _, t := range status.Transitions {
			fmt.Printf("      -> %s\n", t)
		}
	}

	// Metadata line
	var metaInfo []string
	if status.Phase != "" {
		metaInfo = append(metaInfo, fmt.Sprintf("phase: %s", status.Phase))
	}
	if len(status.AgentTypes) > 0 {
		metaInfo = append(metaInfo, fmt.Sprintf("agents: %s", strings.Join(status.AgentTypes, ", ")))
	}
	if status.Color != "" {
		metaInfo = append(metaInfo, fmt.Sprintf("color: %s", status.Color))
	}
	if len(metaInfo) > 0 {
		fmt.Printf("      [%s]\n", strings.Join(metaInfo, " | "))
	}
	fmt.Println()
}

// levelValidationResult holds validation results for a single workflow level.
type levelValidationResult struct {
	Level       string `json:"level"`
	Valid       bool   `json:"valid"`
	Source      string `json:"source"` // "custom" or "default"
	Statuses    int    `json:"statuses"`
	Transitions int    `json:"transitions"`
	Error       string `json:"error,omitempty"`
}

// runWorkflowValidate implements the workflow validate command.
// Validates all three workflow levels (epic, feature, task).
func runWorkflowValidate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Context available for future use

	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load multi-level workflow (nil fields = using default)
	multi, err := config.LoadMultiLevelWorkflow(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// Validate each level
	levels := []struct {
		name     string
		raw      *config.WorkflowConfig // nil means default
		resolved *config.WorkflowConfig // always non-nil (with default fallback)
	}{
		{"epic", multi.Epic, multi.GetWorkflowForLevel("epic")},
		{"feature", multi.Feature, multi.GetWorkflowForLevel("feature")},
		{"task", multi.Task, multi.GetWorkflowForLevel("task")},
		{"bug", multi.Bug, multi.GetWorkflowForLevel("bug")},
		{"change", multi.Change, multi.GetWorkflowForLevel("change")},
	}

	allValid := true
	var results []levelValidationResult

	for _, lvl := range levels {
		source := "default"
		if lvl.raw != nil {
			source = "custom"
		}

		validationErr := config.ValidateWorkflow(lvl.resolved)

		transitionCount := 0
		for _, transitions := range lvl.resolved.StatusFlow {
			transitionCount += len(transitions)
		}

		lr := levelValidationResult{
			Level:       lvl.name,
			Valid:       validationErr == nil,
			Source:      source,
			Statuses:    len(lvl.resolved.StatusFlow),
			Transitions: transitionCount,
		}

		if validationErr != nil {
			lr.Error = validationErr.Error()
			allValid = false
		}

		results = append(results, lr)
	}

	// Prepare combined result
	result := map[string]interface{}{
		"valid":       allValid,
		"config_path": configPath,
		"levels":      results,
	}

	// JSON output
	if cli.GlobalConfig.JSON {
		if !allValid {
			_ = cli.OutputJSON(result)
			return fmt.Errorf("validation failed")
		}
		return cli.OutputJSON(result)
	}

	// Human-readable output
	fmt.Println()
	for _, lr := range results {
		titleCase := cases.Title(language.English)
		if lr.Valid {
			fmt.Printf("  %s workflow: valid (%d statuses, %s)\n", titleCase.String(lr.Level), lr.Statuses, lr.Source)
		} else {
			fmt.Printf("  %s workflow: INVALID (%s)\n", titleCase.String(lr.Level), lr.Source)
			fmt.Printf("    Error: %s\n", lr.Error)
		}
	}
	fmt.Println()

	if allValid {
		cli.Success("All workflow levels are valid")
		return nil
	}

	cli.Error("One or more workflow levels have validation errors")
	return fmt.Errorf("workflow configuration is invalid")
}
