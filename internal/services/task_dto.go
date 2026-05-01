package services

import (
	"time"
)

// CreateTaskInput contains the parameters for creating a new task.
type CreateTaskInput struct {
	// Required fields
	EpicKey    string `json:"epic_key"`    // Epic key (E##)
	FeatureKey string `json:"feature_key"` // Feature key (F## or E##-F##)
	Title      string `json:"title"`       // Task title

	// Optional fields
	AgentType      string   `json:"agent_type,omitempty"`      // Agent type (frontend, backend, qa, etc.)
	Priority       int      `json:"priority,omitempty"`        // Priority 1-10 (default: 5)
	ExecutionOrder int      `json:"execution_order,omitempty"` // Execution order for sequencing
	DependsOn      []string `json:"depends_on,omitempty"`      // Task keys this task depends on
	FilePath       string   `json:"file_path,omitempty"`       // Custom file path (relative to project root)
	TemplatePath   string   `json:"template_path,omitempty"`   // Custom template file path
	CreateFile     bool     `json:"create_file,omitempty"`     // Create file if it doesn't exist
	Force          bool     `json:"force,omitempty"`           // Force file reassignment if already claimed
	Description    string   `json:"description,omitempty"`     // Task description
	// Tags lists the names of registered tags to attach after the task is
	// created. Each name must already exist in the vocabulary
	// (`shark tags add`) — TaskService resolves each name through
	// TagService.AttachMany post-persistence and returns
	// *UnregisteredTagError on the first miss. Nil or empty slice means
	// "no tags"; see E28-F04 REQ-F-011 and spec §2.6 for the full contract.
	Tags []string `json:"tags,omitempty"`
	// Size is an optional canonical Fibonacci size value {1,2,3,5,8,13}.
	// Nil means "no size set" (stores NULL). Use models.ParseSize to convert
	// t-shirt labels (XS/S/M/L/XL/XXL) to numeric form before setting.
	// E07-F42 REQ-F-004.
	Size *int `json:"size,omitempty"`
}

// TaskUpdates contains fields that can be updated on an existing task.
// Only non-nil pointer fields will be updated.
type TaskUpdates struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	AgentType      *string `json:"agent_type,omitempty"`
	ExecutionOrder *int    `json:"execution_order,omitempty"`
	FilePath       *string `json:"file_path,omitempty"`
	// Tags is ADDITIVE on update (E28-F04 REQ-F-010): a non-empty slice
	// attaches each registered name; an empty or nil slice is a no-op
	// (does NOT detach). Detachment is only available via
	// `shark task tag rm`. Type is []string (not *[]string) because
	// empty-means-no-change.
	Tags []string `json:"tags,omitempty"`
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting. E07-F42 REQ-F-005.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the task's size to NULL regardless of the
	// Size field value. ClearSize takes precedence over Size.
	// Corresponds to `--size clear` on the CLI. E07-F42 REQ-F-005.
	ClearSize bool `json:"clear_size,omitempty"`
}

// TaskFilters contains criteria for filtering task lists.
type TaskFilters struct {
	EpicKey     string   `json:"epic_key,omitempty"`     // Filter by epic
	FeatureKey  string   `json:"feature_key,omitempty"`  // Filter by feature
	Status      string   `json:"status,omitempty"`       // Filter by status
	AgentType   string   `json:"agent_type,omitempty"`   // Filter by agent type
	Statuses    []string `json:"statuses,omitempty"`     // Filter by multiple statuses
	ShowAll     bool     `json:"show_all,omitempty"`     // Include completed tasks
	Blocked     bool     `json:"blocked,omitempty"`      // Only blocked tasks
	Limit       int      `json:"limit,omitempty"`        // Pagination: max results (0 = all)
	Offset      int      `json:"offset,omitempty"`       // Pagination: skip N results
	TitleSearch string   `json:"title_search,omitempty"` // Fuzzy search in title (case-insensitive substring)
	MinPriority int      `json:"min_priority,omitempty"` // Minimum priority (1-10)
	MaxPriority int      `json:"max_priority,omitempty"` // Maximum priority (1-10)
	// Tags filters tasks to those tagged with ALL of the supplied names
	// (AND semantics, E28-F05 REQ-F-005). Empty/nil means no tag filter.
	// When non-empty, requires tagSvc to be wired; otherwise
	// *TagFilterUnavailableError is returned (AC-30).
	Tags []string `json:"tags,omitempty"`
	// HasRejections, when true, restricts results to tasks with at least one
	// rejection note (RejectionCount > 0). Wired from the `--has-rejections`
	// CLI flag. The filter is applied after rejection-count enrichment.
	HasRejections bool `json:"has_rejections,omitempty"`
}

// DependencyTree represents the hierarchical dependency structure for a task.
type DependencyTree struct {
	Task         *TaskNode   `json:"task"`                   // The root task
	Dependencies []*TaskNode `json:"dependencies,omitempty"` // Tasks this task depends on
	Dependents   []*TaskNode `json:"dependents,omitempty"`   // Tasks that depend on this task
	Blocked      bool        `json:"blocked"`                // Whether any dependencies are unmet
	BlockedBy    []string    `json:"blocked_by,omitempty"`   // Keys of blocking tasks
	CanStart     bool        `json:"can_start"`              // Whether task can be started
	Depth        int         `json:"depth"`                  // Depth in dependency graph
}

// TaskNode represents a single task in a dependency tree.
type TaskNode struct {
	Key         string    `json:"key"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	AgentType   string    `json:"agent_type,omitempty"`
	IsCompleted bool      `json:"is_completed"`
	IsBlocked   bool      `json:"is_blocked"`
	UpdatedAt   time.Time `json:"updated_at"`
}
