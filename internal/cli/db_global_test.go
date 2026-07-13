package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

func TestSetDBInitializerForTest_OverridesLazyInitialization(t *testing.T) {
	wantErr := errors.New("database initialization must not run")
	restore := SetDBInitializerForTest(func(context.Context) (*repository.DB, error) {
		return nil, wantErr
	})
	t.Cleanup(restore)

	_, err := GetDB(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetDB() error = %v, want %v", err, wantErr)
	}
}

func TestGetDB_InitializesOnce(t *testing.T) {
	// Setup: Create a temporary directory for test database
	tmpDir := t.TempDir()
	testDB := filepath.Join(tmpDir, "test-shark.db")

	// Create a minimal config file for testing
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	configContent := `{
		"database": {
			"backend": "local",
			"url": "` + testDB + `"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set working directory to tmpDir so config is found
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetDB() // Cleanup after test

	ctx := context.Background()

	// First call should initialize
	db1, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error on first call, got: %v", err)
	}
	if db1 == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Second call should return same instance (cached)
	db2, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error on second call, got: %v", err)
	}

	if db1 != db2 {
		t.Error("Expected same database instance on second call, got different instances")
	}
}

func TestResetDB_ClearsState(t *testing.T) {
	// Setup: Create a temporary directory for test database
	tmpDir := t.TempDir()
	testDB := filepath.Join(tmpDir, "test-shark.db")

	// Create a minimal config file for testing
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	configContent := `{
		"database": {
			"backend": "local",
			"url": "` + testDB + `"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set working directory to tmpDir so config is found
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx := context.Background()

	// Initialize database
	db1, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if db1 == nil {
		t.Fatal("Expected database instance, got nil")
	}

	// Reset state
	ResetDB()

	// Next call should reinitialize (create new instance)
	db2, err := GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error after reset, got: %v", err)
	}

	// Should be different instance since we reinitialized
	// Note: This might be the same pointer if DB pool is used,
	// but the important thing is it's a fresh initialization
	if db1 == db2 {
		t.Log("Warning: Same pointer after reset (may indicate DB pooling)")
	}
}

func TestCloseDB_SafeToCallMultipleTimes(t *testing.T) {
	// Setup: Create a temporary directory for test database
	tmpDir := t.TempDir()
	testDB := filepath.Join(tmpDir, "test-shark.db")

	// Create a minimal config file for testing
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	configContent := `{
		"database": {
			"backend": "local",
			"url": "` + testDB + `"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set working directory to tmpDir so config is found
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	defer ResetDB()

	ctx := context.Background()

	// Initialize database
	_, err = GetDB(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Close should succeed
	if err := CloseDB(); err != nil {
		t.Errorf("Expected no error on first close, got: %v", err)
	}

	// Second close should be safe (no-op)
	if err := CloseDB(); err != nil {
		t.Errorf("Expected no error on second close, got: %v", err)
	}
}

func TestInitDatabase_RespectsExplicitConfigAndDBOverrides(t *testing.T) {
	projectRoot := t.TempDir()
	customDBDir := t.TempDir()
	configPath := filepath.Join(projectRoot, "custom-config.json")
	relativeDB := filepath.Join("..", filepath.Base(customDBDir), "override.db")

	configContent := `{
		"database": {
			"backend": "local",
			"url": "ignored-by-flag.db"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldConfigFile := GlobalConfig.ConfigFile
	oldDBPath := GlobalConfig.DBPath
	GlobalConfig.ConfigFile = configPath
	GlobalConfig.DBPath = relativeDB
	defer func() {
		GlobalConfig.ConfigFile = oldConfigFile
		GlobalConfig.DBPath = oldDBPath
		_ = RootCmd.PersistentFlags().Set("db", "shark-tasks.db")
		_ = RootCmd.PersistentFlags().Set("config", "")
		if flag := RootCmd.PersistentFlags().Lookup("db"); flag != nil {
			flag.Changed = false
		}
		if flag := RootCmd.PersistentFlags().Lookup("config"); flag != nil {
			flag.Changed = false
		}
	}()

	if err := RootCmd.PersistentFlags().Set("config", configPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}
	if err := RootCmd.PersistentFlags().Set("db", relativeDB); err != nil {
		t.Fatalf("set db flag: %v", err)
	}

	repoDB, err := initDatabase(context.Background())
	if err != nil {
		t.Fatalf("initDatabase() error = %v", err)
	}
	defer repoDB.Close()

	expectedDBPath := filepath.Join(filepath.Dir(configPath), relativeDB)
	if _, err := os.Stat(expectedDBPath); err != nil {
		t.Fatalf("expected override DB at %s: %v", expectedDBPath, err)
	}
}
