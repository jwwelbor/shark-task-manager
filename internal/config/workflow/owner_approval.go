package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/entitytype"
)

// OwnerApprovalStepName is the step injected by the require_owner_approval
// config flag. When a workflow already defines a step with this name, the
// injection is a no-op and the workflow's own definition is trusted.
const OwnerApprovalStepName = "owner_approval"

// requireOwnerApprovalKey is the .sharkconfig.json field that enables the gate.
const requireOwnerApprovalKey = "require_owner_approval"

// NormalizeOwnerApprovalLevels converts a raw require_owner_approval config
// value into the list of entity workflow levels to gate:
//
//   - true            -> every level in KnownLevels
//   - false / nil     -> nil (disabled)
//   - "feature"       -> single-level list
//   - ["feature",...] -> the listed levels
//
// Level names are normalized via entitytype aliases ("tech-debt" ->
// "tech_debt", …). Unknown levels and non-string entries are an error.
func NormalizeOwnerApprovalLevels(v interface{}) ([]string, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	case bool:
		if !val {
			return nil, nil
		}
		return append([]string(nil), KnownLevels...), nil
	case string:
		level, err := normalizeOwnerApprovalLevel(val)
		if err != nil {
			return nil, err
		}
		return []string{level}, nil
	case []string:
		levels := make([]string, 0, len(val))
		for _, s := range val {
			level, err := normalizeOwnerApprovalLevel(s)
			if err != nil {
				return nil, err
			}
			levels = append(levels, level)
		}
		return levels, nil
	case []interface{}:
		levels := make([]string, 0, len(val))
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("invalid %s entry %v: must be an entity type string", requireOwnerApprovalKey, item)
			}
			level, err := normalizeOwnerApprovalLevel(s)
			if err != nil {
				return nil, err
			}
			levels = append(levels, level)
		}
		return levels, nil
	default:
		return nil, fmt.Errorf("invalid %s: must be true, false, or a list of entity types", requireOwnerApprovalKey)
	}
}

func normalizeOwnerApprovalLevel(raw string) (string, error) {
	level := entitytype.WorkflowLevelOrSelf(strings.TrimSpace(raw))
	for _, known := range KnownLevels {
		if level == known {
			return level, nil
		}
	}
	return "", fmt.Errorf("invalid %s entry %q: unknown entity type (valid: %s)", requireOwnerApprovalKey, raw, strings.Join(KnownLevels, ", "))
}

// applyOwnerApprovalGates reads require_owner_approval from the raw
// .sharkconfig.json and injects the owner-approval gate into each configured
// level's workflow. Levels whose slot is nil (embedded default) are
// materialized first so the gated copy is what GetWorkflowForLevel returns.
func applyOwnerApprovalGates(result *MultiLevelWorkflow, rawConfig map[string]json.RawMessage) error {
	raw, ok := rawConfig[requireOwnerApprovalKey]
	if !ok {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid %s: %w", requireOwnerApprovalKey, err)
	}
	levels, err := NormalizeOwnerApprovalLevels(v)
	if err != nil {
		return err
	}
	for _, level := range levels {
		wf := result.GetByType(level)
		if wf == nil {
			wf = defaultForType(level)
			result.setByType(level, wf)
		}
		if err := wf.InjectOwnerApprovalGate(); err != nil {
			return fmt.Errorf("%s (%s): %w", requireOwnerApprovalKey, level, err)
		}
	}
	return nil
}

// InjectOwnerApprovalGate rewrites every route into the workflow's primary
// archive terminal (ArchiveTerminalStatus) so it passes through an injected
// owner_approval step first. The injected step pauses the dispatch loop
// (action: pause, responsibility: human); the owner completes the entity with
// `shark status advance <key> --outcome pass` or sends it back with fail.
//
// No-ops when the workflow already defines an owner_approval step or when
// nothing routes to the terminal. Routes to non-primary terminals (e.g.
// cancelled) are untouched.
func (w *WorkflowConfig) InjectOwnerApprovalGate() error {
	if w == nil || !w.HasSteps() {
		return fmt.Errorf("requires a route-based workflow (steps:)")
	}
	if _, ok := w.Steps[OwnerApprovalStepName]; ok {
		return nil
	}
	// Inline sections may not have the derived legacy view yet; the terminal
	// selector reads SpecialStatuses, so derive before selecting.
	w.DeriveLegacy()
	terminal, err := w.ArchiveTerminalStatus()
	if err != nil {
		return fmt.Errorf("cannot determine completion terminal: %w", err)
	}

	var sources []string
	for name, st := range w.Steps {
		if st == nil || st.Terminal || st.Parking {
			continue
		}
		rewrote := false
		for outcome, target := range st.Outcomes {
			if target == terminal {
				st.Outcomes[outcome] = OwnerApprovalStepName
				rewrote = true
			}
		}
		if rewrote {
			sources = append(sources, name)
		}
	}
	if len(sources) == 0 {
		return nil
	}

	// Derive fail/blocked routing from the (sorted-first) gate step so an
	// owner rejection lands where the gate's own rejection would. Fall back to
	// the gate step itself when the target is missing or was itself rewritten.
	sort.Strings(sources)
	src := w.Steps[sources[0]]
	failTarget := src.Outcomes[OutcomeFail]
	if failTarget == "" || failTarget == OwnerApprovalStepName {
		failTarget = sources[0]
	}
	blockedTarget := src.Outcomes[OutcomeBlocked]
	if blockedTarget == "" || blockedTarget == OwnerApprovalStepName {
		blockedTarget = sources[0]
	}

	w.Steps[OwnerApprovalStepName] = &Step{
		Phase:          "approval",
		Color:          "yellow",
		DisplayToken:   "OAP",
		Description:    "Awaiting owner sign-off (require_owner_approval): advance with --outcome pass to complete, fail to send back",
		ProgressWeight: src.ProgressWeight,
		Responsibility: "human",
		Action:         action.ActionPause,
		Outcomes: map[string]string{
			OutcomePass:    terminal,
			OutcomeFail:    failTarget,
			OutcomeBlocked: blockedTarget,
		},
	}
	// Refresh the derived StatusFlow/StatusMetadata view with the new routing.
	w.DeriveLegacy()
	return nil
}
