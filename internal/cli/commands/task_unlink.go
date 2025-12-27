package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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

// runTaskUnlink handles the task unlink command
func runTaskUnlink(cmd *cobra.Command, args []string) error {
	taskKey := args[0]

	// Get all relationship flags
	relationships := map[string]string{
		"depends_on":    cmd.Flag("depends-on").Value.String(),
		"blocks":        cmd.Flag("blocks").Value.String(),
		"related_to":    cmd.Flag("related-to").Value.String(),
		"follows":       cmd.Flag("follows").Value.String(),
		"spawned_from":  cmd.Flag("spawned-from").Value.String(),
		"duplicates":    cmd.Flag("duplicates").Value.String(),
		"references":    cmd.Flag("references").Value.String(),
	}

	removeType, _ := cmd.Flags().GetString("type")
	removeAll, _ := cmd.Flags().GetBool("all")

	// Check if at least one relationship flag or --all was provided
	hasRelationships := false
	for _, value := range relationships {
		if value != "" {
			hasRelationships = true
			break
		}
	}

	if !hasRelationships && !removeAll {
		return fmt.Errorf("at least one relationship flag required, or use --type with --all")
	}

	if removeAll && removeType == "" {
		return fmt.Errorf("--all requires --type to be specified")
	}

	// Get database connection
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	ctx := context.Background()
	dbWrapper := repository.NewDB(database)
	taskRepo := repository.NewTaskRepository(dbWrapper)
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)

	// Get source task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Track removed relationships for output
	removedCount := 0

	// Handle --all flag
	if removeAll {
		rels, err := relRepo.GetOutgoing(ctx, task.ID, []string{removeType})
		if err != nil {
			return fmt.Errorf("failed to get relationships: %w", err)
		}

		for _, rel := range rels {
			err := relRepo.Delete(ctx, rel.ID)
			if err != nil {
				cli.Warning(fmt.Sprintf("Failed to remove relationship %d: %v", rel.ID, err))
				continue
			}
			removedCount++
		}
	} else {
		// Process each relationship type
		for relType, targetKeysStr := range relationships {
			if targetKeysStr == "" {
				continue
			}

			targetKeys := strings.Split(targetKeysStr, ",")
			for _, targetKey := range targetKeys {
				targetKey = strings.TrimSpace(targetKey)
				if targetKey == "" {
					continue
				}

				// Get target task
				targetTask, err := taskRepo.GetByKey(ctx, targetKey)
				if err != nil {
					cli.Warning(fmt.Sprintf("Target task %s not found, skipping", targetKey))
					continue
				}

				// Remove relationship
				err = relRepo.DeleteByTasksAndType(ctx, task.ID, targetTask.ID, relType)
				if err != nil {
					if strings.Contains(err.Error(), "not found") {
						cli.Warning(fmt.Sprintf("Relationship not found: %s %s %s", taskKey, relType, targetKey))
						continue
					}
					cli.Error(fmt.Sprintf("Failed to remove relationship: %v", err))
					return fmt.Errorf("failed to remove relationship: %w", err)
				}

				removedCount++
			}
		}
	}

	// Output results
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":       taskKey,
			"removed_count":  removedCount,
		}
		return cli.OutputJSON(output)
	}

	// Human-readable output
	if removedCount == 0 {
		cli.Warning(fmt.Sprintf("No relationships removed for %s", taskKey))
	} else {
		cli.Success(fmt.Sprintf("Removed %d relationship(s) for %s", removedCount, taskKey))
	}

	return nil
}
