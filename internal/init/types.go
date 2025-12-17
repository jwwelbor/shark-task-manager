package init

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
}
