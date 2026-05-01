package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <KEY> [flags]",
	Short: "Update an epic, feature, task, bug, change, tech-debt, or idea",
	Long: `Update an entity by key. The entity type is auto-detected from the key format.

Key format detection:
  E##                        Epic
  E##-F## or F##             Feature
  E##-F##-### or T-E##-F##-### Task
  B###                       Bug
  C###                       Change card
  TD-###                     Tech-debt
  I-YYYY-MM-DD-##            Idea

Use 'shark status set' to change entity status.

Common flags (all entities):
  --title          New title
  --description    New description
  --order          New execution order
  --key            Rename entity key
  --file           New file path
  --force          Force file reassignment
  --size           New size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove)

Priority & agent flags:
  --priority       New priority (1-10 for tasks; low/medium/high for epics)
  --agent          New agent type (task only)
  --depends-on     New dependency keys, comma-separated (task only)

Epic-specific flags:
  --business-value New business value: low, medium, high

Examples:
  shark update E07 --title="New Epic Title"
  shark update E07 --business-value=high
  shark update E07-F01 --title="New Feature Title" --file=docs/custom/feature.md
  shark update E07-F01 --key=E07-F02
  shark update E07-F01-001 --title="New Task Title" --priority=8
  shark update E07-F01-001 --agent=backend
  shark update E07-F01-001 --size=L
  shark update TD-001 --size=clear`,
	GroupID: "manage",
	Args:    cobra.ExactArgs(1),
	RunE:    runUpdate,
}

func init() {
	// Common flags (all entities)
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().StringP("description", "d", "", "New description")
	updateCmd.Flags().Int("order", -1, "New execution order (-1=no change)")

	// Key rename
	updateCmd.Flags().String("key", "", "New key (must be unique, no spaces)")

	// File path flags
	updateCmd.Flags().String("file", "", "New file path (e.g., docs/custom/feature.md)")
	updateCmd.Flags().String("filename", "", "Alias for --file")
	updateCmd.Flags().String("path", "", "Alias for --file")
	_ = updateCmd.Flags().MarkHidden("filename")
	_ = updateCmd.Flags().MarkHidden("path")
	updateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")

	// Priority (string: "1"-"10" for tasks, "low"/"medium"/"high" for epics)
	updateCmd.Flags().StringP("priority", "p", "", "New priority (1-10 for tasks; low/medium/high for epics)")

	// Task-specific flags
	updateCmd.Flags().StringP("agent", "a", "", "New agent type (task only)")
	updateCmd.Flags().String("depends-on", "", "New dependency keys, comma-separated (task only)")

	// Epic-specific flags
	updateCmd.Flags().String("business-value", "", "New business value: low, medium, high (epic only)")

	// Size flag (all entities) — string form per Decision D4: accepts numeric (5)
	// or t-shirt label (L) and the literal "clear" sentinel for removal.
	updateCmd.Flags().String("size", "",
		"New size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove)")

	// Deprecated aliases
	updateCmd.Flags().Int("execution-order", -1, "New execution order (-1=no change)")
	_ = updateCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
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
	case "bug":
		return runBugUpdate(cmd, args)
	case "change", "change_card":
		return runChangeUpdate(cmd, args)
	case "tech_debt":
		return runTdUpdate(cmd, args)
	case "idea":
		return runIdeaUpdate(cmd, args)
	default:
		return fmt.Errorf("cannot determine entity type from key: %s\nExpected format: E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), C### (change card), TD-### (tech-debt), or I-YYYY-MM-DD-## (idea)", key)
	}
}
