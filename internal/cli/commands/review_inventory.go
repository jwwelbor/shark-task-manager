package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/spf13/cobra"
)

// This read-only helper lets hosts without descriptor-relative Python opens
// use the same confined inventory implementation as parent gate admission.
func init() {
	cmd := &cobra.Command{
		Use:    "review-source-inventory <absolute-project-root>",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			digest, err := runner.ReviewSourceInventory(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), digest)
			return err
		},
	}
	adminCmd.AddCommand(cmd)
}
