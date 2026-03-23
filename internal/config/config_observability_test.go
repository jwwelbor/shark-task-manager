package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservabilityConfig_MissingKey_DefaultsToZeroValue(t *testing.T) {
	// Config without observability key
	configJSON := `{
		"color_enabled": true
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	// Observability pointer should be nil
	assert.Nil(t, cfg.Observability)

	// GetObservability helper should return zero value safely
	obs := cfg.GetObservability()
	assert.False(t, obs.Enabled)
	assert.False(t, obs.TracingEnabled)
	assert.False(t, obs.MetricsEnabled)
	assert.Empty(t, obs.LogLevel)
	assert.Empty(t, obs.LogFormat)
	assert.Empty(t, obs.Exporter)
	assert.Empty(t, obs.OTLPEndpoint)
	assert.Empty(t, obs.OTLPProtocol)
	assert.Empty(t, obs.ServiceName)
	assert.Equal(t, float64(0), obs.SampleRate)
}

func TestObservabilityConfig_FullConfig_ParsesCorrectly(t *testing.T) {
	configJSON := `{
		"observability": {
			"enabled": true,
			"tracing_enabled": true,
			"metrics_enabled": true,
			"log_level": "debug",
			"log_format": "json",
			"exporter": "otlp",
			"otlp_endpoint": "collector:4317",
			"otlp_protocol": "grpc",
			"service_name": "my-shark",
			"sample_rate": 0.5
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	require.NotNil(t, cfg.Observability)
	assert.True(t, cfg.Observability.Enabled)
	assert.True(t, cfg.Observability.TracingEnabled)
	assert.True(t, cfg.Observability.MetricsEnabled)
	assert.Equal(t, "debug", cfg.Observability.LogLevel)
	assert.Equal(t, "json", cfg.Observability.LogFormat)
	assert.Equal(t, "otlp", cfg.Observability.Exporter)
	assert.Equal(t, "collector:4317", cfg.Observability.OTLPEndpoint)
	assert.Equal(t, "grpc", cfg.Observability.OTLPProtocol)
	assert.Equal(t, "my-shark", cfg.Observability.ServiceName)
	assert.InDelta(t, 0.5, cfg.Observability.SampleRate, 0.001)
}

func TestObservabilityConfig_PartialConfig_OnlySetFields(t *testing.T) {
	configJSON := `{
		"observability": {
			"enabled": true,
			"log_format": "text"
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	require.NotNil(t, cfg.Observability)
	assert.True(t, cfg.Observability.Enabled)
	assert.Equal(t, "text", cfg.Observability.LogFormat)
	// Unset fields remain zero value
	assert.False(t, cfg.Observability.TracingEnabled)
	assert.False(t, cfg.Observability.MetricsEnabled)
	assert.Empty(t, cfg.Observability.LogLevel)
	assert.Empty(t, cfg.Observability.Exporter)
	assert.Empty(t, cfg.Observability.OTLPEndpoint)
}

func TestObservabilityConfig_EmptyConfig_NoError(t *testing.T) {
	configJSON := `{}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)
	assert.Nil(t, cfg.Observability)
	assert.False(t, cfg.GetObservability().Enabled)
}

func TestObservabilityConfig_NonexistentFile_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)
	assert.Nil(t, cfg.Observability)
	assert.False(t, cfg.GetObservability().Enabled)
}

func TestObservabilityConfig_ExistingFieldsPreserved(t *testing.T) {
	// Ensure adding observability doesn't break existing fields
	configJSON := `{
		"color_enabled": true,
		"require_rejection_reason": true,
		"template_directory": "custom-templates",
		"observability": {
			"enabled": true
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	mgr := NewManager(configPath)
	cfg, err := mgr.Load()
	require.NoError(t, err)

	// Existing fields preserved
	require.NotNil(t, cfg.ColorEnabled)
	assert.True(t, *cfg.ColorEnabled)
	assert.True(t, cfg.RequireRejectionReason)
	require.NotNil(t, cfg.TemplateDirectory)
	assert.Equal(t, "custom-templates", *cfg.TemplateDirectory)

	// Observability also parsed
	require.NotNil(t, cfg.Observability)
	assert.True(t, cfg.Observability.Enabled)
}
