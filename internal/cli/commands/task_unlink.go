package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// taskUnlinkCmd removes typed relationships between tasks via the
// entity_relationships table.
var taskUnlinkCmd = &cobra.Command{
	Use:   "unlink <task-key>",
	Short: "Remove typed relationships between tasks",
	Long: `Remove typed relationships between tasks.

Uses the unified entity_relationships table. For cross-entity relationships,
use 'shark unlink' instead.

Relationship Types:
  depends_on, blocks, related_to, follows, spawned_from, duplicates, references

Examples:
  shark task unlink T-E10-F03-004 --depends-on T-E10-F03-003

  # Remove multiple relationships
  shark task unlink T-E10-F03-004 --depends-on T-E10-F03-003,T-E10-F03-001

  # JSON output
  shark task unlink T-E10-F03-004 --depends-on T-E10-F03-003 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUnlink,
}

func init() {
	taskUnlinkCmd.Flags().String("depends-on", "", "Remove depends_on relationships (comma-separated task keys)")
	taskUnlinkCmd.Flags().String("blocks", "", "Remove blocks relationships (comma-separated task keys)")
	taskUnlinkCmd.Flags().String("related-to", "", "Remove related_to relationships (comma-separated task keys)")
	taskUnlinkCmd.Flags().String("follows", "", "Remove follows relationships (comma-separated task keys)")
	taskUnlinkCmd.Flags().String("spawned-from", "", "Remove spawned_from relationships (comma-separated task keys)")
	taskUnlinkCmd.Flags().String("duplicates", "", "Remove duplicates relationships (comma-separated task keys)")
	taskUnlinkCmd.Flags().String("references", "", "Remove references relationships (comma-separated task keys)")

	taskCmd.AddCommand(taskUnlinkCmd)
}

// parseUnlinkInput parses flag inputs into (relType, targetKeys) pairs for processing
func parseUnlinkInput(cmd *cobra.Command) (map[string][]string, error) {
	// Map of flag name -> relationship type
	flagToRelType := map[string]string{
		"depends-on":   "depends_on",
		"blocks":       "blocks",
		"related-to":   "related_to",
		"follows":      "follows",
		"spawned-from": "spawned_from",
		"duplicates":   "duplicates",
		"references":   "references",
	}

	relationships := make(map[string][]string)
	for flagName, relType := range flagToRelType {
		val := cmd.Flag(flagName).Value.String()
		if val == "" {
			continue
		}
		var keys []string
		for _, k := range strings.Split(val, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) > 0 {
			relationships[relType] = keys
		}
	}

	if len(relationships) == 0 {
		return nil, fmt.Errorf("at least one relationship flag required (--depends-on, --blocks, etc.)")
	}

	return relationships, nil
}

// runTaskUnlink handles the task unlink command
func runTaskUnlink(cmd *cobra.Command, args []string) error {
	taskKey := args[0]

	// Step 1: Parse arguments
	relationships, err := parseUnlinkInput(cmd)
	if err != nil {
		return err
	}

	// Resolve the source task key to its database ID
	fromType, fromID, err := resolveEntityKeyToTypeAndID(cmd, taskKey)
	if err != nil {
		return fmt.Errorf("failed to resolve task key %s: %w", taskKey, err)
	}
	if fromType != models.EntityTypeTask {
		return fmt.Errorf("key %s is not a task key", taskKey)
	}

	// Step 2: Call EntityRelationshipService for each relationship
	svc := cli.GetEntityRelationshipService()
	ctx := cmd.Context()
	totalRemoved := 0

	for relType, targetKeys := range relationships {
		for _, targetKey := range targetKeys {
			// Resolve target task key to ID
			toType, toID, resolveErr := resolveEntityKeyToTypeAndID(cmd, targetKey)
			if resolveErr != nil {
				cli.Warning(fmt.Sprintf("Failed to resolve target key %s: %v", targetKey, resolveErr))
				continue
			}

			err := svc.UnlinkEntities(ctx, fromType, fromID, toType, toID, models.EntityRelationshipType(relType))
			if err != nil {
				cli.Warning(fmt.Sprintf("Failed to remove %s relationship to %s: %v", relType, targetKey, err))
				continue
			}
			totalRemoved++
		}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key":      taskKey,
			"removed_count": totalRemoved,
		})
	}
	if totalRemoved == 0 {
		cli.Warning(fmt.Sprintf("No relationships removed for %s", taskKey))
	} else {
		cli.Success(fmt.Sprintf("Removed %d relationship(s) for %s", totalRemoved, taskKey))
	}
	return nil
}
