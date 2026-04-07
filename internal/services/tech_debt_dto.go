package services

import "github.com/jwwelbor/shark-task-manager/internal/models"

// CreateTechDebtInput contains the parameters for creating a new tech-debt item.
type CreateTechDebtInput struct {
	Title          string                  `json:"title"`
	Description    string                  `json:"description,omitempty"`
	Category       models.TechDebtCategory `json:"category"`
	Severity       models.TechDebtSeverity `json:"severity"`
	EffortEstimate string                  `json:"effort_estimate,omitempty"`
	// FilePath overrides the default file path for the tech-debt markdown file.
	// When nil, defaults to docs/plan/tech-debt/<key>.md.
	FilePath *string `json:"file_path,omitempty"`
	// Force allows overwriting an existing file at the target path.
	Force bool `json:"force,omitempty"`
}

// TechDebtUpdates contains optional fields for updating a tech-debt item.
// Pointer fields allow partial updates: nil means "don't change this field".
type TechDebtUpdates struct {
	Title          *string                  `json:"title,omitempty"`
	Description    *string                  `json:"description,omitempty"`
	Category       *models.TechDebtCategory `json:"category,omitempty"`
	Severity       *models.TechDebtSeverity `json:"severity,omitempty"`
	EffortEstimate *string                  `json:"effort_estimate,omitempty"`
	FilePath       *string                  `json:"file_path,omitempty"`
}

// TechDebtFilters defines filter options for listing tech-debt items via the service layer.
type TechDebtFilters struct {
	Status   *string                  `json:"status,omitempty"`
	Category *models.TechDebtCategory `json:"category,omitempty"`
	Severity *models.TechDebtSeverity `json:"severity,omitempty"`
	ShowAll  bool                     `json:"show_all,omitempty"` // include terminal statuses
}

// TriageTechDebtInput contains the parameters for triaging a tech-debt item.
// Triage sets category, severity, and effort estimate, advancing status to "triaged"
// if currently in "identified" status.
type TriageTechDebtInput struct {
	Severity       string `json:"severity,omitempty"`
	Category       string `json:"category,omitempty"`
	EffortEstimate string `json:"effort_estimate,omitempty"`
}
