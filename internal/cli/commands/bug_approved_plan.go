package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
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
	return runner.ApprovedBugPlan(ctx, root, workingDir, key, notes, cli.GetEntityHistoryService(), cli.GetWorkflowService().ForLevel("bug").NormalizeStatus)
}

func bugParentProjectInstructions(prompt string) (string, error) {
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
	return fmt.Sprintf("%s\n\nFor Shark record and related-document commands, run %s from the parent project directory %s, followed by the desired subcommand. Keep source work in the assigned worker working directory.\n", prompt, command, quote(root)), nil
}
