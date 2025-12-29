package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	// Global workflow config cache
	workflowCache     *WorkflowConfig
	workflowCacheLock sync.RWMutex
	workflowCachePath string
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
		"status_flow_version": rawConfig["status_flow_version"],
		"status_flow":         rawConfig["status_flow"],
		"status_metadata":     rawConfig["status_metadata"],
		"special_statuses":    rawConfig["special_statuses"],
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

	// Validate version is supported
	if workflow.Version != "1.0" {
		return nil, fmt.Errorf("unsupported workflow config version %s (supported: 1.0). Upgrade Shark to use this config", workflow.Version)
	}

	// Cache the parsed config
	workflowCache = &workflow
	workflowCachePath = configPath

	return &workflow, nil
}

// ClearWorkflowCache clears the in-memory workflow config cache
// Used for testing or when config file changes
func ClearWorkflowCache() {
	workflowCacheLock.Lock()
	defer workflowCacheLock.Unlock()
	workflowCache = nil
	workflowCachePath = ""
}

// GetWorkflowOrDefault loads workflow config or returns default if not configured
// This is the primary API for getting workflow config throughout Shark
func GetWorkflowOrDefault(configPath string) *WorkflowConfig {
	workflow, err := LoadWorkflowConfig(configPath)
	if err != nil {
		// Log warning and fall back to default
		fmt.Fprintf(os.Stderr, "Warning: Failed to load workflow config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default workflow. Define status_flow in %s to customize.\n", configPath)
		return DefaultWorkflow()
	}

	if workflow == nil {
		// No workflow config defined, use default
		// Don't log warning here - missing config is expected for existing projects
		return DefaultWorkflow()
	}

	return workflow
}
