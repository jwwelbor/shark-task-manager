// Package runner provides the AgentDispatcher interface and related types for
// invoking external AI agents (Claude, Codex, etc.) as part of the E22 run loop.
// This file defines the RunController, its interfaces, and supporting data types.
package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// RunOptions controls run loop behavior.
type RunOptions struct {
	// DryRun, when true, prints actions but does not dispatch agents or advance status.
	DryRun bool

	// Verbose, when true, prints detailed stage progress to stderr.
	Verbose bool

	// WorkingDir is an optional working directory override for agent processes.
	WorkingDir string

	// RunID is a correlation identifier generated once at the top of runRun.
	// It is threaded through all stage events so a single run can be grepped
	// from shark.log.
	RunID string

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
}

// StageLog captures per-stage execution details.
type StageLog struct {
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
}

// RunController implements the core orchestration loop: read entity state,
// read orchestrator action, dispatch to agents, gate on exit code, loop until
// terminal/pause/failure.
//
// All dependencies are injected via RunControllerDeps; no global state is used,
// enabling full mock-based testing without a real database or agent processes.
type RunController struct {
	transitioner EntityTransitioner
	placeholders PlaceholderGenerator
	actionSvc    config.ActionService
	workflowSvc  *workflow.Service
	dispatchers  map[string]AgentDispatcher
	assembler    PromptAssembler
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
		transitioner: deps.Transitioner,
		placeholders: deps.Placeholders,
		actionSvc:    deps.ActionSvc,
		workflowSvc:  deps.WorkflowSvc,
		dispatchers:  deps.Dispatchers,
		assembler:    assembler,
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
//  4. Handles the action type: spawn_agent, advance_status, pause/wait_for_triage/check_or_resume, archive.
//  5. For spawn_agent: dispatches agent, gates advancement on exit code 0.
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
		result.Outcome = "already_terminal"
		result.TotalDuration = time.Since(startTime)
		return result, nil
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

		// Step 6: Route by action type.
		var outcome stageOutcome
		switch action.Action {
		case config.ActionPause, config.ActionWaitForTriage, config.ActionCheckOrResume:
			result.FinalStatus = currentStatus
			result.Outcome = "paused"
			result.TotalDuration = time.Since(startTime)
			return result, nil

		case config.ActionArchive:
			result.Stages = append(result.Stages, StageLog{
				Status:   currentStatus,
				Action:   action.Action,
				Duration: time.Since(stageStart),
			})
			result.StagesCompleted++
			result.FinalStatus = currentStatus
			result.Outcome = "completed"
			result.TotalDuration = time.Since(startTime)
			return result, nil

		case config.ActionAdvanceStatus:
			outcome = c.handleAdvanceStatus(ctx, key, currentStatus, nextInfo, action, opts, result, stageStart, startTime)

		case config.ActionSpawnAgent:
			outcome = c.handleSpawnAgent(ctx, key, currentStatus, nextInfo, action, vars, opts, result, stageStart, startTime, iteration, &transcriptDisabled)

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
		}
	}
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
	transResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, services.TransitionOptions{})
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
		Status:    currentStatus,
		Action:    action.Action,
		AgentType: action.AgentType,
		Provider:  action.Provider,
		Duration:  dispatchResult.Duration,
		ExitCode:  dispatchResult.ExitCode,
	}
	if dispatchResult.ExitCode == 0 {
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
			stageN, currentStatus, action.Provider,
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

	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets

	// Write the per-dispatch transcript when capture is enabled. Stdout is
	// DELIBERATELY excluded from the run.stage.complete event because
	// transcripts are captured on this separate channel; the complete event
	// is on a hot path. relPath is "" when capture is disabled, the run-scoped
	// latch has tripped, or the write failed — matching the contract that
	// the `transcript_path` attribute is emitted ONLY on success.
	relPath := c.maybeWriteTranscript(
		ctx, opts, transcriptDisabled,
		stageN, currentStatus, action.Provider,
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

	transResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, services.TransitionOptions{})
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

	if c.workflowSvc.IsTerminalStatus(transResult.ToStatus) {
		result.FinalStatus = transResult.ToStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}
	return stageOutcome{nextStatus: transResult.ToStatus}
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
		stageN, currentStatus, provider,
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
	stageN int, status, provider, command string,
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
		opts.ProjectRoot, opts.RunID, stageN,
		status, provider, command,
		exitCode, durationMS, stdout, stderr,
	)
	if err != nil {
		// Warning carries the intended project-relative path so operators can
		// still locate the missing transcript in their mental model even when
		// the file itself could not be created.
		emitTranscriptWarning(ctx, opts.Observability, opts.RunID,
			relTranscriptPath(opts.RunID, stageN, status, provider), err)
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
