package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/pathutil"
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

// ErrDeprecatedWorkflowConfigJSON identifies an explicit workflow_config target
// that points at a deprecated Shark 1.x JSON workflow file.
var ErrDeprecatedWorkflowConfigJSON = errors.New("deprecated workflow_config JSON file")

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

	// Check if a workflow section exists in either the legacy (status_flow) or
	// route-based (steps) shape. Absence of both means "use default workflow".
	_, hasStatusFlow := rawConfig["status_flow"]
	_, hasSteps := rawConfig["steps"]
	if !hasStatusFlow && !hasSteps {
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
		"start":                    rawConfig["start"],
		"steps":                    rawConfig["steps"],
	}

	workflowJSON, err := json.Marshal(workflowData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow data: %w", err)
	}

	var workflow WorkflowConfig
	if err := json.Unmarshal(workflowJSON, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}

	// Route-based schema (E35-F01): project consolidated steps: onto legacy maps.
	buildWorkflowMapsFromSteps(&workflow)

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
//
// This is a thin wrapper around LoadMultiLevelWorkflowFromBytes that handles
// the os.ReadFile call. Callers that have already read .sharkconfig.json
// (e.g. defaultWorkflowDataLoader) should call LoadMultiLevelWorkflowFromBytes
// directly to avoid re-reading the file (see TD-023).
func LoadMultiLevelWorkflow(configPath string) (*MultiLevelWorkflow, error) {
	// Check cache first (fast path)
	multiLevelCacheLock.RLock()
	if multiLevelCache != nil && multiLevelCachePath == configPath {
		defer multiLevelCacheLock.RUnlock()
		return multiLevelCache, nil
	}
	multiLevelCacheLock.RUnlock()

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			multiLevelCacheLock.Lock()
			defer multiLevelCacheLock.Unlock()
			// Double-check cache (another goroutine may have populated it).
			if multiLevelCache != nil && multiLevelCachePath == configPath {
				return multiLevelCache, nil
			}
			result := &MultiLevelWorkflow{}
			multiLevelCache = result
			multiLevelCachePath = configPath
			return result, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	return LoadMultiLevelWorkflowFromBytes(configPath, data)
}

