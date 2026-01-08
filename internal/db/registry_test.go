package db

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestRegisterDriver tests driver registration
func TestRegisterDriver(t *testing.T) {
	// Reset registry for testing
	ResetRegistry()

	// Register SQLite driver
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	// Verify driver is registered
	if _, exists := drivers["sqlite"]; !exists {
		t.Error("SQLite driver not registered")
	}
}

// TestDetectBackend tests backend auto-detection from URL
func TestDetectBackend(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"SQLite file path", "./shark-tasks.db", "sqlite"},
		{"SQLite absolute path", "/var/data/shark.db", "sqlite"},
		{"Turso libsql URL", "libsql://db.turso.io", "turso"},
		{"Turso https URL", "https://db.turso.io", "turso"},
		{"Empty string", "", "sqlite"},
		{"Relative path", "data/db.sqlite", "sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := DetectBackend(tt.url)
			if backend != tt.expected {
				t.Errorf("DetectBackend(%q) = %q, want %q", tt.url, backend, tt.expected)
			}
		})
	}
}

// TestNewDatabase tests creating database instances from config
func TestNewDatabase(t *testing.T) {
	// Reset and register driver
	ResetRegistry()
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	tests := []struct {
		name      string
		config    config.DatabaseConfig
		shouldErr bool
	}{
		{
			name: "SQLite with file path",
			config: config.DatabaseConfig{
				Backend: "sqlite",
				URL:     "./test.db",
			},
			shouldErr: false,
		},
		{
			name: "Auto-detect SQLite from file path",
			config: config.DatabaseConfig{
				URL: "./test.db",
			},
			shouldErr: false,
		},
		{
			name: "Unknown backend",
			config: config.DatabaseConfig{
				Backend: "postgres",
				URL:     "postgres://localhost",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDatabase(tt.config)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if db == nil {
					t.Error("Expected database instance, got nil")
				}
			}
		})
	}
}

// TestNewDatabase_UnregisteredBackend tests error for unregistered backend
func TestNewDatabase_UnregisteredBackend(t *testing.T) {
	ResetRegistry()

	config := config.DatabaseConfig{
		Backend: "sqlite",
		URL:     "./test.db",
	}

	_, err := NewDatabase(config)
	if err == nil {
		t.Error("Expected error for unregistered backend, got nil")
	}
}

// TestInitDatabase tests the high-level database initialization function
func TestInitDatabase(t *testing.T) {
	ctx := context.Background()

	// Reset and register driver
	ResetRegistry()
	RegisterDriver("sqlite", func() Database {
		return NewSQLiteDriver()
	})

	// Create temporary database
	tmpDB := t.TempDir() + "/test.db"

	config := config.DatabaseConfig{
		Backend: "sqlite",
		URL:     tmpDB,
	}

	db, err := InitDatabase(ctx, config)
	if err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Ping failed after InitDatabase: %v", err)
	}

	// Verify driver name
	if db.DriverName() != "sqlite3" {
		t.Errorf("Expected driver name 'sqlite3', got %q", db.DriverName())
	}
}

// TestGetRegisteredDrivers tests retrieving list of registered drivers
func TestGetRegisteredDrivers(t *testing.T) {
	ResetRegistry()

	// Initially empty
	drivers := GetRegisteredDrivers()
	if len(drivers) != 0 {
		t.Errorf("Expected 0 drivers, got %d", len(drivers))
	}

	// Register drivers
	RegisterDriver("sqlite", func() Database { return NewSQLiteDriver() })
	RegisterDriver("turso", func() Database { return nil }) // Mock

	drivers = GetRegisteredDrivers()
	if len(drivers) != 2 {
		t.Errorf("Expected 2 drivers, got %d", len(drivers))
	}

	// Verify driver names
	foundSQLite := false
	foundTurso := false
	for _, name := range drivers {
		if name == "sqlite" {
			foundSQLite = true
		}
		if name == "turso" {
			foundTurso = true
		}
	}

	if !foundSQLite {
		t.Error("SQLite driver not found in registered drivers")
	}
	if !foundTurso {
		t.Error("Turso driver not found in registered drivers")
	}
}
