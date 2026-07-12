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

// nextStatusGetter can retrieve next-status info for an entity.
type nextStatusGetter interface {
	entityTransitioner
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// makeNextStatusCmd creates a "next-status" command for the given entity type.
func makeNextStatusCmd(entityName string, getSvc func() nextStatusGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("next-status <%s-key>", entityName),
		Short: fmt.Sprintf("Progress %s to next workflow status", entityName),
		Long: fmt.Sprintf(`Progress %s through its configured workflow by selecting from available transitions.

When %s has multiple valid next statuses, this command auto-selects the first one.
For automation/scripting, use --status to specify the target directly.

Flags:
  --status=<name>   Transition directly to this status (non-interactive)
  --preview         Show available transitions without making changes
  --force           Bypass workflow validation (administrative override)

Examples:
  shark %s next-status <key>              Auto-advance to next status
  shark %s next-status <key> --preview    Show available transitions
  shark %s next-status <key> --status=active  Direct transition
  shark %s next-status <key> --json       JSON output (for scripting)`,
			articleEntity(entityName), articleEntity(entityName),
			entityName, entityName, entityName, entityName),
		Args: cobra.ExactArgs(1),
		RunE: makeRunNextStatus(entityName, getSvc),
	}

	cmd.Flags().String("status", "", "Target status for direct transition (non-interactive)")
	cmd.Flags().Bool("preview", false, "Show available transitions without making changes")
	cmd.Flags().Bool("force", false, "Bypass workflow validation")
	cmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	cmd.Flags().String("agent", "", "Agent or user performing the transition")
	return cmd
}

func makeRunNextStatus(entityName string, getSvc func() nextStatusGetter) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		entityKey := strings.ToUpper(strings.TrimSpace(args[0]))

		targetStatus, _ := cmd.Flags().GetString("status")
		preview, _ := cmd.Flags().GetBool("preview")
		force, _ := cmd.Flags().GetBool("force")
		reason, _ := cmd.Flags().GetString("reason")
		agent, _ := cmd.Flags().GetString("agent")

		svc := getSvc()

		info, err := svc.GetNextStatus(ctx, entityKey)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				displayName := strings.ToUpper(entityName[:1]) + entityName[1:]
				cli.Error(fmt.Sprintf("%s %s not found", displayName, entityKey))
				os.Exit(1)
			}
			return fmt.Errorf("failed to get %s status: %w", entityName, err)
		}

		result := buildNextStatusResult(entityName, info)

		if info.IsTerminal {
			displayName := strings.ToUpper(entityName[:1]) + entityName[1:]
			result.Message = fmt.Sprintf("%s is in terminal status '%s' - no transitions available", displayName, info.CurrentStatus)

			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(result)
			}

			cli.Warning(result.Message)
			return nil
		}

		if len(info.AvailableTransitions) == 0 {
			result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)

			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(result)
			}

			cli.Warning(result.Message)
			return nil
		}

		if preview {
			result.Message = "Preview mode - no changes made"

			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(result)
			}

			displayName := strings.ToUpper(entityName[:1]) + entityName[1:]
			fmt.Printf("\n%s: %s\n", displayName, info.EntityKey)
			fmt.Printf("Current status: %s", info.CurrentStatus)
			if info.CurrentPhase != "" {
				fmt.Printf(" (phase: %s)", info.CurrentPhase)
			}
			fmt.Println()
			fmt.Println()
			fmt.Println("Available transitions:")
			printEntityTransitions(result.AvailableTransitions)
			fmt.Println()
			fmt.Printf("Use 'shark %s next-status %s' to transition\n", entityName, info.EntityKey)
			return nil
		}

		if targetStatus != "" {
			targetStatus = strings.TrimSpace(strings.ToLower(targetStatus))

			if !force {
				valid := false
				for _, t := range info.AvailableTransitions {
					if strings.EqualFold(t.TargetStatus, targetStatus) {
						valid = true
						targetStatus = t.TargetStatus
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
			return performEntityTransition(ctx, svc, info.EntityKey, targetStatus, opts, result)
		}

		targetStatus = info.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets

		if cli.GlobalConfig.JSON {
			result.Message = "Use --status=<name> to specify target status for JSON output"
			return cli.OutputJSON(result)
		}

		cli.Info(fmt.Sprintf("Auto-selected next status: %s (from %d options)", targetStatus, len(info.AvailableTransitions)))
		opts := services.TransitionOptions{Force: force, Reason: reason, Agent: agent}
		return performEntityTransition(ctx, svc, info.EntityKey, targetStatus, opts, result)
	}
}

// makeSetStatusCmd creates a "set-status" command for the given entity type.
func makeSetStatusCmd(entityName string, getSvc func() entityTransitioner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("set-status <%s-key> <status>", entityName),
		Short: fmt.Sprintf("Set %s to a specific workflow status", entityName),
		Long: fmt.Sprintf(`Set %s to a specific workflow status with validation and backward transition guards.

Backward transitions (moving to an earlier workflow phase) require --reason.
Use --force to bypass workflow validation (requires --reason).

Flags:
  --reason=<text>   Reason for backward or forced transitions (required for backward)
  --force           Bypass workflow validation (administrative override, requires --reason)
  --agent=<name>    Agent or user performing the transition

Examples:
  shark %s set-status <key> active                        Forward transition
  shark %s set-status <key> draft --reason="Rework needed"  Backward transition
  shark %s set-status <key> custom --force --reason="Override"  Force transition
  shark %s set-status <key> active --json                 JSON output`,
			articleEntity(entityName), entityName, entityName, entityName, entityName),
		Args: cobra.ExactArgs(2),
		RunE: makeRunSetStatus(entityName, getSvc),
	}

	cmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	cmd.Flags().Bool("force", false, "Bypass workflow validation (requires --reason)")
	cmd.Flags().String("agent", "", "Agent or user performing the transition")
	return cmd
}