// LoadMultiLevelWorkflowFromBytes is the same as LoadMultiLevelWorkflow except
// it accepts the raw .sharkconfig.json bytes that the caller has already read.
// This avoids redundant os.ReadFile calls on the hot startup path (TD-023).
//
// configPath is still required for:
//   - Resolving relative paths in workflow_config
//   - Cache keying (so subsequent LoadMultiLevelWorkflow calls hit cache)
//   - Source tracking ("source": configPath strings on the returned MultiLevelWorkflow)
//
// If data is empty, behaves the same as LoadMultiLevelWorkflow when the file
// is missing: returns an empty MultiLevelWorkflow.
func LoadMultiLevelWorkflowFromBytes(configPath string, data []byte) (*MultiLevelWorkflow, error) {
	// Check cache first (fast path) — callers may have already populated it.
	multiLevelCacheLock.RLock()
	if multiLevelCache != nil && multiLevelCachePath == configPath {
		defer multiLevelCacheLock.RUnlock()
		return multiLevelCache, nil
	}
	multiLevelCacheLock.RUnlock()

	// Slow path: parse from bytes
	multiLevelCacheLock.Lock()
	defer multiLevelCacheLock.Unlock()

	// Double-check cache
	if multiLevelCache != nil && multiLevelCachePath == configPath {
		return multiLevelCache, nil
	}

	// Empty data => no config file present. Cache the empty result.
	if len(data) == 0 {
		result := &MultiLevelWorkflow{}
		multiLevelCache = result
		multiLevelCachePath = configPath
		return result, nil
	}

	// Parse full config as raw JSON
	var rawConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			return nil, fmt.Errorf("invalid JSON in %s at byte offset %d: %w", configPath, syntaxErr.Offset, err)
		}
		return nil, fmt.Errorf("failed to parse JSON in %s: %w", configPath, err)
	}

	result := &MultiLevelWorkflow{
		Sources: make(map[string]string),
	}

	// --- Workflow file loading (E20-F04) ---
	// Determine workflow file path and attempt to load it.
	// Workflow file entities take precedence over .sharkconfig.json inline definitions.
	workflowFilePath, userAbsolute := resolveWorkflowFilePath(configPath, rawConfig)

	// Validate that relative workflow file paths do not escape the project root
	// via path traversal. User-configured absolute paths (including ~/... expansion)
	// are trusted and skip this check.
	if !userAbsolute {
		configDir := filepath.Dir(configPath)
		if err := validateWorkflowFilePath(configDir, workflowFilePath); err != nil {
			return nil, fmt.Errorf("invalid workflow_config path: %w", err)
		}
	}

	if hasExplicitDeprecatedJSONWorkflowConfig(rawConfig) {
		return nil, DeprecatedWorkflowConfigJSONError()
	}

	// E35-F04: workflow_config may point at a master index file that maps each
	// entity to its workflow file, rooted at the index's bundle directory. This
	// is detected before the directory/JSON-file handling because a YAML index
	// would fail the JSON parse in loadWorkflowFile.
	if info, statErr := os.Stat(workflowFilePath); statErr == nil && !info.IsDir() {
		idxMLW, isIndex, idxErr := LoadWorkflowIndexFile(workflowFilePath)
		if idxErr != nil {
			return nil, idxErr
		}
		if isIndex {
			applyIndexResult(result, idxMLW, workflowFilePath)
			fillDefaultSources(result)
			cacheMultiLevel(result, configPath)
			return result, nil
		}
	}

	workflowFileData, err := loadWorkflowFile(workflowFilePath)
	if err != nil {
		return nil, err
	}

	// If workflow_config points at a directory, treat it as the Shark 2.0
	// per-entity YAML layout. Load each entity workflow from its YAML file
	// and overlay overrides from <parent>/overrides/workflow/. YAML-dir
	// entries take precedence over inline definitions, matching JSON-file
	// precedence below.
	if info, statErr := os.Stat(workflowFilePath); statErr == nil && info.IsDir() {
		overridesDir := filepath.Join(filepath.Dir(workflowFilePath), "overrides", "workflow")
		// B026 regression: even when LoadMultiLevelWorkflowFromYAMLDir returns
		// a non-nil error (e.g. one sibling YAML is malformed), the returned
		// MultiLevelWorkflow may carry slots that loaded successfully. Consume
		// those partial results so a single bad file doesn't silently reset
		// every entity workflow to its hardcoded default.
		if mlw, _ := LoadMultiLevelWorkflowFromYAMLDir(workflowFilePath, overridesDir); mlw != nil {
			if mlw.Epic != nil {
				result.Epic = mlw.Epic
				result.Sources["epic"] = workflowFilePath
			}
			if mlw.Feature != nil {
				result.Feature = mlw.Feature
				result.Sources["feature"] = workflowFilePath
			}
			if mlw.Task != nil {
				result.Task = mlw.Task
				result.Sources["task"] = workflowFilePath
			}
			if mlw.Bug != nil {
				result.Bug = mlw.Bug
				result.Sources["bug"] = workflowFilePath
			}
			if mlw.Change != nil {
				result.Change = mlw.Change
				result.Sources["change"] = workflowFilePath
			}
			if mlw.TechDebt != nil {
				result.TechDebt = mlw.TechDebt
				result.Sources["tech_debt"] = workflowFilePath
			}
			if mlw.Sprint != nil {
				result.Sprint = mlw.Sprint
				result.Sources["sprint"] = workflowFilePath
			}
		}
	}

	// Map workflow keys to entity level names for source tracking
	workflowKeyToLevel := map[string]string{
		"epic_workflow":      "epic",
		"feature_workflow":   "feature",
		"task_workflow":      "task",
		"bug_workflow":       "bug",
		"change_workflow":    "change",
		"tech_debt_workflow": "tech_debt",
	}

	// Parse entity blocks from workflow file first (highest precedence)
	if workflowFileData != nil {
		entityKeys := map[string]**WorkflowConfig{
			"epic_workflow":      &result.Epic,
			"feature_workflow":   &result.Feature,
			"task_workflow":      &result.Task,
			"bug_workflow":       &result.Bug,
			"change_workflow":    &result.Change,
			"tech_debt_workflow": &result.TechDebt,
		}
		for key, field := range entityKeys {
			if raw, ok := workflowFileData[key]; ok {
				wf, parseErr := parseWorkflowSection(raw, key)
				if parseErr != nil {
					return nil, fmt.Errorf("invalid %s in %s: %w", key, workflowFilePath, parseErr)
				}
				if wf != nil {
					*field = wf
					result.Sources[workflowKeyToLevel[key]] = workflowFilePath
				}
			}
		}

		// Extract template_directory from workflow file if present
		if tdRaw, ok := workflowFileData["template_directory"]; ok {
			var td string
			if json.Unmarshal(tdRaw, &td) == nil && td != "" {
				result.TemplateDirectory = &td
			}
		}
	}

	// --- Fill remaining nil levels from .sharkconfig.json ---

	// Parse epic_workflow section (only if not already set from workflow file)
	if result.Epic == nil {
		if epicRaw, ok := rawConfig["epic_workflow"]; ok {
			epicWf, parseErr := parseWorkflowSection(epicRaw, "epic_workflow")
			if parseErr != nil {
				return nil, fmt.Errorf("invalid epic_workflow: %w", parseErr)
			}
			if epicWf != nil {
				result.Epic = epicWf
				result.Sources["epic"] = configPath
			}
		}
	}

	// Parse feature_workflow section
	if result.Feature == nil {
		if featureRaw, ok := rawConfig["feature_workflow"]; ok {
			featureWf, parseErr := parseWorkflowSection(featureRaw, "feature_workflow")
			if parseErr != nil {
				return nil, fmt.Errorf("invalid feature_workflow: %w", parseErr)
			}
			if featureWf != nil {
				result.Feature = featureWf
				result.Sources["feature"] = configPath
			}
		}
	}

	// Parse bug_workflow section
	if result.Bug == nil {
		if bugRaw, ok := rawConfig["bug_workflow"]; ok {
			bugWf, parseErr := parseWorkflowSection(bugRaw, "bug_workflow")
			if parseErr != nil {
				return nil, fmt.Errorf("invalid bug_workflow: %w", parseErr)
			}
			if bugWf != nil {
				result.Bug = bugWf
				result.Sources["bug"] = configPath
			}
		}
	}

	// Parse change_workflow section
	if result.Change == nil {
		if changeRaw, ok := rawConfig["change_workflow"]; ok {
			changeWf, parseErr := parseWorkflowSection(changeRaw, "change_workflow")
			if parseErr != nil {
				return nil, fmt.Errorf("invalid change_workflow: %w", parseErr)
			}
			if changeWf != nil {
				result.Change = changeWf
				result.Sources["change"] = configPath
			}
		}
	}

	// Parse tech_debt_workflow section
	if result.TechDebt == nil {
		if tdRaw, ok := rawConfig["tech_debt_workflow"]; ok {
			tdWf, parseErr := parseWorkflowSection(tdRaw, "tech_debt_workflow")
			if parseErr != nil {
				return nil, fmt.Errorf("invalid tech_debt_workflow: %w", parseErr)
			}
			if tdWf != nil {
				result.TechDebt = tdWf
				result.Sources["tech_debt"] = configPath
			}
		}
	}

	// Parse task_workflow block from .sharkconfig.json (consistent with other entities)
	hasInlineTaskWorkflow := false
	if result.Task == nil {
		if taskRaw, ok := rawConfig["task_workflow"]; ok {
			taskWf, parseErr := parseWorkflowSection(taskRaw, "task_workflow")
			if parseErr != nil {
				return nil, fmt.Errorf("invalid task_workflow: %w", parseErr)
			}
			if taskWf != nil {
				result.Task = taskWf
				result.Sources["task"] = configPath
				hasInlineTaskWorkflow = true
			}
		}
	}

	// Detect legacy top-level keys for deprecation warning
	_, hasLegacyStatusFlow := rawConfig["status_flow"]
	if hasLegacyStatusFlow {
		// Check if task_workflow block also exists (in workflow file or inline)
		taskAlreadySet := result.Task != nil || hasInlineTaskWorkflow
		if taskAlreadySet {
			result.HasLegacyTaskKeys = true
		}
	}

	// Fall back to legacy top-level keys for task workflow (backward compatible)
	if result.Task == nil {
		taskWf, parseErr := parseTopLevelTaskWorkflow(rawConfig)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid task workflow: %w", parseErr)
		}
		if taskWf != nil {
			result.Task = taskWf
			result.Sources["task"] = configPath + " (legacy)"
		}
	}

	// Set "default" source for entities not found in any file. Derive the list
	// from EntityTypes() so a new entity type is covered automatically (this
	// previously omitted "sprint").
	for _, level := range EntityTypes() {
		if _, ok := result.Sources[level]; !ok {
			result.Sources[level] = "default"
		}
	}

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

