// Package runner provides the AgentDispatcher interface and related types for
// invoking external AI agents (Claude, Codex, etc.) as part of the E22 run loop.
// This file defines the RunController, its interfaces, and supporting data types.
package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// GateIngestDeps carries T-E34-F05-004's shared GateResult ingestion
// coordinator and outcome-role resolution for gate_result_v1 steps. Optional
// on RunControllerDeps: nil is safe as long as no configured step's
// result_contract resolves to gate_result_v1 — dispatch reaches this branch
// only for a step whose `result_contract: gate_result_v1` YAML resolves
// through resultContractFor (T-E34-F05-005).
type GateIngestDeps struct {
	// Coordinator persists a validated GateResult and applies the guarded
	// main-entity transition (T-E34-F05-003).
	Coordinator *gatepersist.Coordinator
	// OutcomeRoles is a fallback flat, run-wide outcome-role map used only
	// when a dispatched step's own resolved NextStatusInfo.OutcomeRoles is
	// empty (e.g. tests wiring a coordinator without a real workflow config
	// source). Real dispatch resolves outcome_roles per step from the
	// workflow's `outcome_roles` YAML map (T-E34-F05-005); see
	// ingestGateResultForDispatch.
	OutcomeRoles map[string]gateresult.OutcomeRole
}

// RunOptions controls run loop behavior.
type RunOptions struct {
	// DryRun, when true, prints actions but does not dispatch agents or advance status.
	DryRun bool

	// Verbose, when true, prints detailed stage progress to stderr.
	Verbose bool

	// Progress, when set, is called for high-signal run loop checkpoints.
	// Callers can use this for user-facing progress output without coupling
	// to internal logger internals.
	Progress func(RunProgress)

	// WorkingDir is an optional working directory override for agent processes.
	WorkingDir string

	// RunID is a correlation identifier generated once at the top of runRun.
	// It is threaded through all stage events so a single run can be grepped
	// from shark.log.
	RunID string

	// SessionID is the lease session acquired for this run. Guarded status
	// advances use it to bind a worker result to this particular run.
	SessionID string

	// Observability carries the observability configuration for this run. The
	// controller uses it to decide whether to emit per-stage slog events and
	// how aggressively to truncate large payloads (stderr/stdout) in error
	// events. When Enabled is false, no run.stage.* events are emitted.
	Observability config.ObservabilityConfig

	// ProjectRoot is the absolute path to the project root. It is used as the
	// base directory for transcript file capture ({project_root}/.shark/runs/...)
	// when observability.capture_agent_transcripts == true. When empty, the
	// controller skips transcript writing even if transcripts are enabled.
	ProjectRoot string

	// EntityType identifies the current entity type being run.
	// It is required for cascade child service lookups when the current action
	// is "cascade".
	EntityType string

	// HarnessOverride carries the explicit --harness/--harness-version/
	// --harness-model flag values read once by the CLI command (spec.md
	// §3.3 AC-T2). It is the top precedence tier for every entity the
	// controller visits during this run, including cascade children (copied
	// via childOpts := opts).
	HarnessOverride services.HarnessIdentity
}

// RunProgress carries coarse-grained run-loop state for progress callbacks.
// It intentionally contains only stable, human-readable fields and is emitted
// at deterministic checkpoints in the loop.
type RunProgress struct {
	// Iteration is the 1-based run loop counter.
	Iteration int
	// EntityKey is the normalized key being processed.
	EntityKey string
	// Status is the current status for this iteration before the action runs.
	Status string
	// Phase indicates the run checkpoint being reported.
	Phase string
	// Action is the resolved workflow action for this stage (if available).
	Action string
	// AgentType and Provider are action metadata where applicable.
	AgentType string
	Provider  string
}

// RunResult captures the outcome of a run loop execution.
type RunResult struct {
	// EntityKey is the entity that was run.
	EntityKey string `json:"entity_key"`

	// FinalStatus is the entity's status when the loop stopped.
	FinalStatus string `json:"final_status"`

	// StagesCompleted is the number of stages successfully completed.
	StagesCompleted int `json:"stages_completed"`

	// Stages contains per-stage log entries.
	Stages []StageLog `json:"stages"`

	// Outcome is one of: "completed", "paused", "failed", "already_terminal", "no_action".
	Outcome string `json:"outcome"`

	// TotalDuration is the wall-clock time for the entire run.
	TotalDuration time.Duration `json:"total_duration_ns"`

	// Error is the error message if outcome is "failed" (empty otherwise).
	Error string `json:"error,omitempty"`

	// QuestionBlock is the optional compact I-03 handoff when a directly
	// linked open blocking Question pauses this run before dispatch work.
	QuestionBlock *services.QuestionBlock `json:"question_block,omitempty"`
}

// StageLog captures per-stage execution details.
type StageLog struct {
	// EntityKey is the entity this stage was executed for. Cascade children
	// populate their own key so stages remain attributable after the parent
	// flattens child Stages into its own result.
	EntityKey string `json:"entity_key"`

	// Status is the workflow status this stage executed.
	Status string `json:"status"`

	// Action is the action type (e.g., "spawn_agent", "advance_status").
	Action string `json:"action"`

	// AgentType is the agent type from the orchestrator action (e.g., "developer").
	AgentType string `json:"agent_type,omitempty"`

	// Provider is the provider used (e.g., "anthropic").
	Provider string `json:"provider,omitempty"`

	// Duration is the wall-clock time for this stage.
	Duration time.Duration `json:"duration_ns"`

	// ExitCode is the agent exit code (0 for non-agent actions).
	ExitCode int `json:"exit_code"`

	// OutputSummary is the captured standard output from the agent process for
	// this stage. It is only populated for successful spawn_agent stages (exit
	// code 0). Use this field for non-verbose display summaries.
	OutputSummary string `json:"output_summary,omitempty"`

	// GateStatus is the T-E34-F05-004 REQ-F-005 operator status projection
	// (worker phase, nested operation, elapsed time, retirement state, result
	// location) for a gate_result_v1 stage, populated from the same
	// gaterun.StatusProjection --resume-run reports. Nil for a legacy stage.
	GateStatus *gaterun.StatusProjection `json:"gate_status,omitempty"`
}

// EntityTransitioner abstracts per-entity-type status transition dispatch.
// It is defined at point of use in this package, following the project convention
// of defining interfaces on the consumer side (.claude/rules/go/patterns.md).
//
// Implementations are provided in internal/cli/commands/run.go as per-entity-type
// adapters dispatching to the correct per-type service.
type EntityTransitioner interface {
	// TransitionStatus transitions the entity to the given target status.
	// opts carries optional force/reason/agent metadata.
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)

	// GetNextStatus returns the current status and available next transitions
	// for the entity, along with a terminal flag.
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// CascadeChildrenService enumerates dispatchable children for cascade actions.
// It is used only by ActionCascade handling.
type CascadeChildrenService interface {
	DescribeDispatchableChildren(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error)
}

// CascadeChildRunner runs a nested child entity through its own controller loop.
type CascadeChildRunner func(ctx context.Context, entityType, key string, opts RunOptions) (*RunResult, error)

// QuestionBlockChecker is the narrow, read-only I-03 gate used by runner
// entry points. It deliberately exposes no Question mutation capability.
type QuestionBlockChecker interface {
	Check(ctx context.Context, candidateType models.EntityType, candidateKey string) (*services.QuestionBlock, error)
}

// QuestionResponseHandoff is the bounded worker result that the parent loop
// persists for a Question responder. The worker supplies only the response
// body; the parent supplies the entity key, lease session, and responder from
// the already-rendered dispatch context.
type QuestionResponseHandoff struct {
	Key             string `json:"-"`
	SessionID       string `json:"-"`
	Responder       string `json:"-"`
	Summary         string `json:"summary"`
	EvidencePointer string `json:"evidence_pointer"`
}

// QuestionResponsePersister is the parent-owned persistence seam for a
// successful Question responder result. It deliberately exposes neither
// claim/release nor generic status mutation, so a child worker cannot own
// Shark lifecycle state.
type QuestionResponsePersister interface {
	PersistQuestionResponse(ctx context.Context, handoff QuestionResponseHandoff) error
}

// QuestionResponsePersisterFunc adapts a function for focused runner tests
// and the CLI wiring adapter.
type QuestionResponsePersisterFunc func(ctx context.Context, handoff QuestionResponseHandoff) error

func (f QuestionResponsePersisterFunc) PersistQuestionResponse(ctx context.Context, handoff QuestionResponseHandoff) error {
	return f(ctx, handoff)
}

