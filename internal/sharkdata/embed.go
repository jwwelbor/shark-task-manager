// Package sharkdata embeds the canonical shark-data/ tree shipped with the
// shark binary and exposes lifecycle helpers (init / upgrade / validate).
//
// Layout shipped:
//
//	shark-data/
//	  prompts/   skills/   agents/   workflow/   file_templates/   overrides/
//	  README.md
//
// All operations work against a project-local <root>/shark-data/ directory.
// `shark init` copies the embedded tree there; `shark upgrade` refreshes
// everything except <root>/shark-data/overrides/; `shark validate` checks the
// tree against the engine's expectations.
package sharkdata

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// includeRegexp returns the same syntactic pattern the engine resolver uses
// to spot {{include: <path>}} / {{augment: <path>}} directives. We mirror
// the pattern locally rather than importing the templates package to avoid
// pulling in its global state during validation.
func includeRegexp() *regexp.Regexp {
	return regexp.MustCompile(`\{\{\s*(include|augment)\s*:\s*([^}\s]+(?:\s+[^}\s]+)*?)\s*\}\}`)
}

// embeddedFS holds the canonical default shark-data/ tree shipped in the
// binary. The internal/sharkdata/default_data/ directory is the source (it is
// what ships with the tool); `shark init` materializes it to <project>/shark-data/.
// Files under default_data/overrides/ are intentionally embedded too — `shark
// init` copies them so a fresh project starts with the overrides/ skeleton
// already in place. `shark upgrade` skips overrides/.
//
//go:embed all:default_data
var embeddedFS embed.FS

// embedRootDir is the prefix every entry in embeddedFS carries. We strip it
// when materializing onto disk so that <project>/shark-data/<path> mirrors
// shark-data/<path> from the embedded tree.
const embedRootDir = "default_data"

// SharkDataDirName is the directory name that init lays down at the project
// root. It's a constant (not configurable) to keep harness/skill assumptions
// stable; users override behavior via shark-data/overrides/, not by renaming.
const SharkDataDirName = "shark-data"

