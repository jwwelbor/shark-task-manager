package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:     "team",
	Short:   "Plan and execute council team runs",
	GroupID: "advanced",
}

func init() {
	cli.RootCmd.AddCommand(teamCmd)
}
