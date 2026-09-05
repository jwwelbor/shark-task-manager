// Package commands provides CLI command implementations.
//
// This file wires the shark-data lifecycle commands (install-shark-data /
// upgrade / validate-data) into the cobra command tree under `shark admin`.
// The actual logic lives in internal/sharkdata/embed.go; these handlers are
// thin adapters.
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
	workflowconfig "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// defaultWorkflowConfigDir is the default directory
// `shark admin install-shark-data` writes into `.sharkconfig.json`'s
// `workflow_config` field when shark_data_path uses the default bundle root.
// The runtime resolves per-entity YAMLs through this field (see
// resolveWorkflowDir in internal/config/aliases.go).
const defaultWorkflowConfigDir = "shark-data/workflow/"

var (
	upgradeDryRun bool
)

var sharkInstallDataCmd = &cobra.Command{
	Use:   "install-shark-data",
	Short: "Extract the embedded content bundle to disk for local customization",
	Long: `Extract the embedded canonical content bundle to the configured
shark_data_path, defaulting to <project>/shark-data/.

This is an explicit opt-in for authors and customizers who want to edit
prompts, skills, workflow YAML, or agents without rebuilding the binary.
After extraction, disk files take precedence over the embedded defaults
on a per-file basis; the embed remains the backstop for any file absent
from disk.

Also writes shark_data_path and workflow_config into .sharkconfig.json when
those fields are absent or empty, and migrates deprecated JSON workflow_config
targets to the installed bundle's workflow directory.

To refresh an existing shark-data/ with the latest embedded defaults, run:
  shark admin upgrade

Examples:
  shark admin install-shark-data           # Extract at current project root
  shark admin install-shark-data --json    # Machine-readable output`,
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

Also reports override drift counts (current / upstream_changed /
identical_redundant / orphaned / baseline_unknown); run
'shark admin overrides status' for per-file detail.

Examples:
  shark admin upgrade                   # Apply latest defaults
  shark admin upgrade --dry-run         # Show what would change without writing
  shark admin upgrade --json            # Machine-readable diff summary`,
	Args: cobra.NoArgs,
	RunE: runSharkUpgrade,
}

