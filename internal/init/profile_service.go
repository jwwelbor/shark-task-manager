package init

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// ProfileService orchestrates profile application for config updates
type ProfileService struct {
	configManager *config.Manager
	merger        *ConfigMerger
}

// NewProfileService creates a new profile service
func NewProfileService(configPath string) *ProfileService {
	return &ProfileService{
		configManager: config.NewManager(configPath),
		merger:        NewConfigMerger(),
	}
}

// resolveProfileName returns the given name, or "basic" if empty.
func (s *ProfileService) resolveProfileName(name string) string {
	if name == "" {
		return "basic"
	}
	return name
}

// ApplyProfile applies a workflow profile to existing config
// Returns UpdateResult with success status, backup path, and change details
func (s *ProfileService) ApplyProfile(opts UpdateOptions) (*UpdateResult, error) {
	// 1. Load current config
	currentConfig, err := s.configManager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// If config doesn't exist, create empty base
	if currentConfig == nil {
		currentConfig = &config.Config{
			RawData: make(map[string]interface{}),
		}
	}

	// 2. Get profile map (or use basic for missing fields)
	profileName := s.resolveProfileName(opts.WorkflowName)

	profileMap, err := GetProfileMap(profileName)
	if err != nil {
		return nil, err
	}

	// 3. Merge configs
	mergeOpts := ConfigMergeOptions{
		PreserveFields: []string{
			"database",
			"project_root",
			"viewer",
			"last_sync_time",
			"interactive_mode",
			"require_rejection_reason",
		},
		OverwriteFields: []string{
			"status_metadata",
			"status_flow",
			"special_statuses",
			"status_flow_version",
			"epic_workflow",
			"feature_workflow",
			"bug_workflow",
			"change_workflow",
		},
		Force: opts.Force,
	}

	mergedConfig, changeReport, err := s.merger.Merge(
		currentConfig.RawData,
		profileMap,
		mergeOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	// 4. Build workflow file data from the merged config
	workflowData := extractWorkflowData(mergedConfig)
	workflowFilePath := resolveWorkflowFilePath(opts.ConfigPath)

	// 5. If dry run, return preview without writing
	if opts.DryRun {
		return &UpdateResult{
			Success:          true,
			ProfileName:      profileName,
			Changes:          changeReport,
			ConfigPath:       opts.ConfigPath,
			WorkflowFilePath: workflowFilePath,
			DryRun:           true,
		}, nil
	}

	// 6. Create backup before writing
	backupPath, err := s.createConfigBackup(opts.ConfigPath)
	if err != nil {
		// Log warning but don't fail - backup is nice-to-have
		if opts.Verbose {
			slog.Warn("failed to create backup", "error", err)
		}
	}

	// 7. Write merged config (atomic)
	if err := s.writeConfig(opts.ConfigPath, mergedConfig); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// 8. Write workflow file if a profile was specified
	var workflowBackupPath string
	if opts.WorkflowName != "" && len(workflowData) > 0 {
		workflowBackupPath, err = s.createConfigBackup(workflowFilePath)
		if err != nil && opts.Verbose {
			slog.Warn("failed to create workflow file backup", "error", err)
		}

		if err := s.writeConfig(workflowFilePath, workflowData); err != nil {
			return nil, fmt.Errorf("failed to write workflow file: %w", err)
		}
	}

	return &UpdateResult{
		Success:            true,
		ProfileName:        profileName,
		BackupPath:         backupPath,
		WorkflowFilePath:   workflowFilePath,
		WorkflowBackupPath: workflowBackupPath,
		Changes:            changeReport,
		ConfigPath:         opts.ConfigPath,
		DryRun:             false,
	}, nil
}

// GetChangePreview shows what would change without applying
func (s *ProfileService) GetChangePreview(opts UpdateOptions) (*ChangeReport, error) {
	// Load current config
	currentConfig, err := s.configManager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if currentConfig == nil {
		currentConfig = &config.Config{
			RawData: make(map[string]interface{}),
		}
	}

	// Get profile map
	profileName := s.resolveProfileName(opts.WorkflowName)

	profileMap, err := GetProfileMap(profileName)
	if err != nil {
		return nil, err
	}

	// Merge configs
	mergeOpts := ConfigMergeOptions{
		PreserveFields: []string{
			"database",
			"project_root",
			"viewer",
			"last_sync_time",
			"interactive_mode",
			"require_rejection_reason",
		},
		OverwriteFields: []string{
			"status_metadata",
			"status_flow",
			"special_statuses",
			"status_flow_version",
			"epic_workflow",
			"feature_workflow",
			"bug_workflow",
			"change_workflow",
		},
		Force: opts.Force,
	}

	_, changeReport, err := s.merger.Merge(
		currentConfig.RawData,
		profileMap,
		mergeOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview: %w", err)
	}

	return changeReport, nil
}

// AddMissingFields adds missing config fields without overwriting existing
func (s *ProfileService) AddMissingFields(opts UpdateOptions) (*UpdateResult, error) {
	// Force opts.WorkflowName to empty and Force to false
	opts.WorkflowName = ""
	opts.Force = false

	// Use ApplyProfile with these settings
	// This will merge basic profile but preserve all existing fields
	return s.ApplyProfile(opts)
}

// createConfigBackup creates a timestamped backup of the config file
func (s *ProfileService) createConfigBackup(configPath string) (string, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", nil // No file to backup
	}

	// Read current config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, timestamp)

	// Write backup
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// resolveWorkflowFilePath returns the .sharkworkflow.json path next to the config file.
func resolveWorkflowFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), ".sharkworkflow.json")
}

