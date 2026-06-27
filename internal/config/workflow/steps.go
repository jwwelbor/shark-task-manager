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

// AliasMap returns a map from old status name -> new step name, built from each
// step's Aliases list (E35-F05, §7). The map drives the three alias duties:
// one-shot migration, input compat shim, and history-read resolution.
//
// The second return value collects collisions — an old name claimed by more
// than one step — so validate can surface them. On collision the first step
// (in sorted order, for determinism) wins in the returned map.
func (w *WorkflowConfig) AliasMap() (map[string]string, []error) {
	if w == nil || len(w.Steps) == 0 {
		return map[string]string{}, nil
	}
	names := make([]string, 0, len(w.Steps))
	for name := range w.Steps {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make(map[string]string)
	var errs []error
	for _, step := range names {
		st := w.Steps[step]
		if st == nil {
			continue
		}
		for _, old := range st.Aliases {
			if existing, ok := out[old]; ok && existing != step {
				errs = append(errs, fmt.Errorf("alias %q is claimed by both %q and %q", old, existing, step))
				continue
			}
			out[old] = step
		}
	}
	return out, errs
}

// ResolveAlias maps an old status name to its new step name. If status is
// already a step name it is returned unchanged; if it is a known alias the
// target step is returned; otherwise the input is returned unchanged (the
// caller decides whether an unknown status is an error).
func (w *WorkflowConfig) ResolveAlias(status string) string {
	if w == nil {
		return status
	}
	if _, ok := w.GetStep(status); ok {
		return status
	}
	aliases, _ := w.AliasMap()
	if target, ok := aliases[status]; ok {
		return target
	}
	// Case-insensitive alias match.
	for old, target := range aliases {
		if strings.EqualFold(old, status) {
			return target
		}
	}
	return status
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

// DeriveLegacy projects the route-based Steps map onto the legacy
// StatusFlow/StatusMetadata/SpecialStatuses maps. The loader calls this
// automatically; it is exported so callers that construct a WorkflowConfig
// in-memory (e.g. tests) can populate the legacy compatibility view.
func (w *WorkflowConfig) DeriveLegacy() {
	buildWorkflowMapsFromSteps(w)
}

// buildWorkflowMapsFromSteps projects a route-based Steps map onto the
// StatusFlow / StatusMetadata / SpecialStatuses maps so that every existing
// reader (status calculations, display, validation, the dispatch path) keeps
// working unchanged. Steps is the source of truth; the three maps are a
// derived compatibility view built from it (E35-F01).
//
// It is a no-op when Steps is empty. Existing explicit StatusFlow/StatusMetadata
// values are overwritten for step names present in Steps, but any keys the
// caller set that are NOT step names are preserved.
func buildWorkflowMapsFromSteps(cfg *WorkflowConfig) {
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
			DisplayToken:        st.DisplayToken,
			Description:         st.Description,
			Phase:               st.Phase,
			AgentTypes:          st.AgentTypes,
			ProgressWeight:      st.ProgressWeight,
			Responsibility:      st.Responsibility,
			BlocksFeature:       st.BlocksFeature,
			IsPlanning:          st.IsPlanning,
			AggregatesFrom:      st.AggregatesFrom,
			ExcludeFromProgress: st.ExcludeFromProgress,
			SprintBucket:        st.SprintBucket,
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

// uniqueSortedOutcomeTargets returns the distinct target step names from an
// outcomes map ordered so that AvailableTransitions[0] is always the
// forward/happy-path route. Targets are ordered by the semantic priority of
// their outcome key (pass=0, fail=1, blocked=2, everything else=3), with
// alphabetical order as a tiebreaker within the same tier. When a target
// appears under multiple outcome keys the lowest (best) priority wins.
func uniqueSortedOutcomeTargets(outcomes map[string]string) []string {
	if len(outcomes) == 0 {
		return []string{}
	}
	keyPriority := map[string]int{"pass": 0, "fail": 1, "blocked": 2}
	// bestPriority tracks the lowest outcome-key priority seen for each target.
	bestPriority := make(map[string]int, len(outcomes))
	for key, target := range outcomes {
		if target == "" {
			continue
		}
		p, ok := keyPriority[key]
		if !ok {
			p = 3
		}
		if cur, seen := bestPriority[target]; !seen || p < cur {
			bestPriority[target] = p
		}
	}
	type entry struct {
		target   string
		priority int
	}
	entries := make([]entry, 0, len(bestPriority))
	for target, p := range bestPriority {
		entries = append(entries, entry{target, p})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].priority != entries[j].priority {
			return entries[i].priority < entries[j].priority
		}
		return entries[i].target < entries[j].target
	})
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.target
	}
	return out
}
