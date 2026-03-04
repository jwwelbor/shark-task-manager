package services

import "github.com/jwwelbor/shark-task-manager/internal/models"

// CreateBugInput contains the parameters for creating a new bug.
type CreateBugInput struct {
	Title            string             `json:"title"`
	Description      string             `json:"description,omitempty"`
	Severity         models.BugSeverity `json:"severity"`
	LinkedEntityType string             `json:"linked_entity_type,omitempty"`
	LinkedEntityKey  string             `json:"linked_entity_key,omitempty"`
}

// BugUpdates contains optional fields for updating a bug.
type BugUpdates struct {
	Title            *string             `json:"title,omitempty"`
	Description      *string             `json:"description,omitempty"`
	Severity         *models.BugSeverity `json:"severity,omitempty"`
	LinkedEntityType *string             `json:"linked_entity_type,omitempty"`
	LinkedEntityKey  *string             `json:"linked_entity_key,omitempty"`
}

// BugFilters defines filter options for listing bugs via the service layer.
type BugFilters struct {
	Status   *models.BugStatus   `json:"status,omitempty"`
	Severity *models.BugSeverity `json:"severity,omitempty"`
}

// TriageBugInput contains the parameters for triaging a bug.
// Triage may update severity, assign the bug to an agent, and optionally
// update the linked entity.  All fields are optional; only non-nil fields
// are applied.
type TriageBugInput struct {
	Severity         *models.BugSeverity `json:"severity,omitempty"`
	AssignedTo       *string             `json:"assigned_to,omitempty"`
	LinkedEntityType *string             `json:"linked_entity_type,omitempty"`
	LinkedEntityKey  *string             `json:"linked_entity_key,omitempty"`
}
