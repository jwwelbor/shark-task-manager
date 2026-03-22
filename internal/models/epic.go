package models

// EpicStatus represents the status of an epic
type EpicStatus string

const (
	EpicStatusDraft     EpicStatus = "draft"
	EpicStatusActive    EpicStatus = "active"
	EpicStatusCompleted EpicStatus = "completed"
	EpicStatusArchived  EpicStatus = "archived"
)

// Priority represents priority level (used by Epic and other entities)
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// Epic represents a top-level project organization unit
type Epic struct {
	BaseEntity                           // 9 shared fields + 10 accessor methods
	Status        EpicStatus             `json:"status" db:"status"`
	Priority      Priority               `json:"priority" db:"priority"`
	BusinessValue *Priority              `json:"business_value,omitempty" db:"business_value"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"-"` // Not persisted to DB, derived from related data
}

// Entity interface implementation for Epic.

func (e *Epic) GetEntityType() EntityType { return EntityTypeEpic }
func (e *Epic) GetStatus() string         { return string(e.Status) }
func (e *Epic) SetStatus(status string)   { e.Status = EpicStatus(status) }

// Validate validates the Epic fields
func (e *Epic) Validate() error {
	if err := ValidateEpicKey(e.Key); err != nil {
		return err
	}
	if e.Title == "" {
		return ErrEmptyTitle
	}
	if err := ValidateEpicStatus(string(e.Status)); err != nil {
		return err
	}
	if err := ValidatePriority(string(e.Priority)); err != nil {
		return err
	}
	if e.BusinessValue != nil {
		if err := ValidatePriority(string(*e.BusinessValue)); err != nil {
			return err
		}
	}
	return nil
}
