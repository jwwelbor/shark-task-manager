// Package commands provides CLI command implementations.
//
// This file wires the shark-data lifecycle commands (init / upgrade / validate)
// into the cobra command tree. The actual logic lives in
// internal/sharkdata/embed.go; these handlers are thin adapters.
//
// Spec: F3/E6-E9 of E02 (Shark 2.0 — Single-Artifact Consolidation).
package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// defaultWorkflowConfigDir is the directory `shark init` writes into
// `.sharkconfig.json`'s `workflow_config` field on fresh setups. Once the
// project ships per-entity YAMLs in `shark-data/workflow/`, the runtime
// resolves them through this field (see resolveWorkflowDir in
// internal/config/aliases.go).
const defaultWorkflowConfigDir = "shark-data/workflow/"

var (
	upgradeDryRun bool
)

var sharkInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Idempotent config setup for shark (no-op if already initialized)",
	Long: `Ensure shark is ready for use in the current project.

shark init is now a lightweight, idempotent operation:
  - Writes shark_data_path and workflow_config into .sharkconfig.json if absent.
  - Does NOT extract shark-data/ to disk — the binary serves content from its
    embedded canonical tree by default (zero-config).
  - If shark-data/ already exists, prints a status message and exits cleanly.

To explicitly extract the embedded content for local customization, use:
  shark install-shark-data

Examples:
  shark init                       # Idempotent config setup
  shark init --json                # Machine-readable output`,
	Args: cobra.NoArgs,
	RunE: runSharkInit,
}

var sharkInstallDataCmd = &cobra.Command{
	Use:   "install-shark-data",
	Short: "Extract embedded shark-data/ to disk for local customization",
	Long: `Extract the embedded canonical shark-data/ tree to <project>/shark-data/.

This is an explicit opt-in for authors and customizers who want to edit
prompts, skills, workflow YAML, or agents without rebuilding the binary.
After extraction, disk files take precedence over the embedded defaults
on a per-file basis; the embed remains the backstop for any file absent
from disk.

To refresh an existing shark-data/ with the latest embedded defaults, run:
  shark upgrade

Examples:
  shark install-shark-data           # Extract at current project root
  shark install-shark-data --json    # Machine-readable output`,
	Args: cobra.NoArgs,
	RunE: runSharkInstallData,
}

var sharkUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Refresh shark-data/ from the embedded canonical defaults",
	Long: `Refresh canonical defaults under <project>/shark-data/, preserving
<project>/shark-data/overrides/ untouched.

Override semantics: any file under shark-data/overrides/ is treated as
local user content and is never modified by upgrade. Files added locally
outside overrides/ are left in place but reported in the diff summary so
the user can decide.

Examples:
  shark upgrade                   # Apply latest defaults
  shark upgrade --dry-run         # Show what would change without writing
  shark upgrade --json            # Machine-readable diff summary`,
	Args: cobra.NoArgs,
	RunE: runSharkUpgrade,
}

var sharkValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate shark-data/ structure and references",
	Long: `Run structural checks against <project>/shark-data/.

Checks performed:
  1. shark-data/ exists and is a directory.
  2. shark-data/workflow/*.yaml — well-formed YAML with required keys.
  3. shark-data/prompts/**/*.md — every {{include:}} resolves to a real
     file under the data root (or its overrides/ mirror).
  4. shark-data/skills/<skill>/SKILL.md — frontmatter, when present, parses
     as YAML.
  5. Override-only files (no canonical default) flagged as warnings.

Exit codes:
  0  → no errors
  1  → one or more error-level issues found
  2  → unexpected internal failure

Examples:
  shark validate
  shark validate --json`,
	Args: cobra.NoArgs,
	RunE: runSharkValidate,
}

func init() {
	sharkUpgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Show diff without writing files")
	cli.RootCmd.AddCommand(sharkInitCmd)
	cli.RootCmd.AddCommand(sharkInstallDataCmd)
	cli.RootCmd.AddCommand(sharkUpgradeCmd)
	cli.RootCmd.AddCommand(sharkValidateCmd)
}

func runSharkInit(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark init: failed to locate project root: %w", err)
	}

	// Ensure .sharkconfig.json's `workflow_config` and `shark_data_path`
	// fields are present. This is the lightweight idempotent operation for
	// CC-039: config setup only, no disk extraction required.
	configUpdated, migratedFrom, err := ensureWorkflowConfigField(root, defaultWorkflowConfigDir)
	if err != nil {
		// Non-fatal: config couldn't be written, but the binary still works
		// via embedded defaults. Report and continue.
		fmt.Fprintf(os.Stderr, "warning: failed to update .sharkconfig.json: %v\n", err)
	}

	// Check whether shark-data/ already exists (informational only).
	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, resolveErr := config.ResolveSharkDataRoot(root, configBytes)
	diskExists := resolveErr == nil && func() bool {
		_, statErr := os.Stat(dataRoot)
		return statErr == nil
	}()

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status":          "ready",
			"config_updated":  configUpdated,
			"migrated_from":   migratedFrom,
			"shark_data_path": dataRoot,
			"disk_exists":     diskExists,
		})
	}

	printConfigUpdateMessage(configUpdated, migratedFrom)
	if diskExists {
		fmt.Printf("shark-data/ exists at %s (disk files take precedence over embedded defaults).\n", dataRoot)
	} else {
		fmt.Println("shark is ready. Content is served from the embedded bundle.")
		fmt.Println("Run 'shark install-shark-data' to extract it for local customization.")
	}
	return nil
}

