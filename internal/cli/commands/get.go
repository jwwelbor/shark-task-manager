package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// getCmd represents the unified get command
var getCmd = &cobra.Command{
	Use:     "get <KEY>",
	Short:   "Get epic, feature, or task details",
	GroupID: "essentials",
	Long: `Smart get command that dispatches to the appropriate subcommand based on arguments.

Positional Arguments:
  EPIC                  Get epic details (e.g., E04)
  EPIC FEATURE          Get feature details (e.g., E04 F01 or E04-F01)
  EPIC FEATURE TASKNUM  Get task details (e.g., E04 F01 001 or E04 F01 1)
  FULL_TASK_KEY         Get task details (e.g., T-E04-F01-001)

Examples:
  shark get E10                    Get epic E10 details
  shark get E10 F01                Get feature E10-F01 details
  shark get E10-F01                Get feature E10-F01 details (combined format)
  shark get E10 F01 001            Get task T-E10-F01-001 details
  shark get E10 F01 1              Get task T-E10-F01-001 details (short form)
  shark get T-E10-F01-001          Get task T-E10-F01-001 details (full key)
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

	// Dispatch to appropriate subcommand
	switch command {
	case "epic":
		// Call epic get command
		return runEpicGet(epicGetCmd, []string{key})

	case "feature":
		// Call feature get command
		return runFeatureGet(featureGetCmd, []string{key})

	case "task":
		// Call task get command
		return runTaskGet(taskGetCmd, []string{key})

	default:
		// Should never happen
		return nil
	}
}
