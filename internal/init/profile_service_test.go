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

// Test 12: GetProfileMap for basic profile
func TestGetProfileMap_BasicHasStatusMetadata(t *testing.T) {
	m, err := GetProfileMap("basic")
	if err != nil {
		t.Fatalf("GetProfileMap('basic') error = %v", err)
	}

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

// Test 13: GetProfileMap for advanced profile
func TestGetProfileMap_AdvancedHasWorkflowFields(t *testing.T) {
	m, err := GetProfileMap("advanced")
	if err != nil {
		t.Fatalf("GetProfileMap('advanced') error = %v", err)
	}

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

// Test: Advanced profile includes epic_workflow in merged config
func TestApplyProfile_AdvancedIncludesEpicWorkflow(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		Force:        true,
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

	if _, ok := content["epic_workflow"]; !ok {
		t.Error("expected epic_workflow to be present in advanced profile merge")
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
	}
}

// Test: Advanced profile includes feature_workflow in merged config
func TestApplyProfile_AdvancedIncludesFeatureWorkflow(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	defer os.Remove(configPath)

	service := NewProfileService(configPath)

	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		Force:        true,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	content := readConfigFile(t, configPath)

	if _, ok := content["feature_workflow"]; !ok {
		t.Error("expected feature_workflow to be present in advanced profile merge")
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
	}
}

// Test: Basic profile doesn't add epic_workflow
func TestApplyProfile_BasicNoEpicWorkflow(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
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

	if _, ok := content["epic_workflow"]; ok {
		t.Error("basic profile should not include epic_workflow")
	}
	if _, ok := content["feature_workflow"]; ok {
		t.Error("basic profile should not include feature_workflow")
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
	}
}

// Test: interactive_mode and require_rejection_reason are preserved
func TestApplyProfile_PreservesProjectFields(t *testing.T) {
	existingConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"backend": "local",
		},
		"interactive_mode":         false,
		"require_rejection_reason": true,
		"last_sync_time":           "2026-01-01T00:00:00Z",
	}

	configPath := createTempConfigFile(t, existingConfig)
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

	content := readConfigFile(t, configPath)

	// All project fields should be preserved
	if content["interactive_mode"] != false {
		t.Errorf("expected interactive_mode to be preserved as false, got %v", content["interactive_mode"])
	}
	if content["require_rejection_reason"] != true {
		t.Errorf("expected require_rejection_reason to be preserved as true, got %v", content["require_rejection_reason"])
	}
	if content["last_sync_time"] != "2026-01-01T00:00:00Z" {
		t.Errorf("expected last_sync_time to be preserved, got %v", content["last_sync_time"])
	}

	if result.BackupPath != "" {
		defer os.Remove(result.BackupPath)
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

// ===== Workflow file generation tests (E20-F05) =====

// TestApplyProfile_GeneratesWorkflowFile verifies that applying a workflow profile
// generates a .sharkworkflow.json file with all 5 entity workflow blocks.
func TestApplyProfile_GeneratesWorkflowFile(t *testing.T) {
	configPath := createTempConfigFile(t, nil)

	service := NewProfileService(configPath)
	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	// Verify workflow file path is set in result
	if result.WorkflowFilePath == "" {
		t.Fatal("expected WorkflowFilePath to be set")
	}

	// Verify workflow file was created
	wfPath := filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")
	data, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("workflow file not created: %v", err)
	}

	var wfData map[string]interface{}
	if err := json.Unmarshal(data, &wfData); err != nil {
		t.Fatalf("invalid JSON in workflow file: %v", err)
	}

	// Verify all 5 entity workflow blocks exist
	requiredKeys := []string{"epic_workflow", "feature_workflow", "task_workflow", "bug_workflow", "change_workflow"}
	for _, key := range requiredKeys {
		if _, ok := wfData[key]; !ok {
			t.Errorf("missing required key %q in workflow file", key)
		}
	}
}

// TestApplyProfile_BasicGeneratesWorkflowFile verifies basic profile also generates workflow file.
func TestApplyProfile_BasicGeneratesWorkflowFile(t *testing.T) {
	configPath := createTempConfigFile(t, nil)

	service := NewProfileService(configPath)
	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if result.WorkflowFilePath == "" {
		t.Fatal("expected WorkflowFilePath to be set")
	}

	wfPath := filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")
	data, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("workflow file not created: %v", err)
	}

	var wfData map[string]interface{}
	if err := json.Unmarshal(data, &wfData); err != nil {
		t.Fatalf("invalid JSON in workflow file: %v", err)
	}

	// Basic profile should still have task_workflow (constructed from legacy keys)
	if _, ok := wfData["task_workflow"]; !ok {
		t.Error("missing task_workflow in basic workflow file")
	}
}

