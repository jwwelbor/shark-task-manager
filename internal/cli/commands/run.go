// Package commands provides CLI command implementations.
// This file implements the `shark run <entity-key>` command, which drives an
// entity through its workflow by dispatching AI agents for each status stage.
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

var (
	runDryRun   bool
	runVerbose  bool
	runWorkDir  string
	runWorktree bool
	runResumeID string
	runSession  string
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
	runCmd.Flags().String(
		"harness",
		"",
		"Override the resolved harness type (e.g. claude, codex); wins over the active claim and SHARK_HARNESS",
	)
	runCmd.Flags().String(
		"harness-version",
		"",
		"Override the resolved harness version; wins over the active claim and SHARK_HARNESS_VERSION",
	)
	runCmd.Flags().String(
		"harness-model",
		"",
		"Override the resolved harness model; wins over the active claim and SHARK_HARNESS_MODEL",
	)
	runCmd.Flags().StringVar(&runResumeID, "resume-run", "", "Report durable resume status for an existing run_id's GateResult sidecar (T-E34-F05-002); accepts no new result bytes and does not dispatch or transition")
	runCmd.Flags().StringVar(&runSession, "session", "", "Authorized session id for --resume-run")
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

	// Non-fatal project-root lookup for the liveness recorder (D6 edit 1): an
	// empty root disables the recorder's file sink only (D5), mirroring
	// NewFileJSONLExporter("")'s silent-skip convention. Separate from the
	// hard-error lookup below (used for opts.ProjectRoot) — moving or
	// softening that call would change which error a malformed key produces.
	recorderProjectRoot, _ := cli.FindProjectRoot()

	// Step 1: Detect entity type from key format.
	entityType, normalizedKey, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid entity key %q: %w", entityKey, err)
	}

	// T-E34-F05-002: --resume-run reports the durable GateResult sidecar's
	// resume status and returns without dispatching an agent, claiming a
	// lease, or applying any transition — it "accepts no new result bytes"
	// per the task spec. Full resume (re-applying the guarded transition
	// and releasing the lease) is T-E34-F05-003/004's persistence
	// coordinator; this is the read-only status/decision surface it and
	// operators consume. Short-circuits before any claim/dispatch state is
	// touched.
	if runResumeID != "" {
		return runResumeRun(entityType, normalizedKey)
	}

	// T-E34-F05-004: --apply-result is Rider's initial-ingestion surface. It
	// calls the same runner.IngestGateResult boundary the core runner calls
	// directly, and short-circuits before any claim/dispatch state is
	// touched, matching --resume-run's contract above.
	if runApplyResultSet() {
		return runApplyResult(cmd, entityType, normalizedKey)
	}

	// Read the --harness/--harness-version/--harness-model override flags
	// once, per spec.md §3.3 AC-T2. Required for REQ-F-006/AC-08: without
	// this, precedence tier 1 (flags) has no entry point under `shark run`.
	harnessOverride, err := harnessOverrideFromFlags(cmd)
	if err != nil {
		return err
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

	// Construct and start the liveness recorder (D6 edit 2): after
	// emitRunStart (topLevelKey/runID are now known) and before preflight, so
	// a run that pauses at a Question block still leaves a log. Start()
	// announces LogPath() on stderr and launches the fixed 10s heartbeat
	// ticker (T-E40-F04-003). Teardown is a closure, never
	// `defer rec.Finish(runResult)` — a direct method-value defer evaluates
	// its argument at registration time and would capture nil forever (the
	// emitRunEnd defer above documents this exact trap). Stop() precedes
	// Finish() so the ticker cannot race a final stage_end.
	rec := runner.NewLivenessRecorder(recorderProjectRoot, runID, normalizedKey, cli.GlobalConfig.JSON, runStart)
	rec.Start()
	defer func() { rec.Stop(); rec.Finish(runResult) }()

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

	// A Question's responder identity is meaningful only for a dispatchable
	// worker action. Check the current status/action before deriving that
	// identity or acquiring a lease: a ready-for-resolution checkpoint (and
	// every other pause-only action) must remain claim-free.
	questionBlocker := cli.GetQuestionBlocker()
	childrenSvc := cli.GetCascadeService()
	if cascadeBlock, preflightStatus, err := preflightCascadeQuestionBlock(ctx, transitioner, actionSvcRoot, childrenSvc, questionBlocker, buildTransitioner, entityType, normalizedKey); err != nil {
		return fmt.Errorf("preflight cascade Question block for %s: %w", normalizedKey, err)
	} else if cascadeBlock != nil {
		runResult = &runner.RunResult{
			EntityKey: normalizedKey, FinalStatus: preflightStatus, Outcome: "paused", QuestionBlock: cascadeBlock,
		}
		return outputRunResult(runResult)
	}
	runLease, questionBlock, preflightStatus, err := acquireRunLeaseForRunnableAction(ctx, transitioner, actionSvc, questionBlocker, entityType, normalizedKey, runDryRun, harnessOverride)
	if err != nil {
		return fmt.Errorf("claim %s before run: %w", normalizedKey, err)
	}
	if questionBlock != nil {
		runResult = &runner.RunResult{
			EntityKey: normalizedKey, FinalStatus: preflightStatus, Outcome: "paused", QuestionBlock: questionBlock,
		}
		return outputRunResult(runResult)
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
	var runChild runner.CascadeChildRunner
	runChild = func(ctx context.Context, childType, key string, childOpts runner.RunOptions) (*runner.RunResult, error) {
		childTransitioner, err := buildTransitioner(ctx, childType)
		if err != nil {
			return nil, fmt.Errorf("failed to build transitioner for %s %s: %w", childType, key, err)
		}
		childPlaceholderGen := buildPlaceholderGenerator(ctx, childType)
		childActionSvc := narrowActionServiceForEntity(actionSvcRoot, childType)

		childController, err := runner.NewRunController(runner.RunControllerDeps{
			Transitioner: childTransitioner,
			Placeholders: childPlaceholderGen,
			ActionSvc:    childActionSvc,
			WorkflowSvc:  workflowSvc,
			Dispatchers:  dispatchers,
			PromptAssembler: runner.PromptAssemblerFunc(func(ctx context.Context, input runner.PromptAssemblyInput) (string, error) {
				return assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
			}),
			ChildrenSvc:       childrenSvc,
			RunChild:          runChild,
			QuestionResponses: buildQuestionResponsePersister(childType),
			QuestionBlocker:   questionBlocker,
			HarnessResolver:   cli.GetHarnessResolver(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create cascade child controller for %s %s: %w", childType, key, err)
		}

		// Cascade children use the same preflight as the top-level run. In
		// particular, a parked Question must not attempt responder lookup or
		// create a lease before its pause action is observed.
		if cascadeBlock, cascadeStatus, err := preflightCascadeQuestionBlock(ctx, childTransitioner, actionSvcRoot, childrenSvc, questionBlocker, buildTransitioner, childType, key); err != nil {
			return nil, fmt.Errorf("preflight cascade Question block for %s %s: %w", childType, key, err)
		} else if cascadeBlock != nil {
			return &runner.RunResult{EntityKey: key, FinalStatus: cascadeStatus, Outcome: "paused", QuestionBlock: cascadeBlock}, nil
		}
		childLease, childBlock, childStatus, err := acquireRunLeaseForRunnableAction(ctx, childTransitioner, childActionSvc, questionBlocker, childType, key, childOpts.DryRun, childOpts.HarnessOverride)
		if err != nil {
			if errors.Is(err, claimrepo.ErrAlreadyClaimed) {
				return &runner.RunResult{
					EntityKey: key,
					Outcome:   "paused",
					Error:     fmt.Sprintf("cascade child %s %s is already claimed", childType, key),
				}, nil
			}
			return nil, fmt.Errorf("claim cascade child %s %s: %w", childType, key, err)
		}
		if childBlock != nil {
			return &runner.RunResult{EntityKey: key, FinalStatus: childStatus, Outcome: "paused", QuestionBlock: childBlock}, nil
		}
		childOpts.EntityType = childType
		if childLease != nil {
			childOpts.SessionID = childLease.sessionID
		}
		childResult, runErr := childController.Run(ctx, key, childOpts)
		outcome := "failed"
		if childResult != nil && childResult.Outcome != "" {
			outcome = childResult.Outcome
		}
		if childLease != nil {
			if releaseErr := childLease.Release(outcome); releaseErr != nil {
				return nil, fmt.Errorf("release cascade child claim for %s %s: %w", childType, key, releaseErr)
			}
		}
		if runErr != nil {
			return nil, runErr
		}
		return childResult, nil
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
				fmt.Fprintf(os.Stderr, "warning: failed to remove worktree %s: %v\n", worktreePath, removeErr)
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
		PromptAssembler: runner.PromptAssemblerFunc(func(ctx context.Context, input runner.PromptAssemblyInput) (string, error) {
			return assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
		}),
		ChildrenSvc:       childrenSvc,
		RunChild:          runChild,
		QuestionResponses: buildQuestionResponsePersister(entityType),
		QuestionBlocker:   questionBlocker,
		HarnessResolver:   cli.GetHarnessResolver(),
	})
	if err != nil {
		return fmt.Errorf("failed to create run controller: %w", err)
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to locate project root: %w", err)
	}

	var runSessionID string
	if runLease != nil {
		runSessionID = runLease.sessionID
	}

	opts := runner.RunOptions{
		DryRun:          runDryRun,
		Verbose:         runVerbose,
		WorkingDir:      workingDir,
		RunID:           runID,
		SessionID:       runSessionID,
		ProjectRoot:     projectRoot,
		EntityType:      entityType,
		Observability:   obs,
		HarnessOverride: harnessOverride,
	}

	// D6 edit 3: the liveness recorder replaces the inline JSON-gated ticker
	// and progress printer. This single line resolves all three original
	// defects at once: the JSON gate disappears (REQ-F-001/002), the inline
	// ticker is replaced by the recorder's own fixed 10s ticker (REQ-F-006),
	// and normalizedKey is no longer read here (REQ-F-005 — the recorder
	// derives entity_key from RunProgress.EntityKey / update.EntityKey).
	opts.Progress = rec.Observe

	result, err := controller.Run(ctx, normalizedKey, opts)
	if err != nil {
		return fmt.Errorf("run failed for %s: %w", normalizedKey, err)
	}
	runResult = result // captured for deferred run.end emitter

	return outputRunResult(result)
}

func outputRunResult(result *runner.RunResult) error {
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

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

// resolveHarnessForClaim computes the harness identity to persist onto the
// lease this run acquires: the explicit --harness/--harness-version/
// --harness-model override wins per field (matching the flag tier of
// REQ-F-002's precedence), else the SHARK_HARNESS/_VERSION/_MODEL env vars —
// there is no pre-existing claim to consult yet, since this call is what
// creates the claim. Values are normalized (type trimmed+lowercased;
// version/model trimmed only) per REQ-F-001, mirroring `shark claim`'s own
// --harness normalization in claim.go's runClaim, before being handed to
// ClaimInput.
//
// Without this, a claim `shark run` itself creates never carries harness
// identity, so HarnessResolver.Resolve's claim tier is unreachable for any
// entity actually driven through `shark run` (as opposed to a claim seeded
// directly by a test's mocked ClaimReader) — the T-E34-F01-005 rework's
// defect. Seeding from override-else-env (not override-only) matters: it is
// what lets the claim tier decide a render on its own, distinct from the
// flag tier, when this run's own claim outlives the env value that seeded it
// (see TestRunClaimTierReachableThroughRealAcquisition).
func resolveHarnessForClaim(override services.HarnessIdentity) services.HarnessIdentity {
	pick := func(overrideValue, envKey string) string {
		if overrideValue != "" {
			return overrideValue
		}
		return os.Getenv(envKey)
	}
	return services.HarnessIdentity{
		Type:    strings.ToLower(strings.TrimSpace(pick(override.Type, "SHARK_HARNESS"))),
		Version: strings.TrimSpace(pick(override.Version, "SHARK_HARNESS_VERSION")),
		Model:   strings.TrimSpace(pick(override.Model, "SHARK_HARNESS_MODEL")),
	}
}

func acquireRunLease(ctx context.Context, entityType, entityKey, claimedBy string, dryRun bool, harnessOverride services.HarnessIdentity) (*activeRunLease, error) {
	if dryRun {
		return nil, nil
	}

	svc := getRunClaimService()
	harness := resolveHarnessForClaim(harnessOverride)
	claim, err := svc.Claim(ctx, services.ClaimInput{
		EntityType:     entityType,
		EntityKey:      entityKey,
		ClaimedBy:      claimedBy,
		Harness:        harness.Type,
		HarnessVersion: harness.Version,
		HarnessModel:   harness.Model,
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

// acquireRunLeaseForRunnableAction owns the action-ordering boundary shared by
// top-level and cascade runs. It intentionally reads the unpopulated action:
// deciding whether a lease is needed must not render Question responder
// placeholders (or derive a responder) for a non-dispatch checkpoint.
func acquireRunLeaseForRunnableAction(ctx context.Context, transitioner runner.EntityTransitioner, actionSvc config.ActionService, blocker questionBlockChecker, entityType, entityKey string, dryRun bool, harnessOverride services.HarnessIdentity) (*activeRunLease, *services.QuestionBlock, string, error) {
	nextInfo, err := transitioner.GetNextStatus(ctx, entityKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("get status for %s before run claim: %w", entityKey, err)
	}
	if nextInfo.IsTerminal {
		return nil, nil, nextInfo.CurrentStatus, nil
	}
	if blocker != nil {
		block, err := blocker.Check(ctx, models.EntityType(entityType), entityKey)
		if err != nil {
			return nil, nil, nextInfo.CurrentStatus, fmt.Errorf("check Question block for %s before run claim: %w", entityKey, err)
		}
		if block != nil {
			return nil, block, nextInfo.CurrentStatus, nil
		}
	}

	workflowAction, err := actionSvc.GetStatusAction(ctx, nextInfo.CurrentStatus)
	if err != nil {
		return nil, nil, nextInfo.CurrentStatus, fmt.Errorf("get action for %s before run claim: %w", entityKey, err)
	}
	if workflowAction == nil || runActionDoesNotDispatch(workflowAction.Action) {
		return nil, nil, nextInfo.CurrentStatus, nil
	}

	claimedBy, err := runClaimedBy(ctx, entityType, entityKey)
	if err != nil {
		return nil, nil, nextInfo.CurrentStatus, err
	}
	lease, err := acquireRunLease(ctx, entityType, entityKey, claimedBy, dryRun, harnessOverride)
	return lease, nil, nextInfo.CurrentStatus, err
}

// preflightCascadeQuestionBlock walks only the configured workflow hierarchy
// before the parent run acquires a lease. A blocked child is parked, matching
// keyed-next semantics: an eligible sibling keeps the cascade live, while an
// all-parked subtree returns its first compact handoff without a parent lease.
// It is intentionally separate from QuestionBlocker: hierarchy traversal
// belongs to the runner, while the gate remains a direct candidate lookup.
func preflightCascadeQuestionBlock(
	ctx context.Context,
	rootTransitioner runner.EntityTransitioner,
	actionSvcRoot config.ActionService,
	childrenSvc runner.CascadeChildrenService,
	blocker questionBlockChecker,
	buildChildTransitioner func(context.Context, string) (runner.EntityTransitioner, error),
	entityType, entityKey string,
) (*services.QuestionBlock, string, error) {
	if blocker == nil {
		return nil, "", nil
	}

	var rootStatus string
	type cascadeAvailability struct {
		live  bool
		block *services.QuestionBlock
	}
	var walk func(runner.EntityTransitioner, string, string) (cascadeAvailability, error)
	walk = func(transitioner runner.EntityTransitioner, candidateType, candidateKey string) (cascadeAvailability, error) {
		info, err := transitioner.GetNextStatus(ctx, candidateKey)
		if err != nil {
			return cascadeAvailability{}, fmt.Errorf("get status for %s: %w", candidateKey, err)
		}
		if rootStatus == "" {
			rootStatus = info.CurrentStatus
		}
		if info.IsTerminal {
			return cascadeAvailability{}, nil
		}
		block, err := blocker.Check(ctx, models.EntityType(candidateType), candidateKey)
		if err != nil {
			return cascadeAvailability{}, fmt.Errorf("check Question block for %s: %w", candidateKey, err)
		}
		if block != nil {
			return cascadeAvailability{block: block}, nil
		}

		actionSvc := narrowActionServiceForEntity(actionSvcRoot, candidateType)
		workflowAction, err := actionSvc.GetStatusAction(ctx, info.CurrentStatus)
		if err != nil {
			return cascadeAvailability{}, fmt.Errorf("get action for %s: %w", candidateKey, err)
		}
		if workflowAction == nil || runActionDoesNotDispatch(workflowAction.Action) {
			return cascadeAvailability{}, nil
		}
		if workflowAction.Action != config.ActionCascade {
			return cascadeAvailability{live: true}, nil
		}
		if childrenSvc == nil {
			return cascadeAvailability{}, fmt.Errorf("cascade action for %s has no children service", candidateKey)
		}
		children, err := childrenSvc.DescribeDispatchableChildren(ctx, candidateType, candidateKey)
		if err != nil {
			return cascadeAvailability{}, fmt.Errorf("list cascade children for %s: %w", candidateKey, err)
		}
		var firstBlock *services.QuestionBlock
		for _, child := range children.Children {
			childTransitioner, err := buildChildTransitioner(ctx, string(child.EntityType))
			if err != nil {
				return cascadeAvailability{}, fmt.Errorf("build transitioner for cascade child %s %s: %w", child.EntityType, child.Key, err)
			}
			childState, err := walk(childTransitioner, string(child.EntityType), child.Key)
			if err != nil {
				return cascadeAvailability{}, err
			}
			if childState.live {
				return cascadeAvailability{live: true}, nil
			}
			if firstBlock == nil && childState.block != nil {
				firstBlock = childState.block
			}
		}
		return cascadeAvailability{block: firstBlock}, nil
	}

	state, err := walk(rootTransitioner, entityType, entityKey)
	return state.block, rootStatus, err
}

func runActionDoesNotDispatch(action string) bool {
	switch action {
	case config.ActionPause, config.ActionWaitForTriage, config.ActionCheckOrResume, config.ActionArchive:
		return true
	default:
		return false
	}
}

// runClaimedBy binds a Question lease to the derived current responder before
// dispatch. Other entity types retain ClaimService's normal actor identity.
// This is parent-loop work: a worker never selects an identity or writes a
// response directly.
func runClaimedBy(ctx context.Context, entityType, entityKey string) (string, error) {
	if entityType != "question" {
		return "", nil
	}
	question, err := getQuestionService().GetQuestion(ctx, entityKey)
	if err != nil {
		return "", fmt.Errorf("load Question %s before run claim: %w", entityKey, err)
	}
	state, err := models.DecodeQuestionState(question.ContextData)
	if err != nil {
		return "", fmt.Errorf("decode Question %s before run claim: %w", entityKey, err)
	}
	if state == nil || state.CurrentResponder() == "" {
		return "", fmt.Errorf("Question %s has no current responder to claim", entityKey)
	}
	return state.CurrentResponder(), nil
}

func buildQuestionResponsePersister(entityType string) runner.QuestionResponsePersister {
	if entityType != "question" {
		return nil
	}
	return runner.QuestionResponsePersisterFunc(func(ctx context.Context, handoff runner.QuestionResponseHandoff) error {
		_, err := getQuestionService().RecordResponse(ctx, services.RecordQuestionResponseInput{
			Key:             handoff.Key,
			SessionID:       handoff.SessionID,
			Responder:       handoff.Responder,
			Summary:         handoff.Summary,
			EvidencePointer: handoff.EvidencePointer,
		})
		return err
	})
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
// TaskService, FeatureService, EpicService, BugService, ChangeCardService,
// TechDebtService, QuestionService, and SprintService directly satisfy
// runner.EntityTransitioner via their TransitionStatus and GetNextStatus
// methods (SprintService only after EnableWorkflowDispatch is called).
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
	case "question":
		return getQuestionService(), nil
	case "sprint":
		return cli.GetSprintService(), nil
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
	case "question":
		return &questionPlaceholderAdapter{svc: getQuestionService()}
	case "sprint":
		return &sprintPlaceholderAdapter{svc: cli.GetSprintService()}
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

type questionPlaceholderAdapter struct{ svc questionServicer }

func (a *questionPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	question, err := a.svc.GetQuestion(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get question %s for placeholders: %w", key, err)
	}
	state, err := models.DecodeQuestionState(question.ContextData)
	if err != nil {
		return nil, fmt.Errorf("decode Question state for placeholders: %w", err)
	}
	if state == nil || state.CurrentResponder() == "" {
		return nil, fmt.Errorf("Question %s has no current responder", key)
	}
	vars := config.EntityPlaceholders(question)
	vars["summary"] = question.Summary
	vars["requester"] = question.Requester
	vars["current_responder"] = state.CurrentResponder()
	return vars, nil
}

func (a *techDebtPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	td, err := a.svc.GetTechDebt(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt %s for placeholders: %w", key, err)
	}
	return config.TechDebtPlaceholders(td), nil
}

type sprintPlaceholderAdapter struct {
	svc *services.SprintService
}

func (a *sprintPlaceholderAdapter) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	sprint, err := a.svc.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint %s for placeholders: %w", key, err)
	}
	return config.EntityPlaceholders(sprint), nil
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
	_ runner.EntityTransitioner = (*services.QuestionService)(nil)
	_ runner.EntityTransitioner = (*services.SprintService)(nil)

	// Placeholder adapters remain entity-specific.
	_ runner.PlaceholderGenerator = (*taskPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*featurePlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*epicPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*bugPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*changeCardPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*techDebtPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*questionPlaceholderAdapter)(nil)
	_ runner.PlaceholderGenerator = (*sprintPlaceholderAdapter)(nil)
)