// PlaceholderGenerator abstracts template variable generation for different entity types.
// An adapter implementation in run.go dispatches to the correct config.*Placeholders()
// function based on entity type.
type PlaceholderGenerator interface {
	// GeneratePlaceholders returns a map of template variable names to values
	// for the entity identified by key.
	GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error)
}

// PromptAssemblyInput is the fully-rendered workflow action plus metadata
// needed to produce the final host-facing prompt.
type PromptAssemblyInput struct {
	// Instruction is the rendered workflow prompt from the action service.
	Instruction string

	// AgentType is Shark metadata naming the specialist persona to inline.
	AgentType string

	// Vars are the entity placeholders used by both workflow prompt rendering
	// and agent persona rendering.
	Vars map[string]string
}

// PromptAssembler produces the final prompt sent to an agent dispatcher.
type PromptAssembler interface {
	AssemblePrompt(ctx context.Context, input PromptAssemblyInput) (string, error)
}

// PromptAssemblerFunc adapts a function to PromptAssembler.
type PromptAssemblerFunc func(ctx context.Context, input PromptAssemblyInput) (string, error)

func (f PromptAssemblerFunc) AssemblePrompt(ctx context.Context, input PromptAssemblyInput) (string, error) {
	return f(ctx, input)
}

// RunControllerDeps bundles all dependencies for RunController construction.
// All fields are interfaces to enable mock injection for tests.
type RunControllerDeps struct {
	// Transitioner handles status reads and transitions. Required.
	Transitioner EntityTransitioner

	// Placeholders generates template variables for instruction rendering. Optional.
	// When nil, an empty vars map is passed to GetStatusActionPopulated.
	Placeholders PlaceholderGenerator

	// ActionSvc reads orchestrator action configuration. Required.
	ActionSvc config.ActionService

	// WorkflowSvc provides terminal status detection. Required.
	WorkflowSvc *workflow.Service

	// Dispatchers maps provider name to AgentDispatcher implementation.
	// The "" (empty string) key is the default dispatcher (used when action.Provider is "").
	// Required: must have at least one entry.
	Dispatchers map[string]AgentDispatcher

	// PromptAssembler converts a populated action instruction into the final
	// host-facing prompt. Optional; nil preserves the legacy instruction passthrough.
	PromptAssembler PromptAssembler

	// ChildrenSvc lists dispatchable cascade children. Required when ActionCascade
	// is reachable from this controller.
	ChildrenSvc CascadeChildrenService

	// RunChild dispatches a child entity through a nested RunController. Optional
	// by design for non-cascade entry points.
	RunChild CascadeChildRunner

	// QuestionResponses persists the typed response returned by a Question
	// worker. It is required only for Question spawn_agent stages.
	QuestionResponses QuestionResponsePersister

	// QuestionBlocker qualifies directly linked open blocking Questions before
	// placeholder, action, or worker work. Optional for non-CLI embeddings.
	QuestionBlocker QuestionBlockChecker

	// HarnessResolver resolves harness identity per spec.md REQ-F-002's
	// flag > claim > env > zero precedence (T-E34-F01-003/005), giving
	// `shark run` parity with `shark next` (REQ-F-006). Optional: when nil,
	// the zero identity's three empty keys are injected into vars — never
	// an absent key (D-F01-07).
	HarnessResolver *services.HarnessResolver

	// GateIngest wires the T-E34-F05-004 gate_result_v1 ingestion path.
	// Optional — see GateIngestDeps's doc comment.
	GateIngest *GateIngestDeps
}

// RunController implements the core orchestration loop: read entity state,
// read orchestrator action, dispatch to agents, gate on exit code, loop until
// terminal/pause/failure.
//
// All dependencies are injected via RunControllerDeps; no global state is used,
// enabling full mock-based testing without a real database or agent processes.
type RunController struct {
	transitioner      EntityTransitioner
	placeholders      PlaceholderGenerator
	actionSvc         config.ActionService
	workflowSvc       *workflow.Service
	dispatchers       map[string]AgentDispatcher
	assembler         PromptAssembler
	childrenSvc       CascadeChildrenService
	runChild          CascadeChildRunner
	questionResponses QuestionResponsePersister
	questionBlocker   QuestionBlockChecker
	harnessResolver   *services.HarnessResolver
	gateIngest        *GateIngestDeps
}

// NewRunController constructs a RunController with the provided dependencies.
// Panics if any required dependency is nil (Transitioner, ActionSvc, WorkflowSvc).
// Returns a non-nil error if Dispatchers is empty.
func NewRunController(deps RunControllerDeps) (*RunController, error) {
	if deps.Transitioner == nil {
		panic("RunController: Transitioner dependency is required (got nil)")
	}
	if deps.ActionSvc == nil {
		panic("RunController: ActionSvc dependency is required (got nil)")
	}
	if deps.WorkflowSvc == nil {
		panic("RunController: WorkflowSvc dependency is required (got nil)")
	}
	if len(deps.Dispatchers) == 0 {
		return nil, fmt.Errorf("RunController: Dispatchers map must have at least one entry")
	}
	assembler := deps.PromptAssembler
	if assembler == nil {
		assembler = PromptAssemblerFunc(func(_ context.Context, input PromptAssemblyInput) (string, error) {
			return input.Instruction, nil
		})
	}

	return &RunController{
		transitioner:      deps.Transitioner,
		placeholders:      deps.Placeholders,
		actionSvc:         deps.ActionSvc,
		workflowSvc:       deps.WorkflowSvc,
		dispatchers:       deps.Dispatchers,
		assembler:         assembler,
		childrenSvc:       deps.ChildrenSvc,
		runChild:          deps.RunChild,
		questionResponses: deps.QuestionResponses,
		questionBlocker:   deps.QuestionBlocker,
		harnessResolver:   deps.HarnessResolver,
		gateIngest:        deps.GateIngest,
	}, nil
}

// stageOutcome is the return value from per-action handler methods. It tells the
// main loop whether to continue, stop, or report a failure.
type stageOutcome struct {
	// nextStatus is the status to use for the next iteration (when done == false).
	nextStatus string
	// nextInfo is the simulated status metadata to use for the next iteration.
	// It is populated by dry-run paths that must not re-read live entity state.
	nextInfo *services.NextStatusInfo
	// done signals the loop should return result immediately.
	done bool
}

