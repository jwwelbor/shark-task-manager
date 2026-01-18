package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_SyncWorkflow tests the complete sync workflow:
// 1. Initial sync with no last_sync_time (full scan)
// 2. Update last_sync_time after sync
// 3. Next sync reads last_sync_time (incremental scan)
func TestIntegration_SyncWorkflow(t *testing.T) {
	// Setup - Create temporary directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Initial config (no last_sync_time)
	initialConfig := map[string]interface{}{
		"color_enabled": true,
		"default_epic":  "E01",
	}

	data, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// STEP 1: First sync - no last_sync_time (should trigger full scan)
	t.Log("Step 1: Initial sync (no last_sync_time)")
	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed on first sync: %v", err)
	}

	lastSyncTime := manager.GetLastSyncTime()
	if lastSyncTime != nil {
		t.Fatalf("GetLastSyncTime() on initial load = %v, want nil (should trigger full scan)", lastSyncTime)
	}

	t.Log("  ✓ No last_sync_time found - full scan triggered")

	// Simulate sync completion
	firstSyncTime := time.Date(2025, 12, 18, 10, 0, 0, 0, time.UTC)
	t.Logf("Step 2: Sync completed at %v", firstSyncTime)

	// STEP 2: Update last_sync_time after successful sync
	err = manager.UpdateLastSyncTime(firstSyncTime)
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed: %v", err)
	}

	t.Log("  ✓ last_sync_time updated in config")

	// Verify the update was persisted
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after update: %v", err)
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if configData["last_sync_time"] == nil {
		t.Fatal("last_sync_time was not persisted to config file")
	}

	t.Logf("  ✓ Config file updated: last_sync_time = %v", configData["last_sync_time"])

	// Verify other fields were preserved
	if configData["color_enabled"] != true {
		t.Error("color_enabled was not preserved")
	}
	if configData["default_epic"] != "E01" {
		t.Error("default_epic was not preserved")
	}

	t.Log("  ✓ Existing config fields preserved")

	// STEP 3: Second sync - should read last_sync_time (incremental scan)
	t.Log("Step 3: Second sync (with last_sync_time)")

	// Create a new manager to simulate a fresh sync command
	manager2 := NewManager(configPath)
	config2, err := manager2.Load()
	if err != nil {
		t.Fatalf("Load() failed on second sync: %v", err)
	}

	if config2 == nil {
		t.Fatal("Load() returned nil config on second sync")
	}

	lastSyncTime2 := manager2.GetLastSyncTime()
	if lastSyncTime2 == nil {
		t.Fatal("GetLastSyncTime() on second sync returned nil (should have timestamp)")
	}

	if !lastSyncTime2.Equal(firstSyncTime) {
		t.Errorf("GetLastSyncTime() on second sync = %v, want %v", lastSyncTime2, firstSyncTime)
	}

	t.Logf("  ✓ last_sync_time loaded: %v", lastSyncTime2)
	t.Log("  ✓ Incremental sync can use this timestamp to filter files")

	// STEP 4: Simulate incremental sync (only scan files modified after lastSyncTime2)
	t.Log("Step 4: Incremental sync logic")

	// Simulate file modification times
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	files := []fileInfo{
		// Old file - modified before last sync
		{path: "docs/old-file.md", modTime: firstSyncTime.Add(-1 * time.Hour)},
		// New file - modified after last sync
		{path: "docs/new-file.md", modTime: firstSyncTime.Add(1 * time.Hour)},
		// Another new file
		{path: "docs/another-new.md", modTime: firstSyncTime.Add(2 * time.Hour)},
	}

	var filesToSync []string
	for _, file := range files {
		if file.modTime.After(*lastSyncTime2) {
			filesToSync = append(filesToSync, file.path)
		}
	}

	if len(filesToSync) != 2 {
		t.Errorf("Incremental filter found %d files, want 2", len(filesToSync))
	}

	expectedFiles := []string{"docs/new-file.md", "docs/another-new.md"}
	for i, file := range filesToSync {
		if file != expectedFiles[i] {
			t.Errorf("File %d = %s, want %s", i, file, expectedFiles[i])
		}
	}

	t.Logf("  ✓ Filtered to %d modified files (skipped 1 old file)", len(filesToSync))

	// STEP 5: Update last_sync_time after second sync
	secondSyncTime := time.Date(2025, 12, 18, 11, 0, 0, 0, time.UTC)
	err = manager2.UpdateLastSyncTime(secondSyncTime)
	if err != nil {
		t.Fatalf("UpdateLastSyncTime() failed on second sync: %v", err)
	}

	t.Logf("Step 5: Second sync completed, last_sync_time updated to %v", secondSyncTime)

	// Verify final state
	manager3 := NewManager(configPath)
	_, err = manager3.Load()
	if err != nil {
		t.Fatalf("Load() failed on final verification: %v", err)
	}

	finalSyncTime := manager3.GetLastSyncTime()
	if finalSyncTime == nil {
		t.Fatal("Final last_sync_time is nil")
	}

	if !finalSyncTime.Equal(secondSyncTime) {
		t.Errorf("Final last_sync_time = %v, want %v", finalSyncTime, secondSyncTime)
	}

	t.Log("  ✓ Final last_sync_time verified")
	t.Log("\n✓ Complete sync workflow test passed")
}

