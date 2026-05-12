package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadConfig_ValidLastSyncTime tests loading config with valid last_sync_time
func TestLoadConfig_ValidLastSyncTime(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	expectedTime := time.Date(2025, 12, 17, 14, 30, 45, 0, time.FixedZone("PST", -8*3600))
	configData := map[string]interface{}{
		"last_sync_time": expectedTime.Format(time.RFC3339),
		"color_enabled":  true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Act
	manager := NewManager(configPath)
	config, err := manager.Load()

	// Assert
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	lastSyncTime := manager.GetLastSyncTime()
	if lastSyncTime == nil {
		t.Fatal("GetLastSyncTime() returned nil")
	}

	// Compare times (allowing for sub-second precision differences due to formatting)
	if !lastSyncTime.Equal(expectedTime) {
		t.Errorf("GetLastSyncTime() = %v, want %v", lastSyncTime, expectedTime)
	}
}

// TestLoadConfig_MissingLastSyncTime tests loading config without last_sync_time
func TestLoadConfig_MissingLastSyncTime(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Act
	manager := NewManager(configPath)
	config, err := manager.Load()

	// Assert
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	lastSyncTime := manager.GetLastSyncTime()
	if lastSyncTime != nil {
		t.Errorf("GetLastSyncTime() = %v, want nil", lastSyncTime)
	}
}

// TestLoadConfig_InvalidTimestamp tests loading config with invalid timestamp
func TestLoadConfig_InvalidTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
	}{
		{
			name:      "invalid format",
			timestamp: "2025-12-17 14:30:45",
		},
		{
			name:      "missing timezone",
			timestamp: "2025-12-17T14:30:45",
		},
		{
			name:      "invalid date",
			timestamp: "2025-13-45T14:30:45Z",
		},
		{
			name:      "empty string",
			timestamp: "",
		},
		{
			name:      "random text",
			timestamp: "not a timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, ".sharkconfig.json")

			configData := map[string]interface{}{
				"last_sync_time": tt.timestamp,
				"color_enabled":  true,
			}

			data, err := json.MarshalIndent(configData, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			// Act
			manager := NewManager(configPath)
			config, err := manager.Load()

			// Assert - should not return error, but treat as nil
			if err != nil {
				t.Fatalf("Load() should not fail with invalid timestamp: %v", err)
			}

			if config == nil {
				t.Fatal("Load() returned nil config")
			}

			lastSyncTime := manager.GetLastSyncTime()
			if lastSyncTime != nil {
				t.Errorf("GetLastSyncTime() with invalid timestamp = %v, want nil", lastSyncTime)
			}
		})
	}
}

// TestUpdateLastSyncTime tests updating last_sync_time
func TestUpdateLastSyncTime(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create initial config
	configData := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act
	updateTime := time.Date(2025, 12, 18, 10, 15, 30, 0, time.UTC)
	err = manager.UpdateLastSyncTime(updateTime)

	// Assert
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed: %v", err)
	}

	// Reload and verify
	manager2 := NewManager(configPath)
	_, err = manager2.Load()
	if err != nil {
		t.Fatalf("Load() after update failed: %v", err)
	}

	lastSyncTime := manager2.GetLastSyncTime()
	if lastSyncTime == nil {
		t.Fatal("GetLastSyncTime() after update returned nil")
	}

	if !lastSyncTime.Equal(updateTime) {
		t.Errorf("GetLastSyncTime() after update = %v, want %v", lastSyncTime, updateTime)
	}
}

// TestUpdateLastSyncTime_PreservesExistingFields tests that update preserves other config fields
func TestUpdateLastSyncTime_PreservesExistingFields(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create config with multiple fields
	configData := map[string]interface{}{
		"color_enabled": true,
		"default_epic":  "E01",
		"default_agent": "backend",
		"json_output":   false,
		"custom_field":  "custom_value",
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act
	updateTime := time.Now()
	err = manager.UpdateLastSyncTime(updateTime)

	// Assert
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed: %v", err)
	}

	// Read config directly to verify all fields preserved
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify all original fields exist
	if result["color_enabled"] != true {
		t.Error("color_enabled field was not preserved")
	}
	if result["default_epic"] != "E01" {
		t.Error("default_epic field was not preserved")
	}
	if result["default_agent"] != "backend" {
		t.Error("default_agent field was not preserved")
	}
	if result["json_output"] != false {
		t.Error("json_output field was not preserved")
	}
	if result["custom_field"] != "custom_value" {
		t.Error("custom_field was not preserved")
	}

	// Verify last_sync_time was added
	if result["last_sync_time"] == nil {
		t.Error("last_sync_time was not added")
	}
}

