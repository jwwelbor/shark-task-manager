package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <KEY> [flags]",
	Short: "Update an epic, feature, or task",
	Long: `Update an entity by key. The entity type is auto-detected from the key format.

Key format detection:
  E##                        Epic
  E##-F## or F##             Feature
  E##-F##-### or T-E##-F##-### Task

Use 'shark status set' to change entity status.

Examples:
  shark update E07 --title="New Epic Title"
  shark update E07-F01 --title="New Feature Title"
  shark update E07-F01-001 --title="New Task Title"
  shark update E07-F01-001 --priority=8`,
	GroupID: "manage",
	Args:    cobra.ExactArgs(1),
	RunE:    runUpdate,
}

func init() {
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().StringP("description", "d", "", "New description")
	updateCmd.Flags().IntP("priority", "p", -1, "New priority (1-10, -1=no change)")
	updateCmd.Flags().Int("order", -1, "New execution order (-1=no change)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	key := args[0]
	entityType := DetectEntityType(key)

	switch entityType {
	case "epic":
		return runEpicUpdate(cmd, args)
	case "feature":
		return runFeatureUpdate(cmd, args)
	case "task":
		return runTaskUpdate(cmd, args)
	default:
		return fmt.Errorf("cannot determine entity type from key: %s\nExpected format: E## (epic), E##-F## (feature), or E##-F##-### (task)", key)
	}
}
