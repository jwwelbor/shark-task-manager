package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// Top-level command aliases for common task lifecycle operations.
// These provide shorter, more convenient alternatives to "shark task <command>".

// nextCmd is a top-level alias for "shark task next"
var nextCmd = &cobra.Command{
	Use:     "next",
	Short:   "Get next available task (alias for 'task next')",
	GroupID: "workflow",
	Long: `Shortcut for 'shark task next'. Gets the next available task based on priority and dependencies.

Find the next available task based on dependencies, priority, and agent type.

Examples:
  shark next                     Get next task
  shark next --agent=frontend    Get next frontend task`,
	RunE: runTaskNext,
}

// startCmd is a top-level alias for "shark task start"
var startCmd = &cobra.Command{
	Use:     "start <task-key>",
	Short:   "Start working on a task (alias for 'task start')",
	GroupID: "workflow",
	Long: `Shortcut for 'shark task start'. Mark a task as in_progress and update timestamps.

Use --force to bypass status transition validation. This allows starting a task
from any status (not just 'todo'). Use with caution as this is an administrative override.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark start E07-F01-001
  shark start T-E04-F01-001
  shark start T-E04-F01-001-user-auth`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskStart,
}

// doneCmd is a top-level alias for "shark task complete"
var doneCmd = &cobra.Command{
	Use:     "done <task-key>",
	Short:   "Mark task as complete (alias for 'task complete')",
	GroupID: "workflow",
	Long: `Shortcut for 'shark task complete'. Mark a task as ready_for_review and update timestamps.

Use --force to bypass status transition validation. This allows marking a task complete
from any status (not just 'in_progress'). Use with caution as this is an administrative override.

Supports multiple key formats (numeric, full, or slugged).

Examples:
  shark done E07-F01-001
  shark done T-E04-F01-001 --notes "Implemented feature X"
  shark done T-E04-F01-001-user-auth --summary "Added JWT authentication"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskComplete,
}

// blockCmd is a top-level alias for "shark task block"
var blockCmd = &cobra.Command{
	Use:     "block <task-key>",
	Short:   "Block a task (alias for 'task block')",
	GroupID: "workflow",
	Long: `Shortcut for 'shark task block'. Mark a task as blocked with a required reason.

Use --force to bypass status transition validation. This allows blocking a task
from any status (not just 'todo' or 'in_progress'). Use with caution as this is an administrative override.

Examples:
  shark block E07-F01-001 --reason "Waiting for API spec"
  shark block T-E04-F01-001 -r "Dependencies not ready"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskBlock,
}

// unblockCmd is a top-level alias for "shark task unblock"
var unblockCmd = &cobra.Command{
	Use:     "unblock <task-key>",
	Short:   "Unblock a task (alias for 'task unblock')",
	GroupID: "workflow",
	Long: `Shortcut for 'shark task unblock'. Unblock a task and return it to todo status.

Use --force to bypass status transition validation. This allows unblocking a task
from any status (not just 'blocked'). Use with caution as this is an administrative override.

Examples:
  shark unblock E07-F01-001
  shark unblock T-E04-F01-001`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUnblock,
}

func init() {
	// Register top-level aliases
	cli.RootCmd.AddCommand(nextCmd)
	cli.RootCmd.AddCommand(startCmd)
	cli.RootCmd.AddCommand(doneCmd)
	cli.RootCmd.AddCommand(blockCmd)
	cli.RootCmd.AddCommand(unblockCmd)

	// Add flags for next command
	nextCmd.Flags().StringP("agent", "a", "", "Agent type to match")
	nextCmd.Flags().StringP("epic", "e", "", "Filter by epic key")

	// Add flags for start command
	startCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	startCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Add flags for done command
	doneCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	doneCmd.Flags().StringP("notes", "n", "", "Completion notes")
	doneCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Completion metadata flags
	doneCmd.Flags().StringSlice("files-created", []string{}, "Files created during task (repeatable)")
	doneCmd.Flags().StringSlice("files-modified", []string{}, "Files modified during task (repeatable)")
	doneCmd.Flags().String("tests", "", "Test status summary (e.g., '16/16 passing')")
	doneCmd.Flags().String("summary", "", "Completion summary describing what was delivered")
	doneCmd.Flags().Bool("verified", false, "Mark task as verified")
	doneCmd.Flags().String("agent-id", "", "Agent execution ID for traceability")
	doneCmd.Flags().Int("time-spent", 0, "Time spent in minutes")

	// Add flags for block command
	blockCmd.Flags().StringP("reason", "r", "", "Reason for blocking (required)")
	blockCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	blockCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Add flags for unblock command
	unblockCmd.Flags().StringP("agent", "", "", "Agent identifier (defaults to USER env var)")
	unblockCmd.Flags().Bool("force", false, "Force status change bypassing validation (use with caution)")

	// Override unblock help text with config-driven default status
	defaultStatus := cli.GetWorkflowService().GetDefaultStatus()
	unblockCmd.Long = fmt.Sprintf(`Shortcut for 'shark task unblock'. Unblock a task and return it to %s status.

Use --force to bypass status transition validation. This allows unblocking a task
from any status (not just 'blocked'). Use with caution as this is an administrative override.

Examples:
  shark unblock E07-F01-001
  shark unblock T-E04-F01-001`, defaultStatus)
}
