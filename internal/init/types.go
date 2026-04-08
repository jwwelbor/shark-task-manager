package init

// InitOptions contains initialization configuration
type InitOptions struct {
	DBPath         string // Database file path
	ConfigPath     string // Config file path
	NonInteractive bool   // Skip prompts
	Force          bool   // Overwrite existing config and modified templates
	TemplateDir    string // Template directory name (default: "shark-templates")
}

// InitResult contains initialization results
type InitResult struct {
	DatabaseCreated    bool     `json:"database_created"`
	DatabasePath       string   `json:"database_path"`
	FoldersCreated     []string `json:"folders_created"`
	ConfigCreated      bool     `json:"config_created"`
	ConfigPath         string   `json:"config_path"`
	TemplatesCopied    int      `json:"templates_copied"`    // Files newly copied (didn't exist)
	TemplatesRefreshed int      `json:"templates_refreshed"` // Files overwritten via --force
	TemplatesDiffered  []string `json:"templates_differed"`  // Files skipped because local copy differs (need --force to overwrite)
}

// ConfigDefaults is the JSON structure written by `shark admin init` when
// creating a fresh .sharkconfig.json. It mirrors the relevant subset of
// internal/config.Config so the file is immediately usable.
type ConfigDefaults struct {
	ColorEnabled           bool                   `json:"color_enabled"`
	JSONOutput             bool                   `json:"json_output"`
	InteractiveMode        bool                   `json:"interactive_mode"`
	RequireRejectionReason bool                   `json:"require_rejection_reason"`
	Database               *DatabaseConfigDefault `json:"database"`
	WorkflowConfig         string                 `json:"workflow_config"`
}

// DatabaseConfigDefault is the database section of the default config.
// Mirrors internal/db.DatabaseConfig (kept local to avoid an import cycle risk
// and to keep init self-contained).
type DatabaseConfigDefault struct {
	Backend        string `json:"backend"`
	URL            string `json:"url"`
	SkipMigrations bool   `json:"skip_migrations"`
}
