package config

import (
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

// DefaultTemplateDir is the default template directory name used when no
// custom template_directory is configured in .sharkconfig.json.
const DefaultTemplateDir = "shark-templates"

// Config represents the .sharkconfig.json structure
type Config struct {
	// LastSyncTime is the timestamp of the last successful sync
	// Stored as RFC3339 format with timezone
	LastSyncTime *time.Time `json:"last_sync_time,omitempty"`

	// Database configuration for backend selection (local SQLite or cloud Turso)
	Database *db.DatabaseConfig `json:"database,omitempty"`

	// Other config fields (can be extended as needed)
	ColorEnabled           *bool                  `json:"color_enabled,omitempty"`
	JSONOutput             *bool                  `json:"json_output,omitempty"`
	InteractiveMode        *bool                  `json:"interactive_mode,omitempty"`         // Enable interactive prompts (default: false for automation)
	RequireRejectionReason bool                   `json:"require_rejection_reason,omitempty"` // NEW: Require rejection reason for backward transitions (default: false)
	Viewer                 *string                `json:"viewer,omitempty"`                   // External viewer command for spec files (glow, nano, bat, less, cat, etc). Default: "cat"
	TemplateDirectory      *string                `json:"template_directory,omitempty"`       // Template directory path relative to project root. Default: "shark-templates"
	WorkflowConfig         *string                `json:"workflow_config,omitempty"`          // Path to workflow config file (default: .sharkworkflow.json). Read-only directive.
	Observability          *ObservabilityConfig   `json:"observability,omitempty"`            // Observability subsystem configuration
	RawData                map[string]interface{} `json:"-"`                                  // Store raw config data to preserve unknown fields

	// statusMetadata holds status metadata for work breakdown calculations
	// Internal field for testing and programmatic access
	statusMetadata map[string]*StatusMetadata `json:"-"`
}

// GetStatusMetadata returns metadata for a given status
// Returns nil if status metadata is not configured
func (c *Config) GetStatusMetadata(status string) *StatusMetadata {
	if c == nil || c.statusMetadata == nil {
		return nil
	}
	return c.statusMetadata[status]
}

// SetStatusMetadata sets the status metadata map (used for testing and configuration)
func (c *Config) SetStatusMetadata(metadata map[string]*StatusMetadata) {
	if c == nil {
		return
	}
	c.statusMetadata = metadata
}

// IsInteractiveModeEnabled returns true if interactive mode is enabled in config
// Defaults to false (non-interactive) for automation/agent workflows
func (c *Config) IsInteractiveModeEnabled() bool {
	if c == nil || c.InteractiveMode == nil {
		return false // Default: non-interactive for automation
	}
	return *c.InteractiveMode
}

// IsRequireRejectionReasonEnabled returns true if rejection reason is required for backward transitions
// Defaults to false (optional) for backward compatibility
func (c *Config) IsRequireRejectionReasonEnabled() bool {
	if c == nil {
		return false // Default: rejection reason optional
	}
	return c.RequireRejectionReason
}

// GetViewer returns the configured viewer command or default "cat"
// The viewer is used by the shark view command to open specification files
// Examples: "glow", "nano", "bat", "less", "cat"
func (c *Config) GetViewer() string {
	if c == nil || c.Viewer == nil || *c.Viewer == "" {
		return "cat" // Default viewer
	}
	return *c.Viewer
}

// GetTemplateDirectory returns the configured template directory or default "shark-templates"
// The directory path is relative to the project root.
func (c *Config) GetTemplateDirectory() string {
	if c == nil || c.TemplateDirectory == nil || *c.TemplateDirectory == "" {
		return DefaultTemplateDir
	}
	return *c.TemplateDirectory
}

// IsBackwardTransition determines whether a transition from oldStatus to newStatus is backward
// based on ProgressWeight values. A backward transition has lower weight in the new status.
// This method is used to determine if rejection reason validation should be applied (E07-F22).
func (c *Config) IsBackwardTransition(oldStatus, newStatus string, weights map[string]float64) bool {
	// If old and new status are the same, it's not backward
	if oldStatus == newStatus {
		return false
	}

	// If weights map is nil or empty, we can't determine (return false for safety)
	if len(weights) == 0 {
		return false
	}

	// Get the weight for each status
	oldWeight, oldFound := weights[oldStatus]
	newWeight, newFound := weights[newStatus]

	// If either status weight is not found, it's not backward
	if !oldFound || !newFound {
		return false
	}

	// Backward transition = moving to lower progress weight
	return newWeight < oldWeight
}

// GetObservability returns the observability config, or a zero-value config if nil.
// This ensures callers never need nil checks on the pointer.
func (c *Config) GetObservability() ObservabilityConfig {
	if c == nil || c.Observability == nil {
		return ObservabilityConfig{}
	}
	return *c.Observability
}

// ObservabilityConfig holds configuration for the observability subsystem.
// All fields have sensible defaults; the zero value means "disabled".
// When Enabled is false, no OTel SDK is initialized and no network connections
// are made. Existing users without this key in their config are unaffected.
type ObservabilityConfig struct {
	Enabled        bool    `json:"enabled"`
	TracingEnabled bool    `json:"tracing_enabled"`
	MetricsEnabled bool    `json:"metrics_enabled"`
	LogLevel       string  `json:"log_level"`
	LogFormat      string  `json:"log_format"`
	Exporter       string  `json:"exporter"`
	OTLPEndpoint   string  `json:"otlp_endpoint"`
	OTLPProtocol   string  `json:"otlp_protocol"`
	ServiceName    string  `json:"service_name"`
	SampleRate     float64 `json:"sample_rate,omitempty"`
}

// GetTemplateDirectoryFromConfig loads the template directory setting from the given config file path.
// Returns "shark-templates" if the config file doesn't exist, is unreadable, or doesn't contain the field.
func GetTemplateDirectoryFromConfig(configPath string) string {
	if configPath == "" {
		return DefaultTemplateDir
	}
	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		return DefaultTemplateDir
	}
	return cfg.GetTemplateDirectory()
}
