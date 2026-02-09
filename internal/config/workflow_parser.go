package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	// Global workflow config cache (single-level, legacy)
	workflowCache     *WorkflowConfig
	workflowCacheLock sync.RWMutex
	workflowCachePath string

	// Multi-level workflow cache
	multiLevelCache     *MultiLevelWorkflow
	multiLevelCacheLock sync.RWMutex
	multiLevelCachePath string
)

// LoadWorkflowConfig loads workflow configuration from .sharkconfig.json
//
// Returns:
// - WorkflowConfig: parsed workflow configuration
// - error: nil if successful, error with context if parsing fails
//
// Behavior:
// - Missing config file: returns nil, nil (will use default workflow)
// - Invalid JSON: returns nil, error with line number if possible
// - Missing status_flow section: returns nil, nil (will use default workflow)
// - Valid config: returns parsed WorkflowConfig, nil
//
// Performance:
// - First call parses file and caches in memory
// - Subsequent calls return cached config (fast path)
// - Cache invalidated if config file path changes
func LoadWorkflowConfig(configPath string) (*WorkflowConfig, error) {
	// Check cache first (fast path)
	workflowCacheLock.RLock()
	if workflowCache != nil && workflowCachePath == configPath {
		defer workflowCacheLock.RUnlock()
		return workflowCache, nil
	}
	workflowCacheLock.RUnlock()

	// Slow path: load from file
	workflowCacheLock.Lock()
	defer workflowCacheLock.Unlock()

	// Double-check cache (another goroutine may have loaded it)
	if workflowCache != nil && workflowCachePath == configPath {
		return workflowCache, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist - return nil, no error
			// Caller will use default workflow
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse full config as map to extract status_flow section
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		// Provide helpful error message with line number if available
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			return nil, fmt.Errorf("invalid JSON in %s at byte offset %d: %w", configPath, syntaxErr.Offset, err)
		}
		return nil, fmt.Errorf("failed to parse JSON in %s: %w", configPath, err)
	}

	// Check if status_flow section exists
	_, hasStatusFlow := rawConfig["status_flow"]
	if !hasStatusFlow {
		// No workflow config defined - return nil, no error
		// Caller will use default workflow
		return nil, nil
	}

	// Parse workflow config from raw data
	// Re-marshal just the workflow-related fields for clean parsing
	workflowData := map[string]interface{}{
		"status_flow_version":      rawConfig["status_flow_version"],
		"status_flow":              rawConfig["status_flow"],
		"status_metadata":          rawConfig["status_metadata"],
		"special_statuses":         rawConfig["special_statuses"],
		"require_rejection_reason": rawConfig["require_rejection_reason"],
	}

	workflowJSON, err := json.Marshal(workflowData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow data: %w", err)
	}

	var workflow WorkflowConfig
	if err := json.Unmarshal(workflowJSON, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}

	// Set default version if not specified
	if workflow.Version == "" {
		workflow.Version = DefaultWorkflowVersion
	}

	// Initialize maps if nil (for safety)
	if workflow.StatusFlow == nil {
		workflow.StatusFlow = make(map[string][]string)
	}
	if workflow.StatusMetadata == nil {
		workflow.StatusMetadata = make(map[string]StatusMetadata)
	}
	if workflow.SpecialStatuses == nil {
		workflow.SpecialStatuses = make(map[string][]string)
	}

	// Validate version is supported (any 1.x version)
	if !strings.HasPrefix(workflow.Version, "1.") {
		return nil, fmt.Errorf("unsupported workflow config version %s (supported: 1.x). Upgrade Shark to use this config", workflow.Version)
	}

	// Cache the parsed config
	workflowCache = &workflow
	workflowCachePath = configPath

	return &workflow, nil
}

// ClearWorkflowCache clears all workflow caches (multi-level and legacy).
// This should be used in tests when the config file is modified.
func ClearWorkflowCache() {
	multiLevelCacheLock.Lock()
	defer multiLevelCacheLock.Unlock()
	multiLevelCache = nil
	multiLevelCachePath = ""

	workflowCacheLock.Lock()
	defer workflowCacheLock.Unlock()
	workflowCache = nil
	workflowCachePath = ""
}

// GetWorkflowOrDefault loads workflow config or returns default if not configured
// This is the primary API for getting workflow config throughout Shark
func GetWorkflowOrDefault(configPath string) *WorkflowConfig {
	// Delegate to multi-level loader for consistency
	multi := LoadMultiLevelWorkflowOrDefault(configPath)
	return multi.GetWorkflowForLevel("task")
}

