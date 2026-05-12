package models

import (
	"fmt"
	"regexp"
	"time"
)

// IdeaStatus represents the status of an idea
type IdeaStatus string

const (
	IdeaStatusNew       IdeaStatus = "new"
	IdeaStatusOnHold    IdeaStatus = "on_hold"
	IdeaStatusConverted IdeaStatus = "converted"
	IdeaStatusArchived  IdeaStatus = "archived"
)

// Idea represents a lightweight idea capture before committing to full epic/feature/task structure
//
// Note: Idea does not embed BaseEntity (it predates the polymorphic refactor).
// The Size field is added directly here rather than via BaseEntity embedding.
// See spec.md Architecture Section 3.8 Decision D8.
type Idea struct {
	ID           int64      `json:"id" db:"id"`
	Key          string     `json:"key" db:"key"` // Format: I-YYYY-MM-DD-xx
	Title        string     `json:"title" db:"title"`
	Description  *string    `json:"description,omitempty" db:"description"`
	CreatedDate  time.Time  `json:"created_date" db:"created_date"`
	Priority     *int       `json:"priority,omitempty" db:"priority"`         // 1-10 scale
	Order        *int       `json:"order,omitempty" db:"order"`               // For ordering ideas
	Notes        *string    `json:"notes,omitempty" db:"notes"`               // Additional notes
	RelatedDocs  *string    `json:"related_docs,omitempty" db:"related_docs"` // JSON array of document paths
	Dependencies *string    `json:"dependencies,omitempty" db:"dependencies"` // JSON array of idea keys
	Status       IdeaStatus `json:"status" db:"status"`
	// Size holds the idea size as a canonical Fibonacci integer
	// (1=XS, 2=S, 3=M, 5=L, 8=XL, 13=XXL). NULL means "not sized".
	// See models.ValidateSize, models.ParseSize, and models.SizeLabel.
	// Part of E07-F42 (Add size field to all entities).
	Size      *int      `json:"size,omitempty" db:"size"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Conversion tracking (for E08-F03)
	ConvertedToType *string    `json:"converted_to_type,omitempty" db:"converted_to_type"` // "epic", "feature", or "task"
	ConvertedToKey  *string    `json:"converted_to_key,omitempty" db:"converted_to_key"`   // The key of the created entity
	ConvertedAt     *time.Time `json:"converted_at,omitempty" db:"converted_at"`
}

// GetSize returns the idea's Size field (nil if not set).
func (i *Idea) GetSize() *int { return i.Size }

// SetSize sets the idea's Size field. Pass nil to clear.
func (i *Idea) SetSize(s *int) { i.Size = s }

// ----------------------------------------------------------------------------
// Entity interface implementation (B030).
//
// Idea pre-dates the polymorphic BaseEntity refactor and does not embed
// BaseEntity. The methods below adapt the Idea fields to the shared
// models.Entity interface so cross-cutting services (NoteService,
// EntityRegistry, etc.) can treat ideas polymorphically.
//
// Fields the ideas table does not carry (Slug, FilePath, ContextData)
// degrade to empty strings / nil; their setters are no-ops.
// ----------------------------------------------------------------------------

// GetID returns the idea database ID.
func (i *Idea) GetID() int64 { return i.ID }

// GetKey returns the idea key (e.g., "I-2026-05-11-01").
func (i *Idea) GetKey() string { return i.Key }

// GetTitle returns the idea title.
func (i *Idea) GetTitle() string { return i.Title }

// GetSlug returns an empty string; ideas have no slug column.
func (i *Idea) GetSlug() string { return "" }

// GetEntityType returns EntityTypeIdea.
func (i *Idea) GetEntityType() EntityType { return EntityTypeIdea }

// GetStatus returns the idea status as a string.
func (i *Idea) GetStatus() string { return string(i.Status) }

// SetStatus updates the idea status from a string value.
func (i *Idea) SetStatus(status string) { i.Status = IdeaStatus(status) }

// GetDescription returns the idea description, or "" if not set.
func (i *Idea) GetDescription() string {
	if i.Description != nil {
		return *i.Description
	}
	return ""
}

// GetFilePath returns an empty string; ideas have no file_path column.
func (i *Idea) GetFilePath() string { return "" }

// GetContextData returns nil; ideas do not carry context_data.
func (i *Idea) GetContextData() *string { return nil }

// SetContextData is a no-op; ideas do not carry context_data.
func (i *Idea) SetContextData(_ *string) {}

// GetCreatedAt returns the idea creation timestamp.
func (i *Idea) GetCreatedAt() time.Time { return i.CreatedAt }

// GetUpdatedAt returns the idea last-updated timestamp.
func (i *Idea) GetUpdatedAt() time.Time { return i.UpdatedAt }

// Validate validates the Idea fields
func (i *Idea) Validate() error {
	// Validate key format
	if err := ValidateIdeaKey(i.Key); err != nil {
		return err
	}

	// Validate title
	if i.Title == "" {
		return ErrEmptyTitle
	}

	// Validate status
	if err := ValidateIdeaStatus(string(i.Status)); err != nil {
		return err
	}

	// Validate priority if provided
	if i.Priority != nil {
		if *i.Priority < 1 || *i.Priority > 10 {
			return ErrInvalidPriority
		}
	}

	// Validate dependencies if provided (should be valid JSON array)
	if i.Dependencies != nil {
		if err := ValidateJSONArray(*i.Dependencies); err != nil {
			return fmt.Errorf("invalid dependencies JSON: %w", err)
		}
	}

	// Validate related docs if provided (should be valid JSON array)
	if i.RelatedDocs != nil {
		if err := ValidateJSONArray(*i.RelatedDocs); err != nil {
			return fmt.Errorf("invalid related_docs JSON: %w", err)
		}
	}

	// Validate size if set (E07-F42: canonical Fibonacci values only).
	if i.Size != nil {
		if err := ValidateSize(*i.Size); err != nil {
			return err
		}
	}

	return nil
}

// ValidateIdeaKey validates the idea key format (I-YYYY-MM-DD-xx)
func ValidateIdeaKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	// Pattern: I-YYYY-MM-DD-xx where xx is 01-99
	pattern := `^I-\d{4}-\d{2}-\d{2}-\d{2}$`
	matched, err := regexp.MatchString(pattern, key)
	if err != nil {
		return fmt.Errorf("error validating idea key pattern: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid idea key format %q: must match I-YYYY-MM-DD-xx (e.g., I-2026-01-01-01)", key)
	}

	return nil
}

// ValidateIdeaStatus validates the idea status enum
func ValidateIdeaStatus(status string) error {
	validStatuses := map[string]bool{
		"new":       true,
		"on_hold":   true,
		"converted": true,
		"archived":  true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid idea status %q: must be one of new, on_hold, converted, archived", status)
	}

	return nil
}
