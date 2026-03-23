package commands

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// taskLinkCmd creates typed relationships between tasks via the
// entity_relationships table.
var taskLinkCmd = &cobra.Command{
	Use:   "link <task-key>",
	Short: "Create typed relationships between tasks",
	Long: `Create typed relationships between tasks to track dependencies, blockers, and related work.

Uses the unified entity_relationships table. For cross-entity relationships
(task-to-epic, bug-to-feature, etc.), use 'shark link' instead.

Relationship Types:
  depends_on    - Task depends on another completing (hard dependency)
  blocks        - Task blocks another from proceeding
  related_to    - Tasks share common code/concerns
  follows       - Task naturally follows another (soft ordering)
  spawned_from  - Task was created from UAT/bugs in another
  duplicates    - Tasks represent duplicate work
  references    - Task consults/uses output of another

Examples:
  shark task link T-E10-F03-004 --depends-on T-E10-F03-003

  # Multiple dependencies
  shark task link T-E10-F03-004 --depends-on T-E10-F03-003,T-E10-F03-001

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
	// Step 1: Parse arguments
	taskKey := args[0]

	// Map CLI flag names to relationship type strings
	relationships := map[string]string{
		"depends_on":   cmd.Flag("depends-on").Value.String(),
		"blocks":       cmd.Flag("blocks").Value.String(),
		"related_to":   cmd.Flag("related-to").Value.String(),
		"follows":      cmd.Flag("follows").Value.String(),
		"spawned_from": cmd.Flag("spawned-from").Value.String(),
		"duplicates":   cmd.Flag("duplicates").Value.String(),
		"references":   cmd.Flag("references").Value.String(),
	}

	// Validate that at least one relationship flag was provided
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

	type createdRel struct {
		Type      string
		TargetKey string
	}
	var createdRels []createdRel

	for relType, targetKeysStr := range relationships {
		if targetKeysStr == "" {
			continue
		}

		for _, targetKey := range strings.Split(targetKeysStr, ",") {
			targetKey = strings.TrimSpace(targetKey)
			if targetKey == "" {
				continue
			}

			// Resolve target task key to ID
			toType, toID, resolveErr := resolveEntityKeyToTypeAndID(cmd, targetKey)
			if resolveErr != nil {
				cli.Error(fmt.Sprintf("Failed to resolve target key %s: %v", targetKey, resolveErr))
				return fmt.Errorf("failed to resolve target key %s: %w", targetKey, resolveErr)
			}

			_, err := svc.CreateRelationship(ctx, fromType, fromID, toType, toID, models.EntityRelationshipType(relType))
			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "already exists") {
					cli.Warning(fmt.Sprintf("Relationship already exists: %s %s %s", taskKey, relType, targetKey))
					continue
				}
				if strings.Contains(err.Error(), "cycle") {
					cli.Error(fmt.Sprintf("Circular dependency detected: %v", err))
					return err
				}
				cli.Error(fmt.Sprintf("Failed to create relationship: %v", err))
				return fmt.Errorf("failed to create relationship: %w", err)
			}

			createdRels = append(createdRels, createdRel{
				Type:      relType,
				TargetKey: targetKey,
			})
		}
	}

	// Step 3: Format output
	if cli.GlobalConfig.JSON {
		relMaps := make([]map[string]string, 0, len(createdRels))
		for _, rel := range createdRels {
			relMaps = append(relMaps, map[string]string{
				"type":       rel.Type,
				"target_key": rel.TargetKey,
			})
		}
		return cli.OutputJSON(map[string]interface{}{
			"task_key":      taskKey,
			"relationships": relMaps,
		})
	}

	cli.Success(fmt.Sprintf("Created %d relationship(s) for %s:", len(createdRels), taskKey))
	for _, rel := range createdRels {
		fmt.Printf("  %s -> %s\n", rel.Type, rel.TargetKey)
	}

	return nil
}