// runSharkInstallData implements `shark install-shark-data`: explicit extraction
// of the embedded canonical tree to disk for local authoring.
func runSharkInstallData(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark install-shark-data: failed to locate project root: %w", err)
	}

	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return fmt.Errorf("shark install-shark-data: %w", err)
	}

	dest, sharkdataErr := sharkdata.InitAt(dataRoot)
	alreadyInitialized := errors.Is(sharkdataErr, sharkdata.ErrAlreadyInitialized)
	if sharkdataErr != nil && !alreadyInitialized {
		return sharkdataErr
	}

	// Also ensure config fields are present.
	configUpdated, migratedFrom, err := ensureWorkflowConfigField(root, defaultWorkflowConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to update .sharkconfig.json: %v\n", err)
	}

	if alreadyInitialized {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status":         "already_exists",
				"path":           dest,
				"config_updated": configUpdated,
				"migrated_from":  migratedFrom,
			})
		} else {
			fmt.Printf("shark-data/ already exists at %s. Run 'shark upgrade' to refresh.\n", dest)
			printConfigUpdateMessage(configUpdated, migratedFrom)
		}
		// Idempotent — already extracted, report as success.
		return nil
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status":         "extracted",
			"path":           dest,
			"config_updated": configUpdated,
			"migrated_from":  migratedFrom,
		})
	}
	fmt.Printf("Extracted shark-data/ to %s\n", dest)
	fmt.Println("Disk files now take precedence over embedded defaults on a per-file basis.")
	printConfigUpdateMessage(configUpdated, migratedFrom)
	return nil
}

// printConfigUpdateMessage emits the human-readable summary of what
// `ensureWorkflowConfigField` did to `.sharkconfig.json`. Stays a separate
// helper so both the fresh-init and already-initialized code paths share
// identical wording.
func printConfigUpdateMessage(configUpdated bool, migratedFrom string) {
	if !configUpdated {
		return
	}
	if migratedFrom != "" {
		fmt.Printf("Migrated workflow_config: %q -> %q in .sharkconfig.json\n", migratedFrom, defaultWorkflowConfigDir)
		return
	}
	fmt.Printf("Set workflow_config: %q in .sharkconfig.json\n", defaultWorkflowConfigDir)
}

// ensureWorkflowConfigField writes "workflow_config": defaultPath into
// <projectRoot>/.sharkconfig.json. Behavior matrix:
//
//   - Config missing: create a minimal {"workflow_config": "..."} file.
//   - Config exists, field absent or empty: add the field, preserving
//     other top-level keys verbatim (JSON-level merge, no schema dependency).
//   - Config exists, field set to a legacy JSON file path
//     (`.sharkworkflow*.json` or any path ending in `.json`): auto-migrate
//     to the directory default. This heals projects whose `shark admin init`
//     wrote the old `shark-templates/.sharkworkflow-short.json` value.
//   - Config exists, field set to anything else (a custom path): leave
//     alone — the user picked it intentionally.
//
// Returns (updated, migratedFrom, err) where `updated` is true iff the file
// was rewritten and `migratedFrom` is the old legacy value when an
// auto-migration kicked in (empty string otherwise).
func ensureWorkflowConfigField(projectRoot, defaultPath string) (updated bool, migratedFrom string, err error) {
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

	// Route the read-modify-write through config.Manager so unknown keys are
	// preserved and the on-disk format (indent, HTML escaping, atomic rename)
	// stays consistent with other writers (UpdateLastSyncTime,
	// SetSprintCapacityDefault). Load() handles the file-missing case by
	// returning an empty RawData map, so the "create minimal config" branch
	// folds into the same flow as the "edit existing" branch.
	mgr := config.NewManager(configPath)
	cfg, loadErr := mgr.Load()
	if loadErr != nil {
		return false, "", fmt.Errorf("load %s: %w", configPath, loadErr)
	}
	raw := cfg.RawData
	if raw == nil {
		raw = map[string]interface{}{}
	}

	existing, _ := raw["workflow_config"].(string)

	// Ensure shark_data_path is present (E: content-bundle root). This is
	// independent of workflow_config — it selects the bundle holding skills/,
	// prompts/, agents/, overrides/. Only set it when absent so a user's
	// custom value (including an absolute shared-bundle path) is preserved.
	sharkDataAdded := false
	if v, ok := raw["shark_data_path"].(string); !ok || v == "" {
		raw["shark_data_path"] = config.DefaultSharkDataPath
		sharkDataAdded = true
	}

	switch {
	case existing == "":
		// Add the field.
	case isLegacyWorkflowConfigValue(existing):
		// Heal it: auto-migrate the legacy path.
		migratedFrom = existing
	default:
		// Custom workflow_config — respect it. Still persist a freshly-added
		// shark_data_path (the two fields are independent).
		if sharkDataAdded {
			if writeErr := mgr.SaveRaw(configPath, raw); writeErr != nil {
				return false, "", fmt.Errorf("write %s: %w", configPath, writeErr)
			}
			return true, "", nil
		}
		return false, "", nil
	}

	raw["workflow_config"] = defaultPath
	if writeErr := mgr.SaveRaw(configPath, raw); writeErr != nil {
		return false, "", fmt.Errorf("write %s: %w", configPath, writeErr)
	}
	return true, migratedFrom, nil
}

