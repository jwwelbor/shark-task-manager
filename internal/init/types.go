package init

import "encoding/json"

// InitOptions contains initialization configuration
type InitOptions struct {
	DBPath         string // Database file path
	ConfigPath     string // Config file path
	NonInteractive bool   // Skip prompts
	Force          bool   // Overwrite existing files
}

// InitResult contains initialization results
type InitResult struct {
	DatabaseCreated bool     `json:"database_created"`
	DatabasePath    string   `json:"database_path"`
	FoldersCreated  []string `json:"folders_created"`
	ConfigCreated   bool     `json:"config_created"`
	ConfigPath      string   `json:"config_path"`
	TemplatesCopied int      `json:"templates_copied"`
}

// ConfigDefaults contains default configuration values
type ConfigDefaults struct {
	DefaultEpic  *string `json:"default_epic"`
	DefaultAgent *string `json:"default_agent"`
	ColorEnabled bool    `json:"color_enabled"`
	JSONOutput   bool    `json:"json_output"`
	// Patterns is defined in internal/patterns package to avoid import cycle
	PatternsRaw json.RawMessage `json:"patterns,omitempty"`
}

// WorkflowProfile represents a predefined workflow configuration
type WorkflowProfile struct {
	Name              string                   `json:"name"`
	Description       string                   `json:"description"`
	StatusMetadata    map[string]*StatusMetadata `json:"status_metadata"`
	StatusFlow        map[string][]string      `json:"status_flow,omitempty"`
	SpecialStatuses   map[string][]string      `json:"special_statuses,omitempty"`
	StatusFlowVersion string                   `json:"status_flow_version,omitempty"`
}

// StatusMetadata represents metadata for a single status
type StatusMetadata struct {
	Color          string   `json:"color"`
	Phase          string   `json:"phase"`
	ProgressWeight float64  `json:"progress_weight"`
	Responsibility string   `json:"responsibility"`
	BlocksFeature  bool     `json:"blocks_feature"`
	AgentTypes     []string `json:"agent_types,omitempty"`
	Description    string   `json:"description,omitempty"`
}

// UpdateOptions represents options for updating config
type UpdateOptions struct {
	ConfigPath     string
	WorkflowName   string
	Force          bool
	DryRun         bool
	NonInteractive bool
	Verbose        bool
}

// UpdateResult represents the result of a config update
type UpdateResult struct {
	Success     bool            `json:"success"`
	ProfileName string          `json:"profile_name,omitempty"`
	BackupPath  string          `json:"backup_path,omitempty"`
	Changes     *ChangeReport   `json:"changes"`
	ConfigPath  string          `json:"config_path"`
	DryRun      bool            `json:"dry_run"`
}

// ChangeReport details what changed during update
type ChangeReport struct {
	Added       []string      `json:"added"`
	Preserved   []string      `json:"preserved"`
	Overwritten []string      `json:"overwritten"`
	Stats       *ChangeStats  `json:"stats"`
}

// ChangeStats provides detailed change statistics
type ChangeStats struct {
	StatusesAdded   int `json:"statuses_added"`
	FlowsAdded      int `json:"flows_added"`
	GroupsAdded     int `json:"groups_added"`
	FieldsPreserved int `json:"fields_preserved"`
}