// ReadEmbedded reads a file from the embedded canonical shark-data/ tree using
// a relative path (e.g. "prompts/task/draft.md"). This enables hybrid
// embed/disk resolution: callers first check disk, then fall back to this
// function when the file is absent on disk.
//
// The relPath must not be absolute and must not escape the bundle root via "..".
// Returns (nil, fs.ErrNotExist) when the path does not exist in the embedded tree.
func ReadEmbedded(relPath string) ([]byte, error) {
	if filepath.IsAbs(relPath) {
		return nil, fmt.Errorf("sharkdata.ReadEmbedded: path must be relative: %q", relPath)
	}
	// Convert OS-native separators to forward slashes (embed.FS always uses /).
	fsPath := strings.ReplaceAll(relPath, string(filepath.Separator), "/")
	// Reject upward traversal.
	if cleaned := path.Clean(fsPath); cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return nil, fmt.Errorf("sharkdata.ReadEmbedded: path must not escape bundle root: %q", relPath)
	}
	data, err := embeddedFS.ReadFile(embedRootDir + "/" + fsPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// EmbeddedFS returns the raw embed.FS rooted at the default_data/ tree.
// The FS root prefix is embedRootDir ("default_data"); callers accessing
// individual files must include this prefix. Prefer ReadEmbedded for
// single-file access.
func EmbeddedFS() (fs.FS, string) {
	return embeddedFS, embedRootDir
}

// Init copies the embedded canonical shark-data/ tree to <projectRoot>/shark-data/.
//
// Behavior:
//   - Idempotent. If <projectRoot>/shark-data/ already exists, returns
//     ErrAlreadyInitialized rather than overwriting. Callers wanting a refresh
//     should use Upgrade.
//   - Creates parent directories as needed.
//   - Preserves embedded-file content byte-for-byte.
//   - Returns the destination path on success.
func Init(projectRoot string) (string, error) {
	return InitAt(filepath.Join(projectRoot, SharkDataDirName))
}

// InitAt is the resolution-aware variant of Init: it materializes the embedded
// tree at an explicit bundle root (dest) rather than assuming
// <projectRoot>/shark-data. Callers pass the root selected by shark_data_path
// so init writes to the same directory validate/workflow/prompt resolution read
// from. Init delegates here with the default <projectRoot>/shark-data.
func InitAt(dest string) (string, error) {
	if _, err := os.Stat(dest); err == nil {
		return dest, ErrAlreadyInitialized
	}

	if err := copyEmbedded(dest); err != nil {
		return "", fmt.Errorf("shark init: %w", err)
	}
	return dest, nil
}

// Upgrade refreshes <projectRoot>/shark-data/ from the embedded canonical
// tree, preserving <projectRoot>/shark-data/overrides/ untouched.
//
// Behavior:
//   - Refuses to operate when <projectRoot>/shark-data/ does not exist
//     (run Init first).
//   - For every embedded file outside overrides/, writes the embedded content
//     to disk (creating directories as needed). Removes nothing — files added
//     locally outside overrides/ are left in place but flagged in the diff
//     summary so the user can decide.
//   - Embedded files under overrides/ are NEVER applied. The overrides/
//     directory is exclusively user territory after Init.
//
// dryRun=true returns a non-nil DiffSummary describing what would change
// without writing anything.
func Upgrade(projectRoot string, dryRun bool) (*DiffSummary, error) {
	return UpgradeAt(filepath.Join(projectRoot, SharkDataDirName), dryRun)
}

// UpgradeAt is the resolution-aware variant of Upgrade: it refreshes an
// explicit bundle root (dest) rather than assuming <projectRoot>/shark-data,
// so it honors a custom shark_data_path. Upgrade delegates here with the
// default <projectRoot>/shark-data. overrides/ is still never overwritten.
func UpgradeAt(dest string, dryRun bool) (*DiffSummary, error) {
	info, err := os.Stat(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("shark upgrade: %s does not exist — run 'shark init' first", dest)
		}
		return nil, fmt.Errorf("shark upgrade: failed to stat %s: %w", dest, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("shark upgrade: %s is not a directory", dest)
	}

	summary := &DiffSummary{}
	err = walkEmbedded(func(relPath string, data []byte, isDir bool) error {
		// Skip the overrides/ subtree entirely. walkEmbedded always reports
		// POSIX (forward-slash) relative paths, so a single "overrides/" prefix
		// check is sufficient.
		if relPath == "overrides" || strings.HasPrefix(relPath, "overrides/") {
			summary.SkippedOverrides = append(summary.SkippedOverrides, relPath)
			return nil
		}

		dstPath := filepath.Join(dest, relPath)
		if isDir {
			if !dryRun {
				if err := os.MkdirAll(dstPath, 0755); err != nil {
					return fmt.Errorf("mkdir %s: %w", dstPath, err)
				}
			}
			return nil
		}

		// File: compare against existing.
		existing, readErr := os.ReadFile(dstPath)
		switch {
		case errors.Is(readErr, fs.ErrNotExist):
			summary.Added = append(summary.Added, relPath)
		case readErr != nil:
			return fmt.Errorf("read %s: %w", dstPath, readErr)
		case string(existing) == string(data):
			summary.Unchanged = append(summary.Unchanged, relPath)
		default:
			summary.Updated = append(summary.Updated, relPath)
		}

		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", filepath.Dir(dstPath), err)
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return fmt.Errorf("write %s: %w", dstPath, err)
			}
		}
		return nil
	})
	if err != nil {
		return summary, err
	}
	return summary, nil
}

// DiffSummary describes what `shark upgrade` did or would do. Files outside
// overrides/ are categorized; the overrides/ subtree is reported separately
// for visibility.
type DiffSummary struct {
	Added            []string
	Updated          []string
	Unchanged        []string
	SkippedOverrides []string
}

// ErrAlreadyInitialized signals that <projectRoot>/shark-data/ already exists
// and Init declined to overwrite it.
var ErrAlreadyInitialized = errors.New("shark-data/ already exists at project root (use 'shark upgrade' to refresh)")

