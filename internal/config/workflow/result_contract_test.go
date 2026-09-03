package workflow

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

// gateResultConfig returns a minimal route-based config with one
// gate_result_v1 step ("qa") and one legacy step ("draft"), for
// result_contract/outcome_roles schema tests.
func gateResultConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Version: "1.0",
		Start:   "draft",
		Steps: map[string]*Step{
			"draft": {
				Phase:    "planning",
				Action:   action.ActionAdvanceStatus,
				Prompt:   "epic/draft.md",
				Outcomes: map[string]string{"pass": "qa", "fail": "draft", "blocked": "blocked"},
			},
			"qa": {
				Phase:          "qa",
				Action:         action.ActionSpawnAgent,
				Agent:          "qa",
				Prompt:         "feature/qa.md",
				ResultContract: ResultContractGateResultV1,
				Outcomes:       map[string]string{"pass": "completed", "fail": "draft", "blocked": "blocked"},
				OutcomeRoles: map[string]string{
					"pass":    "success",
					"fail":    "route_rework",
					"blocked": "blocked",
				},
			},
			"blocked": {
				Phase:   "blocked",
				Action:  action.ActionPause,
				Parking: true,
				Prompt:  "epic/blocked.md",
			},
			"completed": {
				Phase:    "done",
				Action:   action.ActionArchive,
				Terminal: true,
				Primary:  true,
				Prompt:   "epic/completed.md",
			},
		},
	}
}

func TestGetResultContract_DefaultsToLegacy(t *testing.T) {
	cfg := gateResultConfig()
	if got := cfg.GetResultContract("draft"); got != ResultContractLegacy {
		t.Fatalf("expected legacy for omitted result_contract, got %q", got)
	}
	if got := cfg.GetResultContract("unknown_status"); got != ResultContractLegacy {
		t.Fatalf("expected legacy for unknown status, got %q", got)
	}
	var nilCfg *WorkflowConfig
	if got := nilCfg.GetResultContract("draft"); got != ResultContractLegacy {
		t.Fatalf("expected legacy for nil config, got %q", got)
	}
}

func TestGetResultContract_ResolvesConfiguredValue(t *testing.T) {
	cfg := gateResultConfig()
	if got := cfg.GetResultContract("qa"); got != ResultContractGateResultV1 {
		t.Fatalf("expected gate_result_v1, got %q", got)
	}
}

func TestGetOutcomeRoles_ResolvesConfiguredMap(t *testing.T) {
	cfg := gateResultConfig()
	roles := cfg.GetOutcomeRoles("qa")
	if len(roles) != 3 || roles["pass"] != "success" || roles["fail"] != "route_rework" || roles["blocked"] != "blocked" {
		t.Fatalf("unexpected outcome roles: %#v", roles)
	}
	if got := cfg.GetOutcomeRoles("draft"); got != nil {
		t.Fatalf("expected nil outcome_roles for a legacy step, got %#v", got)
	}
}

