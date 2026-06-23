package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

// Core outcome vocabulary (E35-F02, decision D7). Every non-terminal,
// non-parking step is expected to define these; steps may add extras
// (dead-end, needs-info, …).
const (
	OutcomePass    = "pass"
	OutcomeFail    = "fail"
	OutcomeBlocked = "blocked"
)

// CoreOutcomes are the outcome names every workable step must define.
var CoreOutcomes = []string{OutcomePass, OutcomeFail, OutcomeBlocked}

// HasSteps reports whether this config uses the route-based per-step schema.
func (w *WorkflowConfig) HasSteps() bool {
	return w != nil && len(w.Steps) > 0
}

// GetStep returns the step with the given name (case-insensitive) and whether
// it was found. Returns nil, false when the config is not route-based or the
// step is absent.
func (w *WorkflowConfig) GetStep(name string) (*Step, bool) {
	if w == nil || len(w.Steps) == 0 {
		return nil, false
	}
	if st, ok := w.Steps[name]; ok {
		return st, st != nil
	}
	// Case-insensitive fallback.
	for k, st := range w.Steps {
		if strings.EqualFold(k, name) {
			return st, st != nil
		}
	}
	return nil, false
}

// ResolveOutcome resolves the target step for a (fromStep, outcome) pair using
// the route-based outcomes map (E35-F02, D2/D4). It returns the target step
// name and true on success, or "" and false when the step is unknown, has no
// outcomes, or does not define the requested outcome.
//
// Parking and terminal steps have no static outcomes; callers handle those
// separately (parking resume walks history; terminal steps never advance).
func (w *WorkflowConfig) ResolveOutcome(fromStep, outcome string) (string, bool) {
	st, ok := w.GetStep(fromStep)
	if !ok || st == nil {
		return "", false
	}
	if target, ok := st.Outcomes[outcome]; ok {
		return target, true
	}
	// Case-insensitive outcome match for robustness.
	for k, v := range st.Outcomes {
		if strings.EqualFold(k, outcome) {
			return v, true
		}
	}
	return "", false
}

// CoreOutcomeError describes a step that is missing one or more core outcomes.
type CoreOutcomeError struct {
	Step    string
	Missing []string
}

func (e *CoreOutcomeError) Error() string {
	return "step \"" + e.Step + "\" is missing required outcome(s): " + strings.Join(e.Missing, ", ")
}

