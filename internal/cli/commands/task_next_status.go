package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
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
	taskNextStatusCmd.Flags().String("reason", "", "Rejection reason for backward transitions")
	taskNextStatusCmd.Flags().String("reason-doc", "", "Path to document detailing rejection reason (relative to project root)")
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
	AutoUnblocked        []string           `json:"auto_unblocked,omitempty"`
}

func runTaskNextStatus(cmd *cobra.Command, args []string) error {
	// Step 1: Parse arguments
	taskKey, err := NormalizeTaskKey(args[0])
	if err != nil {
		return fmt.Errorf("invalid task key: %w", err)
	}

	targetStatus, _ := cmd.Flags().GetString("status")
	preview, _ := cmd.Flags().GetBool("preview")
	force, _ := cmd.Flags().GetBool("force")
	reason, _ := cmd.Flags().GetString("reason")
	reasonDocFlag, _ := cmd.Flags().GetString("reason-doc")

	// Validate and resolve reason-doc path
	var documentPath string
	if reasonDocFlag != "" {
		if err := ValidateRejectionReasonDocPath(reasonDocFlag); err != nil {
			cli.Error(fmt.Sprintf("Invalid document path: %s", err.Error()))
			os.Exit(1)
		}

		projectRoot, findErr := cli.FindProjectRoot()
		if findErr != nil {
			return fmt.Errorf("failed to find project root: %w", findErr)
		}

		fullPath := filepath.Join(projectRoot, reasonDocFlag)
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			cli.Error(fmt.Sprintf("Document not found: %s", reasonDocFlag))
			cli.Info(fmt.Sprintf("Looked for file at: %s", fullPath))
			os.Exit(1)
		}

		documentPath = reasonDocFlag
	}

	// Step 2: Call service to get current status and available transitions
	svc := cli.GetTaskServiceWithDocs()
	info, err := svc.GetNextStatus(cmd.Context(), taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task status: %w", err)
	}

	// Build result for display/JSON output
	result := NextStatusResult{
		TaskKey:       info.EntityKey,
		CurrentStatus: info.CurrentStatus,
		CurrentPhase:  info.CurrentPhase,
	}

	// Build transition choices from service response
	for i, t := range info.AvailableTransitions {
		result.AvailableTransitions = append(result.AvailableTransitions, TransitionChoice{
			Number:      i + 1,
			Status:      t.TargetStatus,
			Description: t.Description,
			Phase:       t.Phase,
			AgentTypes:  t.AgentTypes,
			Color:       t.Color,
		})
	}

	// Step 3: Handle terminal / no-transitions case
	if len(info.AvailableTransitions) == 0 {
		if info.IsTerminal {
			result.Message = fmt.Sprintf("Task is in terminal status '%s' - no transitions available", info.CurrentStatus)
		} else {
			result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Preview mode - show transitions without making changes
	if preview {
		result.Message = "Preview mode - no changes made"

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		fmt.Printf("\nTask: %s\n", info.EntityKey)
		fmt.Printf("Current status: %s", info.CurrentStatus)
		if info.CurrentPhase != "" {
			fmt.Printf(" (phase: %s)", info.CurrentPhase)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("Available transitions:")
		printTransitions(result.AvailableTransitions)
		fmt.Println()
		fmt.Println("Use 'shark task next-status " + info.EntityKey + "' to transition")
		return nil
	}

	// Direct transition mode (--status flag)
	if targetStatus != "" {
		// Validate target status against available transitions
		valid := false
		for _, t := range info.AvailableTransitions {
			if strings.EqualFold(t.TargetStatus, targetStatus) {
				valid = true
				targetStatus = t.TargetStatus // use canonical case
				break
			}
		}

		if !valid && !force {
			cli.Error(fmt.Sprintf("Invalid transition: '%s' -> '%s'", info.CurrentStatus, targetStatus))
			fmt.Println()
			fmt.Println("Valid transitions from current status:")
			for _, t := range info.AvailableTransitions {
				fmt.Printf("  - %s\n", t.TargetStatus)
			}
			fmt.Println()
			fmt.Println("Use --force to bypass workflow validation")
			return fmt.Errorf("invalid transition")
		}

		return doTransition(svc, cmd, taskKey, targetStatus, force, reason, documentPath, &result)
	}

	// Load config to check interactive mode setting
	projectRoot, _ := cli.FindProjectRoot()
	if projectRoot == "" {
		projectRoot = "."
	}
	cfgManager := config.NewManager(projectRoot)
	cfg, cfgErr := cfgManager.Load()
	if cfgErr != nil {
		cfg = &config.Config{}
	}

	interactiveMode := cfg.IsInteractiveModeEnabled()

	// Non-interactive mode: auto-select first transition
	if !interactiveMode {
		if cli.GlobalConfig.JSON {
			result.Message = "Use --status=<name> to specify target status for JSON output"
			return cli.OutputJSON(result)
		}

		targetStatus = info.AvailableTransitions[0].TargetStatus
		cli.Info(fmt.Sprintf("Auto-selected next status: %s (from %d options)", targetStatus, len(info.AvailableTransitions)))
		return doTransition(svc, cmd, taskKey, targetStatus, force, reason, documentPath, &result)
	}

	// Interactive mode
	if cli.GlobalConfig.JSON {
		result.Message = "Use --status=<name> to specify target status for JSON output"
		return cli.OutputJSON(result)
	}

	// Show interactive prompt
	fmt.Printf("\nTask: %s\n", info.EntityKey)
	fmt.Printf("Current status: %s", info.CurrentStatus)
	if info.CurrentPhase != "" {
		fmt.Printf(" (phase: %s)", info.CurrentPhase)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("Available transitions:")
	printTransitions(result.AvailableTransitions)
	fmt.Println()

	selection, err := promptForSelection(len(info.AvailableTransitions))
	if err != nil {
		cli.Info("Cancelled - no changes made")
		return nil
	}

	targetStatus = info.AvailableTransitions[selection-1].TargetStatus
	return doTransition(svc, cmd, taskKey, targetStatus, force, reason, documentPath, &result)
}

// doTransition calls the service to perform the actual status transition.
func doTransition(svc *services.TaskService, cmd *cobra.Command, taskKey, targetStatus string, force bool, reason, documentPath string, result *NextStatusResult) error {
	opts := services.TransitionOptions{
		Force:        force,
		Reason:       reason,
		DocumentPath: documentPath,
	}

	transResult, err := svc.TransitionStatus(cmd.Context(), taskKey, targetStatus, opts)
	if err != nil {
		return fmt.Errorf("failed to transition status: %w", err)
	}

	result.NewStatus = transResult.ToStatus
	result.Transitioned = transResult.Transitioned

	// Extract auto-unblocked keys from message if present
	if strings.Contains(transResult.Message, "auto-unblocked:") {
		parts := strings.SplitN(transResult.Message, "auto-unblocked:", 2)
		if len(parts) == 2 {
			for _, k := range strings.Split(strings.TrimSpace(parts[1]), ", ") {
				k = strings.TrimSpace(k)
				if k != "" {
					result.AutoUnblocked = append(result.AutoUnblocked, k)
				}
			}
		}
	}

	if cli.GlobalConfig.JSON {
		result.Message = transResult.Message
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("Transitioned: %s -> %s", result.CurrentStatus, result.NewStatus))
	displayAutoUnblockedTasks(result.AutoUnblocked)
	return nil
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
