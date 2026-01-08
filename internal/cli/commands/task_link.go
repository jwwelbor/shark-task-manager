package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// taskLinkCmd creates typed relationships between tasks
var taskLinkCmd = &cobra.Command{
	Use:   "link <task-key>",
	Short: "Create typed relationships between tasks",
	Long: `Create typed relationships between tasks to track dependencies, blockers, and related work.

Relationship Types:
  depends_on    - Task depends on another completing (hard dependency)
  blocks        - Task blocks another from proceeding
  related_to    - Tasks share common code/concerns
  follows       - Task naturally follows another (soft ordering)
  spawned_from  - Task was created from UAT/bugs in another
  duplicates    - Tasks represent duplicate work
  references    - Task consults/uses output of another

Examples:
  # Single dependency
  shark task link T-E10-F03-004 --depends-on T-E10-F03-003

  # Multiple dependencies
  shark task link T-E10-F03-004 --depends-on T-E10-F03-003,T-E10-F03-001

  # Multiple relationship types
  shark task link T-E10-F03-004 --depends-on T-E10-F03-003 --related-to T-E10-F03-002

  # Spawned task from UAT findings
  shark task link T-E10-F03-008 --spawned-from T-E10-F03-002

  # JSON output
  shark task link T-E10-F03-004 --depends-on T-E10-F03-003 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskLink,
}

func init() {
	taskLinkCmd.Flags().String("depends-on", "", "Create depends_on relationships (comma-separated task keys)")
	taskLinkCmd.Flags().String("blocks", "", "Create blocks relationships (comma-separated task keys)")
	taskLinkCmd.Flags().String("related-to", "", "Create related_to relationships (comma-separated task keys)")
	taskLinkCmd.Flags().String("follows", "", "Create follows relationships (comma-separated task keys)")
	taskLinkCmd.Flags().String("spawned-from", "", "Create spawned_from relationships (comma-separated task keys)")
	taskLinkCmd.Flags().String("duplicates", "", "Create duplicates relationships (comma-separated task keys)")
	taskLinkCmd.Flags().String("references", "", "Create references relationships (comma-separated task keys)")

	taskCmd.AddCommand(taskLinkCmd)
}

// runTaskLink handles the task link command
func runTaskLink(cmd *cobra.Command, args []string) error {
	taskKey := args[0]

	// Get all relationship flags
	relationships := map[string]string{
		"depends_on":   cmd.Flag("depends-on").Value.String(),
		"blocks":       cmd.Flag("blocks").Value.String(),
		"related_to":   cmd.Flag("related-to").Value.String(),
		"follows":      cmd.Flag("follows").Value.String(),
		"spawned_from": cmd.Flag("spawned-from").Value.String(),
		"duplicates":   cmd.Flag("duplicates").Value.String(),
		"references":   cmd.Flag("references").Value.String(),
	}

	// Check if at least one relationship flag was provided
	hasRelationships := false
	for _, value := range relationships {
		if value != "" {
			hasRelationships = true
			break
		}
	}

	if !hasRelationships {
		return fmt.Errorf("at least one relationship flag required (--depends-on, --blocks, etc.)")
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	ctx := context.Background()
	dbWrapper := repoDb
	taskRepo := repository.NewTaskRepository(dbWrapper)
	relRepo := repository.NewTaskRelationshipRepository(dbWrapper)

	// Get source task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		cli.Error(fmt.Sprintf("Task %s not found", taskKey))
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Track created relationships for output
	var createdRels []struct {
		Type       string
		TargetKey  string
		TargetTask *models.Task
	}

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
				cli.Error(fmt.Sprintf("Target task %s not found", targetKey))
				return fmt.Errorf("target task %s not found", targetKey)
			}

			// Check for cycle (for depends_on and blocks relationships)
			if relType == "depends_on" || relType == "blocks" {
				if err := relRepo.DetectCycle(ctx, task.ID, targetTask.ID, relType); err != nil {
					cli.Error(fmt.Sprintf("Circular dependency detected: %v", err))
					return err
				}
			}

			// Create relationship
			rel := &models.TaskRelationship{
				FromTaskID:       task.ID,
				ToTaskID:         targetTask.ID,
				RelationshipType: models.RelationshipType(relType),
			}

			err = relRepo.Create(ctx, rel)
			if err != nil {
				if strings.Contains(err.Error(), "relationship already exists") {
					cli.Warning(fmt.Sprintf("Relationship already exists: %s %s %s", taskKey, relType, targetKey))
					continue
				}
				cli.Error(fmt.Sprintf("Failed to create relationship: %v", err))
				return fmt.Errorf("failed to create relationship: %w", err)
			}

			createdRels = append(createdRels, struct {
				Type       string
				TargetKey  string
				TargetTask *models.Task
			}{
				Type:       relType,
				TargetKey:  targetKey,
				TargetTask: targetTask,
			})
		}
	}

	// Output results
	if cli.GlobalConfig.JSON {
		output := map[string]interface{}{
			"task_key":      taskKey,
			"relationships": []map[string]string{},
		}

		relationships := output["relationships"].([]map[string]string)
		for _, rel := range createdRels {
			relationships = append(relationships, map[string]string{
				"type":         rel.Type,
				"target_key":   rel.TargetKey,
				"target_title": rel.TargetTask.Title,
			})
		}
		output["relationships"] = relationships

		return cli.OutputJSON(output)
	}

	// Human-readable output
	cli.Success(fmt.Sprintf("Created %d relationship(s) for %s:", len(createdRels), taskKey))
	for _, rel := range createdRels {
		fmt.Printf("  %s → %s (%s)\n", rel.Type, rel.TargetKey, rel.TargetTask.Title)
	}

	return nil
}