func makeRunSetStatus(entityName string, getSvc func() entityTransitioner) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		entityKey := strings.ToUpper(strings.TrimSpace(args[0]))
		targetStatus := strings.TrimSpace(strings.ToLower(args[1]))

		reason, _ := cmd.Flags().GetString("reason")
		force, _ := cmd.Flags().GetBool("force")
		agent, _ := cmd.Flags().GetString("agent")

		svc := getSvc()

		opts := services.TransitionOptions{
			Force:  force,
			Reason: reason,
			Agent:  agent,
		}

		result, err := svc.TransitionStatus(ctx, entityKey, targetStatus, opts)
		if err != nil {
			if strings.Contains(err.Error(), "reason required") || strings.Contains(err.Error(), "reason is required") {
				cli.Error(err.Error())
				os.Exit(3)
			}
			if strings.Contains(err.Error(), "not found") {
				displayName := strings.ToUpper(entityName[:1]) + entityName[1:]
				cli.Error(fmt.Sprintf("%s %s not found", displayName, entityKey))
				os.Exit(1)
			}
			return fmt.Errorf("failed to set %s status: %w", entityName, err)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		displayName := strings.ToUpper(entityName[:1]) + entityName[1:]
		cli.Success(fmt.Sprintf("%s %s: %s -> %s", displayName, result.EntityKey, result.FromStatus, result.ToStatus))
		if result.IsBackward && result.Reason != "" {
			cli.Info(fmt.Sprintf("Reason: %s", result.Reason))
		}
		if result.IsForced {
			cli.Warning("Workflow validation was bypassed with --force")
		}
		if result.ChildCount > 0 {
			cli.Warning(fmt.Sprintf("%d child entities remain in current states.", result.ChildCount))
		}
		cli.Info(fmt.Sprintf("Run `shark get %s --field orchestrator_action` to get your next instructions.", result.EntityKey))
		return nil
	}
}
