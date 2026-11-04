package commands

import (
	"fmt"
	"strconv"

	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/spf13/cobra"
)

// This read-only helper lets hosts without descriptor-relative Python opens
// use the same confined inventory implementation as parent gate admission.
func init() {
	cmd := &cobra.Command{
		Use:    "review-source-inventory <absolute-repository-root> [relative-project-root]",
		Hidden: true,
		Args:   cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeProject := "."
			if len(args) == 2 {
				runtimeProject = args[1]
			}
			digest, err := runner.ReviewSourceInventoryForProject(cmd.Context(), args[0], runtimeProject)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), digest)
			return err
		},
	}
	adminCmd.AddCommand(cmd)
	read := &cobra.Command{
		Use:    "review-read-artifact <absolute-path> <byte-limit>",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			data, err := runner.ReviewReadArtifact(args[0], limit)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
	adminCmd.AddCommand(read)
}
