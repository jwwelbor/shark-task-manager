package init

import (
	"bytes"
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
			if _, err := fmt.Scanln(&response); err != nil {
				return false, fmt.Errorf("failed to read user input: %w", err)
			}
			if response != "y" && response != "Y" {
				return false, nil
			}
		}
	}

	// Create default config pointing at the shark-data/ content bundle. Workflow
	// definitions are resolved from shark-data/workflow/ (or the embedded bundle
	// when shark-data/ is absent from disk). Run 'shark admin install-shark-data'
	// to materialize the bundle on disk for local customization.
	config := ConfigDefaults{
		ColorEnabled:           true,
		JSONOutput:             false,
		InteractiveMode:        false,
		RequireRejectionReason: true,
		Database: &DatabaseConfigDefault{
			Backend:        "local",
			URL:            "./shark-tasks.db",
			SkipMigrations: false,
		},
		SharkDataPath:  "shark-data",
		WorkflowConfig: "shark-data/workflow/",
		Observability: &ObservabilityConfigDefault{
			Enabled:        false,
			TracingEnabled: false,
			MetricsEnabled: false,
			LogLevel:       "info",
			LogFormat:      "json",
			LogFile:        "",
			Exporter:       "stdout",
			OTLPEndpoint:   "",
			OTLPProtocol:   "grpc",
			ServiceName:    "shark-task-manager",
		},
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
