// Package commands provides CLI command implementations.
// This file implements the `shark run <entity-key>` command, which drives an
// entity through its workflow by dispatching AI agents for each status stage.
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

var (
	runDryRun   bool
	runVerbose  bool
	runWorkDir  string
	runWorktree bool
)

var runCmd = &cobra.Command{
	Use:   "run <entity-key>",
	Short: "Run the orchestration loop for an entity",
	Long: `Drive an entity through its workflow by dispatching AI agents.

The run command reads the current status of the entity, looks up the
orchestrator action for that status, dispatches the appropriate AI agent,
and advances the status upon success. The loop continues until the entity
reaches a terminal status or a pause/wait action is encountered.

Examples:
  shark run E07-F01-001          # Run a task through its workflow
  shark run E07-F01              # Run a feature
  shark run E07                  # Run an epic
  shark run B001                 # Run a bug
  shark run CC-001               # Run a change card
  shark run E07-F01-001 --dry-run    # Preview actions without dispatching
  shark run E07-F01-001 --verbose    # Show detailed stage progress
  shark run E07-F01-001 --worktree   # Run agent in an isolated git worktree`,
	Args: cobra.ExactArgs(1),
	RunE: runRun,
}

func init() {
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Preview actions without dispatching agents or advancing status")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Show detailed stage progress")
	runCmd.Flags().StringVar(&runWorkDir, "workdir", "", "Working directory override for agent processes")
	runCmd.Flags().BoolVar(&runWorktree, "worktree", false, "Create an isolated git worktree for agent dispatch and clean up on completion")
	cli.RootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	entityKey := args[0]

	// Step 1: Detect entity type from key format.
	entityType, normalizedKey, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid entity key %q: %w", entityKey, err)
	}

	// Step 2: Build entity-type adapters.
	transitioner, err := buildTransitioner(ctx, entityType)
	if err != nil {
		return fmt.Errorf("failed to build transitioner for %s: %w", entityType, err)
	}

	placeholderGen := buildPlaceholderGenerator(ctx, entityType)

	// Step 3: Get shared services.
	actionSvc, err := cli.GetActionService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize action service: %w", err)
	}

	workflowSvc := cli.GetWorkflowService()

	// Step 4: Build dispatcher map (REQ-F02-011).
	claudeDispatcher := runner.NewClaudeDispatcher()
	codexDispatcher := runner.NewCodexDispatcher()
	dispatchers := map[string]runner.AgentDispatcher{
		"":          claudeDispatcher,
		"anthropic": claudeDispatcher,
		"codex":     codexDispatcher,
		"openai":    codexDispatcher,
	}

	// Step 5: Determine working directory, creating a git worktree if requested.
	workingDir := runWorkDir
	if runWorktree {
		creator := runner.NewGitWorktreeCreator()
		var worktreePath string
		workingDir, worktreePath, err = setupWorktree(ctx, normalizedKey, creator)
		if err != nil {
			return fmt.Errorf("failed to create worktree for %s: %w", normalizedKey, err)
		}
		defer func() {
			if removeErr := creator.RemoveWorktree(context.Background(), worktreePath); removeErr != nil {
				fmt.Printf("warning: failed to remove worktree %s: %v\n", worktreePath, removeErr)
			}
		}()
	}

	// Step 6: Construct and run the controller.
	controller, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: transitioner,
		Placeholders: placeholderGen,
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflowSvc,
		Dispatchers:  dispatchers,
	})
	if err != nil {
		return fmt.Errorf("failed to create run controller: %w", err)
	}

	opts := runner.RunOptions{
		DryRun:     runDryRun,
		Verbose:    runVerbose,
		WorkingDir: workingDir,
	}

	result, err := controller.Run(ctx, normalizedKey, opts)
	if err != nil {
		return fmt.Errorf("run failed for %s: %w", normalizedKey, err)
	}

	// Step 7: Format output.
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	// Human-readable output.
	fmt.Printf("Run complete for %s\n", result.EntityKey)
	fmt.Printf("  Outcome:    %s\n", result.Outcome)
	fmt.Printf("  Status:     %s\n", result.FinalStatus)
	fmt.Printf("  Stages:     %d completed\n", result.StagesCompleted)
	fmt.Printf("  Duration:   %s\n", result.TotalDuration)

	if len(result.Stages) > 0 {
		fmt.Println("\nStage details:")
		for i, stage := range result.Stages {
			fmt.Printf("  [%d] %s (%s", i+1, stage.Status, stage.Action)
			if stage.AgentType != "" {
				fmt.Printf(", agent=%s", stage.AgentType)
			}
			if stage.Provider != "" {
				fmt.Printf(", provider=%s", stage.Provider)
			}
			fmt.Printf(", exit=%d, duration=%s)\n", stage.ExitCode, stage.Duration)
			if stage.OutputSummary != "" {
				fmt.Printf("      output: %s\n", runner.TruncateOutput(stage.OutputSummary, 120))
			}
		}
	}

	return nil
}

