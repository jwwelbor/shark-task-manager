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
	Short: "Initialize shark-data/ in the current project",
	Long: `Lay down the canonical shark-data/ tree at the project root.

shark init copies the embedded default shark-data/ directory (skills,
prompts, agents, workflow YAML) into <project>/shark-data/. The operation
is idempotent: if shark-data/ already exists, the command refuses to
overwrite and tells you to run 'shark upgrade' instead.

This is the bootstrap step for any project adopting Shark 2.0.

Examples:
  shark init                       # Initialize at the current working directory
  shark init --json                # Machine-readable output`,
	Args: cobra.NoArgs,
	RunE: runSharkInit,
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
	cli.RootCmd.AddCommand(sharkUpgradeCmd)
	cli.RootCmd.AddCommand(sharkValidateCmd)
}

func runSharkInit(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark init: failed to locate project root: %w", err)
	}

	dest, sharkdataErr := sharkdata.Init(root)
	alreadyInitialized := errors.Is(sharkdataErr, sharkdata.ErrAlreadyInitialized)
	if sharkdataErr != nil && !alreadyInitialized {
		return sharkdataErr
	}

	// Ensure .sharkconfig.json's `workflow_config` points at the
	// `shark-data/workflow/` directory. We run this even when shark-data/
	// was already present so a partial setup (shark-data exists but config
	// lacks the field, or still points at the legacy JSON file) gets healed
	// by a re-run of `shark init`.
	//
	// When the existing field already points at a custom directory the user
	// configured, we leave it alone.
	configUpdated, migratedFrom, err := ensureWorkflowConfigField(root, defaultWorkflowConfigDir)
	if err != nil {
		// Non-fatal: shark-data/ is fine, we just couldn't update
		// .sharkconfig.json. Report and continue.
		fmt.Fprintf(os.Stderr, "warning: failed to update workflow_config in .sharkconfig.json: %v\n", err)
	}

	if alreadyInitialized {
		if cli.GlobalConfig.JSON {
			_ = cli.OutputJSON(map[string]interface{}{
				"status":         "already_initialized",
				"path":           dest,
				"config_updated": configUpdated,
				"migrated_from":  migratedFrom,
			})
		} else {
			fmt.Printf("shark-data/ already exists at %s. Run 'shark upgrade' to refresh.\n", dest)
			printConfigUpdateMessage(configUpdated, migratedFrom)
		}
		// Return a non-zero exit so callers (scripts, CI) can distinguish
		// fresh init from no-op. The "exit code 1:" prefix is the project
		// convention used by main.go to map error strings to shell exit codes.
		return fmt.Errorf("exit code 1: %w", sharkdata.ErrAlreadyInitialized)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status":         "initialized",
			"path":           dest,
			"config_updated": configUpdated,
			"migrated_from":  migratedFrom,
		})
	}
	fmt.Printf("Initialized shark-data/ at %s\n", dest)
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
	switch {
	case existing == "":
		// Add the field.
	case isLegacyWorkflowConfigValue(existing):
		// Heal it: auto-migrate the legacy path.
		migratedFrom = existing
	default:
		// Custom path — respect it.
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

	summary, err := sharkdata.Upgrade(root, upgradeDryRun)
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

	report, err := sharkdata.Validate(root)
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
