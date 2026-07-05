package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/spf13/cobra"
)

// MultiLevelWorkflowDisplay holds the display data for all workflow levels.
//
// Levels is ordered by config.KnownWorkflowLevels so adding a new entity
// workflow there automatically flows through to `shark admin workflow list`
// without further changes to this command.
type MultiLevelWorkflowDisplay struct {
	Levels     []*LevelWorkflowDisplay `json:"levels"`
	ConfigPath string                  `json:"config_path"`
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

	Long: `Workflow configuration operations including listing, validation, and inspection.

The workflow system allows customizing entity status transitions via .sharkconfig.json.

Examples:
  shark admin workflow list              Display compact workflow flows
  shark admin workflow list task --all   Display expanded task workflow details
  shark admin workflow validate          Validate workflow configuration
  shark admin workflow show-actions      Show all orchestrator actions
  shark admin workflow validate-actions  Validate orchestrator actions
  shark admin workflow show-action E07-F01-001 ready_for_development`,
}

// workflowListCmd displays the configured workflow
var workflowListCmd = &cobra.Command{
	Use:   "list [entity-type]",
	Short: "Display configured workflow",
	Long: `Display the configured status workflow from .sharkconfig.json.

By default, this command renders compact ASCII status-flow lines. Pass an
entity type to show one workflow level. Use --all to show expanded metadata,
including special statuses, descriptions, phases, colors, and agent types.

If no custom workflow is configured, this command displays the default workflow.

Examples:
  shark admin workflow list               Display compact workflow flows
  shark admin workflow list task          Display only the task workflow
  shark admin workflow list change --all  Display expanded change workflow details
  shark admin workflow list --json        Display workflow as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWorkflowList,
}

var workflowListAll bool

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
  shark admin workflow validate         Validate configuration
  shark admin workflow validate --json  Validate with JSON output`,
	RunE: runWorkflowValidate,
}

func init() {
	adminCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowValidateCmd)
	workflowListCmd.Flags().BoolVar(&workflowListAll, "all", false, "Render expanded workflow details")
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

	levelFilter := ""
	if len(args) > 0 {
		levelFilter, err = normalizeWorkflowListLevel(args[0])
		if err != nil {
			return err
		}
	}

	// Build display structs
	display := buildMultiLevelWorkflowDisplay(multi, configPath, levelFilter)

	// Output as JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(display)
	}

	if workflowListAll {
		return displayMultiLevelWorkflowHumanReadable(display)
	}

	return displayMultiLevelWorkflowSimple(display)
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

	// Validate each level. The level set is driven by config.KnownWorkflowLevels
	// so adding a new entity workflow there flows through automatically.
	type levelEntry struct {
		name     string
		raw      *config.WorkflowConfig // nil means default
		resolved *config.WorkflowConfig // always non-nil (with default fallback)
	}
	levels := make([]levelEntry, 0, len(config.KnownWorkflowLevels))
	for _, lvl := range config.KnownWorkflowLevels {
		levels = append(levels, levelEntry{
			name:     lvl,
			raw:      multi.RawForLevel(lvl),
			resolved: multi.GetWorkflowForLevel(lvl),
		})
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
		if lr.Valid {
			fmt.Printf("  %s workflow: valid (%d statuses, %s)\n", levelDisplayLabel(lr.Level), lr.Statuses, lr.Source)
		} else {
			fmt.Printf("  %s workflow: INVALID (%s)\n", levelDisplayLabel(lr.Level), lr.Source)
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
