// This file guards against the F-01 defect class found during T-E34-F05-004
// rework: a gate_result_v1 step's shipped prompt telling the worker its fail
// outcome's role is one value (route_rework or kickback_rework) while the
// workflow YAML's own outcome_roles.fail declares a different one. Both
// sides must agree — gateresult.ValidateRole enforces the schema's role
// (route_rework requires zero kickbacks, kickback_rework requires at least
// one), so a worker following a prompt that states the wrong role is
// fail-closed rejected on every occurrence of that outcome. This exact
// defect shipped for feature/task_review.md (prompt said route_rework,
// schema said kickback_rework) and, in the opposite direction, was already
// caught once for feature/approval.md during the original T-E34-F05-004
// migration (commit 94c81847).
package commands

import (
	"regexp"
	"testing"

	configworkflow "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/require"
)

// gateResultRoleLineRe matches this project's two prompt phrasings for
// stating a fail outcome's semantic role: "This outcome's role is
// `<role>`" (most prompts) and "fail's `<role>` role" (tech_debt/in_progress.md).
// \s+ (not a literal space) between the phrase and the backtick because
// several shipped prompts wrap the role onto the next markdown line.
var gateResultRoleLineRe = regexp.MustCompile("(?:outcome's role is|fail's)\\s+`([a-z_]+)`")

// TestGateResultV1PromptRoleMatchesWorkflowOutcomeRole sweeps every
// gate_result_v1-configured step whose shipped prompt literally states its
// fail outcome's role, and asserts that stated role matches the step's own
// outcome_roles.fail in the workflow YAML. epic/feature_review.md is
// intentionally excluded: its DECISION section describes per-feature
// kickback behavior in prose ("kicks back individual features") without
// ever printing "role is `...`", so there is no literal claim to check here
// — its correctness was verified by hand during the sweep.
func TestGateResultV1PromptRoleMatchesWorkflowOutcomeRole(t *testing.T) {
	cases := []struct {
		name string
		cfg  *configworkflow.WorkflowConfig
		step string
		tmpl string
	}{
		{"feature approval", configworkflow.DefaultFeatureWorkflow(), "approval", "feature/approval.md"},
		{"feature code_review", configworkflow.DefaultFeatureWorkflow(), "code_review", "feature/code_review.md"},
		{"feature qa", configworkflow.DefaultFeatureWorkflow(), "qa", "feature/qa.md"},
		{"feature specification", configworkflow.DefaultFeatureWorkflow(), "specification", "feature/specification.md"},
		{"feature task_review", configworkflow.DefaultFeatureWorkflow(), "task_review", "feature/task_review.md"},
		{"feature test_planning", configworkflow.DefaultFeatureWorkflow(), "test_planning", "feature/test_planning.md"},
		{"change code_review", configworkflow.DefaultChangeCardWorkflow(), "code_review", "change/code_review.md"},
		{"change development", configworkflow.DefaultChangeCardWorkflow(), "development", "change/development.md"},
		{"change qa", configworkflow.DefaultChangeCardWorkflow(), "qa", "change/qa.md"},
		{"tech_debt in_progress", configworkflow.DefaultTechDebtWorkflow(), "in_progress", "tech_debt/in_progress.md"},
		{"tech_debt triaged", configworkflow.DefaultTechDebtWorkflow(), "triaged", "tech_debt/triaged.md"},
	}

	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes resolved")
	vars := goldenVars()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			step, ok := tc.cfg.Steps[tc.step]
			if !ok || step == nil {
				t.Fatalf("workflow has no step %q — case table is stale", tc.step)
			}
			if step.ResultContract != "gate_result_v1" {
				t.Fatalf("step %q is result_contract %q, not gate_result_v1 — case table is stale", tc.step, step.ResultContract)
			}
			wantRole, ok := step.OutcomeRoles["fail"]
			if !ok {
				t.Fatalf("step %q has no outcome_roles.fail", tc.step)
			}

			rendered, err := renderer.Render(tc.tmpl, vars)
			require.NoError(t, err, "render %s", tc.tmpl)

			m := gateResultRoleLineRe.FindStringSubmatch(rendered)
			if m == nil {
				t.Fatalf("prompt %s does not state its fail outcome's role in the expected \"role is `X`\" form — cannot cross-check against workflow outcome_roles.fail=%q", tc.tmpl, wantRole)
			}
			if m[1] != wantRole {
				t.Fatalf("prompt %s states its fail outcome's role as %q, but the workflow's outcome_roles.fail for step %q is %q — a worker following the prompt literally will be rejected by gateresult.ValidateRole (route_rework requires zero kickbacks, kickback_rework requires at least one)", tc.tmpl, m[1], tc.step, wantRole)
			}
		})
	}

	// Coverage guard: the case table above is a hand-maintained enumeration
	// of every gate_result_v1 step across the four workflows that adopt it
	// (epic, feature, change, tech_debt — bug intentionally stays legacy,
	// per E34-F05's documented design decision). Without this check, adding
	// a NEW gate_result_v1 step with a contradictory prompt would silently
	// skip this test entirely rather than fail it — the same defect class,
	// recurring in an uncovered place. The table has one fewer entry than
	// the true gate_result_v1 step count: epic/feature_review is
	// intentionally excluded (see the test's doc comment) rather than
	// counted here.
	wantCovered := len(cases) + 1
	var gotCovered int
	var uncovered []string
	configs := map[string]*configworkflow.WorkflowConfig{
		"epic":      configworkflow.DefaultEpicWorkflow(),
		"feature":   configworkflow.DefaultFeatureWorkflow(),
		"change":    configworkflow.DefaultChangeCardWorkflow(),
		"tech_debt": configworkflow.DefaultTechDebtWorkflow(),
	}
	covered := make(map[string]bool, len(cases))
	for _, tc := range cases {
		covered[tc.name] = true
	}
	covered["epic feature_review"] = true // documented exclusion, not a table entry
	for level, cfg := range configs {
		for stepName, step := range cfg.Steps {
			if step == nil || step.ResultContract != "gate_result_v1" {
				continue
			}
			gotCovered++
			if !covered[level+" "+stepName] {
				uncovered = append(uncovered, level+"."+stepName)
			}
		}
	}
	if len(uncovered) > 0 {
		t.Fatalf("gate_result_v1 step(s) not covered by this test's case table or its documented exclusion: %v — add a case (or an explicit, reasoned exclusion) so its prompt's stated role is cross-checked against outcome_roles.fail", uncovered)
	}
	if gotCovered != wantCovered {
		t.Fatalf("found %d gate_result_v1 steps across epic/feature/change/tech_debt workflows, want %d (case table + the epic/feature_review exclusion) — this test's coverage bookkeeping is out of sync with the workflow YAML", gotCovered, wantCovered)
	}
}