// Run executes the orchestration loop for the given entity key.
//
// The loop:
//  1. Reads entity's current status via GetNextStatus.
//  2. Returns immediately if entity is already in a terminal status.
//  3. Gets orchestrator action for the current status via ActionSvc.
//  4. Handles the action type: spawn_agent/check_or_resume, advance_status, pause/wait_for_triage, archive.
//  5. For spawn_agent/check_or_resume: dispatches agent, gates advancement on exit code 0.
//  6. After successful advance, checks for terminal and loops.
//  7. Stops on non-zero exit, missing dispatcher, or context cancellation.
func (c *RunController) Run(ctx context.Context, key string, opts RunOptions) (*RunResult, error) {
	startTime := time.Now()
	result := &RunResult{
		EntityKey: key,
		Stages:    make([]StageLog, 0),
	}

	// Step 1: Get initial entity status.
	nextInfo, err := c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get status for %s: %w", key, err)
	}

	currentStatus := nextInfo.CurrentStatus

	// Step 2: Already terminal? Return immediately.
	if nextInfo.IsTerminal {
		result.FinalStatus = currentStatus
		// Question uses NextStatusInfo.IsTerminal as a non-dispatching signal
		// for responder-less checkpoints as well as durable terminal states.
		// Those checkpoints must preserve keyed-next's pause semantics in every
		// runner entry point; only resolved/withdrawn/superseded are completed
		// Question terminals.
		if isQuestionResponderPauseCheckpoint(opts.EntityType, currentStatus) {
			result.Outcome = "paused"
		} else {
			result.Outcome = "already_terminal"
		}
		result.TotalDuration = time.Since(startTime)
		return result, nil
	}

	// Gate a direct runner invocation after identity/status are known but before
	// any placeholder, action, worker, or transition work. CLI preflight uses
	// this same checker before lease acquisition; retaining it here protects
	// direct controller consumers and cascade child controllers as well.
	if c.questionBlocker != nil && opts.EntityType != "" {
		block, err := c.questionBlocker.Check(ctx, models.EntityType(opts.EntityType), key)
		if err != nil {
			return nil, fmt.Errorf("check Question block for %s: %w", key, err)
		}
		if block != nil {
			result.FinalStatus = currentStatus
			result.Outcome = "paused"
			result.QuestionBlock = block
			result.TotalDuration = time.Since(startTime)
			return result, nil
		}
	}

	// Main loop.
	iteration := 0
	// transcriptDisabled is a RUN-SCOPED latch: once any transcript write fails,
	// we emit run.transcript.warning exactly once (see handleSpawnAgent) and set
	// this flag to true to suppress all further write attempts for the remainder
	// of the run — at most one warning per run, without caching prior errors.
	transcriptDisabled := false
	for {
		// Check context cancellation at the top of each iteration.
		select {
		case <-ctx.Done():
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "context",
				Error:     ctx.Err().Error(),
				RunID:     opts.RunID,
			})
			return result, nil
		default:
		}

		// Emit run.stage.start for this iteration.
		iteration++
		if opts.Progress != nil {
			opts.Progress(RunProgress{
				Iteration: iteration,
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "iteration",
			})
		}
		emitStageStart(ctx, opts.Observability, stageStartParams{
			EntityKey: key,
			Status:    currentStatus,
			Iteration: iteration,
			RunID:     opts.RunID,
		})

		stageStart := time.Now()

		// Step 3: Get template variables for instruction rendering.
		var vars map[string]string
		if c.placeholders != nil {
			vars, err = c.placeholders.GeneratePlaceholders(ctx, key)
			if err != nil {
				recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
					EntityKey: key,
					Status:    currentStatus,
					Phase:     "placeholders",
					Error:     fmt.Sprintf("failed to generate placeholders for %s: %v", key, err),
					RunID:     opts.RunID,
				})
				return result, nil
			}
		}
		if vars == nil {
			vars = map[string]string{}
		}
		templates.AugmentPlaceholderAliases(vars)

		// Resolve harness identity (spec.md REQ-F-006/AC-08) and merge it
		// into vars before the action renders, mirroring next.go's
		// resolveEntity so `{{if isClaude .harness}}` branches see the same
		// values under `shark run` as under `shark next` for identical
		// inputs. HarnessIdentity.Vars() never omits a key, even when
		// unresolved (D-F01-07).
		if c.harnessResolver != nil {
			identity, hErr := c.harnessResolver.Resolve(ctx, opts.EntityType, key, opts.HarnessOverride)
			if hErr != nil {
				// HarnessResolver.Resolve is documented to always return a
				// nil error (claim-read failures degrade internally per
				// D-F01-05); this branch exists only to fail loudly if that
				// contract is ever violated, rather than silently dropping
				// the error.
				recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
					EntityKey: key,
					Status:    currentStatus,
					Phase:     "harness_resolution",
					Error:     fmt.Sprintf("failed to resolve harness identity for %s: %v", key, hErr),
					RunID:     opts.RunID,
				})
				return result, nil
			}
			for k, v := range identity.Vars() {
				vars[k] = v
			}
		} else {
			zero := services.HarnessIdentity{}
			for k, v := range zero.Vars() {
				vars[k] = v
			}
		}

		// Step 4: Get populated orchestrator action for current status.
		action, err := c.actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)
		if err != nil {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "action_lookup",
				Error:     fmt.Sprintf("failed to get action for status %s: %v", currentStatus, err),
				RunID:     opts.RunID,
			})
			return result, nil
		}
		// Step 5: No action configured for this status — stop with no_action.
		if action == nil {
			result.FinalStatus = currentStatus
			result.Outcome = "no_action"
			result.TotalDuration = time.Since(startTime)
			return result, nil
		}
		if opts.Progress != nil {
			opts.Progress(RunProgress{
				Iteration: iteration,
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "action",
				Action:    action.Action,
				AgentType: action.AgentType,
				Provider:  action.Provider,
			})
		}

		// Step 6: Route by action type.
		var outcome stageOutcome
		switch action.Action {
		case config.ActionPause, config.ActionWaitForTriage:
			result.FinalStatus = currentStatus
			result.Outcome = "paused"
			result.TotalDuration = time.Since(startTime)
			return result, nil

		case config.ActionArchive:
			result.Stages = append(result.Stages, StageLog{
				EntityKey: key,
				Status:    currentStatus,
				Action:    action.Action,
				Duration:  time.Since(stageStart),
			})
			result.StagesCompleted++
			result.FinalStatus = currentStatus
			result.Outcome = "completed"
			result.TotalDuration = time.Since(startTime)
			return result, nil

		case config.ActionAdvanceStatus:
			outcome = c.handleAdvanceStatus(ctx, key, currentStatus, nextInfo, action, opts, result, stageStart, startTime)

		case config.ActionSpawnAgent, config.ActionCheckOrResume:
			outcome = c.handleSpawnAgent(ctx, key, currentStatus, nextInfo, action, vars, opts, result, stageStart, startTime, iteration, &transcriptDisabled)

		case config.ActionCascade:
			outcome = c.handleCascade(ctx, key, currentStatus, nextInfo, action, opts, result, stageStart, startTime)

		default:
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "unknown_action",
				Error:     fmt.Sprintf("unknown action type %q for status %s", action.Action, currentStatus),
				RunID:     opts.RunID,
			})
			return result, nil
		}

		if outcome.done {
			return result, nil
		}
		currentStatus = outcome.nextStatus
		if outcome.nextInfo != nil {
			nextInfo = outcome.nextInfo
		} else if !opts.DryRun {
			// code-review round-7 Finding 1 sweep: handleAdvanceStatus and
			// handleSpawnAgent's live (non-dry-run) paths return a bare
			// nextStatus with no nextInfo, so without this refresh the `nextInfo`
			// this loop hands to the NEXT iteration's handleSpawnAgent would
			// still be whatever NextStatusInfo was current BEFORE this stage
			// transitioned — a stale snapshot of an earlier status. handleSpawnAgent
			// pins that parameter as `stepInfo` and resolves resultContractFor,
			// outcome_roles, AND the gate_result_v1 target-status Outcomes map
			// from it, so a second consecutive spawn_agent stage in one Run()
			// invocation (e.g. code_review -> qa) would silently resolve gate
			// outcomes against the FIRST stage's configuration instead of its
			// own — for a self-transition outcome map (a stage whose "pass"
			// outcome happens to equal its own status) this manifests as an
			// infinite dispatch loop that never reaches a terminal status,
			// discovered by TestRunController_Run_MultiStageGateResultV1DispatchDoesNotReuseRunID.
			// A dry run intentionally skips this: it must not re-read live
			// entity state (see dryRunPostActionStatus's own doc comment) and
			// already carries its own simulated nextInfo via
			// simulatedDryRunNextStatus in every non-terminal dryRunNextOutcome
			// return.
			refreshed, err := c.transitioner.GetNextStatus(ctx, key)
			if err != nil {
				recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
					EntityKey: key,
					Status:    currentStatus,
					Phase:     "next_stage_status",
					Error:     fmt.Sprintf("failed to refresh status for %s before the next stage: %v", key, err),
					RunID:     opts.RunID,
				})
				return result, nil
			}
			nextInfo = refreshed
		}
	}

}

// isQuestionResponderPauseCheckpoint identifies Question states where the
// Question service deliberately suppresses responder dispatch with
// NextStatusInfo.IsTerminal. The draft case preserves F01 compatibility;
// open/answering cover migrated or otherwise unconfigured Questions; and
// ready_for_resolution is the resolution-owner checkpoint. Durable Question
// terminal states intentionally do not appear here and remain already_terminal.
func isQuestionResponderPauseCheckpoint(entityType, status string) bool {
	if entityType != string(models.EntityTypeQuestion) {
		return false
	}
	switch models.QuestionStatus(status) {
	case models.QuestionStatusDraft, models.QuestionStatusOpen, models.QuestionStatusAnswering, models.QuestionStatusReadyForResolution:
		return true
	default:
		return false
	}
}

