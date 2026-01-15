package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
			log.Printf("Warning: Invalid last_sync_time format in config: %v", err)
			config.LastSyncTime = nil
		} else {
			config.LastSyncTime = &parsedTime
		}
	}

	// Parse other known fields
	if colorEnabled, ok := rawData["color_enabled"].(bool); ok {
		config.ColorEnabled = &colorEnabled
	}

	if defaultEpic, ok := rawData["default_epic"].(string); ok {
		config.DefaultEpic = &defaultEpic
	}

	if defaultAgent, ok := rawData["default_agent"].(string); ok {
		config.DefaultAgent = &defaultAgent
	}

	if jsonOutput, ok := rawData["json_output"].(bool); ok {
		config.JSONOutput = &jsonOutput
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