// setupWorktree creates a git worktree for the given entity key and returns
// the working directory (same as worktree path) and the worktree path for later cleanup.
// The creator parameter allows injection in tests.
func setupWorktree(ctx context.Context, entityKey string, creator runner.WorktreeCreator) (workingDir, worktreePath string, err error) {
	worktreePath, branch := runner.WorktreePaths(runner.DefaultWorktreeBaseDir, entityKey, time.Now())
	if createErr := creator.CreateWorktree(ctx, "", worktreePath, branch); createErr != nil {
		return "", "", fmt.Errorf("git worktree add failed: %w", createErr)
	}
	return worktreePath, worktreePath, nil
}

// buildTransitioner returns an EntityTransitioner for the given entity type.
func buildTransitioner(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
	switch entityType {
	case "task":
		return &taskTransitionerAdapter{svc: cli.GetTaskService()}, nil
	case "feature":
		return &featureTransitionerAdapter{svc: cli.GetFeatureService()}, nil
	case "epic":
		return &epicTransitionerAdapter{svc: cli.GetEpicService()}, nil
	case "bug":
		return &bugTransitionerAdapter{svc: cli.GetBugService()}, nil
	case "change", "change_card":
		return &changeCardTransitionerAdapter{svc: cli.GetChangeCardService()}, nil
	default:
		return nil, fmt.Errorf("unsupported entity type: %q", entityType)
	}
}

// buildPlaceholderGenerator returns a PlaceholderGenerator for the given entity type.
// Returns nil for unsupported types (controller handles nil gracefully).
func buildPlaceholderGenerator(_ context.Context, entityType string) runner.PlaceholderGenerator {
	switch entityType {
	case "task":
		return &taskPlaceholderAdapter{svc: cli.GetTaskService()}
	case "feature":
		return &featurePlaceholderAdapter{svc: cli.GetFeatureService()}
	case "epic":
		return &epicPlaceholderAdapter{svc: cli.GetEpicService()}
	case "bug":
		return &bugPlaceholderAdapter{svc: cli.GetBugService()}
	case "change", "change_card":
		return &changeCardPlaceholderAdapter{svc: cli.GetChangeCardService()}
	default:
		return nil
	}
}

// ─── Task adapters ────────────────────────────────────────────────────────────

type taskTransitionerAdapter struct {
	svc *services.TaskService
}

func (a *taskTransitionerAdapter) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return a.svc.TransitionStatus(ctx, key, targetStatus, opts)
}

func (a *taskTransitionerAdapter) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return a.svc.GetNextStatus(ctx, key)
}

type taskPlaceholderAdapter struct {
	svc *services.TaskService
}

func (a *taskPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	task, err := a.svc.GetTask(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s for placeholders: %w", key, err)
	}
	return config.TaskPlaceholders(task), nil
}

// ─── Feature adapters ─────────────────────────────────────────────────────────

type featureTransitionerAdapter struct {
	svc *services.FeatureService
}

func (a *featureTransitionerAdapter) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return a.svc.TransitionStatus(ctx, key, targetStatus, opts)
}

func (a *featureTransitionerAdapter) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return a.svc.GetNextStatus(ctx, key)
}

type featurePlaceholderAdapter struct {
	svc *services.FeatureService
}

func (a *featurePlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	feature, err := a.svc.GetFeature(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s for placeholders: %w", key, err)
	}
	return config.FeaturePlaceholders(feature), nil
}

