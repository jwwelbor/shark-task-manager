package commands

import "github.com/jwwelbor/shark-task-manager/internal/cli"

// init registers all "manage" group commands in the desired display order.
// This file is named with a zzz_ prefix so its init() runs after all command
// variables have been created by other init() functions in this package.
func init() {
	cli.RootCmd.AddCommand(createCmd)
	cli.RootCmd.AddCommand(updateCmd)
	cli.RootCmd.AddCommand(deleteCmd)
	cli.RootCmd.AddCommand(statusCmd)
	cli.RootCmd.AddCommand(notesCmd)
	cli.RootCmd.AddCommand(contextCmd)
	cli.RootCmd.AddCommand(relatedDocsCmd)
	cli.RootCmd.AddCommand(historyCmd) // hidden
}
