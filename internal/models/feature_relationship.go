package models

import (
	"time"
)

// FeatureRelationship represents a typed relationship between two features.
//
// LEGACY: FeatureRelationship uses the legacy feature_relationships table.
// New code should use EntityRelationship (entity_relationships table) which
// supports polymorphic cross-entity-type linking. This model will be removed
// once all callers are migrated to EntityRelationship.
type FeatureRelationship struct {
	ID               int64            `json:"id" db:"id"`
	FromFeatureID    int64            `json:"from_feature_id" db:"from_feature_id"`
	ToFeatureID      int64            `json:"to_feature_id" db:"to_feature_id"`
	RelationshipType RelationshipType `json:"relationship_type" db:"relationship_type"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// Validate validates the FeatureRelationship fields
func (fr *FeatureRelationship) Validate() error {
	if fr.FromFeatureID == 0 {
		return ErrInvalidFeatureID
	}
	if fr.ToFeatureID == 0 {
		return ErrInvalidFeatureID
	}
	if fr.FromFeatureID == fr.ToFeatureID {
		return ErrSelfRelationship
	}
	if err := ValidateRelationshipType(string(fr.RelationshipType)); err != nil {
		return err
	}
	return nil
}
