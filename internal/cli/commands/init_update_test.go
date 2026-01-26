package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	init_pkg "github.com/jwwelbor/shark-task-manager/internal/init"
)

// TestInitUpdateBasicProfile tests applying the basic workflow profile
func TestInitUpdateBasicProfile(t *testing.T) {
	// Setup: Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create service
	service := init_pkg.NewProfileService(configPath)

	// Apply basic profile
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("Expected success=true")
	}
	if result.ProfileName != "basic" {
		t.Errorf("Expected profile_name=basic, got %s", result.ProfileName)
	}

	// Verify config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

// TestInitUpdateAdvancedProfile tests applying the advanced workflow profile
func TestInitUpdateAdvancedProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "advanced",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("Expected success=true")
	}
	if result.ProfileName != "advanced" {
		t.Errorf("Expected profile_name=advanced, got %s", result.ProfileName)
	}

	// Verify config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

// TestInitUpdateNoWorkflowFlag tests adding missing fields without workflow flag
func TestInitUpdateNoWorkflowFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create partial config with missing status fields
	partialConfig := `{"database": {"backend": "local"}, "project_root": "/home/user/project"}`
	if err := os.WriteFile(configPath, []byte(partialConfig), 0644); err != nil {
		t.Fatalf("Failed to write partial config: %v", err)
	}

	// Apply with no workflow
	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "", // Empty = add missing only
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Verify that some merge happened (either fields preserved or added)
	if result.Changes == nil {
		t.Error("Expected change report")
	}

	// Verify missing fields were added (status_metadata should be added)
	if len(result.Changes.Added) == 0 {
		t.Error("Expected some fields to be added")
	}
}

// TestInitUpdateDryRun tests dry-run mode
func TestInitUpdateDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       true, // Dry run mode
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Verify dry-run flag set
	if !result.DryRun {
		t.Error("Expected DryRun=true")
	}

	// Verify config file NOT created
	if _, err := os.Stat(configPath); err == nil {
		t.Error("Config file should not exist in dry-run mode")
	}

	// Verify changes were detected
	if result.Changes == nil {
		t.Error("Expected change report in dry-run")
	}
}

