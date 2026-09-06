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
  init                Initialize Shark CLI infrastructure (DB, docs/plan/, config)
  install-shark-data  Extract embedded content bundle to shark-data/ for customization
  upgrade             Upgrade on-disk shark-data/ to the version in the current binary
  validate-data       Validate on-disk shark-data/ content bundle
  validate            Validate database integrity
  config              Manage CLI configuration
  cloud               Manage cloud database configuration
  migrate             Database migration utilities
  workflow            Manage workflow configuration
  overrides           Inspect and reconcile local shark-data overrides
  maintainer          Maintainer authorization commands`,
	GroupID: "advanced",
}

func init() {
	cli.RootCmd.AddCommand(adminCmd)
}
