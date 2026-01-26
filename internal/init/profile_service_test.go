package init

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to create a temporary config file with content
// If content is nil, still creates an empty config file
func createTempConfigFile(t *testing.T, content map[string]interface{}) string {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	if content == nil {
		content = make(map[string]interface{})
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal test content: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	return configPath
}

// Helper to read config file content
func readConfigFile(t *testing.T, path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var content map[string]interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	return content
}

// Test 1: Apply basic profile to empty config
func TestApplyProfile_EmptyConfig(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil UpdateResult")
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.ProfileName != "basic" {
		t.Errorf("expected ProfileName = 'basic', got %q", result.ProfileName)
	}

	if result.DryRun {
		t.Error("expected DryRun to be false")
	}

	// Verify file was written
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}

	// Verify status metadata was added
	content := readConfigFile(t, configPath)
	if _, ok := content["status_metadata"]; !ok {
		t.Error("expected status_metadata to be present")
	}
}

// Test 2: Apply advanced profile to empty config
func TestApplyProfile_AdvancedProfile(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.ProfileName != "advanced" {
		t.Errorf("expected ProfileName = 'advanced', got %q", result.ProfileName)
	}

	content := readConfigFile(t, configPath)

	// Verify advanced profile fields
	if _, ok := content["status_flow"]; !ok {
		t.Error("expected status_flow to be present in advanced profile")
	}

	if _, ok := content["special_statuses"]; !ok {
		t.Error("expected special_statuses to be present in advanced profile")
	}
}

// Test 3: Dry-run mode doesn't write files
func TestApplyProfile_DryRun(t *testing.T) {
	// Start with empty config
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       true,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if !result.DryRun {
		t.Error("expected DryRun to be true in result")
	}

	if result.BackupPath != "" {
		t.Error("expected no backup in dry-run mode")
	}

	// Verify file was not written (should still be empty)
	content := readConfigFile(t, configPath)
	if _, ok := content["status_metadata"]; ok {
		t.Error("expected status_metadata NOT to be written in dry-run mode")
	}
}

// Test 4: Database config is preserved
func TestApplyProfile_PreserveDatabase(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "turso",
			"url":     "libsql://example.turso.io",
		},
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	content := readConfigFile(t, configPath)

	// Verify database config was preserved
	db, ok := content["database"].(map[string]interface{})
	if !ok {
		t.Fatal("expected database config in result")
	}

	if db["backend"] != "turso" {
		t.Errorf("expected database.backend = 'turso', got %v", db["backend"])
	}

	if db["url"] != "libsql://example.turso.io" {
		t.Errorf("expected database.url = 'libsql://example.turso.io', got %v", db["url"])
	}

	// Verify status metadata was added
	if _, ok := content["status_metadata"]; !ok {
		t.Error("expected status_metadata to be present")
	}
}

// Test 5: Backup is created before writing
func TestApplyProfile_BackupCreated(t *testing.T) {
	existingConfig := map[string]interface{}{
		"color_enabled": true,
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if result.BackupPath == "" {
		t.Error("expected BackupPath to be set")
	}

	// Verify backup file exists
	if result.BackupPath != "" {
		if _, err := os.Stat(result.BackupPath); err != nil {
			t.Errorf("backup file not found: %v", err)
		}
		defer os.Remove(result.BackupPath)

		// Verify backup contains original content
		backupContent := readConfigFile(t, result.BackupPath)
		if backupContent["color_enabled"] != true {
			t.Error("expected backup to contain original color_enabled")
		}
	}
}

// Test 6: Custom fields are preserved
func TestApplyProfile_PreserveCustomFields(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
		"custom_field": "custom_value",
		"project_root": "/path/to/project",
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	_, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	content := readConfigFile(t, configPath)

	// Verify custom fields preserved
	if content["custom_field"] != "custom_value" {
		t.Errorf("expected custom_field = 'custom_value', got %v", content["custom_field"])
	}

	if content["project_root"] != "/path/to/project" {
		t.Errorf("expected project_root = '/path/to/project', got %v", content["project_root"])
	}
}

// Test 7: Add missing fields only (no workflow specified)
func TestAddMissingFields(t *testing.T) {
	existingConfig := map[string]interface{}{
		"color_enabled": true,
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "", // No workflow name
		Force:        false,
		DryRun:       false,
	}

	_, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	content := readConfigFile(t, configPath)

	// Verify original fields preserved
	if content["color_enabled"] != true {
		t.Error("expected color_enabled to be preserved")
	}

	// Verify status_metadata was added
	if _, ok := content["status_metadata"]; !ok {
		t.Error("expected status_metadata to be added")
	}
}

// Test 8: Invalid profile name returns error
func TestApplyProfile_InvalidProfile(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "nonexistent",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)

	if err == nil {
		t.Fatal("expected error for invalid profile")
	}

	if !strings.Contains(err.Error(), "profile not found") {
		t.Errorf("expected 'profile not found' error, got: %v", err)
	}

	if result != nil {
		t.Error("expected nil result on error")
	}
}