// TestApplyProfile_WorkflowFileDryRun verifies dry-run does not create workflow file.
func TestApplyProfile_WorkflowFileDryRun(t *testing.T) {
	configPath := createTempConfigFile(t, nil)

	service := NewProfileService(configPath)
	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		DryRun:       true,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	if !result.DryRun {
		t.Error("expected DryRun to be true")
	}

	// WorkflowFilePath should be set (for preview purposes)
	if result.WorkflowFilePath == "" {
		t.Error("expected WorkflowFilePath to be set even in dry-run")
	}

	// But the file should NOT be created
	wfPath := filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")
	if _, err := os.Stat(wfPath); !os.IsNotExist(err) {
		t.Error("workflow file should not be created in dry-run mode")
	}
}

// TestApplyProfile_WorkflowFileBackup verifies existing workflow file gets backed up.
func TestApplyProfile_WorkflowFileBackup(t *testing.T) {
	configPath := createTempConfigFile(t, nil)
	wfPath := filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")

	// Create an existing workflow file
	existingContent := map[string]interface{}{
		"task_workflow": map[string]interface{}{
			"version": "old",
		},
	}
	existingData, _ := json.Marshal(existingContent)
	if err := os.WriteFile(wfPath, existingData, 0644); err != nil {
		t.Fatalf("failed to write existing workflow file: %v", err)
	}

	service := NewProfileService(configPath)
	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	// Verify backup was created
	if result.WorkflowBackupPath == "" {
		t.Fatal("expected WorkflowBackupPath to be set")
	}

	if !strings.HasPrefix(result.WorkflowBackupPath, wfPath+".backup.") {
		t.Errorf("unexpected backup path: %s", result.WorkflowBackupPath)
	}

	// Verify backup contains old content
	backupData, err := os.ReadFile(result.WorkflowBackupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}

	var backupContent map[string]interface{}
	if err := json.Unmarshal(backupData, &backupContent); err != nil {
		t.Fatalf("invalid JSON in backup: %v", err)
	}

	tw, ok := backupContent["task_workflow"].(map[string]interface{})
	if !ok {
		t.Fatal("backup missing task_workflow")
	}
	if tw["version"] != "old" {
		t.Errorf("backup has wrong version: %v", tw["version"])
	}
}

// TestApplyProfile_WorkflowFileTaskBlock verifies task_workflow uses block format.
func TestApplyProfile_WorkflowFileTaskBlock(t *testing.T) {
	configPath := createTempConfigFile(t, nil)

	service := NewProfileService(configPath)
	opts := UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
	}

	_, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	wfPath := filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")
	data, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	var wfData map[string]interface{}
	if err := json.Unmarshal(data, &wfData); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// task_workflow should be a block (map), not a scalar
	tw, ok := wfData["task_workflow"].(map[string]interface{})
	if !ok {
		t.Fatal("task_workflow should be a map (block format)")
	}

	// Verify it has expected sub-keys
	expectedSubKeys := []string{"status_flow", "status_metadata"}
	for _, key := range expectedSubKeys {
		if _, ok := tw[key]; !ok {
			t.Errorf("task_workflow missing sub-key %q", key)
		}
	}

	// Verify NO legacy top-level keys in the workflow file
	legacyKeys := []string{"status_flow", "status_metadata", "special_statuses", "status_flow_version"}
	for _, key := range legacyKeys {
		if _, ok := wfData[key]; ok {
			t.Errorf("workflow file should not have legacy top-level key %q", key)
		}
	}
}