// applyIndexResult copies the entity slots, sources, and bundle template
// directory from a master-index load (E35-F04) into the result being built.
func applyIndexResult(result, idx *MultiLevelWorkflow, sourcePath string) {
	if idx == nil {
		return
	}
	set := func(dst **WorkflowConfig, src *WorkflowConfig, slot string) {
		if src != nil {
			*dst = src
			result.Sources[slot] = sourcePath
		}
	}
	set(&result.Epic, idx.Epic, "epic")
	set(&result.Feature, idx.Feature, "feature")
	set(&result.Task, idx.Task, "task")
	set(&result.Bug, idx.Bug, "bug")
	set(&result.Change, idx.Change, "change")
	set(&result.TechDebt, idx.TechDebt, "tech_debt")
	set(&result.Sprint, idx.Sprint, "sprint")
	if idx.TemplateDirectory != nil {
		result.TemplateDirectory = idx.TemplateDirectory
	}
	// Prefer the per-entity source paths the index recorded (more precise than
	// the index path itself) when available.
	for slot, src := range idx.Sources {
		result.Sources[slot] = src
	}
}

// fillDefaultSources marks any entity slot not loaded from a file as "default".
func fillDefaultSources(result *MultiLevelWorkflow) {
	for _, level := range []string{"epic", "feature", "task", "bug", "change", "tech_debt"} {
		if _, ok := result.Sources[level]; !ok {
			result.Sources[level] = "default"
		}
	}
}