// ValidateCoreOutcomes checks that every workable step (non-terminal,
// non-parking) defines the core outcome vocabulary (pass/fail/blocked) per
// decision D7. Returns one error per offending step; empty when the config is
// not route-based or all workable steps are complete.
//
// It also verifies that every outcome target names a real step.
func (w *WorkflowConfig) ValidateCoreOutcomes() []error {
	if w == nil || len(w.Steps) == 0 {
		return nil
	}
	var errs []error
	for name, st := range w.Steps {
		if st == nil || st.Terminal || st.Parking {
			continue
		}
		var missing []string
		for _, core := range CoreOutcomes {
			if _, ok := st.Outcomes[core]; !ok {
				missing = append(missing, core)
			}
		}
		if len(missing) > 0 {
			errs = append(errs, &CoreOutcomeError{Step: name, Missing: missing})
		}
		// Every outcome target must resolve to a defined step.
		for outcome, target := range st.Outcomes {
			if _, ok := w.Steps[target]; !ok {
				errs = append(errs, fmt.Errorf("step %q outcome %q targets unknown step %q", name, outcome, target))
			}
		}
	}
	// Deterministic ordering for stable validate output.
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// deriveLegacyFromSteps projects a route-based Steps map back onto the legacy
// StatusFlow / StatusMetadata / SpecialStatuses maps so that every existing
// reader (status calculations, display, validation, the dispatch path) keeps
// working unchanged. This is the strangler seam: Steps is the source of truth;
// the two legacy maps become a derived compatibility view (E35-F01).
//
// It is a no-op when Steps is empty. Existing explicit StatusFlow/StatusMetadata
// values are overwritten for step names present in Steps, but any keys the
// caller set that are NOT step names are preserved.
func deriveLegacyFromSteps(cfg *WorkflowConfig) {
	if cfg == nil || len(cfg.Steps) == 0 {
		return
	}
	if cfg.StatusFlow == nil {
		cfg.StatusFlow = make(map[string][]string)
	}
	if cfg.StatusMetadata == nil {
		cfg.StatusMetadata = make(map[string]StatusMetadata)
	}
	if cfg.SpecialStatuses == nil {
		cfg.SpecialStatuses = make(map[string][]string)
	}

	// First pass: classify steps for special-status derivation.
	var terminals, aggregations, workable []string
	for name, st := range cfg.Steps {
		if st == nil {
			continue
		}
		switch {
		case st.Terminal:
			terminals = append(terminals, name)
		case !st.Parking:
			workable = append(workable, name)
		}
		if st.AggregatesFrom != "" || st.Action == action.ActionCascade {
			aggregations = append(aggregations, name)
		}
	}
	sort.Strings(terminals)
	sort.Strings(aggregations)
	sort.Strings(workable)

	// Second pass: derive per-step metadata and transitions.
	for name, st := range cfg.Steps {
		if st == nil {
			continue
		}
		meta := StatusMetadata{
			Color:               st.Color,
			Description:         st.Description,
			Phase:               st.Phase,
			AgentTypes:          st.AgentTypes,
			ProgressWeight:      st.ProgressWeight,
			Responsibility:      st.Responsibility,
			BlocksFeature:       st.BlocksFeature,
			IsPlanning:          st.IsPlanning,
			AggregatesFrom:      st.AggregatesFrom,
			ExcludeFromProgress: st.ExcludeFromProgress,
		}
		// Build orchestrator action from the consolidated step fields.
		if st.Action != "" || st.Prompt != "" {
			meta.OrchestratorAction = &action.OrchestratorAction{
				Action:              st.Action,
				AgentType:           st.Agent,
				Provider:            st.Provider,
				Model:               st.Model,
				Skills:              st.Skills,
				InstructionTemplate: st.Prompt,
			}
		}
		// Surface the spawn agent as an agent type for `--agent` targeting when
		// the step didn't list agent_types explicitly.
		if st.Agent != "" && len(meta.AgentTypes) == 0 {
			meta.AgentTypes = []string{st.Agent}
		}
		cfg.StatusMetadata[name] = meta

		switch {
		case st.Terminal:
			cfg.StatusFlow[name] = []string{}
		case st.Parking:
			// Parking resume targets are computed from history at runtime; for
			// the legacy transition view we expose every workable step as a
			// permissible return target (mirrors old blocked -> ready_for_X).
			cfg.StatusFlow[name] = append([]string{}, workable...)
		default:
			cfg.StatusFlow[name] = uniqueSortedOutcomeTargets(st.Outcomes)
		}
	}

	// Derive special statuses from the step set. Only set keys the steps imply;
	// never clobber an explicitly-provided value with an empty one.
	if cfg.Start != "" {
		cfg.SpecialStatuses[StartStatusKey] = []string{cfg.Start}
	}
	if len(terminals) > 0 {
		cfg.SpecialStatuses[CompleteStatusKey] = terminals
	}
	if len(aggregations) > 0 {
		cfg.SpecialStatuses[AggregationStatusKey] = aggregations
	}
}

// uniqueSortedOutcomeTargets returns the distinct target step names referenced
// by an outcomes map, in sorted order for deterministic output.
func uniqueSortedOutcomeTargets(outcomes map[string]string) []string {
	if len(outcomes) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(outcomes))
	out := make([]string, 0, len(outcomes))
	for _, target := range outcomes {
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		out = append(out, target)
	}
	sort.Strings(out)
	return out
}
