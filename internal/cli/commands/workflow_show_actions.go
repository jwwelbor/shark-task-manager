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

Flags:
  --status <status>      Show action for specific status only
  --action-type <type>   Filter by action type (spawn_agent, pause, wait_for_triage, archive)

Exit codes:
  0 - Success
  1 - Status or action type not found
  2 - Configuration error

Examples:
  shark workflow show-actions                        Show all actions
  shark workflow show-actions --json                 JSON format
  shark workflow show-actions --status=ready_for_development
  shark workflow show-actions --action-type=spawn_agent --json`,
	RunE: runWorkflowShowActions,
}

// Flags for show-actions
var (
	showActionsStatus     string
	showActionsActionType string
)

func init() {
	workflowCmd.AddCommand(workflowShowActionsCmd)
	workflowShowActionsCmd.Flags().StringVar(&showActionsStatus, "status", "",
		"Filter to show action for specific status")
	workflowShowActionsCmd.Flags().StringVar(&showActionsActionType, "action-type", "",
		"Filter by action type (spawn_agent, pause, wait_for_triage, archive)")
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

// runWorkflowShowActions implements the show-actions command
func runWorkflowShowActions(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = ctx // Context available for future use

	// Get config path using centralized helper
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load workflow config
	workflow, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow config: %w", err)
	}

	// If no custom workflow, use default
	if workflow == nil {
		workflow = config.DefaultWorkflow()
		if !cli.GlobalConfig.JSON {
			cli.Warning("No custom workflow configured in .sharkconfig.json, showing default workflow actions")
		}
	}

	// Validate filters
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

	// Build display data
	display := buildActionsDisplay(workflow, showActionsStatus, showActionsActionType)

	// Check if status was requested but not found
	if showActionsStatus != "" && len(display.WorkflowActions) == 0 {
		cli.Error(fmt.Sprintf("Status '%s' not found in workflow configuration", showActionsStatus))
		os.Exit(1)
	}

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(display)
	}

	// Human-readable output
	displayActionsHumanReadable(display, workflow)

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

// displayActionsHumanReadable displays actions in human-readable grouped format
func displayActionsHumanReadable(display *WorkflowActionsDisplay, workflow *config.WorkflowConfig) {
	if len(display.WorkflowActions) == 0 {
		cli.Warning("No orchestrator actions defined in workflow configuration")
		return
	}

	fmt.Println("Workflow Orchestrator Actions")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Group by phase
	grouped := groupByPhase(display.WorkflowActions)

	// Display in phase order
	phaseOrder := []string{"planning", "development", "review", "qa", "approval", "done", "any"}
	for _, phase := range phaseOrder {
		actions, exists := grouped[phase]
		if !exists || len(actions) == 0 {
			continue
		}

		// Display phase header
		var phaseLabel string
		switch phase {
		case "any":
			phaseLabel = "Special States"
		default:
			phaseLabel = cases.Title(language.English).String(phase) + " Phase"
		}
		fmt.Printf("%s:\n", phaseLabel)

		// Prepare table data
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

		// Output table
		cli.OutputTable(headers, rows)
		fmt.Println()
	}

	// Display summary
	fmt.Println("Summary:")
	fmt.Printf("  Total Statuses: %d\n", display.Summary.TotalStatuses)
	fmt.Printf("  With Actions: %d\n", display.Summary.StatusesWithActions)

	// Display action counts
	actionTypes := []string{"spawn_agent", "pause", "wait_for_triage", "archive"}
	for _, actionType := range actionTypes {
		count := display.Summary.ActionCounts[actionType]
		fmt.Printf("  %s: %d\n", actionType, count)
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