func (c *RunController) handleCascade(
	ctx context.Context,
	key, currentStatus string,
	nextInfo *services.NextStatusInfo,
	_ *config.PopulatedAction,
	opts RunOptions,
	result *RunResult,
	stageStart,
	startTime time.Time,
) stageOutcome {
	if c.childrenSvc == nil || c.runChild == nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "cascade",
			Error:     "cascade action configured but no child runner/dependency was injected",
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}
	childrenState, err := c.childrenSvc.DescribeDispatchableChildren(ctx, opts.EntityType, key)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "cascade_children_lookup",
			Error:     fmt.Sprintf("failed to list dispatchable children for %s %s: %v", opts.EntityType, key, err),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	progressed := false
	// Keep the first compact handoff only if every cascade child is parked.
	// A directly blocked child is unavailable work, not a reason to prevent a
	// later independent sibling from running.
	var allParkedQuestionBlock *services.QuestionBlock
	nonProgressingChildren := 0
	questionBlockedChildren := 0
	for _, child := range childrenState.Children {
		childOpts := opts
		childOpts.EntityType = string(child.EntityType)
		childResult, err := c.runChild(ctx, string(child.EntityType), child.Key, childOpts)
		if err != nil {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "cascade_child_run",
				Error:     fmt.Sprintf("failed to run cascade child %s %s: %v", child.EntityType, child.Key, err),
				RunID:     opts.RunID,
			})
			return stageOutcome{done: true}
		}
		if childResult != nil {
			result.Stages = append(result.Stages, childResult.Stages...)
			result.StagesCompleted += childResult.StagesCompleted
		}
		// A directly blocked cascade child is parked. Preserve its compact I-03
		// handoff in case every child is parked, but continue to later siblings
		// so cascade run has the same fall-through semantics as keyed next.
		if childResult != nil && childResult.QuestionBlock != nil {
			nonProgressingChildren++
			questionBlockedChildren++
			if allParkedQuestionBlock == nil {
				allParkedQuestionBlock = childResult.QuestionBlock
			}
			continue
		}
		if childResult != nil && childResult.Outcome == "failed" {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "cascade_child",
				Error:     fmt.Sprintf("cascade child %s %s failed: %s", child.EntityType, child.Key, childResult.Error),
				RunID:     opts.RunID,
			})
			return stageOutcome{done: true}
		}

		if childResult != nil {
			result.FinalStatus = childResult.FinalStatus
		}
		if childResult != nil && childResult.Outcome == "completed" {
			progressed = true
		} else {
			nonProgressingChildren++
		}
	}

	if progressed {
		// Child progress may have moved the parent status, so refresh once and
		// let the top-level loop resolve the next status transition. When the
		// parent itself remains at the cascade status, stopping here preserves
		// the sibling's successful result: re-entering the same cascade would
		// see only the earlier parked child and incorrectly resurrect its
		// Question handoff (and loops forever in dry-run, where no child write
		// can change the hierarchy).
		refreshed, err := c.transitioner.GetNextStatus(ctx, key)
		if err != nil {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "cascade_parent_status",
				Error:     fmt.Sprintf("failed to refresh parent status after cascade children: %v", err),
				RunID:     opts.RunID,
			})
			return stageOutcome{done: true}
		}
		if refreshed.CurrentStatus == currentStatus {
			refreshedChildren, err := c.childrenSvc.DescribeDispatchableChildren(ctx, opts.EntityType, key)
			if err != nil {
				recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
					EntityKey: key,
					Status:    currentStatus,
					Phase:     "cascade_children_refresh",
					Error:     fmt.Sprintf("failed to refresh cascade children for %s %s: %v", opts.EntityType, key, err),
					RunID:     opts.RunID,
				})
				return stageOutcome{done: true}
			}
			if refreshedChildren.TotalChildren > 0 && refreshedChildren.NonTerminalChildren == 0 {
				return c.autoAdvanceCascadeParent(ctx, key, currentStatus, nextInfo, opts, result, stageStart, startTime)
			}
			return pauseCascade(result, currentStatus, nil, startTime)
		}
		return stageOutcome{nextStatus: refreshed.CurrentStatus, nextInfo: refreshed}
	}

	// If no child completed, either pause (partially available progress) or
	// auto-advance (every child is terminal) depending on child summary.
	if childrenState.TotalChildren > 0 && childrenState.NonTerminalChildren == 0 {
		return c.autoAdvanceCascadeParent(ctx, key, currentStatus, nextInfo, opts, result, stageStart, startTime)
	}
	if questionBlockedChildren != nonProgressingChildren {
		allParkedQuestionBlock = nil
	}
	return pauseCascade(result, currentStatus, allParkedQuestionBlock, startTime)
}

// pauseCascade records a cascade parent's stall (partial or fully blocked
// progress, no auto-advance) as a completed-but-paused run stage. block is
// nil when the parent simply has non-terminal children still pending, or the
// compact handoff when every candidate is Question-blocked.
func pauseCascade(result *RunResult, currentStatus string, block *services.QuestionBlock, startTime time.Time) stageOutcome {
	result.FinalStatus = currentStatus
	result.Outcome = "paused"
	result.QuestionBlock = block
	result.TotalDuration = time.Since(startTime)
	return stageOutcome{done: true}
}

func (c *RunController) autoAdvanceCascadeParent(
	ctx context.Context,
	key, currentStatus string,
	nextInfo *services.NextStatusInfo,
	opts RunOptions,
	result *RunResult,
	stageStart,
	startTime time.Time,
) stageOutcome {
	if nextInfo == nil || len(nextInfo.AvailableTransitions) == 0 {
		result.FinalStatus = currentStatus
		result.Outcome = "paused"
		result.Error = fmt.Sprintf(
			"all child work is terminal, but %s has no forward transition to auto-advance to",
			key,
		)
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets
	if opts.DryRun {
		result.Stages = append(result.Stages, StageLog{
			EntityKey: key,
			Status:    currentStatus,
			Action:    config.ActionAdvanceStatus,
			Duration:  0,
		})
		postInfo, ok := c.dryRunPostActionStatus(ctx, key, currentStatus, nextInfo, "cascade", opts, result, startTime)
		if !ok {
			return stageOutcome{done: true}
		}
		result.FinalStatus = currentStatus
		return c.dryRunNextOutcome(currentStatus, postInfo, result, startTime)
	}

	transitionResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, guardedTransitionOptions(opts, currentStatus, targetStatus, nextInfo))
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "cascade_auto_advance",
			Error:     fmt.Sprintf("cascade completion auto-advance from %s to %s failed: %v", currentStatus, targetStatus, err),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	emitStageTransition(ctx, opts.Observability, stageTransitionParams{
		EntityKey:  key,
		FromStatus: currentStatus,
		ToStatus:   transitionResult.ToStatus,
		RunID:      opts.RunID,
	})
	result.Stages = append(result.Stages, StageLog{
		EntityKey: key,
		Status:    currentStatus,
		Action:    config.ActionAdvanceStatus,
		Duration:  time.Since(stageStart),
	})
	result.StagesCompleted++
	if c.workflowSvc.IsTerminalStatus(transitionResult.ToStatus) {
		result.FinalStatus = transitionResult.ToStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}
	result.FinalStatus = transitionResult.ToStatus
	return stageOutcome{nextStatus: transitionResult.ToStatus}
}

// handleAdvanceStatus handles the advance_status action type: transitions the
// entity to the next status without dispatching an agent.
func (c *RunController) handleAdvanceStatus(
	ctx context.Context, key, currentStatus string,
	nextInfo *services.NextStatusInfo,
	action *config.PopulatedAction, opts RunOptions,
	result *RunResult, stageStart, startTime time.Time,
) stageOutcome {
	if opts.DryRun {
		result.Stages = append(result.Stages, StageLog{
			EntityKey: key,
			Status:    currentStatus,
			Action:    action.Action,
			AgentType: action.AgentType,
			Provider:  action.Provider,
			Duration:  time.Since(stageStart),
		})
		result.StagesCompleted++
		postInfo, ok := c.dryRunPostActionStatus(ctx, key, currentStatus, nextInfo, "advance_status", opts, result, startTime)
		if !ok {
			return stageOutcome{done: true}
		}
		return c.dryRunNextOutcome(currentStatus, postInfo, result, startTime)
	}

	nextInfo, err := c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "advance_status",
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	if len(nextInfo.AvailableTransitions) == 0 {
		result.FinalStatus = currentStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets
	transResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, guardedTransitionOptions(opts, currentStatus, targetStatus, nextInfo))
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "transition",
			Error:     fmt.Sprintf("transition to %s failed: %v", targetStatus, err),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	// Emit run.stage.transition after a successful transition.
	emitStageTransition(ctx, opts.Observability, stageTransitionParams{
		EntityKey:  key,
		FromStatus: currentStatus,
		ToStatus:   transResult.ToStatus,
		RunID:      opts.RunID,
	})

	result.Stages = append(result.Stages, StageLog{
		EntityKey: key,
		Status:    currentStatus,
		Action:    action.Action,
		AgentType: action.AgentType,
		Provider:  action.Provider,
		Duration:  time.Since(stageStart),
	})
	result.StagesCompleted++

	if c.workflowSvc.IsTerminalStatus(transResult.ToStatus) {
		result.FinalStatus = transResult.ToStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}
	return stageOutcome{nextStatus: transResult.ToStatus}
}

// resultContractLegacy and resultContractGateResultV1 are the REQ-F-006
// `result_contract` values a workflow step can select.
const (
	resultContractLegacy       = "legacy"
	resultContractGateResultV1 = "gate_result_v1"
)

