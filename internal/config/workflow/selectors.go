package workflow

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

// Named selectors: the single place where "which status do we mean?" is
// answered for selections that used to be positional [0] picks scattered
// across call sites.
//
// The contract (the designation rule):
//  1. Exactly one candidate — use it. A [0] pick over a single-element slice
//     is trivially correct, so most workflows never need anything more.
//  2. Several candidates — use the step tagged primary: true. Zero or several
//     primary tags is a config error (validatePrimaryDesignations rejects it
//     at load/validate time for the selections it can see).
//  3. At runtime against an already-loaded ambiguous config the selectors
//     hard-fail with an AmbiguousSelectionError naming the candidates and the
//     fix. An arbitrary (alphabetical, map-order) pick never happens.
//
// Callers distinguish the two failure modes with errors.As: a
// *NoCandidateError usually means "fall back / skip", an
// *AmbiguousSelectionError must surface to the user.

// NoCandidateError reports a selection with no candidate statuses (e.g. a
// workflow with no aggregation step). Callers typically fall back to a
// default or skip the operation.
type NoCandidateError struct {
	// Selection names what was being selected, e.g. `aggregation (reopen-target)`.
	Selection string
}

func (e *NoCandidateError) Error() string {
	return fmt.Sprintf("workflow defines no %s status", e.Selection)
}

// AmbiguousSelectionError reports a selection with several candidates and no
// single primary: true designation. It must surface to the user — callers
// never break the tie themselves.
type AmbiguousSelectionError struct {
	// Selection names what was being selected.
	Selection string
	// Candidates are the competing status names.
	Candidates []string
	// Primaries are the candidates tagged primary: true — empty when none is
	// tagged, more than one when the tag itself is ambiguous.
	Primaries []string
}

func (e *AmbiguousSelectionError) Error() string {
	if len(e.Primaries) > 1 {
		return fmt.Sprintf("ambiguous %s selection: multiple steps tagged primary (%s); keep primary: true on exactly one, then run 'shark admin workflow validate'",
			e.Selection, strings.Join(e.Primaries, ", "))
	}
	return fmt.Sprintf("ambiguous %s selection: candidates are %s; tag exactly one of these steps with primary: true, then run 'shark admin workflow validate'",
		e.Selection, strings.Join(e.Candidates, ", "))
}

// designate applies the designation rule to one candidate set.
func (w *WorkflowConfig) designate(selection string, candidates []string) (string, error) {
	switch len(candidates) {
	case 0:
		return "", &NoCandidateError{Selection: selection}
	case 1:
		return candidates[0], nil
	}
	var primaries []string
	for _, name := range candidates {
		if st, ok := w.GetStep(name); ok && st.Primary {
			primaries = append(primaries, name)
		}
	}
	if len(primaries) == 1 {
		return primaries[0], nil
	}
	if !w.HasSteps() {
		// A legacy (status_flow schema) config cannot express primary: true,
		// so the strict rule would hard-fail with a fix the author cannot
		// apply. Preserve the pre-2.x behavior for legacy configs: first
		// candidate wins (declaration order for special_statuses arrays,
		// sorted order for phase lookups). The strict designation rule applies
		// only to route-based (steps:) workflows.
		return candidates[0], nil //shark:ordered legacy-schema back-compat, see designate doc
	}
	return "", &AmbiguousSelectionError{Selection: selection, Candidates: candidates, Primaries: primaries}
}

// DefaultTransition returns the default (happy-path) transition out of a
// status: StatusFlow[from][0]. That position is a guaranteed contract, not an
// accident — for route-based workflows every StatusFlow slice is produced by
// uniqueSortedOutcomeTargets, which orders targets semantically (pass, then
// fail, then blocked, then extras). Returns a *NoCandidateError for a
// terminal or unknown status.
func (w *WorkflowConfig) DefaultTransition(from string) (string, error) {
	for status, targets := range w.StatusFlow {
		if strings.EqualFold(status, from) {
			if len(targets) == 0 {
				return "", &NoCandidateError{Selection: fmt.Sprintf("transition out of terminal status %q", from)}
			}
			return targets[0], nil
		}
	}
	return "", &NoCandidateError{Selection: fmt.Sprintf("transition out of unknown status %q", from)}
}

// PrimaryAggregationStatus returns the workflow's aggregation status — the
// step a reopened parent entity returns to when a child is added or reopened
// under a terminal parent.
func (w *WorkflowConfig) PrimaryAggregationStatus() (string, error) {
	return w.designate("aggregation (reopen-target)", w.SpecialStatuses[AggregationStatusKey])
}

// StatusForPhase returns the canonical status of a phase (e.g. the sprint
// workflow's "execution" status).
func (w *WorkflowConfig) StatusForPhase(phase string) (string, error) {
	return w.designate(fmt.Sprintf("%q-phase", phase), w.GetStatusesByPhase(phase))
}

// CompletedSprintStatus returns the done-phase, non-terminal status a sprint
// moves to once its carryover is processed but before it is archived. Phase
// alone can't disambiguate this from the archive terminal (both typically
// share the "done" phase), so terminal statuses are excluded from the
// candidate set.
func (w *WorkflowConfig) CompletedSprintStatus() (string, error) {
	terminal := make(map[string]bool)
	for _, status := range w.SpecialStatuses[CompleteStatusKey] {
		terminal[strings.ToLower(status)] = true
	}
	var candidates []string
	for _, status := range w.GetStatusesByPhase("done") {
		if !terminal[strings.ToLower(status)] {
			candidates = append(candidates, status)
		}
	}
	return w.designate("completed (done-phase, non-terminal)", candidates)
}

// ArchiveTerminalStatus returns the terminal status an archive operation
// should transition into. When a workflow defines several terminals (e.g. an
// "archived" success path alongside a "cancelled" abandon path), the ones
// whose action is "archive" take precedence; the designation rule then breaks
// any remaining tie.
func (w *WorkflowConfig) ArchiveTerminalStatus() (string, error) {
	if archival := w.archiveActionTerminals(); len(archival) > 0 {
		return w.designate("archive terminal", archival)
	}
	return w.designate("terminal", w.SpecialStatuses[CompleteStatusKey])
}

// archiveActionTerminals returns the terminal statuses whose orchestrator
// action is archive — the operation-specific subset ArchiveTerminalStatus
// prefers and validatePrimaryDesignations checks, so validate-time and
// runtime always agree on the candidate set.
func (w *WorkflowConfig) archiveActionTerminals() []string {
	var archival []string
	for _, status := range w.SpecialStatuses[CompleteStatusKey] {
		if meta, ok := w.GetStatusMetadata(status); ok &&
			meta.OrchestratorAction != nil && strings.EqualFold(meta.OrchestratorAction.Action, action.ActionArchive) {
			archival = append(archival, status)
		}
	}
	return archival
}
