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

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

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

	dest, err := sharkdata.Init(root)
	if err != nil {
		if errors.Is(err, sharkdata.ErrAlreadyInitialized) {
			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(map[string]interface{}{
					"status": "already_initialized",
					"path":   dest,
				})
			}
			fmt.Printf("shark-data/ already exists at %s. Run 'shark upgrade' to refresh.\n", dest)
			return nil
		}
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status": "initialized",
			"path":   dest,
		})
	}
	fmt.Printf("Initialized shark-data/ at %s\n", dest)
	return nil
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
