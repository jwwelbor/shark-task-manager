package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

var linkRelType string

var linkCmd = &cobra.Command{
	Use:   "link <from-key> <to-key>",
	Short: "Create a relationship between two entities",
	Long: `Create a typed relationship between any two entities.

Entity type is auto-detected from the key format:
  E##            -> epic
  E##-F##        -> feature
  E##-F##-###    -> task
  B###           -> bug
  CC-###         -> change

Relationship types: depends_on, blocks, related_to, follows,
  spawned_from, duplicates, references, linked_to

Examples:
  shark link E07-F01-001 E07-F01-002 --type=depends_on
  shark link B001 E07-F01-003 --type=related_to
  shark link E07-F01 E07-F02 --type=follows`,
	Args: cobra.ExactArgs(2),
	RunE: runLink,
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <from-key> <to-key>",
	Short: "Remove a relationship between two entities",
	Long: `Remove a typed relationship between two entities.

Examples:
  shark unlink E07-F01-001 E07-F01-002 --type=depends_on
  shark unlink B001 E07-F01-003 --type=related_to`,
	Args: cobra.ExactArgs(2),
	RunE: runUnlink,
}

var linksCmd = &cobra.Command{
	Use:   "links <key>",
	Short: "List all relationships for an entity",
	Long: `List all incoming and outgoing relationships for an entity.

Examples:
  shark links E07-F01-001
  shark links B001
  shark links E07-F01`,
	Args: cobra.ExactArgs(1),
	RunE: runLinks,
}

func init() {
	linkCmd.Flags().StringVar(&linkRelType, "type", "", "Relationship type (required): depends_on, blocks, related_to, follows, spawned_from, duplicates, references, linked_to")
	if err := linkCmd.MarkFlagRequired("type"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}

	unlinkCmd.Flags().StringVar(&linkRelType, "type", "", "Relationship type (required)")
	if err := unlinkCmd.MarkFlagRequired("type"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}

	cli.RootCmd.AddCommand(linkCmd)
	cli.RootCmd.AddCommand(unlinkCmd)
	cli.RootCmd.AddCommand(linksCmd)
}

// resolveEntityKeyToTypeAndID resolves an entity key string to its EntityType and database ID.
// Uses DetectEntityType for key format detection and the EntityRegistry for lookup.
func resolveEntityKeyToTypeAndID(cmd *cobra.Command, key string) (models.EntityType, int64, error) {
	normalized := NormalizeKey(key)
	detected := DetectEntityType(normalized)

	entityType, err := mapDetectedTypeToEntityType(detected)
	if err != nil {
		return "", 0, fmt.Errorf("cannot determine entity type for key %q: %w", key, err)
	}

	registry := cli.GetEntityRegistry()
	repo, err := registry.GetRepository(entityType)
	if err != nil {
		return "", 0, fmt.Errorf("no repository for entity type %q: %w", entityType, err)
	}

	entity, err := repo.GetByKey(cmd.Context(), normalized)
	if err != nil {
		return "", 0, fmt.Errorf("entity not found: %s: %w", key, err)
	}

	return entityType, entity.GetID(), nil
}

// mapDetectedTypeToEntityType converts the string returned by DetectEntityType
// to a models.EntityType value.
func mapDetectedTypeToEntityType(detected string) (models.EntityType, error) {
	switch detected {
	case "epic":
		return models.EntityTypeEpic, nil
	case "feature":
		return models.EntityTypeFeature, nil
	case "task":
		return models.EntityTypeTask, nil
	case "bug":
		return models.EntityTypeBug, nil
	case "change", "change_card":
		return models.EntityTypeChange, nil
	case "tech_debt":
		return models.EntityTypeTechDebt, nil
	default:
		return "", fmt.Errorf("unrecognized entity type: %s", detected)
	}
}

func runLink(cmd *cobra.Command, args []string) error {
	fromKey := args[0]
	toKey := args[1]
	relType := models.EntityRelationshipType(linkRelType)

	// Validate relationship type
	if !models.ValidEntityRelationshipTypeSet[relType] {
		validTypes := models.ValidEntityRelationshipTypes()
		return fmt.Errorf("invalid relationship type %q; valid types: %s", linkRelType, strings.Join(validTypes, ", "))
	}

	// Resolve both entity keys
	fromType, fromID, err := resolveEntityKeyToTypeAndID(cmd, fromKey)
	if err != nil {
		return err
	}

	toType, toID, err := resolveEntityKeyToTypeAndID(cmd, toKey)
	if err != nil {
		return err
	}

	// Create relationship via service
	svc := cli.GetEntityRelationshipService()
	rel, err := svc.CreateRelationship(cmd.Context(), fromType, fromID, toType, toID, relType)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(rel)
	}

	cli.Success(fmt.Sprintf("Created %s relationship: %s -> %s", relType, fromKey, toKey))
	return nil
}

