package services

import "time"

// CreateIdeaInput contains the parameters for creating a new idea.
// The Key field is automatically generated as I-YYYY-MM-DD-xx by IdeaService.
type CreateIdeaInput struct {
	// Title is the idea title (required).
	Title string

	// Description is an optional longer description of the idea.
	Description *string

	// Priority is an optional 1-10 scale priority (1 = highest).
	Priority *int

	// Order is an optional display order for sorting ideas.
	Order *int

	// Notes is an optional free-form text for additional notes.
	Notes *string

	// RelatedDocs is an optional JSON-encoded list of related document paths.
	RelatedDocs *string

	// Dependencies is an optional JSON-encoded list of dependent idea keys.
	Dependencies *string

	// Status sets the initial status (defaults to "new" if empty).
	Status string

	// CreatedDate overrides the creation date (optional, defaults to today).
	// Use for testing or backfilling historical ideas.
	CreatedDate *time.Time

	// Tags are optional tag names to attach after creation. E28-F04 REQ-F-012.
	// Each name must be registered in the vocabulary; unknown names surface
	// as *UnregisteredTagError from the tag service. When `tag_required_for`
	// includes "idea" and Tags is empty, CreateIdea returns *TagRequiredError
	// before persistence (REQ-F-008).
	Tags []string
}

// UpdateIdeaInput contains the fields that can be updated on an existing idea.
// Only non-nil fields are applied.
type UpdateIdeaInput struct {
	// Title updates the idea title if non-nil.
	Title *string

	// Description updates the description if non-nil.
	Description *string

	// Priority updates the priority if non-nil.
	Priority *int

	// Order updates the display order if non-nil.
	Order *int

	// Notes updates the notes if non-nil.
	Notes *string

	// RelatedDocs updates the JSON-encoded list of related document paths if non-nil.
	RelatedDocs *string

	// Dependencies updates the JSON-encoded list of dependent idea keys if non-nil.
	Dependencies *string

	// Status updates the idea status if non-nil.
	Status *string

	// Tags are optional tag names to ADDITIVELY attach. E28-F04 REQ-F-010.
	// Empty (nil or []string{}) is a no-op — detach is not supported here;
	// removal goes through `shark idea tag rm`. Errors propagate unchanged.
	Tags []string
}

// IdeaFilters defines filtering options for listing ideas.
type IdeaFilters struct {
	// Status filters ideas by status ("new", "on_hold", "converted", "archived").
	// Empty string means no filter.
	Status string

	// DateFrom filters ideas created on or after this date (format: YYYY-MM-DD).
	DateFrom string

	// DateTo filters ideas created on or before this date (format: YYYY-MM-DD).
	DateTo string

	// Tags filters ideas to those tagged with ALL of the supplied names
	// (AND semantics, E28-F05 REQ-F-005). Empty/nil means no tag filter.
	Tags []string
}
