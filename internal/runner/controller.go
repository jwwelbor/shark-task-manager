// Package runner provides the AgentDispatcher interface and related types for
// invoking external AI agents (Claude, Codex, etc.) as part of the E22 run loop.
// This file defines the RunController, its interfaces, and supporting data types.
package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
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

	return &RunController{
		transitioner: deps.Transitioner,
		placeholders: deps.Placeholders,
		actionSvc:    deps.ActionSvc,
		workflowSvc:  deps.WorkflowSvc,
		dispatchers:  deps.Dispatchers,
	}, nil
}

// stageOutcome is the return value from per-action handler methods. It tells the
// main loop whether to continue, stop, or report a failure.
type stageOutcome struct {
	// nextStatus is the status to use for the next iteration (when done == false).
	nextStatus string
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
	for {
		// Check context cancellation at the top of each iteration.
		select {
		case <-ctx.Done():
			result.FinalStatus = currentStatus
			result.Outcome = "failed"
			result.Error = ctx.Err().Error()
			result.TotalDuration = time.Since(startTime)
			return result, nil
		default:
		}

		stageStart := time.Now()

		// Step 3: Get template variables for instruction rendering.
		var vars map[string]string
		if c.placeholders != nil {
			vars, err = c.placeholders.GeneratePlaceholders(ctx, key)
			if err != nil {
				result.FinalStatus = currentStatus
				result.Outcome = "failed"
				result.Error = fmt.Sprintf("failed to generate placeholders for %s: %v", key, err)
				result.TotalDuration = time.Since(startTime)
				return result, nil
			}
		}

		// Step 4: Get populated orchestrator action for current status.
		action, err := c.actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)
		if err != nil {
			result.FinalStatus = currentStatus
			result.Outcome = "failed"
			result.Error = fmt.Sprintf("failed to get action for status %s: %v", currentStatus, err)
			result.TotalDuration = time.Since(startTime)
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
			outcome = c.handleAdvanceStatus(ctx, key, currentStatus, action, opts, result, stageStart, startTime)

		case config.ActionSpawnAgent:
			outcome = c.handleSpawnAgent(ctx, key, currentStatus, action, opts, result, stageStart, startTime)

		default:
			result.FinalStatus = currentStatus
			result.Outcome = "failed"
			result.Error = fmt.Sprintf("unknown action type %q for status %s", action.Action, currentStatus)
			result.TotalDuration = time.Since(startTime)
			return result, nil
		}

		if outcome.done {
			return result, nil
		}
		currentStatus = outcome.nextStatus
	}
}

// handleAdvanceStatus handles the advance_status action type: transitions the
// entity to the next status without dispatching an agent.
func (c *RunController) handleAdvanceStatus(
	ctx context.Context, key, currentStatus string,
	action *config.PopulatedAction, opts RunOptions,
	result *RunResult, stageStart, startTime time.Time,
) stageOutcome {
	if opts.DryRun {
		nextInfo, err := c.transitioner.GetNextStatus(ctx, key)
		if err != nil {
			result.FinalStatus = currentStatus
			result.Outcome = "failed"
			result.Error = err.Error()
			result.TotalDuration = time.Since(startTime)
			return stageOutcome{done: true}
		}
		result.Stages = append(result.Stages, StageLog{
			Status:    currentStatus,
			Action:    action.Action,
			AgentType: action.AgentType,
			Provider:  action.Provider,
			Duration:  time.Since(stageStart),
		})
		result.StagesCompleted++
		if len(nextInfo.AvailableTransitions) == 0 || nextInfo.IsTerminal {
			result.FinalStatus = currentStatus
			result.Outcome = "completed"
			result.TotalDuration = time.Since(startTime)
			return stageOutcome{done: true}
		}
		return stageOutcome{nextStatus: nextInfo.AvailableTransitions[0].TargetStatus}
	}

	nextInfo, err := c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = err.Error()
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	if len(nextInfo.AvailableTransitions) == 0 {
		result.FinalStatus = currentStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus
	transResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, services.TransitionOptions{})
	if err != nil {
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = fmt.Sprintf("transition to %s failed: %v", targetStatus, err)
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

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

// handleSpawnAgent handles the spawn_agent action type: dispatches an agent,
// gates status advancement on exit code 0.
func (c *RunController) handleSpawnAgent(
	ctx context.Context, key, currentStatus string,
	action *config.PopulatedAction, opts RunOptions,
	result *RunResult, stageStart, startTime time.Time,
) stageOutcome {
	dispatcher, err := c.selectDispatcher(action.Provider)
	if err != nil {
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = err.Error()
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	input := DispatchInput{
		Instruction: action.Instruction,
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

		nextInfo, err := c.transitioner.GetNextStatus(ctx, key)
		if err != nil || len(nextInfo.AvailableTransitions) == 0 || nextInfo.IsTerminal {
			result.FinalStatus = currentStatus
			result.Outcome = "completed"
			result.TotalDuration = time.Since(startTime)
			return stageOutcome{done: true}
		}
		return stageOutcome{nextStatus: nextInfo.AvailableTransitions[0].TargetStatus}
	}

	dispatchResult, err := dispatcher.Dispatch(ctx, input)
	if err != nil {
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = err.Error()
		result.TotalDuration = time.Since(startTime)
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
		result.StagesCompleted = len(result.Stages) - 1
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = fmt.Sprintf("agent exited with code %d", dispatchResult.ExitCode)
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	result.StagesCompleted++

	nextInfo, err := c.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = err.Error()
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	if len(nextInfo.AvailableTransitions) == 0 {
		result.FinalStatus = currentStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus
	transResult, err := c.transitioner.TransitionStatus(ctx, key, targetStatus, services.TransitionOptions{})
	if err != nil {
		result.FinalStatus = currentStatus
		result.Outcome = "failed"
		result.Error = fmt.Sprintf("transition to %s failed: %v", targetStatus, err)
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}

	if c.workflowSvc.IsTerminalStatus(transResult.ToStatus) {
		result.FinalStatus = transResult.ToStatus
		result.Outcome = "completed"
		result.TotalDuration = time.Since(startTime)
		return stageOutcome{done: true}
	}
	return stageOutcome{nextStatus: transResult.ToStatus}
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