// copyEmbedded materializes the embedded tree under dest. Only called from
// Init, where the caller has already verified the destination does not exist.
func copyEmbedded(dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}

	return walkEmbedded(func(relPath string, data []byte, isDir bool) error {
		if relPath == "" {
			return nil
		}
		dstPath := filepath.Join(dest, relPath)
		if isDir {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dstPath, err)
			}
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dstPath), err)
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", dstPath, err)
		}
		return nil
	})
}

// walkEmbedded invokes visit for every file and directory under embedRootDir
// in embeddedFS, passing paths RELATIVE to embedRootDir. Empty directories
// are reported (so init/upgrade preserve directory layout even when the only
// content is a .gitkeep).
func walkEmbedded(visit func(relPath string, data []byte, isDir bool) error) error {
	return fs.WalkDir(embeddedFS, embedRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(embedRootDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = ""
		}
		// Convert OS-native separator output of filepath.Rel back to
		// forward slashes for consistency in summary output.
		relPosix := filepath.ToSlash(rel)
		if d.IsDir() {
			return visit(relPosix, nil, true)
		}
		// Files: read content.
		data, err := embeddedFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		return visit(relPosix, data, false)
	})
}

// Validate runs a series of structural checks against <projectRoot>/shark-data/.
// Returns a non-nil ValidationReport describing every issue found; the
// report is empty (Issues == nil) when validation passes.
//
// Checks performed (F3/E9 scope):
//
//  1. shark-data/ exists and is a directory.
//  2. shark-data/workflow/*.yaml — every file is well-formed YAML and decodes
//     into a workflow.WorkflowConfig with the required minimum fields
//     (status_flow + special_statuses present).
//  3. shark-data/prompts/**/*.md — every {{include: <path>}} resolves
//     against the data root via the engine's IncludeResolver. Cycles and
//     depth-cap violations bubble up.
//  4. shark-data/skills/<skill>/SKILL.md — when present, frontmatter
//     parses as YAML.
//  5. No prompt or skill file references a path under shark-data/overrides/
//     that doesn't exist alongside a default — a missing default for an
//     override is suspicious (the override would never be the canonical).
//
// Future checks (out of scope here, see follow-up idea I-2026-05-10-06 —
// golden-output diff suite):
//   - render every prompt with a stub variable map and compare to a frozen
//     baseline.
//   - cross-check workflow YAML agent references against shark-data/agents/.
func Validate(projectRoot string) (*ValidationReport, error) {
	return ValidateAt(filepath.Join(projectRoot, SharkDataDirName))
}

// ValidateAt runs the same structural checks as Validate against an explicit
// content-bundle root (dataRoot), rather than assuming <projectRoot>/shark-data.
// This is the resolution-aware entry point used when shark_data_path selects a
// non-default or absolute bundle root. Validate delegates here with the default
// <projectRoot>/shark-data so existing behavior is unchanged.
func ValidateAt(dataRoot string) (*ValidationReport, error) {
	dest := dataRoot
	report := &ValidationReport{Path: dest}

	if info, err := os.Stat(dest); err != nil {
		report.AddIssue(IssueLevelError, "", fmt.Sprintf("%s does not exist or is unreadable: %v", dest, err))
		return report, nil
	} else if !info.IsDir() {
		report.AddIssue(IssueLevelError, "", fmt.Sprintf("%s is not a directory", dest))
		return report, nil
	}

	validateWorkflowYAML(filepath.Join(dest, "workflow"), report)
	// E9 AC6: verify that agent_type and instruction_template references in
	// workflow YAML resolve to real files under shark-data/agents/,
	// shark-data/prompts/, and shark-data/skills/ respectively.
	validateWorkflowAgentRefs(filepath.Join(dest, "workflow"), dest, report)
	validateLegacyPromptNames(dest, report)
	validateLegacyInstructionLiterals(dest, report)
	// Prompt include validation requires the engine's IncludeResolver; we
	// avoid a hard import cycle by re-implementing the include scan here as
	// a lightweight regex pass against the same syntax. The engine's
	// resolver is the source of truth at render time; this static pass is
	// purely a pre-flight catcher.
	validatePromptIncludes(dest, report)
	validateSkillFrontmatter(filepath.Join(dest, "skills"), report)

	return report, nil
}