func TestValidateResultContracts_ValidConfigPasses(t *testing.T) {
	cfg := gateResultConfig()
	if errs := cfg.ValidateResultContracts(); len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateResultContracts_UnknownValueFails(t *testing.T) {
	cfg := gateResultConfig()
	cfg.Steps["qa"].ResultContract = "not_a_real_contract"
	errs := cfg.ValidateResultContracts()
	if len(errs) == 0 {
		t.Fatal("expected an error for an unknown result_contract value")
	}
	if !strings.Contains(errs[0].Error(), "unknown result_contract") {
		t.Fatalf("unexpected error message: %v", errs[0])
	}
}

func TestValidateResultContracts_MissingRoleFails(t *testing.T) {
	cfg := gateResultConfig()
	delete(cfg.Steps["qa"].OutcomeRoles, "blocked")
	errs := cfg.ValidateResultContracts()
	if len(errs) == 0 {
		t.Fatal("expected an error for a missing outcome role")
	}
	if !strings.Contains(errs[0].Error(), "missing a role") || !strings.Contains(errs[0].Error(), "blocked") {
		t.Fatalf("unexpected error message: %v", errs[0])
	}
}

func TestValidateResultContracts_ExtraRoleFails(t *testing.T) {
	cfg := gateResultConfig()
	cfg.Steps["qa"].OutcomeRoles["deep_verify"] = "success"
	errs := cfg.ValidateResultContracts()
	if len(errs) == 0 {
		t.Fatal("expected an error for an outcome_roles entry with no matching outcome")
	}
	if !strings.Contains(errs[0].Error(), "not configured on this step") {
		t.Fatalf("unexpected error message: %v", errs[0])
	}
}

func TestValidateResultContracts_UnknownRoleTokenFails(t *testing.T) {
	cfg := gateResultConfig()
	cfg.Steps["qa"].OutcomeRoles["pass"] = "rework" // obsolete/unsupported role token
	errs := cfg.ValidateResultContracts()
	if len(errs) == 0 {
		t.Fatal("expected an error for an unsupported role token")
	}
	if !strings.Contains(errs[0].Error(), "is not a supported role") {
		t.Fatalf("unexpected error message: %v", errs[0])
	}
}

func TestValidateResultContracts_LegacyStepNeedsNoRoles(t *testing.T) {
	cfg := gateResultConfig()
	// "draft" is legacy and has outcomes but no outcome_roles at all — must
	// not be flagged.
	if errs := cfg.ValidateResultContracts(); len(errs) != 0 {
		t.Fatalf("expected legacy step to require no roles, got: %v", errs)
	}
}

func TestValidateResultContracts_NonRouteBasedConfigIsNoOp(t *testing.T) {
	cfg := &WorkflowConfig{}
	if errs := cfg.ValidateResultContracts(); len(errs) != 0 {
		t.Fatalf("expected no-op for a non-route-based config, got: %v", errs)
	}
}

// TestParseWorkflowYAML_ResultContractAndOutcomeRoles proves the new Step
// fields actually bind from real YAML bytes through the loader's YAML→JSON
// conversion path (parseWorkflowYAML/ParseWorkflowYAMLBytes), not just from a
// Go-constructed Step{} literal. A missing/incorrect json tag would silently
// leave these fields at their zero value here even though a Go-only test
// would never catch it.
func TestParseWorkflowYAML_ResultContractAndOutcomeRoles(t *testing.T) {
	yamlSrc := `
version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    prompt: epic/draft.md
    outcomes:
      pass: qa
  qa:
    phase: qa
    action: spawn_agent
    agent: qa
    prompt: feature/qa.md
    result_contract: gate_result_v1
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
    outcome_roles:
      pass: success
      fail: route_rework
      blocked: blocked
  blocked:
    phase: blocked
    action: pause
    parking: true
    prompt: epic/blocked.md
  completed:
    phase: done
    action: archive
    terminal: true
    primary: true
    prompt: epic/completed.md
`
	cfg, err := ParseWorkflowYAMLBytes([]byte(yamlSrc), "test.yaml")
	if err != nil {
		t.Fatalf("failed to parse workflow YAML: %v", err)
	}
	qa, ok := cfg.GetStep("qa")
	if !ok || qa == nil {
		t.Fatal("expected qa step to be present")
	}
	if qa.ResultContract != ResultContractGateResultV1 {
		t.Fatalf("expected result_contract to bind from YAML as %q, got %q (json tag likely mismatched)", ResultContractGateResultV1, qa.ResultContract)
	}
	if len(qa.OutcomeRoles) != 3 || qa.OutcomeRoles["pass"] != "success" {
		t.Fatalf("expected outcome_roles to bind from YAML, got %#v", qa.OutcomeRoles)
	}
	if errs := cfg.ValidateResultContracts(); len(errs) != 0 {
		t.Fatalf("expected the parsed config to validate cleanly, got: %v", errs)
	}
}

// TestValidateWorkflow_RejectsBadResultContract proves the schema-level
// result_contract/outcome_roles checks are wired into the full
// ValidateWorkflow entry point (not just the standalone
// ValidateResultContracts helper), so a malformed workflow YAML fails
// `shark admin validate` end-to-end.
func TestValidateWorkflow_RejectsBadResultContract(t *testing.T) {
	cfg := gateResultConfig()
	delete(cfg.Steps["qa"].OutcomeRoles, "fail")
	if err := ValidateWorkflow(cfg); err == nil {
		t.Fatal("expected ValidateWorkflow to reject an incomplete outcome_roles map")
	}
}