// TestUpdateLastSyncTime_AtomicWrite tests atomic file update
func TestUpdateLastSyncTime_AtomicWrite(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act
	updateTime := time.Now()
	err = manager.UpdateLastSyncTime(updateTime)

	// Assert
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed: %v", err)
	}

	// Verify no temp file exists
	tmpPath := configPath + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("Temporary file still exists after update")
	}

	// Verify final file exists and is valid JSON
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file does not exist after update")
	}

	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after update: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Config is not valid JSON after update: %v", err)
	}
}

// TestUpdateLastSyncTime_PreservesPermissions tests that file permissions are preserved
func TestUpdateLastSyncTime_PreservesPermissions(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Write with specific permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify initial permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config: %v", err)
	}
	initialPerms := info.Mode().Perm()

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act
	updateTime := time.Now()
	err = manager.UpdateLastSyncTime(updateTime)

	// Assert
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed: %v", err)
	}

	// Check permissions after update
	info, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config after update: %v", err)
	}

	finalPerms := info.Mode().Perm()
	if finalPerms != initialPerms {
		t.Errorf("File permissions changed: got %o, want %o", finalPerms, initialPerms)
	}
}

// TestUpdateLastSyncTime_TimezonePreserved tests that timezone is included in timestamp
func TestUpdateLastSyncTime_TimezonePreserved(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act - use a time with specific timezone
	pst := time.FixedZone("PST", -8*3600)
	updateTime := time.Date(2025, 12, 18, 10, 15, 30, 0, pst)
	err = manager.UpdateLastSyncTime(updateTime)

	// Assert
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed: %v", err)
	}

	// Read raw config to verify timezone is included
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	timestampStr, ok := result["last_sync_time"].(string)
	if !ok {
		t.Fatal("last_sync_time is not a string")
	}

	// Parse and verify timezone
	parsedTime, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	if !parsedTime.Equal(updateTime) {
		t.Errorf("Parsed time = %v, want %v", parsedTime, updateTime)
	}

	_, offset := parsedTime.Zone()
	if offset != -8*3600 {
		t.Errorf("Timezone offset = %d, want %d", offset, -8*3600)
	}
}

// TestGetLastSyncTime_BeforeLoad tests calling GetLastSyncTime before Load
func TestGetLastSyncTime_BeforeLoad(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	manager := NewManager(configPath)

	// Act
	lastSyncTime := manager.GetLastSyncTime()

	// Assert
	if lastSyncTime != nil {
		t.Errorf("GetLastSyncTime() before Load() = %v, want nil", lastSyncTime)
	}
}

// TestUpdateLastSyncTime_ConfigNotExists tests updating when config file doesn't exist
func TestUpdateLastSyncTime_ConfigNotExists(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	manager := NewManager(configPath)

	// Act
	updateTime := time.Now()
	err := manager.UpdateLastSyncTime(updateTime)

	// Assert - should create config file with last_sync_time
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() on non-existent config failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify
	manager2 := NewManager(configPath)
	_, err = manager2.Load()
	if err != nil {
		t.Fatalf("Load() after create failed: %v", err)
	}

	lastSyncTime := manager2.GetLastSyncTime()
	if lastSyncTime == nil {
		t.Fatal("GetLastSyncTime() after create returned nil")
	}

	// Compare with 1 second tolerance (RFC3339 may lose sub-second precision)
	diff := lastSyncTime.Sub(updateTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("GetLastSyncTime() after create = %v, want %v (diff: %v)", lastSyncTime, updateTime, diff)
	}
}

