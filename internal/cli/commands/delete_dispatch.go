package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Delete an epic, feature, task, bug, change-card, tech-debt, or idea",
	Long: `Delete an entity by key. The entity type is auto-detected from the key format.

Key format detection:
  E##                        Epic
  E##-F## or F##             Feature
  E##-F##-### or T-E##-F##-### Task
  B###                       Bug
  CC-###                     Change-card
  TD-###                     Tech-debt
  I-YYYY-MM-DD-##            Idea

Examples:
  shark delete E07                    Delete epic E07
  shark delete E07-F01                Delete feature E07-F01
  shark delete E07-F01-001            Delete task E07-F01-001
  shark delete B001                   Delete bug B001
  shark delete CC-001                 Delete change-card CC-001
  shark delete TD-001                 Delete tech-debt TD-001
  shark delete I-2026-01-15-01        Delete idea I-2026-01-15-01
  shark delete E07-F01 --force        Force delete (cascade children)`,
	GroupID: "manage",
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func init() {
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
	case "bug":
		return runBugDelete(cmd, args)
	case "change":
		return runChangeCardDelete(cmd, args)
	case "change_card":
		return runChangeCardDelete(cmd, args)
	case "tech_debt":
		return runTdDelete(cmd, args)
	case "idea":
		return runIdeaDelete(cmd, args)
	case "sprint":
		return runSprintDelete(cmd, args)
	default:
		return fmt.Errorf("cannot determine entity type from key: %s\nExpected format: E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), CC-### (change card, C###/CC### aliases accepted), TD-### (tech-debt), I-YYYY-MM-DD-## (idea), or S### (sprint)", key)
	}
}