// extractWorkflowData extracts workflow-related keys from a merged config map
// and returns them in the .sharkworkflow.json format with all five entity workflow blocks.
// Task workflow is constructed from legacy top-level keys if task_workflow block is absent.
func extractWorkflowData(merged map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Entity workflow block keys to extract directly
	entityKeys := []string{"epic_workflow", "feature_workflow", "bug_workflow", "change_workflow"}
	for _, key := range entityKeys {
		if val, ok := merged[key]; ok {
			result[key] = val
		}
	}

	// Handle task_workflow: prefer block format, fall back to constructing from legacy keys
	if tw, ok := merged["task_workflow"]; ok {
		result["task_workflow"] = tw
	} else {
		// Construct task_workflow block from legacy top-level keys
		taskWorkflow := make(map[string]interface{})
		if sf, ok := merged["status_flow"]; ok {
			taskWorkflow["status_flow"] = sf
		}
		if sm, ok := merged["status_metadata"]; ok {
			taskWorkflow["status_metadata"] = sm
		}
		if ss, ok := merged["special_statuses"]; ok {
			taskWorkflow["special_statuses"] = ss
		}
		if sv, ok := merged["status_flow_version"]; ok {
			taskWorkflow["version"] = sv
		}
		// Also check for orchestrator_actions at top level
		if oa, ok := merged["orchestrator_actions"]; ok {
			taskWorkflow["orchestrator_actions"] = oa
		}
		// Also check for require_rejection_reason at top level
		if rr, ok := merged["require_rejection_reason"]; ok {
			taskWorkflow["require_rejection_reason"] = rr
		}
		if len(taskWorkflow) > 0 {
			result["task_workflow"] = taskWorkflow
		}
	}

	return result
}

// writeConfig writes config to file atomically
func (s *ProfileService) writeConfig(configPath string, data map[string]interface{}) error {
	// Marshal to JSON with HTML escaping disabled and indentation
	// Get current file permissions if file exists, default to 0644
	filePerms := os.FileMode(0644)
	if info, err := os.Stat(configPath); err == nil {
		filePerms = info.Mode().Perm()
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write: write to temp file, then rename
	dir := filepath.Dir(configPath)
	tmpFile, err := os.CreateTemp(dir, ".sharkconfig.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Cleanup temp file on error
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	// Write to temp file
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Apply original file permissions to temp file before rename
	if err := os.Chmod(tmpPath, filePerms); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
