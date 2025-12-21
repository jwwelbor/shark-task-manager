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
			args:    []string{"init", "--non-interactive"},
			setup:   nil,
			wantErr: false,
			verify: func(t *testing.T, tempDir string) {
				// Verify database exists
				dbPath := filepath.Join(tempDir, "shark-tasks.db")
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Error("Database file was not created")
				}

				// Verify folders exist
				for _, folder := range []string{"docs/plan", "shark-templates"} {
					folderPath := filepath.Join(tempDir, folder)
					if _, err := os.Stat(folderPath); os.IsNotExist(err) {
						t.Errorf("Folder %s was not created", folder)
					}
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
			args:    []string{"init", "--non-interactive", "--db", "custom-db.db"},
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
			args: []string{"init", "--non-interactive", "--force"},
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
			args: []string{"init", "--non-interactive", "--db", "shark-tasks.db"},
			setup: func(tempDir string) error {
				// Run init once first
				cli.RootCmd.SetArgs([]string{"init", "--non-interactive", "--db", "shark-tasks.db"})
				return cli.RootCmd.Execute()
			},
			wantErr: false,
			verify: func(t *testing.T, tempDir string) {
				// Verify database exists after second run
				dbPath := filepath.Join(tempDir, "shark-tasks.db")
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
	cli.RootCmd.SetArgs([]string{"init", "--non-interactive", "--json", "--db", "shark-tasks.db"})
	err = cli.RootCmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify database and config were created even with JSON output
	dbPath := filepath.Join(tempDir, "shark-tasks.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created with --json flag")
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created with --json flag")
	}
}
