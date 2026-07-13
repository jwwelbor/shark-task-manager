// Package dispatch contains service-level dispatch metadata contracts shared by
// CLI callers and higher-level orchestration services.
package dispatch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// EntityTransitioner is the read-only portion of the entity status service
// needed to resolve a step. The transition method remains part of the
// interface because existing service adapters satisfy the full contract; the
// resolver never calls it.
type EntityTransitioner interface {
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

type PlaceholderGenerator interface {
	GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error)
}

type PromptAssemblyInput struct {
	Instruction string
	AgentType   string
	Vars        map[string]string
}

type PromptAssembler interface {
	AssemblePrompt(ctx context.Context, input PromptAssemblyInput) (string, error)
}

type PromptAssemblerFunc func(ctx context.Context, input PromptAssemblyInput) (string, error)

func (f PromptAssemblerFunc) AssemblePrompt(ctx context.Context, input PromptAssemblyInput) (string, error) {
	return f(ctx, input)
}

// GateClassification describes why a resolved step does not represent an
// ordinary worker dispatch. The values are intentionally independent of the
// CLI wire vocabulary so team planning can classify a step without invoking
// Cobra.
type GateClassification string

const (
	GateNone     GateClassification = ""
	GatePause    GateClassification = "pause"
	GateHuman    GateClassification = "human_gate"
	GateTerminal GateClassification = "terminal"
)

// DispatchStep is the transient, service-level result of resolving one entity's
// current workflow step. Prompt is deliberately transient: callers that build
// durable team plans should copy metadata and diagnostics only.
type DispatchStep struct {
	EntityKey              string             `json:"entity_key"`
	EntityType             models.EntityType  `json:"entity_type"`
	Status                 string             `json:"status"`
	Action                 string             `json:"action"`
	AgentType              string             `json:"agent_type,omitempty"`
	Provider               string             `json:"provider,omitempty"`
	Model                  string             `json:"model,omitempty"`
	Effort                 string             `json:"effort,omitempty"`
	HasAction              bool               `json:"-"`
	GateClassification     GateClassification `json:"gate_classification,omitempty"`
	UnresolvedPlaceholders []string           `json:"unresolved_placeholders,omitempty"`
	Prompt                 string             `json:"-"`
	Vars                   map[string]string  `json:"-"`
	Error                  string             `json:"error,omitempty"`

	// NextStatus is transient context retained for the CLI's existing
	// cascade/auto-advance behavior. It is excluded from serialized metadata.
	NextStatus *services.NextStatusInfo `json:"-"`
}

// DispatchStepResolver is the narrow contract consumed by planning and
// ordinary single-entity callers. It has no Cobra or mutation dependency.
type DispatchStepResolver interface {
	Resolve(ctx context.Context, entityType models.EntityType, key string) (DispatchStep, error)
}

// StepResolverDeps are the lower-level seams needed to resolve one step.
// Transition and placeholder implementations are the same adapters used by
// the existing next/run paths.
type StepResolverDeps struct {
	Transitioner     EntityTransitioner
	Placeholders     PlaceholderGenerator
	ActionService    action.ActionService
	PromptAssembler  PromptAssembler
	IsArchivedStatus func(entityType models.EntityType, status string) bool
}

// StepResolver implements DispatchStepResolver.
type StepResolver struct {
	transitioner     EntityTransitioner
	placeholders     PlaceholderGenerator
	actionService    action.ActionService
	promptAssembler  PromptAssembler
	isArchivedStatus func(entityType models.EntityType, status string) bool
}

// NewStepResolver constructs a resolver around injected service seams. The
// action service is required for non-terminal steps; terminal steps can be
// resolved without loading workflow actions.
func NewStepResolver(deps StepResolverDeps) (*StepResolver, error) {
	if deps.Transitioner == nil {
		return nil, errors.New("dispatch step resolver: transitioner is required")
	}
	return &StepResolver{
		transitioner:     deps.Transitioner,
		placeholders:     deps.Placeholders,
		actionService:    deps.ActionService,
		promptAssembler:  deps.PromptAssembler,
		isArchivedStatus: deps.IsArchivedStatus,
	}, nil
}

// NewDispatchStepResolver is an explicit alias for callers that prefer the
// contract name when constructing the concrete implementation.
func NewDispatchStepResolver(deps StepResolverDeps) (*StepResolver, error) {
	return NewStepResolver(deps)
}

var _ DispatchStepResolver = (*StepResolver)(nil)

