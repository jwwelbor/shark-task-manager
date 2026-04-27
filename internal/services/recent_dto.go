package services

import "time"

// RecentFilters is the input DTO for RecentService.ListRecent.
// Limit must be a resolved positive integer (caller is responsible for bounds-checking).
// When all Include* flags are false, all entity types are included.
type RecentFilters struct {
	Limit           int // resolved positive value
	IncludeTasks    bool
	IncludeFeatures bool
	IncludeEpics    bool
	IncludeBugs     bool
	IncludeChanges  bool
	IncludeIdeas    bool
	IncludeTechDebt bool
}

// RecentItem is the output DTO representing a single recently-created entity.
// Type is one of "epic", "feature", "task", "bug", "change", "idea", or "tech_debt".
type RecentItem struct {
	Type      string    `json:"type"`
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}