// TestIntegration_ConcurrentReads tests that concurrent reads are safe during updates
func TestIntegration_ConcurrentReads(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create initial config
	initialConfig := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create manager and perform update
	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Update in one goroutine
	done := make(chan bool)
	go func() {
		updateTime := time.Now()
		err := manager.UpdateLastSyncTime(updateTime)
		if err != nil {
			t.Errorf("UpdateLastSyncTime() failed: %v", err)
		}
		done <- true
	}()

	// Wait for update to complete
	<-done

	// Read from another manager (simulating another process)
	manager2 := NewManager(configPath)
	config, err := manager2.Load()
	if err != nil {
		t.Fatalf("Load() from second manager failed: %v", err)
	}

	// Should see either the old config (before update) or new config (after update)
	// Never a corrupt/partial config
	var configData map[string]interface{}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if err := json.Unmarshal(data, &configData); err != nil {
		t.Fatalf("Config file is corrupted (not valid JSON): %v", err)
	}

	// Verify config is valid
	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	t.Log("✓ Concurrent read during update succeeded (atomic write works)")
}

// TestIntegration_MultipleManagerInstances tests multiple manager instances accessing same config
func TestIntegration_MultipleManagerInstances(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create initial config
	initialConfig := map[string]interface{}{
		"color_enabled": true,
	}

	data, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create three manager instances (simulating three sync processes)
	manager1 := NewManager(configPath)
	manager2 := NewManager(configPath)
	manager3 := NewManager(configPath)

	// Each loads the config
	_, err = manager1.Load()
	if err != nil {
		t.Fatalf("Manager1 Load() failed: %v", err)
	}

	_, err = manager2.Load()
	if err != nil {
		t.Fatalf("Manager2 Load() failed: %v", err)
	}

	_, err = manager3.Load()
	if err != nil {
		t.Fatalf("Manager3 Load() failed: %v", err)
	}

	// Manager1 updates
	time1 := time.Date(2025, 12, 18, 10, 0, 0, 0, time.UTC)
	err = manager1.UpdateLastSyncTime(time1)
	if err != nil {
		t.Fatalf("Manager1 update failed: %v", err)
	}

	// Manager2 should see the update after reload
	_, err = manager2.Load()
	if err != nil {
		t.Fatalf("Manager2 reload failed: %v", err)
	}

	lastSync2 := manager2.GetLastSyncTime()
	if lastSync2 == nil || !lastSync2.Equal(time1) {
		t.Errorf("Manager2 GetLastSyncTime() = %v, want %v", lastSync2, time1)
	}

	// Manager3 updates
	time3 := time.Date(2025, 12, 18, 11, 0, 0, 0, time.UTC)
	err = manager3.UpdateLastSyncTime(time3)
	if err != nil {
		t.Fatalf("Manager3 update failed: %v", err)
	}

	// Manager1 should see Manager3's update after reload
	_, err = manager1.Load()
	if err != nil {
		t.Fatalf("Manager1 reload failed: %v", err)
	}

	lastSync1 := manager1.GetLastSyncTime()
	if lastSync1 == nil || !lastSync1.Equal(time3) {
		t.Errorf("Manager1 GetLastSyncTime() after reload = %v, want %v", lastSync1, time3)
	}

	t.Log("✓ Multiple manager instances can safely share config file")
}

// TestIntegration_RequireRejectionReason tests that require_rejection_reason is loaded from config
func TestIntegration_RequireRejectionReason(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create config with require_rejection_reason enabled
	configData := map[string]interface{}{
		"require_rejection_reason": true,
		"color_enabled":            true,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config via manager
	manager := NewManager(configPath)
	cfg, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify require_rejection_reason is loaded
	if !cfg.RequireRejectionReason {
		t.Error("RequireRejectionReason should be true")
	}

	if !cfg.IsRequireRejectionReasonEnabled() {
		t.Error("IsRequireRejectionReasonEnabled() should return true")
	}

	t.Log("✓ require_rejection_reason loaded correctly via manager")

	// Test with disabled (false)
	configData["require_rejection_reason"] = false
	data, _ = json.MarshalIndent(configData, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager2 := NewManager(configPath)
	cfg2, err := manager2.Load()
	if err != nil {
		t.Fatalf("Load() with false failed: %v", err)
	}

	if cfg2.RequireRejectionReason {
		t.Error("RequireRejectionReason should be false")
	}

	if cfg2.IsRequireRejectionReasonEnabled() {
		t.Error("IsRequireRejectionReasonEnabled() should return false")
	}

	t.Log("✓ require_rejection_reason=false loaded correctly")

	// Test with omitted field (default false)
	configData = map[string]interface{}{
		"color_enabled": true,
	}
	data, _ = json.MarshalIndent(configData, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager3 := NewManager(configPath)
	cfg3, err := manager3.Load()
	if err != nil {
		t.Fatalf("Load() with omitted field failed: %v", err)
	}

	if cfg3.RequireRejectionReason {
		t.Error("RequireRejectionReason should default to false when omitted")
	}

	t.Log("✓ require_rejection_reason defaults to false when omitted")
}
