package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

func TestEnsureWorkflowConfigField_MigratesLegacyJSONPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	legacyPath := "shark-templates/.sharkworkflow-short.json"
	require.NoError(t, os.WriteFile(configPath,
		[]byte(`{"workflow_config":"`+legacyPath+`"}`), 0644))

	updated, migratedFrom, err := ensureWorkflowConfigField(dir, defaultWorkflowConfigDir)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, legacyPath, migratedFrom)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, defaultWorkflowConfigDir, raw["workflow_config"])
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

// ============================================================================
// isLegacyWorkflowConfigValue
// ============================================================================

func TestIsLegacyWorkflowConfigValue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"shark-data/workflow/", false},
		{"my-custom-workflows/", false},
		{"shark-templates/.sharkworkflow-short.json", true},
		{"path/to/something.json", true},
		{".sharkworkflow.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isLegacyWorkflowConfigValue(tt.input))
		})
	}
}
