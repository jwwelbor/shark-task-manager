package init

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateConfig(t *testing.T) {
	tests := []struct {
		name        string
		opts        InitOptions
		setupFunc   func(string) error
		wantCreated bool
		wantErr     bool
	}{
		{
			name: "creates new config file",
			opts: InitOptions{
				ConfigPath:     ".sharkconfig.json",
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
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: func(baseDir string) error {
				configPath := filepath.Join(baseDir, ".sharkconfig.json")
				return os.WriteFile(configPath, []byte(`{"existing":"config"}`), 0644)
			},
			wantCreated: false,
			wantErr:     false,
		},
		{
			name: "overwrites existing config with force flag",
			opts: InitOptions{
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          true,
			},
			setupFunc: func(baseDir string) error {
				configPath := filepath.Join(baseDir, ".sharkconfig.json")
				return os.WriteFile(configPath, []byte(`{"existing":"config"}`), 0644)
			},
			wantCreated: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer func() { _ = os.Chdir(originalDir) }()

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			initializer := NewInitializer()
			created, err := initializer.createConfig(tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("createConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if created != tt.wantCreated {
				t.Errorf("createConfig() created = %v, want %v", created, tt.wantCreated)
			}

			if !created {
				return
			}

			configPath := filepath.Join(tempDir, tt.opts.ConfigPath)
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			var cfg ConfigDefaults
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("Config file is not valid JSON: %v", err)
			}

			if !cfg.ColorEnabled {
				t.Errorf("ColorEnabled = false, want true")
			}
			if cfg.JSONOutput {
				t.Errorf("JSONOutput = true, want false")
			}
			if !cfg.RequireRejectionReason {
				t.Errorf("RequireRejectionReason = false, want true")
			}
			if cfg.WorkflowConfig != "shark-templates/.sharkworkflow-short.json" {
				t.Errorf("WorkflowConfig = %q, want %q", cfg.WorkflowConfig, "shark-templates/.sharkworkflow-short.json")
			}
			if cfg.Database == nil {
				t.Fatal("Database section missing")
			}
			if cfg.Database.Backend != "local" {
				t.Errorf("Database.Backend = %q, want %q", cfg.Database.Backend, "local")
			}
			if cfg.Database.URL != "./shark-tasks.db" {
				t.Errorf("Database.URL = %q, want %q", cfg.Database.URL, "./shark-tasks.db")
			}
			if cfg.Database.SkipMigrations {
				t.Errorf("Database.SkipMigrations = true, want false")
			}
		})
	}
}

func TestCreateConfigAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

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

	if _, err := os.Stat(configPath + ".tmp"); err == nil {
		t.Error("Temporary file still exists after config creation")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file does not exist")
	}
}

func TestCreateConfigPermissions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	initializer := NewInitializer()
	opts := InitOptions{
		ConfigPath:     configPath,
		NonInteractive: true,
		Force:          false,
	}
	if _, err := initializer.createConfig(opts); err != nil {
		t.Fatalf("createConfig() failed: %v", err)
	}

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

func TestCreateConfigShape(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	initializer := NewInitializer()
	opts := InitOptions{
		ConfigPath:     configPath,
		NonInteractive: true,
		Force:          false,
	}
	if _, err := initializer.createConfig(opts); err != nil {
		t.Fatalf("createConfig() failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var actual map[string]interface{}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Config is not valid JSON: %v", err)
	}

	requiredFields := []string{
		"color_enabled",
		"json_output",
		"interactive_mode",
		"require_rejection_reason",
		"database",
		"workflow_config",
	}
	for _, field := range requiredFields {
		if _, exists := actual[field]; !exists {
			t.Errorf("Config missing required field: %s", field)
		}
	}

	// status_metadata must NOT be inlined — it lives in the workflow file now.
	if _, exists := actual["status_metadata"]; exists {
		t.Error("Config should not contain inline status_metadata; it should be loaded from workflow_config")
	}
	if _, exists := actual["patterns"]; exists {
		t.Error("Config should not contain patterns field")
	}
}
