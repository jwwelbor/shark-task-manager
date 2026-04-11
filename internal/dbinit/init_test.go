package dbinit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- helpers ------------------------------------------------------------------

// writeConfig writes a JSON-encoded .sharkconfig.json into dir.
func writeConfig(t *testing.T, dir string, data map[string]interface{}) {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), b, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// --- loadDatabaseConfig -------------------------------------------------------

func TestLoadDatabaseConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := loadDatabaseConfig(filepath.Join(dir, ".sharkconfig.json"), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("backend = %q, want %q", cfg.Backend, "sqlite")
	}
	if cfg.URL != filepath.Join(dir, "shark-tasks.db") {
		t.Errorf("url = %q, want %q", cfg.URL, filepath.Join(dir, "shark-tasks.db"))
	}
}

func TestLoadDatabaseConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{})
	cfg, err := loadDatabaseConfig(filepath.Join(dir, ".sharkconfig.json"), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("backend = %q, want %q", cfg.Backend, "sqlite")
	}
}

func TestLoadDatabaseConfig_LocalBackend(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "sqlite",
			"url":     "/tmp/custom.db",
		},
	})
	cfg, err := loadDatabaseConfig(filepath.Join(dir, ".sharkconfig.json"), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("backend = %q, want %q", cfg.Backend, "sqlite")
	}
	if cfg.URL != "/tmp/custom.db" {
		t.Errorf("url = %q, want %q", cfg.URL, "/tmp/custom.db")
	}
}

func TestLoadDatabaseConfig_EnvExpansion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TEST_DB_URL", "/tmp/from-env.db")
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "sqlite",
			"url":     "$TEST_DB_URL",
		},
	})
	cfg, err := loadDatabaseConfig(filepath.Join(dir, ".sharkconfig.json"), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.URL != "/tmp/from-env.db" {
		t.Errorf("url = %q, want %q", cfg.URL, "/tmp/from-env.db")
	}
}

func TestLoadDatabaseConfig_TursoBackend(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend":         "turso",
			"url":             "libsql://example.turso.io",
			"auth_token_file": "/home/user/.turso/token",
			"skip_migrations": true,
		},
	})
	cfg, err := loadDatabaseConfig(filepath.Join(dir, ".sharkconfig.json"), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Backend != "turso" {
		t.Errorf("backend = %q, want %q", cfg.Backend, "turso")
	}
	if cfg.URL != "libsql://example.turso.io" {
		t.Errorf("url = %q, want %q", cfg.URL, "libsql://example.turso.io")
	}
	if cfg.AuthTokenFile != "/home/user/.turso/token" {
		t.Errorf("auth_token_file = %q, want %q", cfg.AuthTokenFile, "/home/user/.turso/token")
	}
	if !cfg.SkipMigrations {
		t.Errorf("skip_migrations should be true")
	}
}

func TestLoadDatabaseConfig_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": "not-a-map",
	})
	_, err := loadDatabaseConfig(filepath.Join(dir, ".sharkconfig.json"), dir)
	if err == nil {
		t.Error("expected error for invalid database config format, got nil")
	}
}

// --- resolveRoots -------------------------------------------------------------

func TestResolveRoots_ExplicitProjectRoot(t *testing.T) {
	dir := t.TempDir()
	root, cfgPath, err := resolveRoots(Options{ProjectRoot: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("projectRoot = %q, want %q", root, dir)
	}
	if cfgPath != filepath.Join(dir, ".sharkconfig.json") {
		t.Errorf("configPath = %q, want %q", cfgPath, filepath.Join(dir, ".sharkconfig.json"))
	}
}

func TestResolveRoots_ExplicitConfigPath(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.json")
	root, cfgPath, err := resolveRoots(Options{ProjectRoot: dir, ConfigPath: custom})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("projectRoot = %q, want %q", root, dir)
	}
	if cfgPath != custom {
		t.Errorf("configPath = %q, want %q", cfgPath, custom)
	}
}

// --- resolveRoots with auto-discovery -----------------------------------------