// resultContractFor resolves the REQ-F-006 `result_contract` for the
// dispatched step from stepInfo — the NextStatusInfo the workflow layer
// resolved for the entity's pre-dispatch status (services.EntityService
// populates ResultContract from the step's `result_contract` YAML field,
// defaulting to "legacy" on omission — T-E34-F05-005). Every caller in this
// file goes through this one function, so a future contract value only
// needs a case added here.
func resultContractFor(stepInfo *services.NextStatusInfo) (string, error) {
	contract := resultContractLegacy
	if stepInfo != nil && stepInfo.ResultContract != "" {
		contract = stepInfo.ResultContract
	}
	switch contract {
	case resultContractLegacy, resultContractGateResultV1:
		return contract, nil
	default:
		return "", fmt.Errorf("unknown result_contract %q", contract)
	}
}

// ingestGateResultForDispatch is handleSpawnAgent's gate_result_v1
// continuation: it treats dispatchResult.Stdout as a candidate worker-control
// envelope (the ENTIRE trimmed stdout must be the envelope JSON object,
// matching recommendedOutcome's own "whole trimmed value only" safety
// property) and delegates to the shared IngestGateResult boundary
// (gate_ingest.go) that Rider's `--apply-result` CLI surface also calls.
//
// RetirementConfirmed is resolved in two phases rather than passed as true
// unconditionally (T-E34-F05-004 rework, UAT CRITICAL finding #2): the
// Run() main loop (see the `for { ... currentStatus = outcome.nextStatus }`
// loop above) keeps dispatching further stages for the SAME entity, SAME
// claim/lease session, within this SAME `shark run` invocation whenever the
// resolved target status is non-terminal — a gate_result_v1 step is no
// different from any other step in that respect. Confirming retirement
// (which releases the lease via gatepersist.Coordinator's Lease.Release)
// unconditionally on the first gate stage would free the lease while later
// stages are still about to dispatch agents against the same entity,
// exactly the cross-stage lease-lifetime bug the UAT report identified.
// So: ingest once with RetirementConfirmed: false to learn the resolved
// target status, then — only if that status is terminal (i.e. only if the
// Run() loop is actually about to stop for this entity, mirroring the
// `c.workflowSvc.IsTerminalStatus(toStatus)` check immediately below this
// function's call site) — ingest again with RetirementConfirmed: true to
// finalize retirement. The second call is safe and non-duplicating: per
// gatepersist.Coordinator.Persist's PersistenceStateTransitioned branch, it
// only re-verifies the already-applied transition and (idempotently) closes
// out retirement — it does not repeat any note write or the transition
// itself (the same pattern run_resume.go's resumeGateIngestIfConfigured
// already documents and relies on).
// gateStageRunID scopes gaterun's create-once persistence identity
// (gaterun.RunDir/CreateResult, and the durable "run_id" note-metadata tag
// used for reconciliation) to ONE dispatched gate_result_v1 stage, rather
// than reusing runID (opts.RunID) — the single correlation identifier
// generated once at the top of a `shark run` invocation and threaded
// unchanged through every stage's observability events — as the persistence
// identity too.
//
// code-review round-7 Finding 1: gaterun.CreateResult's create-once contract
// treats a run_id as identifying exactly ONE persist operation and returns a
// *gaterun.ConflictError when a second, differently-digested envelope is
// written under the same run_id. Run()'s main loop (controller.go's
// `for { ... currentStatus = outcome.nextStatus }`) keeps opts.RunID
// constant across every iteration of one invocation, so a workflow with two
// or more consecutive gate_result_v1 steps for the same entity in one
// invocation (e.g. code_review -> qa) reused opts.RunID for both stages and
// the second stage's envelope collided with the first stage's
// already-accepted result.json.
//
// stageN (the 1-based dispatch iteration counter, already passed to this
// function for transcript naming) is a stable, monotonically-increasing
// per-stage discriminator: every retry of ingestGateResultForDispatch WITHIN
// one dispatched stage (its own initial ingest call plus a possible
// follow-up retirement-confirm call, both inside this one function
// invocation) computes the identical gateStageRunID, so gaterun's own
// replay/idempotent-resume semantics for a SINGLE stage are unaffected. A
// later, different stage is a different iteration and therefore gets its
// own run directory, unable to collide with an earlier stage's accepted
// result. opts.RunID itself is left untouched everywhere else in this file
// (every emitStage*/transcript call keeps using opts.RunID directly) so a
// full `shark run` invocation remains greppable from shark.log by one
// correlation id.
func gateStageRunID(runID string, stageN int) string {
	return fmt.Sprintf("%s-g%d", runID, stageN)
}

func (c *RunController) ingestGateResultForDispatch(
	ctx context.Context, key, currentStatus string,
	nextInfo *services.NextStatusInfo, action *config.PopulatedAction, opts RunOptions,
	dispatchResult *DispatchResult, transcriptDisabled *bool, stageN int,
) (string, *gaterun.StatusProjection, error) {
	if c.gateIngest == nil || c.gateIngest.Coordinator == nil {
		return "", nil, fmt.Errorf("gate_result_v1 step %s requires a configured GateResult persistence coordinator", key)
	}

	// T-E34-F05-005: the workflow layer now resolves outcome_roles per step
	// (nextInfo.OutcomeRoles, from the step's own `outcome_roles` YAML map).
	// c.gateIngest.OutcomeRoles is kept only as a fallback for callers that
	// still inject a flat run-wide override (e.g. existing tests) when the
	// resolved step map is empty.
	outcomeRoles := nextInfo.OutcomeRoles
	if len(outcomeRoles) == 0 {
		outcomeRoles = c.gateIngest.OutcomeRoles
	}

	envelopeBytes := []byte(strings.TrimSpace(dispatchResult.Stdout))
	baseReq := GateIngestRequest{
		EnvelopeBytes: envelopeBytes,
		Coordinator:   c.gateIngest.Coordinator,
		ProjectRoot:   opts.ProjectRoot,
		RunID:         gateStageRunID(opts.RunID, stageN),
		EntityKey:     key,
		EntityType:    models.EntityType(opts.EntityType),
		SourceStatus:  currentStatus,
		Gate:          currentStatus,
		Session:       gatepersist.Session{ID: opts.SessionID},
		OutcomeRoles:  outcomeRoles,
		Outcomes:      nextInfo.Outcomes,
	}

	req := baseReq
	req.RetirementConfirmed = false
	ingestResult, err := IngestGateResult(ctx, req)
	if err != nil {
		return "", nil, err
	}

	// Only confirm retirement (and release the lease) when the resolved
	// target status is terminal — i.e. only when the Run() loop is about to
	// stop dispatching this entity in this invocation, not merely because
	// this one gate stage finished.
	if c.workflowSvc.IsTerminalStatus(ingestResult.ToStatus) {
		retireReq := baseReq
		retireReq.RetirementConfirmed = true
		// RunConcluded: true — this IS the Run() loop's last dispatch for
		// this entity/session (the terminal status just resolved means the
		// main loop's `if outcome.done { return }` fires next), so both
		// signals gatepersist.Coordinator requires for release are true here.
		retireReq.RunConcluded = true
		retireResult, retireErr := IngestGateResult(ctx, retireReq)
		if retireErr != nil {
			return "", nil, retireErr
		}
		ingestResult = retireResult
	}

	relPath := c.maybeWriteTranscript(
		ctx, opts, transcriptDisabled,
		key, stageN, currentStatus, action.Provider,
		dispatchResult.Command, dispatchResult.ExitCode,
		dispatchResult.Duration.Milliseconds(),
		dispatchResult.Stdout, dispatchResult.Stderr,
	)
	emitStageComplete(ctx, opts.Observability, stageCompleteParams{
		EntityKey:      key,
		Status:         currentStatus,
		AgentType:      action.AgentType,
		Provider:       action.Provider,
		ExitCode:       dispatchResult.ExitCode,
		DurationMS:     dispatchResult.Duration.Milliseconds(),
		NextStatus:     ingestResult.ToStatus,
		RunID:          opts.RunID,
		TranscriptPath: relPath,
	})
	emitStageTransition(ctx, opts.Observability, stageTransitionParams{
		EntityKey:  key,
		FromStatus: currentStatus,
		ToStatus:   ingestResult.ToStatus,
		RunID:      opts.RunID,
	})
	return ingestResult.ToStatus, ingestResult.Status, nil
}

