package init

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
)

func TestCreateDatabase(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(string) error
		wantCreated   bool
		wantErr       bool
		checkPerms    bool
		expectedPerms os.FileMode
	}{
		{
			name:          "creates new database successfully",
			setupFunc:     nil,
			wantCreated:   true,
			wantErr:       false,
			checkPerms:    runtime.GOOS != "windows",
			expectedPerms: 0600,
		},
		{
			name: "idempotent - database already exists",
			setupFunc: func(dbPath string) error {
				// Ensure parent directory exists
				if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
					return err
				}
				// Create database first
				database, err := db.InitDB(dbPath)
				if err != nil {
					return err
				}
				return database.Close()
			},
			wantCreated: false,
			wantErr:     false,
		},
		{
			name: "fails with invalid directory path",
			setupFunc: func(dbPath string) error {
				// Create a file where the parent directory should be
				parent := filepath.Dir(dbPath)
				return os.WriteFile(parent, []byte("block"), 0644)
			},
			wantCreated: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test-db", "shark-tasks.db")

			// Setup
			if tt.setupFunc != nil {
				if err := tt.setupFunc(dbPath); err != nil {
					t.Logf("Setup function failed (expected for some tests): %v", err)
				}
			}

			// Execute
			initializer := NewInitializer()
			ctx := context.Background()
			created, err := initializer.createDatabase(ctx, dbPath)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("createDatabase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if created != tt.wantCreated {
				t.Errorf("createDatabase() created = %v, want %v", created, tt.wantCreated)
			}

			// Verify database exists and is valid (if no error expected)
			if !tt.wantErr {
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Errorf("Database file does not exist at %s", dbPath)
				}

				// Check permissions on Unix systems
				if tt.checkPerms {
					info, err := os.Stat(dbPath)
					if err != nil {
						t.Fatalf("Failed to stat database: %v", err)
					}
					gotPerms := info.Mode().Perm()
					if gotPerms != tt.expectedPerms {
						t.Errorf("Database permissions = %o, want %o", gotPerms, tt.expectedPerms)
					}
				}

				// Verify database schema is valid
				database, err := db.InitDB(dbPath)
				if err != nil {
					t.Errorf("Failed to open created database: %v", err)
				} else {
					defer database.Close()

					// Verify tables exist
					var tableCount int
					err = database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
					if err != nil {
						t.Errorf("Failed to query database: %v", err)
					}
					if tableCount < 4 { // epics, features, tasks, task_history
						t.Errorf("Database has %d tables, expected at least 4", tableCount)
					}
				}
			}
		})
	}
}

func TestCreateDatabaseFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix file permissions test on Windows")
	}

	// Create temp directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "shark-tasks.db")

	// Create database
	initializer := NewInitializer()
	ctx := context.Background()
	_, err := initializer.createDatabase(ctx, dbPath)
	if err != nil {
		t.Fatalf("createDatabase() failed: %v", err)
	}

	// Check permissions
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Failed to stat database: %v", err)
	}

	gotPerms := info.Mode().Perm()
	wantPerms := os.FileMode(0600)

	if gotPerms != wantPerms {
		t.Errorf("Database permissions = %o, want %o", gotPerms, wantPerms)
	}
}

func TestCreateDatabaseForeignKeysEnabled(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "shark-tasks.db")

	// Create database
	initializer := NewInitializer()
	ctx := context.Background()
	_, err := initializer.createDatabase(ctx, dbPath)
	if err != nil {
		t.Fatalf("createDatabase() failed: %v", err)
	}

	// Open database and verify foreign keys
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	var fkEnabled int
	err = database.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys: %v", err)
	}

	if fkEnabled != 1 {
		t.Errorf("Foreign keys enabled = %d, want 1", fkEnabled)
	}
}
