package models

import (
	"time"
)

// EpicRelationship represents a typed relationship between two epics.
//
// LEGACY: EpicRelationship uses the legacy epic_relationships table.
// New code should use EntityRelationship (entity_relationships table) which
// supports polymorphic cross-entity-type linking. This model will be removed
// once all callers are migrated to EntityRelationship.
type EpicRelationship struct {
	ID               int64            `json:"id" db:"id"`
	FromEpicID       int64            `json:"from_epic_id" db:"from_epic_id"`
	ToEpicID         int64            `json:"to_epic_id" db:"to_epic_id"`
	RelationshipType RelationshipType `json:"relationship_type" db:"relationship_type"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// Validate validates the EpicRelationship fields
func (er *EpicRelationship) Validate() error {
	if er.FromEpicID == 0 {
		return ErrInvalidEpicID
	}
	if er.ToEpicID == 0 {
		return ErrInvalidEpicID
	}
	if er.FromEpicID == er.ToEpicID {
		return ErrSelfRelationship
	}
	if err := ValidateRelationshipType(string(er.RelationshipType)); err != nil {
		return err
	}
	return nil
}