// cacheMultiLevel updates the legacy single-level cache and the multi-level
// cache. The caller MUST already hold multiLevelCacheLock (write).
func cacheMultiLevel(result *MultiLevelWorkflow, configPath string) {
	workflowCacheLock.Lock()
	if result.Task != nil {
		workflowCache = result.Task
	}
	workflowCachePath = configPath
	workflowCacheLock.Unlock()

	multiLevelCache = result
	multiLevelCachePath = configPath
}

// LoadMultiLevelWorkflowOrDefault loads configs or returns defaults for missing sections.
// Never returns nil, never returns an error (falls back to defaults on any failure).
func LoadMultiLevelWorkflowOrDefault(configPath string) *MultiLevelWorkflow {
	multi, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		slog.Warn("Failed to load workflow config", "error", err)
		return &MultiLevelWorkflow{}
	}
	if multi == nil {
		return &MultiLevelWorkflow{}
	}
	return multi
}

// LoadMultiLevelWorkflowOrDefaultFromBytes is the bytes-accepting variant of
// LoadMultiLevelWorkflowOrDefault. Callers that have already read
// .sharkconfig.json should use this to avoid redundant os.ReadFile calls
// (see TD-023).
func LoadMultiLevelWorkflowOrDefaultFromBytes(configPath string, data []byte) *MultiLevelWorkflow {
	multi, err := LoadMultiLevelWorkflowFromBytes(configPath, data)
	if err != nil {
		slog.Warn("Failed to load workflow config", "error", err)
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

	// Route-based schema (E35-F01): project consolidated steps: onto the legacy
	// maps before the emptiness check so a steps-only block is recognized.
	buildWorkflowMapsFromSteps(&wf)

	// Check if it has any meaningful content
	if len(wf.StatusFlow) == 0 {
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
	// Check if a top-level workflow section exists in either shape.
	_, hasStatusFlow := rawConfig["status_flow"]
	_, hasSteps := rawConfig["steps"]
	if !hasStatusFlow && !hasSteps {
		return nil, nil
	}

	// Build a workflow-only JSON object from top-level fields
	workflowData := make(map[string]json.RawMessage)
	for _, key := range []string{"status_flow_version", "status_flow", "status_metadata", "special_statuses", "require_rejection_reason", "start", "steps"} {
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

	// Route-based schema (E35-F01): project consolidated steps: onto legacy maps.
	buildWorkflowMapsFromSteps(&wf)

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

// validateWorkflowFilePath checks that the resolved workflow file path does not
// escape the project root directory via path traversal (e.g. "../../../etc/passwd").
// It resolves both projectRoot and filePath to absolute paths and verifies that the
// file path is contained within the project root.
func validateWorkflowFilePath(projectRoot, filePath string) error {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve project root: %w", err)
	}

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve workflow file path: %w", err)
	}

	// Ensure the root path ends with separator for correct prefix matching
	rootPrefix := absRoot + string(filepath.Separator)
	if !strings.HasPrefix(absFile, rootPrefix) && absFile != absRoot {
		return fmt.Errorf("workflow file path %q escapes project root %q", filePath, projectRoot)
	}

	return nil
}

// resolveWorkflowFilePath determines the workflow file path from config.
// If workflow_config is set in rawConfig and is a non-empty string, it is used
// (resolved relative to the config directory if relative). Otherwise, the default
// .sharkworkflow.json in the config directory is returned.
// Paths starting with "~/" are expanded to the user's home directory.
// The second return value indicates whether the path was explicitly absolute
// (user-configured absolute path or ~/... expansion), as opposed to a relative
// path that was resolved to absolute via filepath.Join.
func resolveWorkflowFilePath(configPath string, rawConfig map[string]json.RawMessage) (string, bool) {
	configDir := filepath.Dir(configPath)

	// Check for workflow_config key in raw config
	if wcRaw, ok := rawConfig["workflow_config"]; ok {
		var wc string
		if json.Unmarshal(wcRaw, &wc) == nil && wc != "" {
			wc = expandHome(wc)
			if filepath.IsAbs(wc) {
				return wc, true
			}
			return filepath.Join(configDir, wc), false
		}
	}

	// Default: .sharkworkflow.json in the same directory as .sharkconfig.json
	return filepath.Join(configDir, ".sharkworkflow.json"), false
}

func hasExplicitDeprecatedJSONWorkflowConfig(rawConfig map[string]json.RawMessage) bool {
	wcRaw, ok := rawConfig["workflow_config"]
	if !ok {
		return false
	}
	var wc string
	if json.Unmarshal(wcRaw, &wc) != nil || strings.TrimSpace(wc) == "" {
		return false
	}
	return IsDeprecatedWorkflowConfigTarget(wc)
}

// IsDeprecatedWorkflowConfigTarget reports whether workflow_config explicitly
// points at a Shark 1.x JSON workflow target. YAML directories and YAML master
// index files are supported, even when their basename starts with
// ".sharkworkflow".
func IsDeprecatedWorkflowConfigTarget(value string) bool {
	value = expandHome(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	return strings.EqualFold(filepath.Ext(filepath.Base(value)), ".json")
}

// DeprecatedWorkflowConfigJSONError returns the migration error used when an
// explicit workflow_config target points at a deprecated Shark 1.x JSON file.
func DeprecatedWorkflowConfigJSONError() error {
	return fmt.Errorf(
		"%w: explicit JSON workflow_config overrides "+
			"Shark's embedded workflow defaults, but JSON workflow files are no longer "+
			"supported as workflow_config targets. Remove or empty workflow_config in "+
			".sharkconfig.json, and remove or rename a root .sharkworkflow.json if "+
			"present, to use embedded defaults. Or run `shark admin install-shark-data` "+
			"to extract the content bundle and set workflow_config to the installed bundle's "+
			"workflow directory.",
		ErrDeprecatedWorkflowConfigJSON,
	)
}

// expandHome expands a leading "~/" to the user's home directory. It delegates
// to pathutil.ExpandHome so "~/" handling stays identical across packages.
func expandHome(path string) string {
	return pathutil.ExpandHome(path)
}

// loadWorkflowFile reads and parses the workflow file at the given path.
// Returns (nil, nil) if the file does not exist (silent fallback).
// Returns (nil, nil) if the path resolves to a directory — this is the
// Shark 2.0 case where `workflow_config` points at a per-entity YAML folder
// rather than a single JSON file. The legacy JSON loader silently no-ops so
// the new YAML-loading path can take over without spamming warnings.
// Returns (nil, error) if the file exists but cannot be parsed.
// Returns (data, nil) if the file is successfully parsed.
// An empty file (0 bytes) is treated as {} (empty JSON object, all entities nil).
func loadWorkflowFile(path string) (map[string]json.RawMessage, error) {
	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		// Shark 2.0 layout — let the YAML loader handle it.
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Silent fallback
		}
		return nil, fmt.Errorf("failed to read workflow file %s: %w", path, err)
	}

	// Empty file treated as {}
	if len(data) == 0 {
		return make(map[string]json.RawMessage), nil
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			return nil, fmt.Errorf("invalid JSON in %s at byte offset %d: %w", path, syntaxErr.Offset, err)
		}
		return nil, fmt.Errorf("failed to parse JSON in %s: %w", path, err)
	}

	return rawData, nil
}
