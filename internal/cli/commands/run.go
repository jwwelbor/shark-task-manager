// Package commands provides CLI command implementations.
// This file implements the `shark run <entity-key>` command, which drives an
// entity through its workflow by dispatching AI agents for each status stage.
package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

var (
	runDryRun   bool
	runVerbose  bool
	runWorkDir  string
	runWorktree bool
)

type runClaimServicer interface {
	Claim(ctx context.Context, in services.ClaimInput) (*models.EntityClaim, error)
	Release(ctx context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error)
	Heartbeat(ctx context.Context, entityType, entityKey, sessionID string, progress *float64, note string) error
	TTL() time.Duration
}

var runClaimSvcOverride runClaimServicer

func getRunClaimService() runClaimServicer {
	if runClaimSvcOverride != nil {
		return runClaimSvcOverride
	}
	return cli.GetClaimService()
}

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

	// ── Observability: generate run_id and emit run.start / run.end (T-E07-F41-002) ──
	//
	// Load observability config. Non-fatal: if config is unavailable the emitters
	// are no-ops (obs.Enabled defaults to false).
	var obs config.ObservabilityConfig
	if cfg, cfgErr := cli.GetConfig(); cfgErr == nil {
		obs = cfg.GetObservability()
	}

	runID := generateRunID()

	// Step 1: Detect entity type from key format.
	entityType, normalizedKey, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid entity key %q: %w", entityKey, err)
	}

	// Emit run.start now that we know entity_type.
	emitRunStart(obs, runStartParams{
		Args:         args,
		EntityKey:    normalizedKey,
		EntityType:   entityType,
		DryRun:       runDryRun,
		Worktree:     runWorktree,
		WorktreePath: runWorkDir,
		RunID:        runID,
	})

	// run.end is deferred so it fires on every return path (AC-T3).
	// We use a pointer to a *RunResult so the defer closure captures the
	// final value written by the controller. durationStart captures wall time.
	var runResult *runner.RunResult
	var runLease *activeRunLease
	runStart := time.Now()
	defer func() {
		durationMS := time.Since(runStart).Milliseconds()
		r := runResult
		if r == nil {
			// Error return before controller ran; emit a minimal failed event.
			r = &runner.RunResult{
				EntityKey: normalizedKey,
				Outcome:   "failed",
				Error:     "run did not complete",
			}
		}
		emitRunEnd(ctx, obs, runID, r, durationMS)
	}()
	defer func() {
		if runLease == nil {
			return
		}
		outcome := "failed"
		if runResult != nil && runResult.Outcome != "" {
			outcome = runResult.Outcome
		}
		if err := runLease.Release(outcome); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to release run claim for %s: %v\n", normalizedKey, err)
		}
	}()

	runLease, err = acquireRunLease(ctx, entityType, normalizedKey, runDryRun)
	if err != nil {
		return fmt.Errorf("claim %s before run: %w", normalizedKey, err)
	}

	// Step 2: Build entity-type adapters.
	transitioner, err := buildTransitioner(ctx, entityType)
	if err != nil {
		return fmt.Errorf("failed to build transitioner for %s: %w", entityType, err)
	}

	placeholderGen := buildPlaceholderGenerator(ctx, entityType)

	// Step 3: Get shared services. Narrow the action service to this entity
	// type so status lookups in the run loop resolve against the right
	// per-entity workflow (cross-entity status name collisions like
	// "completed" become unambiguous).
	actionSvcRoot, err := cli.GetActionService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize action service: %w", err)
	}
	actionSvc := narrowActionServiceForEntity(actionSvcRoot, entityType)
	stepResolver, err := dispatch.NewStepResolver(dispatch.StepResolverDeps{
		Transitioner:  transitioner,
		Placeholders:  placeholderGen,
		ActionService: actionSvc,
		IsArchivedStatus: func(_ models.EntityType, status string) bool {
			return isArchivedStatus(entityType, status)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize dispatch-step resolver for %s: %w", entityType, err)
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
		StepResolver: stepResolver,
		EntityType:   models.EntityType(entityType),
		WorkflowSvc:  workflowSvc,
		Dispatchers:  dispatchers,
		PromptAssembler: runner.PromptAssemblerFunc(func(ctx context.Context, input runner.PromptAssemblyInput) (string, error) {
			return assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create run controller: %w", err)
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to locate project root: %w", err)
	}

	opts := runner.RunOptions{
		DryRun:        runDryRun,
		Verbose:       runVerbose,
		WorkingDir:    workingDir,
		RunID:         runID,
		ProjectRoot:   projectRoot,
		Observability: obs,
	}

	result, err := controller.Run(ctx, normalizedKey, opts)
	if err != nil {
		return fmt.Errorf("run failed for %s: %w", normalizedKey, err)
	}
	runResult = result // captured for deferred run.end emitter

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
	if result.Error != "" {
		fmt.Printf("  Error:      %s\n", result.Error)
	}

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

type activeRunLease struct {
	svc             runClaimServicer
	entityType      string
	entityKey       string
	sessionID       string
	stopHeartbeat   context.CancelFunc
	heartbeatDoneCh <-chan struct{}
}

func acquireRunLease(ctx context.Context, entityType, entityKey string, dryRun bool) (*activeRunLease, error) {
	if dryRun {
		return nil, nil
	}

	svc := getRunClaimService()
	claim, err := svc.Claim(ctx, services.ClaimInput{
		EntityType: entityType,
		EntityKey:  entityKey,
	})
	if err != nil {
		return nil, err
	}

	hbCtx, stopHeartbeat := context.WithCancel(ctx)
	doneCh := startRunHeartbeat(hbCtx, svc, entityType, entityKey, claim.SessionID)
	return &activeRunLease{
		svc:             svc,
		entityType:      entityType,
		entityKey:       entityKey,
		sessionID:       claim.SessionID,
		stopHeartbeat:   stopHeartbeat,
		heartbeatDoneCh: doneCh,
	}, nil
}

func (l *activeRunLease) Release(outcome string) error {
	if l == nil {
		return nil
	}
	if l.stopHeartbeat != nil {
		l.stopHeartbeat()
	}
	if l.heartbeatDoneCh != nil {
		<-l.heartbeatDoneCh
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := l.svc.Release(ctx, l.entityType, l.entityKey, l.sessionID, outcome, false)
	return err
}

func startRunHeartbeat(ctx context.Context, svc runClaimServicer, entityType, entityKey, sessionID string) <-chan struct{} {
	done := make(chan struct{})
	interval := svc.TTL() / 3
	if interval < time.Second {
		interval = time.Second
	}

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := svc.Heartbeat(ctx, entityType, entityKey, sessionID, nil, "shark run active"); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to heartbeat run claim for %s: %v\n", entityKey, err)
				}
			}
		}
	}()

	return done
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
// All five service types (TaskService, FeatureService, EpicService, BugService,
// ChangeCardService) directly satisfy runner.EntityTransitioner via their
// TransitionStatus and GetNextStatus methods.
func buildTransitioner(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
	switch entityType {
	case "task":
		return cli.GetTaskService(), nil
	case "feature":
		return cli.GetFeatureService(), nil
	case "epic":
		return cli.GetEpicService(), nil
	case "bug":
		return cli.GetBugService(), nil
	case "change", "change_card":
		return cli.GetChangeCardService(), nil
	case "tech_debt":
		return cli.GetTechDebtService(), nil
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
	case "tech_debt":
		return &techDebtPlaceholderAdapter{svc: cli.GetTechDebtService()}
	default:
		return nil
	}
}

// ─── Placeholder adapters ─────────────────────────────────────────────────────
// Placeholder adapters remain entity-specific because each generates different
// placeholder maps (task vs feature vs epic vs bug vs change-card fields).

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

type techDebtPlaceholderAdapter struct {
	svc *services.TechDebtService
}

func (a *techDebtPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	td, err := a.svc.GetTechDebt(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt %s for placeholders: %w", key, err)
	}
	return config.TechDebtPlaceholders(td), nil
}

// ─── compile-time interface assertions ────────────────────────────────────────

var (
	// Services satisfy EntityTransitioner directly (no adapter needed).
	_ runner.EntityTransitioner = (*services.TaskService)(nil)
	_ runner.EntityTransitioner = (*services.FeatureService)(nil)
	_ runner.EntityTransitioner = (*services.EpicService)(nil)
	_ runner.EntityTransitioner = (*services.BugService)(nil)
	_ runner.EntityTransitioner = (*services.ChangeCardService)(nil)
	_ runner.EntityTransitioner = (*services.TechDebtService)(nil)

	// Placeholder adapters remain entity-specific.
	_ runner.PlaceholderGenerator = (*taskPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*featurePlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*epicPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*bugPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*changeCardPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*techDebtPlaceholderAdapter)(nil)
)