// ValidationReport collects validation issues. An empty Issues slice means
// the tree passed.
type ValidationReport struct {
	Path   string
	Issues []ValidationIssue
}

// ValidationIssue is a single finding from Validate.
type ValidationIssue struct {
	Level   string // error | warning | info
	Path    string // path within shark-data/, posix slashes (may be "" for tree-wide)
	Message string
}

// IssueLevelError, IssueLevelWarning, IssueLevelInfo classify validation
// findings.
const (
	IssueLevelError   = "error"
	IssueLevelWarning = "warning"
	IssueLevelInfo    = "info"
)

// AddIssue appends a finding to the report.
func (r *ValidationReport) AddIssue(level, path, message string) {
	r.Issues = append(r.Issues, ValidationIssue{
		Level:   level,
		Path:    path,
		Message: message,
	})
}

// HasErrors reports whether the report contains any error-level issues.
func (r *ValidationReport) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Level == IssueLevelError {
			return true
		}
	}
	return false
}

// expectedWorkflowFiles is the canonical set of per-entity workflow YAML
// filenames the engine expects to find under shark-data/workflow/.  When any
// of these files is absent, shark validate reports an error: the engine will
// silently fall back to hardcoded defaults at runtime, which can change
// dispatch behavior in hard-to-diagnose ways (B023).
//
// This list mirrors the yamlEntityFiles table in
// internal/config/workflow/yaml_loader.go.  tech-debt.yaml and sprint.yaml
// are intentionally omitted: they are supplementary — most projects do not
// define them and the fallback for those entity types is less disruptive.
var expectedWorkflowFiles = []string{
	"epic.yaml",
	"feature.yaml",
	"task.yaml",
	"bug.yaml",
	"change.yaml",
}

func validateWorkflowYAML(workflowDir string, report *ValidationReport) {
	if _, err := os.Stat(workflowDir); err != nil {
		// workflow/ is optional in F3; F4 populates it.
		return
	}

	// Check that every expected per-entity workflow file is present.
	// A missing file means the engine will silently fall back to hardcoded
	// defaults at runtime — that config drift is precisely what shark validate
	// is meant to catch before the worker hits it.
	for _, filename := range expectedWorkflowFiles {
		full := filepath.Join(workflowDir, filename)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			report.AddIssue(
				IssueLevelError,
				"workflow/"+filename,
				fmt.Sprintf("missing expected workflow file %q — engine will silently fall back to built-in defaults (run 'shark upgrade' to restore)", filename),
			)
		}
	}

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		report.AddIssue(IssueLevelError, "workflow/", fmt.Sprintf("read workflow dir: %v", err))
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		full := filepath.Join(workflowDir, name)
		data, err := os.ReadFile(full)
		if err != nil {
			report.AddIssue(IssueLevelError, "workflow/"+name, fmt.Sprintf("read: %v", err))
			continue
		}
		// Decode to a generic map and check the required keys are present.
		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			report.AddIssue(IssueLevelError, "workflow/"+name, fmt.Sprintf("invalid YAML: %v", err))
			continue
		}
		// Route-based (steps:) vs legacy (status_flow:) shape. The route-based
		// schema (E35) merges status_flow + status_metadata + special_statuses
		// into a single steps: block; the loader derives the legacy maps at load
		// time, so steps-only files carry neither legacy map on disk. Branch on
		// the shape and apply the matching shape-level checks. Deep route-based
		// validation (core outcomes, alias collisions, reachability) lives in the
		// config/workflow validator; keeping this pass raw-map/shape-level avoids
		// a sharkdata -> config/workflow import edge.
		_, hasSteps := raw["steps"]
		_, hasFlow := raw["status_flow"]
		switch {
		case hasSteps:
			steps, ok := raw["steps"].(map[string]interface{})
			if !ok || len(steps) == 0 {
				report.AddIssue(IssueLevelError, "workflow/"+name, "route-based workflow has an empty or malformed 'steps' block")
			}
			if _, ok := raw["start"]; !ok {
				report.AddIssue(IssueLevelWarning, "workflow/"+name, "route-based workflow missing 'start' (engine cannot determine the entry step)")
			}
		case hasFlow:
			if _, ok := raw["special_statuses"]; !ok {
				report.AddIssue(IssueLevelWarning, "workflow/"+name, "missing 'special_statuses' (engine will use defaults)")
			}
		default:
			report.AddIssue(IssueLevelError, "workflow/"+name, "missing required key 'steps' (route-based) or 'status_flow' (legacy)")
		}
	}
}

