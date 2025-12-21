package config

import "time"

// Config represents the .sharkconfig.json structure
type Config struct {
	// LastSyncTime is the timestamp of the last successful sync
	// Stored as RFC3339 format with timezone
	LastSyncTime *time.Time `json:"last_sync_time,omitempty"`

	// Other config fields (can be extended as needed)
	ColorEnabled *bool                  `json:"color_enabled,omitempty"`
	DefaultEpic  *string                `json:"default_epic,omitempty"`
	DefaultAgent *string                `json:"default_agent,omitempty"`
	JSONOutput   *bool                  `json:"json_output,omitempty"`
	RawData      map[string]interface{} `json:"-"` // Store raw config data to preserve unknown fields
}
