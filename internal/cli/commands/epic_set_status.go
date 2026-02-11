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

// epicSetStatusCmd sets an epic to a specific workflow status
var epicSetStatusCmd = &cobra.Command{
	Use:   "set-status <epic-key> <status>",
	Short: "Set epic to a specific workflow status",
	Long: `Set an epic to a specific workflow status with validation and backward transition guards.

Backward transitions (moving to an earlier workflow phase) require --reason.
Use --force to bypass workflow validation (requires --reason).

Flags:
  --reason=<text>   Reason for backward or forced transitions (required for backward)
  --force           Bypass workflow validation (administrative override, requires --reason)
  --agent=<name>    Agent or user performing the transition

Examples:
  shark epic set-status E16 active                        Forward transition
  shark epic set-status E16 draft --reason="Rework needed"  Backward transition
  shark epic set-status E16 custom --force --reason="Override"  Force transition
  shark epic set-status E16 active --json                 JSON output`,
	Args: cobra.ExactArgs(2),
	RunE: runEpicSetStatus,
}

func init() {
	epicSetStatusCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	epicSetStatusCmd.Flags().Bool("force", false, "Bypass workflow validation (requires --reason)")
	epicSetStatusCmd.Flags().String("agent", "", "Agent or user performing the transition")
	epicCmd.AddCommand(epicSetStatusCmd)
}

func runEpicSetStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := strings.ToUpper(strings.TrimSpace(args[0]))
	targetStatus := strings.TrimSpace(strings.ToLower(args[1]))

	// Get flags
	reason, _ := cmd.Flags().GetString("reason")
	force, _ := cmd.Flags().GetBool("force")
	agent, _ := cmd.Flags().GetString("agent")

	// Get service
	epicSvc := cli.GetEpicService()

	// Build options
	opts := services.TransitionOptions{
		Force:  force,
		Reason: reason,
		Agent:  agent,
	}

	// Perform transition
	result, err := epicSvc.TransitionStatus(ctx, epicKey, targetStatus, opts)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			cli.Error(fmt.Sprintf("Epic %s not found", epicKey))
			os.Exit(1)
		}
		if strings.Contains(errMsg, "requires --reason") {
			cli.Error(errMsg)
			os.Exit(3)
		}
		return fmt.Errorf("failed to set epic status: %w", err)
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("Epic %s: %s -> %s", result.EntityKey, result.FromStatus, result.ToStatus))
	if result.IsBackward && result.Reason != "" {
		cli.Info(fmt.Sprintf("Reason: %s", result.Reason))
	}
	if result.IsForced {
		cli.Warning("Workflow validation was bypassed with --force")
	}
	if result.ChildCount > 0 {
		cli.Warning(fmt.Sprintf("%d features remain in current states.", result.ChildCount))
	}
	displayOrchestratorAction(result.OrchestratorAction)
	return nil
}
