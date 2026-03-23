package models

import (
	"fmt"
	"time"
)

// EntityRelationshipType represents the type of a polymorphic entity relationship.
// These types carry over from task_relationships and gain new semantics
// applicable across entity type combinations.
type EntityRelationshipType string

const (
	EntityRelDependsOn   EntityRelationshipType = "depends_on"   // Cannot start until target completes
	EntityRelBlocks      EntityRelationshipType = "blocks"       // Prevents target from starting
	EntityRelRelatedTo   EntityRelationshipType = "related_to"   // Informational link
	EntityRelFollows     EntityRelationshipType = "follows"      // Should be done after target
	EntityRelSpawnedFrom EntityRelationshipType = "spawned_from" // Created as result of target
	EntityRelDuplicates  EntityRelationshipType = "duplicates"   // Same work as target
	EntityRelReferences  EntityRelationshipType = "references"   // Mentions or uses target output
	EntityRelLinkedTo    EntityRelationshipType = "linked_to"    // Informal context link (bug pattern)
)

// Backward-compatible untyped constants (used by existing code).
const (
	RelDependsOn   = "depends_on"
	RelBlocks      = "blocks"
	RelRelatedTo   = "related_to"
	RelFollows     = "follows"
	RelSpawnedFrom = "spawned_from"
	RelDuplicates  = "duplicates"
	RelReferences  = "references"
	RelLinkedTo    = "linked_to"
)

// ValidEntityRelationshipTypeSet is the set of all valid relationship types.
var ValidEntityRelationshipTypeSet = map[EntityRelationshipType]bool{
	EntityRelDependsOn:   true,
	EntityRelBlocks:      true,
	EntityRelRelatedTo:   true,
	EntityRelFollows:     true,
	EntityRelSpawnedFrom: true,
	EntityRelDuplicates:  true,
	EntityRelReferences:  true,
	EntityRelLinkedTo:    true,
}

// CyclicRelationshipTypes are the relationship types for which circular
// chains must be prevented. Cycle detection is enforced across all entity
// type combinations (e.g., task depends_on bug depends_on task is a cycle).
var CyclicRelationshipTypes = map[EntityRelationshipType]bool{
	EntityRelDependsOn: true,
	EntityRelBlocks:    true,
}

// ValidEntityRelationshipTypes returns the set of valid relationship types
// for the polymorphic entity_relationships table.
// Kept for backward compatibility with existing callers.
func ValidEntityRelationshipTypes() []string {
	return []string{
		RelDependsOn, RelBlocks, RelRelatedTo, RelFollows,
		RelSpawnedFrom, RelDuplicates, RelReferences, RelLinkedTo,
	}
}

// EntityRelationship represents a typed link between any two entities.
type EntityRelationship struct {
	ID               int64                  `json:"id" db:"id"`
	FromEntityType   EntityType             `json:"from_entity_type" db:"from_entity_type"`
	FromEntityID     int64                  `json:"from_entity_id" db:"from_entity_id"`
	ToEntityType     EntityType             `json:"to_entity_type" db:"to_entity_type"`
	ToEntityID       int64                  `json:"to_entity_id" db:"to_entity_id"`
	RelationshipType EntityRelationshipType `json:"relationship_type" db:"relationship_type"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
}

// Validate performs structural validation on the EntityRelationship.
// Business rules (cycle detection, entity existence) are enforced at the service layer.
func (er *EntityRelationship) Validate() error {
	if er.FromEntityID == 0 {
		return fmt.Errorf("from_entity_id must not be zero")
	}
	if er.ToEntityID == 0 {
		return fmt.Errorf("to_entity_id must not be zero")
	}
	if !ValidEntityTypes[er.FromEntityType] {
		return fmt.Errorf("invalid from_entity_type: %s", er.FromEntityType)
	}
	if !ValidEntityTypes[er.ToEntityType] {
		return fmt.Errorf("invalid to_entity_type: %s", er.ToEntityType)
	}
	if !ValidEntityRelationshipTypeSet[er.RelationshipType] {
		return fmt.Errorf("invalid relationship_type: %s", er.RelationshipType)
	}
	if er.FromEntityType == er.ToEntityType && er.FromEntityID == er.ToEntityID {
		return fmt.Errorf("entity cannot have a relationship with itself")
	}
	return nil
}

// IsCyclic reports whether this relationship type requires cycle detection.
// Cycle detection is enforced regardless of entity type combinations.
func (er *EntityRelationship) IsCyclic() bool {
	return CyclicRelationshipTypes[er.RelationshipType]
}
