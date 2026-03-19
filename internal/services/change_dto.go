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
}

// ChangeCardFilters contains filtering options for listing change-cards.
type ChangeCardFilters struct {
	Status     string `json:"status,omitempty"`
	EpicKey    string `json:"epic_key,omitempty"`
	FeatureKey string `json:"feature_key,omitempty"`
	ShowAll    bool   `json:"show_all,omitempty"` // include terminal statuses
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
}
