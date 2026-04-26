package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// tagNamePattern is the compiled allowlist regex for tag names per ADR-4.
// Tag names match ^[a-z0-9][a-z0-9-]{0,63}$ — lowercase ASCII, digits, hyphens only;
// must start with a letter or digit; max 64 characters total.
// Input is lowercased via strings.ToLower before validation.
var tagNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// Tag represents a named tag in the managed vocabulary.
// The name is always stored as a lowercase ASCII slug (see ADR-4).
type Tag struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// EntityTagLink represents a polymorphic join between a tag and an entity
// in the entity_tags table.
type EntityTagLink struct {
	ID         int64      `json:"id" db:"id"`
	EntityType EntityType `json:"entity_type" db:"entity_type"`
	EntityID   int64      `json:"entity_id" db:"entity_id"`
	TagID      int64      `json:"tag_id" db:"tag_id"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// ValidateTagName validates a tag name according to the allowlist regex from ADR-4.
// It lowercases the input before validation, so "VOICE" passes and is treated as "voice".
// Returns a descriptive error for non-matching input.
func ValidateTagName(name string) error {
	lowered := strings.ToLower(name)
	if !tagNamePattern.MatchString(lowered) {
		return fmt.Errorf("tag name %q is invalid: must match ^[a-z0-9][a-z0-9-]{0,63}$ (lowercase ASCII letters, digits, hyphens; must start with letter or digit; max 64 characters)", name)
	}
	return nil
}
