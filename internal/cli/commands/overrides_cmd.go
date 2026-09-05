// Package commands provides CLI command implementations.
//
// This file wires the override drift-visibility classifier and baseline
// acknowledge mechanism (internal/sharkdata/overrides_status.go and
// override_baseline.go) into the cobra command tree under `shark admin
// overrides`. All classification/write logic lives in internal/sharkdata;
// these handlers are thin adapters, matching the `cloudCmd`/`configCmd`
// nested-group pattern in internal/cli/commands/cloud.go.
//
// Spec: E34-F09 (Override Drift Visibility and WWGM Reconciliation),
// REQ-F-001 (status) and REQ-F-002 (acknowledge).
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

// overridesCmd is the parent command for override drift-visibility and
// baseline reconciliation subcommands.
var overridesCmd = &cobra.Command{
	Use:   "overrides",
	Short: "Inspect and reconcile local shark-data overrides against canonical defaults",
	Long: `Commands for classifying override drift and recording baselines.

Every file under <shark_data_path>/overrides/ is compared against its
canonical (embedded) counterpart and classified as one of: current,
upstream_changed, identical_redundant, orphaned, or baseline_unknown.

Subcommands:
  status       Show drift classification for every override
  acknowledge  Record the current canonical digest as the new baseline`,
}

// overridesStatusCmd reports the drift classification of every override.
var overridesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show drift classification for every file under overrides/",
	Long: `Walk <shark_data_path>/overrides/ and classify each file's drift state
relative to its canonical counterpart. Read-only: no file is written.

Examples:
  shark admin overrides status
  shark admin overrides status --json`,
	Args: cobra.NoArgs,
	RunE: runOverridesStatus,
}

// overridesAcknowledgeCmd records a fresh baseline digest for one or more
// override paths.
var overridesAcknowledgeCmd = &cobra.Command{
	Use:   "acknowledge <relative-override-path>...",
	Short: "Record the current canonical digest as the baseline for one or more overrides",
	Long: `Record the current canonical SHA-256 digest as the recorded baseline for
each given path, reclassifying it as "current" on the next status check.

Each path must have both a regular override file at
<shark_data_path>/overrides/<path> and a canonical counterpart; a path
failing either check aborts the whole call with zero manifest mutation.
Acknowledge never touches override file bytes.

Examples:
  shark admin overrides acknowledge workflow/sprint.yaml
  shark admin overrides acknowledge workflow/sprint.yaml prompts/feature/qa.md --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runOverridesAcknowledge,
}

func init() {
	adminCmd.AddCommand(overridesCmd)
	overridesCmd.AddCommand(overridesStatusCmd)
	overridesCmd.AddCommand(overridesAcknowledgeCmd)
}

// resolveOverridesDataRoot resolves the project root and shark-data root
// exactly as runSharkUpgrade does in sharkdata_cmd.go (REQ-F-001, AC-T1):
// cli.FindProjectRoot() then config.ResolveSharkDataRoot(root, configBytes).
func resolveOverridesDataRoot() (string, error) {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return "", fmt.Errorf("shark admin overrides: failed to locate project root: %w", err)
	}

	configBytes, _ := os.ReadFile(filepath.Join(root, ".sharkconfig.json")) // missing/unreadable config is fine: ResolveSharkDataRoot defaults to <root>/shark-data
	dataRoot, err := config.ResolveSharkDataRoot(root, configBytes)
	if err != nil {
		return "", fmt.Errorf("shark admin overrides: %w", err)
	}
	return dataRoot, nil
}

func runOverridesStatus(_ *cobra.Command, _ []string) error {
	dataRoot, err := resolveOverridesDataRoot()
	if err != nil {
		return err
	}

	report, err := sharkdata.OverrideStatusAt(dataRoot)
	if err != nil {
		return fmt.Errorf("shark admin overrides status: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(report)
	}

	printOverrideStatusReport(report)
	return nil
}

func runOverridesAcknowledge(_ *cobra.Command, args []string) error {
	dataRoot, err := resolveOverridesDataRoot()
	if err != nil {
		return err
	}

	report, err := sharkdata.AcknowledgeOverrides(dataRoot, args)
	if err != nil {
		return fmt.Errorf("shark admin overrides acknowledge: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(report)
	}

	fmt.Printf("Acknowledged %d override(s):\n", len(args))
	for _, p := range args {
		fmt.Printf("  %s\n", p)
	}
	printOverrideStatusReport(report)
	return nil
}

// printOverrideStatusReport renders the human-readable summary and per-row
// detail shared by both `status` and `acknowledge` (the latter prints the
// refreshed report after recording new baselines).
func printOverrideStatusReport(report *sharkdata.OverrideStatusReport) {
	fmt.Println("Override status:")
	fmt.Printf("  current:             %d\n", report.Summary[sharkdata.ClassificationCurrent])
	fmt.Printf("  upstream_changed:    %d\n", report.Summary[sharkdata.ClassificationUpstreamChanged])
	fmt.Printf("  identical_redundant: %d\n", report.Summary[sharkdata.ClassificationIdenticalRedundant])
	fmt.Printf("  orphaned:            %d\n", report.Summary[sharkdata.ClassificationOrphaned])
	fmt.Printf("  baseline_unknown:    %d\n", report.Summary[sharkdata.ClassificationBaselineUnknown])

	if len(report.Rows) == 0 {
		fmt.Println("  (no overrides found)")
		return
	}
	for _, row := range report.Rows {
		if row.SuggestedAction == "" {
			fmt.Printf("  [%s] %s\n", row.Classification, row.Path)
			continue
		}
		fmt.Printf("  [%s] %s -- %s\n", row.Classification, row.Path, row.SuggestedAction)
	}
}
