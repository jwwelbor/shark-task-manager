package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkflowIndexFilename is the conventional name of the master index file that
// maps each entity type to its workflow file (E35-F04, decision D6).
const WorkflowIndexFilename = "workflow.yaml"

// workflowIndex is the parsed shape of a master index file:
//
//	entities:
//	  task:      workflow/task.yaml
//	  feature:   workflow/feature.yaml
//	  epic:      workflow/epic.yaml
//
// Entity paths resolve relative to the index file's directory (the bundle
// root); absolute paths point anywhere on the filesystem (the entire "remote"
// story — shared mount / monorepo / submodule). Local overrides/workflow/
// layer on top, matching the per-entity-directory loader.
type workflowIndex struct {
	Entities map[string]string `yaml:"entities" json:"entities"`
}

// isWorkflowIndex parses the bytes (YAML, which is a JSON superset) and reports
// whether they describe a master index — i.e. a non-empty top-level `entities:`
// map. Files that fail to parse or lack `entities:` are not indexes.
func isWorkflowIndex(data []byte) (*workflowIndex, bool) {
	var idx workflowIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, false
	}
	if len(idx.Entities) == 0 {
		return nil, false
	}
	return &idx, true
}

// LoadWorkflowIndexFile reads the file at indexPath and, if it is a master
// index, loads every referenced entity workflow rooted at the index's bundle
// directory. It returns:
//   - the assembled MultiLevelWorkflow (entity slots populated, Sources set,
//     TemplateDirectory pointed at <bundleRoot>/prompts so prompts/skills/
//     agents resolve from the same bundle),
//   - isIndex=true when the file was a master index (false means the caller
//     should fall back to directory/JSON-file handling),
//   - an error when the file is an index but a referenced workflow fails to load.
//
// indexPath must be a regular file. Relative entity paths resolve against
// filepath.Dir(indexPath); absolute entity paths are used as-is.
func LoadWorkflowIndexFile(indexPath string) (*MultiLevelWorkflow, bool, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read workflow index %s: %w", indexPath, err)
	}

	idx, ok := isWorkflowIndex(data)
	if !ok {
		return nil, false, nil
	}

	bundleRoot := filepath.Dir(indexPath)
	overridesDir := filepath.Join(bundleRoot, "overrides", "workflow")

	mlw := &MultiLevelWorkflow{Sources: map[string]string{}}

	for rawEntity, relPath := range idx.Entities {
		slot := normalizeIndexEntity(rawEntity)
		if slot == "" {
			return nil, true, fmt.Errorf("workflow index %s references unknown entity %q", indexPath, rawEntity)
		}
		entityPath := relPath
		if !filepath.IsAbs(entityPath) {
			// Reject a relative entry that escapes the bundle root via "..".
			// filepath.Clean collapses interior ".." so an escaping path
			// surfaces as a leading ".." segment (mirrors the includes guard).
			if cleaned := filepath.Clean(filepath.FromSlash(relPath)); cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
				return nil, true, fmt.Errorf("workflow index %s: entity %q path %q must not escape the bundle root", indexPath, rawEntity, relPath)
			}
			entityPath = filepath.Join(bundleRoot, relPath)
		}

		// Override file (overrides/workflow/<base>) takes precedence, mirroring
		// the per-entity directory loader's full-replacement semantics.
		loadPath := entityPath
		overridePath := filepath.Join(overridesDir, filepath.Base(entityPath))
		if info, statErr := os.Stat(overridePath); statErr == nil && !info.IsDir() {
			loadPath = overridePath
		}

		fileData, readErr := os.ReadFile(loadPath)
		if readErr != nil {
			return nil, true, fmt.Errorf("workflow index %s: failed to read %s workflow %s: %w", indexPath, slot, loadPath, readErr)
		}
		cfg, parseErr := parseWorkflowYAML(fileData, loadPath)
		if parseErr != nil {
			return nil, true, fmt.Errorf("workflow index %s: failed to parse %s workflow %s: %w", indexPath, slot, loadPath, parseErr)
		}
		assignSlot(mlw, slot, cfg)
		mlw.Sources[slot] = loadPath
	}

	// Root prompt/skill/agent resolution at the bundle. Pointing
	// TemplateDirectory at <bundleRoot>/prompts lets the orchestrator renderer
	// resolve includes/overrides from the same bundle (including absolute,
	// out-of-project bundles).
	promptsDir := filepath.Join(bundleRoot, "prompts")
	mlw.TemplateDirectory = &promptsDir

	return mlw, true, nil
}

// normalizeIndexEntity maps an index entity key to a MultiLevelWorkflow slot,
// accepting both kebab-case (tech-debt) and snake_case (tech_debt). Returns ""
// for unknown entities.
func normalizeIndexEntity(entity string) string {
	switch entity {
	case "epic", "feature", "task", "sprint", "bug", "change":
		return entity
	case "tech-debt", "tech_debt":
		return "tech_debt"
	default:
		return ""
	}
}
