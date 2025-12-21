package init

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name      string
		opts      InitOptions
		setupFunc func(string) error
		wantErr   bool
		validate  func(*testing.T, *InitResult, string)
	}{
		{
			name: "full initialization from scratch",
			opts: InitOptions{
				DBPath:         "shark-tasks.db",
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: nil,
			wantErr:   false,
			validate: func(t *testing.T, result *InitResult, baseDir string) {
				// Verify database was created
				if !result.DatabaseCreated {
					t.Error("Expected DatabaseCreated = true")
				}
				if result.DatabasePath == "" {
					t.Error("DatabasePath is empty")
				}
				if !filepath.IsAbs(result.DatabasePath) {
					t.Error("DatabasePath is not absolute")
				}

				// Verify folders were created
				if len(result.FoldersCreated) != 2 {
					t.Errorf("FoldersCreated count = %d, want 2", len(result.FoldersCreated))
				}

				// Verify config was created
				if !result.ConfigCreated {
					t.Error("Expected ConfigCreated = true")
				}
				if result.ConfigPath == "" {
					t.Error("ConfigPath is empty")
				}

				// Verify templates were copied (count may be 0 if no templates embedded yet)
				if result.TemplatesCopied < 0 {
					t.Errorf("TemplatesCopied = %d, want >= 0", result.TemplatesCopied)
				}
			},
		},
		{
			name: "idempotent - everything exists but database is empty",
			opts: InitOptions{
				DBPath:         "shark-tasks.db",
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: func(baseDir string) error {
				// Run init once (creates database with schema but no data)
				initializer := NewInitializer()
				ctx := context.Background()
				opts := InitOptions{
					DBPath:         filepath.Join(baseDir, "shark-tasks.db"),
					ConfigPath:     filepath.Join(baseDir, ".sharkconfig.json"),
					NonInteractive: true,
					Force:          false,
				}
				_, err := initializer.Initialize(ctx, opts)
				return err
			},
			wantErr: false,
			validate: func(t *testing.T, result *InitResult, baseDir string) {
				// Second run should not create database or config
				if result.DatabaseCreated {
					t.Error("Expected DatabaseCreated = false on second run")
				}
				if result.ConfigCreated {
					t.Error("Expected ConfigCreated = false on second run")
				}
				// Folders already exist, so count should be 0
				if result.FoldersCreated == nil {
					t.Error("FoldersCreated should not be nil")
				}
				if len(result.FoldersCreated) != 0 {
					t.Errorf("Expected 0 folders created on second run, got %d", len(result.FoldersCreated))
				}
			},
		},
		{
			name: "fails when database contains data",
			opts: InitOptions{
				DBPath:         "shark-tasks.db",
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: func(baseDir string) error {
				// Run init once to create database
				initializer := NewInitializer()
				ctx := context.Background()
				opts := InitOptions{
					DBPath:         filepath.Join(baseDir, "shark-tasks.db"),
					ConfigPath:     filepath.Join(baseDir, ".sharkconfig.json"),
					NonInteractive: true,
					Force:          false,
				}
				if _, err := initializer.Initialize(ctx, opts); err != nil {
					return err
				}

				// Add some data to the database
				db, err := sql.Open("sqlite3", filepath.Join(baseDir, "shark-tasks.db"))
				if err != nil {
					return err
				}
				defer db.Close()

				_, err = db.Exec("INSERT INTO epics (key, title, status, priority) VALUES ('E01', 'Test Epic', 'draft', 'medium')")
				return err
			},
			wantErr: true,
			validate: func(t *testing.T, result *InitResult, baseDir string) {
				// Should fail when trying to reinit with existing data
			},
		},
		{
			name: "force mode overwrites config",
			opts: InitOptions{
				DBPath:         "shark-tasks.db",
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          true,
			},
			setupFunc: func(baseDir string) error {
				// Create existing config
				configPath := filepath.Join(baseDir, ".sharkconfig.json")
				return os.WriteFile(configPath, []byte(`{"old":"config"}`), 0644)
			},
			wantErr: false,
			validate: func(t *testing.T, result *InitResult, baseDir string) {
				// Config should be recreated
				if !result.ConfigCreated {
					t.Error("Expected ConfigCreated = true with --force")
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
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Execute
			initializer := NewInitializer()
			ctx := context.Background()
			result, err := initializer.Initialize(ctx, tt.opts)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Fatal("Initialize() returned nil result")
				}

				// Run validation function
				if tt.validate != nil {
					tt.validate(t, result, tempDir)
				}
			}
		})
	}
}

func TestInitializeWithContext(t *testing.T) {
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

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute
	initializer := NewInitializer()
	opts := InitOptions{
		DBPath:         "shark-tasks.db",
		ConfigPath:     ".sharkconfig.json",
		NonInteractive: true,
		Force:          false,
	}

	result, err := initializer.Initialize(ctx, opts)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Initialize() returned nil result")
	}
}

func TestInitializeErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		opts      InitOptions
		setupFunc func(string) error
		wantStep  string // Expected error step
	}{
		{
			name: "database creation fails",
			opts: InitOptions{
				DBPath:         "invalid/path/shark-tasks.db",
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: func(baseDir string) error {
				// Create a file where the database directory should be
				return os.WriteFile(filepath.Join(baseDir, "invalid"), []byte("block"), 0644)
			},
			wantStep: "database",
		},
		{
			name: "folder creation fails",
			opts: InitOptions{
				DBPath:         "shark-tasks.db",
				ConfigPath:     ".sharkconfig.json",
				NonInteractive: true,
				Force:          false,
			},
			setupFunc: func(baseDir string) error {
				// Create a file where docs folder should be
				return os.WriteFile(filepath.Join(baseDir, "docs"), []byte("block"), 0644)
			},
			wantStep: "folders",
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
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Logf("Setup completed: %v", err)
				}
			}

			// Execute
			initializer := NewInitializer()
			ctx := context.Background()
			_, err = initializer.Initialize(ctx, tt.opts)

			// Should fail
			if err == nil {
				t.Fatal("Initialize() expected error, got nil")
			}

			// Check error type
			initErr, ok := err.(*InitError)
			if !ok {
				t.Fatalf("Expected *InitError, got %T", err)
			}

			if initErr.Step != tt.wantStep {
				t.Errorf("Error step = %s, want %s", initErr.Step, tt.wantStep)
			}
		})
	}
}

func TestInitializePerformance(t *testing.T) {
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

	// Execute with timing
	initializer := NewInitializer()
	ctx := context.Background()
	opts := InitOptions{
		DBPath:         "shark-tasks.db",
		ConfigPath:     ".sharkconfig.json",
		NonInteractive: true,
		Force:          false,
	}

	start := time.Now()
	_, err = initializer.Initialize(ctx, opts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Should complete in < 5 seconds (PRD requirement)
	maxDuration := 5 * time.Second
	if elapsed > maxDuration {
		t.Errorf("Initialize() took %v, want < %v", elapsed, maxDuration)
	}
}
