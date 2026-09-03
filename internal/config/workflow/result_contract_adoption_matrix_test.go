package workflow

import (
	"sort"
	"testing"
)

// canonicalAdoptionMatrix is architecture.md's "Compatibility and migration"
// 13-entry canonical adoption matrix (T-E34-F05-005), minus the one entry
// this task cannot land yet: epic `integration_review` does not exist in the
// shipped epic workflow — it is added by E34-F08 (the epic integration
// candidate feature). Until that step exists, this test asserts the 12
// entries that do.
var canonicalAdoptionMatrix = map[string][]string{
	"epic":     {"feature_review"},
	"feature":  {"specification", "test_planning", "task_review", "code_review", "qa", "approval"},
	"techDebt": {"triaged", "in_progress"},
	"change":   {"development", "code_review", "qa"},
}

// TestCanonicalAdoptionMatrix_ResolvesGateResultV1 proves every named
// adoption-matrix step resolves result_contract: gate_result_v1 with a
// complete outcome_roles map, and that ValidateWorkflow (which now runs
// ValidateResultContracts) still accepts the shipped bundle.
func TestCanonicalAdoptionMatrix_ResolvesGateResultV1(t *testing.T) {
	mlw, err := LoadMultiLevelWorkflowFromYAMLDir(canonicalWorkflowDir(t), "")
	if err != nil {
		t.Fatalf("failed to load canonical workflows: %v", err)
	}

	slots := map[string]*WorkflowConfig{
		"epic":     mlw.Epic,
		"feature":  mlw.Feature,
		"techDebt": mlw.TechDebt,
		"change":   mlw.Change,
	}

	for slot, steps := range canonicalAdoptionMatrix {
		cfg := slots[slot]
		if cfg == nil {
			t.Fatalf("canonical %s workflow did not load", slot)
		}
		for _, stepName := range steps {
			st, ok := cfg.GetStep(stepName)
			if !ok || st == nil {
				t.Errorf("%s.%s: step not found", slot, stepName)
				continue
			}
			if st.ResultContract != ResultContractGateResultV1 {
				t.Errorf("%s.%s: expected result_contract %q, got %q", slot, stepName, ResultContractGateResultV1, st.ResultContract)
			}
			if len(st.OutcomeRoles) != len(st.Outcomes) {
				t.Errorf("%s.%s: outcome_roles (%d) does not exactly cover outcomes (%d): roles=%#v outcomes=%#v",
					slot, stepName, len(st.OutcomeRoles), len(st.Outcomes), st.OutcomeRoles, st.Outcomes)
			}
			for outcome := range st.Outcomes {
				if _, ok := st.OutcomeRoles[outcome]; !ok {
					t.Errorf("%s.%s: outcome_roles missing entry for outcome %q", slot, stepName, outcome)
				}
			}
		}
	}

	if errs := mlw.Epic.ValidateResultContracts(); len(errs) != 0 {
		t.Errorf("epic workflow result_contract validation failed: %v", errs)
	}
	if errs := mlw.Feature.ValidateResultContracts(); len(errs) != 0 {
		t.Errorf("feature workflow result_contract validation failed: %v", errs)
	}
	if errs := mlw.TechDebt.ValidateResultContracts(); len(errs) != 0 {
		t.Errorf("tech-debt workflow result_contract validation failed: %v", errs)
	}
	if errs := mlw.Change.ValidateResultContracts(); len(errs) != 0 {
		t.Errorf("change workflow result_contract validation failed: %v", errs)
	}
}

// TestCanonicalAdoptionMatrix_OtherStepsStayLegacy proves every step NOT
// named in the adoption matrix stays "legacy" (REQ-F-006: "Other steps
// remain legacy unless a later versioned migration names them") — a
// migration typo that accidentally opts in an extra step, or a step this
// task should have migrated but silently skipped, both surface here.
func TestCanonicalAdoptionMatrix_OtherStepsStayLegacy(t *testing.T) {
	mlw, err := LoadMultiLevelWorkflowFromYAMLDir(canonicalWorkflowDir(t), "")
	if err != nil {
		t.Fatalf("failed to load canonical workflows: %v", err)
	}
	slots := map[string]*WorkflowConfig{
		"epic":     mlw.Epic,
		"feature":  mlw.Feature,
		"task":     mlw.Task,
		"bug":      mlw.Bug,
		"change":   mlw.Change,
		"techDebt": mlw.TechDebt,
		"sprint":   mlw.Sprint,
	}
	for slot, cfg := range slots {
		if cfg == nil {
			continue
		}
		adopted := make(map[string]bool)
		for _, name := range canonicalAdoptionMatrix[slot] {
			adopted[name] = true
		}
		names := make([]string, 0, len(cfg.Steps))
		for name := range cfg.Steps {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			st := cfg.Steps[name]
			if st == nil {
				continue
			}
			if adopted[name] {
				continue
			}
			if st.ResultContract != "" && st.ResultContract != ResultContractLegacy {
				t.Errorf("%s.%s: not in the canonical adoption matrix but result_contract is %q (expected legacy/omitted)", slot, name, st.ResultContract)
			}
		}
	}
}
