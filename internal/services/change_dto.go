package services

// CreateChangeCardInput contains the parameters for creating a new change-card.
type CreateChangeCardInput struct {
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Priority      int    `json:"priority,omitempty"`
	RequestedBy   string `json:"requested_by,omitempty"`
	EpicKey       string `json:"epic_key,omitempty"`    // e.g., "E07" -- resolved to epic_id
	FeatureKey    string `json:"feature_key,omitempty"` // e.g., "E07-F01" -- resolved to feature_id
	Justification string `json:"justification,omitempty"`
	// Tags lists the names of registered tags to attach after the
	// change-card is created. Each name must already exist in the
	// vocabulary (`shark tags add`) — ChangeCardService resolves each
	// name through TagService.AttachMany post-persistence and returns
	// *UnregisteredTagError on the first miss. Nil or empty slice means
	// "no tags"; see REQ-F-011 and spec §2.6 for the full contract.
	Tags []string `json:"tags,omitempty"`
	// Size is an optional canonical Fibonacci size value {1,2,3,5,8,13}.
	// Nil means "no size set" (stores NULL). Use models.ParseSize to convert
	// t-shirt labels (XS/S/M/L/XL/XXL) to numeric form before setting.
	// E07-F42 REQ-F-004.
	Size *int `json:"size,omitempty"`
	// Body, when non-empty, replaces the rendered placeholder body of the
	// change-card's markdown file (frontmatter is preserved). Sourced from
	// the CLI `--content` flag or piped stdin via cli.ResolveContentInput.
	Body string `json:"body,omitempty"`
}

// ChangeCardFilters contains filtering options for listing change-cards.
type ChangeCardFilters struct {
	Status     string `json:"status,omitempty"`
	EpicKey    string `json:"epic_key,omitempty"`
	FeatureKey string `json:"feature_key,omitempty"`
	ShowAll    bool   `json:"show_all,omitempty"` // include terminal statuses
	// Tags filters change-cards to those tagged with ALL of the supplied names
	// (AND semantics, E28-F05 REQ-F-005). Empty/nil means no tag filter.
	Tags []string `json:"tags,omitempty"`
}

// ChangeCardUpdates contains optional fields for updating a change-card.
type ChangeCardUpdates struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	RequestedBy    *string `json:"requested_by,omitempty"`
	AssignedTo     *string `json:"assigned_to,omitempty"`
	Justification  *string `json:"justification,omitempty"`
	ImpactAnalysis *string `json:"impact_analysis,omitempty"`
	RollbackPlan   *string `json:"rollback_plan,omitempty"`
	FilePath       *string `json:"file_path,omitempty"`
	// Tags is ADDITIVE on update (REQ-F-010): a non-empty slice attaches
	// each registered name; an empty or nil slice is a no-op (does NOT
	// detach). Detachment is only available via `shark change tag rm`.
	// Type is []string (not *[]string) because empty-means-no-change.
	Tags []string `json:"tags,omitempty"`
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting. E07-F42 REQ-F-005.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the change-card's size to NULL regardless of
	// the Size field value. ClearSize takes precedence over Size.
	// Corresponds to `--size clear` on the CLI. E07-F42 REQ-F-005.
	ClearSize bool `json:"clear_size,omitempty"`
}