// validateWorkflowAgentRefs walks all workflow YAML files and cross-checks
// every agent_type, instruction_template, prompt, and skills reference against
// the agents/, prompts/, and skills/ directories in the data root. It also
// rejects legacy route aliases in shipped workflow content.
func validateWorkflowAgentRefs(workflowDir, dataRoot string, report *ValidationReport) {
	if _, err := os.Stat(workflowDir); err != nil {
		return
	}

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return
	}

	agentsDir := filepath.Join(dataRoot, "agents")
	promptsDir := filepath.Join(dataRoot, "prompts")
	skillsDir := filepath.Join(dataRoot, "skills")

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		full := filepath.Join(workflowDir, name)
		data, err := os.ReadFile(full)
		if err != nil {
			continue // already reported by validateWorkflowYAML
		}
		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			continue // already reported by validateWorkflowYAML
		}
		yamlPath := "workflow/" + name
		extractWorkflowRefs(raw, yamlPath, agentsDir, promptsDir, skillsDir, report)
	}
}

// extractWorkflowRefs recursively walks a decoded YAML map and checks agent and
// prompt references against the agents/ and prompts/ directories. It resolves
// both the legacy keys ("agent_type", "instruction_template") and the
// route-based step keys ("agent", "prompt") introduced by the consolidated
// steps: schema (E35) — without the route-based keys, ref validation goes dark
// for migrated workflow files.
func extractWorkflowRefs(v interface{}, yamlPath, agentsDir, promptsDir, skillsDir string, report *ValidationReport) {
	switch node := v.(type) {
	case map[string]interface{}:
		for key, val := range node {
			switch key {
			case "agent_type", "agent":
				agentType, ok := val.(string)
				if !ok || agentType == "" {
					break
				}
				agentFile := filepath.Join(agentsDir, agentType+".md")
				if !fileExists(agentFile) {
					report.AddIssue(
						IssueLevelError,
						yamlPath,
						fmt.Sprintf("workflow references %s %q but %s does not exist", key, agentType, filepath.ToSlash(agentFile)),
					)
				}
			case "instruction_template", "prompt":
				tmpl, ok := val.(string)
				if !ok || tmpl == "" {
					break
				}
				promptFile := filepath.Join(promptsDir, filepath.FromSlash(tmpl))
				overrideFile := filepath.Join(filepath.Dir(promptsDir), "overrides", "prompts", filepath.FromSlash(tmpl))
				if !fileExists(promptFile) && !fileExists(overrideFile) {
					report.AddIssue(
						IssueLevelError,
						yamlPath,
						fmt.Sprintf("workflow references %s %q but %s does not exist", key, tmpl, filepath.ToSlash(promptFile)),
					)
				}
			case "skills":
				validateWorkflowSkillRefs(val, yamlPath, skillsDir, report)
			case "aliases":
				validateWorkflowAliases(val, yamlPath, report)
			default:
				extractWorkflowRefs(val, yamlPath, agentsDir, promptsDir, skillsDir, report)
			}
		}
	case []interface{}:
		for _, item := range node {
			extractWorkflowRefs(item, yamlPath, agentsDir, promptsDir, skillsDir, report)
		}
	}
}

