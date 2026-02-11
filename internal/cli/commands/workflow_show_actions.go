package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// workflowShowActionsCmd displays all orchestrator actions in the workflow
var workflowShowActionsCmd = &cobra.Command{
	Use:   "show-actions",
	Short: "Display workflow orchestrator actions",
	Long: `Display all orchestrator actions defined in the workflow configuration.

Shows actions grouped by workflow phase with agent types and skills.
Provides a complete overview of which agents handle which statuses.
Displays all three entity levels (epic, feature, task) by default.

Flags:
  --status <status>      Show action for specific status only
  --action-type <type>   Filter by action type (spawn_agent, pause, wait_for_triage, archive)
  --level <level>        Filter by entity level (epic, feature, task)

Exit codes:
  0 - Success
  1 - Status or action type not found
  2 - Configuration error

Examples:
  shark workflow show-actions                        Show all levels
  shark workflow show-actions --level=epic           Show only epic actions
  shark workflow show-actions --level=task --json    Task actions in JSON (backward compatible)
  shark workflow show-actions --status=ready_for_development
  shark workflow show-actions --action-type=spawn_agent --json`,
	RunE: runWorkflowShowActions,
}

// Flags for show-actions
var (
	showActionsStatus     string
	showActionsActionType string
	showActionsLevel      string
)

func init() {
	workflowCmd.AddCommand(workflowShowActionsCmd)
	workflowShowActionsCmd.Flags().StringVar(&showActionsStatus, "status", "",
		"Filter to show action for specific status")
	workflowShowActionsCmd.Flags().StringVar(&showActionsActionType, "action-type", "",
		"Filter by action type (spawn_agent, pause, wait_for_triage, archive)")
	workflowShowActionsCmd.Flags().StringVar(&showActionsLevel, "level", "",
		"Filter by entity level (epic, feature, task)")
}

// WorkflowActionsDisplay is the output structure for show-actions command
type WorkflowActionsDisplay struct {
	WorkflowActions []StatusActionDisplay `json:"workflow_actions"`
	Summary         ActionsSummary        `json:"summary"`
}

// StatusActionDisplay represents a single status with its action
type StatusActionDisplay struct {
	Status             string                     `json:"status"`
	Phase              string                     `json:"phase"`
	Color              string                     `json:"color"`
	OrchestratorAction *config.OrchestratorAction `json:"orchestrator_action"`
}

// ActionsSummary contains summary statistics
type ActionsSummary struct {
	TotalStatuses       int            `json:"total_statuses"`
	StatusesWithActions int            `json:"statuses_with_actions"`
	ActionCounts        map[string]int `json:"action_counts"`
}

// PhaseOrder defines the display order of phases
var PhaseOrder = map[string]int{
	"planning":    1,
	"development": 2,
	"review":      3,
	"qa":          4,
	"approval":    5,
	"done":        6,
	"any":         7,
}

// MultiLevelActionsDisplay wraps per-level action displays for multi-level output
type MultiLevelActionsDisplay struct {
	EpicActions    *WorkflowActionsDisplay  `json:"epic_actions,omitempty"`
	FeatureActions *WorkflowActionsDisplay  `json:"feature_actions,omitempty"`
	TaskActions    *WorkflowActionsDisplay  `json:"task_actions,omitempty"`
	Summary        MultiLevelActionsSummary `json:"summary"`
}

// MultiLevelActionsSummary provides per-level totals and action counts
type MultiLevelActionsSummary struct {
	EpicTotal          int `json:"epic_total"`
	EpicWithActions    int `json:"epic_with_actions"`
	FeatureTotal       int `json:"feature_total"`
	FeatureWithActions int `json:"feature_with_actions"`
	TaskTotal          int `json:"task_total"`
	TaskWithActions    int `json:"task_with_actions"`
}

