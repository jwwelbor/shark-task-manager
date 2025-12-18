package patterns

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the full configuration with patterns
type Config struct {
	DefaultEpic  *string        `json:"default_epic"`
	DefaultAgent *string        `json:"default_agent"`
	ColorEnabled bool           `json:"color_enabled"`
	JSONOutput   bool           `json:"json_output"`
	Patterns     *PatternConfig `json:"patterns,omitempty"`
}

// LoadConfig loads configuration from the specified path with fallback to defaults
// This supports backward compatibility: if patterns are not specified in the config,
// default patterns are automatically applied
func LoadConfig(configPath string) (*Config, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into config struct
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply default patterns if not specified (backward compatibility)
	if config.Patterns == nil {
		config.Patterns = GetDefaultPatterns()
	}

	return &config, nil
}

// MergeWithDefaults merges user-provided patterns with defaults
// User patterns take precedence, but default patterns are used for any missing sections
func MergeWithDefaults(userPatterns *PatternConfig) *PatternConfig {
	if userPatterns == nil {
		return GetDefaultPatterns()
	}

	defaults := GetDefaultPatterns()
	result := &PatternConfig{}

	// Merge epic patterns
	result.Epic = mergeEntityPatterns(userPatterns.Epic, defaults.Epic)

	// Merge feature patterns
	result.Feature = mergeEntityPatterns(userPatterns.Feature, defaults.Feature)

	// Merge task patterns
	result.Task = mergeEntityPatterns(userPatterns.Task, defaults.Task)

	return result
}

// mergeEntityPatterns merges user and default patterns for a single entity type
func mergeEntityPatterns(user, defaults EntityPatterns) EntityPatterns {
	result := EntityPatterns{}

	// Use user folder patterns if provided, otherwise use defaults
	if len(user.Folder) > 0 {
		result.Folder = user.Folder
	} else {
		result.Folder = defaults.Folder
	}

	// Use user file patterns if provided, otherwise use defaults
	if len(user.File) > 0 {
		result.File = user.File
	} else {
		result.File = defaults.File
	}

	// Use user generation format if provided, otherwise use defaults
	if user.Generation.Format != "" {
		result.Generation = user.Generation
	} else {
		result.Generation = defaults.Generation
	}

	return result
}
