package config

import (
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

// DefaultTemplateDir is the default template directory name used when no
// custom template_directory is configured in .sharkconfig.json.
const DefaultTemplateDir = "shark-templates"

// DefaultConsoleWidth is the fallback console width used by GetConsoleWidth
// when (a) the config field is unset/zero AND (b) terminal-size detection by
// the caller has failed. It is also the value returned by GetConsoleWidth on
// a nil *Config. Chosen to match the historical layout assumption for shark
// list views (~85-90 column terminals leave room for surrounding chrome).
const DefaultConsoleWidth = 120

// MinConsoleWidth is the lower clamp applied to GetConsoleWidth when the
// configured value is positive but extremely small. Below ~40 columns column
// titles and dashes do not fit in any reasonable list view, so we clamp.
const MinConsoleWidth = 40

// SprintDefaultsConfig holds team-level defaults for sprint creation.
// It is parsed from the "sprint_defaults" key in .sharkconfig.json.
type SprintDefaultsConfig struct {
	// Capacity is a map of agent_type -> default capacity_points.
	// Applied to new sprints at creation time when sprint_capacity rows are absent.
	Capacity map[string]float64 `json:"capacity,omitempty"`

	// CarryoverBehavior is the default --carryover flag value for shark sprint close.
	// Valid values: "next" (move to next planning sprint) or "backlog" (unassign).
	// Default: "next" when absent (resolveCarryoverMode() returns CarryoverNext).
	CarryoverBehavior string `json:"carryover_behavior,omitempty"`

	// AutoCreate, when true, causes shark sprint close to create a new sprint
	// automatically if no planning sprint exists. REQ-F-016 (Could Have).
	AutoCreate bool `json:"auto_create,omitempty"`
}

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
	Web                    *WebConfig             `json:"web,omitempty"`                      // Web dashboard server configuration
	RawData                map[string]interface{} `json:"-"`                                  // Store raw config data to preserve unknown fields

	// SprintDefaults holds team-level default configuration for sprint creation.
	// When sprint_defaults.capacity is non-empty, capacity rows are automatically
	// inserted into sprint_capacity at sprint creation time.
	// A nil or absent SprintDefaults means "no defaults configured."
	SprintDefaults *SprintDefaultsConfig `json:"sprint_defaults,omitempty"`

	// Maintainer holds the optional maintainer authorization gate configuration.
	// A nil or absent Maintainer is equivalent to "no password configured."
	Maintainer *MaintainerConfig `json:"maintainer,omitempty"`

	// Recent holds optional configuration for the `shark recent` command.
	// A nil or absent Recent means "use built-in defaults" (limit = 5).
	Recent *RecentConfig `json:"recent,omitempty"`

	// ConsoleWidth is the width (in columns) used to size CLI list views
	// (description column truncation, etc.). Zero or negative means
	// "auto-detect from the controlling terminal, falling back to
	// DefaultConsoleWidth." Positive values are clamped to >= MinConsoleWidth.
	// See GetConsoleWidth for resolution rules.
	ConsoleWidth int `json:"console_width,omitempty"`

	// TagRequiredForTypes lists entity types that MUST carry at least one tag
	// at creation time. Values are entity-type strings as returned by
	// models.EntityType.String() ("task", "feature", "epic", "bug", "change",
	// "idea"). Absent or empty = no enforcement. The backing field name differs
	// from the JSON tag so the exported accessor TagRequiredFor() can share
	// the same name as the JSON field.
	TagRequiredForTypes []string `json:"tag_required_for,omitempty"`

	// SizeRequiredForTypes lists entity types that MUST carry a non-nil --size
	// at creation time. Values match models.EntityType.String() ("task",
	// "feature", "epic", "bug", "change", "idea", "tech-debt"). Absent or
	// empty = no enforcement. Mirrors TagRequiredForTypes; see SizeRequiredFor.
	SizeRequiredForTypes []string `json:"size_required_for,omitempty"`

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
	Enabled        bool   `json:"enabled"`
	TracingEnabled bool   `json:"tracing_enabled"`
	MetricsEnabled bool   `json:"metrics_enabled"`
	LogLevel       string `json:"log_level"`
	LogFormat      string `json:"log_format"`
	LogFile        string `json:"log_file,omitempty"`
	// Exporter is the OTel span exporter backend. Valid values: "stdout", "otlp", "file_jsonl".
	// "file_jsonl" appends one JSON line per span to <project>/shark-data/.stats/events.jsonl.
	Exporter     string  `json:"exporter"`
	OTLPEndpoint string  `json:"otlp_endpoint"`
	OTLPProtocol string  `json:"otlp_protocol"`
	ServiceName  string  `json:"service_name"`
	SampleRate   float64 `json:"sample_rate,omitempty"`

	// CaptureAgentTranscripts controls whether full agent stdout/stderr is written
	// to per-dispatch transcript files under .shark/runs/{run_id}/. Default: false.
	CaptureAgentTranscripts bool `json:"capture_agent_transcripts,omitempty"`

	// LogTruncateBytes is the maximum number of bytes to include from agent stderr
	// and stdout tail in error log events. Zero is treated as 4096 by GetLogTruncateBytes.
	LogTruncateBytes int `json:"log_truncate_bytes,omitempty"`
}