// Test 9: GetChangePreview works without writing
func TestGetChangePreview(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	changeReport, err := service.GetChangePreview(opts)
	if err != nil {
		t.Fatalf("GetChangePreview() error = %v", err)
	}

	if changeReport == nil {
		t.Fatal("expected non-nil ChangeReport")
	}

	// Verify file was not written
	content := readConfigFile(t, configPath)
	if _, ok := content["status_metadata"]; ok {
		t.Error("expected status_metadata NOT to be written")
	}

	// Verify change report has data
	if changeReport.Stats == nil {
		t.Error("expected Stats in ChangeReport")
	}
}

// Test 10: Atomic write prevents corruption
func TestApplyProfile_AtomicWrite(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	// Verify config is valid JSON (not corrupted)
	content := readConfigFile(t, configPath)

	// Verify all expected fields are present
	if _, ok := content["status_metadata"]; !ok {
		t.Error("expected status_metadata in final config")
	}

	if _, ok := content["database"]; !ok {
		t.Error("expected database in final config")
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
	}
}

// Test 11: Force mode overwrites status metadata
func TestApplyProfile_ForceMode(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
		"status_metadata": map[string]interface{}{
			"custom_status": map[string]interface{}{
				"color": "blue",
			},
		},
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        true,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	content := readConfigFile(t, configPath)

	// Verify status_metadata was overwritten with profile's metadata
	statusMeta, ok := content["status_metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected status_metadata to be a map")
	}

	// Basic profile should have "todo" status
	if _, ok := statusMeta["todo"]; !ok {
		t.Error("expected 'todo' status from basic profile")
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
	}
}

// Test 12: Profile to map conversion
func TestProfileToMap_BasicProfile(t *testing.T) {
	profile, _ := GetProfile("basic")

	m := profileToMap(profile)

	if m == nil {
		t.Fatal("expected non-nil map")
	}

	if _, ok := m["status_metadata"]; !ok {
		t.Error("expected status_metadata in map")
	}

	if _, ok := m["color_enabled"]; !ok {
		t.Error("expected color_enabled in map")
	}
}

// Test 13: Profile to map with advanced profile
func TestProfileToMap_AdvancedProfile(t *testing.T) {
	profile, _ := GetProfile("advanced")

	m := profileToMap(profile)

	if m == nil {
		t.Fatal("expected non-nil map")
	}

	if _, ok := m["status_flow"]; !ok {
		t.Error("expected status_flow in map for advanced profile")
	}

	if _, ok := m["special_statuses"]; !ok {
		t.Error("expected special_statuses in map for advanced profile")
	}

	if m["status_flow_version"] != "1.0" {
		t.Errorf("expected status_flow_version = '1.0', got %v", m["status_flow_version"])
	}
}

// Test 14: Missing config file is created gracefully
func TestApplyProfile_MissingConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	// Don't create the file

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	// Verify file was created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
	}
}

// Test 15: Change report contains accurate statistics
func TestGetChangePreview_ChangeStats(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
		},
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	changeReport, err := service.GetChangePreview(opts)
	if err != nil {
		t.Fatalf("GetChangePreview() error = %v", err)
	}

	if changeReport == nil {
		t.Fatal("expected non-nil ChangeReport")
	}

	if changeReport.Stats == nil {
		t.Fatal("expected non-nil Stats")
	}

	if changeReport.Stats.StatusesAdded < 4 {
		t.Errorf("expected StatusesAdded >= 4 for basic profile, got %d", changeReport.Stats.StatusesAdded)
	}
}

