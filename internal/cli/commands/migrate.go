package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:     "migrate",
	Short:   "Database migration utilities",
	GroupID: "setup",
	Long: `Run database migrations and data transformations.

These commands help maintain and upgrade the Shark task manager database schema
and data structures. Migrations are typically one-time operations that evolve
the database as new features are added.`,
	Example: `  # Backfill slugs from file paths
  shark migrate backfill-slugs

  # Preview backfill without applying changes
  shark migrate backfill-slugs --dry-run

  # Verbose output with detailed logging
  shark migrate backfill-slugs --verbose`,
}

func init() {
	cli.RootCmd.AddCommand(migrateCmd)
}
