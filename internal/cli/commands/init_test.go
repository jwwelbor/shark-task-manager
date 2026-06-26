package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(string) error
		wantErr bool
		verify  func(*testing.T, string)
	}{
		{
			name:    "basic initialization",
			args:    []string{"admin", "init", "--non-interactive", "--db", "test-shark.db"},
			setup:   nil,
			wantErr: false,
			verify: func(t *testing.T, tempDir string) {
				// Verify database exists
				dbPath := filepath.Join(tempDir, "test-shark.db")
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Error("Database file was not created")
				}

				// Verify docs/plan folder exists (shark-templates is no longer created by init)
				folderPath := filepath.Join(tempDir, "docs/plan")
				if _, err := os.Stat(folderPath); os.IsNotExist(err) {
					t.Error("Folder docs/plan was not created")
				}

				// Verify shark-templates is NOT created (content served from embedded bundle)
				if _, err := os.Stat(filepath.Join(tempDir, "shark-templates")); err == nil {
					t.Error("shark-templates should not be created by shark admin init")
				}

				// Verify config exists
				configPath := filepath.Join(tempDir, ".sharkconfig.json")
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Error("Config file was not created")
				}
			},
		},
		{
			name:    "init with custom db path",
			args:    []string{"admin", "init", "--non-interactive", "--db", "custom-db.db"},
			setup:   nil,
			wantErr: false,
			verify: func(t *testing.T, tempDir string) {
				// Verify custom database exists
				dbPath := filepath.Join(tempDir, "custom-db.db")
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Error("Custom database file was not created")
				}
			},
		},
		{
			name: "init with force flag",
			args: []string{"admin", "init", "--non-interactive", "--force"},
			setup: func(tempDir string) error {
				// Create existing config
				configPath := filepath.Join(tempDir, ".sharkconfig.json")
				return os.WriteFile(configPath, []byte(`{"old":"config"}`), 0644)
			},
			wantErr: false,
			verify: func(t *testing.T, tempDir string) {
				// Verify config was overwritten
				configPath := filepath.Join(tempDir, ".sharkconfig.json")
				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("Failed to read config: %v", err)
				}
				// Should contain new default config, not old one
				if string(data) == `{"old":"config"}` {
					t.Error("Config was not overwritten with --force")
				}
			},
		},
		{
			name: "idempotent initialization",
			args: []string{"admin", "init", "--non-interactive", "--db", "test-idempotent.db"},
			setup: func(tempDir string) error {
				// Run init once first
				cli.RootCmd.SetArgs([]string{"admin", "init", "--non-interactive", "--db", "test-idempotent.db"})
				return cli.RootCmd.Execute()
			},
			wantErr: false,
			verify: func(t *testing.T, tempDir string) {
				// Verify database exists after second run
				dbPath := filepath.Join(tempDir, "test-idempotent.db")
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Error("Database file does not exist after second init")
				}
			},
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
			defer func() { _ = os.Chdir(originalDir) }()

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup
			if tt.setup != nil {
				if err := tt.setup(tempDir); err != nil {
					t.Logf("Setup completed: %v", err)
				}
			}

			// Execute command
			cli.RootCmd.SetArgs(tt.args)
			err = cli.RootCmd.Execute()

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("Command error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify
			if tt.verify != nil && !tt.wantErr {
				tt.verify(t, tempDir)
			}
		})
	}
}

func TestInitCommandJSON(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Execute with --json flag
	cli.RootCmd.SetArgs([]string{"admin", "init", "--non-interactive", "--json", "--db", "test-json.db"})
	err = cli.RootCmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify database and config were created even with JSON output
	dbPath := filepath.Join(tempDir, "test-json.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created with --json flag")
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created with --json flag")
	}
}
