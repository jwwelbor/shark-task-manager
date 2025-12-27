package models

import (
	"time"
)

// RelationshipType represents the type of relationship between tasks
type RelationshipType string

const (
	RelationshipDependsOn   RelationshipType = "depends_on"   // Task depends on another completing (hard dependency)
	RelationshipBlocks      RelationshipType = "blocks"       // Task blocks another from proceeding
	RelationshipRelatedTo   RelationshipType = "related_to"   // Tasks share common code/concerns
	RelationshipFollows     RelationshipType = "follows"      // Task naturally follows another (soft ordering)
	RelationshipSpawnedFrom RelationshipType = "spawned_from" // Task was created from UAT/bugs in another
	RelationshipDuplicates  RelationshipType = "duplicates"   // Tasks represent duplicate work
	RelationshipReferences  RelationshipType = "references"   // Task consults/uses output of another
)

// ValidRelationshipTypes returns all valid relationship types
func ValidRelationshipTypes() []RelationshipType {
	return []RelationshipType{
		RelationshipDependsOn,
		RelationshipBlocks,
		RelationshipRelatedTo,
		RelationshipFollows,
		RelationshipSpawnedFrom,
		RelationshipDuplicates,
		RelationshipReferences,
	}
}

// TaskRelationship represents a typed relationship between two tasks
type TaskRelationship struct {
	ID               int64            `json:"id" db:"id"`
	FromTaskID       int64            `json:"from_task_id" db:"from_task_id"`
	ToTaskID         int64            `json:"to_task_id" db:"to_task_id"`
	RelationshipType RelationshipType `json:"relationship_type" db:"relationship_type"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// Validate validates the TaskRelationship fields
func (tr *TaskRelationship) Validate() error {
	if tr.FromTaskID == 0 {
		return ErrInvalidTaskID
	}
	if tr.ToTaskID == 0 {
		return ErrInvalidTaskID
	}
	if tr.FromTaskID == tr.ToTaskID {
		return ErrSelfRelationship
	}
	if err := ValidateRelationshipType(string(tr.RelationshipType)); err != nil {
		return err
	}
	return nil
}
