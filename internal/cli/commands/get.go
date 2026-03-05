package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// getCmd represents the unified get command
var getCmd = &cobra.Command{
	Use:     "get <KEY>",
	Short:   "Get epic, feature, task, bug, change, or idea details",
	GroupID: "inspect",
	Long: `Smart get command that dispatches to the appropriate subcommand based on arguments.

Positional Arguments:
  EPIC                  Get epic details (e.g., E04)
  EPIC FEATURE          Get feature details (e.g., E04 F01 or E04-F01)
  EPIC FEATURE TASKNUM  Get task details (e.g., E04 F01 001 or E04 F01 1)
  FULL_TASK_KEY         Get task details (e.g., T-E04-F01-001)
  BUG_KEY               Get bug details (e.g., B001)
  CHANGE_CARD_KEY       Get change-card details (e.g., CC-001)
  IDEA_KEY              Get idea details (e.g., I-2026-01-01-01)

Examples:
  shark get E10                    Get epic E10 details
  shark get E10 F01                Get feature E10-F01 details
  shark get E10-F01                Get feature E10-F01 details (combined format)
  shark get E10 F01 001            Get task T-E10-F01-001 details
  shark get E10 F01 1              Get task T-E10-F01-001 details (short form)
  shark get T-E10-F01-001          Get task T-E10-F01-001 details (full key)
  shark get B001                   Get bug B001 details
  shark get CC-001                 Get change-card CC-001 details
  shark get I-2026-01-01-01        Get idea details
  shark get E10 --json             Output as JSON`,
	RunE: runGet,
}

func init() {
	// Register get command with root
	cli.RootCmd.AddCommand(getCmd)
}

// runGet executes the get command dispatcher
func runGet(cmd *cobra.Command, args []string) error {
	// Parse arguments to determine which subcommand to invoke
	command, key, err := ParseGetArgs(args)
	if err != nil {
		return err
	}

	// Dispatch to appropriate subcommand, passing cmd so the live context is
	// preserved. Static vars (epicGetCmd, featureGetCmd, taskGetCmd) are never
	// executed through Cobra's lifecycle and always have nil Context().
	switch command {
	case "epic":
		return runEpicGet(cmd, []string{key})

	case "feature":
		return runFeatureGet(cmd, []string{key})

	case "task":
		return runTaskGet(cmd, []string{key})

	case "bug":
		return runBugGet(cmd, []string{key})

	case "change":
		return runChangeGet(cmd, []string{key})

	case "change_card":
		return runChangeCardGet(cmd, []string{key})

	case "idea":
		return runIdeaGet(cmd, []string{key})

	default:
		// Should never happen
		return nil
	}
}
