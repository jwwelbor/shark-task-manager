package config

import (
	"encoding/json"
	"testing"
)

// TestDatabaseConfig_Marshaling tests that DatabaseConfig can be marshaled and unmarshaled
func TestDatabaseConfig_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "config with turso backend",
			config: Config{
				Database: &DatabaseConfig{
					Backend:         "turso",
					URL:             "libsql://shark-tasks.turso.io",
					AuthTokenFile:   "~/.shark/turso-token",
					EmbeddedReplica: true,
				},
			},
			expected: `{"database":{"backend":"turso","url":"libsql://shark-tasks.turso.io","auth_token_file":"~/.shark/turso-token","embedded_replica":true}}`,
		},
		{
			name: "config with local backend",
			config: Config{
				Database: &DatabaseConfig{
					Backend: "local",
					URL:     "./shark-tasks.db",
				},
			},
			expected: `{"database":{"backend":"local","url":"./shark-tasks.db"}}`,
		},
		{
			name:     "config without database (backward compat)",
			config:   Config{},
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("marshaled JSON mismatch\ngot:  %s\nwant: %s", string(data), tt.expected)
			}

			// Unmarshal back
			var unmarshaled Config
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			// Verify fields
			if tt.config.Database != nil {
				if unmarshaled.Database == nil {
					t.Error("database config was lost during unmarshal")
					return
				}
				if unmarshaled.Database.Backend != tt.config.Database.Backend {
					t.Errorf("backend mismatch: got %q, want %q", unmarshaled.Database.Backend, tt.config.Database.Backend)
				}
				if unmarshaled.Database.URL != tt.config.Database.URL {
					t.Errorf("url mismatch: got %q, want %q", unmarshaled.Database.URL, tt.config.Database.URL)
				}
			}
		})
	}
}

// TestDatabaseConfig_DefaultValues tests that nil database config is handled gracefully
func TestDatabaseConfig_DefaultValues(t *testing.T) {
	config := Config{}

	if config.Database != nil {
		t.Error("expected Database to be nil by default")
	}

	// Should be safe to check nil database config
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config with nil database: %v", err)
	}

	expected := `{}`
	if string(jsonData) != expected {
		t.Errorf("expected empty object, got: %s", string(jsonData))
	}
}

// TestDatabaseConfig_ValidationBackend tests backend validation
func TestDatabaseConfig_ValidationBackend(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		url     string
		valid   bool
	}{
		{"turso backend", "turso", "libsql://db.turso.io", true},
		{"local backend", "local", "./shark-tasks.db", true},
		{"sqlite backend (alias for local)", "sqlite", "./shark-tasks.db", true},
		{"empty backend (auto-detect)", "", "./shark-tasks.db", true},
		{"invalid backend", "postgres", "./shark-tasks.db", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DatabaseConfig{
				Backend: tt.backend,
				URL:     tt.url,
			}
			err := config.Validate()

			if tt.valid && err != nil {
				t.Errorf("expected backend %q to be valid, got error: %v", tt.backend, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected backend %q to be invalid, but validation passed", tt.backend)
			}
		})
	}
}

// TestDatabaseConfig_ValidationURL tests URL validation
func TestDatabaseConfig_ValidationURL(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		url     string
		valid   bool
	}{
		{"turso with libsql URL", "turso", "libsql://shark-tasks.turso.io", true},
		{"turso with https URL", "turso", "https://shark-tasks.turso.io", true},
		{"local with file path", "local", "./shark-tasks.db", true},
		{"local with absolute path", "local", "/home/user/shark-tasks.db", true},
		{"turso with empty URL", "turso", "", false},
		{"local with empty URL", "local", "", false},
		{"turso with file path", "turso", "./shark-tasks.db", false},
		{"local with libsql URL", "local", "libsql://db.turso.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DatabaseConfig{
				Backend: tt.backend,
				URL:     tt.url,
			}
			err := config.Validate()

			if tt.valid && err != nil {
				t.Errorf("expected valid config, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected validation error, but validation passed")
			}
		})
	}
}

// TestDetectBackend tests automatic backend detection from URL
func TestDetectBackend(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"libsql URL", "libsql://shark-tasks.turso.io", "turso"},
		{"https URL", "https://shark-tasks.turso.io", "turso"},
		{"relative file path", "./shark-tasks.db", "local"},
		{"absolute file path", "/home/user/shark-tasks.db", "local"},
		{"relative path", "data/shark.db", "local"},
		{"empty string", "", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := DetectBackend(tt.url)
			if backend != tt.expected {
				t.Errorf("DetectBackend(%q) = %q; want %q", tt.url, backend, tt.expected)
			}
		})
	}
}
