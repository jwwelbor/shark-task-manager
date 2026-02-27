package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Change status for epic, feature, or task",
	GroupID: "workflow",
	Long: `Commands for managing entity statuses: set a status directly, advance through the
workflow, view available transitions, or inspect status change history.

Subcommands:
  set <key> <status>     Set entity to a specific status
  advance <key>          Advance entity to next workflow status
  transitions <key>      Show available status transitions
  history <key>          Show task status change history

Examples:
  shark status set E07-F01-001 in_development          Set task status
  shark status set E07-F01-001 todo --reason="Rework"   Force backward transition
  shark status advance E07-F01-001                      Advance to next status
  shark status transitions E07-F01-001                  Show valid transitions
  shark status history E07-F01-001                      View change history

Use 'shark progress' for project/entity progress dashboards.
Use 'shark get <key>' for entity details.`,
}

func init() {
	// Register status command with root
	cli.RootCmd.AddCommand(statusCmd)
}
