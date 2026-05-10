package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// YAMLWorkflowDir is the subdirectory under shark-data/ that holds per-entity
// workflow YAML files.
const YAMLWorkflowDir = "workflow"

// yamlEntityFiles maps the MultiLevelWorkflow entity slot to its YAML filename
// under shark-data/workflow/. Filenames use kebab-case to match the canonical
// Shark 2.0 layout (tech-debt.yaml, not tech_debt.yaml).
var yamlEntityFiles = []struct {
	Slot     string // matches MultiLevelWorkflow.Sources keys ("epic", "feature", etc.)
	Filename string // <slot>.yaml under shark-data/workflow/
}{
	{Slot: "epic", Filename: "epic.yaml"},
	{Slot: "feature", Filename: "feature.yaml"},
	{Slot: "task", Filename: "task.yaml"},
	{Slot: "bug", Filename: "bug.yaml"},
	{Slot: "change", Filename: "change.yaml"},
	{Slot: "tech_debt", Filename: "tech-debt.yaml"},
	{Slot: "sprint", Filename: "sprint.yaml"},
}

// LoadMultiLevelWorkflowFromYAML reads per-entity workflow YAML files from
// <dataDir>/workflow/ and stitches them into a MultiLevelWorkflow.
//
// dataDir is the Shark 2.0 data root (the parent of prompts/, skills/, agents/,
// workflow/, overrides/). Typically <project>/shark-data/.
//
// Each entity file is independently optional — missing files leave the
// corresponding slot as nil, and GetWorkflowForLevel falls back to the
// hardcoded default. Empty Sources map keys are populated for every file that
// was actually loaded so the caller can tell what was YAML-driven vs default.
//
// File precedence: <dataDir>/overrides/workflow/<file>.yaml is checked before
// <dataDir>/workflow/<file>.yaml. Override semantics are the same as the
// {{include:}} resolver: full replacement, never merge.
//
// Returns an empty MultiLevelWorkflow with nil slots if dataDir is empty or
// has no workflow/ subdirectory; in that case the caller should fall back to
// JSON loading via the existing LoadWorkflowConfig path.
func LoadMultiLevelWorkflowFromYAML(dataDir string) (*MultiLevelWorkflow, error) {
	mlw := &MultiLevelWorkflow{
		Sources: map[string]string{},
	}

	if dataDir == "" {
		return mlw, nil
	}

	workflowDir := filepath.Join(dataDir, YAMLWorkflowDir)
	if _, err := os.Stat(workflowDir); err != nil {
		// No workflow/ subdirectory — caller should fall back to JSON loading.
		if os.IsNotExist(err) {
			return mlw, nil
		}
		return nil, fmt.Errorf("failed to stat %s: %w", workflowDir, err)
	}

	for _, entry := range yamlEntityFiles {
		// Override path takes precedence.
		overridePath := filepath.Join(dataDir, "overrides", YAMLWorkflowDir, entry.Filename)
		defaultPath := filepath.Join(workflowDir, entry.Filename)

		var loadedFrom string
		var data []byte
		var err error

		if info, statErr := os.Stat(overridePath); statErr == nil && !info.IsDir() {
			data, err = os.ReadFile(overridePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read override workflow %s: %w", overridePath, err)
			}
			loadedFrom = overridePath
		} else if info, statErr := os.Stat(defaultPath); statErr == nil && !info.IsDir() {
			data, err = os.ReadFile(defaultPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read workflow %s: %w", defaultPath, err)
			}
			loadedFrom = defaultPath
		} else {
			// File absent — leave slot nil so GetWorkflowForLevel uses the default.
			continue
		}

		cfg, err := parseWorkflowYAML(data, loadedFrom)
		if err != nil {
			return nil, fmt.Errorf("failed to parse workflow YAML %s: %w", loadedFrom, err)
		}

		assignSlot(mlw, entry.Slot, cfg)
		mlw.Sources[entry.Slot] = loadedFrom
	}

	return mlw, nil
}

// parseWorkflowYAML converts YAML bytes into a *WorkflowConfig by routing
// through JSON. The schema is JSON-tagged; rather than duplicate the tags in
// YAML we let yaml.v3 decode into a generic map[string]interface{}, then
// re-encode to JSON and Unmarshal into the typed config. This keeps a single
// source of truth (the JSON tags) for the schema.
func parseWorkflowYAML(yamlData []byte, sourcePath string) (*WorkflowConfig, error) {
	var generic map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &generic); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", sourcePath, err)
	}

	// Normalize map[interface{}]interface{} -> map[string]interface{} which
	// some yaml.v3 setups produce for nested maps. yaml.v3 actually uses
	// map[string]interface{} by default, but defensive normalization makes
	// the round-trip robust.
	normalized := normalizeYAMLValue(generic)

	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML-derived map to JSON for %s: %w", sourcePath, err)
	}

	var cfg WorkflowConfig
	if err := json.Unmarshal(jsonBytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow config from %s: %w", sourcePath, err)
	}

	if cfg.Version == "" {
		cfg.Version = "1.0"
	}
	return &cfg, nil
}

// normalizeYAMLValue recursively walks a YAML-decoded value and converts
// map[interface{}]interface{} (legacy yaml.v2 shape) into
// map[string]interface{} so encoding/json can marshal it.
func normalizeYAMLValue(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = normalizeYAMLValue(val)
		}
		return out
	case map[string]interface{}:
		for k, val := range x {
			x[k] = normalizeYAMLValue(val)
		}
		return x
	case []interface{}:
		for i, val := range x {
			x[i] = normalizeYAMLValue(val)
		}
		return x
	default:
		return v
	}
}

// assignSlot writes cfg into the appropriate field of the MultiLevelWorkflow.
// Mirrors the switch in GetWorkflowForLevel.
func assignSlot(mlw *MultiLevelWorkflow, slot string, cfg *WorkflowConfig) {
	switch slot {
	case "epic":
		mlw.Epic = cfg
	case "feature":
		mlw.Feature = cfg
	case "task":
		mlw.Task = cfg
	case "sprint":
		mlw.Sprint = cfg
	case "bug":
		mlw.Bug = cfg
	case "change":
		mlw.Change = cfg
	case "tech_debt":
		mlw.TechDebt = cfg
	}
}