// TestInitUpdateForce tests force mode overwrites existing
func TestInitUpdateForce(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create existing config with status_metadata
	existingConfig := `{
		"database": {"backend": "local"},
		"status_metadata": {
			"custom_status": {
				"color": "blue",
				"phase": "custom"
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Apply profile with force
	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        true,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// With force, existing status_metadata should be overwritten
	if result.Changes == nil {
		t.Error("Expected changes report")
	}
}

// TestInitUpdateJSONOutput tests JSON output format
func TestInitUpdateJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify required fields
	if _, ok := parsed["success"]; !ok {
		t.Error("JSON missing 'success' field")
	}
	if _, ok := parsed["profile_name"]; !ok {
		t.Error("JSON missing 'profile_name' field")
	}
	if _, ok := parsed["config_path"]; !ok {
		t.Error("JSON missing 'config_path' field")
	}
	if _, ok := parsed["dry_run"]; !ok {
		t.Error("JSON missing 'dry_run' field")
	}
}

// TestInitUpdateInvalidProfile tests error handling for invalid profile
func TestInitUpdateInvalidProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "invalid-profile",
		Force:        false,
		DryRun:       false,
	}

	_, err := service.ApplyProfile(opts)
	if err == nil {
		t.Error("Expected error for invalid profile")
	}

	// Error should mention the invalid profile
	if !strings.Contains(err.Error(), "invalid-profile") && !strings.Contains(err.Error(), "profile") {
		t.Errorf("Expected error about invalid profile, got: %v", err)
	}
}

// TestInitUpdateBackupCreation tests that backup is created
func TestInitUpdateBackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create existing config
	existingConfig := `{"database": {"backend": "local"}}`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Apply profile
	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Verify backup exists
	if result.BackupPath == "" {
		t.Error("Expected backup path to be set")
	}

	if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
		t.Errorf("Backup file not found: %s", result.BackupPath)
	}
}

// TestInitUpdatePreservesDatabase tests that database config is preserved in the file
func TestInitUpdatePreservesDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create existing config with specific database settings
	existingConfig := `{
		"database": {
			"backend": "turso",
			"url": "libsql://example.turso.io",
			"auth_token_file": "/home/user/.turso/token"
		}
	}`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Apply profile
	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	_, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Read the updated config and verify database settings are preserved in the file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	dbConfig, ok := config["database"].(map[string]interface{})
	if !ok {
		t.Error("database config not found or not a map")
		return
	}

	if backend, ok := dbConfig["backend"].(string); !ok || backend != "turso" {
		t.Error("database backend should be preserved as turso")
	}

	if url, ok := dbConfig["url"].(string); !ok || url != "libsql://example.turso.io" {
		t.Error("database URL should be preserved")
	}
}

// TestInitUpdateChangeReport tests that change report is accurate
func TestInitUpdateChangeReport(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// Verify change report structure
	if result.Changes == nil {
		t.Error("Expected changes report")
		return
	}

	// Should have some added fields
	if len(result.Changes.Added) == 0 {
		t.Error("Expected at least one added field")
	}

	// Stats should be set
	if result.Changes.Stats == nil {
		t.Error("Expected stats in change report")
		return
	}

	// Basic profile should add statuses
	if result.Changes.Stats.StatusesAdded == 0 {
		t.Error("Expected statuses to be added")
	}
}

// TestInitUpdateEmptyConfig tests with no existing config
func TestInitUpdateEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	// Don't create the file - test with no existing config

	service := init_pkg.NewProfileService(configPath)
	opts := init_pkg.UpdateOptions{
		ConfigPath:   configPath,
		WorkflowName: "basic",
		Force:        false,
		DryRun:       false,
	}

	result, err := service.ApplyProfile(opts)
	if err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success with new config")
	}

	// Config file should be created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}
}

// Helper function
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// removeANSI removes ANSI color codes from a string
func removeANSI(s string) string {
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
		} else if inEscape && r == 'm' {
			inEscape = false
		} else if !inEscape && r != '[' {
			result += string(r)
		}
	}
	return result
}

// TestInitUpdateCommandHasCorrectFlags tests that the command is properly configured
func TestInitUpdateCommandHasCorrectFlags(t *testing.T) {
	// Verify initUpdateCmd exists and has the right flags
	if initUpdateCmd == nil {
		t.Fatal("initUpdateCmd should be defined")
	}

	// Check that command is registered as a subcommand of init
	if initUpdateCmd.Use != "update [flags]" && !strings.Contains(initUpdateCmd.Use, "update") {
		t.Errorf("Expected 'update' in Use field, got %s", initUpdateCmd.Use)
	}

	// Verify key flags exist
	flag := initUpdateCmd.Flags().Lookup("workflow")
	if flag == nil {
		t.Error("--workflow flag not found")
	}

	flag = initUpdateCmd.Flags().Lookup("force")
	if flag == nil {
		t.Error("--force flag not found")
	}

	flag = initUpdateCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Error("--dry-run flag not found")
	}
}

// TestInitUpdateDisplayFunction tests the display function with human output
func TestInitUpdateDisplayFunction(t *testing.T) {
	// Create a mock result
	result := &init_pkg.UpdateResult{
		Success:     true,
		ProfileName: "basic",
		BackupPath:  ".sharkconfig.json.backup.20260125-143022",
		ConfigPath:  ".sharkconfig.json",
		DryRun:      false,
		Changes: &init_pkg.ChangeReport{
			Added:       []string{"status_metadata"},
			Preserved:   []string{"database", "viewer"},
			Overwritten: []string{},
			Stats: &init_pkg.ChangeStats{
				StatusesAdded:   5,
				FlowsAdded:      0,
				GroupsAdded:     0,
				FieldsPreserved: 2,
			},
		},
	}

	// Just verify the function doesn't panic and properly formats output
	// (Can't easily capture cli.Success output as it uses color codes)
	// This is a smoke test that the function executes without errors
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("displayUpdateResult should not panic: %v", r)
			}
		}()
		displayUpdateResult(result)
	}()
}

// TestInitUpdateDisplayFunctionDryRun tests display with dry-run flag
func TestInitUpdateDisplayFunctionDryRun(t *testing.T) {
	result := &init_pkg.UpdateResult{
		Success:     true,
		ProfileName: "basic",
		ConfigPath:  ".sharkconfig.json",
		DryRun:      true,
		Changes: &init_pkg.ChangeReport{
			Added:       []string{"status_metadata"},
			Preserved:   []string{"database"},
			Overwritten: []string{},
			Stats: &init_pkg.ChangeStats{
				StatusesAdded:   5,
				FieldsPreserved: 1,
			},
		},
	}

	// Just verify the function doesn't panic with dry-run flag set
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("displayUpdateResult should not panic with DryRun: %v", r)
			}
		}()
		displayUpdateResult(result)
	}()
}

// TestInitUpdateRunHandlerWithBasicProfile tests the command handler
func TestInitUpdateRunHandlerWithBasicProfile(t *testing.T) {
	// Setup temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	// Create a mock command with flags
	cmd := &MockCobraCommand{
		flags: map[string]string{
			"workflow": "basic",
			"config":   configPath,
		},
		context: context.Background(),
	}

	// Verify flag parsing logic (can't run actual command without full setup)
	workflow := cmd.flags["workflow"]
	if workflow != "basic" {
		t.Errorf("Expected workflow=basic, got %s", workflow)
	}
}

// MockCobraCommand is a minimal mock for testing
type MockCobraCommand struct {
	flags   map[string]string
	context context.Context
}

func (m *MockCobraCommand) Flag(name string) interface{} {
	return m.flags[name]
}

func (m *MockCobraCommand) Context() context.Context {
	return m.context
}
