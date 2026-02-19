package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// taskUnlinkCmd removes typed relationships between tasks
var taskUnlinkCmd = &cobra.Command{
	Use:   "unlink <task-key>",
	Short: "Remove typed relationships between tasks",
	Long: `Remove typed relationships between tasks.

Relationship Types:
  depends_on, blocks, related_to, follows, spawned_from, duplicates, references

Examples:
  # Remove specific dependency
  shark task unlink T-E10-F03-004 --depends-on T-E10-F03-003

  # Remove multiple relationships
  shark task unlink T-E10-F03-004 --depends-on T-E10-F03-003,T-E10-F03-001

  # Remove all relationships of a type
  shark task unlink T-E10-F03-004 --type depends_on --all

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
	taskUnlinkCmd.Flags().String("type", "", "Relationship type to remove (use with --all)")
	taskUnlinkCmd.Flags().Bool("all", false, "Remove all relationships of the specified type")

	taskCmd.AddCommand(taskUnlinkCmd)
}

// parseUnlinkInput parses flag inputs into (relType, targetKeys) pairs for processing
func parseUnlinkInput(cmd *cobra.Command) (map[string][]string, string, bool, error) {
	// Map of flag name → relationship type
	flagToRelType := map[string]string{
		"depends-on":   "depends_on",
		"blocks":       "blocks",
		"related-to":   "related_to",
		"follows":      "follows",
		"spawned-from": "spawned_from",
		"duplicates":   "duplicates",
		"references":   "references",
	}

	removeType, _ := cmd.Flags().GetString("type")
	removeAll, _ := cmd.Flags().GetBool("all")

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

	hasRelationships := len(relationships) > 0

	if !hasRelationships && !removeAll {
		return nil, "", false, fmt.Errorf("at least one relationship flag required, or use --type with --all")
	}

	if removeAll && removeType == "" {
		return nil, "", false, fmt.Errorf("--all requires --type to be specified")
	}

	return relationships, removeType, removeAll, nil
}

// runTaskUnlink handles the task unlink command
func runTaskUnlink(cmd *cobra.Command, args []string) error {
	taskKey := args[0]

	// Step 1: Parse arguments
	relationships, removeType, removeAll, err := parseUnlinkInput(cmd)
	if err != nil {
		return err
	}

	// Step 2: Call service
	totalRemoved, err := performUnlink(cmd, taskKey, relationships, removeType, removeAll)
	if err != nil {
		return err
	}

	// Step 3: Format output
	return printUnlinkResult(taskKey, totalRemoved)
}

// performUnlink calls the service to remove relationships and returns the total count removed.
func performUnlink(cmd *cobra.Command, taskKey string, relationships map[string][]string, removeType string, removeAll bool) (int, error) {
	svc := cli.GetTaskServiceWithDeps()
	totalRemoved := 0
	if removeAll {
		count, err := svc.UnlinkRelationships(cmd.Context(), taskKey, removeType, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to unlink relationships: %w", err)
		}
		totalRemoved += count
	} else {
		for relType, targetKeys := range relationships {
			count, err := svc.UnlinkRelationships(cmd.Context(), taskKey, relType, targetKeys)
			if err != nil {
				cli.Warning(fmt.Sprintf("Failed to remove %s relationships: %v", relType, err))
				continue
			}
			totalRemoved += count
		}
	}
	return totalRemoved, nil
}

// printUnlinkResult prints the human-readable result of an unlink operation.
func printUnlinkResult(taskKey string, totalRemoved int) error {
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"task_key": taskKey, "removed_count": totalRemoved,
		})
	}
	if totalRemoved == 0 {
		cli.Warning(fmt.Sprintf("No relationships removed for %s", taskKey))
	} else {
		cli.Success(fmt.Sprintf("Removed %d relationship(s) for %s", totalRemoved, taskKey))
	}
	return nil
}
