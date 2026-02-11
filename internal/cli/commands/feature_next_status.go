package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// featureNextStatusCmd progresses a feature to its next workflow status
var featureNextStatusCmd = &cobra.Command{
	Use:   "next-status <feature-key>",
	Short: "Progress feature to next workflow status",
	Long: `Progress a feature through its configured workflow by selecting from available transitions.

When a feature has multiple valid next statuses, this command auto-selects the first one.
For automation/scripting, use --status to specify the target directly.

Flags:
  --status=<name>   Transition directly to this status (non-interactive)
  --preview         Show available transitions without making changes
  --force           Bypass workflow validation (administrative override)

Examples:
  shark feature next-status E16-F01              Auto-advance to next status
  shark feature next-status E16-F01 --preview    Show available transitions
  shark feature next-status E16-F01 --status=active  Direct transition
  shark feature next-status E16-F01 --json       JSON output (for scripting)`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureNextStatus,
}

func init() {
	featureNextStatusCmd.Flags().String("status", "", "Target status for direct transition (non-interactive)")
	featureNextStatusCmd.Flags().Bool("preview", false, "Show available transitions without making changes")
	featureNextStatusCmd.Flags().Bool("force", false, "Bypass workflow validation")
	featureNextStatusCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	featureNextStatusCmd.Flags().String("agent", "", "Agent or user performing the transition")
	featureCmd.AddCommand(featureNextStatusCmd)
}

func runFeatureNextStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureKey := strings.ToUpper(strings.TrimSpace(args[0]))

	// Get flags
	targetStatus, _ := cmd.Flags().GetString("status")
	preview, _ := cmd.Flags().GetBool("preview")
	force, _ := cmd.Flags().GetBool("force")
	reason, _ := cmd.Flags().GetString("reason")
	agent, _ := cmd.Flags().GetString("agent")

	// Get service
	featureSvc := cli.GetFeatureService()

	// Get current status info
	info, err := featureSvc.GetNextStatus(ctx, featureKey)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			cli.Error(fmt.Sprintf("Feature %s not found", featureKey))
			os.Exit(1)
		}
		return fmt.Errorf("failed to get feature status: %w", err)
	}

	// Build result for JSON output
	result := buildNextStatusResult("feature", info)

	// Handle terminal status
	if info.IsTerminal {
		result.Message = fmt.Sprintf("Feature is in terminal status '%s' - no transitions available", info.CurrentStatus)

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

		fmt.Printf("\nFeature: %s\n", info.EntityKey)
		fmt.Printf("Current status: %s", info.CurrentStatus)
		if info.CurrentPhase != "" {
			fmt.Printf(" (phase: %s)", info.CurrentPhase)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("Available transitions:")
		printEntityTransitions(result.AvailableTransitions)
		fmt.Println()
		fmt.Println("Use 'shark feature next-status " + info.EntityKey + "' to transition")
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
		return performEntityTransition(ctx, featureSvc, info.EntityKey, targetStatus, opts, result)
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
	return performEntityTransition(ctx, featureSvc, info.EntityKey, targetStatus, opts, result)
}
