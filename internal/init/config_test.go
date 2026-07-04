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
			if cfg.SharkDataPath != "shark-data" {
				t.Errorf("SharkDataPath = %q, want %q", cfg.SharkDataPath, "shark-data")
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
			if cfg.Observability == nil {
				t.Fatal("Observability section missing")
			}
			if cfg.Observability.Enabled {
				t.Errorf("Observability.Enabled = true, want false (scaffolded disabled)")
			}
			if cfg.Observability.TracingEnabled {
				t.Errorf("Observability.TracingEnabled = true, want false")
			}
			if cfg.Observability.MetricsEnabled {
				t.Errorf("Observability.MetricsEnabled = true, want false")
			}
			if cfg.Observability.LogLevel != "info" {
				t.Errorf("Observability.LogLevel = %q, want %q", cfg.Observability.LogLevel, "info")
			}
			if cfg.Observability.LogFormat != "json" {
				t.Errorf("Observability.LogFormat = %q, want %q", cfg.Observability.LogFormat, "json")
			}
			if cfg.Observability.ServiceName != "shark-task-manager" {
				t.Errorf("Observability.ServiceName = %q, want %q", cfg.Observability.ServiceName, "shark-task-manager")
			}
			// Test Plan case 9: log_file field is present and empty in scaffold.
			if cfg.Observability.LogFile != "" {
				t.Errorf("Observability.LogFile = %q, want %q (empty string)", cfg.Observability.LogFile, "")
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
		"shark_data_path",
		"observability",
	}
	for _, field := range requiredFields {
		if _, exists := actual[field]; !exists {
			t.Errorf("Config missing required field: %s", field)
		}
	}

	// workflow_config must NOT be written by init — a bare init has no
	// shark-data/ on disk, so the field would be a dangling pointer. Workflow
	// definitions resolve from the embedded bundle until
	// `shark admin install-shark-data` materializes shark-data/workflow/ and
	// re-adds workflow_config itself.
	if _, exists := actual["workflow_config"]; exists {
		t.Error("Config should not contain workflow_config; init no longer points at an unmaterialized bundle")
	}

	// status_metadata must NOT be inlined — it lives in the workflow file now.
	if _, exists := actual["status_metadata"]; exists {
		t.Error("Config should not contain inline status_metadata; it should be loaded from workflow_config")
	}
	if _, exists := actual["patterns"]; exists {
		t.Error("Config should not contain patterns field")
	}

	// Test Plan case 10: "log_file" key present in the observability object.
	obsRaw, ok := actual["observability"]
	if !ok {
		t.Fatal("Config missing observability field")
	}
	obsMap, ok := obsRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("observability field is not a JSON object, got %T", obsRaw)
	}
	if _, exists := obsMap["log_file"]; !exists {
		t.Error("observability object missing required key: log_file")
	}
}
