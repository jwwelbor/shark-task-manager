package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Delete an epic, feature, or task",
	Long: `Delete an entity by key. The entity type is auto-detected from the key format.

Key format detection:
  E##                        Epic
  E##-F## or F##             Feature
  E##-F##-### or T-E##-F##-### Task

Examples:
  shark delete E07                    Delete epic E07
  shark delete E07-F01                Delete feature E07-F01
  shark delete E07-F01-001            Delete task E07-F01-001
  shark delete E07-F01 --force        Force delete (cascade children)`,
	GroupID: "essentials",
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func init() {
	cli.RootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().Bool("force", false, "Force deletion (cascade delete children)")
}

func runDelete(cmd *cobra.Command, args []string) error {
	key := args[0]
	entityType := DetectEntityType(key)

	switch entityType {
	case "epic":
		return runEpicDelete(cmd, args)
	case "feature":
		return runFeatureDelete(cmd, args)
	case "task":
		return runTaskDelete(cmd, args)
	default:
		return fmt.Errorf("cannot determine entity type from key: %s\nExpected format: E## (epic), E##-F## (feature), or E##-F##-### (task)", key)
	}
}
