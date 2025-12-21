package init

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
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
			if _, err := fmt.Scanln(&response); err != nil {
				return false, fmt.Errorf("failed to read user input: %w", err)
			}
			if response != "y" && response != "Y" {
				return false, nil
			}
		}
	}

	// Get default patterns
	defaultPatterns := patterns.GetDefaultPatterns()

	// Marshal patterns to JSON without HTML escaping
	var patternsBuf bytes.Buffer
	patternsEncoder := json.NewEncoder(&patternsBuf)
	patternsEncoder.SetEscapeHTML(false)
	if err := patternsEncoder.Encode(defaultPatterns); err != nil {
		return false, fmt.Errorf("failed to marshal patterns: %w", err)
	}
	patternsData := patternsBuf.Bytes()

	// Create default config with patterns
	config := ConfigDefaults{
		DefaultEpic:  nil,
		DefaultAgent: nil,
		ColorEnabled: true,
		JSONOutput:   false,
		PatternsRaw:  patternsData,
	}

	// Marshal to JSON without HTML escaping for readability
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return false, fmt.Errorf("failed to marshal config: %w", err)
	}
	data := buf.Bytes()

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