// TestUpdateLastSyncTime_MultipleUpdates tests multiple sequential updates
func TestUpdateLastSyncTime_MultipleUpdates(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act - perform multiple updates
	time1 := time.Date(2025, 12, 18, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2025, 12, 18, 11, 0, 0, 0, time.UTC)
	time3 := time.Date(2025, 12, 18, 12, 0, 0, 0, time.UTC)

	err = manager.UpdateLastSyncTime(time1)
	if err != nil {
		t.Fatalf("First update failed: %v", err)
	}

	err = manager.UpdateLastSyncTime(time2)
	if err != nil {
		t.Fatalf("Second update failed: %v", err)
	}

	err = manager.UpdateLastSyncTime(time3)
	if err != nil {
		t.Fatalf("Third update failed: %v", err)
	}

	// Assert - should have the latest time
	manager2 := NewManager(configPath)
	_, err = manager2.Load()
	if err != nil {
		t.Fatalf("Load() after updates failed: %v", err)
	}

	lastSyncTime := manager2.GetLastSyncTime()
	if lastSyncTime == nil {
		t.Fatal("GetLastSyncTime() after updates returned nil")
	}

	if !lastSyncTime.Equal(time3) {
		t.Errorf("GetLastSyncTime() after multiple updates = %v, want %v", lastSyncTime, time3)
	}
}

// TestManager_GetActionService returns working action service
func TestManager_GetActionService(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create minimal config
	configData := map[string]interface{}{
		"status_flow": map[string]interface{}{
			"todo":      []string{"in_progress"},
			"completed": []string{},
		},
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Clear workflow cache to force reload
	ClearWorkflowCache()

	// Act
	manager := NewManager(configPath)
	service, err := manager.GetActionService()

	// Assert
	if err != nil {
		t.Fatalf("GetActionService() failed: %v", err)
	}

	if service == nil {
		t.Fatal("GetActionService() returned nil")
	}

	// Verify service works
	ctx := context.Background()
	actions, err := service.GetAllActions(ctx)
	if err != nil {
		t.Fatalf("GetAllActions() failed: %v", err)
	}

	if actions == nil {
		t.Fatal("GetAllActions() returned nil")
	}
}

// TestLoadConfig_ObservabilityLogFile tests that manager.Load() populates LogFile
// from the observability.log_file key in .sharkconfig.json.
// This is a regression test for BUG-001: log_file was not extracted from the raw
// JSON map in the observability block, so ObservabilityConfig.LogFile was always "".
func TestLoadConfig_ObservabilityLogFile(t *testing.T) {
	tests := []struct {
		name        string
		configData  map[string]interface{}
		wantLogFile string
	}{
		{
			name: "log_file populated from JSON",
			configData: map[string]interface{}{
				"observability": map[string]interface{}{
					"enabled":  true,
					"log_file": "./shark.log",
				},
			},
			wantLogFile: "./shark.log",
		},
		{
			name: "absolute log_file path",
			configData: map[string]interface{}{
				"observability": map[string]interface{}{
					"enabled":  true,
					"log_file": "/var/log/shark.log",
				},
			},
			wantLogFile: "/var/log/shark.log",
		},
		{
			name: "empty log_file remains empty",
			configData: map[string]interface{}{
				"observability": map[string]interface{}{
					"enabled":  true,
					"log_file": "",
				},
			},
			wantLogFile: "",
		},
		{
			name: "log_file absent defaults to empty",
			configData: map[string]interface{}{
				"observability": map[string]interface{}{
					"enabled": true,
				},
			},
			wantLogFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, ".sharkconfig.json")

			data, err := json.MarshalIndent(tt.configData, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			// Act
			manager := NewManager(configPath)
			cfg, err := manager.Load()

			// Assert
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg == nil {
				t.Fatal("Load() returned nil config")
			}

			obs := cfg.GetObservability()

			if obs.LogFile != tt.wantLogFile {
				t.Errorf("manager.Load() LogFile = %q, want %q — log_file is not parsed from config JSON", obs.LogFile, tt.wantLogFile)
			}
		})
	}
}

