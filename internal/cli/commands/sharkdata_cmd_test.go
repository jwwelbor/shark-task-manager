package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ensureWorkflowConfigField
// ============================================================================

func TestEnsureWorkflowConfigField_CreatesConfigWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Empty(t, migratedFrom)

	configPath := filepath.Join(dir, ".sharkconfig.json")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
}

func TestEnsureWorkflowConfigField_AddsFieldToExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"some_other_key":"value"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Empty(t, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
	// Existing key must be preserved.
	assert.Equal(t, "value", raw["some_other_key"])
}

func TestEnsureWorkflowConfigField_MigratesExplicitJSONWorkflowConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	jsonWorkflowPath := "legacy/workflow.json"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+jsonWorkflowPath+`"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, jsonWorkflowPath, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
	assert.Equal(t, "shark-data", raw["shark_data_path"])
}

func TestEnsureWorkflowConfigField_MigratesSharkWorkflowTarget(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	jsonWorkflowPath := ".sharkworkflow.json"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+jsonWorkflowPath+`","shark_data_path":"shark-data/"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, jsonWorkflowPath, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
	assert.Equal(t, "shark-data/", raw["shark_data_path"])
}

func TestEnsureWorkflowConfigField_RespectsYAMLSharkWorkflowIndex(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	indexPath := ".sharkworkflow.yaml"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+indexPath+`","shark_data_path":"shark-data/"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, indexPath, raw["workflow_config"])
}

func TestEnsureWorkflowConfigField_RespectsCustomPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	customPath := "my-custom-workflows/"
	// Include shark_data_path so the function has nothing to add and truly
	// returns updated=false when it finds a non-legacy workflow_config.
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+customPath+`","shark_data_path":"shark-data/"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, migratedFrom)

	// Config must be untouched.
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, customPath, raw["workflow_config"])
}

func TestEnsureWorkflowConfigField_IdempotentOnAlreadyCorrectPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	// Include shark_data_path so no field is missing and the function has
	// nothing to add — both fields are already correct.
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+defaultWorkflowConfigDir+`","shark_data_path":"shark-data/"}`), 0644))

	// Both fields are present and non-legacy, so the function should leave
	// the config untouched (false, "", nil).
	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Empty(t, migratedFrom)
}

func TestWorkflowConfigDirForDataRoot_ProjectRelative(t *testing.T) {
	dir := t.TempDir()

	got := workflowConfigDirForDataRoot(dir, filepath.Join(dir, "custom-bundle"))

	assert.Equal(t, "custom-bundle/workflow/", got)
}

func TestRunSharkInstallData_JSONMigratesDeprecatedConfigWithCustomBundle(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"shark_data_path":"custom-bundle","workflow_config":"legacy/workflow.json"}`), 0644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	t.Cleanup(func() { cli.GlobalConfig.JSON = origJSON })

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkInstallData(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	assert.Equal(t, "extracted", payload["status"])
	assert.Equal(t, "legacy/workflow.json", payload["migrated_from"])
	assert.Equal(t, true, payload["config_updated"])

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, "custom-bundle/workflow/", raw["workflow_config"])
	assert.Equal(t, "custom-bundle", raw["shark_data_path"])

	_, err = os.Stat(filepath.Join(dir, "custom-bundle", "workflow", "task.yaml"))
	require.NoError(t, err)
}

func TestRunSharkInstallData_JSONMigratesDeprecatedConfigWithAbsoluteBundle(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(t.TempDir(), "shared-bundle")
	configPath := filepath.Join(dir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"shark_data_path":"`+filepath.ToSlash(bundleDir)+`","workflow_config":"legacy/workflow.json"}`), 0644))

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	t.Cleanup(func() { cli.GlobalConfig.JSON = origJSON })

	var runErr error
	out := captureStdout(t, func() {
		runErr = runSharkInstallData(&cobra.Command{}, nil)
	})
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	assert.Equal(t, "extracted", payload["status"])
	assert.Equal(t, "legacy/workflow.json", payload["migrated_from"])
	assert.Equal(t, true, payload["config_updated"])

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	expectedWorkflowConfig := filepath.ToSlash(filepath.Join(bundleDir, "workflow")) + "/"
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, expectedWorkflowConfig, raw["workflow_config"])
	assert.Equal(t, filepath.ToSlash(bundleDir), raw["shark_data_path"])

	_, err = os.Stat(filepath.Join(bundleDir, "workflow", "task.yaml"))
	require.NoError(t, err)
}

func TestPrintConfigUpdateMessage_MigrationMentionsTarget(t *testing.T) {
	out := captureStdout(t, func() {
		printConfigUpdateMessage(true, "legacy/workflow.json", "custom-bundle/workflow/")
	})

	assert.True(t, strings.Contains(out, `deprecated JSON target "legacy/workflow.json"`))
	assert.True(t, strings.Contains(out, `"custom-bundle/workflow/"`))
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(out)
}
