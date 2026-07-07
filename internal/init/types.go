package init

// InitOptions contains initialization configuration
type InitOptions struct {
	DBPath         string // Database file path
	ConfigPath     string // Config file path
	NonInteractive bool   // Skip prompts
	Force          bool   // Overwrite existing config
}

// InitResult contains initialization results
type InitResult struct {
	DatabaseCreated bool     `json:"database_created"`
	DatabasePath    string   `json:"database_path"`
	FoldersCreated  []string `json:"folders_created"`
	ConfigCreated   bool     `json:"config_created"`
	ConfigPath      string   `json:"config_path"`
}

// ConfigDefaults is the JSON structure written by `shark admin init` when
// creating a fresh .sharkconfig.json. It mirrors the relevant subset of
// internal/config.Config so the file is immediately usable.
type ConfigDefaults struct {
	ColorEnabled           bool                        `json:"color_enabled"`
	JSONOutput             bool                        `json:"json_output"`
	InteractiveMode        bool                        `json:"interactive_mode"`
	RequireRejectionReason bool                        `json:"require_rejection_reason"`
	Backups                bool                        `json:"backups"`
	BackupFiles            int                         `json:"backup_files"`
	Database               *DatabaseConfigDefault      `json:"database"`
	SharkDataPath          string                      `json:"shark_data_path"`
	Observability          *ObservabilityConfigDefault `json:"observability"`
}

// DatabaseConfigDefault is the database section of the default config.
// Mirrors internal/db.DatabaseConfig (kept local to avoid an import cycle risk
// and to keep init self-contained).
type DatabaseConfigDefault struct {
	Backend        string `json:"backend"`
	URL            string `json:"url"`
	SkipMigrations bool   `json:"skip_migrations"`
}

// ObservabilityConfigDefault is the observability section of the default
// config. Mirrors internal/config.ObservabilityConfig (kept local to avoid an
// import cycle and to keep init self-contained). The scaffolded block is
// disabled so users can discover available fields without incurring any
// runtime overhead until they flip `enabled` to true.
type ObservabilityConfigDefault struct {
	Enabled        bool   `json:"enabled"`
	TracingEnabled bool   `json:"tracing_enabled"`
	MetricsEnabled bool   `json:"metrics_enabled"`
	LogLevel       string `json:"log_level"`
	LogFormat      string `json:"log_format"`
	LogFile        string `json:"log_file"`
	Exporter       string `json:"exporter"`
	OTLPEndpoint   string `json:"otlp_endpoint"`
	OTLPProtocol   string `json:"otlp_protocol"`
	ServiceName    string `json:"service_name"`
}