// TestManager_Load_Maintainer tests that Manager.Load() correctly populates
// Config.Maintainer from the "maintainer" block in the config file.
// This is a regression test for the gap identified in QA: the production
// Manager.Load() path did not parse the maintainer key, so Config.Maintainer
// was always nil when loaded from a real .sharkconfig.json file.
func TestManager_Load_Maintainer(t *testing.T) {
	t.Run("maintainer block populated from JSON", func(t *testing.T) {
		// Arrange: write a config file with a maintainer block
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configData := map[string]interface{}{
			"maintainer": map[string]interface{}{
				"password_hash":        "abc123deadbeef",
				"cache_window_seconds": 120,
			},
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
		if cfg.Maintainer == nil {
			t.Fatal("Config.Maintainer is nil after Load() — Manager.Load() did not parse the maintainer block")
		}
		if cfg.Maintainer.PasswordHash != "abc123deadbeef" {
			t.Errorf("PasswordHash = %q, want %q", cfg.Maintainer.PasswordHash, "abc123deadbeef")
		}
		if cfg.Maintainer.CacheWindowSeconds != 120 {
			t.Errorf("CacheWindowSeconds = %d, want %d", cfg.Maintainer.CacheWindowSeconds, 120)
		}
	})

	t.Run("config without maintainer key leaves Maintainer nil", func(t *testing.T) {
		// Arrange: write a config file with no maintainer block
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configData := map[string]interface{}{
			"color_enabled": true,
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
		if cfg.Maintainer != nil {
			t.Errorf("Config.Maintainer = %+v, want nil when maintainer key is absent", cfg.Maintainer)
		}
	})

	t.Run("maintainer with only password_hash (no cache_window_seconds)", func(t *testing.T) {
		// Arrange: maintainer block without optional cache_window_seconds
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configData := map[string]interface{}{
			"maintainer": map[string]interface{}{
				"password_hash": "onlyhash",
			},
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Maintainer == nil {
			t.Fatal("Config.Maintainer is nil — expected non-nil for maintainer block with only password_hash")
		}
		if cfg.Maintainer.PasswordHash != "onlyhash" {
			t.Errorf("PasswordHash = %q, want %q", cfg.Maintainer.PasswordHash, "onlyhash")
		}
		if cfg.Maintainer.CacheWindowSeconds != 0 {
			t.Errorf("CacheWindowSeconds = %d, want 0 (default) when absent", cfg.Maintainer.CacheWindowSeconds)
		}
	})
}

// TestManager_Load_TagRequiredFor tests that Manager.Load() correctly populates
// Config.TagRequiredForTypes from the "tag_required_for" key in the config file.
//
// This is a regression test for the UAT-identified wiring gap (T-E28-F04-001):
// the production Manager.Load() path hand-parses rawData per-key and did NOT
// extract "tag_required_for" into Config.TagRequiredForTypes, so callers that
// reached *config.Config via cli.GetConfig() → Manager.Load() would always see
// an empty slice even when the user had set the key in .sharkconfig.json.
//
// Mirrors the pattern established by TestManager_Load_Maintainer, which
// covered the same class of bug for the maintainer block.
func TestManager_Load_TagRequiredFor(t *testing.T) {
	t.Run("tag_required_for populated from JSON", func(t *testing.T) {
		// Arrange: write a config file with a tag_required_for list
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configData := map[string]interface{}{
			"tag_required_for": []string{"task", "bug"},
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
		got := cfg.TagRequiredFor()
		want := []string{"task", "bug"}
		if len(got) != len(want) {
			t.Fatalf("TagRequiredFor() = %v (len=%d), want %v (len=%d) — Manager.Load() did not parse the tag_required_for key",
				got, len(got), want, len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("TagRequiredFor()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("config without tag_required_for key leaves TagRequiredForTypes nil", func(t *testing.T) {
		// Arrange: write a config file with no tag_required_for key
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configData := map[string]interface{}{
			"color_enabled": true,
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
		if cfg.TagRequiredForTypes != nil {
			t.Errorf("TagRequiredForTypes = %v, want nil when tag_required_for key is absent", cfg.TagRequiredForTypes)
		}
		// TagRequiredFor() accessor should also return nil/empty.
		if got := cfg.TagRequiredFor(); got != nil {
			t.Errorf("TagRequiredFor() = %v, want nil when tag_required_for key is absent", got)
		}
	})

	t.Run("empty tag_required_for array yields nil slice", func(t *testing.T) {
		// Arrange: config file with an empty array (consistent with omitempty semantics).
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		configData := map[string]interface{}{
			"tag_required_for": []string{},
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert: TagRequiredFor() returns nil for empty input (see config.go:203-205).
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got := cfg.TagRequiredFor(); got != nil {
			t.Errorf("TagRequiredFor() = %v, want nil for empty tag_required_for array", got)
		}
	})

	t.Run("non-string elements are skipped", func(t *testing.T) {
		// Arrange: config file with a tag_required_for containing a non-string
		// element. The loader should skip non-string values rather than panic,
		// consistent with the tolerant parsing approach used for other fields.
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".sharkconfig.json")

		// Build JSON manually to inject a mixed-type array.
		rawJSON := `{"tag_required_for": ["task", 42, "bug", null]}`
		if err := os.WriteFile(configPath, []byte(rawJSON), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Act
		mgr := NewManager(configPath)
		cfg, err := mgr.Load()

		// Assert: only string elements are retained, order preserved.
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		got := cfg.TagRequiredFor()
		want := []string{"task", "bug"}
		if len(got) != len(want) {
			t.Fatalf("TagRequiredFor() = %v (len=%d), want %v (len=%d)",
				got, len(got), want, len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("TagRequiredFor()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})
}

// TestManager_GetActionService_Caching returns same instance on multiple calls
func TestManager_GetActionService_Caching(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"status_flow": map[string]interface{}{
			"todo": []string{"done"},
			"done": []string{},
		},
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	ClearWorkflowCache()

	// Act
	manager := NewManager(configPath)
	service1, err := manager.GetActionService()
	if err != nil {
		t.Fatalf("First GetActionService() failed: %v", err)
	}

	service2, err := manager.GetActionService()
	if err != nil {
		t.Fatalf("Second GetActionService() failed: %v", err)
	}

	// Assert - should be same instance
	if service1 != service2 {
		t.Error("expected same service instance on multiple calls")
	}
}

// TestLoadConfig_ConsoleWidth_Present verifies that Manager.Load() populates
// Config.ConsoleWidth from the `console_width` JSON field. CC-036.
func TestLoadConfig_ConsoleWidth_Present(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"console_width": 100,
	}
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := NewManager(configPath).Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.ConsoleWidth != 100 {
		t.Errorf("ConsoleWidth = %d, want 100", cfg.ConsoleWidth)
	}
}

// TestLoadConfig_ConsoleWidth_Absent verifies that absence of console_width
// in JSON leaves ConsoleWidth at its zero value (which means "auto-detect"
// per GetConsoleWidth contract). CC-036.
func TestLoadConfig_ConsoleWidth_Absent(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"color_enabled": true,
	}
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := NewManager(configPath).Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.ConsoleWidth != 0 {
		t.Errorf("ConsoleWidth = %d, want 0 (unset → auto-detect)", cfg.ConsoleWidth)
	}
}

// --- E19-F05-001: SetSprintCapacityDefault ---

// TC-015-07: SetSprintCapacityDefault persists to .sharkconfig.json correctly.
// Production entrypoint: config.Manager.SetSprintCapacityDefault("backend", 21).
// Lowest allowed mock seam: File I/O (use temp directory with real file writes).
// Counter-factual: an impl that only mutates in-memory returns nil but the file
// is unchanged; a subsequent Load() would still see the old value (nil capacity).
func TestSetSprintCapacityDefault_PersistsToFile(t *testing.T) {
	// Arrange: write initial config to temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	initialData := map[string]interface{}{
		"color_enabled": true,
	}
	data, err := json.MarshalIndent(initialData, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal initial config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	mgr := NewManager(configPath)
	if _, err := mgr.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Act: call SetSprintCapacityDefault
	if err := mgr.SetSprintCapacityDefault("backend", 21); err != nil {
		t.Fatalf("SetSprintCapacityDefault() error = %v", err)
	}

	// Assert: read config file from disk (not in-memory) and verify persisted value
	reloadedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config after SetSprintCapacityDefault: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(reloadedData, &raw); err != nil {
		t.Fatalf("failed to unmarshal config after update: %v", err)
	}

	sprintDefaults, ok := raw["sprint_defaults"].(map[string]interface{})
	if !ok {
		t.Fatalf("sprint_defaults not found or not a map in config file; raw = %v", raw)
	}
	capacity, ok := sprintDefaults["capacity"].(map[string]interface{})
	if !ok {
		t.Fatalf("sprint_defaults.capacity not found or not a map; sprint_defaults = %v", sprintDefaults)
	}
	if capacity["backend"] != float64(21) {
		t.Errorf("sprint_defaults.capacity.backend = %v, want 21", capacity["backend"])
	}
}

// TestSetSprintCapacityDefault_CreatesSprintDefaultsSection verifies that the
// sprint_defaults section is created if absent (not just the capacity entry).
func TestSetSprintCapacityDefault_CreatesSprintDefaultsSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	initialData := `{}`
	if err := os.WriteFile(configPath, []byte(initialData), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	mgr := NewManager(configPath)
	if _, err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := mgr.SetSprintCapacityDefault("frontend", 13); err != nil {
		t.Fatalf("SetSprintCapacityDefault() error = %v", err)
	}

	// Load again from disk using a fresh manager
	mgr2 := NewManager(configPath)
	cfg2, err := mgr2.Load()
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if cfg2.SprintDefaults == nil {
		t.Fatal("SprintDefaults must not be nil after SetSprintCapacityDefault")
	}
	if cfg2.SprintDefaults.Capacity["frontend"] != 13 {
		t.Errorf("Capacity[frontend] = %v, want 13", cfg2.SprintDefaults.Capacity["frontend"])
	}
}

// TestSetSprintCapacityDefault_UpdatesExistingEntry verifies that calling
// SetSprintCapacityDefault on an already-set agent type updates the value (no duplicates).
func TestSetSprintCapacityDefault_UpdatesExistingEntry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	initialData := `{
		"sprint_defaults": {
			"capacity": {"backend": 10}
		}
	}`
	if err := os.WriteFile(configPath, []byte(initialData), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	mgr := NewManager(configPath)
	if _, err := mgr.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Update existing entry
	if err := mgr.SetSprintCapacityDefault("backend", 21); err != nil {
		t.Fatalf("SetSprintCapacityDefault() error = %v", err)
	}

	// Read from disk
	rawBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config error = %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}
	sprintDefaults := raw["sprint_defaults"].(map[string]interface{})
	capacity := sprintDefaults["capacity"].(map[string]interface{})
	if capacity["backend"] != float64(21) {
		t.Errorf("capacity.backend = %v, want 21 after update", capacity["backend"])
	}
}

// --- TD-016: SaveRaw round-trip helper ---

// TestSaveRaw_CreatesNewFile verifies SaveRaw can write to a path that does
// not yet exist. This is the "fresh init" code path for ensureWorkflowConfigField.
func TestSaveRaw_CreatesNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	mgr := NewManager(configPath)
	raw := map[string]interface{}{
		"workflow_config": "shark-data/workflow/",
	}
	if err := mgr.SaveRaw(configPath, raw); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}

	// File must now exist and be valid JSON.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read after SaveRaw: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	if parsed["workflow_config"] != "shark-data/workflow/" {
		t.Errorf("workflow_config = %v, want %q", parsed["workflow_config"], "shark-data/workflow/")
	}
}

// TestSaveRaw_PreservesUnknownKeys ensures that round-tripping arbitrary
// top-level keys through SaveRaw does not drop them. The whole motivation
// for TD-016 was that the old inline path used a generic map and so did
// the round-trip in Manager.Load — this test pins that the new helper
// keeps that property.
func TestSaveRaw_PreservesUnknownKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Seed with a mix of known and unknown top-level keys.
	seed := map[string]interface{}{
		"color_enabled":           true,
		"workflow_config":         "shark-data/workflow/",
		"some_third_party_plugin": map[string]interface{}{"version": "1.2.3"},
		"opaque_string":           "carry this through verbatim",
	}
	seedBytes, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	if err := os.WriteFile(configPath, seedBytes, 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	// Load → mutate → SaveRaw.
	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	cfg.RawData["workflow_config"] = "shark-data/workflow-v2/"
	if err := mgr.SaveRaw(configPath, cfg.RawData); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}

	// Read back from disk; all original keys must survive, with workflow_config
	// updated.
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read after SaveRaw: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if parsed["color_enabled"] != true {
		t.Errorf("color_enabled lost or wrong: got %v", parsed["color_enabled"])
	}
	if parsed["workflow_config"] != "shark-data/workflow-v2/" {
		t.Errorf("workflow_config = %v, want %q", parsed["workflow_config"], "shark-data/workflow-v2/")
	}
	plug, ok := parsed["some_third_party_plugin"].(map[string]interface{})
	if !ok {
		t.Fatalf("some_third_party_plugin lost or wrong type: %v", parsed["some_third_party_plugin"])
	}
	if plug["version"] != "1.2.3" {
		t.Errorf("plugin.version = %v, want %q", plug["version"], "1.2.3")
	}
	if parsed["opaque_string"] != "carry this through verbatim" {
		t.Errorf("opaque_string = %v, want unchanged", parsed["opaque_string"])
	}
}

// TestSaveRaw_AtomicWriteNoTempLeaked verifies SaveRaw uses the temp-then-rename
// pattern and does not leave a .tmp sibling behind on success. Counter-factual:
// a naive os.WriteFile path would never produce a .tmp file in the first place,
// but it would also be vulnerable to torn writes on crash. The .tmp absence
// after success is the on-disk proof we took the atomic path.
func TestSaveRaw_AtomicWriteNoTempLeaked(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	mgr := NewManager(configPath)
	if err := mgr.SaveRaw(configPath, map[string]interface{}{"k": "v"}); err != nil {
		t.Fatalf("SaveRaw(): %v", err)
	}

	if _, err := os.Stat(configPath + ".tmp"); err == nil {
		t.Error("temp file leaked after successful SaveRaw")
	}
}

// TestSaveRaw_PreservesFilePermissions verifies that SaveRaw retains the
// existing file's permission bits on rewrite. Matches the behavior of
// UpdateLastSyncTime so config files keep their mode across mutations.
func TestSaveRaw_PreservesFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Write with a non-default mode so we can detect whether SaveRaw drops it.
	if err := os.WriteFile(configPath, []byte("{}\n"), 0600); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat seed: %v", err)
	}
	initialPerms := info.Mode().Perm()

	mgr := NewManager(configPath)
	if err := mgr.SaveRaw(configPath, map[string]interface{}{"k": "v"}); err != nil {
		t.Fatalf("SaveRaw(): %v", err)
	}

	info, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat after SaveRaw: %v", err)
	}
	if got := info.Mode().Perm(); got != initialPerms {
		t.Errorf("perms = %o, want %o (SaveRaw must preserve existing permissions)", got, initialPerms)
	}
}

// TestSaveRaw_NilRawDataWritesEmptyObject verifies SaveRaw treats a nil
// rawData map as an empty JSON object, matching the defensive
// initialization in writeRawConfig. Prevents NPE-style failures from
// callers that pass cfg.RawData before any keys have been added.
func TestSaveRaw_NilRawDataWritesEmptyObject(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	mgr := NewManager(configPath)
	if err := mgr.SaveRaw(configPath, nil); err != nil {
		t.Fatalf("SaveRaw(nil) error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read after SaveRaw: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("file not valid JSON after SaveRaw(nil): %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("SaveRaw(nil) wrote %d keys, want 0 (empty object)", len(parsed))
	}
}