// targetStatusForDispatch resolves a worker's optional semantic outcome to a
// configured status. Existing prompts that do not emit an outcome retain the
// pass-first transition contract.
func targetStatusForDispatch(nextInfo *services.NextStatusInfo, stdout string) (string, error) {
	if len(nextInfo.AvailableTransitions) == 0 {
		return "", fmt.Errorf("no transition is available")
	}
	outcome, specified, err := recommendedOutcome(stdout)
	if err != nil {
		return "", err
	}
	if !specified {
		return nextInfo.AvailableTransitions[0].TargetStatus, nil //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets
	}
	target, ok := nextInfo.Outcomes[strings.ToLower(outcome)]
	if !ok {
		return "", fmt.Errorf("agent recommended unknown outcome %q", outcome)
	}
	return target, nil
}

// guardedTransitionOptions supplies the runner's lease and observed source
// status to every parent-driven transition. EntityService only enforces these
// fields when advance_guard is enabled, so legacy workflows keep their existing
// behavior while guarded deployments cannot bypass replay protection.
func guardedTransitionOptions(runOpts RunOptions, fromStatus, targetStatus string, nextInfo *services.NextStatusInfo) services.TransitionOptions {
	outcome := "pass"
	for candidate, target := range nextInfo.Outcomes {
		if strings.EqualFold(target, targetStatus) {
			outcome = candidate
			break
		}
	}
	return services.TransitionOptions{
		SessionID:    runOpts.SessionID,
		FromStatus:   fromStatus,
		Outcome:      outcome,
		GuardAdvance: true,
	}
}

// recommendedOutcome extracts the explicit final worker recommendation. It
// intentionally accepts only a whole trimmed line so prose mentioning the
// phrase cannot alter the workflow route.
//
// It also accepts the shark-rider worker-return contract's JSON alternative,
// `{"outcome": "<key>"}` — but only when the ENTIRE trimmed stdout is that
// JSON object. This preserves the same safety property as the text-line
// format: outcome-shaped JSON merely mentioned within a longer message (e.g.
// prose describing what the worker considered returning) must not alter the
// workflow route.
//
// A non-nil error means the worker's stdout was recognized as an attempted
// JSON outcome object (it starts with `{`) but failed to parse, or the
// `outcome` field was the wrong type. B046 was filed because a malformed
// outcome silently fell through to the pass-first target instead of
// surfacing a parse error; this mirrors parseQuestionResponseHandoff's
// fail-loud behavior on invalid JSON rather than repeating that mistake.
func recommendedOutcome(stdout string) (string, bool, error) {
	const prefix = "recommended outcome:"
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if len(line) >= len(prefix) && strings.EqualFold(line[:len(prefix)], prefix) {
			return strings.TrimSpace(line[len(prefix):]), true, nil
		}
	}

	trimmed := strings.TrimSpace(stdout)
	if strings.HasPrefix(trimmed, "{") {
		var body struct {
			Outcome string `json:"outcome"`
		}
		if err := json.Unmarshal([]byte(trimmed), &body); err != nil {
			return "", false, fmt.Errorf("invalid JSON outcome in worker stdout: %w", err)
		}
		// Report as specified even when empty, matching the text-line
		// format: an empty/unrecognized outcome fails loud downstream in
		// targetStatusForDispatch's outcome lookup rather than silently
		// falling back to the pass-first transition.
		return strings.TrimSpace(body.Outcome), true, nil
	}

	return "", false, nil
}

func (c *RunController) dryRunNextOutcome(
	currentStatus string,
	nextInfo *services.NextStatusInfo,
	result *RunResult,
	startTime time.Time,
) stageOutcome {
	if nextInfo == nil || nextInfo.IsTerminal || len(nextInfo.AvailableTransitions) == 0 {
		result.FinalStatus = currentStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	nextStatus := nextInfo.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets
	return stageOutcome{
		nextStatus: nextStatus,
		nextInfo:   c.simulatedDryRunNextStatus(nextStatus),
	}
}

func (c *RunController) simulatedDryRunNextStatus(status string) *services.NextStatusInfo {
	transitions := c.workflowSvc.GetTransitionInfo(status)
	wrapped := make([]services.TransitionInfoWithAction, 0, len(transitions))
	for _, transition := range transitions {
		wrapped = append(wrapped, services.TransitionInfoWithAction{
			TransitionInfo: transition,
		})
	}
	return &services.NextStatusInfo{
		EntityKey:            "__dry_run_simulated__",
		CurrentStatus:        status,
		AvailableTransitions: wrapped,
		IsTerminal:           c.workflowSvc.IsTerminalStatus(status),
	}
}

func (c *RunController) dryRunPostActionStatus(
	ctx context.Context,
	key, currentStatus string,
	nextInfo *services.NextStatusInfo,
	phase string,
	opts RunOptions,
	result *RunResult,
	startTime time.Time,
) (*services.NextStatusInfo, bool) {
	if nextInfo != nil && nextInfo.EntityKey == "__dry_run_simulated__" {
		return nextInfo, true
	}
	if nextInfo != nil && !strings.EqualFold(nextInfo.CurrentStatus, currentStatus) {
		return nextInfo, true
	}

	postInfo, err := c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     phase,
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return nil, false
	}
	return postInfo, true
}

