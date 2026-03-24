package services_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

func newConfigService() *services.ConfigService {
	return services.NewConfigService()
}

// writeConfigFile writes content as JSON to a temp file and returns the path.
func writeConfigFile(t *testing.T, content map[string]interface{}) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".sharkconfig.json")
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// TestConfigService_LoadPatternsFromConfig
// ---------------------------------------------------------------------------

func TestConfigService_LoadPatternsFromConfig(t *testing.T) {
	svc := newConfigService()

	t.Run("returns default patterns when no patterns section", func(t *testing.T) {
		path := writeConfigFile(t, map[string]interface{}{
			"database": map[string]interface{}{"backend": "local"},
		})

		cfg, err := svc.LoadPatternsFromConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil PatternConfig")
		}
		// Default patterns should have at least one epic folder pattern
		if len(cfg.Epic.Folder) == 0 {
			t.Error("expected default epic folder patterns")
		}
	})

	t.Run("loads patterns from config file", func(t *testing.T) {
		defaultPatterns := patterns.GetDefaultPatterns()
		path := writeConfigFile(t, map[string]interface{}{
			"patterns": defaultPatterns,
		})

		cfg, err := svc.LoadPatternsFromConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil PatternConfig")
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := svc.LoadPatternsFromConfig("/nonexistent/path.json")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_LoadDatabaseConfigFromFile
// ---------------------------------------------------------------------------

func TestConfigService_LoadDatabaseConfigFromFile(t *testing.T) {
	svc := newConfigService()

	t.Run("returns database config", func(t *testing.T) {
		path := writeConfigFile(t, map[string]interface{}{
			"database": map[string]interface{}{
				"backend": "local",
				"url":     "./shark-tasks.db",
			},
		})

		dbCfg, err := svc.LoadDatabaseConfigFromFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dbCfg == nil {
			t.Fatal("expected non-nil database config")
		}
		if dbCfg["backend"] != "local" {
			t.Errorf("expected backend=local, got %v", dbCfg["backend"])
		}
	})

	t.Run("returns nil for missing database section", func(t *testing.T) {
		path := writeConfigFile(t, map[string]interface{}{})

		dbCfg, err := svc.LoadDatabaseConfigFromFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dbCfg != nil {
			t.Errorf("expected nil database config, got %v", dbCfg)
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := svc.LoadDatabaseConfigFromFile("/nonexistent/config.json")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_TestPattern
// ---------------------------------------------------------------------------

func TestConfigService_TestPattern(t *testing.T) {
	svc := newConfigService()

	tests := []struct {
		name        string
		pattern     string
		testString  string
		wantMatched bool
		wantGroups  map[string]string
		wantErr     bool
	}{
		{
			name:        "matching pattern with named groups",
			pattern:     `E(?P<number>\d{2})`,
			testString:  "E07",
			wantMatched: true,
			wantGroups:  map[string]string{"number": "07"},
		},
		{
			name:        "non-matching pattern",
			pattern:     `E(?P<number>\d{2})`,
			testString:  "F01",
			wantMatched: false,
		},
		{
			name:       "invalid regex syntax",
			pattern:    `[invalid`,
			testString: "test",
			wantErr:    true,
		},
		{
			name:        "pattern without named groups",
			pattern:     `E\d{2}`,
			testString:  "E07",
			wantMatched: true,
			wantGroups:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.TestPattern(tt.pattern, tt.testString)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if result.Matched != tt.wantMatched {
				t.Errorf("TestPattern() Matched = %v, want %v", result.Matched, tt.wantMatched)
			}
			for k, v := range tt.wantGroups {
				if result.Groups[k] != v {
					t.Errorf("TestPattern() group %q = %v, want %v", k, result.Groups[k], v)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConfigService_FindMatchingPatterns
// ---------------------------------------------------------------------------

func TestConfigService_FindMatchingPatterns(t *testing.T) {
	svc := newConfigService()
	cfg := patterns.GetDefaultPatterns()

	t.Run("finds matching epic folder patterns", func(t *testing.T) {
		// Default epic folder patterns should match something like "E07-*"
		matches := svc.FindMatchingPatterns(cfg, "E07-user-management", "epic")
		// We just check that it doesn't panic and returns a slice
		_ = matches
	})

	t.Run("empty result for unknown entity type", func(t *testing.T) {
		matches := svc.FindMatchingPatterns(cfg, "anything", "unknown")
		if len(matches) != 0 {
			t.Errorf("expected no matches for unknown entity type, got %v", matches)
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_ValidateAllPatterns
// ---------------------------------------------------------------------------

func TestConfigService_ValidateAllPatterns(t *testing.T) {
	svc := newConfigService()

	t.Run("validates default patterns without errors", func(t *testing.T) {
		cfg := patterns.GetDefaultPatterns()
		report := svc.ValidateAllPatterns(cfg)
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		// Default patterns should not have errors
		if report.HasErrors() {
			t.Errorf("expected no errors in default patterns, got: epic=%v feature=%v task=%v",
				report.EpicErrors, report.FeatureErrors, report.TaskErrors)
		}
	})

	t.Run("reports errors for invalid patterns", func(t *testing.T) {
		cfg := &patterns.PatternConfig{
			Epic: patterns.EntityPatterns{
				Folder: []string{"[invalid-regex"},
			},
			Feature: patterns.EntityPatterns{},
			Task:    patterns.EntityPatterns{},
		}
		report := svc.ValidateAllPatterns(cfg)
		if !report.HasErrors() {
			t.Error("expected errors for invalid regex pattern")
		}
		if len(report.EpicErrors) == 0 {
			t.Error("expected epic errors")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_GetFormat
// ---------------------------------------------------------------------------

func TestConfigService_GetFormat(t *testing.T) {
	svc := newConfigService()
	cfg := patterns.GetDefaultPatterns()

	validTypes := []string{"epic", "feature", "task"}
	for _, entityType := range validTypes {
		t.Run("returns format for "+entityType, func(t *testing.T) {
			output, err := svc.GetFormat(cfg, entityType)
			if err != nil {
				t.Fatalf("GetFormat(%q) error: %v", entityType, err)
			}
			if output == nil {
				t.Fatal("expected non-nil output")
			}
			if len(output.Placeholders) == 0 {
				t.Error("expected non-empty placeholders")
			}
		})
	}

	t.Run("errors for invalid type", func(t *testing.T) {
		_, err := svc.GetFormat(cfg, "invalid")
		if err == nil {
			t.Error("expected error for invalid entity type")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_FileExists
// ---------------------------------------------------------------------------

func TestConfigService_FileExists(t *testing.T) {
	svc := newConfigService()

	t.Run("returns true for existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.json")
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if !svc.FileExists(path) {
			t.Error("expected FileExists to return true")
		}
	})

	t.Run("returns false for missing file", func(t *testing.T) {
		if svc.FileExists("/nonexistent/file.json") {
			t.Error("expected FileExists to return false")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_AddPreset
// ---------------------------------------------------------------------------

func TestConfigService_AddPreset(t *testing.T) {
	svc := newConfigService()

	t.Run("errors on unknown preset", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".sharkconfig.json")
		if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := svc.AddPreset(configPath, "nonexistent-preset-xyz")
		if err == nil {
			t.Error("expected error for unknown preset")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigService_PlaceholderString
// ---------------------------------------------------------------------------

func TestConfigService_PlaceholderString(t *testing.T) {
	svc := newConfigService()

	tests := []struct {
		entityType string
		contains   string
	}{
		{"epic", "number"},
		{"feature", "epic"},
		{"task", "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			result := svc.PlaceholderString(tt.entityType)
			if result == "" {
				t.Errorf("expected non-empty placeholder string for %q", tt.entityType)
			}
		})
	}

	t.Run("returns empty for unknown type", func(t *testing.T) {
		result := svc.PlaceholderString("unknown")
		if result != "" {
			t.Errorf("expected empty string for unknown type, got %q", result)
		}
	})
}
