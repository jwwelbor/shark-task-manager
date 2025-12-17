package init

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateConfig(t *testing.T) {
	tests := []struct {
		name          string
		opts          InitOptions
		setupFunc     func(string) error
		wantCreated   bool
		wantErr       bool
		userInput     string
	}{
		{
			name: "creates new config file",
			opts: InitOptions{
				ConfigPath:     ".pmconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc:   nil,
			wantCreated: true,
			wantErr:     false,
		},
		{
			name: "skips existing config in non-interactive mode",
			opts: InitOptions{
				ConfigPath:     ".pmconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: func(baseDir string) error {
				// Create existing config
				configPath := filepath.Join(baseDir, ".pmconfig.json")
				return os.WriteFile(configPath, []byte(`{"existing":"config"}`), 0644)
			},
			wantCreated: false,
			wantErr:     false,
		},
		{
			name: "overwrites existing config with force flag",
			opts: InitOptions{
				ConfigPath:     ".pmconfig.json",
				NonInteractive: true,
				Force:          true,
			},
			setupFunc: func(baseDir string) error {
				// Create existing config
				configPath := filepath.Join(baseDir, ".pmconfig.json")
				return os.WriteFile(configPath, []byte(`{"existing":"config"}`), 0644)
			},
			wantCreated: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Execute
			initializer := NewInitializer()
			created, err := initializer.createConfig(tt.opts)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("createConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if created != tt.wantCreated {
				t.Errorf("createConfig() created = %v, want %v", created, tt.wantCreated)
			}

			// Verify config file if created
			if created {
				configPath := filepath.Join(tempDir, tt.opts.ConfigPath)
				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Errorf("Failed to read config file: %v", err)
					return
				}

				// Verify JSON is valid
				var config ConfigDefaults
				if err := json.Unmarshal(data, &config); err != nil {
					t.Errorf("Config file is not valid JSON: %v", err)
					return
				}

				// Verify default values
				if config.DefaultEpic != nil {
					t.Errorf("DefaultEpic = %v, want nil", *config.DefaultEpic)
				}
				if config.DefaultAgent != nil {
					t.Errorf("DefaultAgent = %v, want nil", *config.DefaultAgent)
				}
				if config.ColorEnabled != true {
					t.Errorf("ColorEnabled = %v, want true", config.ColorEnabled)
				}
				if config.JSONOutput != false {
					t.Errorf("JSONOutput = %v, want false", config.JSONOutput)
				}
			}
		})
	}
}

func TestCreateConfigAtomicWrite(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".pmconfig.json")

	// Execute
	initializer := NewInitializer()
	opts := InitOptions{
		ConfigPath:     configPath,
		NonInteractive: true,
		Force:          false,
	}

	created, err := initializer.createConfig(opts)
	if err != nil {
		t.Fatalf("createConfig() failed: %v", err)
	}

	if !created {
		t.Fatal("createConfig() did not create config")
	}

	// Verify no temp file exists
	tmpPath := configPath + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("Temporary file still exists after config creation")
	}

	// Verify final file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file does not exist")
	}
}

func TestCreateConfigPermissions(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".pmconfig.json")

	// Execute
	initializer := NewInitializer()
	opts := InitOptions{
		ConfigPath:     configPath,
		NonInteractive: true,
		Force:          false,
	}

	_, err := initializer.createConfig(opts)
	if err != nil {
		t.Fatalf("createConfig() failed: %v", err)
	}

	// Check permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config: %v", err)
	}

	gotPerms := info.Mode().Perm()
	wantPerms := os.FileMode(0644)

	if gotPerms != wantPerms {
		t.Errorf("Config permissions = %o, want %o", gotPerms, wantPerms)
	}
}

func TestCreateConfigValidJSON(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".pmconfig.json")

	// Execute
	initializer := NewInitializer()
	opts := InitOptions{
		ConfigPath:     configPath,
		NonInteractive: true,
		Force:          false,
	}

	_, err := initializer.createConfig(opts)
	if err != nil {
		t.Fatalf("createConfig() failed: %v", err)
	}

	// Read and parse JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config ConfigDefaults
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Config is not valid JSON: %v", err)
	}

	// Verify structure
	expectedJSON := `{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false
}`

	var expected, actual map[string]interface{}
	json.Unmarshal([]byte(expectedJSON), &expected)
	json.Unmarshal(data, &actual)

	if len(expected) != len(actual) {
		t.Errorf("Config has %d fields, want %d", len(actual), len(expected))
	}
}