// isLegacyWorkflowConfigValue reports whether v looks like a workflow_config
// value left over from the Shark 1.x JSON-file model — the value `shark
// admin init` historically wrote, or any path ending in `.json`. `shark
// init` rewrites these to the canonical `shark-data/workflow/` directory.
//
// We are deliberately permissive: a custom directory the user picked won't
// match this, so it's safe to auto-migrate. Anyone with a strong custom
// preference can re-set the field after init and we'll respect it next time.
func isLegacyWorkflowConfigValue(v string) bool {
	if v == "" {
		return false
	}
	if strings.HasSuffix(v, ".json") {
		return true
	}
	// Catch the legacy template-bundled path even when users renamed the
	// suffix.
	base := filepath.Base(v)
	return strings.HasPrefix(base, ".sharkworkflow")
}

func runSharkUpgrade(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark upgrade: failed to locate project root: %w", err)
	}

	// Refresh the bundle root selected by shark_data_path (defaults to
	// <root>/shark-data) so upgrade targets the same directory the rest of the
	// CLI resolves.
	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return fmt.Errorf("shark upgrade: %w", err)
	}

	summary, err := sharkdata.UpgradeAt(dataRoot, upgradeDryRun)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"dry_run":           upgradeDryRun,
			"added":             summary.Added,
			"updated":           summary.Updated,
			"unchanged":         summary.Unchanged,
			"skipped_overrides": summary.SkippedOverrides,
		})
	}

	prefix := "Upgrade summary"
	if upgradeDryRun {
		prefix = "Upgrade dry-run summary"
	}
	fmt.Printf("%s:\n", prefix)
	fmt.Printf("  added:     %d\n", len(summary.Added))
	fmt.Printf("  updated:   %d\n", len(summary.Updated))
	fmt.Printf("  unchanged: %d\n", len(summary.Unchanged))
	fmt.Printf("  overrides skipped: %d\n", len(summary.SkippedOverrides))
	for _, p := range summary.Added {
		fmt.Printf("  + %s\n", p)
	}
	for _, p := range summary.Updated {
		fmt.Printf("  ~ %s\n", p)
	}
	if upgradeDryRun {
		fmt.Println("\n(dry run — no files written)")
	}
	return nil
}

func runSharkValidate(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark validate: failed to locate project root: %w", err)
	}

	// Validate against the resolved content-bundle root selected by
	// shark_data_path (defaults to <root>/shark-data). A relative path that
	// escapes the project root is rejected by ResolveSharkDataRoot.
	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return fmt.Errorf("shark validate: %w", err)
	}

	report, err := sharkdata.ValidateAt(dataRoot)
	if err != nil {
		return fmt.Errorf("shark validate: internal failure: %w", err)
	}

	if cli.GlobalConfig.JSON {
		issues := make([]map[string]interface{}, 0, len(report.Issues))
		for _, issue := range report.Issues {
			issues = append(issues, map[string]interface{}{
				"level":   issue.Level,
				"path":    issue.Path,
				"message": issue.Message,
			})
		}
		_ = cli.OutputJSON(map[string]interface{}{
			"path":   report.Path,
			"issues": issues,
			"ok":     !report.HasErrors(),
		})
	} else {
		if len(report.Issues) == 0 {
			fmt.Printf("shark-data/ at %s validated successfully (0 issues)\n", report.Path)
		} else {
			fmt.Printf("shark-data/ at %s — %d issue(s):\n", report.Path, len(report.Issues))
			for _, issue := range report.Issues {
				path := issue.Path
				if path == "" {
					path = "<tree>"
				}
				fmt.Printf("  [%s] %s: %s\n", issue.Level, path, issue.Message)
			}
		}
	}

	if report.HasErrors() {
		// Use the existing exit-code prefix convention so main.go translates
		// it to a non-zero shell exit. The format is "exit code N: msg".
		return fmt.Errorf("exit code 1: validation found %d error(s)", countErrors(report))
	}
	return nil
}

func countErrors(r *sharkdata.ValidationReport) int {
	n := 0
	for _, issue := range r.Issues {
		if issue.Level == sharkdata.IssueLevelError {
			n++
		}
	}
	return n
}
