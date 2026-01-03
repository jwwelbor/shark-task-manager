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
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`

	// Conversion tracking (for E08-F03)
	ConvertedToType *string    `json:"converted_to_type,omitempty" db:"converted_to_type"` // "epic", "feature", or "task"
	ConvertedToKey  *string    `json:"converted_to_key,omitempty" db:"converted_to_key"`   // The key of the created entity
	ConvertedAt     *time.Time `json:"converted_at,omitempty" db:"converted_at"`
}

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
