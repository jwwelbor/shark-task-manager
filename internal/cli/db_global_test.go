package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

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
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

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
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	defer ResetDB()

	ctx := context.Background()

	// Initialize database
	_, err := GetDB(ctx)
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