// Test 16: WriteConfig creates atomic temp file
func TestWriteConfig_Atomic(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := NewProfileService(configPath)

	testData := map[string]interface{}{
		"color_enabled":   true,
		"status_metadata": map[string]interface{}{},
	}

	err := service.writeConfig(configPath, testData)
	if err != nil {
		t.Fatalf("writeConfig() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}

	// Verify content is valid JSON
	content := readConfigFile(t, configPath)
	if content["color_enabled"] != true {
		t.Error("expected color_enabled to be true")
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("found leftover temp file: %s", entry.Name())
		}
	}
}

// Test 17: CreateConfigBackup generates timestamped file
func TestCreateConfigBackup_Timestamp(t *testing.T) {
	existingConfig := map[string]interface{}{
		"color_enabled": true,
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)
	backupPath, err := service.createConfigBackup(configPath)
	if err != nil {
		t.Fatalf("createConfigBackup() error = %v", err)
	}

	if backupPath == "" {
		t.Fatal("expected non-empty backup path")
	}

	defer os.Remove(backupPath)

	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not found: %v", err)
	}

	// Verify backup contains ".backup." in name
	if !strings.Contains(backupPath, ".backup.") {
		t.Errorf("expected .backup. in backup filename, got %s", backupPath)
	}

	// Verify backup contains original content
	backupContent := readConfigFile(t, backupPath)
	if backupContent["color_enabled"] != true {
		t.Error("expected backup to contain original content")
	}
}

// Test 18: CreateConfigBackup returns empty string for missing file
func TestCreateConfigBackup_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	// Don't create the file

	service := NewProfileService(configPath)
	backupPath, err := service.createConfigBackup(configPath)
	if err != nil {
		t.Fatalf("createConfigBackup() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("expected empty backup path for missing file, got %s", backupPath)
	}
}

// Test 19: Multiple profile applications preserve state correctly
func TestApplyProfile_Sequential(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
			"url":     "./shark-tasks.db",
		},
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	// First apply basic profile
	opts1 := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result1, err := service.ApplyProfile(opts1)
	if err != nil {
		t.Fatalf("first ApplyProfile() error = %v", err)
	}

	if result1.BackupPath != "" {
		defer os.Remove(result1.BackupPath)
	}

	// Then apply advanced profile (force)
	opts2 := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		Force:        true,
		DryRun:       false,
	}

	result2, err := service.ApplyProfile(opts2)
	if err != nil {
		t.Fatalf("second ApplyProfile() error = %v", err)
	}

	if result2.ProfileName != "advanced" {
		t.Errorf("expected ProfileName = 'advanced', got %q", result2.ProfileName)
	}

	content := readConfigFile(t, configPath)

	// Verify advanced profile was applied
	if _, ok := content["status_flow"]; !ok {
		t.Error("expected status_flow from advanced profile")
	}

	// Verify database config still preserved
	if _, ok := content["database"]; !ok {
		t.Error("expected database config to be preserved")
	}

	if result2.BackupPath != "" {
		defer os.Remove(result2.BackupPath)
	}
}

// Test 20: Dry-run with existing config shows accurate preview
func TestApplyProfile_DryRunWithExisting(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "turso",
			"url":     "libsql://example.turso.io",
		},
		"color_enabled": true,
	}

	configPath := createTempConfigFile(t, existingConfig)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		Force:        false,
		DryRun:       true,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if !result.DryRun {
		t.Error("expected DryRun to be true")
	}

	// Verify original file unchanged
	originalContent := readConfigFile(t, configPath)
	if colorEnabled, ok := originalContent["color_enabled"]; !ok || colorEnabled != true {
		t.Error("expected original color_enabled to remain in dry-run")
	}

	// Verify change report indicates what would happen
	if result.Changes == nil {
		t.Error("expected non-nil Changes in dry-run result")
	}
}
