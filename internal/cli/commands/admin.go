package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// adminCmd is the parent command for setup, configuration, and maintenance subcommands.
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Setup, configuration, and maintenance",
	Long: `Administrative commands for project setup, configuration management,
cloud database, migrations, and validation.

Subcommands:
  init        Initialize Shark CLI infrastructure
  config      Manage CLI configuration
  cloud       Manage cloud database configuration
  migrate     Database migration utilities
  validate    Validate database integrity
  workflow    Manage workflow configuration`,
	GroupID: "advanced",
}

func init() {
	cli.RootCmd.AddCommand(adminCmd)
}
