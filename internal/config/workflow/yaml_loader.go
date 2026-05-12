package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	if dataDir == "" {
		return &MultiLevelWorkflow{Sources: map[string]string{}}, nil
	}
	workflowDir := filepath.Join(dataDir, YAMLWorkflowDir)
	overridesDir := filepath.Join(dataDir, "overrides", YAMLWorkflowDir)
	return LoadMultiLevelWorkflowFromYAMLDir(workflowDir, overridesDir)
}

// LoadMultiLevelWorkflowFromYAMLDir is the lower-level variant used when the
// caller already knows the workflow directory directly — e.g., when the
// project's `.sharkconfig.json` `workflow_config` field points at a custom
// directory rather than the default `shark-data/workflow/`.
//
// workflowDir is the directory containing the per-entity YAML files
// (<entity>.yaml). overridesDir is the directory containing per-entity
// overrides (typically `<dataDir>/overrides/workflow/`); pass "" to skip
// override resolution.
//
// Like LoadMultiLevelWorkflowFromYAML, missing files leave the slot nil and
// missing overrides are silently skipped. A missing workflowDir returns an
// empty MultiLevelWorkflow with nil slots (caller falls back to defaults).
func LoadMultiLevelWorkflowFromYAMLDir(workflowDir, overridesDir string) (*MultiLevelWorkflow, error) {
	mlw := &MultiLevelWorkflow{
		Sources: map[string]string{},
	}

	if workflowDir == "" {
		return mlw, nil
	}

	if _, err := os.Stat(workflowDir); err != nil {
		// No workflow directory — caller should fall back to defaults.
		if os.IsNotExist(err) {
			return mlw, nil
		}
		return nil, fmt.Errorf("failed to stat %s: %w", workflowDir, err)
	}

	// Per-file errors are accumulated and surfaced once at the end so a single
	// bad YAML (e.g. malformed change.yaml) does not silently discard the
	// other slots that loaded successfully. Regression: B026 — bug.yaml's
	// "draft" status was lost when a sibling YAML had a parse error, causing
	// `shark status advance` to fall back to DefaultBugWorkflow and reject
	// valid transitions.
	var loadErrs []error

	for _, entry := range yamlEntityFiles {
		// Override path takes precedence.
		var overridePath string
		if overridesDir != "" {
			overridePath = filepath.Join(overridesDir, entry.Filename)
		}
		defaultPath := filepath.Join(workflowDir, entry.Filename)

		var loadedFrom string
		var data []byte
		var err error

		if overridePath != "" {
			if info, statErr := os.Stat(overridePath); statErr == nil && !info.IsDir() {
				data, err = os.ReadFile(overridePath)
				if err != nil {
					loadErrs = append(loadErrs, fmt.Errorf("failed to read override workflow %s: %w", overridePath, err))
					continue
				}
				loadedFrom = overridePath
			}
		}
		if loadedFrom == "" {
			if info, statErr := os.Stat(defaultPath); statErr == nil && !info.IsDir() {
				data, err = os.ReadFile(defaultPath)
				if err != nil {
					loadErrs = append(loadErrs, fmt.Errorf("failed to read workflow %s: %w", defaultPath, err))
					continue
				}
				loadedFrom = defaultPath
			}
		}
		if loadedFrom == "" {
			// File absent — leave slot nil so GetWorkflowForLevel uses the
			// embedded default. Emit a verbose-only trace log (slog.Debug,
			// suppressed at the default Info level) so operators running with
			// `log_level=debug` can diagnose divergence between expected and
			// actual dispatch behavior when validate is bypassed or a YAML is
			// moved after validation. See TD-024.
			slog.Debug(
				"workflow yaml not found; using built-in default",
				"entity_type", entry.Slot,
				"path", defaultPath,
				"override_path", overridePath,
			)
			continue
		}

		cfg, err := parseWorkflowYAML(data, loadedFrom)
		if err != nil {
			loadErrs = append(loadErrs, fmt.Errorf("failed to parse workflow YAML %s: %w", loadedFrom, err))
			continue
		}

		assignSlot(mlw, entry.Slot, cfg)
		mlw.Sources[entry.Slot] = loadedFrom
	}

	if len(loadErrs) > 0 {
		return mlw, errors.Join(loadErrs...)
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
