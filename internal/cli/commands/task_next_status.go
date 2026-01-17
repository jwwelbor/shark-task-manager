package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/spf13/cobra"
)

// taskNextStatusCmd progresses a task to its next workflow status
var taskNextStatusCmd = &cobra.Command{
	Use:   "next-status <task-key>",
	Short: "Progress task to next workflow status",
	Long: `Progress a task through the workflow by selecting from available transitions.

When a task has multiple valid next statuses, this command shows them interactively
and lets you choose. For automation/scripting, use --status to specify the target directly.

Flags:
  --status=<name>   Transition directly to this status (non-interactive)
  --preview         Show available transitions without making changes
  --force           Bypass workflow validation (administrative override)

Examples:
  shark task next-status E07-F16-001              Interactive selection
  shark task next-status E07-F16-001 --preview    Show available transitions
  shark task next-status E07-F16-001 --status=ready_for_code_review  Direct transition
  shark task next-status E07-F16-001 --json       JSON output (for scripting)`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskNextStatus,
}

func init() {
	taskNextStatusCmd.Flags().String("status", "", "Target status for direct transition (non-interactive)")
	taskNextStatusCmd.Flags().Bool("preview", false, "Show available transitions without making changes")
	taskNextStatusCmd.Flags().Bool("force", false, "Bypass workflow validation")
}

// TransitionChoice represents a valid status transition for display
type TransitionChoice struct {
	Number      int      `json:"number"`
	Status      string   `json:"status"`
	Description string   `json:"description,omitempty"`
	Phase       string   `json:"phase,omitempty"`
	AgentTypes  []string `json:"agent_types,omitempty"`
	Color       string   `json:"color,omitempty"`
}

// NextStatusResult contains the result of a next-status operation
type NextStatusResult struct {
	TaskKey              string             `json:"task_key"`
	CurrentStatus        string             `json:"current_status"`
	CurrentPhase         string             `json:"current_phase,omitempty"`
	AvailableTransitions []TransitionChoice `json:"available_transitions"`
	NewStatus            string             `json:"new_status,omitempty"`
	Transitioned         bool               `json:"transitioned"`
	Message              string             `json:"message,omitempty"`
}

func runTaskNextStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	// Get flags
	targetStatus, _ := cmd.Flags().GetString("status")
	preview, _ := cmd.Flags().GetBool("preview")
	force, _ := cmd.Flags().GetBool("force")

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	// Get project root for workflow service
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Load workflow config for repository
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	workflowConfig := config.GetWorkflowOrDefault(configPath)

	// Get task - use workflow-aware repository
	taskRepo := repository.NewTaskRepositoryWithWorkflow(repoDb, workflowConfig)
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task not found")
	}

	// Create workflow service
	workflowSvc := workflow.NewService(projectRoot)
	currentStatus := string(task.Status)

	// Get available transitions
	transitions := workflowSvc.GetTransitionInfo(currentStatus)
	currentMeta := workflowSvc.GetStatusMetadata(currentStatus)

	// Build result
	result := NextStatusResult{
		TaskKey:       task.Key,
		CurrentStatus: currentStatus,
		CurrentPhase:  currentMeta.Phase,
	}

	// Build transition choices
	for i, t := range transitions {
		result.AvailableTransitions = append(result.AvailableTransitions, TransitionChoice{
			Number:      i + 1,
			Status:      t.TargetStatus,
			Description: t.Description,
			Phase:       t.Phase,
			AgentTypes:  t.AgentTypes,
			Color:       t.Color,
		})
	}

	// Handle terminal status
	if len(transitions) == 0 {
		if workflowSvc.IsTerminalStatus(currentStatus) {
			result.Message = fmt.Sprintf("Task is in terminal status '%s' - no transitions available", currentStatus)
		} else {
			result.Message = fmt.Sprintf("No valid transitions from status '%s'", currentStatus)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Preview mode - just show transitions
	if preview {
		result.Message = "Preview mode - no changes made"

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		fmt.Printf("\nTask: %s\n", task.Key)
		fmt.Printf("Current status: %s", currentStatus)
		if currentMeta.Phase != "" {
			fmt.Printf(" (phase: %s)", currentMeta.Phase)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("Available transitions:")
		printTransitions(result.AvailableTransitions)
		fmt.Println()
		fmt.Println("Use 'shark task next-status " + task.Key + "' to transition")
		return nil
	}

	// Direct transition mode (--status flag)
	if targetStatus != "" {
		// Validate target status
		targetStatus = workflowSvc.NormalizeStatus(targetStatus)
		valid := false
		for _, t := range transitions {
			if strings.EqualFold(t.TargetStatus, targetStatus) {
				valid = true
				targetStatus = t.TargetStatus // Use canonical case
				break
			}
		}

		if !valid && !force {
			cli.Error(fmt.Sprintf("Invalid transition: '%s' -> '%s'", currentStatus, targetStatus))
			fmt.Println()
			fmt.Println("Valid transitions from current status:")
			for _, t := range transitions {
				fmt.Printf("  - %s\n", t.TargetStatus)
			}
			fmt.Println()
			fmt.Println("Use --force to bypass workflow validation")
			return fmt.Errorf("invalid transition")
		}

		// Perform transition
		return performTransition(ctx, taskRepo, repoDb, task, targetStatus, force, &result)
	}

	// Load config to check interactive mode setting
	cfgManager := config.NewManager(projectRoot)
	cfg, err := cfgManager.Load()
	if err != nil {
		// If config fails to load, default to non-interactive
		cfg = &config.Config{}
	}

	// Check if interactive mode is enabled
	interactiveMode := cfg.IsInteractiveModeEnabled()

	// Non-interactive mode: auto-select first transition
	if !interactiveMode {
		if cli.GlobalConfig.JSON {
			// JSON mode - return available transitions
			result.Message = "Use --status=<name> to specify target status for JSON output"
			return cli.OutputJSON(result)
		}

		// Auto-select first transition
		targetStatus = transitions[0].TargetStatus
		cli.Info(fmt.Sprintf("Auto-selected next status: %s (from %d options)", targetStatus, len(transitions)))
		return performTransition(ctx, taskRepo, repoDb, task, targetStatus, force, &result)
	}

	// Interactive mode
	if cli.GlobalConfig.JSON {
		// JSON mode but no target status - return available transitions
		result.Message = "Use --status=<name> to specify target status for JSON output"
		return cli.OutputJSON(result)
	}

	// Show interactive prompt
	fmt.Printf("\nTask: %s\n", task.Key)
	fmt.Printf("Title: %s\n", task.Title)
	fmt.Printf("Current status: %s", currentStatus)
	if currentMeta.Phase != "" {
		fmt.Printf(" (phase: %s)", currentMeta.Phase)
	}
	if currentMeta.Description != "" {
		fmt.Printf("\n  %s", currentMeta.Description)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("Available transitions:")
	printTransitions(result.AvailableTransitions)
	fmt.Println()

	// Get user selection
	selection, err := promptForSelection(len(transitions))
	if err != nil {
		cli.Info("Cancelled - no changes made")
		return nil
	}

	targetStatus = transitions[selection-1].TargetStatus
	return performTransition(ctx, taskRepo, repoDb, task, targetStatus, force, &result)
}

// printTransitions prints available transitions in a formatted list
func printTransitions(transitions []TransitionChoice) {
	for _, t := range transitions {
		fmt.Printf("  %d) %s", t.Number, t.Status)
		if t.Phase != "" {
			fmt.Printf(" (phase: %s)", t.Phase)
		}
		fmt.Println()
		if t.Description != "" {
			fmt.Printf("     \"%s\"\n", t.Description)
		}
		if len(t.AgentTypes) > 0 {
			fmt.Printf("     Agents: %s\n", strings.Join(t.AgentTypes, ", "))
		}
		fmt.Println()
	}
}

// promptForSelection prompts user to select a transition number
func promptForSelection(max int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter selection [1-%d] or Ctrl+C to cancel: ", max)

	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > max {
		return 0, fmt.Errorf("invalid selection")
	}

	return selection, nil
}

// performTransition executes the status transition
func performTransition(ctx context.Context, taskRepo *repository.TaskRepository, repoDb *repository.DB, task *models.Task, targetStatus string, force bool, result *NextStatusResult) error {
	var err error
	if force {
		err = taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus(targetStatus), nil, nil, true)
	} else {
		err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatus(targetStatus), nil, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	result.NewStatus = targetStatus
	result.Transitioned = true

	// Trigger cascade
	triggerStatusCascade(ctx, repoDb, task.FeatureID)

	if cli.GlobalConfig.JSON {
		result.Message = fmt.Sprintf("Transitioned: %s -> %s", result.CurrentStatus, targetStatus)
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("Transitioned: %s -> %s", result.CurrentStatus, targetStatus))
	return nil
}