// Resolve reads and classifies the current workflow step. It performs no
// claims, status transitions, file writes, or dispatches.
func (r *StepResolver) Resolve(ctx context.Context, entityType models.EntityType, key string) (DispatchStep, error) {
	info, err := r.getNextStatus(ctx, key)
	if err != nil {
		return DispatchStep{}, err
	}
	step := newDispatchStep(entityType, key, info)
	if r.isTerminal(entityType, info) {
		return terminalStep(step), nil
	}

	vars, err := r.generatePlaceholders(ctx, key)
	if err != nil {
		return DispatchStep{}, err
	}
	resolved, err := r.resolveAction(ctx, key, info.CurrentStatus, vars)
	if err != nil {
		return DispatchStep{}, err
	}
	if resolved.pauseError != "" {
		return pausedStep(step, resolved.pauseError), nil
	}
	if resolved.populated == nil {
		return pausedStep(step, ""), nil
	}

	step = applyActionMetadata(step, resolved.populated, vars)
	step.Prompt, step.UnresolvedPlaceholders, err = r.assemblePrompt(ctx, key, info.CurrentStatus, resolved.populated, vars)
	if err != nil {
		return DispatchStep{}, err
	}
	return step, nil
}

func (r *StepResolver) getNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	info, err := r.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("resolve dispatch step status for %s: %w", key, err)
	}
	if info == nil {
		return nil, fmt.Errorf("resolve dispatch step status for %s: empty status response", key)
	}
	return info, nil
}

func newDispatchStep(entityType models.EntityType, key string, info *services.NextStatusInfo) DispatchStep {
	return DispatchStep{EntityKey: key, EntityType: entityType, Status: info.CurrentStatus, NextStatus: info}
}

func (r *StepResolver) isTerminal(entityType models.EntityType, info *services.NextStatusInfo) bool {
	return info.IsTerminal || (r.isArchivedStatus != nil && r.isArchivedStatus(entityType, info.CurrentStatus))
}

func terminalStep(step DispatchStep) DispatchStep {
	step.Action = action.ActionArchive
	step.GateClassification = GateTerminal
	return step
}

func (r *StepResolver) generatePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	vars := map[string]string{}
	if r.placeholders != nil {
		generated, err := r.placeholders.GeneratePlaceholders(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("generate dispatch placeholders for %s: %w", key, err)
		}
		if generated != nil {
			vars = generated
		}
	}
	templates.AugmentPlaceholderAliases(vars)
	return vars, nil
}

type actionResolution struct {
	populated  *action.PopulatedAction
	pauseError string
}

func (r *StepResolver) resolveAction(ctx context.Context, key, status string, vars map[string]string) (actionResolution, error) {
	if r.actionService == nil {
		return actionResolution{}, errors.New("resolve dispatch step: action service is required for a non-terminal step")
	}
	populated, err := r.actionService.GetStatusActionPopulated(ctx, status, vars)
	if err == nil {
		return actionResolution{populated: populated}, nil
	}
	var statusNotFound *action.StatusNotFoundError
	if errors.As(err, &statusNotFound) {
		return actionResolution{pauseError: fmt.Sprintf("status %q is not defined in the workflow configuration; this may be a legacy status that has been removed", status)}, nil
	}
	return actionResolution{}, fmt.Errorf("populate action for status %q on %s: %w", status, key, err)
}

func pausedStep(step DispatchStep, reason string) DispatchStep {
	step.Action = action.ActionPause
	step.GateClassification = GatePause
	step.Error = reason
	return step
}

func applyActionMetadata(step DispatchStep, populated *action.PopulatedAction, vars map[string]string) DispatchStep {
	step.HasAction = true
	step.Vars = vars
	step.Action = populated.Action
	step.AgentType = populated.AgentType
	step.Provider = populated.Provider
	step.Model = populated.Model
	step.Effort = populated.Effort
	step.GateClassification = classifyGate(populated.Action)
	return step
}

func (r *StepResolver) assemblePrompt(ctx context.Context, key, status string, populated *action.PopulatedAction, vars map[string]string) (string, []string, error) {
	if !requiresPrompt(populated) {
		return "", nil, nil
	}
	if populated.Instruction == "" {
		return "", nil, fmt.Errorf("workflow action for %s status %q rendered an empty instruction; check the configured prompt path/template", key, status)
	}
	prompt := populated.Instruction
	if r.promptAssembler != nil {
		var err error
		prompt, err = r.promptAssembler.AssemblePrompt(ctx, PromptAssemblyInput{Instruction: populated.Instruction, AgentType: populated.AgentType, Vars: vars})
		if err != nil {
			return "", nil, fmt.Errorf("assemble dispatch prompt for %s: %w", key, err)
		}
	}
	return prompt, templates.UnrenderedTokens(prompt), nil
}

func requiresPrompt(populated *action.PopulatedAction) bool {
	if populated == nil {
		return false
	}
	return populated.Action == action.ActionSpawnAgent ||
		populated.Action == action.ActionCheckOrResume ||
		(populated.Action == action.ActionAdvanceStatus && populated.AgentType != "") ||
		(populated.Action == "" && populated.AgentType != "")
}

func classifyGate(actionName string) GateClassification {
	switch actionName {
	case action.ActionArchive:
		return GateTerminal
	case action.ActionPause, action.ActionWaitForTriage:
		return GatePause
	case action.ActionCheckOrResume:
		return GateHuman
	default:
		return GateNone
	}
}