// runWorkflowShowActions implements the show-actions command
func runWorkflowShowActions(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Context available for future use

	// Validate --level flag
	if showActionsLevel != "" {
		validLevels := map[string]bool{"epic": true, "feature": true, "task": true}
		if !validLevels[showActionsLevel] {
			cli.Error(fmt.Sprintf("Invalid level '%s'. Valid levels: epic, feature, task", showActionsLevel))
			os.Exit(1)
		}
	}

	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Validate action type filter
	if showActionsActionType != "" {
		validTypes := map[string]bool{
			"spawn_agent":     true,
			"pause":           true,
			"wait_for_triage": true,
			"archive":         true,
		}
		if !validTypes[showActionsActionType] {
			cli.Error(fmt.Sprintf("Invalid action type '%s'. Valid types: spawn_agent, pause, wait_for_triage, archive", showActionsActionType))
			os.Exit(1)
		}
	}

	// Load multi-level workflow config
	multiWorkflow := config.LoadMultiLevelWorkflowOrDefault(configPath)

	// Build multi-level display
	display := &MultiLevelActionsDisplay{}

	if showActionsLevel == "" || showActionsLevel == "epic" {
		epicWf := multiWorkflow.GetWorkflowForLevel("epic")
		display.EpicActions = buildActionsDisplay(epicWf, showActionsStatus, showActionsActionType)
	}
	if showActionsLevel == "" || showActionsLevel == "feature" {
		featureWf := multiWorkflow.GetWorkflowForLevel("feature")
		display.FeatureActions = buildActionsDisplay(featureWf, showActionsStatus, showActionsActionType)
	}
	if showActionsLevel == "" || showActionsLevel == "task" {
		taskWf := multiWorkflow.GetWorkflowForLevel("task")
		display.TaskActions = buildActionsDisplay(taskWf, showActionsStatus, showActionsActionType)
	}

	display.Summary = buildMultiLevelSummary(display)

	// Check if status was requested but not found across all visible levels
	if showActionsStatus != "" {
		totalActions := 0
		if display.EpicActions != nil {
			totalActions += len(display.EpicActions.WorkflowActions)
		}
		if display.FeatureActions != nil {
			totalActions += len(display.FeatureActions.WorkflowActions)
		}
		if display.TaskActions != nil {
			totalActions += len(display.TaskActions.WorkflowActions)
		}
		if totalActions == 0 {
			cli.Error(fmt.Sprintf("Status '%s' not found in workflow configuration", showActionsStatus))
			os.Exit(1)
		}
	}

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(display)
	}

	// Human-readable output
	displayMultiLevelActionsHumanReadable(display, multiWorkflow)

	return nil
}

// buildActionsDisplay builds the display structure for actions
func buildActionsDisplay(workflow *config.WorkflowConfig, statusFilter string, actionTypeFilter string) *WorkflowActionsDisplay {
	display := &WorkflowActionsDisplay{
		WorkflowActions: make([]StatusActionDisplay, 0),
		Summary: ActionsSummary{
			ActionCounts: make(map[string]int),
		},
	}

	// Collect all statuses with actions
	var statusesWithActions []StatusActionDisplay

	// Iterate through all statuses
	for statusName, metadata := range workflow.StatusMetadata {
		// Skip if status filter is set and doesn't match
		if statusFilter != "" && statusName != statusFilter {
			continue
		}

		// Only include if action is defined
		if metadata.OrchestratorAction == nil {
			continue
		}

		// Apply action type filter if set
		if actionTypeFilter != "" && metadata.OrchestratorAction.Action != actionTypeFilter {
			continue
		}

		statusesWithActions = append(statusesWithActions, StatusActionDisplay{
			Status:             statusName,
			Phase:              metadata.Phase,
			Color:              metadata.Color,
			OrchestratorAction: metadata.OrchestratorAction,
		})

		// Count action types
		display.Summary.ActionCounts[metadata.OrchestratorAction.Action]++
	}

	display.WorkflowActions = statusesWithActions
	display.Summary.TotalStatuses = len(workflow.StatusMetadata)
	display.Summary.StatusesWithActions = len(statusesWithActions)

	return display
}

// buildMultiLevelSummary extracts per-level totals from the display
func buildMultiLevelSummary(display *MultiLevelActionsDisplay) MultiLevelActionsSummary {
	summary := MultiLevelActionsSummary{}
	if display.EpicActions != nil {
		summary.EpicTotal = display.EpicActions.Summary.TotalStatuses
		summary.EpicWithActions = display.EpicActions.Summary.StatusesWithActions
	}
	if display.FeatureActions != nil {
		summary.FeatureTotal = display.FeatureActions.Summary.TotalStatuses
		summary.FeatureWithActions = display.FeatureActions.Summary.StatusesWithActions
	}
	if display.TaskActions != nil {
		summary.TaskTotal = display.TaskActions.Summary.TotalStatuses
		summary.TaskWithActions = display.TaskActions.Summary.StatusesWithActions
	}
	return summary
}

