package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gitutil"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// approvedPlanForBug is shared by keyed next and the in-process runner so both
// dispatch paths provide the same parent-retained plan revision.
func approvedPlanForBug(ctx context.Context, key, workingDir string) (runner.ApprovedPlan, error) {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return runner.ApprovedPlan{}, fmt.Errorf("locate Shark project for approved plan: %w", err)
	}
	notes, err := cli.GetNoteService(ctx)
	if err != nil {
		return runner.ApprovedPlan{}, fmt.Errorf("read approved plan notes: %w", err)
	}
	wf := cli.GetWorkflowService().ForLevel("bug")
	return runner.ApprovedBugPlan(ctx, root, workingDir, key, notes, cli.GetEntityHistoryService(), wf.NormalizeStatus, runner.BugApprovalTargetResolver(wf))
}

func bugParentProjectInstructions(ctx context.Context, prompt, workingDir string) (string, error) {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return "", err
	}
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	command := "shark"
	configPath := cli.GlobalConfig.ConfigFile
	if configPath != "" {
		configPath, err = filepath.Abs(configPath)
		if err != nil {
			return "", err
		}
		command += " --config " + quote(configPath)
	}
	if flag := cli.RootCmd.PersistentFlags().Lookup("db"); flag != nil && flag.Changed {
		dbPath := cli.GlobalConfig.DBPath
		if !filepath.IsAbs(dbPath) {
			base := root
			if configPath != "" {
				base = filepath.Dir(configPath)
			}
			dbPath = filepath.Join(base, dbPath)
		}
		command += " --db " + quote(dbPath)
	}
	source, err := bugSourceWorkSentence(ctx, root, workingDir, quote)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n\nFor Shark record, related-document, and bundled skill/agent commands, run %s from the parent project directory %s, followed by the desired subcommand. %s\n", prompt, command, quote(root), source), nil
}

// bugSourceWorkSentence names the directory (and branch) a bug worker must edit,
// so the worker never has to guess which checkout it was assigned.
func bugSourceWorkSentence(ctx context.Context, root, workingDir string, quote func(string) string) (string, error) {
	if workingDir == "" {
		return fmt.Sprintf("No separate worker working directory was assigned: this dispatch is in the parent tree, so source work belongs in %s%s.", quote(root), gitBranchClause(ctx, root)), nil
	}
	dir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("resolve worker working directory %q: %w", workingDir, err)
	}
	return fmt.Sprintf("Keep source work in the assigned worker working directory %s%s.", quote(dir), gitBranchClause(ctx, dir)), nil
}

// gitBranchClause reports the checked-out branch of dir. It returns an empty
// clause when dir is not a Git checkout, rather than inventing a branch.
func gitBranchClause(ctx context.Context, dir string) string {
	out, err := gitutil.Command(ctx, "-c", "core.fsmonitor=false", "-C", dir, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
	if err == nil {
		return fmt.Sprintf(" (branch %q)", strings.TrimSpace(string(out)))
	}
	if gitutil.Command(ctx, "-c", "core.fsmonitor=false", "-C", dir, "rev-parse", "--verify", "--quiet", "HEAD").Run() == nil {
		return " (detached HEAD)"
	}
	return ""
}