// GetLogTruncateBytes returns the configured truncation limit for agent output
// in error log events. When the configured value is zero (unset), it returns
// the default of 4096 bytes.
func (o ObservabilityConfig) GetLogTruncateBytes() int {
	if o.LogTruncateBytes <= 0 {
		return 4096
	}
	return o.LogTruncateBytes
}

// WebConfig holds configuration for the shark web dashboard server.
// Port 0 means "use the default" (currently 7777, falling back to 7778–7790).
type WebConfig struct {
	Port int `json:"port,omitempty"` // TCP port for shark web; 0 means use default
}

// TagRequiredFor returns the configured list of entity types that require at
// least one tag on create. Returns nil when the config is nil or the field is
// absent/empty. The returned slice is a defensive copy — callers cannot
// mutate the underlying configuration. Satisfies services.TagEnforcementConfig.
func (c *Config) TagRequiredFor() []string {
	if c == nil {
		return nil
	}
	if len(c.TagRequiredForTypes) == 0 {
		return nil
	}
	out := make([]string, len(c.TagRequiredForTypes))
	copy(out, c.TagRequiredForTypes)
	return out
}

// SizeRequiredFor returns the configured list of entity types that require a
// non-nil --size on create. Returns nil when the config is nil or the field is
// absent/empty. Defensive copy — callers cannot mutate the underlying config.
// Satisfies services.SizeEnforcementConfig.
func (c *Config) SizeRequiredFor() []string {
	if c == nil {
		return nil
	}
	if len(c.SizeRequiredForTypes) == 0 {
		return nil
	}
	out := make([]string, len(c.SizeRequiredForTypes))
	copy(out, c.SizeRequiredForTypes)
	return out
}

// GetWebPort returns the configured web server port, or 0 if not set.
func (c *Config) GetWebPort() int {
	if c == nil || c.Web == nil {
		return 0
	}
	return c.Web.Port
}

// RecentConfig holds configuration for the `shark recent` command.
// All fields have sensible defaults; the zero value means "use built-in default".
// When Recent is nil or absent from .sharkconfig.json the built-in default of 5
// is used for GetRecentDefaultLimit. Existing configs without a "recent" section
// continue to load and validate without error.
type RecentConfig struct {
	// DefaultLimit is the maximum number of items returned by `shark recent`
	// when no positional argument or --limit flag is given.
	// A value of 0 or negative is treated as "use built-in default (5)".
	DefaultLimit int `json:"default_limit,omitempty"`
}

// GetRecentDefaultLimit returns the configured default limit for `shark recent`,
// or 5 if the field is missing, the section is absent, or the value is <= 0.
// It is nil-safe: calling it on a nil *Config returns the built-in default.
func (c *Config) GetRecentDefaultLimit() int {
	const builtinDefault = 5
	if c == nil || c.Recent == nil || c.Recent.DefaultLimit <= 0 {
		return builtinDefault
	}
	return c.Recent.DefaultLimit
}

// GetConsoleWidth returns the console width to use when rendering CLI list
// views.
//
// Resolution rules:
//
//  1. If c is nil or c.ConsoleWidth is zero/negative, the caller-supplied
//     detected width is used. Callers detect the controlling terminal width
//     once (e.g., via golang.org/x/term.GetSize) and pass it in. A
//     detectedWidth of zero or negative means "detection failed" and is
//     replaced by DefaultConsoleWidth.
//  2. If c.ConsoleWidth is positive, it is used as-is, clamped to
//     >= MinConsoleWidth (so config values like 5 don't produce unrenderable
//     tables).
//
// This method is the single source of truth for resolved console width and is
// safe to call on a nil *Config.
func (c *Config) GetConsoleWidth(detectedWidth int) int {
	if c == nil || c.ConsoleWidth <= 0 {
		if detectedWidth <= 0 {
			return DefaultConsoleWidth
		}
		return detectedWidth
	}
	if c.ConsoleWidth < MinConsoleWidth {
		return MinConsoleWidth
	}
	return c.ConsoleWidth
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
