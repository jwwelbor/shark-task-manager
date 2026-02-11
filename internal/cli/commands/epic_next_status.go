package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/spf13/cobra"
)

// epicNextStatusCmd progresses an epic to its next workflow status
var epicNextStatusCmd = &cobra.Command{
	Use:   "next-status <epic-key>",
	Short: "Progress epic to next workflow status",
	Long: `Progress an epic through its configured workflow by selecting from available transitions.

When an epic has multiple valid next statuses, this command auto-selects the first one.
For automation/scripting, use --status to specify the target directly.

Flags:
  --status=<name>   Transition directly to this status (non-interactive)
  --preview         Show available transitions without making changes
  --force           Bypass workflow validation (administrative override)

Examples:
  shark epic next-status E16              Auto-advance to next status
  shark epic next-status E16 --preview    Show available transitions
  shark epic next-status E16 --status=active  Direct transition
  shark epic next-status E16 --json       JSON output (for scripting)`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicNextStatus,
}

func init() {
	epicNextStatusCmd.Flags().String("status", "", "Target status for direct transition (non-interactive)")
	epicNextStatusCmd.Flags().Bool("preview", false, "Show available transitions without making changes")
	epicNextStatusCmd.Flags().Bool("force", false, "Bypass workflow validation")
	epicNextStatusCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	epicNextStatusCmd.Flags().String("agent", "", "Agent or user performing the transition")
	epicCmd.AddCommand(epicNextStatusCmd)
}

func runEpicNextStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := strings.ToUpper(strings.TrimSpace(args[0]))

	// Get flags
	targetStatus, _ := cmd.Flags().GetString("status")
	preview, _ := cmd.Flags().GetBool("preview")
	force, _ := cmd.Flags().GetBool("force")
	reason, _ := cmd.Flags().GetString("reason")
	agent, _ := cmd.Flags().GetString("agent")

	// Get service
	epicSvc := cli.GetEpicService()

	// Get current status info
	info, err := epicSvc.GetNextStatus(ctx, epicKey)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			cli.Error(fmt.Sprintf("Epic %s not found", epicKey))
			os.Exit(1)
		}
		return fmt.Errorf("failed to get epic status: %w", err)
	}

	// Build result for JSON output
	result := buildNextStatusResult("epic", info)

	// Handle terminal status
	if info.IsTerminal {
		result.Message = fmt.Sprintf("Epic is in terminal status '%s' - no transitions available", info.CurrentStatus)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Handle no transitions
	if len(info.AvailableTransitions) == 0 {
		result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Preview mode
	if preview {
		result.Message = "Preview mode - no changes made"

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		fmt.Printf("\nEpic: %s\n", info.EntityKey)
		fmt.Printf("Current status: %s", info.CurrentStatus)
		if info.CurrentPhase != "" {
			fmt.Printf(" (phase: %s)", info.CurrentPhase)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("Available transitions:")
		printEntityTransitions(result.AvailableTransitions)
		fmt.Println()
		fmt.Println("Use 'shark epic next-status " + info.EntityKey + "' to transition")
		return nil
	}

	// Direct transition mode (--status flag)
	if targetStatus != "" {
		targetStatus = strings.TrimSpace(strings.ToLower(targetStatus))

		// Validate target against available transitions (unless force)
		if !force {
			valid := false
			for _, t := range info.AvailableTransitions {
				if strings.EqualFold(t.TargetStatus, targetStatus) {
					valid = true
					targetStatus = t.TargetStatus // Use canonical case
					break
				}
			}

			if !valid {
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
		}

		opts := services.TransitionOptions{Force: force, Reason: reason, Agent: agent}
		return performEntityTransition(ctx, epicSvc, info.EntityKey, targetStatus, opts, result)
	}

	// Auto-select first transition
	targetStatus = info.AvailableTransitions[0].TargetStatus

	if cli.GlobalConfig.JSON {
		// JSON mode - return available transitions for scripting
		result.Message = "Use --status=<name> to specify target status for JSON output"
		return cli.OutputJSON(result)
	}

	cli.Info(fmt.Sprintf("Auto-selected next status: %s (from %d options)", targetStatus, len(info.AvailableTransitions)))
	opts := services.TransitionOptions{Force: force, Reason: reason, Agent: agent}
	return performEntityTransition(ctx, epicSvc, info.EntityKey, targetStatus, opts, result)
}

// entityTransitioner is an interface for performing status transitions on entities.
type entityTransitioner interface {
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}

// performEntityTransition executes a status transition via the service layer.
func performEntityTransition(ctx context.Context, svc entityTransitioner, entityKey string, targetStatus string, opts services.TransitionOptions, result *EntityNextStatusResult) error {
	if opts.Force {
		cli.Warning("Workflow validation bypassed with --force")
	}

	transResult, err := svc.TransitionStatus(ctx, entityKey, targetStatus, opts)
	if err != nil {
		return fmt.Errorf("failed to transition status: %w", err)
	}

	result.NewStatus = transResult.ToStatus
	result.Transitioned = true

	if cli.GlobalConfig.JSON {
		result.Message = fmt.Sprintf("Transitioned: %s -> %s", transResult.FromStatus, transResult.ToStatus)
		result.IsBackward = transResult.IsBackward
		result.Reason = transResult.Reason
		result.ChildCount = transResult.ChildCount
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("Transitioned: %s -> %s", transResult.FromStatus, transResult.ToStatus))
	if transResult.IsBackward && transResult.Reason != "" {
		cli.Info(fmt.Sprintf("Reason: %s", transResult.Reason))
	}
	if transResult.ChildCount > 0 {
		cli.Warning(fmt.Sprintf("%d child entities remain in current states.", transResult.ChildCount))
	}
	displayOrchestratorAction(transResult.OrchestratorAction)
	return nil
}

// EntityNextStatusResult contains the result of a next-status operation for epics/features.
type EntityNextStatusResult struct {
	EntityType           string                   `json:"entity_type"`
	EntityKey            string                   `json:"entity_key"`
	CurrentStatus        string                   `json:"current_status"`
	CurrentPhase         string                   `json:"current_phase,omitempty"`
	AvailableTransitions []EntityTransitionChoice `json:"available_transitions"`
	NewStatus            string                   `json:"new_status,omitempty"`
	Transitioned         bool                     `json:"transitioned"`
	Message              string                   `json:"message,omitempty"`
	IsBackward           bool                     `json:"is_backward,omitempty"`
	Reason               string                   `json:"reason,omitempty"`
	ChildCount           int                      `json:"child_count,omitempty"`
}

// EntityTransitionChoice represents a valid status transition for display.
type EntityTransitionChoice struct {
	Number      int    `json:"number"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Phase       string `json:"phase,omitempty"`
}

// buildNextStatusResult constructs an EntityNextStatusResult from service info.
func buildNextStatusResult(entityType string, info *services.NextStatusInfo) *EntityNextStatusResult {
	result := &EntityNextStatusResult{
		EntityType:    entityType,
		EntityKey:     info.EntityKey,
		CurrentStatus: info.CurrentStatus,
		CurrentPhase:  info.CurrentPhase,
	}

	for i, t := range info.AvailableTransitions {
		result.AvailableTransitions = append(result.AvailableTransitions, EntityTransitionChoice{
			Number:      i + 1,
			Status:      t.TargetStatus,
			Description: t.Description,
			Phase:       t.Phase,
		})
	}

	return result
}

// printEntityTransitions prints available transitions in a formatted list.
func printEntityTransitions(transitions []EntityTransitionChoice) {
	for _, t := range transitions {
		fmt.Printf("  %d) %s", t.Number, t.Status)
		if t.Phase != "" {
			fmt.Printf(" (phase: %s)", t.Phase)
		}
		fmt.Println()
		if t.Description != "" {
			fmt.Printf("     \"%s\"\n", t.Description)
		}
		fmt.Println()
	}
}

// GetWorkflowServiceForLevel is a helper to get a level-specific workflow service.
func GetWorkflowServiceForLevel(level string) *workflow.Service {
	return cli.GetWorkflowService().ForLevel(level)
}
