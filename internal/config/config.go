package config

import (
	"fmt"
	"strings"
	"time"
)

// Config represents the .sharkconfig.json structure
type Config struct {
	// LastSyncTime is the timestamp of the last successful sync
	// Stored as RFC3339 format with timezone
	LastSyncTime *time.Time `json:"last_sync_time,omitempty"`

	// Database configuration for backend selection (local SQLite or cloud Turso)
	Database *DatabaseConfig `json:"database,omitempty"`

	// Other config fields (can be extended as needed)
	ColorEnabled           *bool                  `json:"color_enabled,omitempty"`
	DefaultEpic            *string                `json:"default_epic,omitempty"`
	DefaultAgent           *string                `json:"default_agent,omitempty"`
	JSONOutput             *bool                  `json:"json_output,omitempty"`
	InteractiveMode        *bool                  `json:"interactive_mode,omitempty"`         // Enable interactive prompts (default: false for automation)
	RequireRejectionReason bool                   `json:"require_rejection_reason,omitempty"` // NEW: Require rejection reason for backward transitions (default: false)
	RawData                map[string]interface{} `json:"-"`                                  // Store raw config data to preserve unknown fields

	// statusMetadata holds status metadata for work breakdown calculations
	// Internal field for testing and programmatic access
	statusMetadata map[string]*StatusMetadata `json:"-"`
}

// DatabaseConfig holds configuration for database backend selection
type DatabaseConfig struct {
	// Backend specifies the database type: "local" (SQLite) or "turso" (cloud)
	Backend string `json:"backend,omitempty"`

	// URL is the database connection string or file path
	// For local: "./shark-tasks.db" or absolute path
	// For turso: "libsql://your-db.turso.io" or "https://your-db.turso.io"
	URL string `json:"url,omitempty"`

	// AuthTokenFile is the path to a file containing the Turso auth token
	// Should be outside project directory with permissions 600
	AuthTokenFile string `json:"auth_token_file,omitempty"`

	// EmbeddedReplica enables offline mode with local replica that syncs to cloud
	// Only valid for turso backend
	EmbeddedReplica bool `json:"embedded_replica,omitempty"`
}

// Validate checks if the DatabaseConfig is valid
func (dc *DatabaseConfig) Validate() error {
	if dc == nil {
		return nil // nil config is valid (uses defaults)
	}

	// Validate URL is provided
	if dc.URL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}

	// If backend is empty, auto-detection will be used (valid)
	if dc.Backend == "" {
		return nil
	}

	// Validate backend if provided
	validBackends := map[string]bool{
		"local":  true,
		"sqlite": true, // alias for local
		"turso":  true,
	}

	if !validBackends[dc.Backend] {
		return fmt.Errorf("invalid database backend %q; must be 'local', 'sqlite', or 'turso'", dc.Backend)
	}

	// Validate URL format matches backend
	if dc.Backend == "turso" {
		if !strings.HasPrefix(dc.URL, "libsql://") && !strings.HasPrefix(dc.URL, "https://") {
			return fmt.Errorf("turso backend requires URL starting with 'libsql://' or 'https://', got: %s", dc.URL)
		}
	} else { // local or sqlite
		if strings.HasPrefix(dc.URL, "libsql://") || strings.HasPrefix(dc.URL, "https://") {
			return fmt.Errorf("local/sqlite backend requires file path, not URL: %s", dc.URL)
		}
	}

	return nil
}

// DetectBackend automatically detects the backend type from a database URL
// Returns "turso" for libsql:// or https:// URLs, "local" for file paths
func DetectBackend(url string) string {
	if strings.HasPrefix(url, "libsql://") || strings.HasPrefix(url, "https://") {
		return "turso"
	}
	return "local"
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
