package models

import (
	"fmt"
	"strings"
	"time"
)

// EntityType represents the type of entity a note is attached to
type EntityType string

const (
	EntityTypeEpic     EntityType = "epic"
	EntityTypeFeature  EntityType = "feature"
	EntityTypeTask     EntityType = "task"
	EntityTypeChange   EntityType = "change"
	EntityTypeBug      EntityType = "bug"
	EntityTypeTechDebt EntityType = "tech_debt"
	EntityTypeIdea     EntityType = "idea"
	// EntityTypeSprint identifies a sprint entity for polymorphic operations
	// such as notes (B030). Added so `shark create note S###` resolves to the
	// sprint repository via the EntityRegistry.
	EntityTypeSprint EntityType = "sprint"
)

// ValidEntityTypes is the set of valid entity types
var ValidEntityTypes = map[EntityType]bool{
	EntityTypeEpic:     true,
	EntityTypeFeature:  true,
	EntityTypeTask:     true,
	EntityTypeChange:   true,
	EntityTypeBug:      true,
	EntityTypeTechDebt: true,
	EntityTypeIdea:     true,
	EntityTypeSprint:   true,
}

// EntityNote represents a typed note attached to any entity (epic, feature, or task)
type EntityNote struct {
	ID         int64      `json:"id" db:"id"`
	EntityType EntityType `json:"entity_type" db:"entity_type"`
	EntityID   int64      `json:"entity_id" db:"entity_id"`
	NoteType   NoteType   `json:"note_type" db:"note_type"`
	Content    string     `json:"content" db:"content"`
	CreatedBy  *string    `json:"created_by,omitempty" db:"created_by"`
	Metadata   *string    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// Validate validates the EntityNote fields
func (n *EntityNote) Validate() error {
	if !ValidEntityTypes[n.EntityType] {
		return fmt.Errorf("invalid entity_type: %s", n.EntityType)
	}
	if n.EntityID <= 0 {
		return fmt.Errorf("entity_id must be positive")
	}
	if err := ValidateNoteType(string(n.NoteType)); err != nil {
		return err
	}
	if strings.TrimSpace(n.Content) == "" {
		return ErrEmptyContent
	}
	return nil
}
