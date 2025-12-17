package init

import (
	"encoding/json"
	"fmt"
	"os"
)

// createConfig creates configuration file
// Returns true if config was created, false if skipped
func (i *Initializer) createConfig(opts InitOptions) (bool, error) {
	configPath := opts.ConfigPath

	// Check if config exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists
		if !opts.Force {
			if opts.NonInteractive {
				// Skip in non-interactive mode
				return false, nil
			}

			// Prompt user (in interactive mode)
			fmt.Printf("Config file already exists at %s. Overwrite? (y/N): ", configPath)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return false, nil
			}
		}
	}

	// Create default config
	config := ConfigDefaults{
		DefaultEpic:  nil,
		DefaultAgent: nil,
		ColorEnabled: true,
		JSONOutput:   false,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temp file
	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return false, fmt.Errorf("failed to write config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath) // Cleanup
		return false, fmt.Errorf("failed to rename config: %w", err)
	}

	return true, nil
}