func validateWorkflowSkillRefs(v interface{}, yamlPath, skillsDir string, report *ValidationReport) {
	for _, skill := range stringList(v) {
		if skill == "" {
			continue
		}
		skillFile := filepath.Join(skillsDir, filepath.FromSlash(skill), "SKILL.md")
		overrideFile := filepath.Join(filepath.Dir(skillsDir), "overrides", "skills", filepath.FromSlash(skill), "SKILL.md")
		if !fileExists(skillFile) && !fileExists(overrideFile) {
			report.AddIssue(
				IssueLevelError,
				yamlPath,
				fmt.Sprintf("workflow references skill %q but %s does not exist", skill, filepath.ToSlash(skillFile)),
			)
		}
	}
}

func validateWorkflowAliases(v interface{}, yamlPath string, report *ValidationReport) {
	for _, alias := range stringList(v) {
		if isLegacyWorkflowAlias(alias, yamlPath) {
			report.AddIssue(
				IssueLevelError,
				yamlPath,
				fmt.Sprintf("workflow alias %q uses removed legacy status language", alias),
			)
		}
	}
}

func stringList(v interface{}) []string {
	switch values := v.(type) {
	case []string:
		return values
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func isLegacyWorkflowAlias(alias, yamlPath string) bool {
	if strings.HasPrefix(alias, "ready_for_") {
		return true
	}
	if !strings.HasPrefix(alias, "in_") {
		return false
	}
	// tech-debt uses in_progress as the canonical step name; it should not be
	// an alias, but older extracted bundles may still carry it while migrating.
	return !(alias == "in_progress" && strings.HasSuffix(yamlPath, "tech-debt.yaml"))
}

func validateLegacyPromptNames(dataRoot string, report *ValidationReport) {
	promptsDir := filepath.Join(dataRoot, "prompts")
	if _, err := os.Stat(promptsDir); err != nil {
		return
	}
	_ = filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if isExtractedSidecar(path) || filepath.Ext(path) != ".md" {
			return nil
		}
		rel := relTo(dataRoot, path)
		base := filepath.Base(path)
		if strings.HasPrefix(base, "ready_for_") || isLegacyPromptInFilename(rel, base) {
			report.AddIssue(
				IssueLevelError,
				rel,
				"prompt filename uses removed legacy status language",
			)
		}
		return nil
	})
}

func isLegacyPromptInFilename(rel, base string) bool {
	if !strings.HasPrefix(base, "in_") {
		return false
	}
	return filepath.ToSlash(rel) != "prompts/tech_debt/in_progress.md"
}

func validateLegacyInstructionLiterals(dataRoot string, report *ValidationReport) {
	for _, subdir := range []string{"agents", "prompts", "skills"} {
		root := filepath.Join(dataRoot, subdir)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if isExtractedSidecar(path) {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".md" && ext != ".tmpl" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				report.AddIssue(IssueLevelError, relTo(dataRoot, path), fmt.Sprintf("read: %v", err))
				return nil
			}
			rel := relTo(dataRoot, path)
			for _, literal := range legacyInstructionLiterals(rel, string(data)) {
				report.AddIssue(
					IssueLevelError,
					rel,
					fmt.Sprintf("instruction contains removed legacy status literal %q", literal),
				)
			}
			return nil
		})
	}
}

func legacyInstructionLiterals(rel, content string) []string {
	pattern := regexp.MustCompile(`\bready_for_[A-Za-z0-9_]+\b|\bin_[A-Za-z0-9_]+\b`)
	matches := pattern.FindAllString(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if match == "in_progress" && filepath.ToSlash(rel) == "prompts/tech_debt/in_progress.md" {
			continue
		}
		if seen[match] {
			continue
		}
		seen[match] = true
		out = append(out, match)
	}
	return out
}

// isExtractedSidecar reports whether path lies inside an `_extracted/`
// directory. These directories hold F1 migration scaffolding (capture notes
// of the original skill craft) — they are kept for reference but are NOT
// canonical, shipped content. The skill-purity gate must skip them so a
// malformed scaffolding sidecar never fails validation (E32-F04 AC-10).
func isExtractedSidecar(path string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == "_extracted" {
			return true
		}
	}
	return false
}

