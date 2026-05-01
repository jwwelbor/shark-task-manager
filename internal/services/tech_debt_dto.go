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
	// Size is an optional canonical Fibonacci size value {1,2,3,5,8,13}.
	// Nil means "no size set" (stores NULL). Use models.ParseSize to convert
	// t-shirt labels (XS/S/M/L/XL/XXL) to numeric form before setting.
	Size *int `json:"size,omitempty"`
	// Tags are repeatable tag names to attach to the new tech-debt item.
	// Each tag must already be registered in the vocabulary; see `shark tags
	// list` / `shark tags add`. Mirrors bug/change-card semantics on create.
	Tags []string `json:"tags,omitempty"`
	// Body, when non-empty, replaces the rendered placeholder body of the
	// tech-debt's markdown file (frontmatter is preserved). Sourced from the
	// CLI `--content` flag or piped stdin via cli.ResolveContentInput.
	Body string `json:"body,omitempty"`
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
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the tech-debt's size to NULL regardless of the
	// Size field value. ClearSize takes precedence over Size. Corresponds to
	// `--size clear` on the CLI.
	ClearSize bool `json:"clear_size,omitempty"`
	// Tags is additive on update — listed tags are attached to the entity but
	// no detachment is performed. To remove a tag, use `shark td tag rm`.
	// Empty/nil means no change. Mirrors bug/change-card semantics on update.
	Tags []string `json:"tags,omitempty"`
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