// displayLevelSection renders a single level's action section with header and content
func displayLevelSection(levelName string, actions *WorkflowActionsDisplay, isDefault bool) {
	if actions == nil {
		return
	}
	fmt.Printf("--- %s Workflow Actions ---\n", cases.Title(language.English).String(levelName))
	if isDefault && len(actions.WorkflowActions) == 0 {
		fmt.Println("(using defaults, no actions configured)")
	} else if len(actions.WorkflowActions) == 0 {
		fmt.Printf("No orchestrator actions defined for %s workflow\n", levelName)
	} else {
		displayActionsForLevel(actions)
	}
	fmt.Println()
}

// displayMultiLevelActionsHumanReadable displays all levels with section headers
func displayMultiLevelActionsHumanReadable(display *MultiLevelActionsDisplay, multi *config.MultiLevelWorkflow) {
	fmt.Println("Workflow Orchestrator Actions")
	fmt.Println("================================================================")
	fmt.Println()

	displayLevelSection("epic", display.EpicActions, multi.Epic == nil)
	displayLevelSection("feature", display.FeatureActions, multi.Feature == nil)
	displayLevelSection("task", display.TaskActions, false)

	// Display cross-level summary
	fmt.Println("Summary:")
	if display.EpicActions != nil {
		label := ""
		if multi.Epic == nil {
			label = " (defaults)"
		}
		fmt.Printf("  Epic:    %d of %d statuses have actions%s\n",
			display.Summary.EpicWithActions, display.Summary.EpicTotal, label)
	}
	if display.FeatureActions != nil {
		label := ""
		if multi.Feature == nil {
			label = " (defaults)"
		}
		fmt.Printf("  Feature: %d of %d statuses have actions%s\n",
			display.Summary.FeatureWithActions, display.Summary.FeatureTotal, label)
	}
	if display.TaskActions != nil {
		fmt.Printf("  Task:    %d of %d statuses have actions\n",
			display.Summary.TaskWithActions, display.Summary.TaskTotal)
	}
}

// displayActionsForLevel renders a single level's actions grouped by phase
func displayActionsForLevel(display *WorkflowActionsDisplay) {
	grouped := groupByPhase(display.WorkflowActions)

	phaseOrder := []string{"planning", "development", "review", "qa", "approval", "done", "any"}
	for _, phase := range phaseOrder {
		actions, exists := grouped[phase]
		if !exists || len(actions) == 0 {
			continue
		}

		var phaseLabel string
		switch phase {
		case "any":
			phaseLabel = "Special States"
		default:
			phaseLabel = cases.Title(language.English).String(phase) + " Phase"
		}
		fmt.Printf("  %s:\n", phaseLabel)

		headers := []string{"Status", "Action", "Agent Type", "Skills"}
		rows := make([][]string, len(actions))
		for i, statusAction := range actions {
			action := statusAction.OrchestratorAction
			agentType := "-"
			if action.AgentType != "" {
				agentType = action.AgentType
			}
			skillsList := "-"
			if len(action.Skills) > 0 {
				skillsList = strings.Join(action.Skills, ", ")
				if len(skillsList) > 50 {
					skillsList = skillsList[:47] + "…"
				}
			}
			rows[i] = []string{
				statusAction.Status,
				action.Action,
				agentType,
				skillsList,
			}
		}
		cli.OutputTable(headers, rows)
		fmt.Println()
	}
}

// groupByPhase groups status actions by workflow phase
func groupByPhase(actions []StatusActionDisplay) map[string][]StatusActionDisplay {
	grouped := make(map[string][]StatusActionDisplay)

	for _, action := range actions {
		phase := action.Phase
		if phase == "" {
			phase = "any" // Default to "any" if no phase specified
		}
		grouped[phase] = append(grouped[phase], action)
	}

	// Sort each group by status name
	for _, actions := range grouped {
		sort.Slice(actions, func(i, j int) bool {
			return actions[i].Status < actions[j].Status
		})
	}

	return grouped
}