func validatePromptIncludes(dataRoot string, report *ValidationReport) {
	promptsDir := filepath.Join(dataRoot, "prompts")
	if _, err := os.Stat(promptsDir); err != nil {
		return
	}

	includePat := includeRegexp() // shared simple regex; mirrors the engine resolver's
	_ = filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if isExtractedSidecar(path) {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".tmpl" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.AddIssue(IssueLevelError, relTo(dataRoot, path), fmt.Sprintf("read: %v", err))
			return nil
		}
		matches := includePat.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			if len(m) != 3 {
				continue
			}
			target := strings.TrimSpace(m[2])
			if target == "" {
				report.AddIssue(IssueLevelError, relTo(dataRoot, path), "empty include path")
				continue
			}
			if filepath.IsAbs(target) {
				report.AddIssue(IssueLevelError, relTo(dataRoot, path), fmt.Sprintf("absolute include path %q (must be relative to data root)", target))
				continue
			}
			if strings.Contains(target, "..") {
				report.AddIssue(IssueLevelError, relTo(dataRoot, path), fmt.Sprintf("include path %q contains '..' (rejected)", target))
				continue
			}
			osTarget := filepath.FromSlash(target)
			defaultPath := filepath.Join(dataRoot, osTarget)
			overridePath := filepath.Join(dataRoot, "overrides", osTarget)
			defaultExists := fileExists(defaultPath)
			overrideExists := fileExists(overridePath)
			if !defaultExists && !overrideExists {
				report.AddIssue(
					IssueLevelError,
					relTo(dataRoot, path),
					fmt.Sprintf("include %q resolves to neither %s nor %s", target, defaultPath, overridePath),
				)
				continue
			}
			if !defaultExists && overrideExists {
				report.AddIssue(
					IssueLevelWarning,
					relTo(dataRoot, path),
					fmt.Sprintf("include %q has only an override file (no canonical default); upgrade may not refresh as expected", target),
				)
			}
		}
		return nil
	})
}

func validateSkillFrontmatter(skillsDir string, report *ValidationReport) {
	if _, err := os.Stat(skillsDir); err != nil {
		return
	}
	_ = filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if isExtractedSidecar(path) {
			return nil
		}
		if filepath.Base(path) != "SKILL.md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.AddIssue(IssueLevelError, relTo(filepath.Dir(skillsDir), path), fmt.Sprintf("read: %v", err))
			return nil
		}
		// Trivial frontmatter check: starts with --- and contains a closing ---.
		s := string(data)
		if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
			report.AddIssue(IssueLevelInfo, relTo(filepath.Dir(skillsDir), path), "no frontmatter (skill router skills are allowed to omit it)")
			return nil
		}
		// Find the closing delimiter and try to parse the block as YAML.
		body := s[strings.Index(s, "\n")+1:]
		closeIdx := strings.Index(body, "\n---")
		if closeIdx < 0 {
			report.AddIssue(IssueLevelError, relTo(filepath.Dir(skillsDir), path), "frontmatter opened with --- but never closed")
			return nil
		}
		var raw map[string]interface{}
		if err := yaml.Unmarshal([]byte(body[:closeIdx]), &raw); err != nil {
			// Frontmatter on skill files is informational — the engine strips
			// it from .md files before rendering and doesn't depend on it being
			// machine-parseable. Author conventions (e.g., pipes in description
			// values to enumerate options) can produce YAML that strict parsers
			// reject. Surface as a warning so authors see it, but don't fail
			// validation.
			report.AddIssue(IssueLevelWarning, relTo(filepath.Dir(skillsDir), path), fmt.Sprintf("frontmatter is not strict YAML: %v (informational only; engine strips frontmatter at render time)", err))
		}
		return nil
	})
}

func relTo(root, full string) string {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return full
	}
	return filepath.ToSlash(rel)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CopyEmbeddedTreeForTest exposes the embedded tree so tests can build
// expected file lists without re-walking embeddedFS through fs.WalkDir.
func CopyEmbeddedTreeForTest() ([]string, error) {
	var paths []string
	err := walkEmbedded(func(rel string, _ []byte, isDir bool) error {
		if !isDir && rel != "" {
			paths = append(paths, rel)
		}
		return nil
	})
	return paths, err
}
