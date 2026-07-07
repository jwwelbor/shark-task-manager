package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func clearDBOverrideEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"SHARK_DB_BACKEND",
		"SHARK_DB_URL",
		"SHARK_AUTH_TOKEN_FILE",
		"TURSO_AUTH_TOKEN",
	} {
		t.Setenv(key, "")
	}
}

func TestInitializeDatabase_LocalBackend(t *testing.T) {
	clearDBOverrideEnv(t)

	// Create temp directory for test
	tempDir := t.TempDir()

	// Create test config with local backend
	cfg := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "sqlite",
			"url":     filepath.Join(tempDir, "test.db"),
		},
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Test: Initialize database using config
	ctx := context.Background()
	db, err := InitializeDatabaseFromConfig(ctx, configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer db.Close()

	// Verify: Database is connected and working
	if err := db.Ping(ctx); err != nil {
		t.Errorf("expected database to be connected, got ping error: %v", err)
	}

	// Verify: Driver name is correct
	if db.DriverName() != "sqlite" {
		t.Errorf("expected driver 'sqlite', got: %s", db.DriverName())
	}
}

func TestInitializeDatabase_TursoBackend(t *testing.T) {
	clearDBOverrideEnv(t)

	// Skip if no Turso credentials available (integration test)
	tursoURL := os.Getenv("TEST_TURSO_URL")
	tursoToken := os.Getenv("TEST_TURSO_TOKEN")
	if tursoURL == "" || tursoToken == "" {
		t.Skip("Skipping Turso test: TEST_TURSO_URL and TEST_TURSO_TOKEN not set")
	}

	// Create temp directory for test
	tempDir := t.TempDir()

	// Write auth token to file
	tokenFile := filepath.Join(tempDir, "turso-token")
	if err := os.WriteFile(tokenFile, []byte(tursoToken), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	// Create test config with Turso backend
	cfg := map[string]interface{}{
		"database": map[string]interface{}{
			"backend":         "turso",
			"url":             tursoURL,
			"auth_token_file": tokenFile,
		},
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Test: Initialize database using config
	ctx := context.Background()
	db, err := InitializeDatabaseFromConfig(ctx, configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer db.Close()

	// Verify: Database is connected and working
	if err := db.Ping(ctx); err != nil {
		t.Errorf("expected database to be connected, got ping error: %v", err)
	}

	// Verify: Driver name is correct
	if db.DriverName() != "libsql" {
		t.Errorf("expected driver 'libsql', got: %s", db.DriverName())
	}
}

func TestInitializeDatabase_MissingConfig(t *testing.T) {
	clearDBOverrideEnv(t)

	ctx := context.Background()
	_, err := InitializeDatabaseFromConfig(ctx, "/nonexistent/path/.sharkconfig.json")
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
}

func TestInitializeDatabase_InvalidConfig(t *testing.T) {
	clearDBOverrideEnv(t)

	// Create temp directory for test
	tempDir := t.TempDir()

	// Create invalid config (Turso URL with local backend)
	cfg := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "libsql://invalid.turso.io", // Wrong URL type for local
		},
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Test: Should fail validation
	ctx := context.Background()
	_, err = InitializeDatabaseFromConfig(ctx, configPath)
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

func TestGetDatabaseConfig_ParsesConfigCorrectly(t *testing.T) {
	clearDBOverrideEnv(t)

	// Create temp directory for test
	tempDir := t.TempDir()

	// Create test config
	expectedURL := "libsql://test.turso.io"
	expectedTokenFile := "/path/to/token"

	cfg := map[string]interface{}{
		"database": map[string]interface{}{
			"backend":         "turso",
			"url":             expectedURL,
			"auth_token_file": expectedTokenFile,
		},
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Test: Parse config
	dbConfig, err := GetDatabaseConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify: Config fields are correct
	if dbConfig.Backend != "turso" {
		t.Errorf("expected backend 'turso', got: %s", dbConfig.Backend)
	}

	if dbConfig.URL != expectedURL {
		t.Errorf("expected URL %s, got: %s", expectedURL, dbConfig.URL)
	}

	if dbConfig.AuthTokenFile != expectedTokenFile {
		t.Errorf("expected auth_token_file %s, got: %s", expectedTokenFile, dbConfig.AuthTokenFile)
	}
}

func TestGetDatabaseConfig_FallbackToLocalDB(t *testing.T) {
	clearDBOverrideEnv(t)

	// Create temp directory for test
	tempDir := t.TempDir()

	// Create config WITHOUT database section (should fall back to local DB)
	cfg := map[string]interface{}{
		"color_enabled": true,
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Test: Parse config should fall back to local database
	dbConfig, err := GetDatabaseConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify: Falls back to local SQLite
	if dbConfig.Backend != "sqlite" && dbConfig.Backend != "local" {
		t.Errorf("expected backend 'sqlite' or 'local', got: %s", dbConfig.Backend)
	}

	// Should have a default URL (shark-tasks.db)
	if dbConfig.URL == "" {
		t.Error("expected default URL, got empty string")
	}
}

func TestGetDatabaseConfig_IgnoresAmbientDatabaseOverrideEnv(t *testing.T) {
	clearDBOverrideEnv(t)

	tempDir := t.TempDir()
	configuredURL := filepath.Join(tempDir, "configured.db")

	cfg := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "sqlite",
			"url":     configuredURL,
		},
	}

	configPath := filepath.Join(tempDir, ".sharkconfig.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	t.Setenv("SHARK_DB_BACKEND", "turso")
	t.Setenv("SHARK_DB_URL", "libsql://wrong.turso.io")
	t.Setenv("SHARK_AUTH_TOKEN_FILE", "/tmp/wrong-token")

	dbConfig, err := GetDatabaseConfig(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if dbConfig.Backend != "sqlite" {
		t.Fatalf("backend = %q, want %q", dbConfig.Backend, "sqlite")
	}
	if dbConfig.URL != configuredURL {
		t.Fatalf("url = %q, want %q", dbConfig.URL, configuredURL)
	}
	if dbConfig.AuthTokenFile != "" {
		t.Fatalf("auth_token_file = %q, want empty", dbConfig.AuthTokenFile)
	}
}
