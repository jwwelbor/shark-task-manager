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
