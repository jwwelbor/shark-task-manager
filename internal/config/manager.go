package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Manager handles config file operations
type Manager struct {
	configPath    string
	config        *Config
	actionService ActionService
}

// NewManager creates a new config manager
func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
		config:     nil,
	}
}

// Load reads and parses the config file
func (m *Manager) Load() (*Config, error) {
	// Read config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, return empty config
			m.config = &Config{
				RawData: make(map[string]interface{}),
			}
			return m.config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON into raw map first to preserve all fields
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Create config
	config := &Config{
		RawData: rawData,
	}

	// Parse last_sync_time if present
	if lastSyncStr, ok := rawData["last_sync_time"].(string); ok && lastSyncStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, lastSyncStr)
		if err != nil {
			// Invalid timestamp - log error and treat as nil
			slog.Warn("Invalid last_sync_time format in config", "error", err)
			config.LastSyncTime = nil
		} else {
			config.LastSyncTime = &parsedTime
		}
	}

	// Parse other known fields
	if colorEnabled, ok := rawData["color_enabled"].(bool); ok {
		config.ColorEnabled = &colorEnabled
	}

	if jsonOutput, ok := rawData["json_output"].(bool); ok {
		config.JSONOutput = &jsonOutput
	}

	if requireRejection, ok := rawData["require_rejection_reason"].(bool); ok {
		config.RequireRejectionReason = requireRejection
	}

	if templateDir, ok := rawData["template_directory"].(string); ok && templateDir != "" {
		config.TemplateDirectory = &templateDir
	}

	if workflowConfig, ok := rawData["workflow_config"].(string); ok && workflowConfig != "" {
		config.WorkflowConfig = &workflowConfig
	}

	// Parse observability config if present
	if obsRaw, ok := rawData["observability"].(map[string]interface{}); ok {
		var obs ObservabilityConfig
		if enabled, ok := obsRaw["enabled"].(bool); ok {
			obs.Enabled = enabled
		}
		if tracingEnabled, ok := obsRaw["tracing_enabled"].(bool); ok {
			obs.TracingEnabled = tracingEnabled
		}
		if metricsEnabled, ok := obsRaw["metrics_enabled"].(bool); ok {
			obs.MetricsEnabled = metricsEnabled
		}
		if logLevel, ok := obsRaw["log_level"].(string); ok {
			obs.LogLevel = logLevel
		}
		if logFormat, ok := obsRaw["log_format"].(string); ok {
			obs.LogFormat = logFormat
		}
		if logFile, ok := obsRaw["log_file"].(string); ok {
			obs.LogFile = logFile
		}
		if exporter, ok := obsRaw["exporter"].(string); ok {
			obs.Exporter = exporter
		}
		if otlpEndpoint, ok := obsRaw["otlp_endpoint"].(string); ok {
			obs.OTLPEndpoint = otlpEndpoint
		}
		if otlpProtocol, ok := obsRaw["otlp_protocol"].(string); ok {
			obs.OTLPProtocol = otlpProtocol
		}
		if serviceName, ok := obsRaw["service_name"].(string); ok {
			obs.ServiceName = serviceName
		}
		if sampleRate, ok := obsRaw["sample_rate"].(float64); ok {
			obs.SampleRate = sampleRate
		}
		if captureTranscripts, ok := obsRaw["capture_agent_transcripts"].(bool); ok {
			obs.CaptureAgentTranscripts = captureTranscripts
		}
		if logTruncateBytes, ok := obsRaw["log_truncate_bytes"].(float64); ok {
			obs.LogTruncateBytes = int(logTruncateBytes)
		}
		config.Observability = &obs
	}

	// Parse maintainer config if present
	if maintainerRaw, ok := rawData["maintainer"].(map[string]interface{}); ok {
		mc := &MaintainerConfig{}
		if hash, ok := maintainerRaw["password_hash"].(string); ok {
			mc.PasswordHash = hash
		}
		if window, ok := maintainerRaw["cache_window_seconds"].(float64); ok {
			mc.CacheWindowSeconds = int(window)
		}
		config.Maintainer = mc
	}

	m.config = config
	return config, nil
}

// GetLastSyncTime returns the last sync timestamp or nil if not set
func (m *Manager) GetLastSyncTime() *time.Time {
	if m.config == nil {
		return nil
	}
	return m.config.LastSyncTime
}

// UpdateLastSyncTime updates the last_sync_time field in the config file
// Uses atomic write (temp file + rename) to prevent corruption
func (m *Manager) UpdateLastSyncTime(syncTime time.Time) error {
	// Load current config if not loaded
	if m.config == nil {
		_, err := m.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Get current file permissions if file exists
	var filePerms os.FileMode = 0644
	if info, err := os.Stat(m.configPath); err == nil {
		filePerms = info.Mode().Perm()
	}

	// Update the timestamp in raw data
	if m.config.RawData == nil {
		m.config.RawData = make(map[string]interface{})
	}
	m.config.RawData["last_sync_time"] = syncTime.Format(time.RFC3339)

	// Update in-memory config
	m.config.LastSyncTime = &syncTime

	// Marshal to JSON with HTML escaping disabled for readability
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(m.config.RawData); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data := buf.Bytes()

	// Write to temp file
	tmpPath := m.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, filePerms); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, m.configPath); err != nil {
		os.Remove(tmpPath) // Cleanup temp file on failure
		return fmt.Errorf("failed to rename config: %w", err)
	}

	return nil
}

// GetActionService returns the action service for workflow queries
// Creates service lazily on first call
func (m *Manager) GetActionService() (ActionService, error) {
	if m.actionService == nil {
		service, err := NewActionService(m.configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create action service: %w", err)
		}
		m.actionService = service
	}
	return m.actionService, nil
}