func TestResolveRoots_AutoDiscover(t *testing.T) {
	dir := t.TempDir()
	// Write a .sharkconfig.json so findProjectRoot can locate the root.
	if err := os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Change the working directory so auto-discovery finds dir.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	root, cfgPath, err := resolveRoots(Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
	if cfgPath != filepath.Join(dir, ".sharkconfig.json") {
		t.Errorf("configPath = %q, want %q", cfgPath, filepath.Join(dir, ".sharkconfig.json"))
	}
}

// --- Init (local SQLite) ------------------------------------------------------

func TestInit_LocalSQLite(t *testing.T) {
	dir := t.TempDir()
	// No config file → falls back to local SQLite at <dir>/shark-tasks.db
	repoDb, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if repoDb == nil {
		t.Fatal("Init() returned nil DB")
	}
	defer repoDb.Close()

	// Verify the database file was created.
	dbPath := filepath.Join(dir, "shark-tasks.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file %q to be created", dbPath)
	}
}

func TestInit_LocalSQLite_ExplicitConfig(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "my.db")
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     dbPath,
		},
	})
	repoDb, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if repoDb == nil {
		t.Fatal("Init() returned nil DB")
	}
	defer repoDb.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file %q to be created", dbPath)
	}
}

func TestInit_LocalSQLite_UnreachablePath(t *testing.T) {
	// Attempting to create a database inside a non-existent directory hierarchy
	// should cause initLocal to return an error.
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "sqlite",
			"url":     filepath.Join(dir, "nonexistent", "subdir", "db.sqlite"),
		},
	})
	_, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err == nil {
		t.Fatal("expected error when database path is unreachable, got nil")
	}
}

func TestInit_UnsupportedBackend(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "postgres",
			"url":     "postgres://localhost/test",
		},
	})
	_, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err == nil {
		t.Fatal("expected error for unsupported backend, got nil")
	}
}

func TestInit_BadDatabaseConfigFormat_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": "not-a-map",
	})
	_, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err == nil {
		t.Fatal("expected error for invalid database config format, got nil")
	}
}

// --- initTurso error paths (no live Turso required) ---------------------------

func TestInit_Turso_InvalidURL_ReturnsError(t *testing.T) {
	// A turso config with a bad URL format triggers Validate() → error before
	// any network dial occurs.
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "turso",
			"url":     "http://bad-url-format.example.com",
		},
	})
	_, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err == nil {
		t.Fatal("expected error for invalid Turso URL format, got nil")
	}
}

func TestInit_Turso_MissingAuthTokenFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend":         "turso",
			"url":             "libsql://example.turso.io",
			"auth_token_file": filepath.Join(dir, "nonexistent-token-file"),
		},
	})
	_, err := Init(context.Background(), Options{ProjectRoot: dir})
	if err == nil {
		t.Fatal("expected error for missing auth token file, got nil")
	}
}

func TestInit_Turso_EnvVarAuth_FailsConnect(t *testing.T) {
	// Providing a fake env-var token causes the driver to attempt a connection
	// and fail, but it exercises the TURSO_AUTH_TOKEN code path.
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "turso",
			"url":     "libsql://fake.turso.io",
		},
	})
	t.Setenv("TURSO_AUTH_TOKEN", "fake-token-for-test")
	_, err := Init(context.Background(), Options{ProjectRoot: dir})
	// We expect an error (connection failure), not a nil.
	if err == nil {
		t.Fatal("expected error connecting to fake Turso endpoint, got nil")
	}
}

// --- MustInit -----------------------------------------------------------------

func TestMustInit_LocalSQLite(t *testing.T) {
	dir := t.TempDir()
	repoDb := MustInit(context.Background(), Options{ProjectRoot: dir})
	if repoDb == nil {
		t.Fatal("MustInit() returned nil DB")
	}
	defer repoDb.Close()
}

func TestMustInit_Panics_OnUnsupportedBackend(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "mysql",
			"url":     "mysql://localhost/test",
		},
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustInit to panic on unsupported backend")
		}
	}()

	MustInit(context.Background(), Options{ProjectRoot: dir})
}