// handleSpawnAgent handles the spawn_agent action type: dispatches an agent,
// gates status advancement on exit code 0.
//
// stageN is the 1-based iteration counter from the run loop; it becomes the
// numeric prefix on transcript filenames (e.g. "1-in_development-anthropic.log").
//
// transcriptDisabled is a pointer to a run-scoped latch owned by Run(). Once a
// transcript write fails, the caller emits run.transcript.warning exactly once
// and sets *transcriptDisabled = true to suppress all further attempts for the
// rest of the run.
func (c *RunController) handleSpawnAgent(
	ctx context.Context, key, currentStatus string,
	nextInfo *services.NextStatusInfo,
	action *config.PopulatedAction, vars map[string]string, opts RunOptions,
	result *RunResult, stageStart, startTime time.Time,
	stageN int, transcriptDisabled *bool,
) stageOutcome {
	// stepInfo pins the dispatched step's own NextStatusInfo (currentStatus's
	// result_contract/outcome_roles/outcomes) before nextInfo is reassigned
	// below to the POST-dispatch read. The gate_result_v1 branch resolves its
	// contract and role map from this pinned snapshot, not the reassigned
	// variable, so a step can never be validated against a different step's
	// configuration even if the entity's status already moved by the time the
	// post-dispatch read runs (e.g. an out-of-band transition mid-dispatch).
	stepInfo := nextInfo

	dispatcher, err := c.selectDispatcher(action.Provider)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "dispatcher_selection",
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	assembledPrompt, err := c.assembler.AssemblePrompt(ctx, PromptAssemblyInput{
		Instruction: action.Instruction,
		AgentType:   action.AgentType,
		Vars:        vars,
	})
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "prompt_assembly",
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	input := DispatchInput{
		Instruction: assembledPrompt,
		WorkingDir:  opts.WorkingDir,
		EntityKey:   key,
		Status:      currentStatus,
		AgentType:   action.AgentType,
		Model:       action.Model,
	}

	if opts.DryRun {
		result.Stages = append(result.Stages, StageLog{
			EntityKey: key,
			Status:    currentStatus,
			Action:    action.Action,
			AgentType: action.AgentType,
			Provider:  action.Provider,
			Duration:  time.Since(stageStart),
		})
		result.StagesCompleted++

		postInfo, ok := c.dryRunPostActionStatus(ctx, key, currentStatus, nextInfo, "post_dispatch", opts, result, startTime)
		if !ok {
			return stageOutcome{done: true}
		}
		return c.dryRunNextOutcome(currentStatus, postInfo, result, startTime)
	}

	// Emit run.stage.dispatch just before invoking the agent. The command string
	// is pre-built from the dispatcher's BuildCommand(input) so the event carries
	// the EXACT CLI invocation (e.g. "claude -p ...") that will be executed. The
	// command is truncated inside emitStageDispatch.
	//
	// BuildCommand can fail when an argument cannot be represented as a POSIX
	// shell word (currently: NUL-byte in any argv element — see
	// errShellQuoteNUL in shell_quote.go). os/exec would reject such argv
	// with EINVAL anyway, so we surface the condition as a run.stage.error
	// with phase="shell_quote" and skip Dispatch entirely.
	dispatchCmd, buildErr := dispatcher.BuildCommand(input)
	if buildErr != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "shell_quote",
			Error:     buildErr.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}
	emitStageDispatch(ctx, opts.Observability, stageDispatchParams{
		EntityKey: key,
		Status:    currentStatus,
		AgentType: action.AgentType,
		Provider:  action.Provider,
		Command:   dispatchCmd,
		RunID:     opts.RunID,
	})

	dispatchResult, err := dispatcher.Dispatch(ctx, input)
	if err != nil {
		// Prefer *AgentFailedError fields; fall back to the DispatchResult the
		// dispatcher returned alongside the error (real dispatchers return both;
		// tests sometimes return result only). Phase is always "dispatch" for
		// dispatch-path failures.
		//
		// A transcript is written on the error path so operators have a captured
		// record of the failing invocation's stdout/stderr. The transcript path
		// (when successfully written) rides on the same run.stage.error event
		// via TranscriptPath.
		errMsg := err.Error()
		var agentErr *AgentFailedError
		switch {
		case errors.As(err, &agentErr):
			c.recordDispatchFailure(
				ctx, opts, result, startTime,
				transcriptDisabled, stageN,
				currentStatus, action.Provider, errMsg,
				agentErr.Command, agentErr.ExitCode,
				0, // *AgentFailedError has no Duration — use 0
				agentErr.Stdout, agentErr.Stderr,
			)
		case dispatchResult != nil:
			c.recordDispatchFailure(
				ctx, opts, result, startTime,
				transcriptDisabled, stageN,
				currentStatus, action.Provider, errMsg,
				dispatchResult.Command, dispatchResult.ExitCode,
				dispatchResult.Duration.Milliseconds(),
				dispatchResult.Stdout, dispatchResult.Stderr,
			)
		default:
			// No DispatchResult available — fall back to the pre-built command.
			// A transcript is still attempted (it will record exit=0/duration=0/
			// empty stdout/stderr, which is operator-visible evidence that the
			// dispatcher returned an error with no captured output).
			c.recordDispatchFailure(
				ctx, opts, result, startTime,
				transcriptDisabled, stageN,
				currentStatus, action.Provider, errMsg,
				dispatchCmd, 0, 0, "", "",
			)
		}
		return stageOutcome{done: true}
	}

	stage := StageLog{
		EntityKey: key,
		Status:    currentStatus,
		Action:    action.Action,
		AgentType: action.AgentType,
		Provider:  action.Provider,
		Duration:  dispatchResult.Duration,
		ExitCode:  dispatchResult.ExitCode,
	}
	if dispatchResult.ExitCode == 0 && opts.EntityType != string(models.EntityTypeQuestion) {
		stage.OutputSummary = dispatchResult.Stdout
	}
	result.Stages = append(result.Stages, stage)

	// Gate advancement on exit code 0.
	if dispatchResult.ExitCode != 0 {
		// Mock-style path where Dispatch returns (result, nil) with a non-zero
		// ExitCode. The transcript captures the failing stdout/stderr so
		// operators can debug offline.
		errMsg := fmt.Sprintf("agent exited with code %d", dispatchResult.ExitCode)
		c.recordDispatchFailure(
			ctx, opts, result, startTime,
			transcriptDisabled, stageN,
			currentStatus, action.Provider, errMsg,
			dispatchResult.Command, dispatchResult.ExitCode,
			dispatchResult.Duration.Milliseconds(),
			dispatchResult.Stdout, dispatchResult.Stderr,
		)
		// The failing stage is already appended to result.Stages above; adjust
		// StagesCompleted so it reflects only SUCCESSFUL stages.
		result.StagesCompleted = len(result.Stages) - 1
		return stageOutcome{done: true}
	}

	// A Question responder has one additional parent-owned success step, kept
	// out of this already-large function: see handleQuestionResponseHandoff.
	if opts.EntityType == string(models.EntityTypeQuestion) {
		return c.handleQuestionResponseHandoff(ctx, key, currentStatus, action, vars, opts, result, startTime, dispatchResult)
	}

	result.StagesCompleted++

	nextInfo, err = c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "post_dispatch",
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	if len(nextInfo.AvailableTransitions) == 0 {
		// No transitions available. If the post-dispatch CurrentStatus differs
		// from the pre-dispatch status, the agent itself advanced the status
		// (or a transition happened through a side channel) — treat that as an
		// implicit transition so observers still see complete + transition
		// events with the actual landing status. Otherwise report the stage
		// completed with an empty next_status.
		nextStatus := ""
		if nextInfo.CurrentStatus != "" && nextInfo.CurrentStatus != currentStatus {
			nextStatus = nextInfo.CurrentStatus
		}

		// Write the per-dispatch transcript when capture is enabled. relPath
		// is "" when capture is disabled, the run-scoped latch has tripped, or
		// the write failed — matching the contract that the `transcript_path`
		// attribute is emitted ONLY on success.
		relPath := c.maybeWriteTranscript(
			ctx, opts, transcriptDisabled,
			key, stageN, currentStatus, action.Provider,
			dispatchResult.Command, dispatchResult.ExitCode,
			dispatchResult.Duration.Milliseconds(),
			dispatchResult.Stdout, dispatchResult.Stderr,
		)
		emitStageComplete(ctx, opts.Observability, stageCompleteParams{
			EntityKey:      key,
			Status:         currentStatus,
			AgentType:      action.AgentType,
			Provider:       action.Provider,
			ExitCode:       dispatchResult.ExitCode,
			DurationMS:     dispatchResult.Duration.Milliseconds(),
			NextStatus:     nextStatus,
			RunID:          opts.RunID,
			TranscriptPath: relPath,
		})

		if nextStatus != "" {
			emitStageTransition(ctx, opts.Observability, stageTransitionParams{
				EntityKey:  key,
				FromStatus: currentStatus,
				ToStatus:   nextStatus,
				RunID:      opts.RunID,
			})
			result.FinalStatus = nextStatus
		} else {
			result.FinalStatus = currentStatus
		}
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	// T-E34-F05-004/REQ-F-006: resolve this step's result contract before
	// touching either transition path. A gate_result_v1 step never falls
	// through to the legacy parser below on any ingestion failure — it fails
	// closed instead (see ingestGateResultForDispatch/IngestGateResult).
	// Resolved from stepInfo (the dispatched step's own pinned snapshot, see
	// above), never from the post-dispatch nextInfo reassigned above.
	contract, err := resultContractFor(stepInfo)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "result_contract",
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	var toStatus string
	if contract == resultContractGateResultV1 {
		var gateStatus *gaterun.StatusProjection
		toStatus, gateStatus, err = c.ingestGateResultForDispatch(ctx, key, currentStatus, stepInfo, action, opts, dispatchResult, transcriptDisabled, stageN)
		if err != nil {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "gate_ingest",
				Error:     err.Error(),
				RunID:     opts.RunID,
			})
			return stageOutcome{done: true}
		}
		if gateStatus != nil && len(result.Stages) > 0 {
			result.Stages[len(result.Stages)-1].GateStatus = gateStatus
		}
	} else {
		targetStatus, err := targetStatusForDispatch(nextInfo, dispatchResult.Stdout)
		if err != nil {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "outcome",
				Error:     err.Error(),
				RunID:     opts.RunID,
			})
			return stageOutcome{done: true}
		}

		// Write the per-dispatch transcript when capture is enabled. Stdout is
		// DELIBERATELY excluded from the run.stage.complete event because
		// transcripts are captured on this separate channel; the complete event
		// is on a hot path. relPath is "" when capture is disabled, the run-scoped
		// latch has tripped, or the write failed — matching the contract that
		// the `transcript_path` attribute is emitted ONLY on success.
		relPath := c.maybeWriteTranscript(
			ctx, opts, transcriptDisabled,
			key, stageN, currentStatus, action.Provider,
			dispatchResult.Command, dispatchResult.ExitCode,
			dispatchResult.Duration.Milliseconds(),
			dispatchResult.Stdout, dispatchResult.Stderr,
		)

		// Emit run.stage.complete now that we know the next status.
		emitStageComplete(ctx, opts.Observability, stageCompleteParams{
			EntityKey:      key,
			Status:         currentStatus,
			AgentType:      action.AgentType,
			Provider:       action.Provider,
			ExitCode:       dispatchResult.ExitCode,
			DurationMS:     dispatchResult.Duration.Milliseconds(),
			NextStatus:     targetStatus,
			RunID:          opts.RunID,
			TranscriptPath: relPath,
		})

		transResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, guardedTransitionOptions(opts, currentStatus, targetStatus, nextInfo))
		if err != nil {
			recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
				EntityKey: key,
				Status:    currentStatus,
				Phase:     "transition",
				Error:     fmt.Sprintf("transition to %s failed: %v", targetStatus, err),
				RunID:     opts.RunID,
			})
			return stageOutcome{done: true}
		}

		// Emit run.stage.transition after a successful transition.
		emitStageTransition(ctx, opts.Observability, stageTransitionParams{
			EntityKey:  key,
			FromStatus: currentStatus,
			ToStatus:   transResult.ToStatus,
			RunID:      opts.RunID,
		})
		toStatus = transResult.ToStatus
	}

	if c.workflowSvc.IsTerminalStatus(toStatus) {
		result.FinalStatus = toStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}
	return stageOutcome{nextStatus: toStatus}
}