// ─── Epic adapters ────────────────────────────────────────────────────────────

type epicTransitionerAdapter struct {
	svc *services.EpicService
}

func (a *epicTransitionerAdapter) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return a.svc.TransitionStatus(ctx, key, targetStatus, opts)
}

func (a *epicTransitionerAdapter) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return a.svc.GetNextStatus(ctx, key)
}

type epicPlaceholderAdapter struct {
	svc *services.EpicService
}

func (a *epicPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	epic, err := a.svc.GetEpic(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s for placeholders: %w", key, err)
	}
	return config.EpicPlaceholders(epic), nil
}

// ─── Bug adapters ─────────────────────────────────────────────────────────────

// bugTransitionerAdapter adapts BugService to the EntityTransitioner interface.
// BugService does not have TransitionStatus/GetNextStatus, so we synthesise them
// from the lower-level AdvanceBugStatus / SetBugStatus methods.
type bugTransitionerAdapter struct {
	svc *services.BugService
}

func (a *bugTransitionerAdapter) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	bug, err := a.svc.SetBugStatus(ctx, key, targetStatus, opts.Force)
	if err != nil {
		return nil, fmt.Errorf("failed to transition bug %s to %s: %w", key, targetStatus, err)
	}
	return &services.TransitionResult{
		EntityType:   "bug",
		EntityKey:    key,
		FromStatus:   "", // unknown before set
		ToStatus:     string(bug.Status),
		Transitioned: true,
	}, nil
}

func (a *bugTransitionerAdapter) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	bug, err := a.svc.GetBug(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}
	currentStatus := string(bug.Status)
	validTransitions := a.svc.GetValidTransitions(currentStatus)

	info := &services.NextStatusInfo{
		EntityType:    "bug",
		EntityKey:     key,
		CurrentStatus: currentStatus,
		IsTerminal:    len(validTransitions) == 0,
	}
	return info, nil
}

type bugPlaceholderAdapter struct {
	svc *services.BugService
}

func (a *bugPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	bug, err := a.svc.GetBug(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s for placeholders: %w", key, err)
	}
	return config.BugPlaceholders(bug), nil
}

// ─── ChangeCard adapters ──────────────────────────────────────────────────────

// changeCardTransitionerAdapter adapts ChangeCardService to the EntityTransitioner interface.
type changeCardTransitionerAdapter struct {
	svc *services.ChangeCardService
}

func (a *changeCardTransitionerAdapter) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	card, err := a.svc.SetChangeCardStatus(ctx, key, targetStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to transition change card %s to %s: %w", key, targetStatus, err)
	}
	return &services.TransitionResult{
		EntityType:   "change_card",
		EntityKey:    key,
		FromStatus:   "",
		ToStatus:     string(card.Status),
		Transitioned: true,
	}, nil
}

func (a *changeCardTransitionerAdapter) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	card, err := a.svc.GetChangeCard(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change card %s: %w", key, err)
	}
	currentStatus := string(card.Status)
	validTransitions := a.svc.GetValidTransitions(currentStatus)

	info := &services.NextStatusInfo{
		EntityType:    "change_card",
		EntityKey:     key,
		CurrentStatus: currentStatus,
		IsTerminal:    len(validTransitions) == 0,
	}
	return info, nil
}

type changeCardPlaceholderAdapter struct {
	svc *services.ChangeCardService
}

func (a *changeCardPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	card, err := a.svc.GetChangeCard(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change card %s for placeholders: %w", key, err)
	}
	return config.ChangeCardPlaceholders(card), nil
}

// ─── compile-time interface assertions ────────────────────────────────────────

var (
	_ runner.EntityTransitioner = (*taskTransitionerAdapter)(nil)
	_ runner.EntityTransitioner = (*featureTransitionerAdapter)(nil)
	_ runner.EntityTransitioner = (*epicTransitionerAdapter)(nil)
	_ runner.EntityTransitioner = (*bugTransitionerAdapter)(nil)
	_ runner.EntityTransitioner = (*changeCardTransitionerAdapter)(nil)

	_ runner.PlaceholderGenerator = (*taskPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*featurePlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*epicPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*bugPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*changeCardPlaceholderAdapter)(nil)
)
