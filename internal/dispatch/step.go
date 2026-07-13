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
	info, err := r.transitioner.GetNextStatus(ctx, key)
	if err != nil {
		return DispatchStep{}, fmt.Errorf("resolve dispatch step status for %s: %w", key, err)
	}
	if info == nil {
		return DispatchStep{}, fmt.Errorf("resolve dispatch step status for %s: empty status response", key)
	}

	step := DispatchStep{
		EntityKey:  key,
		EntityType: entityType,
		Status:     info.CurrentStatus,
		NextStatus: info,
	}
	if info.IsTerminal || (r.isArchivedStatus != nil && r.isArchivedStatus(entityType, info.CurrentStatus)) {
		step.Action = action.ActionArchive
		step.GateClassification = GateTerminal
		return step, nil
	}

	vars := map[string]string{}
	if r.placeholders != nil {
		vars, err = r.placeholders.GeneratePlaceholders(ctx, key)
		if err != nil {
			return DispatchStep{}, fmt.Errorf("generate dispatch placeholders for %s: %w", key, err)
		}
		if vars == nil {
			vars = map[string]string{}
		}
	}
	templates.AugmentPlaceholderAliases(vars)
	if r.actionService == nil {
		return DispatchStep{}, errors.New("resolve dispatch step: action service is required for a non-terminal step")
	}

	populated, err := r.actionService.GetStatusActionPopulated(ctx, info.CurrentStatus, vars)
	if err != nil {
		var statusNotFound *action.StatusNotFoundError
		if errors.As(err, &statusNotFound) {
			step.Action = action.ActionPause
			step.GateClassification = GatePause
			step.Error = fmt.Sprintf("status %q is not defined in the workflow configuration; this may be a legacy status that has been removed", info.CurrentStatus)
			return step, nil
		}
		return DispatchStep{}, fmt.Errorf("populate action for status %q on %s: %w", info.CurrentStatus, key, err)
	}
	if populated == nil {
		step.Action = action.ActionPause
		step.GateClassification = GatePause
		return step, nil
	}

	step.HasAction = true
	step.Vars = vars
	step.Action = populated.Action
	step.AgentType = populated.AgentType
	step.Provider = populated.Provider
	step.Model = populated.Model
	step.Effort = populated.Effort
	step.GateClassification = classifyGate(populated.Action)

	if requiresPrompt(populated) {
		if populated.Instruction == "" {
			return DispatchStep{}, fmt.Errorf("workflow action for %s status %q rendered an empty instruction; check the configured prompt path/template", key, info.CurrentStatus)
		}
		if r.promptAssembler != nil {
			step.Prompt, err = r.promptAssembler.AssemblePrompt(ctx, PromptAssemblyInput{
				Instruction: populated.Instruction,
				AgentType:   populated.AgentType,
				Vars:        vars,
			})
			if err != nil {
				return DispatchStep{}, fmt.Errorf("assemble dispatch prompt for %s: %w", key, err)
			}
		} else {
			step.Prompt = populated.Instruction
		}
		step.UnresolvedPlaceholders = templates.UnrenderedTokens(step.Prompt)
	}

	return step, nil
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