// TestApplyProfile_NoWorkflowFileWithoutProfileName verifies that when no workflow name
// is specified, no workflow file is generated (only config is updated).
func TestApplyProfile_NoWorkflowFileWithoutProfileName(t *testing.T) {
	configPath := createTempConfigFile(t, nil)

	service := NewProfileService(configPath)
	opts := UpdateOptions{
		ConfigPath: configPath,
		// WorkflowName intentionally empty
	}

	_, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile() error = %v", err)
	}

	// Workflow file should NOT be created when no workflow name specified
	wfPath := filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")
	if _, err := os.Stat(wfPath); !os.IsNotExist(err) {
		t.Error("workflow file should not be created without explicit workflow name")
	}
}

// Test: writeConfig preserves existing file permissions
func TestWriteConfig_PreservesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := NewProfileService(configPath)

	testData := map[string]interface{}{
		"color_enabled": true,
	}

	t.Run("preserves existing non-default permissions", func(t *testing.T) {
		// Create file with restrictive permissions (0600)
		initialContent := []byte(`{"initial": true}`)
		if err := os.WriteFile(configPath, initialContent, 0600); err != nil {
			t.Fatalf("failed to create initial config: %v", err)
		}

		// Verify initial permissions
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("failed to stat initial file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("initial permissions not set correctly: got %o, want 0600", info.Mode().Perm())
		}

		// Write config (should preserve 0600)
		if err := service.writeConfig(configPath, testData); err != nil {
			t.Fatalf("writeConfig() error = %v", err)
		}

		// Verify permissions preserved
		info, err = os.Stat(configPath)
		if err != nil {
			t.Fatalf("failed to stat config after write: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("permissions not preserved: got %o, want 0600", info.Mode().Perm())
		}
	})

	t.Run("defaults to 0644 for new files", func(t *testing.T) {
		newConfigPath := filepath.Join(tmpDir, "new-config.json")

		// File does not exist yet
		if err := service.writeConfig(newConfigPath, testData); err != nil {
			t.Fatalf("writeConfig() error = %v", err)
		}

		info, err := os.Stat(newConfigPath)
		if err != nil {
			t.Fatalf("failed to stat new config: %v", err)
		}
		if info.Mode().Perm() != 0644 {
			t.Errorf("new file permissions: got %o, want 0644", info.Mode().Perm())
		}
	})
}

// Test: extractWorkflowData includes require_rejection_reason when present
func TestExtractWorkflowData_IncludesRequireRejectionReason(t *testing.T) {
	merged := map[string]interface{}{
		"status_flow": map[string]interface{}{
			"todo": []interface{}{"in_progress"},
		},
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{"color": "gray"},
		},
		"require_rejection_reason": true,
	}

	result := extractWorkflowData(merged)

	tw, ok := result["task_workflow"].(map[string]interface{})
	if !ok {
		t.Fatal("expected task_workflow to be a map")
	}

	rr, ok := tw["require_rejection_reason"]
	if !ok {
		t.Fatal("expected require_rejection_reason in task_workflow")
	}

	if rr != true {
		t.Errorf("expected require_rejection_reason to be true, got %v", rr)
	}
}

// Test: extractWorkflowData omits require_rejection_reason when absent
func TestExtractWorkflowData_OmitsRequireRejectionReasonWhenAbsent(t *testing.T) {
	merged := map[string]interface{}{
		"status_flow": map[string]interface{}{
			"todo": []interface{}{"in_progress"},
		},
		"status_metadata": map[string]interface{}{
			"todo": map[string]interface{}{"color": "gray"},
		},
	}

	result := extractWorkflowData(merged)

	tw, ok := result["task_workflow"].(map[string]interface{})
	if !ok {
		t.Fatal("expected task_workflow to be a map")
	}

	if _, ok := tw["require_rejection_reason"]; ok {
		t.Error("expected require_rejection_reason to NOT be in task_workflow when absent from merged config")
	}
}