func runUnlink(cmd *cobra.Command, args []string) error {
	fromKey := args[0]
	toKey := args[1]
	relType := models.EntityRelationshipType(linkRelType)

	// Validate relationship type
	if !models.ValidEntityRelationshipTypeSet[relType] {
		validTypes := models.ValidEntityRelationshipTypes()
		return fmt.Errorf("invalid relationship type %q; valid types: %s", linkRelType, strings.Join(validTypes, ", "))
	}

	// Resolve both entity keys
	fromType, fromID, err := resolveEntityKeyToTypeAndID(cmd, fromKey)
	if err != nil {
		return err
	}

	toType, toID, err := resolveEntityKeyToTypeAndID(cmd, toKey)
	if err != nil {
		return err
	}

	// Delete relationship via service
	svc := cli.GetEntityRelationshipService()
	if err := svc.UnlinkEntities(cmd.Context(), fromType, fromID, toType, toID, relType); err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{
			"status": "unlinked",
			"from":   fromKey,
			"to":     toKey,
			"type":   string(relType),
		})
	}

	cli.Success(fmt.Sprintf("Removed %s relationship: %s -> %s", relType, fromKey, toKey))
	return nil
}

// linksOutputEntry represents a single relationship for display/JSON output.
type linksOutputEntry struct {
	Direction        string                        `json:"direction"`
	RelationshipType models.EntityRelationshipType `json:"relationship_type"`
	EntityType       models.EntityType             `json:"entity_type"`
	EntityID         int64                         `json:"entity_id"`
	EntityKey        string                        `json:"entity_key,omitempty"`
}

func runLinks(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Resolve entity key
	entityType, entityID, err := resolveEntityKeyToTypeAndID(cmd, key)
	if err != nil {
		return err
	}

	// Get all relationships via service
	svc := cli.GetEntityRelationshipService()
	rels, err := svc.GetRelationships(cmd.Context(), entityType, entityID)
	if err != nil {
		return err
	}

	if len(rels) == 0 {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON([]linksOutputEntry{})
		}
		cli.Info(fmt.Sprintf("No relationships found for %s", key))
		return nil
	}

	// Build output entries, enriching with entity keys where possible
	registry := cli.GetEntityRegistry()
	entries := make([]linksOutputEntry, 0, len(rels))

	for _, rel := range rels {
		var direction string
		var otherType models.EntityType
		var otherID int64

		if rel.FromEntityType == entityType && rel.FromEntityID == entityID {
			direction = "outgoing"
			otherType = rel.ToEntityType
			otherID = rel.ToEntityID
		} else {
			direction = "incoming"
			otherType = rel.FromEntityType
			otherID = rel.FromEntityID
		}

		// Try to resolve the other entity's key for display
		otherKey := ""
		if repo, repoErr := registry.GetRepository(otherType); repoErr == nil {
			if entity, entityErr := repo.GetByID(cmd.Context(), otherID); entityErr == nil {
				otherKey = entity.GetKey()
			}
		}

		entries = append(entries, linksOutputEntry{
			Direction:        direction,
			RelationshipType: rel.RelationshipType,
			EntityType:       otherType,
			EntityID:         otherID,
			EntityKey:        otherKey,
		})
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(entries)
	}

	// Table output
	headers := []string{"Direction", "Type", "Entity Type", "Key"}
	var rows [][]string
	for _, e := range entries {
		displayKey := e.EntityKey
		if displayKey == "" {
			displayKey = fmt.Sprintf("(id:%d)", e.EntityID)
		}
		rows = append(rows, []string{
			e.Direction,
			string(e.RelationshipType),
			string(e.EntityType),
			displayKey,
		})
	}

	cli.Info(fmt.Sprintf("Relationships for %s:", key))
	cli.OutputTable(headers, rows)
	return nil
}
