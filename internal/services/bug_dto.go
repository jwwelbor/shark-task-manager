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
	// Tags lists the names of registered tags to attach after the bug is
	// created. Each name must already exist in the vocabulary
	// (`shark tags add`) — BugService resolves each name through
	// TagService.AttachMany post-persistence and returns
	// *UnregisteredTagError on the first miss. Nil or empty slice means
	// "no tags"; see REQ-F-011 and spec §2.6 for the full contract.
	Tags []string `json:"tags,omitempty"`
	// Size is an optional canonical Fibonacci size value {1,2,3,5,8,13}.
	// Nil means "no size set" (stores NULL). Use models.ParseSize to convert
	// t-shirt labels (XS/S/M/L/XL/XXL) to numeric form before setting.
	// E07-F42 REQ-F-004.
	Size *int `json:"size,omitempty"`
}

// BugUpdates contains optional fields for updating a bug.
type BugUpdates struct {
	Title            *string             `json:"title,omitempty"`
	Description      *string             `json:"description,omitempty"`
	Severity         *models.BugSeverity `json:"severity,omitempty"`
	LinkedEntityType *string             `json:"linked_entity_type,omitempty"`
	LinkedEntityKey  *string             `json:"linked_entity_key,omitempty"`
	FilePath         *string             `json:"file_path,omitempty"`
	// Tags is ADDITIVE on update (REQ-F-010): a non-empty slice attaches
	// each registered name; an empty or nil slice is a no-op (does NOT
	// detach). Detachment is only available via `shark bug tag rm`.
	// Type is []string (not *[]string) because empty-means-no-change.
	Tags []string `json:"tags,omitempty"`
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting. E07-F42 REQ-F-005.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the bug's size to NULL regardless of the
	// Size field value. ClearSize takes precedence over Size.
	// Corresponds to `--size clear` on the CLI. E07-F42 REQ-F-005.
	ClearSize bool `json:"clear_size,omitempty"`
}

// BugFilters defines filter options for listing bugs via the service layer.
type BugFilters struct {
	Status          *models.BugStatus   `json:"status,omitempty"`
	Severity        *models.BugSeverity `json:"severity,omitempty"`
	LinkedEntityKey *string             `json:"linked_entity_key,omitempty"`
	ShowAll         bool                `json:"show_all,omitempty"` // include terminal statuses
	// Tags filters bugs to those tagged with ALL of the supplied names
	// (AND semantics, E28-F05 REQ-F-005). Empty/nil means no tag filter.
	Tags []string `json:"tags,omitempty"`
}

// TriageBugInput contains the parameters for triaging a bug.
// Triage sets severity, advancing status from "reported" to "triaged".
type TriageBugInput struct {
	Severity string `json:"severity"` // Required. Must be a valid severity value.
}