var sharkValidateDataCmd = &cobra.Command{
	Use:   "validate-data",
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
  shark admin validate-data
  shark admin validate-data --json`,
	Args: cobra.NoArgs,
	RunE: runSharkValidate,
}

func init() {
	sharkUpgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Show diff without writing files")
	adminCmd.AddCommand(sharkInstallDataCmd)
	adminCmd.AddCommand(sharkUpgradeCmd)
	adminCmd.AddCommand(sharkValidateDataCmd)
}

// runSharkInstallData implements `shark admin install-shark-data`: explicit
// extraction of the embedded canonical tree to disk for local authoring.
func runSharkInstallData(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark admin install-shark-data: failed to locate project root: %w", err)
	}

	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return fmt.Errorf("shark admin install-shark-data: %w", err)
	}

	dest, sharkdataErr := sharkdata.InitAt(dataRoot)
	alreadyInitialized := errors.Is(sharkdataErr, sharkdata.ErrAlreadyInitialized)
	if sharkdataErr != nil && !alreadyInitialized {
		return sharkdataErr
	}

	// Also ensure config fields are present. Supported explicit
	// workflow_config values are preserved; deprecated JSON workflow pointers
	// are migrated to the installed YAML workflow directory.
	workflowConfigDir := workflowConfigDirForDataRoot(root, dataRoot)
	configUpdated, migratedFrom, err := ensureWorkflowConfigField(root, workflowConfigDir)
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
			fmt.Printf("Content bundle already exists at %s. Run 'shark admin upgrade' to refresh.\n", dest)
			printConfigUpdateMessage(configUpdated, migratedFrom, workflowConfigDir)
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
	fmt.Printf("Extracted content bundle to %s\n", dest)
	fmt.Println("Disk files now take precedence over embedded defaults on a per-file basis.")
	printConfigUpdateMessage(configUpdated, migratedFrom, workflowConfigDir)
	return nil
}

// printConfigUpdateMessage emits the human-readable summary of what
// `ensureWorkflowConfigField` did to `.sharkconfig.json`.
func printConfigUpdateMessage(configUpdated bool, migratedFrom, workflowConfigDir string) {
	if !configUpdated {
		return
	}
	if migratedFrom != "" {
		fmt.Printf("Migrated workflow_config from deprecated JSON target %q to %q in .sharkconfig.json\n",
			migratedFrom, workflowConfigDir)
		return
	}
	fmt.Printf("Set workflow_config: %q in .sharkconfig.json\n", workflowConfigDir)
}

// ensureWorkflowConfigField writes "workflow_config": defaultPath into
// <projectRoot>/.sharkconfig.json. Behavior matrix:
//
//   - Config missing: create a minimal {"workflow_config": "..."} file.
//   - Config exists, field absent or empty: add the field.
//   - Config exists, field set to a deprecated JSON workflow target: replace
//     it with defaultPath.
//   - Config exists, field set to any supported target: leave it alone. Still
//     persists a freshly-added shark_data_path if needed.
//
// Returns (updated, migratedFrom, err).
func ensureWorkflowConfigField(projectRoot, defaultPath string) (updated bool, migratedFrom string, err error) {
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

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
	existing = strings.TrimSpace(existing)

	// Ensure shark_data_path is present.
	sharkDataAdded := false
	if v, ok := raw["shark_data_path"].(string); !ok || v == "" {
		raw["shark_data_path"] = config.DefaultSharkDataPath
		sharkDataAdded = true
	}

	switch {
	case existing == "":
		// Add the field.
	case workflowconfig.IsDeprecatedWorkflowConfigTarget(existing):
		migratedFrom = existing
	default:
		// Explicit workflow_config — respect it. Still persist a freshly-added
		// shark_data_path if needed.
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

func workflowConfigDirForDataRoot(projectRoot, dataRoot string) string {
	workflowDir := filepath.Join(dataRoot, workflowconfig.YAMLWorkflowDir)
	if rel, err := filepath.Rel(projectRoot, workflowDir); err == nil && rel != "." && !pathEscapesRoot(rel) {
		return filepath.ToSlash(rel) + "/"
	}
	return filepath.ToSlash(filepath.Clean(workflowDir)) + "/"
}

func pathEscapesRoot(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func runSharkUpgrade(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark admin upgrade: failed to locate project root: %w", err)
	}

	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return fmt.Errorf("shark admin upgrade: %w", err)
	}

	summary, err := sharkdata.UpgradeAt(dataRoot, upgradeDryRun)
	if err != nil {
		return err
	}

	// OverrideStatusAt is read-only (REQ-F-004), so it's safe to call
	// unconditionally for both a real run and --dry-run — no branching
	// needed (REQ-F-003). On a real (non-dry-run) run, UpgradeAt above has
	// already written to disk, so a failure here must never turn an
	// otherwise-successful upgrade into a hard error or drop the four
	// pre-existing summary keys/lines — this call is purely additive. Fall
	// back to an all-zero overrides summary and warn on stderr instead.
	overridesReport, overridesErr := sharkdata.OverrideStatusAt(dataRoot)
	if overridesErr != nil {
		fmt.Fprintf(os.Stderr, "warning: shark admin upgrade: failed to compute overrides status: %v\n", overridesErr)
		overridesReport = &sharkdata.OverrideStatusReport{Summary: zeroOverridesSummary()}
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"dry_run":           upgradeDryRun,
			"added":             summary.Added,
			"updated":           summary.Updated,
			"unchanged":         summary.Unchanged,
			"skipped_overrides": summary.SkippedOverrides,
			"overrides":         overridesReport.Summary,
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
	fmt.Printf("  overrides: %s (run 'shark admin overrides status' for detail)\n",
		formatOverridesSummaryCounts(overridesReport.Summary))
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

// formatOverridesSummaryCounts renders the overrides drift summary as a
// compact "classification=count" list in stable classification order, for
// the human-readable upgrade output line.
func formatOverridesSummaryCounts(summary map[string]int) string {
	parts := make([]string, 0, len(overridesClassificationOrder))
	for _, c := range overridesClassificationOrder {
		parts = append(parts, fmt.Sprintf("%s=%d", c, summary[c]))
	}
	return strings.Join(parts, " ")
}

// overridesClassificationOrder is the stable display order for the five
// override drift classifications, shared by formatOverridesSummaryCounts and
// zeroOverridesSummary.
var overridesClassificationOrder = []string{
	sharkdata.ClassificationCurrent,
	sharkdata.ClassificationUpstreamChanged,
	sharkdata.ClassificationIdenticalRedundant,
	sharkdata.ClassificationOrphaned,
	sharkdata.ClassificationBaselineUnknown,
}

// zeroOverridesSummary returns an all-zero, five-key overrides summary, used
// as the fallback when OverrideStatusAt itself fails (e.g. an unreadable
// overrides/ directory) so `shark admin upgrade` still emits a schema-stable
// "overrides" key/line rather than dropping it.
func zeroOverridesSummary() map[string]int {
	summary := make(map[string]int, len(overridesClassificationOrder))
	for _, c := range overridesClassificationOrder {
		summary[c] = 0
	}
	return summary
}

func runSharkValidate(cmd *cobra.Command, _ []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("shark admin validate-data: failed to locate project root: %w", err)
	}

	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return fmt.Errorf("shark admin validate-data: %w", err)
	}

	report, err := sharkdata.ValidateAt(dataRoot)
	if err != nil {
		return fmt.Errorf("shark admin validate-data: internal failure: %w", err)
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
		return cli.OutputJSON(map[string]interface{}{
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
