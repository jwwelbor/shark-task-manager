package workflow

// StatusInfo provides detailed information about a workflow status.
// Used for display in CLI commands and status breakdowns.
type StatusInfo struct {
	// Name is the canonical status name (e.g., "ready_for_development")
	Name string `json:"name"`

	// Color for CLI display (e.g., "yellow", "green", "red")
	Color string `json:"color,omitempty"`

	// Description is a human-readable explanation of the status
	Description string `json:"description,omitempty"`

	// Phase groups statuses (e.g., "planning", "development", "review", "qa", "done")
	Phase string `json:"phase,omitempty"`

	// AgentTypes lists which agent types should handle tasks in this status
	AgentTypes []string `json:"agent_types,omitempty"`
}

// TransitionInfo describes a valid status transition.
// Used for displaying available next statuses to users.
type TransitionInfo struct {
	// TargetStatus is the status this transition leads to
	TargetStatus string `json:"target_status"`

	// Description of the target status
	Description string `json:"description,omitempty"`

	// Phase of the target status
	Phase string `json:"phase,omitempty"`

	// AgentTypes that handle tasks in the target status
	AgentTypes []string `json:"agent_types,omitempty"`

	// Color for display
	Color string `json:"color,omitempty"`
}

// StatusCount represents a status with its count for status breakdowns.
// Used in feature and epic status displays.
type StatusCount struct {
	// Status name
	Status string `json:"status"`

	// Count of items in this status
	Count int `json:"count"`

	// Color for display
	Color string `json:"color,omitempty"`

	// Phase grouping
	Phase string `json:"phase,omitempty"`
}

// StatusBreakdown provides workflow-ordered status counts with metadata.
// Used for feature and epic status displays.
type StatusBreakdown struct {
	// Counts is an ordered list of statuses with their counts
	Counts []StatusCount `json:"counts"`

	// Total is the sum of all counts
	Total int `json:"total"`

	// CompletedCount is the count of items in terminal statuses
	CompletedCount int `json:"completed_count"`

	// ProgressPct is the percentage of items in terminal statuses
	ProgressPct float64 `json:"progress_pct"`
}

// NewStatusBreakdown creates a StatusBreakdown from a status count map.
// Statuses are ordered by workflow phase.
func NewStatusBreakdown(counts map[string]int, service *Service) StatusBreakdown {
	breakdown := StatusBreakdown{
		Counts: make([]StatusCount, 0),
	}

	// Get all statuses in workflow order
	orderedStatuses := service.GetAllStatusesOrdered()

	// Build ordered counts
	for _, status := range orderedStatuses {
		count, exists := counts[status]
		if !exists || count == 0 {
			continue // Skip zero counts
		}

		meta := service.GetStatusMetadata(status)
		breakdown.Counts = append(breakdown.Counts, StatusCount{
			Status: status,
			Count:  count,
			Color:  meta.Color,
			Phase:  meta.Phase,
		})

		breakdown.Total += count

		// Track completed items
		if service.IsTerminalStatus(status) {
			breakdown.CompletedCount += count
		}
	}

	// Calculate progress percentage
	if breakdown.Total > 0 {
		breakdown.ProgressPct = float64(breakdown.CompletedCount) / float64(breakdown.Total) * 100
	}

	return breakdown
}
