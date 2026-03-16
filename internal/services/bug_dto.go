package services

import "github.com/jwwelbor/shark-task-manager/internal/models"

// CreateBugInput contains the parameters for creating a new bug.
type CreateBugInput struct {
	Title            string             `json:"title"`
	Description      string             `json:"description,omitempty"`
	Severity         models.BugSeverity `json:"severity"`
	LinkedEntityType string             `json:"linked_entity_type,omitempty"`
	LinkedEntityKey  string             `json:"linked_entity_key,omitempty"`
	// FilePath overrides the default file path for the bug markdown file.
	// When nil, defaults to docs/plan/bugs/<key>.md.
	FilePath *string `json:"file_path,omitempty"`
	// Force allows overwriting an existing file at the target path.
	Force bool `json:"force,omitempty"`
}

// BugUpdates contains optional fields for updating a bug.
type BugUpdates struct {
	Title            *string             `json:"title,omitempty"`
	Description      *string             `json:"description,omitempty"`
	Severity         *models.BugSeverity `json:"severity,omitempty"`
	LinkedEntityType *string             `json:"linked_entity_type,omitempty"`
	LinkedEntityKey  *string             `json:"linked_entity_key,omitempty"`
	FilePath         *string             `json:"file_path,omitempty"`
}

// BugFilters defines filter options for listing bugs via the service layer.
type BugFilters struct {
	Status          *models.BugStatus   `json:"status,omitempty"`
	Severity        *models.BugSeverity `json:"severity,omitempty"`
	LinkedEntityKey *string             `json:"linked_entity_key,omitempty"`
}

// TriageBugInput contains the parameters for triaging a bug.
// Triage sets severity, advancing status from "reported" to "triaged".
type TriageBugInput struct {
	Severity string `json:"severity"` // Required. Must be a valid severity value.
}