// LoadMultiLevelWorkflow loads all three workflow configs (epic, feature, task)
// from .sharkconfig.json.
//
// Returns:
// - *MultiLevelWorkflow with any/all of Epic, Feature, Task set (nil means use default)
// - error if file exists but contains invalid JSON
//
// Missing config file returns (&MultiLevelWorkflow{}, nil).
// Missing sections within the file result in nil for that level (will use default).
// Empty sections (e.g., "epic_workflow": {}) are treated as unconfigured (nil).
func LoadMultiLevelWorkflow(configPath string) (*MultiLevelWorkflow, error) {
	// Check cache first (fast path)
	multiLevelCacheLock.RLock()
	if multiLevelCache != nil && multiLevelCachePath == configPath {
		defer multiLevelCacheLock.RUnlock()
		return multiLevelCache, nil
	}
	multiLevelCacheLock.RUnlock()

	// Slow path: load from file
	multiLevelCacheLock.Lock()
	defer multiLevelCacheLock.Unlock()

	// Double-check cache
	if multiLevelCache != nil && multiLevelCachePath == configPath {
		return multiLevelCache, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			result := &MultiLevelWorkflow{}
			multiLevelCache = result
			multiLevelCachePath = configPath
			return result, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse full config as raw JSON
	var rawConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			return nil, fmt.Errorf("invalid JSON in %s at byte offset %d: %w", configPath, syntaxErr.Offset, err)
		}
		return nil, fmt.Errorf("failed to parse JSON in %s: %w", configPath, err)
	}

	result := &MultiLevelWorkflow{}

	// Parse epic_workflow section
	if epicRaw, ok := rawConfig["epic_workflow"]; ok {
		epicWf, err := parseWorkflowSection(epicRaw, "epic_workflow")
		if err != nil {
			return nil, fmt.Errorf("invalid epic_workflow: %w", err)
		}
		result.Epic = epicWf
	}

	// Parse feature_workflow section
	if featureRaw, ok := rawConfig["feature_workflow"]; ok {
		featureWf, err := parseWorkflowSection(featureRaw, "feature_workflow")
		if err != nil {
			return nil, fmt.Errorf("invalid feature_workflow: %w", err)
		}
		result.Feature = featureWf
	}

	// Parse task workflow from top-level fields (backward compatible)
	taskWf, err := parseTopLevelTaskWorkflow(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid task workflow: %w", err)
	}
	result.Task = taskWf

	// Also update the legacy single-level cache for backward compatibility
	workflowCacheLock.Lock()
	if result.Task != nil {
		workflowCache = result.Task
	}
	workflowCachePath = configPath
	workflowCacheLock.Unlock()

	// Cache the multi-level result
	multiLevelCache = result
	multiLevelCachePath = configPath

	return result, nil
}

// LoadMultiLevelWorkflowOrDefault loads configs or returns defaults for missing sections.
// Never returns nil, never returns an error (falls back to defaults on any failure).
func LoadMultiLevelWorkflowOrDefault(configPath string) *MultiLevelWorkflow {
	multi, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load workflow config: %v\n", err)
		return &MultiLevelWorkflow{}
	}
	if multi == nil {
		return &MultiLevelWorkflow{}
	}
	return multi
}

// parseWorkflowSection parses a workflow config from a raw JSON section.
// Returns nil if the section is empty ({}).
func parseWorkflowSection(raw json.RawMessage, sectionName string) (*WorkflowConfig, error) {
	// Check for empty object
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "{}" || trimmed == "null" {
		return nil, nil
	}

	var wf WorkflowConfig
	if err := json.Unmarshal(raw, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", sectionName, err)
	}

	// Check if it has any meaningful content
	if wf.StatusFlow == nil || len(wf.StatusFlow) == 0 {
		return nil, nil
	}

	// Set default version
	if wf.Version == "" {
		wf.Version = DefaultWorkflowVersion
	}

	// Initialize nil maps
	if wf.StatusMetadata == nil {
		wf.StatusMetadata = make(map[string]StatusMetadata)
	}
	if wf.SpecialStatuses == nil {
		wf.SpecialStatuses = make(map[string][]string)
	}

	return &wf, nil
}

// parseTopLevelTaskWorkflow extracts the task workflow from top-level config fields.
// Returns nil if no top-level status_flow is defined.
func parseTopLevelTaskWorkflow(rawConfig map[string]json.RawMessage) (*WorkflowConfig, error) {
	// Check if status_flow section exists at the top level
	if _, ok := rawConfig["status_flow"]; !ok {
		return nil, nil
	}

	// Build a workflow-only JSON object from top-level fields
	workflowData := make(map[string]json.RawMessage)
	for _, key := range []string{"status_flow_version", "status_flow", "status_metadata", "special_statuses", "require_rejection_reason"} {
		if val, ok := rawConfig[key]; ok {
			workflowData[key] = val
		}
	}

	workflowJSON, err := json.Marshal(workflowData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task workflow data: %w", err)
	}

	var wf WorkflowConfig
	if err := json.Unmarshal(workflowJSON, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse task workflow config: %w", err)
	}

	// Set defaults
	if wf.Version == "" {
		wf.Version = DefaultWorkflowVersion
	}
	if wf.StatusFlow == nil {
		wf.StatusFlow = make(map[string][]string)
	}
	if wf.StatusMetadata == nil {
		wf.StatusMetadata = make(map[string]StatusMetadata)
	}
	if wf.SpecialStatuses == nil {
		wf.SpecialStatuses = make(map[string][]string)
	}

	// Validate version
	if !strings.HasPrefix(wf.Version, "1.") {
		return nil, fmt.Errorf("unsupported workflow config version %s (supported: 1.x)", wf.Version)
	}

	return &wf, nil
}