// handleQuestionResponseHandoff is handleSpawnAgent's Question-specific
// success continuation, kept separate to keep handleSpawnAgent's own length
// down. The worker returns bounded data; the parent binds it to the actual
// entity, lease session, and routed responder, then persists it before the
// lease is released by run.go. Do not use the generic transition path here:
// a second responder must not run under the first responder's lease.
func (c *RunController) handleQuestionResponseHandoff(
	ctx context.Context, key, currentStatus string,
	action *config.PopulatedAction, vars map[string]string, opts RunOptions,
	result *RunResult, startTime time.Time,
	dispatchResult *DispatchResult,
) stageOutcome {
	if c.questionResponses == nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "question_response_handoff",
			Error:     "Question response persister is not configured",
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}
	response, err := parseQuestionResponseHandoff(dispatchResult.Stdout)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "question_response_handoff",
			Error:     err.Error(),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}
	responder := strings.TrimSpace(vars["current_responder"])
	if responder == "" || opts.SessionID == "" {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "question_response_handoff",
			Error:     "Question response requires a routed responder and parent lease session",
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}
	response.Key = key
	response.SessionID = opts.SessionID
	response.Responder = responder
	if err := c.questionResponses.PersistQuestionResponse(ctx, response); err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "question_response_handoff",
			Error:     fmt.Sprintf("persist Question response: %v", err),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}

	refreshed, err := c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
			EntityKey: key,
			Status:    currentStatus,
			Phase:     "question_response_handoff",
			Error:     fmt.Sprintf("refresh Question after response: %v", err),
			RunID:     opts.RunID,
		})
		return stageOutcome{done: true}
	}
	result.StagesCompleted++
	emitStageComplete(ctx, opts.Observability, stageCompleteParams{
		EntityKey:  key,
		Status:     currentStatus,
		AgentType:  action.AgentType,
		Provider:   action.Provider,
		ExitCode:   dispatchResult.ExitCode,
		DurationMS: dispatchResult.Duration.Milliseconds(),
		NextStatus: refreshed.CurrentStatus,
		RunID:      opts.RunID,
	})
	result.FinalStatus = refreshed.CurrentStatus
	result.Outcome = "completed"
	result.TotalDuration = time.Since(startTime)
	return stageOutcome{done: true}
}

const questionResponseHandoffPrefix = "QUESTION_RESPONSE_JSON:"

// parseQuestionResponseHandoff accepts exactly one line of compact worker
// output. Keeping the marker line-oriented prevents surrounding explanation
// from becoming persistence input and leaves validation of all byte/content
// limits to QuestionService.
func parseQuestionResponseHandoff(stdout string) (QuestionResponseHandoff, error) {
	var found *QuestionResponseHandoff
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, questionResponseHandoffPrefix) {
			continue
		}
		if found != nil {
			return QuestionResponseHandoff{}, errors.New("Question worker emitted more than one response handoff")
		}
		var handoff QuestionResponseHandoff
		body := strings.TrimSpace(strings.TrimPrefix(line, questionResponseHandoffPrefix))
		if err := json.Unmarshal([]byte(body), &handoff); err != nil {
			return QuestionResponseHandoff{}, fmt.Errorf("invalid Question response handoff: %w", err)
		}
		if strings.TrimSpace(handoff.Summary) == "" || strings.TrimSpace(handoff.EvidencePointer) == "" {
			return QuestionResponseHandoff{}, errors.New("Question response handoff requires summary and evidence_pointer")
		}
		found = &handoff
	}
	if found == nil {
		return QuestionResponseHandoff{}, errors.New("Question worker did not emit QUESTION_RESPONSE_JSON handoff")
	}
	return *found, nil
}

// recordStageFailure marks the run as failed and emits one run.stage.error
// event. Callers are responsible for the return statement (stageOutcome{done:
// true} from handlers, (result, nil) from the main Run loop); this helper
// only mutates result and emits the event.
//
// params must be fully populated by the caller (EntityKey, Status, Phase,
// Error, RunID at minimum; ExitCode/Stderr/Stdout/Command/TranscriptPath for
// dispatch-path failures).
func recordStageFailure(
	ctx context.Context, opts RunOptions,
	result *RunResult, startTime time.Time,
	params stageErrorParams,
) {
	result.FinalStatus = params.Status
	result.Outcome = "failed"
	result.Error = params.Error
	result.TotalDuration = time.Since(startTime)
	emitStageError(ctx, opts.Observability, params)
}

// recordDispatchFailure writes a transcript for a failed dispatch and records
// the stage failure. Phase is always "dispatch". The transcript path rides on
// the same run.stage.error event when the write succeeds.
//
// Callers provide the command/exit/duration/stdout/stderr from whichever
// source carries them (AgentFailedError fields, DispatchResult fields, or the
// pre-built dispatchCmd with zero-valued outputs when neither is available).
func (c *RunController) recordDispatchFailure(
	ctx context.Context, opts RunOptions,
	result *RunResult, startTime time.Time,
	transcriptDisabled *bool, stageN int,
	currentStatus, provider, errMsg, command string,
	exitCode int, durationMS int64,
	stdout, stderr string,
) {
	relPath := c.maybeWriteTranscript(
		ctx, opts, transcriptDisabled,
		result.EntityKey, stageN, currentStatus, provider,
		command, exitCode, durationMS, stdout, stderr,
	)
	recordStageFailure(ctx, opts, result, startTime, stageErrorParams{
		EntityKey:      result.EntityKey,
		Status:         currentStatus,
		Phase:          "dispatch",
		Error:          errMsg,
		ExitCode:       exitCode,
		Stderr:         stderr,
		Stdout:         stdout,
		Command:        command,
		RunID:          opts.RunID,
		TranscriptPath: relPath,
	})
}

// maybeWriteTranscript writes a per-dispatch transcript file if and only if
// observability is configured to capture transcripts, a project root was
// supplied, and the run-scoped disable latch has not tripped. It returns the
// project-relative transcript path on success (empty string in every other
// case). On write failure it emits exactly one run.transcript.warning per run
// and sets *transcriptDisabled = true to suppress all further attempts in
// this run.
//
// Parameters mirror writeTranscript: command/exitCode/durationMS/stdout/stderr
// are the raw dispatch outputs; stageN/status/provider determine the filename.
// The caller is expected to pass the LATEST command (preferring
// dispatchResult.Command, then agentErr.Command, then dispatchCmd) so that
// the transcript's COMMAND line exactly matches what was actually executed.
func (c *RunController) maybeWriteTranscript(
	ctx context.Context, opts RunOptions, transcriptDisabled *bool,
	entityKey string, stageN int, status, provider, command string,
	exitCode int, durationMS int64, stdout, stderr string,
) string {
	if !opts.Observability.CaptureAgentTranscripts {
		return ""
	}
	if opts.ProjectRoot == "" {
		return ""
	}
	if transcriptDisabled == nil || *transcriptDisabled {
		return ""
	}

	rel, err := writeTranscript(
		opts.ProjectRoot, opts.RunID, entityKey, stageN,
		status, provider, command,
		exitCode, durationMS, stdout, stderr,
	)
	if err != nil {
		// Warning carries the intended project-relative path so operators can
		// still locate the missing transcript in their mental model even when
		// the file itself could not be created.
		emitTranscriptWarning(ctx, opts.Observability, opts.RunID,
			relTranscriptPath(opts.RunID, entityKey, stageN, status, provider), err)
		*transcriptDisabled = true
		return ""
	}
	return rel
}

// selectDispatcher returns the AgentDispatcher for the given provider name.
// An empty provider selects the "" key (default dispatcher).
// Returns a descriptive error if the provider is not found.
func (c *RunController) selectDispatcher(provider string) (AgentDispatcher, error) {
	d, ok := c.dispatchers[provider]
	if ok {
		return d, nil
	}

	// Build list of available keys for the error message.
	keys := make([]string, 0, len(c.dispatchers))
	for k := range c.dispatchers {
		if k == "" {
			keys = append(keys, `""`)
		} else {
			keys = append(keys, k)
		}
	}
	return nil, fmt.Errorf(
		"no dispatcher configured for provider %q (available: %s)",
		provider,
		strings.Join(keys, ", "),
	)
}
