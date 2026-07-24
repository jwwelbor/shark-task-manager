package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	if backups, ok := rawData["backups"].(bool); ok {
		config.Backups = &backups
	}
	if backupFiles, ok := rawData["backup_files"].(float64); ok {
		config.BackupFiles = int(backupFiles)
	}

	if templateDir, ok := rawData["template_directory"].(string); ok && templateDir != "" {
		config.TemplateDirectory = &templateDir
	}

	if workflowConfig, ok := rawData["workflow_config"].(string); ok && workflowConfig != "" {
		config.WorkflowConfig = &workflowConfig
	}

	if sharkDataPath, ok := rawData["shark_data_path"].(string); ok && sharkDataPath != "" {
		config.SharkDataPath = &sharkDataPath
	}

	if claimTTLSeconds, ok := rawData["claim_ttl_seconds"].(float64); ok {
		ttl := int(claimTTLSeconds)
		config.ClaimTTLSeconds = &ttl
	}
	if maxParallelItems, ok := rawData["max_parallel_items"].(float64); ok {
		config.MaxParallelItems = int(maxParallelItems)
	}

	// Parse console_width if present (CC-036). JSON numbers decode as float64.
	// A zero or negative value means "auto-detect" (handled in GetConsoleWidth).
	if consoleWidth, ok := rawData["console_width"].(float64); ok {
		config.ConsoleWidth = int(consoleWidth)
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

	// Parse advance_guard config if present. A nil pointer means "disabled /
	// not configured" to preserve backward compatibility.
	if advanceGuardRaw, ok := rawData["advance_guard"].(map[string]interface{}); ok {
		ag := &AdvanceGuardConfig{}
		if enabled, ok := advanceGuardRaw["enabled"].(bool); ok {
			ag.Enabled = enabled
		}
		if mode, ok := advanceGuardRaw["mode"].(string); ok {
			ag.Mode = mode
		}
		if allowRepeat, ok := advanceGuardRaw["allow_repeat_with_force"].(bool); ok {
			ag.AllowRepeatWithForce = allowRepeat
		}
		config.AdvanceGuard = ag
	}

	// Parse tag_required_for list if present. This mirrors the maintainer
	// parse block above and closes the wiring gap identified in the UAT for
	// T-E28-F04-001: without this block, Manager.Load() (the production path
	// via cli.GetConfig()) would never populate Config.TagRequiredForTypes
	// even when the user had set "tag_required_for" in .sharkconfig.json,
	// silently disabling services.TagService.EnforceRequired.
	// See spec.md REQ-F-007, REQ-F-017, §2.3.
	if raw, ok := rawData["tag_required_for"].([]interface{}); ok {
		types := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok {
				types = append(types, s)
			}
		}
		config.TagRequiredForTypes = types
	}

	// Parse size_required_for list if present. Mirrors tag_required_for above.
	// Without this block Manager.Load() would never populate
	// Config.SizeRequiredForTypes even when the user had set
	// "size_required_for" in .sharkconfig.json, silently disabling
	// services.enforceSizeRequired.
	if raw, ok := rawData["size_required_for"].([]interface{}); ok {
		types := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok {
				types = append(types, s)
			}
		}
		config.SizeRequiredForTypes = types
	}

	// Parse sprint_defaults config if present (T-E19-F03-008, E19-F05, REQ-F-006,
	// REQ-F-012, REQ-F-015). A nil SprintDefaults pointer means "not configured"
	// — callers must nil-check before accessing fields (SprintDefaults.Capacity,
	// etc.). Absence of the key is not an error; new sprints simply get no
	// default capacity rows.
	if sprintDefaultsRaw, ok := rawData["sprint_defaults"].(map[string]interface{}); ok {
		sd := &SprintDefaultsConfig{}
		if carryover, ok := sprintDefaultsRaw["carryover_behavior"].(string); ok {
			sd.CarryoverBehavior = carryover
		}
		if autoCreate, ok := sprintDefaultsRaw["auto_create"].(bool); ok {
			sd.AutoCreate = autoCreate
		}
		if capacityRaw, ok := sprintDefaultsRaw["capacity"].(map[string]interface{}); ok {
			sd.Capacity = make(map[string]float64, len(capacityRaw))
			for agentType, points := range capacityRaw {
				if p, ok := points.(float64); ok {
					sd.Capacity[agentType] = p
				}
			}
		}
		config.SprintDefaults = sd
	}

	// Parse recent config if present (E07-F17).
	// A nil Recent pointer means "not configured — use built-in defaults."
	// See spec.md §5.2 and REQ-F-010, REQ-F-011.
	if recentRaw, ok := rawData["recent"].(map[string]interface{}); ok {
		rc := &RecentConfig{}
		if defaultLimit, ok := recentRaw["default_limit"].(float64); ok {
			rc.DefaultLimit = int(defaultLimit)
		}
		config.Recent = rc
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

// UpdateLastSyncTime updates the last_sync_time field in the config file.
// Routes through writeRawConfig for atomic write semantics.
func (m *Manager) UpdateLastSyncTime(syncTime time.Time) error {
	if m.config == nil {
		if _, err := m.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if m.config.RawData == nil {
		m.config.RawData = make(map[string]interface{})
	}
	m.config.RawData["last_sync_time"] = syncTime.Format(time.RFC3339)
	m.config.LastSyncTime = &syncTime

	return writeRawConfig(m.configPath, m.config.RawData)
}

// SetSprintCapacityDefault updates sprint_defaults.capacity.<agentType> in
// .sharkconfig.json. Creates the sprint_defaults section (and the capacity map
// within it) if absent. Follows the same atomic write-to-temp-then-rename pattern
// used by UpdateLastSyncTime so the config file is never left in a partial state.
//
// This method is the production entrypoint for `shark sprint capacity set --default`.
// It mutates only the config file — it does NOT write to the database. Callers that
// need to update a specific sprint's capacity row should use SprintService.SetSprintCapacity.
func (m *Manager) SetSprintCapacityDefault(agentType string, points float64) error {
	// Load current config if not yet loaded
	if m.config == nil {
		if _, err := m.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if m.config.RawData == nil {
		m.config.RawData = make(map[string]interface{})
	}

	// Navigate or create the sprint_defaults.capacity path in the raw map
	sprintDefaultsRaw, _ := m.config.RawData["sprint_defaults"].(map[string]interface{})
	if sprintDefaultsRaw == nil {
		sprintDefaultsRaw = make(map[string]interface{})
		m.config.RawData["sprint_defaults"] = sprintDefaultsRaw
	}

	capacityRaw, _ := sprintDefaultsRaw["capacity"].(map[string]interface{})
	if capacityRaw == nil {
		capacityRaw = make(map[string]interface{})
		sprintDefaultsRaw["capacity"] = capacityRaw
	}

	capacityRaw[agentType] = points

	// Mirror into in-memory SprintDefaultsConfig so subsequent reads see
	// the new value without a reload.
	if m.config.SprintDefaults == nil {
		m.config.SprintDefaults = &SprintDefaultsConfig{}
	}
	if m.config.SprintDefaults.Capacity == nil {
		m.config.SprintDefaults.Capacity = make(map[string]float64)
	}
	m.config.SprintDefaults.Capacity[agentType] = points

	if err := writeRawConfig(m.configPath, m.config.RawData); err != nil {
		return err
	}

	slog.Info("config.sprint_defaults_updated",
		"agent_type", agentType,
		"points", points,
	)
	return nil
}

// SaveRaw writes the supplied rawData map to path as indented JSON, using an
// atomic temp-file-then-rename pattern (same semantics as UpdateLastSyncTime
// and SetSprintCapacityDefault). It is the single round-trip helper for
// `.sharkconfig.json` writes so callers don't reimplement os.ReadFile →
// json.Unmarshal → json.MarshalIndent → os.WriteFile inline.
//
// Behavior:
//   - HTML escaping disabled (matches existing writers; keeps URLs readable).
//   - Two-space indentation, trailing newline (via json.Encoder.Encode).
//   - Preserves existing file permissions when path already exists; falls
//     back to 0644 for fresh files.
//   - Atomic: writes <path>.tmp, fsync-rename. Cleans up temp file on
//     rename failure.
//
// SaveRaw does NOT mutate Manager state. Pass the path explicitly so callers
// that operate on `.sharkconfig.json` outside the Load()/m.config lifecycle
// (e.g. one-shot mutations from `shark init`) can use it without first
// hydrating a Manager. For Manager-scoped writes that round-trip through
// m.config.RawData, call SaveRaw(m.configPath, m.config.RawData).
func (m *Manager) SaveRaw(path string, rawData map[string]interface{}) error {
	if rawData == nil {
		rawData = map[string]interface{}{}
	}
	return writeRawConfig(path, rawData)
}

// writeRawConfig performs the atomic JSON write used by SaveRaw,
// UpdateLastSyncTime, and SetSprintCapacityDefault.
//
// Uses os.CreateTemp so concurrent writers don't collide on a shared
// `<path>.tmp` filename, and a deferred remove cleans up the temp file
// on any failure path (including a panic mid-write).
func writeRawConfig(path string, rawData map[string]interface{}) error {
	// Preserve existing file permissions; fall back to 0644 for new files.
	var filePerms os.FileMode = 0644
	if info, err := os.Stat(path); err == nil {
		filePerms = info.Mode().Perm()
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rawData); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	// Cleanup if anything below fails or panics.
	removed := false
	defer func() {
		if !removed {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp config: %w", err)
	}
	if err := os.Chmod(tmpPath, filePerms); err != nil {
		return fmt.Errorf("failed to set temp config permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename config: %w", err)
	}
	removed = true // rename consumed the temp file
	return nil
}

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
